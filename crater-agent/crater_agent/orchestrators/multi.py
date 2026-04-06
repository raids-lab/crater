"""Loop-based multi-agent orchestrator."""

from __future__ import annotations

import json
import logging
from typing import Any, AsyncIterator

from crater_agent.agents.base import RoleExecutionResult
from crater_agent.agents.coordinator import CoordinatorAgent, LoopDecision
from crater_agent.agents.executor import ExecutorAgent
from crater_agent.agents.explorer import ExplorerAgent
from crater_agent.agents.general import GeneralPurposeAgent
from crater_agent.agents.guide import GuideAgent
from crater_agent.agents.planner import PlannerAgent
from crater_agent.agents.verifier import VerifierAgent
from crater_agent.llm.client import ModelClientFactory
from crater_agent.orchestrators.state import (
    MultiAgentActionItem,
    MultiAgentRoleOutput,
    MultiAgentRuntimeGuard,
    MultiAgentTurnState,
)
from crater_agent.report_utils import build_pipeline_report_payload
from crater_agent.scenarios import (
    NODE_ANALYSIS_ENTRYPOINT,
    OPS_REPORT_ENTRYPOINT,
    extract_node_name,
    infer_entrypoint,
)
from crater_agent.tools.definitions import (
    READ_ONLY_TOOL_NAMES,
    is_tool_allowed_for_role,
)
from crater_agent.tools.executor import GoBackendToolExecutor, ToolExecutorProtocol

logger = logging.getLogger(__name__)

CREATE_JOB_TOOL_NAMES = {"create_training_job", "create_jupyter_job"}


def _normalized_text(value: Any) -> str:
    return str(value or "").strip().lower()


def _looks_like_submission_request(user_message: str, enabled_tools: list[str]) -> bool:
    normalized = _normalized_text(user_message)
    if not normalized:
        return False
    if not any(tool_name in CREATE_JOB_TOOL_NAMES for tool_name in enabled_tools):
        return False
    keywords = (
        "提交",
        "创建",
        "新建",
        "开一个",
        "开个",
        "训练任务",
        "训练作业",
        "llm任务",
        "llm 作业",
        "jupyter",
        "notebook",
    )
    return any(keyword in normalized for keyword in keywords)


def _looks_like_diagnosis_request(user_message: str, page_context: dict[str, Any]) -> bool:
    if _normalized_text(page_context.get("job_name") or page_context.get("jobName")):
        return True
    normalized = _normalized_text(user_message)
    keywords = ("诊断", "分析", "失败", "报错", "日志", "oom", "pending", "排队", "原因", "为什么")
    return any(keyword in normalized for keyword in keywords)


def _derive_runtime_scenario(
    *,
    route: str,
    action_intent: str | None,
    user_message: str,
    page_context: dict[str, Any],
    enabled_tools: list[str],
    workflow: dict[str, Any],
    resume_context: dict[str, Any],
) -> str:
    persisted = _normalized_text(
        workflow.get("runtime_scenario") or resume_context.get("runtime_scenario")
    )
    if persisted:
        return persisted
    entrypoint = infer_entrypoint(page_context, user_message)
    if entrypoint == NODE_ANALYSIS_ENTRYPOINT:
        return "node_analysis"
    if entrypoint == OPS_REPORT_ENTRYPOINT:
        return "ops_report"
    if route in {"guide", "general"}:
        return route
    if action_intent or workflow.get("action_intent") or resume_context.get("action_intent"):
        return "action"
    if _looks_like_submission_request(user_message, enabled_tools):
        return "submission"
    route_hint = _normalized_text(page_context.get("route") or page_context.get("url"))
    if route_hint.startswith("/admin"):
        return "ops"
    if _looks_like_diagnosis_request(user_message, page_context):
        return "diagnosis"
    return "query"


def _build_runtime_guard_for_scenario(scenario: str) -> MultiAgentRuntimeGuard:
    normalized = _normalized_text(scenario) or "query"
    presets = {
        "query": MultiAgentRuntimeGuard(
            scenario="query",
            max_loop_iterations=4,
            max_replans=1,
            max_frontier_actions=1,
            max_read_only_explore_rounds=2,
            max_read_tool_calls=2,
            max_evidence_items=4,
            max_verifications=1,
            max_no_progress_rounds=1,
            max_budget_tokens=6000,
            structured_retry_limit=1,
        ),
        "ops": MultiAgentRuntimeGuard(
            scenario="ops",
            max_loop_iterations=5,
            max_replans=1,
            max_frontier_actions=1,
            max_read_only_explore_rounds=3,
            max_read_tool_calls=2,
            max_evidence_items=6,
            max_verifications=1,
            max_no_progress_rounds=1,
            max_budget_tokens=8000,
            structured_retry_limit=1,
        ),
        "node_analysis": MultiAgentRuntimeGuard(
            scenario="node_analysis",
            max_loop_iterations=4,
            max_replans=1,
            max_frontier_actions=1,
            max_read_only_explore_rounds=2,
            max_read_tool_calls=2,
            max_evidence_items=4,
            max_verifications=1,
            max_no_progress_rounds=1,
            max_budget_tokens=7000,
            structured_retry_limit=1,
        ),
        "ops_report": MultiAgentRuntimeGuard(
            scenario="ops_report",
            max_loop_iterations=4,
            max_replans=1,
            max_frontier_actions=1,
            max_read_only_explore_rounds=2,
            max_read_tool_calls=2,
            max_evidence_items=4,
            max_verifications=1,
            max_no_progress_rounds=1,
            max_budget_tokens=7000,
            structured_retry_limit=1,
        ),
        "diagnosis": MultiAgentRuntimeGuard(
            scenario="diagnosis",
            max_loop_iterations=6,
            max_replans=2,
            max_frontier_actions=2,
            max_read_only_explore_rounds=2,
            max_read_tool_calls=6,
            max_evidence_items=8,
            max_verifications=1,
            max_no_progress_rounds=2,
            max_budget_tokens=10000,
            structured_retry_limit=1,
        ),
        "submission": MultiAgentRuntimeGuard(
            scenario="submission",
            max_loop_iterations=5,
            max_replans=1,
            max_frontier_actions=1,
            max_read_only_explore_rounds=1,
            max_read_tool_calls=4,
            max_evidence_items=6,
            max_verifications=0,
            max_no_progress_rounds=1,
            max_budget_tokens=8000,
            structured_retry_limit=1,
        ),
        "action": MultiAgentRuntimeGuard(
            scenario="action",
            max_loop_iterations=4,
            max_replans=1,
            max_frontier_actions=4,
            max_read_only_explore_rounds=1,
            max_read_tool_calls=2,
            max_evidence_items=4,
            max_verifications=1,
            max_no_progress_rounds=1,
            max_budget_tokens=7000,
            structured_retry_limit=1,
        ),
        "guide": MultiAgentRuntimeGuard(scenario="guide", max_verifications=0),
        "general": MultiAgentRuntimeGuard(scenario="general", max_verifications=0),
    }
    return presets.get(normalized, presets["query"])


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


def _collect_jobs_from_evidence(
    evidence: list[dict[str, Any]],
    *,
    status_filter: set[str] | None = None,
) -> list[dict[str, Any]]:
    jobs: list[dict[str, Any]] = []
    normalized_status_filter = {status.lower() for status in (status_filter or set())}
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
            jobs.append(
                {
                    "jobName": str(job.get("jobName") or "").strip(),
                    "name": str(job.get("name") or "").strip(),
                    "status": status,
                    "jobType": str(job.get("jobType") or "").strip(),
                    "creationTimestamp": str(job.get("creationTimestamp") or "").strip(),
                }
            )
    return _dedupe_jobs(jobs)


def _truncate_text(value: Any, max_chars: int = 320) -> str:
    text = str(value or "").strip()
    if len(text) <= max_chars:
        return text
    return text[:max_chars] + "..."


def _compact_tool_result_for_prompt(tool_name: str, result: dict[str, Any]) -> dict[str, Any]:
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
                    "jobName": job.get("jobName"),
                    "name": job.get("name"),
                    "status": job.get("status"),
                    "jobType": job.get("jobType"),
                    "creationTimestamp": job.get("creationTimestamp"),
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
        return {
            "tool_name": tool_name,
            "jobName": payload.get("jobName") or payload.get("name"),
            "status": payload.get("status"),
            "jobType": payload.get("jobType"),
            "resources": payload.get("resources"),
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
                ),
            }
        )
    return compact


def _format_job_candidates(jobs: list[dict[str, Any]], limit: int = 8) -> str:
    lines: list[str] = []
    for index, job in enumerate(jobs[:limit], start=1):
        display_name = str(job.get("name") or "").strip()
        display_suffix = f" / {display_name}" if display_name else ""
        status = str(job.get("status") or "").strip() or "unknown"
        lines.append(f"{index}. {job.get('jobName')}{display_suffix} ({status})")
    return "\n".join(lines)


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
        "请直接回复一个具体的 jobName；如果你想处理全部候选作业，也可以直接回复“全部”。"
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


def _forced_exploration_tools(
    *,
    action_intent: str | None,
    resolved_job_name: str | None,
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
        add("list_user_jobs", {"statuses": ["Failed"], "days": 30, "limit": 12})
        add("get_health_overview", {"days": 7})
        return forced

    add("list_user_jobs", {"days": 30, "limit": 12})
    return forced


def _candidate_status_filter_for_action(action_intent: str | None) -> set[str] | None:
    if action_intent == "resubmit":
        return {"failed"}
    if action_intent == "stop":
        return {"running", "pending", "inqueue", "prequeue"}
    return None


def _fallback_read_tools_from_context(
    *,
    page_context: dict[str, Any],
    enabled_tools: list[str],
) -> list[tuple[str, dict[str, Any]]]:
    selected: list[tuple[str, dict[str, Any]]] = []
    enabled = set(enabled_tools)
    job_name = str(page_context.get("job_name") or page_context.get("jobName") or "").strip().lower()
    node_name = str(page_context.get("node_name") or page_context.get("nodeName") or "").strip()
    route_hint = str(page_context.get("route") or page_context.get("url") or "").strip().lower()

    def add(tool_name: str, args: dict[str, Any]) -> None:
        if tool_name not in enabled or tool_name not in READ_ONLY_TOOL_NAMES:
            return
        if any(existing_tool == tool_name and existing_args == args for existing_tool, existing_args in selected):
            return
        selected.append((tool_name, args))

    if job_name:
        add("get_job_detail", {"job_name": job_name})
        return selected

    if node_name:
        add("get_node_detail", {"node_name": node_name})
        return selected

    if route_hint.startswith("/admin"):
        add("get_cluster_health_overview", {"days": 7})
        add("list_cluster_jobs", {"days": 7, "limit": 8})
        return selected

    add("list_user_jobs", {"days": 30, "limit": 8})
    add("get_health_overview", {"days": 7})
    return selected


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


def _tool_signature(tool_name: str, tool_args: dict[str, Any]) -> str:
    return json.dumps(
        {
            "tool_name": tool_name,
            "tool_args": tool_args,
        },
        ensure_ascii=False,
        sort_keys=True,
    )


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


def _build_structured_exploration_summary(
    *,
    action_intent: str | None,
    resolved_job_name: str | None,
    requested_scope: str,
    candidate_jobs: list[dict[str, Any]],
    compact_evidence: list[dict[str, Any]],
) -> str:
    action_label = {
        "resubmit": "重新提交",
        "stop": "停止",
        "delete": "删除",
    }.get(action_intent or "", "处理")
    if requested_scope == "all" and candidate_jobs:
        listed = ", ".join(
            str(job.get("jobName") or "").strip()
            for job in candidate_jobs[:4]
            if str(job.get("jobName") or "").strip()
        )
        suffix = " 等" if len(candidate_jobs) > 4 else ""
        return (
            f"已确认当前请求需要{action_label}全部候选作业；"
            f"当前共定位到 {len(candidate_jobs)} 个候选对象"
            + (f"，包括 {listed}{suffix}。" if listed else "。")
        )
    if resolved_job_name:
        return f"已完成目标作业 {resolved_job_name} 的最小只读核验，可进入{action_label}确认。"
    return _build_evidence_summary_fallback(compact_evidence)


def _supports_structured_action_fast_path(action_intent: str | None) -> bool:
    return action_intent in {"resubmit", "stop", "delete"}


def _should_use_structured_action_fast_path(
    *,
    action_intent: str | None,
    resolved_job_name: str | None,
    requested_scope: str,
    candidate_jobs: list[dict[str, Any]],
    enabled_tools: list[str],
) -> bool:
    if not _supports_structured_action_fast_path(action_intent):
        return False
    proposals = _fallback_executor_actions(
        action_intent=action_intent,
        resolved_job_name=resolved_job_name,
        candidate_jobs=candidate_jobs,
        requested_scope=requested_scope,
        enabled_tools=enabled_tools,
    )
    return bool(proposals)


def _build_terminal_action_answer(state: MultiAgentTurnState) -> str | None:
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
            )
            ,
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
    prefers_jupyter = any(token in _normalized_text(user_message) for token in ("jupyter", "notebook", "交互式"))
    tool_name = ""
    if prefers_jupyter and "create_jupyter_job" in enabled_tools:
        tool_name = "create_jupyter_job"
    elif "create_training_job" in enabled_tools:
        tool_name = "create_training_job"
    elif "create_jupyter_job" in enabled_tools:
        tool_name = "create_jupyter_job"
    if not tool_name:
        return []

    gpu_model = _extract_requested_gpu_model(user_message)
    image_link = _extract_recommended_image(evidence)
    if tool_name == "create_jupyter_job":
        tool_args: dict[str, Any] = {
            "name": "workspace-notebook",
            "cpu": "4",
            "memory": "16Gi",
            "gpu_count": 1 if gpu_model else None,
            "gpu_model": gpu_model or None,
        }
        if image_link:
            tool_args["image_link"] = image_link
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


def _controller_step_count(trace: list[dict[str, Any]], step: str) -> int:
    return sum(
        1
        for item in trace
        if isinstance(item, dict) and str(item.get("step") or "").strip().lower() == step
    )


def _runtime_guard_stop_reason(
    state: MultiAgentTurnState,
    *,
    no_progress_rounds: int,
) -> str | None:
    guard = state.runtime_guard
    if guard is None:
        return None
    if guard.max_budget_tokens > 0 and state.usage_summary.total_tokens >= guard.max_budget_tokens:
        return "max_llm_budget_reached"
    if guard.max_no_progress_rounds > 0 and no_progress_rounds >= guard.max_no_progress_rounds:
        return "max_no_progress_reached"
    if (
        guard.max_evidence_items > 0
        and len(state.evidence) >= guard.max_evidence_items
        and not state.action_frontier()
    ):
        return "max_evidence_budget_reached"
    return None


def _should_skip_optional_llm_calls(state: MultiAgentTurnState) -> bool:
    return state.stop_reason == "max_llm_budget_reached"


def _build_runtime_fallback_final_answer(state: MultiAgentTurnState) -> str:
    reason_label = {
        "max_loop_iterations_reached": "达到最大迭代次数",
        "max_read_budget_reached": "达到只读工具预算上限",
        "max_evidence_budget_reached": "达到证据预算上限",
        "max_llm_budget_reached": "达到 LLM 预算上限",
        "max_no_progress_reached": "连续多轮无明显进展",
        "verification_budget_reached": "达到验证预算上限",
    }.get(state.stop_reason, "")
    body = (
        state.execution.summary
        if state.execution
        else state.exploration.summary
        if state.exploration
        else state.verification.summary
        if state.verification
        else "本轮已经停止，但没有足够的结构化结果可直接总结。"
    )
    if not reason_label:
        return body
    return f"本轮已按运行时保护机制收束：{reason_label}。\n\n{body}"


def _fallback_loop_decision(state: MultiAgentTurnState) -> LoopDecision:
    if state.action_frontier():
        return LoopDecision(step="execute", rationale="存在已准备好的待执行动作")
    if state.actions and any(item.status == "awaiting_confirmation" for item in state.actions):
        return LoopDecision(step="finalize", rationale="当前等待用户确认")
    if state.verification and state.verification.status in {"pass", "risk"}:
        return LoopDecision(step="finalize", rationale="已有验证结论")
    if state.action_history and state.verification is None:
        return LoopDecision(step="verify", rationale="已有动作结果，进入验证")
    if not state.evidence:
        return LoopDecision(step="explore", rationale="尚无证据")
    return LoopDecision(step="finalize", rationale="已有证据，可收束")


def _has_minimal_target_evidence(
    *,
    evidence: list[dict[str, Any]],
    resolved_job_name: str | None,
    requested_scope: str,
    candidate_jobs: list[dict[str, Any]],
) -> bool:
    if requested_scope == "all" and candidate_jobs:
        return True
    if not resolved_job_name:
        return False
    target = resolved_job_name.strip().lower()
    for entry in evidence:
        if not isinstance(entry, dict) or entry.get("tool_name") != "get_job_detail":
            continue
        result = entry.get("result") or {}
        payload = result.get("result", result) if isinstance(result, dict) else {}
        if not isinstance(payload, dict):
            continue
        job_name = str(payload.get("jobName") or payload.get("name") or "").strip().lower()
        if job_name == target:
            return True
    return False


class MultiAgentOrchestrator:
    def __init__(self, tool_executor: ToolExecutorProtocol | None = None):
        self.tool_executor = tool_executor or GoBackendToolExecutor()

    async def stream(self, *, request: Any, model_factory: ModelClientFactory) -> AsyncIterator[dict]:
        state = MultiAgentTurnState.from_request(request)
        page_context = dict(state.page_context)
        capabilities = state.capabilities
        continuation = dict(state.continuation)
        enabled_tools = state.enabled_tools
        actor_role = state.actor_role
        history_excerpt = state.recent_history_excerpt()
        goal_message = state.original_user_message

        def make_agent(cls, agent_id: str, role: str):
            return cls(
                agent_id=agent_id,
                role=role,
                llm=model_factory.create(purpose=role, orchestration_mode="multi_agent"),
                json_retry_limit=(
                    state.runtime_guard.structured_retry_limit if state.runtime_guard else 1
                ),
            )

        coordinator = make_agent(CoordinatorAgent, state.root_agent_id, "coordinator")

        def record_agent_usage(agent: Any) -> None:
            state.record_llm_usage(getattr(agent, "last_usage", None))

        async def emit(event: str, data: dict[str, Any]) -> dict[str, Any]:
            return {"event": event, "data": {"turnId": request.turn_id, **data}}

        async def emit_final_answer(
            *,
            agent_id: str,
            agent_role: str,
            content: str,
            continuation_payload: dict[str, Any] | None = None,
        ) -> dict[str, Any]:
            payload: dict[str, Any] = {
                "sessionId": request.session_id,
                "agentId": agent_id,
                "agentRole": agent_role,
                "content": content,
                "stopReason": state.stop_reason or "completed",
                "runtimeScenario": state.runtime_scenario or "query",
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
            result = await self.tool_executor.execute(
                tool_name=tool_name,
                tool_args=tool_args,
                session_id=request.session_id,
                user_id=int(state.actor.get("user_id") or 0),
                turn_id=request.turn_id,
                tool_call_id=tool_call_id,
                agent_id=role_agent_id,
                agent_role=role_name,
            )
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
                    },
                )
            )
            report_payload = build_pipeline_report_payload(tool_name, result)
            if report_payload:
                events.append(await emit("pipeline_report", report_payload))
            return result, events

        yield await emit(
            "agent_run_started",
            {
                "sessionId": request.session_id,
                "agentId": coordinator.agent_id,
                "agentRole": coordinator.role,
                "status": "started",
                "summary": (
                    "确认结果已回流，继续执行上一轮计划"
                    if state.resume_context
                    else "多 Agent 协作已启动"
                ),
            },
        )

        if state.resume_context:
            turn_context = coordinator._parse_turn_context(
                {
                    "route": "diagnostic",
                    "action_intent": state.resume_context.get("action_intent")
                    or state.workflow.get("action_intent"),
                    "selected_job_name": state.workflow.get("selected_job_name"),
                    "requested_scope": state.workflow.get("requested_scope") or "unspecified",
                    "rationale": "resume_after_confirmation",
                }
            )
        else:
            try:
                turn_context = await coordinator.decide_turn_context(
                    user_message=request.message,
                    page_context=page_context,
                    continuation=continuation,
                    recent_history_excerpt=history_excerpt,
                    capabilities=capabilities,
                )
                record_agent_usage(coordinator)
            except Exception:
                logger.exception("Coordinator turn-context decision failed, falling back to general")
                turn_context = coordinator._parse_turn_context(
                    {
                        "route": "general",
                        "rationale": "coordinator_route_failure_fallback",
                    }
                )

        if state.workflow or state.resume_context or state.actions:
            state.route = "diagnostic"
        else:
            state.route = str(turn_context.route or state.route).strip() or "diagnostic"

        action_intent = (
            turn_context.action_intent
            or str(state.workflow.get("action_intent") or "").strip().lower()
            or str(state.resume_context.get("action_intent") or "").strip().lower()
            or None
        )
        resolved_job_name = (
            turn_context.selected_job_name
            or str(state.workflow.get("selected_job_name") or "").strip().lower()
            or str(page_context.get("job_name") or "").strip().lower()
            or None
        )
        requested_scope = turn_context.requested_scope
        if requested_scope == "unspecified":
            requested_scope = str(state.workflow.get("requested_scope") or "").strip().lower() or "unspecified"

        if resolved_job_name and not page_context.get("job_name"):
            page_context["job_name"] = resolved_job_name
            state.page_context = page_context
        if not page_context.get("node_name"):
            resolved_node_name = extract_node_name(page_context, goal_message)
            if resolved_node_name:
                page_context["node_name"] = resolved_node_name
                state.page_context = page_context

        state.runtime_scenario = _derive_runtime_scenario(
            route=state.route,
            action_intent=action_intent,
            user_message=goal_message,
            page_context=page_context,
            enabled_tools=enabled_tools,
            workflow=state.workflow,
            resume_context=state.resume_context,
        )
        state.runtime_guard = state.runtime_guard or _build_runtime_guard_for_scenario(state.runtime_scenario)
        coordinator.json_retry_limit = state.runtime_guard.structured_retry_limit
        if state.resume_context:
            state.stop_reason = ""

        if state.route == "guide" and not state.workflow and not state.resume_context:
            guide = make_agent(GuideAgent, "guide-1", "guide")
            yield await emit(
                "agent_handoff",
                {
                    "agentId": coordinator.agent_id,
                    "agentRole": coordinator.role,
                    "targetAgentId": guide.agent_id,
                    "targetAgentRole": guide.role,
                    "summary": "Coordinator 识别为帮助型问题，转交 Guide Agent",
                    "status": "completed",
                },
            )
            try:
                guide_response = await guide.respond(
                    user_message=request.message,
                    page_context=page_context,
                    capabilities=capabilities,
                    actor_role=actor_role,
                )
                record_agent_usage(guide)
                state.final_answer = guide_response.summary
            except Exception:
                logger.exception("Guide agent respond failed")
                state.final_answer = "抱歉，生成帮助说明时出错，请稍后重试。"
            state.stop_reason = "completed"
            yield await emit(
                "agent_status",
                {
                    "agentId": guide.agent_id,
                    "agentRole": guide.role,
                    "status": "completed",
                    "summary": state.final_answer,
                },
            )
            yield await emit_final_answer(
                agent_id=guide.agent_id,
                agent_role=guide.role,
                content=state.final_answer,
            )
            yield {"event": "done", "data": {}}
            return

        if state.route == "general" and not state.workflow and not state.resume_context:
            general = make_agent(GeneralPurposeAgent, "general-1", "general")
            yield await emit(
                "agent_handoff",
                {
                    "agentId": coordinator.agent_id,
                    "agentRole": coordinator.role,
                    "targetAgentId": general.agent_id,
                    "targetAgentRole": general.role,
                    "summary": "Coordinator 识别为常规平台问答，转交 General Agent",
                    "status": "completed",
                },
            )
            try:
                general_response = await general.respond(
                    user_message=request.message,
                    page_context=page_context,
                    capabilities=capabilities,
                    actor_role=actor_role,
                )
                record_agent_usage(general)
                state.final_answer = general_response.summary
            except Exception:
                logger.exception("General agent respond failed")
                state.final_answer = "抱歉，生成回复时出错，请稍后重试。"
            state.stop_reason = "completed"
            yield await emit(
                "agent_status",
                {
                    "agentId": general.agent_id,
                    "agentRole": general.role,
                    "status": "completed",
                    "summary": state.final_answer,
                },
            )
            yield await emit_final_answer(
                agent_id=general.agent_id,
                agent_role=general.role,
                content=state.final_answer,
            )
            yield {"event": "done", "data": {}}
            return

        planner = make_agent(PlannerAgent, "planner-1", "planner")
        explorer = make_agent(ExplorerAgent, "explorer-1", "explorer")
        executor = make_agent(ExecutorAgent, "executor-1", "executor")
        verifier = make_agent(VerifierAgent, "verifier-1", "verifier")

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

        if state.plan is None:
            fast_track_write = bool(
                action_intent
                or state.resume_context
                or state.actions
                or state.runtime_scenario in {"action", "submission"}
            )
            if fast_track_write:
                if state.runtime_scenario == "submission":
                    synthetic_steps = ["补一轮必要的资源/镜像证据", "产出创建草案", "进入确认表单"]
                    synthetic_tools = ["recommend_training_images", "list_available_gpu_models"]
                else:
                    synthetic_steps = ["最小只读核验目标对象", "进入写操作执行", "必要时等待确认后继续"]
                    synthetic_tools = ["get_job_detail"] if resolved_job_name else ["list_user_jobs"]
                state.plan = MultiAgentRoleOutput(
                    agent_id=planner.agent_id,
                    agent_role=planner.role,
                    summary=(
                        "已识别为创建/提交场景，采用轻量探索后直接进入确认流。"
                        if state.runtime_scenario == "submission"
                        else "已识别为明确写操作请求，采用最小核验后直接推进执行。"
                    ),
                    status="completed",
                    metadata={
                        "plan_output": {
                            "goal": (
                                "创建/提交任务的快速确认路径"
                                if state.runtime_scenario == "submission"
                                else "明确写操作请求的快速执行路径"
                            ),
                            "steps": synthetic_steps,
                            "candidate_tools": synthetic_tools,
                            "risk": "medium",
                            "raw_summary": (
                                "已识别为创建/提交场景，采用轻量探索后直接进入确认流。"
                                if state.runtime_scenario == "submission"
                                else "已识别为明确写操作请求，采用最小核验后直接推进执行。"
                            ),
                        }
                    },
                )
                yield await emit(
                    "agent_status",
                    {
                        "agentId": planner.agent_id,
                        "agentRole": planner.role,
                        "status": state.plan.status,
                        "summary": state.plan.summary,
                    },
                )
            else:
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
                    plan = await planner.plan(
                        user_message=goal_message,
                        page_context=page_context,
                        capabilities=capabilities,
                        actor_role=actor_role,
                        evidence_summary=_build_evidence_summary_fallback(_compact_evidence_for_prompt(state.evidence)),
                        action_history_summary=_build_action_history_summary(state.action_history),
                        continuation=continuation,
                    )
                    record_agent_usage(planner)
                except Exception:
                    logger.exception("Planner failed")
                    plan = RoleExecutionResult(
                        summary="规划失败，将基于直接证据收集继续推进",
                        metadata={"plan_output": {}},
                    )
                state.plan = MultiAgentRoleOutput(
                    agent_id=planner.agent_id,
                    agent_role=planner.role,
                    summary=plan.summary,
                    status=plan.status,
                    metadata=plan.metadata or {},
                )
                yield await emit(
                    "agent_status",
                    {
                        "agentId": planner.agent_id,
                        "agentRole": planner.role,
                        "status": state.plan.status,
                        "summary": state.plan.summary,
                    },
                )

        plan_output = state.plan.metadata.get("plan_output", {}) if state.plan and state.plan.metadata else {}
        plan_candidate_tools = plan_output.get("candidate_tools", [])
        plan_steps = plan_output.get("steps", [])

        no_progress_rounds = 0

        while True:
            if state.runtime_guard and state.loop_iteration >= state.runtime_guard.max_loop_iterations:
                state.stop_reason = "max_loop_iterations_reached"
                break
            stop_reason = _runtime_guard_stop_reason(state, no_progress_rounds=no_progress_rounds)
            if stop_reason:
                state.stop_reason = stop_reason
                break
            state.loop_iteration += 1
            compact_evidence = _compact_evidence_for_prompt(state.evidence)
            evidence_summary_text = (
                state.exploration.summary if state.exploration else _build_evidence_summary_fallback(compact_evidence)
            )
            action_history_summary = _build_action_history_summary(state.action_history)
            pending_actions = _pending_action_dicts(state.actions)
            explore_rounds = _controller_step_count(state.controller_trace, "explore")
            candidate_jobs = _collect_jobs_from_evidence(
                state.evidence,
                status_filter=_candidate_status_filter_for_action(action_intent),
            )
            guard = state.runtime_guard or _build_runtime_guard_for_scenario(state.runtime_scenario)
            if (
                state.runtime_scenario in {"query", "ops", "node_analysis", "ops_report"}
                and state.usage_summary.read_tool_calls >= guard.max_read_tool_calls
                and state.evidence
            ):
                state.stop_reason = "max_read_budget_reached"
                break

            if state.action_frontier():
                decision = LoopDecision(step="execute", rationale="存在已准备好的动作前沿")
            elif state.resume_context and not any(action.status == "pending" for action in state.actions):
                decision = LoopDecision(step="finalize", rationale="确认结果已回流且没有剩余动作")
            elif state.resume_context and state.actions:
                decision = LoopDecision(step="execute", rationale="确认结果已回流，继续执行剩余动作")
            elif state.runtime_scenario == "submission" and not state.evidence:
                decision = LoopDecision(step="explore", rationale="提交场景先补一轮必要的资源和镜像信息")
            elif state.runtime_scenario == "submission" and not state.action_history:
                decision = LoopDecision(step="execute", rationale="提交场景避免重复查询，直接产出创建草案")
            elif action_intent and resolved_job_name and not state.action_history and not state.evidence:
                decision = LoopDecision(step="execute", rationale="明确写操作且目标已绑定，直接进入确认流")
            elif action_intent and not state.evidence:
                decision = LoopDecision(step="explore", rationale="明确写操作但尚未完成最小核验")
            elif action_intent and _has_minimal_target_evidence(
                evidence=state.evidence,
                resolved_job_name=resolved_job_name,
                requested_scope=requested_scope,
                candidate_jobs=candidate_jobs,
            ) and not state.action_history:
                decision = LoopDecision(step="execute", rationale="明确写操作且目标已完成最小核验")
            elif not action_intent and not state.evidence:
                decision = LoopDecision(step="explore", rationale="纯查询场景先执行一轮只读探索")
            elif (
                not action_intent
                and state.verification is not None
                and state.verification.status in {"pass", "risk"}
            ):
                decision = LoopDecision(step="finalize", rationale="纯查询场景已经完成验证")
            elif (
                not action_intent
                and state.verification is not None
                and state.verification.status == "missing_evidence"
            ):
                if explore_rounds < guard.max_read_only_explore_rounds:
                    decision = LoopDecision(step="explore", rationale="验证认为证据不足，补充一轮只读探索")
                else:
                    decision = LoopDecision(step="finalize", rationale="纯查询场景达到探索上限，基于现有证据收束")
            elif (
                not action_intent
                and state.exploration is not None
                and state.verification is None
                and guard.max_verifications > 0
            ):
                decision = LoopDecision(step="verify", rationale="纯查询场景完成探索后先验证再收束")
            else:
                try:
                    decision = await coordinator.decide_next_step(
                        user_message=request.message,
                        page_context=page_context,
                        plan_summary=state.plan.summary if state.plan else "",
                        evidence_summary=evidence_summary_text,
                        action_history_summary=action_history_summary,
                        pending_actions=pending_actions,
                        continuation=continuation,
                        loop_iteration=state.loop_iteration,
                        replan_count=state.replan_count,
                        verification_summary=state.verification.summary if state.verification else "",
                    )
                    record_agent_usage(coordinator)
                except Exception:
                    logger.exception("Coordinator loop decision failed")
                    decision = _fallback_loop_decision(state)

            state.remember_controller_decision(
                {
                    "iteration": state.loop_iteration,
                    "step": decision.step,
                    "rationale": decision.rationale,
                }
            )

            if decision.step == "replan":
                if state.replan_count >= guard.max_replans:
                    decision = LoopDecision(step="finalize", rationale="达到最大重规划次数")
                else:
                    yield await emit(
                        "agent_handoff",
                        {
                            "agentId": coordinator.agent_id,
                            "agentRole": coordinator.role,
                            "targetAgentId": planner.agent_id,
                            "targetAgentRole": planner.role,
                            "summary": "Coordinator 判定当前计划需要重规划",
                            "status": "completed",
                        },
                    )
                    state.replan_count += 1
                    try:
                        replan = await planner.plan(
                            user_message=goal_message,
                            page_context=page_context,
                            capabilities=capabilities,
                            actor_role=actor_role,
                            evidence_summary=evidence_summary_text,
                            action_history_summary=action_history_summary,
                            continuation=continuation,
                            replan_reason=decision.rationale,
                        )
                        record_agent_usage(planner)
                    except Exception:
                        logger.exception("Planner replan failed")
                        replan = RoleExecutionResult(
                            summary="重规划失败，沿用现有计划继续推进",
                            metadata={"plan_output": plan_output},
                        )
                    state.plan = MultiAgentRoleOutput(
                        agent_id=planner.agent_id,
                        agent_role=planner.role,
                        summary=replan.summary,
                        status=replan.status,
                        metadata=replan.metadata or {},
                    )
                    plan_output = state.plan.metadata.get("plan_output", {}) if state.plan.metadata else {}
                    plan_candidate_tools = plan_output.get("candidate_tools", [])
                    plan_steps = plan_output.get("steps", [])
                    yield await emit(
                        "agent_status",
                        {
                            "agentId": planner.agent_id,
                            "agentRole": planner.role,
                            "status": state.plan.status,
                            "summary": state.plan.summary,
                        },
                    )
                    no_progress_rounds = 0
                    continue

            if decision.step == "explore":
                yield await emit(
                    "agent_handoff",
                    {
                        "agentId": coordinator.agent_id,
                        "agentRole": coordinator.role,
                        "targetAgentId": explorer.agent_id,
                        "targetAgentRole": explorer.role,
                        "summary": "Coordinator 要求 Explorer 继续收集证据",
                        "status": "completed",
                    },
                )
                selected_tools = []
                if not state.evidence:
                    if state.runtime_scenario == "submission":
                        selected_tools = [
                            (tool_name, tool_args)
                            for tool_name, tool_args in [
                                (
                                    "recommend_training_images",
                                    {
                                        "task_description": "LLM 大语言模型训练",
                                        "framework": "pytorch",
                                        "limit": 3,
                                    },
                                ),
                                ("list_available_gpu_models", {"limit": 10}),
                            ]
                            if tool_name in enabled_tools
                        ]
                    elif state.runtime_scenario == "ops_report":
                        selected_tools = [
                            (tool_name, tool_args)
                            for tool_name, tool_args in [
                                (
                                    "get_admin_ops_report",
                                    {
                                        "days": 7,
                                        "success_limit": 5,
                                        "failure_limit": 5,
                                        "gpu_threshold": 5,
                                        "idle_hours": 24,
                                    },
                                ),
                                ("get_latest_audit_report", {"report_type": "admin_ops_report"}),
                            ]
                            if tool_name in enabled_tools
                        ]
                    elif state.runtime_scenario == "node_analysis":
                        node_name = str(page_context.get("node_name") or "").strip()
                        if node_name and "get_node_detail" in enabled_tools:
                            selected_tools = [("get_node_detail", {"node_name": node_name})]
                    else:
                        selected_tools = _forced_exploration_tools(
                            action_intent=action_intent,
                            resolved_job_name=resolved_job_name,
                            enabled_tools=enabled_tools,
                        )
                if not selected_tools:
                    try:
                        selected_tools = await explorer.select_tools_with_llm(
                            user_message=goal_message,
                            page_context=page_context,
                            plan_candidate_tools=plan_candidate_tools,
                            plan_steps=plan_steps,
                            enabled_tools=enabled_tools,
                            evidence_summary=evidence_summary_text,
                            attempted_tool_signatures=state.attempted_tool_signatures,
                        )
                        record_agent_usage(explorer)
                    except Exception:
                        logger.exception("Explorer select_tools_with_llm failed")
                        selected_tools = []
                if not selected_tools:
                    selected_tools = _fallback_read_tools_from_context(
                        page_context=page_context,
                        enabled_tools=enabled_tools,
                    )

                remaining_read_budget = max(
                    0,
                    guard.max_read_tool_calls - state.usage_summary.read_tool_calls,
                )
                if remaining_read_budget <= 0:
                    state.stop_reason = "max_read_budget_reached"
                    break

                new_evidence_count = 0
                for index, (tool_name, tool_args) in enumerate(selected_tools[:remaining_read_budget], start=1):
                    if tool_name not in READ_ONLY_TOOL_NAMES:
                        continue
                    signature = _tool_signature(tool_name, tool_args)
                    if signature in state.attempted_tool_signatures:
                        continue
                    state.attempted_tool_signatures.append(signature)
                    result, tool_events = await call_tool(
                        role_agent_id=explorer.agent_id,
                        role_name=explorer.role,
                        tool_name=tool_name,
                        tool_args=tool_args,
                        tool_call_id=f"{explorer.agent_id}-tool-{state.loop_iteration}-{index}",
                    )
                    for tool_event in tool_events:
                        yield tool_event
                    state.remember_tool(
                        agent_id=explorer.agent_id,
                        agent_role=explorer.role,
                        tool_name=tool_name,
                        tool_args=tool_args,
                        tool_call_id=f"{explorer.agent_id}-tool-{state.loop_iteration}-{index}",
                        result=result,
                    )
                    new_evidence_count += 1

                candidate_jobs = _collect_jobs_from_evidence(
                    state.evidence,
                    status_filter=_candidate_status_filter_for_action(action_intent),
                )
                if action_intent and not resolved_job_name and requested_scope != "all":
                    if len(candidate_jobs) == 1:
                        resolved_job_name = str(candidate_jobs[0].get("jobName") or "").strip().lower()
                        if resolved_job_name:
                            page_context["job_name"] = resolved_job_name
                            state.page_context = page_context
                    elif len(candidate_jobs) > 1:
                        state.stop_reason = "awaiting_clarification"
                        state.final_answer = _build_action_clarification_answer(
                            action_intent=action_intent,
                            candidate_jobs=candidate_jobs,
                        )
                        yield await emit_final_answer(
                            agent_id=coordinator.agent_id,
                            agent_role=coordinator.role,
                            content=state.final_answer,
                            continuation_payload=_build_job_selection_continuation(
                                action_intent=action_intent,
                                candidate_jobs=candidate_jobs,
                                requested_all_scope=False,
                            ),
                        )
                        yield {"event": "done", "data": {}}
                        return

                if _should_use_structured_action_fast_path(
                    action_intent=action_intent,
                    resolved_job_name=resolved_job_name,
                    requested_scope=requested_scope,
                    candidate_jobs=candidate_jobs,
                    enabled_tools=enabled_tools,
                ):
                    evidence_summary = RoleExecutionResult(
                        summary=_build_structured_exploration_summary(
                            action_intent=action_intent,
                            resolved_job_name=resolved_job_name,
                            requested_scope=requested_scope,
                            candidate_jobs=candidate_jobs,
                            compact_evidence=_compact_evidence_for_prompt(state.evidence),
                        ),
                        status="completed",
                    )
                else:
                    try:
                        evidence_summary = await explorer.summarize_evidence(
                            user_message=goal_message,
                            plan_summary=state.plan.summary if state.plan else "",
                            evidence=_compact_evidence_for_prompt(state.evidence),
                        )
                        record_agent_usage(explorer)
                    except Exception:
                        logger.exception("Explorer summarize_evidence failed")
                        evidence_summary = RoleExecutionResult(
                            summary=evidence_summary_text,
                            status="completed",
                        )

                state.exploration = MultiAgentRoleOutput(
                    agent_id=explorer.agent_id,
                    agent_role=explorer.role,
                    summary=evidence_summary.summary,
                    status=evidence_summary.status,
                    metadata=evidence_summary.metadata or {},
                )
                if (
                    new_evidence_count > 0
                    and state.verification is not None
                    and state.verification.status == "missing_evidence"
                ):
                    state.verification = None
                yield await emit(
                    "agent_status",
                    {
                        "agentId": explorer.agent_id,
                        "agentRole": explorer.role,
                        "status": state.exploration.status,
                        "summary": state.exploration.summary,
                    },
                )
                no_progress_rounds = 0 if new_evidence_count > 0 else no_progress_rounds + 1
                continue

            if decision.step == "execute":
                yield await emit(
                    "agent_handoff",
                    {
                        "agentId": coordinator.agent_id,
                        "agentRole": coordinator.role,
                        "targetAgentId": executor.agent_id,
                        "targetAgentRole": executor.role,
                        "summary": "Coordinator 要求 Executor 推进写操作",
                        "status": "completed",
                    },
                )

                frontier = state.action_frontier()
                if not frontier:
                    proposals = []
                    if _should_use_structured_action_fast_path(
                        action_intent=action_intent,
                        resolved_job_name=resolved_job_name,
                        requested_scope=requested_scope,
                        candidate_jobs=candidate_jobs,
                        enabled_tools=enabled_tools,
                    ):
                        proposals = _fallback_executor_actions(
                            action_intent=action_intent,
                            resolved_job_name=resolved_job_name,
                            candidate_jobs=candidate_jobs,
                            requested_scope=requested_scope,
                            enabled_tools=enabled_tools,
                        )

                    if not proposals and state.runtime_scenario == "submission":
                        proposals = _fallback_submission_actions(
                            user_message=goal_message,
                            evidence=state.evidence,
                            enabled_tools=enabled_tools,
                        )

                    if not proposals:
                        try:
                            proposals = await executor.decide_actions_with_llm(
                                user_message=goal_message,
                                page_context=page_context,
                                plan_summary=state.plan.summary if state.plan else "",
                                evidence_summary=evidence_summary_text,
                                compact_evidence=compact_evidence,
                                action_intent=action_intent,
                                selected_job_name=resolved_job_name,
                                requested_scope=requested_scope,
                                action_history=state.action_history,
                                pending_actions=pending_actions,
                                enabled_tools=enabled_tools,
                            )
                            record_agent_usage(executor)
                        except Exception:
                            logger.exception("Executor decide_actions_with_llm failed")
                            proposals = []

                    if not proposals:
                        if state.runtime_scenario == "submission":
                            proposals = _fallback_submission_actions(
                                user_message=goal_message,
                                evidence=state.evidence,
                                enabled_tools=enabled_tools,
                            )
                        else:
                            proposals = _fallback_executor_actions(
                                action_intent=action_intent,
                                resolved_job_name=resolved_job_name,
                                candidate_jobs=candidate_jobs,
                                requested_scope=requested_scope,
                                enabled_tools=enabled_tools,
                            )

                    _merge_action_proposals(state.actions, proposals)
                    frontier = state.action_frontier()

                if not frontier:
                    no_progress_rounds += 1
                    if no_progress_rounds >= guard.max_no_progress_rounds:
                        state.stop_reason = "max_no_progress_reached"
                        break
                    continue

                executed_actions: list[dict[str, Any]] = []
                for action in frontier[: guard.max_frontier_actions]:
                    signature = _tool_signature(action.tool_name, action.tool_args)
                    if signature in state.attempted_tool_signatures:
                        action.status = "skipped"
                        continue
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
                        state.pending_confirmation = result
                        state.stop_reason = "awaiting_confirmation"
                        checkpoint = state.build_workflow_checkpoint()
                        checkpoint["route"] = state.route
                        checkpoint["action_intent"] = action_intent
                        checkpoint["selected_job_name"] = resolved_job_name
                        checkpoint["requested_scope"] = requested_scope
                        checkpoint["evidence"] = _compact_evidence_for_prompt(state.evidence)
                        checkpoint["pause_reason"] = "awaiting_confirmation"
                        checkpoint["current_action_id"] = action.action_id
                        yield await emit_checkpoint(
                            summary="已保存当前工作流状态，等待用户确认后继续执行",
                            workflow=checkpoint,
                        )
                        yield await emit(
                            "agent_status",
                            {
                                "agentId": executor.agent_id,
                                "agentRole": executor.role,
                                "status": "awaiting_confirmation",
                                "summary": "Executor 已发起高风险操作，等待用户确认",
                            },
                        )
                        yield {"event": "done", "data": {}}
                        return

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

                if executed_actions:
                    try:
                        execution_summary = await executor.summarize_action(
                            user_message=goal_message,
                            plan_summary=state.plan.summary if state.plan else "",
                            action_result={"actions": executed_actions},
                        )
                        record_agent_usage(executor)
                    except Exception:
                        logger.exception("Executor summarize_action failed")
                        execution_summary = RoleExecutionResult(
                            summary=_build_action_history_summary(state.action_history),
                            status="completed",
                        )

                    state.execution = MultiAgentRoleOutput(
                        agent_id=executor.agent_id,
                        agent_role=executor.role,
                        summary=execution_summary.summary,
                        status=execution_summary.status,
                        metadata=execution_summary.metadata or {},
                    )
                    yield await emit(
                        "agent_status",
                        {
                            "agentId": executor.agent_id,
                            "agentRole": executor.role,
                            "status": state.execution.status,
                            "summary": state.execution.summary,
                        },
                    )
                    no_progress_rounds = 0
                else:
                    no_progress_rounds += 1
                continue

            if decision.step == "verify":
                if guard.max_verifications <= 0 or state.usage_summary.verification_calls >= guard.max_verifications:
                    state.stop_reason = "verification_budget_reached"
                    break
                yield await emit(
                    "agent_handoff",
                    {
                        "agentId": coordinator.agent_id,
                        "agentRole": coordinator.role,
                        "targetAgentId": verifier.agent_id,
                        "targetAgentRole": verifier.role,
                        "summary": "Coordinator 要求 Verifier 对当前结论做挑战式验证",
                        "status": "completed",
                    },
                )
                try:
                    verification = await verifier.verify(
                        user_message=goal_message,
                        evidence_summary=evidence_summary_text,
                        executor_summary=state.execution.summary if state.execution else action_history_summary,
                    )
                    record_agent_usage(verifier)
                except Exception:
                    logger.exception("Verifier failed")
                    verification = RoleExecutionResult(
                        summary="验证跳过（内部错误）",
                        status="missing_evidence",
                        metadata={"verification_result": "missing_evidence"},
                    )
                state.verification = MultiAgentRoleOutput(
                    agent_id=verifier.agent_id,
                    agent_role=verifier.role,
                    summary=verification.summary,
                    status=verification.status,
                    metadata=verification.metadata or {},
                )
                yield await emit(
                    "agent_status",
                    {
                        "agentId": verifier.agent_id,
                        "agentRole": verifier.role,
                        "status": state.verification.status,
                        "summary": state.verification.summary,
                        "verificationResult": state.verification.metadata.get("verification_result"),
                    },
                )
                state.usage_summary.verification_calls += 1
                no_progress_rounds = 0
                continue

            break

        terminal_action_answer = None
        if state.resume_context:
            terminal_action_answer = _build_terminal_action_answer(state)

        if terminal_action_answer:
            state.final_answer = terminal_action_answer
        elif (
            state.verification is None
            and (state.action_history or state.exploration)
            and not _should_skip_optional_llm_calls(state)
            and (state.runtime_guard.max_verifications if state.runtime_guard else 1) > 0
            and state.usage_summary.verification_calls < (state.runtime_guard.max_verifications if state.runtime_guard else 1)
        ):
            try:
                verification = await verifier.verify(
                    user_message=goal_message,
                    evidence_summary=(
                        state.exploration.summary
                        if state.exploration
                        else _build_evidence_summary_fallback(_compact_evidence_for_prompt(state.evidence))
                    ),
                    executor_summary=state.execution.summary if state.execution else _build_action_history_summary(state.action_history),
                )
                record_agent_usage(verifier)
                state.verification = MultiAgentRoleOutput(
                    agent_id=verifier.agent_id,
                    agent_role=verifier.role,
                    summary=verification.summary,
                    status=verification.status,
                    metadata=verification.metadata or {},
                )
                state.usage_summary.verification_calls += 1
            except Exception:
                logger.exception("Verifier final pass failed")

        if not state.final_answer:
            if _should_skip_optional_llm_calls(state):
                state.final_answer = _build_runtime_fallback_final_answer(state)
            else:
                try:
                    final = await coordinator.summarize(
                        user_message=goal_message,
                        plan_summary=state.plan.summary if state.plan else "无规划摘要",
                        evidence_summary=(
                            state.exploration.summary
                            if state.exploration
                            else _build_evidence_summary_fallback(_compact_evidence_for_prompt(state.evidence))
                        ),
                        executor_summary=state.execution.summary if state.execution else _build_action_history_summary(state.action_history),
                        verifier_summary=(
                            f"{state.verification.status}: {state.verification.summary}"
                            if state.verification
                            else "未执行显式验证"
                        ),
                    )
                    record_agent_usage(coordinator)
                    state.final_answer = final.summary
                except Exception:
                    logger.exception("Coordinator summarize failed")
                    state.final_answer = (
                        state.execution.summary
                        if state.execution
                        else state.exploration.summary
                        if state.exploration
                        else "Agent 执行完成，但生成最终答复时出错。"
                    )

        if not state.stop_reason:
            state.stop_reason = "completed"

        yield await emit_final_answer(
            agent_id=coordinator.agent_id,
            agent_role=coordinator.role,
            content=state.final_answer,
        )
        yield {"event": "done", "data": {}}
