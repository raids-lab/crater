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
from dataclasses import dataclass
from datetime import datetime
from typing import Any, AsyncIterator, Awaitable, Callable

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage, ToolMessage

from crater_agent.agents.base import BaseRoleAgent, RoleExecutionResult
from crater_agent.agents.executor import ExecutorAgent
from crater_agent.agents.explorer import ExplorerAgent
from crater_agent.agents.planner import PlannerAgent
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
from crater_agent.orchestrators.intent_router import IntentRouter
from crater_agent.orchestrators.state import (
    MASRuntimeConfig,
    MASState,
    MultiAgentActionItem,
)
from crater_agent.report_utils import build_pipeline_report_payload
from crater_agent.scenarios import (
    NODE_ANALYSIS_ENTRYPOINT,
    extract_node_name,
    infer_entrypoint,
)
from crater_agent.tools.definitions import (
    ALL_TOOLS,
    CONFIRM_TOOL_NAMES,
    READ_ONLY_TOOL_NAMES,
    is_tool_allowed_for_role,
)
from crater_agent.tools.executor import GoBackendToolExecutor, ToolExecutorProtocol

logger = logging.getLogger(__name__)

CREATE_JOB_TOOL_NAMES = {"create_training_job", "create_jupyter_job"}


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
    if not input_tokens:
        input_tokens = _estimate_tokens_from_messages(messages)
    if not output_tokens:
        output_tokens = role_agent._estimate_tokens(output_text)
    return {
        "llm_calls": 1,
        "input_tokens": int(input_tokens),
        "output_tokens": int(output_tokens),
    }


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
    job_type = str(job.get("jobType") or "").strip().lower()
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
        key=lambda job: _parse_creation_timestamp(job.get("creationTimestamp")),
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
                    "jobName": str(job.get("jobName") or "").strip(),
                    "name": str(job.get("name") or "").strip(),
                    "status": status,
                    "jobType": str(job.get("jobType") or "").strip(),
                    "creationTimestamp": str(job.get("creationTimestamp") or "").strip(),
                }
            )
    return _dedupe_jobs(jobs)


def _format_job_candidates(jobs: list[dict[str, Any]], limit: int = 8) -> str:
    lines: list[str] = []
    for index, job in enumerate(jobs[:limit], start=1):
        display_name = str(job.get("name") or "").strip()
        display_suffix = f" / {display_name}" if display_name else ""
        status = str(job.get("status") or "").strip() or "unknown"
        lines.append(f"{index}. {job.get('jobName')}{display_suffix} ({status})")
    return "\n".join(lines)


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

    entrypoint = infer_entrypoint(page_context, user_message)

    # Node analysis entrypoint
    if entrypoint == NODE_ANALYSIS_ENTRYPOINT:
        node_name = extract_node_name(page_context, user_message)
        if node_name and "get_node_detail" in enabled:
            preferred.append(("get_node_detail", {"node_name": node_name}))
        return preferred

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
    if routing.entry_mode == "help":
        return "guide"
    if routing.operation_mode == "write":
        return "action"
    return "query"


# ---------------------------------------------------------------------------
# MultiAgentOrchestrator
# ---------------------------------------------------------------------------

class MultiAgentOrchestrator:
    def __init__(self, tool_executor: ToolExecutorProtocol | None = None):
        self.tool_executor = tool_executor or GoBackendToolExecutor()

    @staticmethod
    def _determine_next_stage_fast(
        state: MASState,
        routing: RoutingDecision,
        resumed_action: dict[str, Any] | None,
    ) -> str | None:
        """Deterministic fast-paths that don't need Coordinator LLM.

        Returns a stage string if a fast path matches, None if Coordinator should decide.
        """
        # Confirmation resume
        if resumed_action and not any(a.status == "pending" for a in state.actions):
            return "finalize"
        if resumed_action and state.actions:
            return "act"

        # Awaiting confirmation → can't proceed
        if state.pending_confirmation:
            return "finalize"

        # Simple known-target write with no prior work
        if (
            routing.operation_mode == "write"
            and routing.requested_action
            and routing.requested_action in {"resubmit", "stop", "delete"}
            and (routing.targets.job_name or routing.targets.scope == "all")
            and not state.observation
            and not state.action_history
        ):
            return "act"

        # First round, no plan yet → always plan first
        if not state.plan and state.loop_round <= 1:
            return "plan"

        # Everything else → Coordinator decides
        return None

    async def stream(
        self,
        *,
        request: Any,
        model_factory: ModelClientFactory,
    ) -> AsyncIterator[dict]:
        state = MASState.from_request(request)
        page_context = dict(state.goal.page_context)
        capabilities = state.capabilities
        enabled_tools = state.enabled_tools
        goal_message = state.goal.original_user_message

        def make_agent(cls: type, agent_id: str, role: str) -> Any:
            return cls(
                agent_id=agent_id,
                role=role,
                llm=model_factory.create(role),
            )

        coordinator = make_agent(BaseRoleAgent, "coordinator-1", "coordinator")
        planner = make_agent(PlannerAgent, "planner-1", "planner")
        explorer = make_agent(ExplorerAgent, "explorer-1", "explorer")
        executor = make_agent(ExecutorAgent, "executor-1", "executor")

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
            aggregate_usage: dict[str, int] = {"llm_calls": 0, "input_tokens": 0, "output_tokens": 0}
            invoked_tool_calls = 0
            stalled_tool_rounds = 0

            for loop_index in range(max(1, max_tool_calls + 1)):
                response = await llm_with_tools.ainvoke(messages)
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
                    return ToolLoopOutcome(
                        summary=selected or role_agent.latest_reasoning_summary(),
                        tool_calls=invoked_tool_calls,
                    ), collected_events

                if invoked_tool_calls >= max_tool_calls:
                    role_agent.last_usage = aggregate_usage
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
                    if stalled_tool_rounds >= 2:
                        role_agent.last_usage = aggregate_usage
                        summary = selected or role_agent.latest_reasoning_summary()
                        if not summary:
                            summary = "工具调用连续重复且没有产生新结果，已停止继续调用。"
                        return ToolLoopOutcome(
                            summary=summary,
                            tool_calls=invoked_tool_calls,
                        ), collected_events

            role_agent.last_usage = aggregate_usage
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
                "summary": (
                    "确认结果已回流，继续执行上一轮计划"
                    if state.resume_context
                    else "多 Agent 协作已启动"
                ),
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
            if not routing.requested_action:
                routing.requested_action = (
                    str(state.resume_context.get("action_intent") or "").strip().lower() or None
                )
                if routing.requested_action:
                    routing.operation_mode = "write"
        else:
            intent_router = IntentRouter(coordinator_agent=coordinator)
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
                record_agent_usage(coordinator)
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
        if state.resume_context:
            state.stop_reason = ""

        # =================================================================
        # STEP 4: Help fast path
        # =================================================================
        if routing.entry_mode == "help" and not state.workflow and not state.resume_context:
            try:
                answer = await coordinator.run_text(
                    system_prompt=(
                        "你是 Crater 的智能运维助手。请用中文回答用户的帮助问题。\n"
                        "回答应该清晰、简洁、有帮助性。\n"
                        "如果涉及具体操作步骤，请给出明确指引。\n"
                        "如果不确定某些细节，请如实说明。"
                    ),
                    user_prompt=state.build_state_view("coordinator").for_prompt(),
                    history_messages=history_messages,
                )
                record_agent_usage(coordinator)
                state.final_answer = answer
            except Exception:
                logger.exception("Coordinator help response failed")
                state.final_answer = "抱歉，生成帮助说明时出错，请稍后重试。"
            state.stop_reason = "completed"
            yield await emit(
                "agent_status",
                {
                    "agentId": coordinator.agent_id,
                    "agentRole": coordinator.role,
                    "status": "completed",
                    "summary": state.final_answer,
                },
            )
            yield await emit_final_answer(
                agent_id=coordinator.agent_id,
                agent_role=coordinator.role,
                content=state.final_answer,
            )
            yield {"event": "done", "data": {}}
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

            # If no fast-path matched, ask Coordinator LLM to decide
            if next_stage is None:
                state_view = state.build_state_view("coordinator")
                try:
                    coordinator_decision = await coordinator.run_json(
                        system_prompt=(
                            "你是 Crater MAS 的 Coordinator 协调者。你根据当前状态决定下一步。\n\n"
                            "可选动作：\n"
                            '- "observe": 需要收集更多信息（调用只读工具）\n'
                            '- "act": 需要执行操作（调用写工具，或 executor 先读后写）\n'
                            '- "replan": 当前计划与实际收集的证据不匹配，需要 Planner 重新规划\n'
                            '- "finalize": 信息已足够，可以回答用户了\n\n'
                            "决策原则：\n"
                            "- 有计划就按计划推进，不要无故偏离\n"
                            "- 如果已收集的证据足以回答用户，选 finalize\n"
                            "- 如果证据和用户请求明显不匹配、计划走偏了，选 replan\n"
                            "- 如果需要执行写操作且目标明确，选 act\n"
                            "- 如果还缺关键信息，选 observe\n"
                            "- 不要反复 observe 相同内容\n\n"
                            '输出 JSON: {"next": "observe|act|replan|finalize", "reason": "简短理由"}\n'
                        ),
                        user_prompt=state_view.for_prompt(),
                        history_messages=history_messages,
                    )
                    record_agent_usage(coordinator)
                    next_stage = str(coordinator_decision.get("next") or "finalize").strip().lower()
                    reason = str(coordinator_decision.get("reason") or "").strip()
                    if next_stage not in {"observe", "act", "replan", "finalize", "plan"}:
                        next_stage = "finalize"
                    logger.info(
                        "Coordinator decision round=%d: %s (%s)",
                        state.loop_round, next_stage, reason,
                    )
                except Exception:
                    logger.exception("Coordinator decision failed, falling back to finalize")
                    next_stage = "finalize"

            # After the first iteration, clear resumed_action
            if state.loop_round > 1:
                resumed_action = None

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
                yield await emit(
                    "agent_status",
                    {
                        "agentId": planner.agent_id,
                        "agentRole": planner.role,
                        "status": "completed",
                        "summary": state.plan.summary,
                    },
                )
                continue

            # ----- OBSERVE stage -----
            if next_stage == "observe":
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
                    loop_system_prompt, loop_user_prompt = explorer.build_tool_loop_prompts(
                        user_message=goal_message,
                        page_context=page_context,
                        plan_candidate_tools=prompt_candidate_tools,
                        plan_steps=state.plan.steps if state.plan else [],
                        enabled_tools=enabled_tools,
                        evidence_summary=evidence_summary_text,
                        attempted_tool_signatures=state.attempted_tool_signatures,
                    )

                    async def on_explorer_tool_result(
                        tool_name: str,
                        tool_args: dict[str, Any],
                        tool_call_id: str,
                        result: dict[str, Any],
                    ) -> ToolLoopStopSignal | None:
                        state.remember_tool(
                            agent_id=explorer.agent_id,
                            agent_role=explorer.role,
                            tool_name=tool_name,
                            tool_args=tool_args,
                            tool_call_id=tool_call_id,
                            result=result,
                        )
                        return None

                    loop_outcome, loop_events = await run_role_tool_loop(
                        role_agent=explorer,
                        role_name=explorer.role,
                        system_prompt=loop_system_prompt,
                        user_prompt=loop_user_prompt,
                        allowed_tool_names=[t for t in enabled_tools if t in READ_ONLY_TOOL_NAMES],
                        max_tool_calls=state.runtime_config.subagent_max_iterations,
                        on_tool_result=on_explorer_tool_result,
                        loop_history_messages=history_messages,
                    )
                    record_agent_usage(explorer)
                    for tool_event in loop_events:
                        yield tool_event
                    exploration_summary = loop_outcome.summary
                except Exception:
                    logger.exception("Explorer native tool loop failed")

                new_evidence = len(state.tool_records) - evidence_before

                # If tool loop produced nothing, try select_tools_with_llm fallback
                if new_evidence <= 0:
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
                            role_agent_id=explorer.agent_id,
                            role_name=explorer.role,
                            tool_name=tool_name,
                            tool_args=tool_args,
                            tool_call_id=f"{explorer.agent_id}-tool-{state.loop_round}-{index}",
                        )
                        for tool_event in tool_events:
                            yield tool_event
                        state.remember_tool(
                            agent_id=explorer.agent_id,
                            agent_role=explorer.role,
                            tool_name=tool_name,
                            tool_args=tool_args,
                            tool_call_id=f"{explorer.agent_id}-tool-{state.loop_round}-{index}",
                            result=result,
                        )
                        new_evidence += 1

                # Handle single-target resolution for action intents
                evidence_dicts = _extract_evidence_from_tool_records(state.tool_records)
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
                            continuation_payload=_build_job_selection_continuation(
                                action_intent=routing.requested_action,
                                candidate_jobs=candidate_jobs,
                                requested_all_scope=False,
                            ),
                        )
                        yield {"event": "done", "data": {}}
                        return

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
                        "agentId": explorer.agent_id,
                        "agentRole": explorer.role,
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

                frontier = state.action_frontier()

                if not frontier:
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
                            enabled_tools=enabled_tools,
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
                                state.pending_confirmation = result
                                return ToolLoopStopSignal(
                                    should_stop=True,
                                    summary="Executor 已发起高风险操作，等待用户确认。",
                                )

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
                            allowed_tool_names=enabled_tools,
                            max_tool_calls=max(1, state.runtime_config.max_actions_per_round + 2),
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
                        state.stop_reason = "awaiting_confirmation"
                        checkpoint = state.build_workflow_checkpoint()
                        checkpoint["pause_reason"] = "awaiting_confirmation"
                        checkpoint["current_action_id"] = awaiting_action.action_id
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

                    # If native tool loop produced results, record execution
                    if native_tool_calls > 0:
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
                            enabled_tools=enabled_tools,
                        )

                    if not proposals and routing.requested_action == "create":
                        proposals = _fallback_submission_actions(
                            user_message=goal_message,
                            evidence=evidence_dicts,
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
                                action_intent=routing.requested_action,
                                selected_job_name=routing.targets.job_name,
                                requested_scope=routing.targets.scope,
                                action_history=state.action_history,
                                pending_actions=pending_actions,
                                enabled_tools=enabled_tools,
                                history_messages=history_messages,
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
                                enabled_tools=enabled_tools,
                            )
                        else:
                            proposals = _fallback_executor_actions(
                                action_intent=routing.requested_action,
                                resolved_job_name=routing.targets.job_name,
                                candidate_jobs=candidate_jobs,
                                requested_scope=routing.targets.scope,
                                enabled_tools=enabled_tools,
                            )

                    _merge_action_proposals(state.actions, proposals)
                    frontier = state.action_frontier()

                # Execute frontier actions
                if not frontier:
                    state.no_progress_count += 1
                    continue

                executed_actions: list[dict[str, Any]] = []
                for action in frontier[:state.runtime_config.max_actions_per_round]:
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
                            "summary": f"Replan: {state.plan.summary}",
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
                        "summary": f"Replan: {state.plan.summary} (剩余 {len(new_steps)} 步)",
                    },
                )
                continue

        # =================================================================
        # STEP 6: Finalize
        # =================================================================
        if not state.final_answer:
            if state.stop_reason in {"max_rounds", "no_progress"}:
                state.final_answer = _build_runtime_fallback_final_answer(state)
            else:
                # Coordinator LLM synthesizes final answer from all artifacts
                state_view = state.build_state_view("coordinator")
                user_prompt = state_view.for_prompt()
                terminal_answer = _build_terminal_action_answer(state)
                if terminal_answer:
                    user_prompt += f"\n\n动作执行摘要:\n{terminal_answer}"
                try:
                    state.final_answer = await coordinator.run_text(
                        system_prompt=(
                            "你是 Crater 的智能运维助手。请基于已收集的证据和执行结果，向用户给出最终答复。\n"
                            "回答要求：\n"
                            "- 使用中文\n"
                            "- 先给结论，再给关键细节\n"
                            "- 如果涉及数据/指标，要引用具体数值\n"
                            "- 如果有建议动作，要明确说明\n"
                            "- 不要重复罗列原始 JSON 字段，要做人类可读的总结\n"
                            "- 严禁虚构任何信息：不要编造不存在的作业名、错误码、指标数值或平台功能\n"
                            "- 只能使用工具调用原始结果中实际存在的数据；如果某些作业尚未被诊断到，如实说明而不是编造结果\n"
                            "- 如果证据不完整（例如只诊断了部分作业），明确告知用户哪些已分析、哪些尚未覆盖\n"
                            "- 如果近期对话里出现过错误结论，本轮必须直接纠正；不要重复沿用旧答案中的作业名或示例"
                        ),
                        user_prompt=user_prompt,
                        history_messages=history_messages,
                    )
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
        )
        yield {"event": "done", "data": {}}
