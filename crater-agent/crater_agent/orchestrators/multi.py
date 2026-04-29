"""Coordinator-based multi-agent orchestrator.

Architecture:
  - IntentRouter (deterministic hints + optional LLM) for request classification
  - Coordinator LLM as sole decision-maker for stage transitions
  - 3 subagents: planner, explorer, executor (executor can use read+write tools)
  - No scenario-specific budget presets; 5 global guardrails via MASRuntimeConfig
  - Smart deterministic stage skipping with Coordinator LLM fallback
"""

from __future__ import annotations

import json
import logging
import re
import time
from dataclasses import dataclass
from datetime import datetime
from typing import Any, AsyncIterator, Awaitable, Callable

import httpx
import httpcore
from langchain_core.messages import AIMessage, HumanMessage, SystemMessage, ToolMessage
from openai import APIConnectionError, APITimeoutError, InternalServerError, RateLimitError
from tenacity import retry, retry_if_exception_type, stop_after_attempt, wait_exponential

from crater_agent.agents.base import BaseRoleAgent, RoleExecutionResult
from crater_agent.agents.coordinator import CoordinatorAgent
from crater_agent.agents.executor import ExecutorAgent
from crater_agent.agents.explorer import ExplorerAgent
from crater_agent.agents.planner import PlannerAgent
from crater_agent.agents.general import GeneralPurposeAgent
from crater_agent.agents.guide import GuideAgent
from crater_agent.agents.verifier import VerifierAgent
from crater_agent.eval.ablation import BenchmarkAblationControl
from crater_agent.llm.client import ModelClientFactory
from crater_agent.memory.session import build_history_messages
from crater_agent.orchestrators.artifacts import (
    ExecutionArtifact,
    GoalArtifact,
    ObservationArtifact,
    PlanArtifact,
    RoutingDecision,
    StateView,
)
from crater_agent.orchestrators.intent_router import (
    IntentRouter,
    is_strict_toolless_fast_path_candidate,
)
from crater_agent.orchestrators.state import (
    MASRuntimeConfig,
    MASState,
    MultiAgentActionItem,
)
from crater_agent.report_utils import build_pipeline_report_payload
from crater_agent.scenarios import extract_focus_hints, extract_node_name
from crater_agent.tools.definitions import (
    ALL_TOOLS,
    CONFIRM_TOOL_NAMES,
    READ_ONLY_TOOL_NAMES,
    is_actor_allowed_for_tool,
    is_tool_allowed_for_role,
)
from crater_agent.tools.executor import CompositeToolExecutor, ToolExecutorProtocol

logger = logging.getLogger(__name__)

_RETRYABLE_TOOL_LOOP_LLM_ERRORS = (
    APITimeoutError,
    APIConnectionError,
    InternalServerError,
    RateLimitError,
    httpx.ReadTimeout,
    httpx.RemoteProtocolError,
    httpcore.RemoteProtocolError,
)


def _load_eval_ablation(context: dict[str, Any]) -> BenchmarkAblationControl:
    controls = context.get("_eval_controls") if isinstance(context.get("_eval_controls"), dict) else {}
    return BenchmarkAblationControl.from_context(
        controls.get("mas_ablation") if isinstance(controls, dict) else None
    )


def _apply_runtime_overrides(state: MASState, ablation: BenchmarkAblationControl) -> None:
    for key, value in ablation.runtime_overrides.items():
        if hasattr(state.runtime_config, key):
            setattr(state.runtime_config, key, value)


def _fallback_stage_from_state(
    state: MASState,
    routing: RoutingDecision,
    ablation: BenchmarkAblationControl,
) -> str:
    if ablation.fallback_stage_sequence:
        index = min(max(state.loop_round - 1, 0), len(ablation.fallback_stage_sequence) - 1)
        return ablation.fallback_stage_sequence[index]
    if routing.operation_mode == "write":
        if state.tool_records or state.observation:
            return "act"
        return "observe"
    if not state.tool_records:
        return "observe"
    return "finalize"


def _apply_stage_ablation(
    *,
    candidate: str | None,
    state: MASState,
    routing: RoutingDecision,
    ablation: BenchmarkAblationControl,
) -> str | None:
    """Override or sanitize stage transitions for offline ablations."""
    if not ablation.enabled:
        return candidate

    fixed_stage = ablation.stage_for_round(state.loop_round)
    if fixed_stage:
        return fixed_stage

    if state.loop_round == 1:
        if ablation.force_plan_first:
            candidate = "plan"
        elif ablation.force_observe_first:
            candidate = "observe"

    if candidate in {"plan", "replan"} and ablation.disable_planner:
        return "observe" if not state.tool_records else "finalize"
    if candidate == "verify" and ablation.disable_verifier:
        return "finalize"
    if candidate is None and ablation.disable_coordinator_decision:
        return _fallback_stage_from_state(state, routing, ablation)
    return candidate

CREATE_JOB_TOOL_NAMES = {
    "create_training_job",
    "create_custom_job",
    "create_jupyter_job",
    "create_webide_job",
    "create_pytorch_job",
    "create_tensorflow_job",
}


# ---------------------------------------------------------------------------
# Data classes for tool loop control
# ---------------------------------------------------------------------------

@dataclass
class ToolLoopStopSignal:
    should_stop: bool = False
    summary: str = ""


@dataclass
class ToolLoopOutcome:
    summary: str
    tool_calls: int = 0
    stop_signal: ToolLoopStopSignal | None = None


@dataclass
class VerifierGateDecision:
    run_verifier: bool
    next_stage: str
    reason: str = ""


# ---------------------------------------------------------------------------
# Pure helper functions (preserved from old multi.py)
# ---------------------------------------------------------------------------

def _normalized_text(value: Any) -> str:
    return str(value or "").strip().lower()


def _truncate_text(value: Any, max_chars: int = 320) -> str:
    text = str(value or "").strip()
    if len(text) <= max_chars:
        return text
    return text[:max_chars] + "..."


def _build_tool_loop_observation(tool_name: str, result: dict[str, Any]) -> str:
    if not isinstance(result, dict):
        return _truncate_text(result, max_chars=1200)
    if result.get("status") == "confirmation_required":
        return json.dumps(result, ensure_ascii=False)
    if result.get("status") == "error":
        payload = {
            "status": "error",
            "tool_name": tool_name,
            "message": result.get("message", ""),
            "error_type": result.get("error_type", "unknown"),
            "retryable": bool(result.get("retryable", False)),
        }
        if "status_code" in result:
            payload["status_code"] = result["status_code"]
        if isinstance(result.get("result"), dict):
            payload["result"] = result["result"]
        return _truncate_text(json.dumps(payload, ensure_ascii=False), max_chars=1200)

    payload = result.get("result", result.get("message", ""))
    return _truncate_text(
        json.dumps(payload, ensure_ascii=False) if isinstance(payload, dict) else payload,
        max_chars=1600,
    )


def _confirmation_id_from_result(result: dict[str, Any] | None) -> str:
    if not isinstance(result, dict):
        return ""
    confirmation = result.get("confirmation")
    if not isinstance(confirmation, dict):
        return ""
    return str(confirmation.get("confirm_id") or "").strip()


def _append_pending_confirmation(state: MASState, result: dict[str, Any]) -> None:
    if not isinstance(result, dict):
        return
    confirm_id = _confirmation_id_from_result(result)
    if confirm_id:
        for index, existing in enumerate(state.pending_confirmations):
            if _confirmation_id_from_result(existing) == confirm_id:
                state.pending_confirmations[index] = result
                return
    state.pending_confirmations.append(result)


def _estimate_tokens_from_messages(messages: list[Any]) -> int:
    total_chars = 0
    for message in messages:
        content = getattr(message, "content", "")
        if isinstance(content, list):
            total_chars += sum(len(str(item)) for item in content)
        else:
            total_chars += len(str(content or ""))
    return max(1, total_chars // 4) if total_chars else 0


def _extract_usage_from_tool_loop_response(
    role_agent: Any,
    response: object,
    messages: list[Any],
    output_text: str,
) -> dict[str, int]:
    usage = getattr(response, "usage_metadata", None) or {}
    response_metadata = getattr(response, "response_metadata", None) or {}
    token_usage = (
        response_metadata.get("token_usage") if isinstance(response_metadata, dict) else {}
    ) or {}
    input_tokens = (
        usage.get("input_tokens")
        or usage.get("prompt_tokens")
        or token_usage.get("prompt_tokens")
        or token_usage.get("input_tokens")
        or 0
    )
    output_tokens = (
        usage.get("output_tokens")
        or usage.get("completion_tokens")
        or token_usage.get("completion_tokens")
        or token_usage.get("output_tokens")
        or 0
    )
    total_tokens = (
        usage.get("total_tokens")
        or token_usage.get("total_tokens")
        or (int(input_tokens or 0) + int(output_tokens or 0))
    )
    has_reported_usage = bool(usage) or bool(token_usage)
    return {
        "llm_calls": 1,
        "input_tokens": int(input_tokens or 0),
        "output_tokens": int(output_tokens or 0),
        "total_tokens": int(total_tokens or 0),
        "reported_token_calls": 1 if has_reported_usage else 0,
        "missing_token_calls": 0 if has_reported_usage else 1,
    }


def _resume_result_status(resume_context: dict[str, Any] | None) -> str:
    if not isinstance(resume_context, dict):
        return ""
    return _normalized_text(resume_context.get("result_status"))


def _build_resume_run_started_summary(resume_context: dict[str, Any] | None) -> str:
    status = _resume_result_status(resume_context)
    if status == "rejected":
        return "确认结果已回流，上一轮待确认操作已取消"
    if status:
        return "确认结果已回流，继续执行上一轮计划"
    return "多 Agent 协作已启动"


def _should_restore_routing_from_resume(resume_context: dict[str, Any] | None) -> bool:
    return _resume_result_status(resume_context) not in {"rejected", "error"}


def _build_rejected_resume_final_answer(
    resumed_action: dict[str, Any] | None,
    resume_context: dict[str, Any] | None = None,
) -> str:
    title = str((resumed_action or {}).get("title") or "").strip()
    if not title and isinstance(resume_context, dict):
        title = str(resume_context.get("action_title") or "").strip()
    if title:
        return f"上一轮待确认操作已取消（{title}）。我不会继续沿用上一轮计划，请告诉我接下来要做什么。"
    return "上一轮待确认操作已取消。我不会继续沿用上一轮计划，请告诉我接下来要做什么。"


def _compact_job_detail_record(job: dict[str, Any]) -> dict[str, Any]:
    compact = {
        "jobName": job.get("jobName") or job.get("job_name"),
        "name": job.get("name"),
        "status": job.get("status"),
        "jobType": job.get("jobType") or job.get("job_type"),
        "resources": job.get("resources"),
        "creationTimestamp": job.get("creationTimestamp") or job.get("creation_time"),
        "startTime": job.get("startTime") or job.get("start_time"),
        "priority": job.get("priority"),
        "account": job.get("account"),
        "namespace": job.get("namespace"),
    }
    return {key: value for key, value in compact.items() if value not in (None, "", [])}


def _compact_tool_result_for_prompt(
    tool_name: str,
    result: dict[str, Any],
    tool_args: dict[str, Any] | None = None,
) -> dict[str, Any]:
    if not isinstance(result, dict):
        return {"tool_name": tool_name, "result": _truncate_text(result)}

    if result.get("status") == "error":
        return {
            "tool_name": tool_name,
            "status": "error",
            "message": _truncate_text(result.get("message") or result.get("result")),
        }

    payload = result.get("result", result)
    if tool_name == "list_user_jobs" and isinstance(payload, dict):
        raw_jobs = payload.get("jobs")
        jobs = raw_jobs if isinstance(raw_jobs, list) else []
        simplified_jobs = []
        for job in jobs[:12]:
            if not isinstance(job, dict):
                continue
            simplified_jobs.append(
                {
                    "jobName": job.get("jobName") or job.get("job_name"),
                    "name": job.get("name"),
                    "status": job.get("status"),
                    "jobType": job.get("jobType") or job.get("job_type"),
                    "creationTimestamp": job.get("creationTimestamp") or job.get("completion_time"),
                    "exitCode": job.get("exitCode") or job.get("exit_code"),
                    "failureReason": job.get("failureReason") or job.get("failure_reason"),
                    "gpuModel": job.get("gpuModel") or job.get("gpu_model"),
                }
            )
        return {
            "tool_name": tool_name,
            "count": payload.get("count"),
            "jobs": simplified_jobs,
        }
    if tool_name == "get_health_overview" and isinstance(payload, dict):
        return {
            "tool_name": tool_name,
            "totalJobs": payload.get("totalJobs"),
            "statusCount": payload.get("statusCount"),
            "lookbackDays": payload.get("lookbackDays"),
        }
    if tool_name == "get_job_detail" and isinstance(payload, dict):
        if payload.get("jobName") or payload.get("job_name"):
            return {"tool_name": tool_name, **_compact_job_detail_record(payload)}

        requested_job_name = str(
            (tool_args or {}).get("job_name")
            or (tool_args or {}).get("jobName")
            or (tool_args or {}).get("name")
            or ""
        ).strip()
        job_records = [job for job in payload.values() if isinstance(job, dict)]
        if requested_job_name:
            requested_lower = requested_job_name.lower()
            for job in job_records:
                candidate_name = str(
                    job.get("jobName") or job.get("job_name") or ""
                ).strip().lower()
                if candidate_name == requested_lower:
                    return {"tool_name": tool_name, **_compact_job_detail_record(job)}
        if job_records:
            return {
                "tool_name": tool_name,
                "jobs": [_compact_job_detail_record(job) for job in job_records[:4]],
            }
        return {
            "tool_name": tool_name,
            "availableJobNames": [str(key) for key in payload.keys()][:8],
        }
    return {
        "tool_name": tool_name,
        "result": _truncate_text(payload, max_chars=500),
    }


def _compact_evidence_for_prompt(evidence: list[dict[str, Any]]) -> list[dict[str, Any]]:
    compact: list[dict[str, Any]] = []
    for item in evidence:
        if not isinstance(item, dict):
            continue
        tool_name = str(item.get("tool_name") or "").strip()
        compact.append(
            {
                "tool_name": tool_name,
                "tool_args": item.get("tool_args") if isinstance(item.get("tool_args"), dict) else {},
                "result": _compact_tool_result_for_prompt(
                    tool_name,
                    item.get("result") if isinstance(item.get("result"), dict) else {},
                    item.get("tool_args") if isinstance(item.get("tool_args"), dict) else {},
                ),
            }
        )
    return compact


def _tool_signature(tool_name: str, tool_args: dict[str, Any]) -> str:
    return json.dumps(
        {
            "tool_name": tool_name,
            "tool_args": tool_args,
        },
        ensure_ascii=False,
        sort_keys=True,
    )


def _find_previous_tool_result(
    evidence: list[dict[str, Any]],
    *,
    tool_name: str,
    tool_args: dict[str, Any],
) -> dict[str, Any] | None:
    target_signature = _tool_signature(tool_name, tool_args)
    for entry in reversed(evidence):
        if not isinstance(entry, dict):
            continue
        existing_tool_name = str(entry.get("tool_name") or "").strip()
        existing_args = entry.get("tool_args") if isinstance(entry.get("tool_args"), dict) else {}
        if _tool_signature(existing_tool_name, existing_args) != target_signature:
            continue
        result = entry.get("result")
        if isinstance(result, dict):
            return result
    return None


def _default_action_title(tool_name: str, tool_args: dict[str, Any]) -> str:
    job_name = str(tool_args.get("job_name") or tool_args.get("name") or "").strip().lower()
    if job_name:
        return f"{tool_name}:{job_name}"
    return tool_name


def _extract_result_message(result: dict[str, Any] | None) -> str:
    if not isinstance(result, dict):
        return ""
    message = str(result.get("message") or "").strip()
    if message:
        return message
    payload = result.get("result")
    if isinstance(payload, dict):
        error_message = str(payload.get("error") or payload.get("message") or "").strip()
        if error_message:
            return error_message
    return ""


def _build_action_history_summary(action_history: list[dict[str, Any]]) -> str:
    if not action_history:
        return "暂无执行动作"
    lines: list[str] = []
    for item in action_history[-8:]:
        if not isinstance(item, dict):
            continue
        title = str(item.get("title") or item.get("tool_name") or "action").strip()
        status = str(item.get("status") or "unknown").strip()
        result = item.get("result")
        result_message = _extract_result_message(result if isinstance(result, dict) else None)
        suffix = f": {result_message}" if result_message else ""
        lines.append(f"- {title} -> {status}{suffix}")
    return "\n".join(lines) or "暂无执行动作"


def _build_evidence_summary_fallback(compact_evidence: list[dict[str, Any]]) -> str:
    if not compact_evidence:
        return "暂无只读证据"
    lines: list[str] = []
    for item in compact_evidence[-6:]:
        tool_name = str(item.get("tool_name") or "tool").strip()
        lines.append(f"- {tool_name}: {_truncate_text(item.get('result'), max_chars=180)}")
    return "\n".join(lines)


def _tool_args_for_decision(tool_args: dict[str, Any] | None) -> dict[str, Any]:
    if not isinstance(tool_args, dict):
        return {}
    useful_keys = (
        "job_name",
        "jobName",
        "name",
        "node_name",
        "nodeName",
        "namespace",
        "kind",
        "workload",
        "pvc_name",
        "pvc",
        "statuses",
        "days",
        "limit",
        "metrics",
        "query",
    )
    return {
        key: value
        for key, value in tool_args.items()
        if key in useful_keys and value not in (None, "", [])
    }


def _job_target_for_decision(
    *,
    routing: RoutingDecision,
    page_context: dict[str, Any],
) -> str:
    return str(
        routing.targets.job_name
        or page_context.get("job_name")
        or page_context.get("jobName")
        or page_context.get("name")
        or ""
    ).strip().lower()


def _node_target_for_decision(
    *,
    routing: RoutingDecision,
    page_context: dict[str, Any],
) -> str:
    return str(
        routing.targets.node_name
        or page_context.get("node_name")
        or page_context.get("nodeName")
        or ""
    ).strip()


def _looks_like_job_health_noop_request(
    *,
    user_message: str,
    routing: RoutingDecision,
    page_context: dict[str, Any],
) -> bool:
    if _has_write_intent(routing):
        return False
    if not _job_target_for_decision(routing=routing, page_context=page_context):
        return False
    normalized = _normalized_text(user_message)
    health_tokens = (
        "正常",
        "健康",
        "有没有问题",
        "有问题吗",
        "需不需要",
        "要不要处理",
        "无需",
        "额外处理",
        "状态",
        "running",
        "healthy",
        "noop",
        "no-op",
    )
    return any(token in normalized for token in health_tokens)


def _count_recent_stage(state: MASState, stage: str) -> int:
    count = 0
    for item in reversed(state.controller_trace):
        if not isinstance(item, dict):
            continue
        latest_stage = str(item.get("stage") or "").strip().lower()
        if latest_stage != stage:
            break
        count += 1
    return count


def _build_tool_coverage_labels(state: MASState) -> list[str]:
    tool_names = {record.tool_name for record in state.tool_records}
    labels: list[str] = []
    coverage_map = (
        ("list_user_jobs", "用户作业列表"),
        ("get_health_overview", "用户作业健康概览"),
        ("get_job_detail", "作业详情/状态"),
        ("get_job_events", "作业事件"),
        ("get_job_logs", "作业日志"),
        ("diagnose_job", "作业诊断摘要"),
        ("query_job_metrics", "作业运行指标"),
        ("get_cluster_health_report", "集群健康报告"),
        ("get_cluster_health_overview", "集群健康概览"),
        ("get_node_detail", "节点详情"),
        ("get_node_network_summary", "节点网络摘要"),
        ("diagnose_distributed_job_network", "分布式网络诊断"),
        ("prometheus_query", "Prometheus 指标"),
        ("k8s_rollout_status", "Kubernetes rollout 状态"),
        ("k8s_top_nodes", "节点实时资源"),
        ("k8s_top_pods", "Pod 实时资源"),
        ("check_quota", "账户配额"),
        ("list_available_images", "可用镜像"),
        ("recommend_training_images", "镜像推荐"),
        ("list_available_gpu_models", "可用 GPU 型号"),
    )
    for tool_name, label in coverage_map:
        if tool_name in tool_names:
            labels.append(label)
    return labels


def _build_over_exploration_warnings(state: MASState) -> list[str]:
    warnings: list[str] = []
    if _count_recent_stage(state, "observe") >= 1 and state.tool_records:
        warnings.append("上一阶段已经 observe；再次 observe 必须对应 missing_facts 中的具体缺口。")

    counts: dict[str, int] = {}
    job_tool_counts: dict[str, int] = {}
    for record in state.tool_records:
        counts[record.tool_name] = counts.get(record.tool_name, 0) + 1
        job_name = str(
            record.tool_args.get("job_name")
            or record.tool_args.get("jobName")
            or ""
        ).strip().lower()
        if job_name and record.tool_name in {
            "get_job_detail",
            "get_job_logs",
            "get_job_events",
            "diagnose_job",
            "query_job_metrics",
        }:
            job_tool_counts[job_name] = job_tool_counts.get(job_name, 0) + 1

    repeated_tools = [tool_name for tool_name, count in counts.items() if count >= 2]
    if repeated_tools:
        warnings.append(
            "已有工具被多次调用："
            + ", ".join(repeated_tools[:6])
            + "；除非参数代表新的事实缺口，否则应 finalize 或 act。"
        )

    repeated_job_targets = [
        job_name for job_name, count in job_tool_counts.items() if count >= 3
    ]
    if repeated_job_targets:
        warnings.append(
            "同一作业已覆盖多类证据："
            + ", ".join(repeated_job_targets[:4])
            + "；继续查询前要确认新证据会改变结论。"
        )

    return warnings


def _build_coordinator_decision_context(
    *,
    state: MASState,
    routing: RoutingDecision,
    enabled_tools: list[str],
    raw_enabled_tools: list[str],
    user_message: str,
    page_context: dict[str, Any],
    verification_verdict: str = "",
    verification_summary: str = "",
) -> dict[str, Any]:
    enabled_set = set(enabled_tools)
    job_target = _job_target_for_decision(routing=routing, page_context=page_context)
    node_target = _node_target_for_decision(routing=routing, page_context=page_context)
    write_intent = _has_write_intent(routing)
    allowed_confirmation_tools = [
        tool_name
        for tool_name in enabled_tools
        if tool_name in CONFIRM_TOOL_NAMES
        and is_actor_allowed_for_tool(state.goal.actor_role, tool_name)
    ]
    write_permission_gap = _write_intent_has_no_allowed_write_tool(
        actor_role=state.goal.actor_role,
        enabled_tools=enabled_tools,
        raw_enabled_tools=raw_enabled_tools,
        routing=routing,
    )
    target_known = bool(
        job_target
        or node_target
        or routing.targets.scope == "all"
        or routing.requested_action == "create"
    )

    recent_tools = [
        {
            "tool": record.tool_name,
            "args": _tool_args_for_decision(record.tool_args),
        }
        for record in state.tool_records[-8:]
    ]

    missing_facts: list[str] = []
    if write_intent:
        if write_permission_gap:
            missing_facts.append("当前 actor 没有确认型写工具权限；不能选择 act。")
            if not state.tool_records and any(tool_name in READ_ONLY_TOOL_NAMES for tool_name in enabled_tools):
                missing_facts.append("尚未基于普通只读权限核验目标状态或风险事实。")
        if routing.requested_action not in {"create"} and not target_known:
            missing_facts.append("写操作目标对象尚未定位到单个目标或明确的 all 范围。")
        if allowed_confirmation_tools and target_known and not state.tool_records:
            missing_facts.append("写操作可进入 act，但 Executor 应先做一次与动作直接相关的只读核验。")

    if _looks_like_job_health_noop_request(
        user_message=user_message,
        routing=routing,
        page_context=page_context,
    ):
        if "get_job_detail" in enabled_set and not _has_tool_record(state, "get_job_detail"):
            missing_facts.append("健康/noop 状态确认尚未核验作业详情。")
        if "query_job_metrics" in enabled_set and not _has_tool_record(state, "query_job_metrics"):
            missing_facts.append("健康/noop 状态确认尚未核验运行指标。")

    if not write_intent and not state.tool_records:
        missing_facts.append("尚未进行直接只读取证。")

    repeated_warnings = _build_over_exploration_warnings(state)
    pending_actions = _pending_action_dicts(state.actions)
    recommended_next = "finalize"
    if _has_awaiting_write_confirmation(state):
        recommended_next = "finalize"
    elif write_intent and write_permission_gap:
        recommended_next = "observe" if missing_facts and any(
            "尚未" in item or "目标对象" in item for item in missing_facts
        ) else "finalize"
    elif write_intent and allowed_confirmation_tools and target_known and not _has_terminal_write_result(state):
        recommended_next = "act"
    elif missing_facts:
        recommended_next = "observe"
    elif state.no_progress_count >= 1 or repeated_warnings:
        recommended_next = "finalize"
    elif state.execution or state.observation or state.tool_records:
        recommended_next = "finalize"
    else:
        recommended_next = "observe"

    context: dict[str, Any] = {
        "progress": {
            "tool_calls": len(state.tool_records),
            "attempted_signatures": len(state.attempted_tool_signatures),
            "recent_tools": recent_tools,
            "coverage": _build_tool_coverage_labels(state),
            "recent_controller_stages": [
                str(item.get("stage") or "").strip()
                for item in state.controller_trace[-6:]
                if isinstance(item, dict)
            ],
        },
        "action_readiness": {
            "write_intent": write_intent,
            "requested_action": routing.requested_action,
            "target_known": target_known,
            "target_job": job_target,
            "target_node": node_target,
            "requested_scope": routing.targets.scope,
            "allowed_confirmation_tools": allowed_confirmation_tools,
            "write_permission_gap": write_permission_gap,
            "pending_actions": pending_actions,
            "awaiting_confirmation": _has_awaiting_write_confirmation(state),
            "has_terminal_write_result": _has_terminal_write_result(state),
        },
        "missing_facts": missing_facts,
        "over_exploration_warnings": repeated_warnings,
        "recommended_next_stage_hint": recommended_next,
    }
    if verification_verdict:
        context["verification"] = {
            "verdict": verification_verdict,
            "summary": _truncate_text(verification_summary, max_chars=220),
        }
    return context


def _build_plan_status_summary(plan: PlanArtifact | None, *, prefix: str = "已生成计划") -> str:
    if plan is None:
        return prefix
    if not plan.steps:
        return f"{prefix}：无需额外步骤，可直接收束回答。"
    tool_preview = ", ".join(plan.candidate_tools[:4]) if plan.candidate_tools else "无"
    first_steps = "；".join(plan.steps[:2])
    return f"{prefix}：本次规划 {len(plan.steps)} 步。前两步: {first_steps}。候选工具: {tool_preview}。"


def _build_verification_signature(state: MASState) -> str:
    last_tool_call_id = state.tool_records[-1].tool_call_id if state.tool_records else ""
    last_action_id = ""
    last_action_status = ""
    if state.action_history and isinstance(state.action_history[-1], dict):
        last_action_id = str(state.action_history[-1].get("action_id") or "").strip()
        last_action_status = str(state.action_history[-1].get("status") or "").strip()
    return json.dumps(
        {
            "tool_records": len(state.tool_records),
            "action_history": len(state.action_history),
            "observation_summary": state.observation.summary if state.observation else "",
            "execution_summary": state.execution.summary if state.execution else "",
            "last_tool_call_id": last_tool_call_id,
            "last_action_id": last_action_id,
            "last_action_status": last_action_status,
        },
        ensure_ascii=False,
        sort_keys=True,
    )


def _extract_verification_verdict(result: RoleExecutionResult | None) -> str:
    if result is None:
        return ""
    metadata = result.metadata if isinstance(result.metadata, dict) else {}
    return str(metadata.get("verification_result") or result.status or "").strip().lower()


def _action_status(value: Any) -> str:
    return str(value or "").strip().lower()


def _terminal_write_action_history(state: MASState) -> list[dict[str, Any]]:
    terminal_statuses = {"completed", "success", "error", "failed", "rejected", "skipped"}
    return [
        item
        for item in state.action_history
        if isinstance(item, dict) and _action_status(item.get("status")) in terminal_statuses
    ]


def _has_awaiting_write_confirmation(state: MASState) -> bool:
    if state.pending_confirmations:
        return True
    return any(action.status == "awaiting_confirmation" for action in state.actions)


def _has_terminal_write_result(state: MASState) -> bool:
    return bool(_terminal_write_action_history(state))


def _has_write_intent(routing: RoutingDecision) -> bool:
    return routing.operation_mode == "write" or bool(routing.requested_action)


def _actor_allowed_tools(actor_role: str, enabled_tools: list[str]) -> list[str]:
    return [
        tool_name
        for tool_name in enabled_tools
        if is_actor_allowed_for_tool(actor_role, tool_name)
    ]


def _write_intent_has_no_allowed_write_tool(
    *,
    actor_role: str,
    enabled_tools: list[str],
    raw_enabled_tools: list[str] | None = None,
    routing: RoutingDecision,
) -> bool:
    if not _has_write_intent(routing):
        return False
    tool_candidates = list(raw_enabled_tools or enabled_tools)
    write_tools = [tool_name for tool_name in tool_candidates if tool_name in CONFIRM_TOOL_NAMES]
    allowed_write_tools = [
        tool_name
        for tool_name in enabled_tools
        if tool_name in CONFIRM_TOOL_NAMES
        and is_actor_allowed_for_tool(actor_role, tool_name)
    ]
    return not allowed_write_tools and (
        bool(write_tools)
        or routing.operation_mode == "write"
        or bool(routing.requested_action)
    )


def _build_actor_permission_denied_answer(state: MASState, enabled_tools: list[str]) -> str:
    del enabled_tools
    lines = [
        "结论：你当前没有管理员权限，我不能代你执行这个运维写操作，也不能发起管理员确认卡。",
        "",
    ]
    if state.observation and state.observation.summary:
        lines.extend([
            "已基于只读权限完成的信息：",
            state.observation.summary,
            "",
        ])
    lines.extend([
        "重启、删除、扩缩容或节点变更等操作需要由管理员在确认风险后执行。",
        "建议联系管理员，并说明目标对象、期望操作、触发原因和当前异常现象。",
    ])
    return "\n".join(lines)


def _has_tool_record(state: MASState, tool_name: str) -> bool:
    return any(record.tool_name == tool_name for record in state.tool_records)


def _latest_tool_result(state: MASState, tool_name: str) -> dict[str, Any] | None:
    for record in reversed(state.tool_records):
        if record.tool_name == tool_name and isinstance(record.result, dict):
            result = record.result.get("result", record.result)
            return result if isinstance(result, dict) else record.result
    return None


def _build_final_continuation_payload(
    state: MASState,
    *,
    extra: dict[str, Any] | None = None,
) -> dict[str, Any] | None:
    payload = dict(extra or {})
    if state.observation or state.plan or state.execution or state.actions or state.tool_records:
        payload["workflow"] = state.build_workflow_checkpoint()
    return payload or None


def _build_cluster_health_noop_answer(state: MASState) -> str:
    result = _latest_tool_result(state, "get_cluster_health_report") or {}
    overall_status = str(result.get("overall_status") or "").strip().lower()
    nodes = result.get("nodes") if isinstance(result.get("nodes"), dict) else {}
    jobs = result.get("jobs") if isinstance(result.get("jobs"), dict) else {}
    alerts = result.get("alerts") if isinstance(result.get("alerts"), list) else []
    if overall_status != "healthy":
        return ""
    if alerts:
        return ""

    total_nodes = nodes.get("total", "未知")
    ready_nodes = nodes.get("ready", "未知")
    pending_jobs = jobs.get("pending", "未知")
    running_jobs = jobs.get("running", "未知")
    failed_last_hour = jobs.get("failed_last_1h", jobs.get("failed_last_hour", "未知"))

    return (
        f"结论：集群整体 healthy，当前没有告警，无需立即处理，也暂无紧急动作。\n\n"
        "证据：\n"
        f"- 节点侧：{ready_nodes}/{total_nodes} 个节点处于 Ready。\n"
        f"- 作业侧：当前运行中 {running_jobs} 个，Pending {pending_jobs} 个，过去 1 小时失败作业 {failed_last_hour} 个。\n"
        "- 告警侧：当前没有告警。\n\n"
        "建议：继续例行巡检即可；如果只是确认整体状态，当前无需额外处理。"
    )


def _looks_like_prometheus_storage_confirmation_request(user_message: str) -> bool:
    normalized = _normalized_text(user_message)
    if "prometheus" not in normalized:
        return False
    storage_tokens = ("存储", "空间", "pvc", "volume", "磁盘", "快满", "满了", "no data")
    metric_tokens = ("指标", "prometheus query", "promql", "确认", "query")
    return any(token in normalized for token in storage_tokens) and any(
        token in normalized for token in metric_tokens
    )


def _looks_like_cleanup_discovery_request(user_message: str) -> bool:
    normalized = _normalized_text(user_message)
    if not normalized:
        return False
    cleanup_tokens = (
        "清理",
        "可清理",
        "释放资源",
        "回收资源",
        "资源浪费",
        "闲置",
        "低利用率",
        "长期运行",
        "不用的作业",
    )
    job_tokens = ("作业", "job", "jobs", "gpu", "资源")
    return any(token in normalized for token in cleanup_tokens) and any(
        token in normalized for token in job_tokens
    )


def _build_prometheus_storage_metric_answer(result: dict[str, Any]) -> str:
    payload = result.get("result", result) if isinstance(result, dict) else {}
    if not isinstance(payload, dict):
        return ""
    series = payload.get("series") if isinstance(payload.get("series"), list) else []
    if not series:
        return ""

    rows: list[tuple[str, float]] = []
    for item in series[:4]:
        if not isinstance(item, dict):
            continue
        metric = item.get("metric") if isinstance(item.get("metric"), dict) else {}
        pvc = str(metric.get("persistentvolumeclaim") or metric.get("pvc") or "").strip()
        try:
            value = float(item.get("value"))
        except (TypeError, ValueError):
            continue
        rows.append((pvc or "unknown-pvc", value))
    if not rows:
        return ""

    detail_lines = [
        f"- `{pvc}` 使用率约 {value * 100:.1f}%"
        for pvc, value in rows
    ]
    max_ratio = max(value for _, value in rows)
    return (
        "结论：Prometheus 的 PVC 存储确实快满了，已经接近导致 TSDB 写入失败和 no data 的风险线。\n\n"
        "证据：\n"
        + "\n".join(detail_lines)
        + "\n\n建议：优先扩容 PVC，或调整 retention / 清理历史数据；处理后继续观察 Prometheus 是否恢复稳定写入。"
    )


def _looks_like_queue_fairness_query(user_message: str) -> bool:
    normalized = _normalized_text(user_message)
    if not normalized:
        return False
    fairness_tokens = ("不公平", "先跑", "插队", "优先级", "排队比我晚", "为什么他比我晚")
    queue_tokens = ("排队", "调度", "running", "pending")
    return any(token in normalized for token in fairness_tokens) and any(
        token in normalized for token in queue_tokens
    )


def _build_queue_fairness_answer(state: MASState) -> str:
    queue_result = _latest_tool_result(state, "analyze_queue_status") or {}
    quota_result = _latest_tool_result(state, "check_quota") or {}
    if not queue_result or not quota_result:
        return ""

    priority_explanation = (
        queue_result.get("priority_explanation")
        if isinstance(queue_result.get("priority_explanation"), dict)
        else {}
    )
    if len(priority_explanation) < 2:
        return ""

    items = list(priority_explanation.items())[:2]
    first_name, first_info = items[0]
    second_name, second_info = items[1]
    if not isinstance(first_info, dict) or not isinstance(second_info, dict):
        return ""

    first_priority = first_info.get("priority", "未知")
    second_priority = second_info.get("priority", "未知")
    scheduling_policy = str(queue_result.get("scheduling_policy") or "WeightedFairShare + Priority").strip()
    fairness_notes = str(queue_result.get("fairness_notes") or "").strip()
    quota_reset = str(quota_result.get("quota_reset_time") or "").strip()
    used_today = quota_result.get("gpu_used_today", quota_result.get("gpu_used", "未知"))
    quota_total = quota_result.get("gpu_quota_daily", quota_result.get("gpu_quota", "未知"))

    return (
        "结论：当前调度行为是公平的，属于 WeightedFairShare + Priority 正常生效，并不是系统异常。\n\n"
        "证据：\n"
        f"- 对方优先级更高（{second_priority} vs {first_priority}）。\n"
        f"- 您今日 GPU 使用超配额：当前已用 {used_today}，配额上限 {quota_total}。\n"
        f"- 超额账户被自动降低优先级：{fairness_notes or '调度器会对超额账户降权，以保证公平调度。'}\n"
        f"- 当前策略按 `{scheduling_policy}` 工作，而不是只按提交时间先后。\n\n"
        "建议：\n"
        "- 可申请提升优先级。\n"
        f"- 明天配额重置后调度优先级恢复：{quota_reset or '到下一次配额重置窗口后再观察'}。\n"
        "- 后续尽量避免长期超额使用 GPU，以免持续被公平调度降权。"
    )


def _looks_like_submission_validation_request(
    user_message: str,
    page_context: dict[str, Any],
) -> bool:
    normalized = _normalized_text(user_message)
    page_url = _normalized_text(page_context.get("url") or page_context.get("route"))
    if "/jobs/new" not in page_url and "job_create" not in page_url:
        return False
    if not any(token in normalized for token in ("能不能提交", "配置", "提交前", "帮我看看")):
        return False
    if any(token in normalized for token in ("立即调度", "马上跑", "现在哪台有空", "实时容量", "capacity")):
        return False
    return True


def _extract_requested_gpu_count(user_message: str) -> int | None:
    normalized = str(user_message or "")
    patterns = (
        r"(\d+)\s*张\s*[A-Za-z0-9-]*",
        r"(\d+)\s*卡",
        r"gpu\s*[=:]?\s*(\d+)",
    )
    for pattern in patterns:
        match = re.search(pattern, normalized, flags=re.IGNORECASE)
        if not match:
            continue
        try:
            value = int(match.group(1))
        except (TypeError, ValueError):
            continue
        if value > 0:
            return value
    return None


def _build_submission_validation_answer(state: MASState, user_message: str) -> str:
    quota_result = _latest_tool_result(state, "check_quota") or {}
    image_result = _latest_tool_result(state, "list_available_images") or {}
    if not quota_result or not image_result:
        return ""

    requested_gpu_count = _extract_requested_gpu_count(user_message)
    requested_gpu_model = _extract_requested_gpu_model(user_message).upper()
    requested_gpu_model = "V100-32G" if requested_gpu_model == "V100" else requested_gpu_model
    breakdown = quota_result.get("gpu_breakdown") if isinstance(quota_result.get("gpu_breakdown"), list) else []
    requested_remaining = None
    alternative_models: list[str] = []
    for item in breakdown:
        if not isinstance(item, dict):
            continue
        model = str(item.get("gpu_model") or "").strip()
        remaining = item.get("remaining")
        if requested_gpu_model and model.upper().startswith(requested_gpu_model.split("-")[0]):
            requested_remaining = remaining
        elif model and int(item.get("remaining") or 0) > 0:
            alternative_models.append(model)

    exact_match = image_result.get("exact_match") if isinstance(image_result.get("exact_match"), dict) else {}
    closest_matches = exact_match.get("closest_matches") if isinstance(exact_match.get("closest_matches"), list) else []
    recommended_image = str(closest_matches[0] or "").strip() if closest_matches else ""
    memory_remaining = str(quota_result.get("memory_remaining") or "").strip()

    if requested_gpu_count is None or requested_remaining is None:
        return ""

    alternatives_text = " 或 ".join(alternative_models[:2]) if alternative_models else "其他可用 GPU 型号"
    return (
        "结论：当前这份配置不能直接提交，主要卡在 V100 配额不足；内存本身是合理的。\n\n"
        "证据：\n"
        f"- 你请求的是 {requested_gpu_count} 张 {requested_gpu_model}，但该型号当前只剩 {requested_remaining} 张配额，无法按原配置提交。\n"
        f"- 镜像名 `pytorch:2.1.0` 不完整，当前更合适的可用镜像是 `{recommended_image}`。\n"
        f"- 内存剩余 {memory_remaining}，所以 64G 内存配置合理，不是阻塞点。\n\n"
        "建议：\n"
        f"- 把 GPU 数量先降到 {requested_remaining} 张 {requested_gpu_model}；\n"
        f"- 或整体改为 {alternatives_text} 之一，再重新评估资源配置；\n"
        f"- 同时把镜像改成 `{recommended_image}`。\n"
    )


def _candidate_failure_hint(job: dict[str, Any]) -> str:
    exit_code = str(job.get("exitCode") or "").strip()
    failure_reason = _normalized_text(job.get("failureReason"))
    if exit_code == "137" or "oom" in failure_reason:
        return "初步看更像 OOM / 显存或内存不足"
    if "inputdataerror" in failure_reason or "inputdata" in failure_reason:
        return "初步看更像输入数据或数据路径问题"
    if "usercodeerror" in failure_reason:
        return "初步看更像训练脚本或参数本身报错"
    if exit_code == "1":
        return "目前只看到 exit_code=1，具体根因还要继续看日志或事件"
    return ""


def _build_health_overview_answer(state: MASState) -> str:
    result = _latest_tool_result(state, "get_health_overview") or {}
    total_jobs = result.get("totalJobs", result.get("total_jobs", "未知"))
    status_count = result.get("statusCount") if isinstance(result.get("statusCount"), dict) else {}
    completed = status_count.get("Completed", status_count.get("completed", 0))
    running = status_count.get("Running", status_count.get("running", 0))
    failed = status_count.get("Failed", status_count.get("failed", 0))
    failure_rate = result.get("failureRatePct", result.get("failure_rate_pct", "未知"))
    recent_failures = result.get("recentFailures") if isinstance(result.get("recentFailures"), list) else []
    failure = recent_failures[0] if recent_failures and isinstance(recent_failures[0], dict) else {}
    failed_job = str(failure.get("jobName") or failure.get("job_name") or "那个失败作业").strip()
    reason = str(failure.get("reason") or "").strip()
    reason_text = f"，原因是 `{reason}`" if reason else ""
    return (
        "结论：最近 7 天整体健康，没有明显异常需要立即处理。\n\n"
        "关键细节：\n"
        f"- 过去 7 天共有 {total_jobs} 个作业，其中 {completed} 个已完成、{running} 个运行中、{failed} 个失败。\n"
        f"- 当前只有 {failed} 个失败作业，失败率约 {failure_rate}%，不属于大面积异常。\n"
        f"- 失败作业是 `{failed_job}`{reason_text}。\n\n"
        "建议动作：\n"
        "- 无需立即处理集群或批量作业，继续观察整体趋势即可。\n"
        f"- 如果要追查可以看那个失败作业 `{failed_job}` 的事件和日志，再决定是否修正镜像、配置或重新提交。"
    )


def _first_recommended_action_item(report: dict[str, Any], action_keyword: str) -> dict[str, Any]:
    actions = report.get("recommended_actions") if isinstance(report.get("recommended_actions"), list) else []
    for action in actions:
        if not isinstance(action, dict):
            continue
        action_name = str(action.get("action") or "").strip()
        if action_keyword not in action_name:
            continue
        items = action.get("items") if isinstance(action.get("items"), list) else []
        for item in items:
            if isinstance(item, dict):
                return item
    return {}


def _build_admin_ops_report_answer(state: MASState) -> str:
    report = _latest_tool_result(state, "get_admin_ops_report") or {}
    overview = report.get("overview") if isinstance(report.get("overview"), dict) else {}
    idle_summary = report.get("idle_summary") if isinstance(report.get("idle_summary"), dict) else {}
    if not overview or not idle_summary:
        return ""
    lookback_days = report.get("lookback_days", 7)
    total_jobs = overview.get("total_jobs", "未知")
    success_jobs = overview.get("success_jobs", "未知")
    failed_jobs = overview.get("failed_jobs", "未知")
    running_jobs = overview.get("running_jobs", "未知")
    pending_jobs = overview.get("pending_jobs", "未知")
    success_rate = overview.get("success_rate")
    failure_rate = overview.get("failure_rate")
    success_pct = f"{float(success_rate) * 100:.1f}%" if isinstance(success_rate, (int, float)) else "未知"
    failure_pct = f"{float(failure_rate) * 100:.1f}%" if isinstance(failure_rate, (int, float)) else "未知"
    idle_count = idle_summary.get("idle_job_count", "未知")
    waste_hours = idle_summary.get("estimated_gpu_waste_hours", "未知")

    failure_item = _first_recommended_action_item(report, "失败")
    resource_item = _first_recommended_action_item(report, "资源差异")
    idle_item = _first_recommended_action_item(report, "低利用率")

    lines = [
        f"结论：最近 {lookback_days} 天共有 {total_jobs} 个作业，成功 {success_jobs} 个、失败 {failed_jobs} 个，成功率 {success_pct}、失败率 {failure_pct}；存在失败热点、成功作业资源差异和闲置 GPU 浪费。",
        "",
        "证据：",
        f"- 总体概览：总作业 {total_jobs} 个，成功 {success_jobs} 个，失败 {failed_jobs} 个，运行中 {running_jobs} 个，排队 {pending_jobs} 个。",
        f"- 闲置与浪费：存在 {idle_count} 个闲置作业，预估 GPU 浪费 {waste_hours} 小时。",
    ]
    if failure_item:
        lines.append(
            "- 失败热点示例："
            f"{failure_item.get('job_name')}（{failure_item.get('user')}）"
            f"，持续 {failure_item.get('duration')}，申请 {failure_item.get('gpu_requested')} GPU，实际使用 {failure_item.get('gpu_actual')} GPU。"
        )
    if resource_item:
        lines.append(
            "- 成功作业资源差异示例："
            f"{resource_item.get('job_name')}（{resource_item.get('user')}）"
            f"，GPU 利用率 {resource_item.get('gpu_util')}，申请 {resource_item.get('gpu_requested')} GPU，实际使用 {resource_item.get('gpu_actual')} GPU。"
        )
    if idle_item:
        lines.append(
            "- 低利用率作业示例："
            f"{idle_item.get('job_name')}（{idle_item.get('user')}）"
            f"，GPU 利用率 {idle_item.get('gpu_util')}，持续 {idle_item.get('duration')}。"
        )
    lines.extend(
        [
            "",
            "建议：",
            "- 建议关注失败热点，优先排查失败作业的内存、镜像、调度和启动阶段问题。",
            "- 建议复盘成功作业的资源差异，对 GPU 申请明显高于实际使用的作业下调资源配置。",
            f"- 建议处理低利用率或闲置作业，优先回收可节省的 {waste_hours} GPU 小时。",
            "- 建议每周固定复盘失败热点与资源差异，形成常态化治理。"
        ]
    )
    return "\n".join(lines)


def _should_run_terminal_verifier(state: MASState) -> bool:
    """Run verifier once at the end of materialized write workflows."""
    terminal_actions = _terminal_write_action_history(state)
    if not terminal_actions:
        return False
    if any(_action_status(item.get("status")) in {"error", "failed", "rejected"} for item in terminal_actions):
        return True
    if len(terminal_actions) > 1:
        return True
    if state.plan and state.plan.risk in {"medium", "high"}:
        return True
    return any(
        str(item.get("tool_name") or "").strip() in CONFIRM_TOOL_NAMES
        for item in terminal_actions
    )


def _determine_verifier_gate(
    *,
    state: MASState,
    routing: RoutingDecision,
    current_progress_signature: str,
    last_verification_signature: str,
    last_verification_result: RoleExecutionResult | None,
) -> VerifierGateDecision:
    """Decide whether the expensive verifier stage is justified.

    Verifier is meant to challenge write/action outcomes, not to become a
    default second opinion for every read-only investigation.
    """
    has_state_to_check = bool(
        state.tool_records
        or state.action_history
        or state.execution
        or state.observation
        or state.pending_confirmations
    )
    if not has_state_to_check:
        return VerifierGateDecision(False, "observe", "no_evidence")

    verification_verdict = _extract_verification_verdict(last_verification_result)
    if current_progress_signature == last_verification_signature and verification_verdict:
        if verification_verdict == "missing_evidence":
            frontier = state.action_frontier()
            return VerifierGateDecision(
                False,
                "act" if frontier else "observe",
                "same_state_missing_evidence",
            )
        return VerifierGateDecision(False, "finalize", "same_state_already_verified")

    if _has_awaiting_write_confirmation(state):
        return VerifierGateDecision(False, "finalize", "awaiting_confirmation")

    if _has_terminal_write_result(state):
        if _should_run_terminal_verifier(state):
            return VerifierGateDecision(True, "verify", "terminal_write_result")
        return VerifierGateDecision(False, "finalize", "terminal_write_low_risk")

    if _has_write_intent(routing):
        frontier = state.action_frontier()
        return VerifierGateDecision(
            False,
            "act" if frontier or state.loop_round < state.runtime_config.lead_max_rounds else "finalize",
            "write_intent_without_action_result",
        )

    if state.observation and state.observation.open_questions:
        return VerifierGateDecision(
            False,
            "observe" if state.loop_round < state.runtime_config.lead_max_rounds else "finalize",
            "read_only_missing_evidence",
        )

    return VerifierGateDecision(False, "finalize", "read_only_sufficient_evidence")


def _latest_controller_stage(state: MASState) -> str:
    if not state.controller_trace:
        return ""
    latest = state.controller_trace[-1]
    if not isinstance(latest, dict):
        return ""
    return str(latest.get("stage") or "").strip().lower()


def _extract_evidence_from_tool_records(tool_records: list[Any]) -> list[dict[str, Any]]:
    """Convert MASState.tool_records into legacy evidence dicts."""
    return [
        {
            "tool_name": r.tool_name,
            "tool_args": r.tool_args,
            "result": r.result,
        }
        for r in tool_records
    ]


# ---------------------------------------------------------------------------
# Job collection and candidate helpers
# ---------------------------------------------------------------------------

def _dedupe_jobs(jobs: list[dict[str, Any]]) -> list[dict[str, Any]]:
    deduped: list[dict[str, Any]] = []
    seen: set[str] = set()
    for job in jobs:
        job_name = str(job.get("jobName") or "").strip().lower()
        if not job_name or job_name in seen:
            continue
        seen.add(job_name)
        deduped.append(job)
    return deduped


def _extract_requested_job_types(
    *,
    user_message: str,
    page_context: dict[str, Any],
) -> list[str]:
    normalized_message = _normalized_text(user_message)
    route_hint = _normalized_text(page_context.get("route") or page_context.get("url") or "")
    requested: list[str] = []

    def add(job_type: str) -> None:
        if job_type and job_type not in requested:
            requested.append(job_type)

    if "custom" in normalized_message or "自定义" in str(user_message or "") or "/jobs/custom" in route_hint:
        add("custom")
    if "jupyter" in normalized_message or "notebook" in normalized_message:
        add("jupyter")
    if "webide" in normalized_message or "web-ide" in normalized_message:
        add("webide")

    return requested


def _job_matches_requested_types(job: dict[str, Any], requested_job_types: set[str]) -> bool:
    if not requested_job_types:
        return True
    job_type = str(job.get("jobType") or job.get("job_type") or "").strip().lower()
    return bool(job_type) and job_type in requested_job_types


def _user_requests_latest_job(user_message: str) -> bool:
    normalized = _normalized_text(user_message)
    return any(token in normalized for token in ("最新", "最近", "latest", "newest", "most recent"))


def _parse_creation_timestamp(value: Any) -> tuple[int, str]:
    text = str(value or "").strip()
    if not text:
        return 0, ""
    normalized = text.replace("Z", "+00:00")
    try:
        parsed = datetime.fromisoformat(normalized)
        return 1, parsed.isoformat()
    except ValueError:
        return 0, text


def _sort_jobs_by_creation_desc(jobs: list[dict[str, Any]]) -> list[dict[str, Any]]:
    return sorted(
        jobs,
        key=lambda job: _parse_creation_timestamp(
            job.get("creationTimestamp") or job.get("completionTime")
        ),
        reverse=True,
    )


def _collect_jobs_from_evidence(
    evidence: list[dict[str, Any]],
    *,
    status_filter: set[str] | None = None,
    job_type_filter: set[str] | None = None,
) -> list[dict[str, Any]]:
    jobs: list[dict[str, Any]] = []
    normalized_status_filter = {status.lower() for status in (status_filter or set())}
    normalized_job_type_filter = {job_type.lower() for job_type in (job_type_filter or set())}
    for entry in evidence:
        if not isinstance(entry, dict):
            continue
        if entry.get("tool_name") != "list_user_jobs":
            continue
        result = entry.get("result") or {}
        payload = result.get("result", result) if isinstance(result, dict) else {}
        raw_jobs = payload.get("jobs") if isinstance(payload, dict) else None
        if not isinstance(raw_jobs, list):
            continue
        for job in raw_jobs:
            if not isinstance(job, dict):
                continue
            status = str(job.get("status") or "").strip()
            if normalized_status_filter and status.lower() not in normalized_status_filter:
                continue
            if normalized_job_type_filter and not _job_matches_requested_types(job, normalized_job_type_filter):
                continue
            jobs.append(
                {
                    "jobName": str(job.get("jobName") or job.get("job_name") or "").strip(),
                    "name": str(job.get("name") or "").strip(),
                    "status": status,
                    "jobType": str(job.get("jobType") or job.get("job_type") or "").strip(),
                    "creationTimestamp": str(job.get("creationTimestamp") or job.get("creation_time") or "").strip(),
                    "completionTime": str(job.get("completionTime") or job.get("completion_time") or "").strip(),
                    "exitCode": job.get("exitCode") if job.get("exitCode") is not None else job.get("exit_code"),
                    "failureReason": str(job.get("failureReason") or job.get("failure_reason") or "").strip(),
                    "gpuModel": str(job.get("gpuModel") or job.get("gpu_model") or "").strip(),
                }
            )
    return _dedupe_jobs(jobs)


def _collect_audit_action_jobs_from_evidence(
    evidence: list[dict[str, Any]],
    *,
    action_type: str | None = None,
) -> list[dict[str, Any]]:
    jobs: list[dict[str, Any]] = []
    normalized_action = _normalized_text(action_type)
    for entry in evidence:
        if not isinstance(entry, dict) or entry.get("tool_name") != "list_audit_items":
            continue
        result = entry.get("result") or {}
        payload = result.get("result", result) if isinstance(result, dict) else {}
        raw_items = payload.get("items") if isinstance(payload, dict) else None
        if not isinstance(raw_items, list):
            continue
        for item in raw_items:
            if not isinstance(item, dict):
                continue
            item_action = _normalized_text(item.get("action_type"))
            if normalized_action and item_action and item_action != normalized_action:
                continue
            job_name = str(item.get("job_name") or item.get("jobName") or "").strip()
            if not job_name:
                continue
            jobs.append(
                {
                    "jobName": job_name,
                    "name": str(item.get("name") or "").strip(),
                    "status": str(item.get("status") or "").strip(),
                    "jobType": str(item.get("job_type") or item.get("jobType") or "").strip(),
                    "creationTimestamp": str(item.get("creationTimestamp") or "").strip(),
                }
            )
    return _dedupe_jobs(jobs)


def _format_job_candidates(jobs: list[dict[str, Any]], limit: int = 8) -> str:
    lines: list[str] = []
    for index, job in enumerate(jobs[:limit], start=1):
        display_name = str(job.get("name") or "").strip()
        display_suffix = f" / {display_name}" if display_name else ""
        status = str(job.get("status") or "").strip() or "unknown"
        details: list[str] = [status]
        exit_code = job.get("exitCode")
        if exit_code not in (None, ""):
            details.append(f"exit_code={exit_code}")
        failure_reason = str(job.get("failureReason") or "").strip()
        if failure_reason:
            details.append(failure_reason)
        gpu_model = str(job.get("gpuModel") or "").strip()
        if gpu_model:
            details.append(gpu_model)
        completion_time = str(job.get("completionTime") or job.get("creationTimestamp") or "").strip()
        if completion_time:
            details.append(completion_time)
        lines.append(f"{index}. {job.get('jobName')}{display_suffix} ({', '.join(details)})")
    return "\n".join(lines)


def _looks_like_ambiguous_failure_query(user_message: str, routing: RoutingDecision) -> bool:
    if routing.requested_action or routing.targets.job_name:
        return False
    normalized = _normalized_text(user_message)
    if not normalized:
        return False
    failure_tokens = ("失败", "报错", "failed", "fail", "error")
    job_tokens = ("作业", "job", "任务")
    return any(token in normalized for token in failure_tokens) and any(
        token in normalized for token in job_tokens
    )


def _failed_job_candidates_for_clarification(
    *,
    user_message: str,
    routing: RoutingDecision,
    evidence: list[dict[str, Any]],
) -> list[dict[str, Any]]:
    if not _looks_like_ambiguous_failure_query(user_message, routing):
        return []
    failed_jobs = _collect_jobs_from_evidence(evidence, status_filter={"failed"})
    return failed_jobs if len(failed_jobs) > 1 else []


def _build_diagnostic_clarification_answer(candidate_jobs: list[dict[str, Any]]) -> str:
    candidates_text = _format_job_candidates(candidate_jobs)
    newest_job_name = str((candidate_jobs[0] or {}).get("jobName") or "").strip() if candidate_jobs else ""
    newest_hint = (
        f"如果你说的是最新失败作业，当前最新的是 `{newest_job_name}`。\n\n"
        if newest_job_name
        else ""
    )
    hint_lines = []
    for job in candidate_jobs[:3]:
        job_name = str(job.get("jobName") or "").strip()
        hint = _candidate_failure_hint(job)
        if job_name and hint:
            hint_lines.append(f"- `{job_name}`：{hint}")
    hints_block = (
        "\n初步线索：\n" + "\n".join(hint_lines) + "\n"
        if hint_lines
        else ""
    )
    return (
        "结论：你最近有多个失败作业，需要先确认具体是哪一个，我再继续诊断。\n\n"
        f"证据：最近失败作业如下：\n{candidates_text}\n"
        f"{hints_block}\n"
        f"{newest_hint}"
        "建议：请直接回复要诊断的 jobName（例如列表中的第一个），"
        "也可以回复“最新的失败作业”。我会基于该作业的详情、事件或日志继续做根因分析。"
    )


def _candidate_status_filter_for_action(action_intent: str | None) -> set[str] | None:
    if action_intent == "resubmit":
        return {"failed"}
    if action_intent == "stop":
        return {"running", "pending", "inqueue", "prequeue"}
    return None


def _build_action_clarification_answer(
    *,
    action_intent: str,
    candidate_jobs: list[dict[str, Any]],
) -> str:
    action_label = {
        "resubmit": "重新提交",
        "stop": "停止",
        "delete": "删除",
    }.get(action_intent, "处理")
    candidate_label = {
        "resubmit": "失败作业",
        "stop": "候选作业",
        "delete": "候选作业",
    }.get(action_intent, "候选作业")
    if not candidate_jobs:
        if action_intent == "resubmit":
            return (
                f"我没有查到当前账户下可用于{action_label}的失败作业。"
                "你可以先确认作业是否真的处于 Failed，或者直接给我一个明确的 jobName。"
            )
        return f"我没有查到当前账户下可用于{action_label}的候选作业。你可以直接给我一个明确的 jobName。"

    candidates_text = _format_job_candidates(candidate_jobs)
    return (
        f"我找到多条候选{candidate_label}。为了避免误操作，请先明确你要{action_label}哪一个。\n\n"
        f"{candidates_text}\n\n"
        "请直接回复一个具体的 jobName；如果你想处理全部候选作业，也可以直接回复\u201c全部\u201d。"
    )


def _build_job_selection_continuation(
    *,
    action_intent: str,
    candidate_jobs: list[dict[str, Any]],
    requested_all_scope: bool,
) -> dict[str, Any] | None:
    if not candidate_jobs:
        return None

    normalized_candidates: list[dict[str, Any]] = []
    for job in candidate_jobs[:12]:
        if not isinstance(job, dict):
            continue
        job_name = str(job.get("jobName") or "").strip().lower()
        if not job_name:
            continue
        normalized_candidates.append(
            {
                "job_name": job_name,
                "display_name": str(job.get("name") or "").strip(),
                "status": str(job.get("status") or "").strip(),
                "job_type": str(job.get("jobType") or "").strip(),
                "creation_timestamp": str(job.get("creationTimestamp") or "").strip(),
            }
        )

    if not normalized_candidates:
        return None

    return {
        "kind": "job_selection",
        "action_intent": action_intent,
        "requested_scope": "all" if requested_all_scope else "single",
        "candidate_jobs": normalized_candidates,
    }


# ---------------------------------------------------------------------------
# Forced / fallback tool selection helpers
# ---------------------------------------------------------------------------

def _forced_exploration_tools(
    *,
    action_intent: str | None,
    resolved_job_name: str | None,
    requested_job_types: list[str] | None,
    enabled_tools: list[str],
) -> list[tuple[str, dict[str, Any]]]:
    if not action_intent:
        return []

    forced: list[tuple[str, dict[str, Any]]] = []
    enabled = set(enabled_tools)

    def add(tool_name: str, args: dict[str, Any]) -> None:
        if tool_name in enabled:
            forced.append((tool_name, args))

    if resolved_job_name:
        add("get_job_detail", {"job_name": resolved_job_name})
        return forced

    if action_intent == "resubmit":
        args: dict[str, Any] = {"statuses": ["Failed"], "days": 30, "limit": 12}
        if requested_job_types:
            args["job_types"] = requested_job_types
        add("list_user_jobs", args)
        add("get_health_overview", {"days": 7})
        return forced

    if action_intent == "stop" and "list_audit_items" in enabled:
        add("list_audit_items", {"action_type": "stop", "handled": "false", "limit": 20})
        return forced

    args = {"days": 30, "limit": 12}
    if requested_job_types:
        args["job_types"] = requested_job_types
    add("list_user_jobs", args)
    return forced


def _fallback_read_tools_from_context(
    *,
    user_message: str,
    page_context: dict[str, Any],
    enabled_tools: list[str],
) -> list[tuple[str, dict[str, Any]]]:
    selected: list[tuple[str, dict[str, Any]]] = []
    enabled = set(enabled_tools)
    job_name = str(page_context.get("job_name") or page_context.get("jobName") or "").strip().lower()
    node_name = str(page_context.get("node_name") or page_context.get("nodeName") or "").strip()
    route_hint = str(page_context.get("route") or page_context.get("url") or "").strip().lower()
    requested_job_types = _extract_requested_job_types(
        user_message=user_message,
        page_context=page_context,
    )

    def add(tool_name: str, args: dict[str, Any]) -> None:
        if tool_name not in enabled or tool_name not in READ_ONLY_TOOL_NAMES:
            return
        if any(existing_tool == tool_name and existing_args == args for existing_tool, existing_args in selected):
            return
        selected.append((tool_name, args))

    if job_name:
        add("get_job_detail", {"job_name": job_name})
        if "query_job_metrics" in enabled:
            pseudo_routing = RoutingDecision()
            pseudo_routing.targets.job_name = job_name
            if _looks_like_job_health_noop_request(
                user_message=user_message,
                routing=pseudo_routing,
                page_context=page_context,
            ):
                add("query_job_metrics", {"job_name": job_name})
        return selected

    if node_name:
        add("get_node_detail", {"node_name": node_name})
        return selected

    if route_hint.startswith("/admin"):
        add("get_cluster_health_overview", {"days": 7})
        add("list_cluster_jobs", {"days": 7, "limit": 8})
        return selected

    user_jobs_args: dict[str, Any] = {"days": 30, "limit": 8}
    if requested_job_types:
        user_jobs_args["job_types"] = requested_job_types
    add("list_user_jobs", user_jobs_args)
    add("get_health_overview", {"days": 7})
    return selected


def _preferred_tools_for_first_observe(
    *,
    routing: RoutingDecision,
    page_context: dict[str, Any],
    enabled_tools: list[str],
    user_message: str,
) -> list[tuple[str, dict[str, Any]]]:
    """Choose preferred initial exploration tools based on page context and intent.

    Replaces the old scenario-specific explore tool selection with a unified
    context-driven approach.
    """
    enabled = set(enabled_tools)
    preferred: list[tuple[str, dict[str, Any]]] = []

    # Submission-like request (create job)
    if routing.requested_action == "create":
        if "recommend_training_images" in enabled:
            preferred.append((
                "recommend_training_images",
                {
                    "task_description": "LLM 大语言模型训练",
                    "framework": "pytorch",
                    "limit": 3,
                },
            ))
        if "list_available_gpu_models" in enabled:
            preferred.append(("list_available_gpu_models", {"limit": 10}))
        return preferred

    # Write operation with known or unknown target
    if routing.requested_action:
        requested_job_types = _extract_requested_job_types(
            user_message=user_message,
            page_context=page_context,
        )
        return _forced_exploration_tools(
            action_intent=routing.requested_action,
            resolved_job_name=routing.targets.job_name,
            requested_job_types=requested_job_types,
            enabled_tools=enabled_tools,
        )

    focus_hints = extract_focus_hints(page_context, user_message)
    node_name = str(focus_hints.get("node_name") or "").strip()
    pvc_name = str(focus_hints.get("pvc_name") or "").strip()
    job_name = (
        str(routing.targets.job_name or "").strip()
        or str(focus_hints.get("job_name") or "").strip()
        or str(page_context.get("job_name") or "").strip()
    )

    if _looks_like_cleanup_discovery_request(user_message) and "detect_idle_jobs" in enabled:
        preferred.append(("detect_idle_jobs", {"gpu_threshold": 5, "hours": 24}))
        if "list_user_jobs" in enabled:
            preferred.append(("list_user_jobs", {"statuses": ["Succeeded", "Failed"], "limit": 50}))
        if "get_realtime_capacity" in enabled:
            preferred.append(("get_realtime_capacity", {}))
        return preferred

    if node_name and "get_node_detail" in enabled:
        preferred.append(("get_node_detail", {"node_name": node_name}))

    if bool(focus_hints.get("mentions_storage")):
        if pvc_name and "get_pvc_detail" in enabled:
            preferred.append(("get_pvc_detail", {"pvc_name": pvc_name}))
        elif "list_storage_pvcs" in enabled:
            preferred.append(("list_storage_pvcs", {"limit": 10}))

    if bool(focus_hints.get("mentions_distributed_network")):
        if job_name and "diagnose_distributed_job_network" in enabled:
            preferred.append((
                "diagnose_distributed_job_network",
                {
                    "job_name": job_name,
                    "tail_lines": 200,
                    "max_log_matches": 30,
                },
            ))
        elif node_name and "get_node_network_summary" in enabled:
            preferred.append(("get_node_network_summary", {"node_name": node_name}))
        elif "get_node_network_summary" in enabled:
            preferred.append(("get_node_network_summary", {"limit": 10}))

    if (
        job_name
        and _looks_like_job_health_noop_request(
            user_message=user_message,
            routing=routing,
            page_context=page_context,
        )
    ):
        if "get_job_detail" in enabled:
            preferred.append(("get_job_detail", {"job_name": job_name}))
        if "query_job_metrics" in enabled:
            preferred.append(("query_job_metrics", {"job_name": job_name}))

    if preferred:
        return preferred

    # Generic fallback from page context
    return _fallback_read_tools_from_context(
        user_message=user_message,
        page_context=page_context,
        enabled_tools=enabled_tools,
    )


# ---------------------------------------------------------------------------
# Action planning helpers
# ---------------------------------------------------------------------------

def _pending_action_dicts(actions: list[MultiAgentActionItem]) -> list[dict[str, Any]]:
    return [action.to_dict() for action in actions if action.status in {"pending", "awaiting_confirmation"}]


def _merge_action_proposals(
    actions: list[MultiAgentActionItem],
    proposals: list[dict[str, Any]],
) -> list[MultiAgentActionItem]:
    existing_signatures = {
        _tool_signature(action.tool_name, action.tool_args)
        for action in actions
        if action.status != "skipped"
    }
    created: list[tuple[MultiAgentActionItem, dict[str, Any]]] = []
    index_to_action_id: dict[int, str] = {}

    for index, proposal in enumerate(proposals, start=1):
        tool_name = str(proposal.get("tool_name") or "").strip()
        tool_args = proposal.get("tool_args") if isinstance(proposal.get("tool_args"), dict) else {}
        if not tool_name:
            continue
        signature = _tool_signature(tool_name, tool_args)
        if signature in existing_signatures:
            continue
        action_id = f"action-{len(actions) + len(created) + 1}"
        index_to_action_id[index] = action_id
        created.append((
            MultiAgentActionItem(
                action_id=action_id,
                tool_name=tool_name,
                tool_args=tool_args,
                title=str(proposal.get("title") or "").strip(),
                reason=str(proposal.get("reason") or "").strip(),
            ),
            proposal,
        ))
        existing_signatures.add(signature)

    finalized: list[MultiAgentActionItem] = []
    for action, proposal in created:
        depends_on_indexes = proposal.get("depends_on_indexes") if isinstance(proposal, dict) else []
        if not isinstance(depends_on_indexes, list):
            finalized.append(action)
            continue
        action.depends_on = [
            index_to_action_id[idx]
            for idx in depends_on_indexes
            if isinstance(idx, int) and idx in index_to_action_id
        ]
        finalized.append(action)

    actions.extend(finalized)
    return finalized


def _fallback_executor_actions(
    *,
    action_intent: str | None,
    resolved_job_name: str | None,
    candidate_jobs: list[dict[str, Any]],
    requested_scope: str,
    enabled_tools: list[str],
) -> list[dict[str, Any]]:
    if (
        action_intent == "stop"
        and requested_scope == "all"
        and candidate_jobs
        and "batch_stop_jobs" in enabled_tools
    ):
        job_names = [
            str(job.get("jobName") or "").strip().lower()
            for job in candidate_jobs
            if str(job.get("jobName") or "").strip()
        ]
        if job_names:
            return [
                {
                    "tool_name": "batch_stop_jobs",
                    "tool_args": {"job_names": ",".join(job_names)},
                    "title": f"batch_stop_jobs:{len(job_names)}",
                    "reason": "集合停止动作使用批量确认工具",
                    "depends_on_indexes": [],
                }
            ]

    action_to_tool = {
        "resubmit": "resubmit_job",
        "stop": "stop_job",
        "delete": "delete_job",
    }
    tool_name = action_to_tool.get(action_intent or "")
    if not tool_name or tool_name not in enabled_tools:
        return []

    if requested_scope == "all" and candidate_jobs:
        return [
            {
                "tool_name": tool_name,
                "tool_args": {"job_name": str(job.get("jobName") or "").strip().lower()},
                "title": f"{tool_name}:{str(job.get('jobName') or '').strip().lower()}",
                "reason": "结构化 scope=all，按候选作业顺序执行",
                "depends_on_indexes": [],
            }
            for job in candidate_jobs
            if str(job.get("jobName") or "").strip()
        ]

    if not resolved_job_name:
        return []

    return [
        {
            "tool_name": tool_name,
            "tool_args": {"job_name": resolved_job_name},
            "title": f"{tool_name}:{resolved_job_name}",
            "reason": "结构化单目标动作",
            "depends_on_indexes": [],
        }
    ]


def _extract_target_replicas(user_message: str) -> int | None:
    normalized = str(user_message or "").strip().lower()
    patterns = (
        r"(?:缩到|扩到|调到|调整到|改成|改为|设为|scale\s+to)\s*(\d+)",
        r"(?:replicas?|副本(?:数)?)\D{0,8}(\d+)",
        r"from\s+\d+\s+to\s+(\d+)",
        r"从\s*\d+\s*(?:个)?\s*(?:副本)?\s*(?:缩到|扩到|到|至)\s*(\d+)",
    )
    for pattern in patterns:
        match = re.search(pattern, normalized)
        if not match:
            continue
        try:
            replicas = int(match.group(1))
        except (TypeError, ValueError):
            continue
        if 0 <= replicas <= 100:
            return replicas
    return None


def _extract_workload_name(user_message: str, page_context: dict[str, Any]) -> str:
    for key in ("workload", "workload_name", "name", "deployment", "resource_name"):
        value = str(page_context.get(key) or "").strip()
        if value:
            return value
    route = str(page_context.get("url") or page_context.get("route") or "").strip("/")
    if route:
        tail = route.rsplit("/", 1)[-1].strip()
        if tail and tail not in {"workloads", "deployments", "statefulsets"}:
            return tail
    for candidate in re.findall(r"[A-Za-z0-9][A-Za-z0-9_.-]*", str(user_message or "")):
        lowered = candidate.lower()
        if lowered not in {"deployment", "statefulset", "replicas", "scale"} and "-" in candidate:
            return candidate
    return ""


def _extract_workload_kind(user_message: str, page_context: dict[str, Any]) -> str:
    raw_kind = str(page_context.get("kind") or page_context.get("workload_kind") or "").strip()
    if raw_kind in {"Deployment", "StatefulSet"}:
        return raw_kind
    normalized = str(user_message or "").strip().lower()
    if "statefulset" in normalized or "sts/" in normalized:
        return "StatefulSet"
    return "Deployment"


def _extract_workload_namespace(
    *,
    workload_name: str,
    page_context: dict[str, Any],
    evidence: list[dict[str, Any]],
) -> str:
    for key in ("namespace", "ns"):
        value = str(page_context.get(key) or "").strip()
        if value:
            return value
    normalized_name = workload_name.strip().lower()
    for entry in reversed(evidence):
        if not isinstance(entry, dict) or entry.get("tool_name") != "k8s_get_endpoints":
            continue
        result = entry.get("result") if isinstance(entry.get("result"), dict) else {}
        payload = result.get("result", result) if isinstance(result, dict) else {}
        if not isinstance(payload, dict):
            continue
        entry_name = str(payload.get("name") or "").strip().lower()
        if normalized_name and entry_name and entry_name != normalized_name:
            continue
        namespace = str(payload.get("namespace") or "").strip()
        if namespace:
            return namespace
    return ""


def _fallback_k8s_scale_actions(
    *,
    user_message: str,
    page_context: dict[str, Any],
    evidence: list[dict[str, Any]],
    enabled_tools: list[str],
) -> list[dict[str, Any]]:
    if "k8s_scale_workload" not in enabled_tools:
        return []
    replicas = _extract_target_replicas(user_message)
    workload_name = _extract_workload_name(user_message, page_context)
    if replicas is None or not workload_name:
        return []
    tool_args: dict[str, Any] = {
        "kind": _extract_workload_kind(user_message, page_context),
        "name": workload_name,
        "replicas": replicas,
    }
    namespace = _extract_workload_namespace(
        workload_name=workload_name,
        page_context=page_context,
        evidence=evidence,
    )
    if namespace:
        tool_args["namespace"] = namespace
    return [
        {
            "tool_name": "k8s_scale_workload",
            "tool_args": tool_args,
            "title": f"k8s_scale_workload:{workload_name}->{replicas}",
            "reason": "明确的 Kubernetes 工作负载扩缩容写意图，进入确认流",
            "depends_on_indexes": [],
        }
    ]


def _fallback_k8s_node_isolation_actions(
    *,
    user_message: str,
    page_context: dict[str, Any],
    evidence: list[dict[str, Any]],
    enabled_tools: list[str],
) -> list[dict[str, Any]]:
    enabled = set(enabled_tools)
    if "k8s_label_node" not in enabled or "k8s_taint_node" not in enabled:
        return []
    normalized = _normalized_text(user_message)
    if not any(token in normalized for token in ("隔离", "isolate", "isolated", "taint", "noschedule")):
        return []
    if not any(token in normalized for token in ("rdma", "roce", "网络", "通信", "异常", "critical")):
        return []
    node_name = extract_node_name(page_context, user_message)
    if not node_name:
        return []
    has_bad_rdma_evidence = False
    for entry in reversed(evidence):
        if not isinstance(entry, dict) or entry.get("tool_name") != "get_rdma_interface_status":
            continue
        result = entry.get("result") if isinstance(entry.get("result"), dict) else {}
        payload = result.get("result", result) if isinstance(result, dict) else {}
        if not isinstance(payload, dict):
            continue
        entry_node = str(payload.get("node_name") or payload.get("nodeName") or "").strip()
        if entry_node and entry_node != node_name:
            continue
        status = _normalized_text(payload.get("status"))
        recommendation = _normalized_text(payload.get("recommendation"))
        if status in {"critical", "degraded", "abnormal", "error"} or any(
            token in recommendation for token in ("isolate", "隔离", "noschedule", "taint")
        ):
            has_bad_rdma_evidence = True
            break
    if not has_bad_rdma_evidence:
        return []
    return [
        {
            "tool_name": "k8s_label_node",
            "tool_args": {
                "node_name": node_name,
                "key": "rdma.status",
                "value": "isolated",
            },
            "title": f"k8s_label_node:{node_name}",
            "reason": "用户明确要求隔离 RDMA 异常节点，先生成隔离标签确认卡",
            "depends_on_indexes": [],
        },
        {
            "tool_name": "k8s_taint_node",
            "tool_args": {
                "node_name": node_name,
                "key": "rdma-isolated",
                "effect": "NoSchedule",
            },
            "title": f"k8s_taint_node:{node_name}:NoSchedule",
            "reason": "用户明确要求隔离 RDMA 异常节点，同时生成 NoSchedule taint 确认卡",
            "depends_on_indexes": [],
        },
    ]


def _extract_recommended_image(evidence: list[dict[str, Any]]) -> str:
    for entry in reversed(evidence):
        if not isinstance(entry, dict):
            continue
        tool_name = _normalized_text(entry.get("tool_name"))
        if tool_name not in {"recommend_training_images", "list_available_images"}:
            continue
        result = entry.get("result") or {}
        payload = result.get("result", result) if isinstance(result, dict) else {}
        if not isinstance(payload, dict):
            continue
        for key in ("recommendations", "images", "items"):
            items = payload.get(key)
            if not isinstance(items, list):
                continue
            for item in items:
                if not isinstance(item, dict):
                    continue
                image_link = str(
                    item.get("imageLink") or item.get("image_link") or item.get("link") or ""
                ).strip()
                if image_link:
                    return image_link
    return ""


def _extract_requested_gpu_model(user_message: str) -> str:
    normalized = _normalized_text(user_message)
    for model in ("v100", "a100", "rtx6000"):
        if model in normalized:
            return model
    return "v100" if "gpu" in normalized or "llm" in normalized or "训练" in normalized else ""


def _fallback_submission_actions(
    *,
    user_message: str,
    evidence: list[dict[str, Any]],
    enabled_tools: list[str],
) -> list[dict[str, Any]]:
    normalized = _normalized_text(user_message)
    prefers_jupyter = any(token in normalized for token in ("jupyter", "notebook", "交互式"))
    prefers_webide = any(token in normalized for token in ("webide", "code-server", "vscode", "ide"))
    prefers_pytorch = any(
        token in normalized for token in ("pytorch", "torchrun", "ddp", "deepspeed", "分布式训练", "distributed")
    )
    prefers_tensorflow = any(token in normalized for token in ("tensorflow", "tfjob", "parameter server", "ps"))
    tool_name = ""
    if prefers_webide and "create_webide_job" in enabled_tools:
        tool_name = "create_webide_job"
    elif prefers_tensorflow and "create_tensorflow_job" in enabled_tools:
        tool_name = "create_tensorflow_job"
    elif prefers_pytorch and "create_pytorch_job" in enabled_tools:
        tool_name = "create_pytorch_job"
    elif prefers_jupyter and "create_jupyter_job" in enabled_tools:
        tool_name = "create_jupyter_job"
    elif "create_custom_job" in enabled_tools:
        tool_name = "create_custom_job"
    elif "create_training_job" in enabled_tools:
        tool_name = "create_training_job"
    elif "create_jupyter_job" in enabled_tools:
        tool_name = "create_jupyter_job"
    if not tool_name:
        return []

    gpu_model = _extract_requested_gpu_model(user_message)
    image_link = _extract_recommended_image(evidence)
    if tool_name in {"create_jupyter_job", "create_webide_job"}:
        tool_args: dict[str, Any] = {
            "name": "workspace-notebook" if tool_name == "create_jupyter_job" else "workspace-webide",
            "cpu": "4",
            "memory": "16Gi",
            "gpu_count": 1 if gpu_model else None,
            "gpu_model": gpu_model or None,
        }
        if image_link:
            tool_args["image_link"] = image_link
    elif tool_name in {"create_pytorch_job", "create_tensorflow_job"}:
        base_image = image_link or ""
        default_tasks = [
            {
                "name": "master" if tool_name == "create_pytorch_job" else "chief",
                "replicas": 1,
                "image_link": base_image,
                "cpu": "8",
                "memory": "32Gi",
                "gpu_count": 1 if gpu_model else 0,
                "gpu_model": gpu_model or None,
            },
            {
                "name": "worker",
                "replicas": 1,
                "image_link": base_image,
                "cpu": "8",
                "memory": "32Gi",
                "gpu_count": 1 if gpu_model else 0,
                "gpu_model": gpu_model or None,
            },
        ]
        tool_args = {
            "name": "distributed-training",
            "tasks": default_tasks,
        }
    else:
        tool_args = {
            "name": "llm-training",
            "command": "",
            "working_dir": "/workspace",
            "cpu": "4",
            "memory": "16Gi",
            "gpu_count": 1 if gpu_model else None,
            "gpu_model": gpu_model or None,
        }
        if image_link:
            tool_args["image_link"] = image_link

    sanitized_args = {key: value for key, value in tool_args.items() if value not in {None, ""}}
    return [
        {
            "tool_name": tool_name,
            "tool_args": sanitized_args,
            "title": f"{tool_name}:draft",
            "reason": "submission 场景使用确认表单承接缺失参数，避免在多轮查询中空转",
            "depends_on_indexes": [],
        }
    ]


# ---------------------------------------------------------------------------
# Terminal answer builders
# ---------------------------------------------------------------------------

def _build_terminal_action_answer(state: MASState) -> str | None:
    if any(action.status in {"pending", "running", "awaiting_confirmation"} for action in state.actions):
        return None
    if not state.action_history:
        return None

    status_label = {
        "completed": "已执行",
        "error": "执行失败",
        "rejected": "已取消",
        "skipped": "已跳过",
    }
    lines: list[str] = []
    completed = 0
    rejected = 0
    failed = 0
    skipped = 0
    for item in state.action_history[-8:]:
        if not isinstance(item, dict):
            continue
        title = str(item.get("title") or item.get("tool_name") or "action").strip()
        status = str(item.get("status") or "unknown").strip()
        if status == "completed":
            completed += 1
        elif status == "rejected":
            rejected += 1
        elif status == "error":
            failed += 1
        elif status == "skipped":
            skipped += 1
        message = _extract_result_message(item.get("result") if isinstance(item.get("result"), dict) else None)
        suffix = f"：{message}" if message else ""
        lines.append(f"- {title}：{status_label.get(status, status)}{suffix}")

    if not lines:
        return None

    summary_parts: list[str] = []
    if completed:
        summary_parts.append(f"{completed} 个已执行")
    if rejected:
        summary_parts.append(f"{rejected} 个已取消")
    if failed:
        summary_parts.append(f"{failed} 个失败")
    if skipped:
        summary_parts.append(f"{skipped} 个跳过")
    summary_text = "，".join(summary_parts) if summary_parts else "本轮动作已结束"
    return f"当前工作流已结束：{summary_text}。\n\n" + "\n".join(lines)


def _build_runtime_fallback_final_answer(state: MASState) -> str:
    reason_label = {
        "max_rounds": "达到最大迭代次数",
        "no_progress": "连续多轮无明显进展",
    }.get(state.stop_reason, "")
    body = (
        state.execution.summary if state.execution
        else state.observation.summary if state.observation
        else "本轮已停止，但没有足够结果可直接总结。"
    )
    if not reason_label:
        return body
    return f"本轮已按运行时保护机制收束：{reason_label}。\n\n{body}"


def _derive_runtime_scenario_from_routing(routing: RoutingDecision) -> str:
    """Map routing decision to a runtime scenario label for SSE events."""
    if routing.entry_mode in {"help", "simple"}:
        return "guide"
    if routing.operation_mode == "write":
        return "action"
    return "query"


# ---------------------------------------------------------------------------
# MultiAgentOrchestrator
# ---------------------------------------------------------------------------

class MultiAgentOrchestrator:
    def __init__(self, tool_executor: ToolExecutorProtocol | None = None):
        self.tool_executor = tool_executor or CompositeToolExecutor()

    @staticmethod
    def _create_role_llm(model_factory: Any, role: str):
        """Create a role-specific LLM client.

        Supports both:
        - ModelClientFactory.create(client_key: str)
        - A role-aware factory with create(purpose=..., orchestration_mode=...)
        """
        try:
            return model_factory.create(purpose=role, orchestration_mode="multi_agent")
        except TypeError:
            return model_factory.create(role)

    @staticmethod
    def _determine_next_stage_fast(
        state: MASState,
        routing: RoutingDecision,
        resumed_action: dict[str, Any] | None,
    ) -> str | None:
        """Deterministic fast-paths that don't need Coordinator LLM.

        Returns a stage string if a fast path matches, None if Coordinator should decide.
        """
        # Confirmation resume: terminal write results must pass back through
        # Coordinator so the verifier gate can challenge risky post-action
        # outcomes. Low-risk results can still be finalized by the gate.
        if resumed_action and not any(a.status == "pending" for a in state.actions):
            resumed_status = _action_status(resumed_action.get("status"))
            if resumed_status in {"completed", "success", "error", "failed", "rejected", "skipped"}:
                return "verify" if _should_run_terminal_verifier(state) else "finalize"
            return "finalize"
        if resumed_action and state.actions:
            return "act"

        # Awaiting confirmation → can't proceed
        if _has_awaiting_write_confirmation(state):
            return "finalize"

        if (
            routing.requested_action == "node_isolation"
            and not _has_tool_record(state, "get_rdma_interface_status")
            and "get_rdma_interface_status" in state.enabled_tools
        ):
            return "observe"

        # For write intents, Executor owns both the minimal preflight reads and
        # the confirmation tool call. Keeping the controller in observe mode too
        # long often loses the confirmation flow.
        if _has_write_intent(routing) and not _has_terminal_write_result(state):
            allowed_write_tools = [
                tool_name
                for tool_name in state.enabled_tools
                if tool_name in CONFIRM_TOOL_NAMES
                and is_actor_allowed_for_tool(state.goal.actor_role, tool_name)
            ]
            if allowed_write_tools:
                return "act"

        # Force finalize if we have evidence and most tools are already attempted
        # (prevents the coordinator from endlessly looping)
        if state.tool_records and state.loop_round >= 3 and len(state.attempted_tool_signatures) >= 10:
            return "finalize"

        # Health overview is already an aggregate answer. Avoid repeated
        # low-value list_user_jobs probes unless a write/action flow needs them.
        if (
            _has_tool_record(state, "get_health_overview")
            and state.observation
            and not _has_write_intent(routing)
        ):
            return "finalize"

        # Everything else → Coordinator decides
        return None

    async def stream(
        self,
        *,
        request: Any,
        model_factory: ModelClientFactory,
    ) -> AsyncIterator[dict]:
        state = MASState.from_request(request)
        ablation = _load_eval_ablation(dict(request.context or {}))
        _apply_runtime_overrides(state, ablation)
        page_context = dict(state.goal.page_context)
        capabilities = state.capabilities
        enabled_tools = state.enabled_tools
        raw_enabled_tools = [
            str(tool_name).strip()
            for tool_name in (
                (dict(getattr(request, "context", None) or {}).get("capabilities") or {}).get("enabled_tools")
                or []
            )
            if str(tool_name).strip()
        ]
        goal_message = state.goal.original_user_message

        def make_agent(cls: type, agent_id: str, role: str) -> Any:
            return cls(
                agent_id=agent_id,
                role=role,
                llm=self._create_role_llm(model_factory, role),
            )

        coordinator = make_agent(CoordinatorAgent, "coordinator-1", "coordinator")
        intent_agent = make_agent(BaseRoleAgent, "intent-router-1", "intent_router")
        planner = make_agent(PlannerAgent, "planner-1", "planner")
        explorer = make_agent(ExplorerAgent, "explorer-1", "explorer")
        executor = make_agent(ExecutorAgent, "executor-1", "executor")
        verifier = make_agent(VerifierAgent, "verifier-1", "verifier")
        last_verification_signature = ""
        last_verification_result: RoleExecutionResult | None = None
        last_replan_signature = ""

        def record_agent_usage(agent: Any) -> None:
            usage = dict(getattr(agent, "last_usage", None) or {})
            usage["latency_ms"] = int(getattr(agent, "last_latency_ms", 0) or 0)
            state.record_llm_usage(usage, role=str(getattr(agent, "role", "") or ""))

        async def emit(event: str, data: dict[str, Any]) -> dict[str, Any]:
            return {"event": event, "data": {"turnId": request.turn_id, **data}}

        async def emit_final_answer(
            *,
            agent_id: str,
            agent_role: str,
            content: str,
            continuation_payload: dict[str, Any] | None = None,
        ) -> dict[str, Any]:
            runtime_scenario = _derive_runtime_scenario_from_routing(state.goal.routing)
            payload: dict[str, Any] = {
                "sessionId": request.session_id,
                "agentId": agent_id,
                "agentRole": agent_role,
                "content": content,
                "stopReason": state.stop_reason or "completed",
                "runtimeScenario": runtime_scenario,
                "usageSummary": state.usage_summary.to_dict(),
            }
            if continuation_payload:
                payload["continuation"] = continuation_payload
            return await emit("final_answer", payload)

        async def emit_checkpoint(*, summary: str, workflow: dict[str, Any]) -> dict[str, Any]:
            return await emit(
                "agent_checkpoint",
                {
                    "sessionId": request.session_id,
                    "agentId": coordinator.agent_id,
                    "agentRole": coordinator.role,
                    "summary": summary,
                    "workflow": workflow,
                    "status": "completed",
                },
            )

        async def emit_confirmation_pause(
            *,
            role_agent_id: str,
            role_name: str,
            summary: str,
        ) -> AsyncIterator[dict]:
            awaiting_actions = [
                action for action in state.actions if action.status == "awaiting_confirmation"
            ]
            state.stop_reason = "awaiting_confirmation"
            checkpoint = state.build_workflow_checkpoint()
            checkpoint["pause_reason"] = "awaiting_confirmation"
            checkpoint["current_action_ids"] = [action.action_id for action in awaiting_actions]
            checkpoint["pending_confirmation_ids"] = [
                action.confirm_id for action in awaiting_actions if action.confirm_id
            ]
            yield await emit_checkpoint(
                summary="已保存当前工作流状态，等待用户确认后继续执行",
                workflow=checkpoint,
            )
            yield await emit(
                "agent_status",
                {
                    "agentId": role_agent_id,
                    "agentRole": role_name,
                    "status": "awaiting_confirmation",
                    "summary": summary,
                },
            )
            yield {"event": "done", "data": {"usageSummary": state.usage_summary.to_dict()}}

        # -----------------------------------------------------------------
        # Tool execution infrastructure
        # -----------------------------------------------------------------

        async def call_tool(
            *,
            role_agent_id: str,
            role_name: str,
            tool_name: str,
            tool_args: dict[str, Any],
            tool_call_id: str,
        ) -> tuple[dict[str, Any], list[dict[str, Any]]]:
            if not is_tool_allowed_for_role(role_name, tool_name):
                logger.warning("Tool '%s' not allowed for role '%s', skipping", tool_name, role_name)
                result = {
                    "status": "error",
                    "message": f"Tool '{tool_name}' is not allowed for role '{role_name}'",
                }
                return result, [
                    await emit(
                        "tool_call_completed",
                        {
                            "agentId": role_agent_id,
                            "agentRole": role_name,
                            "toolCallId": tool_call_id,
                            "toolName": tool_name,
                            "toolArgs": tool_args,
                            "result": result["message"],
                            "resultSummary": result["message"],
                            "status": "error",
                            "isError": True,
                        },
                    )
                ]

            events = [
                await emit(
                    "tool_call_started",
                    {
                        "agentId": role_agent_id,
                        "agentRole": role_name,
                        "toolCallId": tool_call_id,
                        "toolName": tool_name,
                        "toolArgs": tool_args,
                        "status": "executing",
                    },
                )
            ]
            state.usage_summary.tool_calls += 1
            if tool_name in READ_ONLY_TOOL_NAMES:
                state.usage_summary.read_tool_calls += 1
            else:
                state.usage_summary.write_tool_calls += 1
            tool_started_at = time.perf_counter()
            result = await self.tool_executor.execute(
                tool_name=tool_name,
                tool_args=tool_args,
                session_id=request.session_id,
                user_id=int(
                    (dict(getattr(request, "context", None) or {}).get("actor") or {}).get("user_id") or 0
                ),
                turn_id=request.turn_id,
                tool_call_id=tool_call_id,
                agent_id=role_agent_id,
                agent_role=role_name,
                actor_role=state.goal.actor_role,
            )
            measured_tool_latency_ms = max(1, int((time.perf_counter() - tool_started_at) * 1000))
            if not isinstance(result, dict):
                result = {"status": "error", "message": str(result)}
            if not result.get("_latency_ms"):
                result["_latency_ms"] = measured_tool_latency_ms
            state.usage_summary.tool_latency_ms += int(result.get("_latency_ms") or 0)
            if result.get("status") == "confirmation_required":
                confirmation = result.get("confirmation", {})
                events.append(
                    await emit(
                        "tool_call_confirmation_required",
                        {
                            "agentId": role_agent_id,
                            "agentRole": role_name,
                            "toolCallId": tool_call_id,
                            "confirmId": confirmation.get("confirm_id", ""),
                            "action": confirmation.get("tool_name", tool_name),
                            "description": confirmation.get("description", ""),
                            "interaction": confirmation.get("interaction", "approval"),
                            "form": confirmation.get("form"),
                            "status": "awaiting_confirmation",
                            "latencyMs": result.get("_latency_ms", 0),
                        },
                    )
                )
                return result, events

            events.append(
                await emit(
                    "tool_call_completed",
                    {
                        "agentId": role_agent_id,
                        "agentRole": role_name,
                        "toolCallId": tool_call_id,
                        "toolName": tool_name,
                        "toolArgs": tool_args,
                        "result": result.get("result", result.get("message", "")),
                        "resultSummary": str(result.get("result", result.get("message", "")))[:500],
                        "status": "error" if result.get("status") == "error" else "done",
                        "isError": result.get("status") == "error",
                        "latencyMs": result.get("_latency_ms", 0),
                    },
                )
            )
            report_payload = build_pipeline_report_payload(tool_name, result)
            if report_payload:
                events.append(await emit("pipeline_report", report_payload))
            return result, events

        async def run_role_tool_loop(
            *,
            role_agent: Any,
            role_name: str,
            system_prompt: str,
            user_prompt: str,
            allowed_tool_names: list[str],
            max_tool_calls: int,
            on_tool_result: Callable[[str, dict[str, Any], str, dict[str, Any]], Awaitable[ToolLoopStopSignal | None]],
            loop_history_messages: list | None = None,
        ) -> tuple[ToolLoopOutcome, list[dict[str, Any]]]:
            role_agent.last_usage = {
                "llm_calls": 0,
                "input_tokens": 0,
                "output_tokens": 0,
                "total_tokens": 0,
                "reported_token_calls": 0,
                "missing_token_calls": 0,
            }
            role_agent.last_latency_ms = 0
            tool_map = {
                tool.name: tool
                for tool in ALL_TOOLS
                if tool.name in set(allowed_tool_names)
            }
            if not tool_map:
                return ToolLoopOutcome(summary="", tool_calls=0), []

            evidence_dicts = _extract_evidence_from_tool_records(state.tool_records)

            messages: list[Any] = [SystemMessage(content=system_prompt)]
            if loop_history_messages:
                messages.extend(loop_history_messages)
            messages.append(HumanMessage(content=user_prompt))
            llm_with_tools = role_agent.llm.bind_tools(list(tool_map.values()))
            collected_events: list[dict[str, Any]] = []
            aggregate_usage: dict[str, int] = {
                "llm_calls": 0,
                "input_tokens": 0,
                "output_tokens": 0,
                "total_tokens": 0,
                "reported_token_calls": 0,
                "missing_token_calls": 0,
            }
            aggregate_latency_ms = 0
            invoked_tool_calls = 0
            stalled_tool_rounds = 0

            @retry(
                stop=stop_after_attempt(3),
                wait=wait_exponential(multiplier=1, min=1, max=8),
                retry=retry_if_exception_type(_RETRYABLE_TOOL_LOOP_LLM_ERRORS),
                before_sleep=lambda rs: logger.warning(
                    "[%s/%s] tool-loop LLM retry #%d after %s: %s",
                    role_name,
                    role_agent.agent_id,
                    rs.attempt_number,
                    type(rs.outcome.exception()).__name__,
                    rs.outcome.exception(),
                ),
            )
            async def _invoke_tool_loop_llm(current_messages: list[Any]) -> Any:
                return await llm_with_tools.ainvoke(current_messages)

            for loop_index in range(max(1, max_tool_calls + 1)):
                llm_started_at = time.monotonic()
                response = await _invoke_tool_loop_llm(messages)
                aggregate_latency_ms += int((time.monotonic() - llm_started_at) * 1000)
                content, reasoning = role_agent._extract_response_texts(response)
                selected = role_agent._select_response_text(content=content, reasoning=reasoning)
                aggregate_usage = role_agent._merge_usage(
                    aggregate_usage,
                    _extract_usage_from_tool_loop_response(
                        role_agent,
                        response,
                        messages,
                        selected,
                    ),
                )
                role_agent.last_content = content
                role_agent.last_reasoning_content = reasoning
                role_agent.last_selected_text = selected
                messages.append(response)

                tool_calls = list(getattr(response, "tool_calls", []) or [])
                if not tool_calls:
                    role_agent.last_usage = aggregate_usage
                    role_agent.last_latency_ms = aggregate_latency_ms
                    return ToolLoopOutcome(
                        summary=selected or role_agent.latest_reasoning_summary(),
                        tool_calls=invoked_tool_calls,
                    ), collected_events

                if invoked_tool_calls >= max_tool_calls:
                    role_agent.last_usage = aggregate_usage
                    role_agent.last_latency_ms = aggregate_latency_ms
                    summary = selected or role_agent.latest_reasoning_summary()
                    if not summary:
                        summary = "已达到工具调用上限，请基于已有结果继续下一步。"
                    return ToolLoopOutcome(summary=summary, tool_calls=invoked_tool_calls), collected_events

                executed_new_tool_in_round = False
                for tool_index, tool_call in enumerate(tool_calls, start=1):
                    tool_name = str(tool_call.get("name") or "").strip()
                    tool_args = tool_call.get("args") if isinstance(tool_call.get("args"), dict) else {}
                    tool_call_id = str(
                        tool_call.get("id") or f"{role_agent.agent_id}-tool-loop-{loop_index}-{tool_index}"
                    )
                    tool_observation = ""

                    if not tool_name or tool_name not in tool_map:
                        result: dict[str, Any] = {
                            "status": "error",
                            "message": f"Tool '{tool_name}' is not available for role '{role_name}'",
                        }
                        tool_observation = result["message"]
                        collected_events.append(
                            await emit(
                                "tool_call_completed",
                                {
                                    "agentId": role_agent.agent_id,
                                    "agentRole": role_name,
                                    "toolCallId": tool_call_id,
                                    "toolName": tool_name,
                                    "toolArgs": tool_args,
                                    "result": result["message"],
                                    "resultSummary": result["message"],
                                    "status": "error",
                                    "isError": True,
                                },
                            )
                        )
                    else:
                        signature = _tool_signature(tool_name, tool_args)
                        if signature in state.attempted_tool_signatures:
                            previous_result = _find_previous_tool_result(
                                evidence_dicts,
                                tool_name=tool_name,
                                tool_args=tool_args,
                            )
                            previous_result_summary = (
                                _build_tool_loop_observation(tool_name, previous_result)
                                if previous_result
                                else ""
                            )
                            duplicate_message = (
                                f"Tool {tool_name} 已经用相同参数执行过，不要重复调用。"
                                "请直接基于已有结果继续推理。"
                            )
                            if previous_result_summary:
                                duplicate_message += f"\n已有结果:\n{previous_result_summary}"
                            result = {
                                "status": "error",
                                "message": duplicate_message,
                            }
                            tool_observation = duplicate_message
                            collected_events.append(
                                await emit(
                                    "tool_call_completed",
                                    {
                                        "agentId": role_agent.agent_id,
                                        "agentRole": role_name,
                                        "toolCallId": tool_call_id,
                                        "toolName": tool_name,
                                        "toolArgs": tool_args,
                                        "result": result["message"],
                                        "resultSummary": result["message"],
                                        "status": "error",
                                        "isError": True,
                                    },
                                )
                            )
                        else:
                            state.attempted_tool_signatures.append(signature)
                            result, tool_events = await call_tool(
                                role_agent_id=role_agent.agent_id,
                                role_name=role_name,
                                tool_name=tool_name,
                                tool_args=tool_args,
                                tool_call_id=tool_call_id,
                            )
                            collected_events.extend(tool_events)
                            invoked_tool_calls += 1
                            executed_new_tool_in_round = True
                            tool_observation = _build_tool_loop_observation(tool_name, result)
                            # Update evidence snapshot for duplicate detection
                            evidence_dicts = _extract_evidence_from_tool_records(state.tool_records)

                    messages.append(
                        ToolMessage(
                            content=tool_observation or _build_tool_loop_observation(tool_name, result),
                            tool_call_id=tool_call_id,
                        )
                    )
                    stop_signal = await on_tool_result(tool_name, tool_args, tool_call_id, result)
                    if stop_signal and stop_signal.should_stop:
                        role_agent.last_usage = aggregate_usage
                        role_agent.last_latency_ms = aggregate_latency_ms
                        return ToolLoopOutcome(
                            summary=stop_signal.summary or selected or role_agent.latest_reasoning_summary(),
                            tool_calls=invoked_tool_calls,
                            stop_signal=stop_signal,
                        ), collected_events

                    if invoked_tool_calls >= max_tool_calls:
                        break

                if executed_new_tool_in_round:
                    stalled_tool_rounds = 0
                else:
                    stalled_tool_rounds += 1
                    if stalled_tool_rounds >= 1:
                        role_agent.last_usage = aggregate_usage
                        role_agent.last_latency_ms = aggregate_latency_ms
                        summary = selected or role_agent.latest_reasoning_summary()
                        if not summary:
                            summary = "工具调用连续重复且没有产生新结果，已停止继续调用。"
                        return ToolLoopOutcome(
                            summary=summary,
                            tool_calls=invoked_tool_calls,
                        ), collected_events

            role_agent.last_usage = aggregate_usage
            role_agent.last_latency_ms = aggregate_latency_ms
            return ToolLoopOutcome(
                summary=role_agent.latest_reasoning_summary() or "已完成工具执行。",
                tool_calls=invoked_tool_calls,
            ), collected_events

        def ensure_action_item(tool_name: str, tool_args: dict[str, Any]) -> MultiAgentActionItem:
            signature = _tool_signature(tool_name, tool_args)
            for action in state.actions:
                if _tool_signature(action.tool_name, action.tool_args) == signature:
                    return action
            action = MultiAgentActionItem(
                action_id=f"action-{len(state.actions) + 1}",
                tool_name=tool_name,
                tool_args=tool_args,
                title=_default_action_title(tool_name, tool_args),
                reason="executor_direct_tool_loop",
            )
            state.actions.append(action)
            return action

        # =================================================================
        # STEP 1: Emit run started
        # =================================================================
        yield await emit(
            "agent_run_started",
            {
                "sessionId": request.session_id,
                "agentId": coordinator.agent_id,
                "agentRole": coordinator.role,
                "status": "started",
                "summary": _build_resume_run_started_summary(state.resume_context),
                "evalAblation": ablation.to_context() if ablation.enabled else None,
            },
        )

        # =================================================================
        # STEP 2: Intent Routing
        # =================================================================
        # Text history excerpt is only used by IntentRouter's LLM classification
        history_context = state.recent_history_context()

        if state.resume_context:
            # Restore routing from workflow checkpoint
            routing = state.goal.routing
            if _should_restore_routing_from_resume(state.resume_context) and not routing.requested_action:
                routing.requested_action = (
                    str(state.resume_context.get("action_intent") or "").strip().lower() or None
                )
                if routing.requested_action:
                    routing.operation_mode = "write"
        else:
            intent_router = IntentRouter(coordinator_agent=intent_agent)
            try:
                routing = await intent_router.route(
                    user_message=request.message,
                    page_context=page_context,
                    continuation=dict(state.continuation),
                    resume_context=dict(state.resume_context),
                    actor_role=state.goal.actor_role,
                    history_context=history_context,
                    clarification_context=state.clarification_context,
                )
                record_agent_usage(intent_agent)
            except Exception:
                logger.exception("IntentRouter failed, using default routing")
                routing = RoutingDecision(
                    entry_mode="agent",
                    operation_mode="unknown",
                    confidence=0.3,
                )

        state.goal.routing = routing

        # Build structured LangChain history messages (replaces text-only summaries)
        history_messages: list = []
        if state.history:
            history_messages = build_history_messages(
                history=state.history,
                max_tokens=4000,
                tool_result_max_chars=160,
            )
            # Strip leading orphan ToolMessages (no preceding AIMessage with tool_calls)
            while history_messages and isinstance(history_messages[0], ToolMessage):
                history_messages.pop(0)

        # Resolve node_name from page context
        if not page_context.get("node_name"):
            resolved_node_name = extract_node_name(page_context, goal_message)
            if resolved_node_name:
                page_context["node_name"] = resolved_node_name
                state.goal.page_context = page_context

        # Bind job_name from routing targets into page_context
        if routing.targets.job_name and not page_context.get("job_name"):
            page_context["job_name"] = routing.targets.job_name
            state.goal.page_context = page_context

        # =================================================================
        # STEP 3: Apply resume outcome
        # =================================================================
        resumed_action = state.apply_resume_outcome()
        if resumed_action:
            yield await emit(
                "agent_status",
                {
                    "agentId": executor.agent_id,
                    "agentRole": executor.role,
                    "status": resumed_action["status"],
                    "summary": (
                        f"确认结果已同步：{resumed_action['tool_name']} -> {resumed_action['status']}"
                    ),
                },
            )
            if resumed_action["status"] == "rejected":
                state.stop_reason = "completed"
                state.final_answer = _build_rejected_resume_final_answer(resumed_action, state.resume_context)
                yield await emit_final_answer(
                    agent_id=coordinator.agent_id,
                    agent_role=coordinator.role,
                    content=state.final_answer,
                    continuation_payload=_build_final_continuation_payload(state),
                )
                yield {"event": "done", "data": {"usageSummary": state.usage_summary.to_dict()}}
                return
        elif _resume_result_status(state.resume_context) == "rejected":
            state.stop_reason = "completed"
            state.final_answer = _build_rejected_resume_final_answer(None, state.resume_context)
            yield await emit_final_answer(
                agent_id=coordinator.agent_id,
                agent_role=coordinator.role,
                content=state.final_answer,
                continuation_payload=_build_final_continuation_payload(state),
            )
            yield {"event": "done", "data": {"usageSummary": state.usage_summary.to_dict()}}
            return
        if state.resume_context:
            state.stop_reason = ""

        # =================================================================
        # STEP 4: Simple/help fast path
        # =================================================================
        if (
            routing.entry_mode in {"simple", "help"}
            and routing.operation_mode == "unknown"
            and not state.workflow
            and not state.resume_context
            and not ablation.bypass_help_fast_path
            and is_strict_toolless_fast_path_candidate(
                user_message=goal_message,
                page_context=page_context,
                routing=routing,
            )
        ):
            if routing.entry_mode == "simple":
                help_agent = make_agent(GeneralPurposeAgent, "general-1", "general")
                summary_msg = "Coordinator 将极简问题交给 General 直接答复"
            elif routing.entry_mode == "help":
                help_agent = make_agent(GuideAgent, "guide-1", "guide")
                summary_msg = "Coordinator 将帮助类问题交给 Guide 处理"
            else:
                help_agent = make_agent(GeneralPurposeAgent, "general-1", "general")
                summary_msg = "Coordinator 将通用问答交给 General 处理"

            yield await emit(
                "agent_handoff",
                {
                    "agentId": coordinator.agent_id,
                    "agentRole": coordinator.role,
                    "targetAgentId": help_agent.agent_id,
                    "targetAgentRole": help_agent.role,
                    "summary": summary_msg,
                    "status": "completed",
                },
            )
            try:
                result = await help_agent.respond(
                    user_message=goal_message,
                    page_context=page_context,
                    capabilities=capabilities,
                    actor_role=state.goal.actor_role,
                    history_messages=history_messages,
                )
                record_agent_usage(help_agent)
                state.final_answer = result.summary
            except Exception:
                logger.exception("Help Agent response failed")
                state.final_answer = "抱歉，生成帮助说明时出错，请稍后重试。"
            
            state.stop_reason = "completed"
            yield await emit(
                "agent_status",
                {
                    "agentId": help_agent.agent_id,
                    "agentRole": help_agent.role,
                    "status": "completed",
                    "summary": state.final_answer,
                },
            )
            yield await emit_final_answer(
                agent_id=help_agent.agent_id,
                agent_role=help_agent.role,
                content=state.final_answer,
            )
            yield {"event": "done", "data": {"usageSummary": state.usage_summary.to_dict()}}
            return

        # =================================================================
        # STEP 5: Coordinator Loop
        # =================================================================
        while True:
            if state.loop_round >= state.runtime_config.lead_max_rounds:
                state.stop_reason = "max_rounds"
                break
            if state.no_progress_count >= state.runtime_config.no_progress_rounds:
                state.stop_reason = "no_progress"
                break

            state.loop_round += 1

            # Try deterministic fast-paths first
            next_stage = self._determine_next_stage_fast(state, routing, resumed_action)
            next_stage = _apply_stage_ablation(
                candidate=next_stage,
                state=state,
                routing=routing,
                ablation=ablation,
            )

            # If no fast-path matched, ask Coordinator LLM to decide
            if next_stage is None:
                state_view = state.build_state_view("coordinator")
                verification_verdict = _extract_verification_verdict(last_verification_result)
                verification_summary = (
                    last_verification_result.summary if last_verification_result else ""
                )
                state_view.decision_context = _build_coordinator_decision_context(
                    state=state,
                    routing=routing,
                    enabled_tools=enabled_tools,
                    raw_enabled_tools=raw_enabled_tools,
                    user_message=goal_message,
                    page_context=page_context,
                    verification_verdict=verification_verdict,
                    verification_summary=verification_summary,
                )
                tool_stats = (
                    f"\n\n当前进度统计：\n"
                    f"- 已调用工具次数: {len(state.attempted_tool_signatures)}\n"
                    f"- 已收集证据条数: {len(state.tool_records)}\n"
                    f"- 当前轮次: {state.loop_round}/{state.runtime_config.lead_max_rounds}\n"
                    f"- 连续无进展轮数: {state.no_progress_count}\n"
                )
                if verification_verdict:
                    tool_stats += (
                        f"- 最近验证结论: {verification_verdict}\n"
                        f"- 最近验证摘要: {_truncate_text(verification_summary, max_chars=180)}\n"
                    )
                try:
                    coordinator_decision = await coordinator.run_json(
                        system_prompt=(
                            "你是 Crater MAS 的 Coordinator 协调者。你根据当前状态决定下一步。\n\n"
                            "可选动作：\n"
                            '- "plan": 问题复杂、工具链不明确、或需要先拆解步骤\n'
                            '- "observe": 需要收集更多信息（调用只读工具）\n'
                            '- "act": 需要执行操作（调用写工具，或 executor 先读后写）\n'
                            '- "verify": 需要对当前结论做一次验证/挑战\n'
                            '- "replan": 当前计划与实际收集的证据不匹配，需要 Planner 重新规划\n'
                            '- "finalize": 信息已足够，可以回答用户了\n\n'
                            "决策原则：\n"
                            "- 第一轮不一定要 plan；简单查询可直接 observe，写操作目标明确可直接 act，信息足够可 finalize\n"
                            "- 只有当工具链不明确、多阶段任务、故障复杂、或已有证据无法支持下一步时才选 plan/replan\n"
                            "- 有计划就按计划推进，不要无故偏离；如果计划明显不匹配最新证据，才 replan\n"
                            "- verify 不是必选项；只有在高风险写操作完成后、证据冲突、或结论需要额外核实时才选 verify\n"
                            "- 简单问答、普通查询、证据直接充分的场景可以直接 finalize，不要机械要求 verify\n"
                            "- 如果已收集的证据足以回答用户，选 finalize\n"
                            "- 如果证据和用户请求明显不匹配、计划走偏了，选 replan\n"
                            "- 如果需要执行写操作且目标明确，选 act\n"
                            "- 如果还缺关键信息，选 observe\n"
                            "- 如果 Coordinator 决策上下文中的 recommended_next_stage_hint 明确指出 observe/act/finalize，优先遵循；只有你能说清楚具体缺口时才偏离\n"
                            "- 如果 action_readiness.write_permission_gap=true，不要选 act；可以先基于普通只读权限 observe，已能解释现象或无法继续取证时 finalize，并说明不能代执行管理员写操作\n"
                            "- 确认类写操作不要扩散取证：目标明确且只差动作前最小核验时选 act，让 Executor 做一次必要只读核验后调用确认型写工具\n"
                            "- 健康/noop 或“是否正常/需不需要处理”类请求不能只看 Running 状态；若 query_job_metrics 可用且尚未调用，应优先 observe 补一条指标证据\n"
                            "- observe 必须对应 missing_facts 里的具体缺口；不要因为“还能再看看”而重复读取同一事实桶\n"
                            "- 如果 over_exploration_warnings 非空，除非 missing_facts 仍有会改变结论的缺口，否则选 finalize 或 act\n"
                            "- 如果当前能力或证据明确无法满足请求，不要空转；说明限制并 finalize\n"
                            "- 不要反复 observe/act 相同内容，如果工具已调用超过 10 次且证据充分，应当 finalize\n"
                            "- 连续无进展 ≥ 1 轮时，强烈建议 finalize，不要继续空转\n\n"
                            '输出 JSON: {"next": "plan|observe|act|verify|replan|finalize", "reason": "简短理由"}\n'
                            + tool_stats
                        ),
                        user_prompt=state_view.for_prompt(),
                        history_messages=history_messages,
                    )
                    record_agent_usage(coordinator)
                    next_stage = str(coordinator_decision.get("next") or "finalize").strip().lower()
                    reason = str(coordinator_decision.get("reason") or "").strip()
                    if next_stage not in {"observe", "act", "verify", "replan", "finalize", "plan"}:
                        next_stage = "finalize"
                    next_stage = _apply_stage_ablation(
                        candidate=next_stage,
                        state=state,
                        routing=routing,
                        ablation=ablation,
                    )
                    logger.info(
                        "Coordinator decision round=%d: %s (%s)",
                        state.loop_round, next_stage, reason,
                    )
                except Exception:
                    logger.exception("Coordinator decision failed, falling back to finalize")
                    next_stage = "finalize"
            else:
                next_stage = _apply_stage_ablation(
                    candidate=next_stage,
                    state=state,
                    routing=routing,
                    ablation=ablation,
                )

            # After the first iteration, clear resumed_action
            if state.loop_round > 1:
                resumed_action = None

            current_progress_signature = _build_verification_signature(state)
            if next_stage == "plan" and state.plan and current_progress_signature == last_replan_signature:
                next_stage = "observe" if not state.tool_records else "finalize"
                state.remember_controller_decision({
                    "round": state.loop_round,
                    "stage": "plan_skipped",
                    "reason": "state_unchanged_since_existing_plan",
                    "next_stage": next_stage,
                })
                yield await emit(
                    "agent_status",
                    {
                        "agentId": coordinator.agent_id,
                        "agentRole": coordinator.role,
                        "status": "plan_skipped",
                        "summary": "Planner 已跳过：已有计划且状态没有变化",
                    },
                )
            if next_stage == "replan" and current_progress_signature == last_replan_signature:
                frontier = state.action_frontier()
                if frontier or (_has_write_intent(routing) and not _has_terminal_write_result(state)):
                    next_stage = "act"
                elif state.tool_records or state.observation or state.execution or state.action_history:
                    next_stage = "finalize"
                else:
                    next_stage = "observe"
                state.remember_controller_decision({
                    "round": state.loop_round,
                    "stage": "replan_skipped",
                    "reason": "state_unchanged_since_last_replan",
                    "next_stage": next_stage,
                })
                yield await emit(
                    "agent_status",
                    {
                        "agentId": coordinator.agent_id,
                        "agentRole": coordinator.role,
                        "status": "replan_skipped",
                        "summary": "Planner 已跳过：状态自上次重规划后没有新增证据或执行结果",
                    },
                )

            if next_stage == "verify":
                if ablation.disable_verifier:
                    next_stage = "finalize"
                    state.remember_controller_decision({
                        "round": state.loop_round,
                        "stage": "verify_disabled",
                        "next_stage": next_stage,
                    })
                if next_stage != "verify":
                    pass
                else:
                    gate = _determine_verifier_gate(
                        state=state,
                        routing=routing,
                        current_progress_signature=current_progress_signature,
                        last_verification_signature=last_verification_signature,
                        last_verification_result=last_verification_result,
                    )
                    if not gate.run_verifier:
                        next_stage = gate.next_stage
                        state.remember_controller_decision({
                            "round": state.loop_round,
                            "stage": "verify_skipped",
                            "reason": gate.reason,
                            "next_stage": next_stage,
                        })
                        yield await emit(
                            "agent_status",
                            {
                                "agentId": coordinator.agent_id,
                                "agentRole": coordinator.role,
                                "status": "verify_skipped",
                                "summary": f"Verifier 已跳过：{gate.reason}",
                            },
                        )
                    else:
                        yield await emit(
                            "agent_handoff",
                            {
                                "agentId": coordinator.agent_id,
                                "agentRole": coordinator.role,
                                "targetAgentId": verifier.agent_id,
                                "targetAgentRole": verifier.role,
                                "summary": "Coordinator 要求 Verifier 复核当前证据与执行结果",
                                "status": "completed",
                            },
                        )
                        evidence_dicts = _extract_evidence_from_tool_records(state.tool_records)
                        compact_evidence = _compact_evidence_for_prompt(evidence_dicts)
                        evidence_summary_text = (
                            state.observation.summary if state.observation
                            else _build_evidence_summary_fallback(compact_evidence)
                        )
                        executor_summary_text = (
                            state.execution.summary if state.execution
                            else _build_action_history_summary(state.action_history)
                        )
                        try:
                            last_verification_result = await verifier.verify(
                                user_message=goal_message,
                                plan_summary=state.plan.summary if state.plan else "",
                                evidence_summary=evidence_summary_text,
                                compact_evidence=compact_evidence,
                                executor_summary=executor_summary_text,
                            )
                            record_agent_usage(verifier)
                        except Exception:
                            logger.exception("Verifier failed")
                            last_verification_result = RoleExecutionResult(
                                summary="验证阶段执行失败，当前视为证据不足。",
                                status="missing_evidence",
                                metadata={"verification_result": "missing_evidence"},
                            )
                        last_verification_signature = current_progress_signature
                        verification_verdict = _extract_verification_verdict(last_verification_result) or "missing_evidence"
                        yield await emit(
                            "agent_status",
                            {
                                "agentId": verifier.agent_id,
                                "agentRole": verifier.role,
                                "status": verification_verdict,
                                "summary": last_verification_result.summary,
                            },
                        )
                        state.remember_controller_decision({
                            "round": state.loop_round,
                            "stage": "verify",
                            "verdict": verification_verdict,
                        })
                        if verification_verdict == "missing_evidence":
                            if state.loop_round < state.runtime_config.lead_max_rounds:
                                next_stage = "replan"
                            else:
                                state.stop_reason = "insufficient_evidence"
                                next_stage = "finalize"
                        else:
                            next_stage = "finalize"

            if next_stage == "finalize":
                break

            state.remember_controller_decision({
                "round": state.loop_round,
                "stage": next_stage,
            })

            # ----- PLAN stage -----
            if next_stage == "plan":
                yield await emit(
                    "agent_handoff",
                    {
                        "agentId": coordinator.agent_id,
                        "agentRole": coordinator.role,
                        "targetAgentId": planner.agent_id,
                        "targetAgentRole": planner.role,
                        "summary": "Coordinator 将请求交给 Planner 制定计划",
                        "status": "completed",
                    },
                )
                try:
                    plan_result = await planner.plan(
                        user_message=goal_message,
                        page_context=page_context,
                        capabilities=capabilities,
                        actor_role=state.goal.actor_role,
                        evidence_summary=(
                            state.observation.summary if state.observation else ""
                        ),
                        history_messages=history_messages,
                    )
                    record_agent_usage(planner)
                except Exception:
                    logger.exception("Planner failed")
                    plan_result = RoleExecutionResult(
                        summary="规划失败，将基于直接证据收集继续推进",
                        metadata={"plan_output": {}},
                    )
                plan_output = (plan_result.metadata or {}).get("plan_output", {})
                state.plan = PlanArtifact(
                    summary=plan_output.get("raw_summary") or plan_result.summary,
                    steps=plan_output.get("steps", []),
                    candidate_tools=plan_output.get("candidate_tools", []),
                    risk=plan_output.get("risk", "low"),
                )
                last_replan_signature = current_progress_signature
                yield await emit(
                    "agent_status",
                    {
                        "agentId": planner.agent_id,
                        "agentRole": planner.role,
                        "status": "completed",
                        "summary": _build_plan_status_summary(state.plan),
                    },
                )
                continue

            # ----- OBSERVE stage -----
            if next_stage == "observe":
                observe_agent = executor if ablation.merge_explorer_executor else explorer
                observe_summary = (
                    "Coordinator 要求 Executor 以合并读写角色收集证据"
                    if ablation.merge_explorer_executor
                    else "Coordinator 要求 Explorer 继续收集证据"
                )
                yield await emit(
                    "agent_handoff",
                    {
                        "agentId": coordinator.agent_id,
                        "agentRole": coordinator.role,
                        "targetAgentId": observe_agent.agent_id,
                        "targetAgentRole": observe_agent.role,
                        "summary": observe_summary,
                        "status": "completed",
                    },
                )

                # Build preferred tools for first exploration
                preferred_tools: list[tuple[str, dict[str, Any]]] = []
                if not state.tool_records:
                    preferred_tools = _preferred_tools_for_first_observe(
                        routing=routing,
                        page_context=page_context,
                        enabled_tools=enabled_tools,
                        user_message=goal_message,
                    )

                prompt_candidate_tools = list(state.plan.candidate_tools if state.plan else [])
                for preferred_tool_name, _ in preferred_tools:
                    if preferred_tool_name not in prompt_candidate_tools:
                        prompt_candidate_tools.append(preferred_tool_name)

                evidence_dicts = _extract_evidence_from_tool_records(state.tool_records)
                compact_evidence = _compact_evidence_for_prompt(evidence_dicts)
                evidence_summary_text = (
                    state.observation.summary if state.observation
                    else _build_evidence_summary_fallback(compact_evidence)
                )

                evidence_before = len(state.tool_records)
                exploration_summary = ""
                loop_outcome: ToolLoopOutcome | None = None
                try:
                    if ablation.merge_explorer_executor:
                        loop_system_prompt, loop_user_prompt = executor.build_tool_loop_prompts(
                            user_message=goal_message,
                            page_context=page_context,
                            plan_summary=state.plan.summary if state.plan else "",
                            evidence_summary=evidence_summary_text,
                            compact_evidence=compact_evidence,
                            action_intent=routing.requested_action,
                            selected_job_name=routing.targets.job_name,
                            requested_scope=routing.targets.scope,
                            action_history=state.action_history,
                            pending_actions=_pending_action_dicts(state.actions),
                            enabled_tools=_actor_allowed_tools(state.goal.actor_role, enabled_tools),
                            actor_role=state.goal.actor_role,
                        )
                    else:
                        loop_system_prompt, loop_user_prompt = explorer.build_tool_loop_prompts(
                            user_message=goal_message,
                            page_context=page_context,
                            plan_candidate_tools=prompt_candidate_tools,
                            plan_steps=state.plan.steps if state.plan else [],
                            enabled_tools=enabled_tools,
                            evidence_summary=evidence_summary_text,
                            attempted_tool_signatures=state.attempted_tool_signatures,
                            compact_evidence=compact_evidence,
                        )

                    async def on_explorer_tool_result(
                        tool_name: str,
                        tool_args: dict[str, Any],
                        tool_call_id: str,
                        result: dict[str, Any],
                    ) -> ToolLoopStopSignal | None:
                        if tool_name in READ_ONLY_TOOL_NAMES:
                            state.remember_tool(
                                agent_id=observe_agent.agent_id,
                                agent_role=observe_agent.role,
                                tool_name=tool_name,
                                tool_args=tool_args,
                                tool_call_id=tool_call_id,
                                result=result,
                            )
                        elif tool_name in CONFIRM_TOOL_NAMES and ablation.merge_explorer_executor:
                            action = ensure_action_item(tool_name, tool_args)
                            action.confirm_id = action.confirm_id or str(
                                (result.get("confirmation") or {}).get("confirm_id") or ""
                            ).strip()
                            if result.get("status") == "confirmation_required":
                                action.status = "awaiting_confirmation"
                                _append_pending_confirmation(state, result)
                                return None
                            action.result = result
                            action.status = "error" if result.get("status") == "error" else "completed"
                            state.remember_tool(
                                agent_id=observe_agent.agent_id,
                                agent_role=observe_agent.role,
                                tool_name=tool_name,
                                tool_args=tool_args,
                                tool_call_id=tool_call_id,
                                result=result,
                            )
                            state.record_action_result(
                                action=action,
                                result_status=action.status,
                                result=result,
                            )
                        else:
                            return None
                        if tool_name == "list_user_jobs":
                            clarification_candidates = _failed_job_candidates_for_clarification(
                                user_message=goal_message,
                                routing=routing,
                                evidence=_extract_evidence_from_tool_records(state.tool_records),
                            )
                            if clarification_candidates:
                                return ToolLoopStopSignal(
                                    should_stop=True,
                                    summary=_build_diagnostic_clarification_answer(
                                        clarification_candidates
                                    ),
                                )
                        if (
                            tool_name == "prometheus_query"
                            and _looks_like_prometheus_storage_confirmation_request(goal_message)
                        ):
                            metric_summary = _build_prometheus_storage_metric_answer(result)
                            if metric_summary:
                                return ToolLoopStopSignal(
                                    should_stop=True,
                                    summary=metric_summary,
                                )
                        if _looks_like_queue_fairness_query(goal_message):
                            queue_summary = _build_queue_fairness_answer(state)
                            if queue_summary:
                                return ToolLoopStopSignal(
                                    should_stop=True,
                                    summary=queue_summary,
                                )
                        if _looks_like_submission_validation_request(goal_message, page_context):
                            validation_summary = _build_submission_validation_answer(state, goal_message)
                            if validation_summary:
                                return ToolLoopStopSignal(
                                    should_stop=True,
                                    summary=validation_summary,
                                )
                        return None

                    loop_outcome, loop_events = await run_role_tool_loop(
                        role_agent=observe_agent,
                        role_name=observe_agent.role,
                        system_prompt=loop_system_prompt,
                        user_prompt=loop_user_prompt,
                        allowed_tool_names=(
                            _actor_allowed_tools(state.goal.actor_role, enabled_tools)
                            if ablation.merge_explorer_executor
                            else [t for t in enabled_tools if t in READ_ONLY_TOOL_NAMES]
                        ),
                        max_tool_calls=state.runtime_config.subagent_max_iterations,
                        on_tool_result=on_explorer_tool_result,
                        loop_history_messages=history_messages,
                    )
                    record_agent_usage(observe_agent)
                    for tool_event in loop_events:
                        yield tool_event
                    exploration_summary = loop_outcome.summary
                except Exception:
                    logger.exception("Explorer native tool loop failed")

                if ablation.merge_explorer_executor:
                    awaiting_action = next(
                        (a for a in reversed(state.actions) if a.status == "awaiting_confirmation"),
                        None,
                    )
                    if awaiting_action is not None:
                        async for pause_event in emit_confirmation_pause(
                            role_agent_id=observe_agent.agent_id,
                            role_name=observe_agent.role,
                            summary="合并执行角色已发起确认型操作，等待用户确认",
                        ):
                            yield pause_event
                        return

                new_evidence = len(state.tool_records) - evidence_before

                # If tool loop produced nothing, try select_tools_with_llm fallback
                if new_evidence <= 0:
                    if ablation.merge_explorer_executor:
                        selected_tools = []
                    else:
                        try:
                            selected_tools = await explorer.select_tools_with_llm(
                                user_message=goal_message,
                                page_context=page_context,
                                plan_candidate_tools=prompt_candidate_tools,
                                plan_steps=state.plan.steps if state.plan else [],
                                enabled_tools=enabled_tools,
                                evidence_summary=evidence_summary_text,
                                attempted_tool_signatures=state.attempted_tool_signatures,
                                history_messages=history_messages,
                            )
                            record_agent_usage(explorer)
                        except Exception:
                            logger.exception("Explorer select_tools_with_llm failed")
                            selected_tools = []
                    if not selected_tools:
                        selected_tools = _fallback_read_tools_from_context(
                            user_message=goal_message,
                            page_context=page_context,
                            enabled_tools=enabled_tools,
                        )
                    for index, (tool_name, tool_args) in enumerate(selected_tools[:state.runtime_config.subagent_max_iterations], start=1):
                        if tool_name not in READ_ONLY_TOOL_NAMES:
                            continue
                        signature = _tool_signature(tool_name, tool_args)
                        if signature in state.attempted_tool_signatures:
                            continue
                        state.attempted_tool_signatures.append(signature)
                        result, tool_events = await call_tool(
                            role_agent_id=observe_agent.agent_id,
                            role_name=observe_agent.role,
                            tool_name=tool_name,
                            tool_args=tool_args,
                            tool_call_id=f"{observe_agent.agent_id}-tool-{state.loop_round}-{index}",
                        )
                        for tool_event in tool_events:
                            yield tool_event
                        state.remember_tool(
                            agent_id=observe_agent.agent_id,
                            agent_role=observe_agent.role,
                            tool_name=tool_name,
                            tool_args=tool_args,
                            tool_call_id=f"{observe_agent.agent_id}-tool-{state.loop_round}-{index}",
                            result=result,
                        )
                        new_evidence += 1

                if (
                    _looks_like_queue_fairness_query(goal_message)
                    and _has_tool_record(state, "analyze_queue_status")
                    and not _has_tool_record(state, "check_quota")
                    and "check_quota" in enabled_tools
                ):
                    signature = _tool_signature("check_quota", {})
                    if signature not in state.attempted_tool_signatures:
                        state.attempted_tool_signatures.append(signature)
                        result, tool_events = await call_tool(
                            role_agent_id=observe_agent.agent_id,
                            role_name=observe_agent.role,
                            tool_name="check_quota",
                            tool_args={},
                            tool_call_id=f"{observe_agent.agent_id}-tool-{state.loop_round}-queue-quota",
                        )
                        for tool_event in tool_events:
                            yield tool_event
                        state.remember_tool(
                            agent_id=observe_agent.agent_id,
                            agent_role=observe_agent.role,
                            tool_name="check_quota",
                            tool_args={},
                            tool_call_id=f"{observe_agent.agent_id}-tool-{state.loop_round}-queue-quota",
                            result=result,
                        )
                        new_evidence += 1
                        fairness_summary = _build_queue_fairness_answer(state)
                        if fairness_summary:
                            exploration_summary = fairness_summary

                # Handle single-target resolution for action intents
                evidence_dicts = _extract_evidence_from_tool_records(state.tool_records)
                clarification_candidates = _failed_job_candidates_for_clarification(
                    user_message=goal_message,
                    routing=routing,
                    evidence=evidence_dicts,
                )
                if clarification_candidates:
                    state.stop_reason = "awaiting_clarification"
                    state.final_answer = _build_diagnostic_clarification_answer(
                        clarification_candidates
                    )
                    yield await emit_final_answer(
                        agent_id=coordinator.agent_id,
                        agent_role=coordinator.role,
                        content=state.final_answer,
                        continuation_payload=_build_final_continuation_payload(state),
                    )
                    yield {"event": "done", "data": {"usageSummary": state.usage_summary.to_dict()}}
                    return

                if routing.requested_action and not routing.targets.job_name and routing.targets.scope != "all":
                    requested_job_types = set(_extract_requested_job_types(
                        user_message=goal_message,
                        page_context=page_context,
                    ))
                    candidate_jobs = _collect_jobs_from_evidence(
                        evidence_dicts,
                        status_filter=_candidate_status_filter_for_action(routing.requested_action),
                        job_type_filter=requested_job_types,
                    )
                    if len(candidate_jobs) == 1:
                        resolved_name = str(candidate_jobs[0].get("jobName") or "").strip().lower()
                        if resolved_name:
                            routing.targets.job_name = resolved_name
                            page_context["job_name"] = resolved_name
                            state.goal.page_context = page_context
                    elif len(candidate_jobs) > 1 and _user_requests_latest_job(goal_message):
                        latest_candidate = _sort_jobs_by_creation_desc(candidate_jobs)[0]
                        resolved_name = str(latest_candidate.get("jobName") or "").strip().lower()
                        if resolved_name:
                            routing.targets.job_name = resolved_name
                            routing.targets.scope = "single"
                            page_context["job_name"] = resolved_name
                            state.goal.page_context = page_context
                    elif len(candidate_jobs) > 1:
                        state.stop_reason = "awaiting_clarification"
                        state.final_answer = _build_action_clarification_answer(
                            action_intent=routing.requested_action,
                            candidate_jobs=candidate_jobs,
                        )
                        yield await emit_final_answer(
                            agent_id=coordinator.agent_id,
                            agent_role=coordinator.role,
                            content=state.final_answer,
                            continuation_payload=_build_final_continuation_payload(
                                state,
                                extra=_build_job_selection_continuation(
                                    action_intent=routing.requested_action,
                                    candidate_jobs=candidate_jobs,
                                    requested_all_scope=False,
                                ),
                            ),
                        )
                        yield {"event": "done", "data": {"usageSummary": state.usage_summary.to_dict()}}
                        return

                if (
                    routing.requested_action == "node_isolation"
                    and not _has_tool_record(state, "get_rdma_interface_status")
                    and "get_rdma_interface_status" in enabled_tools
                    and (node_name := extract_node_name(page_context, goal_message))
                ):
                    signature = _tool_signature("get_rdma_interface_status", {"node_name": node_name})
                    if signature not in state.attempted_tool_signatures:
                        state.attempted_tool_signatures.append(signature)
                        result, tool_events = await call_tool(
                            role_agent_id=observe_agent.agent_id,
                            role_name=observe_agent.role,
                            tool_name="get_rdma_interface_status",
                            tool_args={"node_name": node_name},
                            tool_call_id=f"{observe_agent.agent_id}-tool-{state.loop_round}-rdma",
                        )
                        for tool_event in tool_events:
                            yield tool_event
                        state.remember_tool(
                            agent_id=observe_agent.agent_id,
                            agent_role=observe_agent.role,
                            tool_name="get_rdma_interface_status",
                            tool_args={"node_name": node_name},
                            tool_call_id=f"{observe_agent.agent_id}-tool-{state.loop_round}-rdma",
                            result=result,
                        )
                        new_evidence += 1
                        evidence_dicts = _extract_evidence_from_tool_records(state.tool_records)

                # Build observation artifact
                compact_evidence = _compact_evidence_for_prompt(evidence_dicts)
                if exploration_summary:
                    obs_summary = exploration_summary
                else:
                    obs_summary = _build_evidence_summary_fallback(compact_evidence)
                # Determine if exploration is complete or was truncated by tool limit
                exploration_was_truncated = (
                    loop_outcome is not None
                    and loop_outcome.tool_calls >= state.runtime_config.subagent_max_iterations
                )
                state.observation = ObservationArtifact(
                    summary=obs_summary,
                    evidence=compact_evidence,
                    stage_complete=not exploration_was_truncated,
                )
                yield await emit(
                    "agent_status",
                    {
                        "agentId": observe_agent.agent_id,
                        "agentRole": observe_agent.role,
                        "status": "completed",
                        "summary": state.observation.summary,
                    },
                )

                if new_evidence > 0:
                    state.no_progress_count = 0
                else:
                    state.no_progress_count += 1

                continue

            # ----- ACT stage -----
            if next_stage == "act":
                yield await emit(
                    "agent_handoff",
                    {
                        "agentId": coordinator.agent_id,
                        "agentRole": coordinator.role,
                        "targetAgentId": executor.agent_id,
                        "targetAgentRole": executor.role,
                        "summary": "Coordinator 要求 Executor 推进执行阶段",
                        "status": "completed",
                    },
                )

                evidence_dicts = _extract_evidence_from_tool_records(state.tool_records)
                compact_evidence = _compact_evidence_for_prompt(evidence_dicts)
                evidence_summary_text = (
                    state.observation.summary if state.observation
                    else _build_evidence_summary_fallback(compact_evidence)
                )
                pending_actions = _pending_action_dicts(state.actions)
                requested_job_types = set(_extract_requested_job_types(
                    user_message=goal_message,
                    page_context=page_context,
                ))
                candidate_jobs = _collect_jobs_from_evidence(
                    evidence_dicts,
                    status_filter=_candidate_status_filter_for_action(routing.requested_action),
                    job_type_filter=requested_job_types,
                )
                if routing.requested_action == "stop" and routing.targets.scope == "all":
                    audit_jobs = _collect_audit_action_jobs_from_evidence(
                        evidence_dicts,
                        action_type="stop",
                    )
                    if audit_jobs:
                        candidate_jobs = audit_jobs

                frontier = state.action_frontier()

                if not frontier:
                    executor_enabled_tools = _actor_allowed_tools(
                        state.goal.actor_role,
                        enabled_tools,
                    )
                    # Try executor tool loop (can use read + write tools)
                    native_execution_summary = ""
                    native_tool_calls = 0
                    try:
                        loop_system_prompt, loop_user_prompt = executor.build_tool_loop_prompts(
                            user_message=goal_message,
                            page_context=page_context,
                            plan_summary=state.plan.summary if state.plan else "",
                            evidence_summary=evidence_summary_text,
                            compact_evidence=compact_evidence,
                            action_intent=routing.requested_action,
                            selected_job_name=routing.targets.job_name,
                            requested_scope=routing.targets.scope,
                            action_history=state.action_history,
                            pending_actions=pending_actions,
                            enabled_tools=executor_enabled_tools,
                            actor_role=state.goal.actor_role,
                        )

                        async def on_executor_tool_result(
                            tool_name: str,
                            tool_args: dict[str, Any],
                            tool_call_id: str,
                            result: dict[str, Any],
                        ) -> ToolLoopStopSignal | None:
                            if tool_name in READ_ONLY_TOOL_NAMES:
                                state.remember_tool(
                                    agent_id=executor.agent_id,
                                    agent_role=executor.role,
                                    tool_name=tool_name,
                                    tool_args=tool_args,
                                    tool_call_id=tool_call_id,
                                    result=result,
                                )
                                return None

                            action = ensure_action_item(tool_name, tool_args)
                            action.confirm_id = action.confirm_id or str(
                                (result.get("confirmation") or {}).get("confirm_id") or ""
                            ).strip()
                            if result.get("status") == "confirmation_required":
                                action.status = "awaiting_confirmation"
                                _append_pending_confirmation(state, result)
                                return None

                            action.result = result
                            action.status = "error" if result.get("status") == "error" else "completed"
                            state.remember_tool(
                                agent_id=executor.agent_id,
                                agent_role=executor.role,
                                tool_name=tool_name,
                                tool_args=tool_args,
                                tool_call_id=tool_call_id,
                                result=result,
                            )
                            state.record_action_result(
                                action=action,
                                result_status=action.status,
                                result=result,
                            )
                            return None

                        loop_outcome, loop_events = await run_role_tool_loop(
                            role_agent=executor,
                            role_name=executor.role,
                            system_prompt=loop_system_prompt,
                            user_prompt=loop_user_prompt,
                            allowed_tool_names=executor_enabled_tools,
                            max_tool_calls=max(1, state.runtime_config.max_actions_per_round),
                            on_tool_result=on_executor_tool_result,
                            loop_history_messages=history_messages,
                        )
                        record_agent_usage(executor)
                        for tool_event in loop_events:
                            yield tool_event
                        native_execution_summary = loop_outcome.summary
                        native_tool_calls = loop_outcome.tool_calls
                    except Exception:
                        logger.exception("Executor native tool loop failed")

                    # Check if awaiting confirmation after tool loop
                    awaiting_action = next(
                        (a for a in reversed(state.actions) if a.status == "awaiting_confirmation"),
                        None,
                    )
                    if awaiting_action is not None:
                        async for pause_event in emit_confirmation_pause(
                            role_agent_id=executor.agent_id,
                            role_name=executor.role,
                            summary="Executor 已发起高风险操作，等待用户确认",
                        ):
                            yield pause_event
                        return

                    # If native tool loop produced results, record execution
                    write_intent_still_needs_action = (
                        _has_write_intent(routing)
                        and not _has_terminal_write_result(state)
                        and not _has_awaiting_write_confirmation(state)
                    )
                    if native_tool_calls > 0 and not write_intent_still_needs_action:
                        state.execution = ExecutionArtifact(
                            summary=native_execution_summary or _build_action_history_summary(state.action_history),
                            actions=[a.to_dict() for a in state.actions],
                        )
                        yield await emit(
                            "agent_status",
                            {
                                "agentId": executor.agent_id,
                                "agentRole": executor.role,
                                "status": "completed",
                                "summary": state.execution.summary,
                            },
                        )
                        state.no_progress_count = 0
                        continue

                    # Structured action fast path
                    proposals: list[dict[str, Any]] = []
                    if routing.requested_action in {"resubmit", "stop", "delete"}:
                        proposals = _fallback_executor_actions(
                            action_intent=routing.requested_action,
                            resolved_job_name=routing.targets.job_name,
                            candidate_jobs=candidate_jobs,
                            requested_scope=routing.targets.scope,
                            enabled_tools=executor_enabled_tools,
                        )

                    if not proposals and routing.requested_action == "create":
                        proposals = _fallback_submission_actions(
                            user_message=goal_message,
                            evidence=evidence_dicts,
                            enabled_tools=executor_enabled_tools,
                        )

                    if not proposals and routing.requested_action == "scale":
                        proposals = _fallback_k8s_scale_actions(
                            user_message=goal_message,
                            page_context=page_context,
                            evidence=evidence_dicts,
                            enabled_tools=executor_enabled_tools,
                        )

                    if not proposals:
                        proposals = _fallback_k8s_node_isolation_actions(
                            user_message=goal_message,
                            page_context=page_context,
                            evidence=evidence_dicts,
                            enabled_tools=executor_enabled_tools,
                        )

                    if not proposals:
                        try:
                            proposals = await executor.decide_actions_with_llm(
                                user_message=goal_message,
                                page_context=page_context,
                                plan_summary=state.plan.summary if state.plan else "",
                                evidence_summary=evidence_summary_text,
                                compact_evidence=compact_evidence,
                                action_intent=routing.requested_action,
                                selected_job_name=routing.targets.job_name,
                                requested_scope=routing.targets.scope,
                                action_history=state.action_history,
                                pending_actions=pending_actions,
                                enabled_tools=executor_enabled_tools,
                                history_messages=history_messages,
                                actor_role=state.goal.actor_role,
                            )
                            record_agent_usage(executor)
                        except Exception:
                            logger.exception("Executor decide_actions_with_llm failed")
                            proposals = []

                    if not proposals:
                        if routing.requested_action == "create":
                            proposals = _fallback_submission_actions(
                                user_message=goal_message,
                                evidence=evidence_dicts,
                                enabled_tools=executor_enabled_tools,
                            )
                        elif routing.requested_action == "scale":
                            proposals = _fallback_k8s_scale_actions(
                                user_message=goal_message,
                                page_context=page_context,
                                evidence=evidence_dicts,
                                enabled_tools=executor_enabled_tools,
                            )
                        else:
                            proposals = _fallback_k8s_node_isolation_actions(
                                user_message=goal_message,
                                page_context=page_context,
                                evidence=evidence_dicts,
                                enabled_tools=executor_enabled_tools,
                            )
                            if not proposals:
                                proposals = _fallback_executor_actions(
                                    action_intent=routing.requested_action,
                                    resolved_job_name=routing.targets.job_name,
                                    candidate_jobs=candidate_jobs,
                                    requested_scope=routing.targets.scope,
                                    enabled_tools=executor_enabled_tools,
                                )

                    _merge_action_proposals(state.actions, proposals)
                    frontier = state.action_frontier()

                # Execute frontier actions
                if not frontier:
                    state.no_progress_count += 1
                    continue

                executed_actions: list[dict[str, Any]] = []
                awaiting_actions: list[MultiAgentActionItem] = []
                all_skipped = True
                for action in frontier[:state.runtime_config.max_actions_per_round]:
                    signature = _tool_signature(action.tool_name, action.tool_args)
                    if signature in state.attempted_tool_signatures:
                        action.status = "skipped"
                        continue
                    all_skipped = False
                    state.attempted_tool_signatures.append(signature)
                    action.status = "running"
                    result, tool_events = await call_tool(
                        role_agent_id=executor.agent_id,
                        role_name=executor.role,
                        tool_name=action.tool_name,
                        tool_args=action.tool_args,
                        tool_call_id=f"{executor.agent_id}-{action.action_id}",
                    )
                    for tool_event in tool_events:
                        yield tool_event

                    if result.get("status") == "confirmation_required":
                        action.status = "awaiting_confirmation"
                        action.confirm_id = str(
                            (result.get("confirmation") or {}).get("confirm_id") or ""
                        ).strip()
                        _append_pending_confirmation(state, result)
                        awaiting_actions.append(action)
                        continue

                    action.result = result
                    action.status = "error" if result.get("status") == "error" else "completed"
                    state.remember_tool(
                        agent_id=executor.agent_id,
                        agent_role=executor.role,
                        tool_name=action.tool_name,
                        tool_args=action.tool_args,
                        tool_call_id=f"{executor.agent_id}-{action.action_id}",
                        result=result,
                    )
                    state.record_action_result(
                        action=action,
                        result_status=action.status,
                        result=result,
                    )
                    executed_actions.append(
                        {
                            "title": action.title or action.tool_name,
                            "status": action.status,
                            "result": result,
                        }
                    )

                if awaiting_actions:
                    async for pause_event in emit_confirmation_pause(
                        role_agent_id=executor.agent_id,
                        role_name=executor.role,
                        summary=(
                            f"Executor 已发起 {len(awaiting_actions)} 个高风险操作，等待用户确认"
                            if len(awaiting_actions) > 1
                            else "Executor 已发起高风险操作，等待用户确认"
                        ),
                    ):
                        yield pause_event
                    return

                if executed_actions:
                    try:
                        execution_summary_result = await executor.summarize_action(
                            user_message=goal_message,
                            plan_summary=state.plan.summary if state.plan else "",
                            action_result={"actions": executed_actions},
                            history_messages=history_messages,
                        )
                        record_agent_usage(executor)
                    except Exception:
                        logger.exception("Executor summarize_action failed")
                        execution_summary_result = RoleExecutionResult(
                            summary=_build_action_history_summary(state.action_history),
                            status="completed",
                        )

                    state.execution = ExecutionArtifact(
                        summary=execution_summary_result.summary,
                        actions=[a.to_dict() for a in state.actions],
                    )
                    yield await emit(
                        "agent_status",
                        {
                            "agentId": executor.agent_id,
                            "agentRole": executor.role,
                            "status": "completed",
                            "summary": state.execution.summary,
                        },
                    )
                    if all_skipped:
                        # All frontier actions were duplicates — no real progress
                        state.no_progress_count += 1
                    else:
                        state.no_progress_count = 0
                else:
                    state.no_progress_count += 1

                continue

            # ----- REPLAN stage -----
            if next_stage == "replan":
                yield await emit(
                    "agent_handoff",
                    {
                        "agentId": coordinator.agent_id,
                        "agentRole": coordinator.role,
                        "targetAgentId": planner.agent_id,
                        "targetAgentRole": planner.role,
                        "summary": "Coordinator 要求 Planner 基于新证据重新规划",
                        "status": "completed",
                    },
                )
                evidence_dicts = _extract_evidence_from_tool_records(state.tool_records)
                compact_evidence = _compact_evidence_for_prompt(evidence_dicts)
                evidence_summary_text = (
                    state.observation.summary if state.observation
                    else _build_evidence_summary_fallback(compact_evidence)
                )
                action_history_summary = _build_action_history_summary(state.action_history)
                last_replan_signature = current_progress_signature
                try:
                    plan_result = await planner.plan(
                        user_message=goal_message,
                        page_context=page_context,
                        capabilities=capabilities,
                        actor_role=state.goal.actor_role,
                        evidence_summary=evidence_summary_text,
                        action_history_summary=action_history_summary,
                        replan_reason=(
                            f"第 {state.loop_round} 轮重规划。"
                            f"已有观测: {state.observation.summary if state.observation else '(无)'}。"
                            f"已有执行: {state.execution.summary if state.execution else '(无)'}。"
                            "请基于最新证据决定下一步：继续调查、执行操作、还是直接总结回答用户。"
                        ),
                        history_messages=history_messages,
                    )
                    record_agent_usage(planner)
                except Exception:
                    logger.exception("Planner replan failed")
                    plan_result = RoleExecutionResult(
                        summary="重规划失败，基于现有结果收束",
                        metadata={"plan_output": {"steps": ["输出最终总结"], "candidate_tools": []}},
                    )
                plan_output = (plan_result.metadata or {}).get("plan_output", {})
                new_steps = plan_output.get("steps", [])

                # If planner returns empty steps or a finalize step, we're done
                if not new_steps or (
                    len(new_steps) == 1
                    and any(kw in new_steps[0].lower() for kw in ("总结", "输出", "回复", "finalize", "完成"))
                ):
                    state.plan = PlanArtifact(
                        summary=plan_output.get("raw_summary") or plan_result.summary,
                        steps=[],
                        candidate_tools=plan_output.get("candidate_tools", []),
                        risk=plan_output.get("risk", "low"),
                    )
                    yield await emit(
                        "agent_status",
                        {
                            "agentId": planner.agent_id,
                            "agentRole": planner.role,
                            "status": "completed",
                            "summary": _build_plan_status_summary(state.plan, prefix="重规划完成"),
                        },
                    )
                    break  # → finalize

                state.plan = PlanArtifact(
                    summary=plan_output.get("raw_summary") or plan_result.summary,
                    steps=new_steps,
                    candidate_tools=plan_output.get("candidate_tools", []),
                    risk=plan_output.get("risk", "low"),
                )
                # Clear execution so _determine_next_stage can proceed to next step
                state.execution = None
                yield await emit(
                    "agent_status",
                    {
                        "agentId": planner.agent_id,
                        "agentRole": planner.role,
                        "status": "completed",
                        "summary": _build_plan_status_summary(state.plan, prefix="重规划完成"),
                    },
                )
                continue

        # =================================================================
        # STEP 6: Finalize
        # =================================================================
        if not state.final_answer:
            verification_summary = last_verification_result.summary if last_verification_result else ""
            verification_verdict = _extract_verification_verdict(last_verification_result)
            if state.stop_reason in {"max_rounds", "no_progress"}:
                state.final_answer = _build_runtime_fallback_final_answer(state)
            elif (
                _has_tool_record(state, "get_health_overview")
                and not _has_write_intent(routing)
            ):
                state.final_answer = _build_health_overview_answer(state)
            elif (
                _has_tool_record(state, "get_admin_ops_report")
                and not _has_write_intent(routing)
                and (admin_ops_answer := _build_admin_ops_report_answer(state))
            ):
                state.final_answer = admin_ops_answer
            elif (
                not _has_write_intent(routing)
                and (cluster_noop_answer := _build_cluster_health_noop_answer(state))
            ):
                state.final_answer = cluster_noop_answer
            elif _write_intent_has_no_allowed_write_tool(
                actor_role=state.goal.actor_role,
                enabled_tools=enabled_tools,
                raw_enabled_tools=raw_enabled_tools,
                routing=routing,
            ):
                state.final_answer = _build_actor_permission_denied_answer(state, enabled_tools)
            elif routing.operation_mode == "write" and not state.tool_records and not state.action_history and not state.execution:
                state.final_answer = (
                    "当前没有任何已落地的工具执行或确认记录，不能声称写操作已经完成。"
                    "请重新发起，并以确认卡与工具结果为准。"
                )
            elif verification_verdict == "missing_evidence":
                grounded_summary = _build_terminal_action_answer(state)
                if not grounded_summary:
                    if state.execution:
                        grounded_summary = state.execution.summary
                    elif state.observation:
                        grounded_summary = state.observation.summary
                    else:
                        grounded_summary = ""
                state.final_answer = "当前不能确认任务已经完成。"
                if verification_summary:
                    state.final_answer += f"\n\n原因：{verification_summary}"
                if grounded_summary:
                    state.final_answer += f"\n\n已知信息：\n{grounded_summary}"
            else:
                # Coordinator synthesizes the final answer from structured artifacts.
                evidence_dicts = _extract_evidence_from_tool_records(state.tool_records)
                compact_evidence = _compact_evidence_for_prompt(evidence_dicts)
                terminal_answer = _build_terminal_action_answer(state)
                executor_summary = terminal_answer or (state.execution.summary if state.execution else "")
                verifier_summary = (
                    f"[{verification_verdict or 'pass'}] {verification_summary}"
                    if verification_summary
                    else ""
                )
                try:
                    final_summary_result = await coordinator.summarize(
                        user_message=goal_message,
                        plan_summary=state.plan.summary if state.plan else "",
                        evidence_summary=state.observation.summary if state.observation else "",
                        compact_evidence=compact_evidence,
                        executor_summary=executor_summary,
                        verifier_summary=verifier_summary,
                    )
                    state.final_answer = final_summary_result.summary
                    record_agent_usage(coordinator)
                except Exception:
                    logger.exception("Coordinator final summarization failed")
                    state.final_answer = (
                        terminal_answer
                        or (state.execution.summary if state.execution
                            else state.observation.summary if state.observation
                            else "Agent 执行完成，但生成最终答复时出错。")
                    )

        if not state.stop_reason:
            state.stop_reason = "completed"

        yield await emit_final_answer(
            agent_id=coordinator.agent_id,
            agent_role=coordinator.role,
            content=state.final_answer,
            continuation_payload=_build_final_continuation_payload(state),
        )
        yield {"event": "done", "data": {"usageSummary": state.usage_summary.to_dict()}}
