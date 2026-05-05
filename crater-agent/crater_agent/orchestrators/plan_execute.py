"""Plan-and-execute baseline orchestrator for offline evaluation.

This orchestrator intentionally stays simpler than the full multi-agent MOPS:
  - Planner generates a structured plan.
  - Executor runs tools in a ReAct-style loop to carry out the plan.
  - Planner may replan when the executor makes no progress.

It reuses the existing PlannerAgent / ExecutorAgent / MASState helpers so
offline benchmark scenarios can exercise confirm/resume and permission-aware
tool execution without pulling in Coordinator / Explorer / Verifier.
"""

from __future__ import annotations

import json
import logging
import time
import uuid
from dataclasses import dataclass
from typing import Any, AsyncIterator

import httpcore
import httpx
from langchain_core.messages import HumanMessage, SystemMessage, ToolMessage
from openai import APIConnectionError, APITimeoutError, InternalServerError, RateLimitError
from tenacity import retry, retry_if_exception_type, stop_after_attempt, wait_exponential

from crater_agent.agents.base import BaseRoleAgent, RoleExecutionResult
from crater_agent.agents.executor import ExecutorAgent
from crater_agent.agents.planner import PlannerAgent
from crater_agent.llm.client import ModelClientFactory
from crater_agent.memory.session import build_history_messages
from crater_agent.orchestrators.artifacts import ExecutionArtifact, ObservationArtifact, PlanArtifact
from crater_agent.orchestrators.state import (
    MASState,
    MultiAgentActionItem,
    _tool_signature,
)
from crater_agent.report_utils import build_pipeline_report_payload
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


def _extract_usage_from_tool_loop_response(
    response: object,
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


def _build_plan_summary(plan: PlanArtifact | None, *, prefix: str = "已生成计划") -> str:
    if plan is None:
        return f"{prefix}：无。"
    if not plan.steps:
        return f"{prefix}：{plan.summary or '无显式步骤。'}"
    tool_preview = ", ".join(plan.candidate_tools[:4]) if plan.candidate_tools else "无"
    first_steps = "；".join(plan.steps[:2])
    return f"{prefix}：本次规划 {len(plan.steps)} 步。前两步: {first_steps}。候选工具: {tool_preview}。"


def _current_plan_focus(plan: PlanArtifact | None) -> tuple[str, list[str]]:
    if plan is None or not plan.steps:
        return "", []
    current_step = str(plan.steps[0] or "").strip()
    remaining_steps = [str(step).strip() for step in plan.steps[1:] if str(step).strip()]
    return current_step, remaining_steps


def _compact_evidence_for_prompt(state: MASState, *, limit: int = 8) -> list[dict[str, Any]]:
    compact: list[dict[str, Any]] = []
    for record in state.tool_records[-limit:]:
        compact.append(
            {
                "tool_name": record.tool_name,
                "tool_args": dict(record.tool_args),
                "result": record.result or {},
            }
        )
    return compact


def _build_evidence_summary(state: MASState) -> str:
    compact = _compact_evidence_for_prompt(state)
    if not compact:
        return ""
    lines: list[str] = []
    for item in compact:
        tool_name = str(item.get("tool_name") or "").strip()
        result = item.get("result") or {}
        payload = result.get("result", result.get("message", "")) if isinstance(result, dict) else result
        lines.append(f"- {tool_name}: {_truncate_text(payload, max_chars=220)}")
    return "\n".join(lines)


def _record_agent_usage(state: MASState, agent: Any) -> None:
    usage = dict(getattr(agent, "last_usage", None) or {})
    usage["latency_ms"] = int(getattr(agent, "last_latency_ms", 0) or 0)
    state.record_llm_usage(usage, role=str(getattr(agent, "role", "") or ""))


def _make_history_messages(state: MASState) -> list[Any]:
    if not state.history:
        return []
    return build_history_messages(
        history=state.history,
        max_tokens=6000,
        tool_result_max_chars=160,
    )


@dataclass
class ToolLoopOutcome:
    summary: str
    tool_calls: int
    made_progress: bool = False


class PlanExecuteOrchestrator:
    def __init__(self, tool_executor: ToolExecutorProtocol | None = None):
        self.tool_executor = tool_executor or CompositeToolExecutor()

    async def stream(self, *, request: Any, model_factory: ModelClientFactory) -> AsyncIterator[dict]:
        state = MASState.from_request(request)
        goal_message = (
            state.goal.original_user_message
            if state.resume_context
            else state.goal.user_message
        )
        page_context = dict(state.goal.page_context)
        capabilities = state.capabilities
        enabled_tools = state.enabled_tools
        history_messages = _make_history_messages(state)

        def make_agent(cls: type, agent_id: str, role: str) -> Any:
            return cls(
                agent_id=agent_id,
                role=role,
                llm=model_factory.create(purpose=role, orchestration_mode="plan_execute"),
            )

        planner = make_agent(PlannerAgent, "ps-planner-1", "planner")
        executor = make_agent(ExecutorAgent, "ps-executor-1", "executor")
        summarizer = make_agent(BaseRoleAgent, "ps-summarizer-1", "plan_execute_summarizer")

        async def emit(event: str, data: dict[str, Any]) -> dict[str, Any]:
            return {"event": event, "data": {"turnId": request.turn_id, **data}}

        async def emit_final_answer(content: str, *, continuation_payload: dict[str, Any] | None = None) -> dict[str, Any]:
            payload: dict[str, Any] = {
                "sessionId": request.session_id,
                "agentId": summarizer.agent_id,
                "agentRole": "plan_execute",
                "content": content,
                "stopReason": state.stop_reason or "completed",
                "usageSummary": state.usage_summary.to_dict(),
            }
            if continuation_payload:
                payload["continuation"] = continuation_payload
            return await emit("final_answer", payload)

        async def emit_checkpoint(summary: str) -> dict[str, Any]:
            checkpoint = state.build_workflow_checkpoint()
            checkpoint["pause_reason"] = "awaiting_confirmation"
            return await emit(
                "agent_checkpoint",
                {
                    "sessionId": request.session_id,
                    "agentId": executor.agent_id,
                    "agentRole": executor.role,
                    "summary": summary,
                    "workflow": checkpoint,
                    "status": "completed",
                },
            )

        async def emit_llm_completed(*, agent_id: str, agent_role: str, usage: dict[str, Any], latency_ms: int) -> dict[str, Any]:
            return await emit(
                "llm_call_completed",
                {
                    "agentId": agent_id,
                    "agentRole": agent_role,
                    "latencyMs": latency_ms,
                    "usage": {
                        "llm_input_tokens": int(usage.get("input_tokens") or 0),
                        "llm_output_tokens": int(usage.get("output_tokens") or 0),
                        "total_tokens": int(usage.get("total_tokens") or 0),
                        "llm_reported_token_calls": int(usage.get("reported_token_calls") or 0),
                        "llm_missing_token_calls": int(usage.get("missing_token_calls") or 0),
                    },
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

            if not is_actor_allowed_for_tool(state.goal.actor_role, tool_name):
                result = {
                    "status": "error",
                    "message": "当前身份无权执行该工具",
                    "error_type": "tool_policy",
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
            started_at = time.perf_counter()
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
            latency_ms = max(1, int((time.perf_counter() - started_at) * 1000))
            if not isinstance(result, dict):
                result = {"status": "error", "message": str(result)}
            if not result.get("_latency_ms"):
                result["_latency_ms"] = latency_ms
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

        async def run_executor_tool_loop() -> tuple[ToolLoopOutcome, list[dict[str, Any]]]:
            allowed_tool_names = list(enabled_tools)
            tool_map = {tool.name: tool for tool in ALL_TOOLS if tool.name in set(allowed_tool_names)}
            if not tool_map:
                return ToolLoopOutcome(summary="", tool_calls=0, made_progress=False), []

            current_step, remaining_steps = _current_plan_focus(state.plan)
            system_prompt, user_prompt = executor.build_tool_loop_prompts(
                user_message=goal_message,
                page_context=page_context,
                plan_summary=(
                    f"{state.plan.summary if state.plan else ''}\n\n"
                    f"当前执行步骤: {current_step or '(empty)'}\n"
                    f"剩余步骤: {remaining_steps or []}\n"
                    "你现在只负责推进当前执行步骤；如果当前步骤已完成或证据已足够回答，直接停止调工具并总结。"
                ).strip(),
                evidence_summary=_build_evidence_summary(state),
                compact_evidence=_compact_evidence_for_prompt(state),
                action_intent=state.goal.routing.requested_action,
                selected_job_name=state.goal.routing.targets.job_name,
                requested_scope=state.goal.routing.targets.scope,
                action_history=state.action_history,
                pending_actions=[item.to_dict() for item in state.actions if item.status == "pending"],
                enabled_tools=allowed_tool_names,
                actor_role=state.goal.actor_role,
                plan_tool_hints=list(state.plan.tool_hints if state.plan else []),
                prompt_profile="plan_execute",
            )

            messages: list[Any] = [SystemMessage(content=system_prompt)]
            if history_messages:
                messages.extend(history_messages)
            messages.append(HumanMessage(content=user_prompt))
            llm_with_tools = executor.llm.bind_tools(list(tool_map.values()))
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
            made_progress = False

            @retry(
                stop=stop_after_attempt(3),
                wait=wait_exponential(multiplier=1, min=1, max=8),
                retry=retry_if_exception_type(_RETRYABLE_TOOL_LOOP_LLM_ERRORS),
                before_sleep=lambda rs: logger.warning(
                    "[plan_execute/%s] tool-loop LLM retry #%d after %s: %s",
                    executor.agent_id,
                    rs.attempt_number,
                    type(rs.outcome.exception()).__name__,
                    rs.outcome.exception(),
                ),
            )
            async def _invoke_tool_loop_llm(current_messages: list[Any]) -> Any:
                return await llm_with_tools.ainvoke(current_messages)

            for loop_index in range(max(1, state.runtime_config.subagent_max_iterations + 1)):
                llm_started_at = time.monotonic()
                response = await _invoke_tool_loop_llm(messages)
                latency_ms = int((time.monotonic() - llm_started_at) * 1000)
                aggregate_latency_ms += latency_ms
                content, reasoning = executor._extract_response_texts(response)
                selected = executor._select_response_text(content=content, reasoning=reasoning)
                aggregate_usage = executor._merge_usage(
                    aggregate_usage,
                    _extract_usage_from_tool_loop_response(response),
                )
                executor.last_content = content
                executor.last_reasoning_content = reasoning
                executor.last_selected_text = selected
                messages.append(response)
                collected_events.append(
                    await emit_llm_completed(
                        agent_id=executor.agent_id,
                        agent_role=executor.role,
                        usage=_extract_usage_from_tool_loop_response(response),
                        latency_ms=latency_ms,
                    )
                )

                tool_calls = list(getattr(response, "tool_calls", []) or [])
                if not tool_calls:
                    executor.last_usage = aggregate_usage
                    executor.last_latency_ms = aggregate_latency_ms
                    return (
                        ToolLoopOutcome(
                            summary=selected or executor.latest_reasoning_summary(),
                            tool_calls=invoked_tool_calls,
                            made_progress=made_progress,
                        ),
                        collected_events,
                    )

                if invoked_tool_calls >= state.runtime_config.subagent_max_iterations:
                    executor.last_usage = aggregate_usage
                    executor.last_latency_ms = aggregate_latency_ms
                    summary = selected or executor.latest_reasoning_summary()
                    if not summary:
                        summary = "已达到工具调用上限，请基于已有结果结束。"
                    return (
                        ToolLoopOutcome(summary=summary, tool_calls=invoked_tool_calls, made_progress=made_progress),
                        collected_events,
                    )

                for tool_index, tool_call in enumerate(tool_calls, start=1):
                    tool_name = str(tool_call.get("name") or "").strip()
                    tool_args = tool_call.get("args") if isinstance(tool_call.get("args"), dict) else {}
                    tool_call_id = str(tool_call.get("id") or f"{executor.agent_id}-tool-loop-{loop_index}-{tool_index}")
                    tool_observation = ""

                    if not tool_name or tool_name not in tool_map:
                        result: dict[str, Any] = {
                            "status": "error",
                            "message": f"Tool '{tool_name}' is not available for role '{executor.role}'",
                        }
                        tool_observation = result["message"]
                        collected_events.append(
                            await emit(
                                "tool_call_completed",
                                {
                                    "agentId": executor.agent_id,
                                    "agentRole": executor.role,
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
                            tool_observation = f"工具 {tool_name} 的相同参数已执行过，请基于现有结果继续。"
                        else:
                            state.attempted_tool_signatures.append(signature)
                            result, tool_events = await call_tool(
                                role_agent_id=executor.agent_id,
                                role_name=executor.role,
                                tool_name=tool_name,
                                tool_args=tool_args,
                                tool_call_id=tool_call_id,
                            )
                            for event in tool_events:
                                collected_events.append(event)
                            invoked_tool_calls += 1
                            tool_observation = _build_tool_loop_observation(tool_name, result)

                            if result.get("status") == "confirmation_required":
                                action = MultiAgentActionItem(
                                    action_id=f"ps-action-{uuid.uuid4().hex[:8]}",
                                    tool_name=tool_name,
                                    tool_args=tool_args,
                                    title=tool_name,
                                    reason="plan_execute_write_action",
                                    status="awaiting_confirmation",
                                    confirm_id=_confirmation_id_from_result(result),
                                )
                                state.actions.append(action)
                                _append_pending_confirmation(state, result)
                                state.execution = ExecutionArtifact(
                                    summary="已生成待确认写操作，等待用户确认。",
                                    actions=[action.to_dict()],
                                    awaiting_confirmation=True,
                                )
                                executor.last_usage = aggregate_usage
                                executor.last_latency_ms = aggregate_latency_ms
                                return (
                                    ToolLoopOutcome(summary="已生成待确认写操作。", tool_calls=invoked_tool_calls, made_progress=True),
                                    collected_events,
                                )

                            made_progress = True
                            state.remember_tool(
                                agent_id=executor.agent_id,
                                agent_role=executor.role,
                                tool_name=tool_name,
                                tool_args=tool_args,
                                tool_call_id=tool_call_id,
                                result=result,
                            )

                    messages.append(ToolMessage(content=tool_observation, tool_call_id=tool_call_id))

            executor.last_usage = aggregate_usage
            executor.last_latency_ms = aggregate_latency_ms
            return (
                ToolLoopOutcome(
                    summary=executor.latest_reasoning_summary() or "执行结束。",
                    tool_calls=invoked_tool_calls,
                    made_progress=made_progress,
                ),
                collected_events,
            )

        yield {
            "event": "agent_run_started",
            "data": {
                "turnId": request.turn_id,
                "sessionId": request.session_id,
                "agentId": "plan-execute",
                "agentRole": "plan_execute",
                "status": "started",
                "summary": "Plan-and-Execute baseline 已启动",
            },
        }

        resumed_action = state.apply_resume_outcome()
        if resumed_action:
            state.execution = ExecutionArtifact(
                summary="已应用上一轮确认结果，继续推进计划。",
                actions=[resumed_action],
                awaiting_confirmation=False,
            )

        max_rounds = max(1, int(state.runtime_config.lead_max_rounds or 6))
        replan_count = 0

        for round_index in range(1, max_rounds + 1):
            state.loop_round = round_index
            if not state.plan:
                yield await emit(
                    "agent_status",
                    {
                        "agentId": planner.agent_id,
                        "agentRole": planner.role,
                        "status": "running",
                        "summary": "Planner 正在生成执行计划",
                    },
                )
                try:
                    plan_result = await planner.plan(
                        user_message=goal_message,
                        page_context=page_context,
                        capabilities=capabilities,
                        actor_role=state.goal.actor_role,
                        evidence_summary=_build_evidence_summary(state),
                        action_history_summary=str(state.action_history or []),
                        continuation=state.continuation,
                        replan_reason=(
                            "基于已有执行结果重规划。"
                            if replan_count > 0
                            else ""
                        ),
                        history_messages=history_messages,
                        prompt_profile="plan_execute",
                    )
                    _record_agent_usage(state, planner)
                    yield await emit_llm_completed(
                        agent_id=planner.agent_id,
                        agent_role=planner.role,
                        usage=planner.last_usage,
                        latency_ms=planner.last_latency_ms,
                    )
                except Exception:
                    logger.exception("PlanExecute planner failed")
                    plan_result = RoleExecutionResult(
                        summary="规划失败，基于直接执行尝试收束。",
                        metadata={"plan_output": {"steps": [], "candidate_tools": [], "risk": "low"}},
                    )

                plan_output = (plan_result.metadata or {}).get("plan_output", {})
                state.plan = PlanArtifact(
                    summary=plan_output.get("raw_summary") or plan_result.summary,
                    steps=plan_output.get("steps", []),
                    candidate_tools=plan_output.get("candidate_tools", []),
                    tool_hints=plan_output.get("tool_hints", []),
                    risk=plan_output.get("risk", "low"),
                )
                yield await emit(
                    "agent_status",
                    {
                        "agentId": planner.agent_id,
                        "agentRole": planner.role,
                        "status": "completed",
                        "summary": _build_plan_summary(state.plan),
                    },
                )

            yield await emit(
                "agent_status",
                {
                    "agentId": executor.agent_id,
                    "agentRole": executor.role,
                    "status": "running",
                    "summary": "Executor 正在按计划推进",
                },
            )
            loop_outcome, loop_events = await run_executor_tool_loop()
            for event in loop_events:
                yield event

            if state.execution and state.execution.awaiting_confirmation:
                state.stop_reason = "awaiting_confirmation"
                yield await emit_checkpoint("已保存计划执行状态，等待用户确认")
                yield await emit(
                    "agent_status",
                    {
                        "agentId": executor.agent_id,
                        "agentRole": executor.role,
                        "status": "awaiting_confirmation",
                        "summary": "Executor 已发起高风险操作，等待用户确认",
                    },
                )
                yield {"event": "done", "data": {"usageSummary": state.usage_summary.to_dict()}}
                return

            if loop_outcome.made_progress:
                state.no_progress_count = 0
            else:
                state.no_progress_count += 1

            execution_summary = loop_outcome.summary or _build_evidence_summary(state) or "执行结束。"
            state.execution = ExecutionArtifact(
                summary=execution_summary,
                actions=[item.to_dict() for item in state.actions],
                awaiting_confirmation=False,
            )
            state.observation = ObservationArtifact(
                summary=_build_evidence_summary(state) or execution_summary,
                evidence=_compact_evidence_for_prompt(state),
                stage_complete=bool(state.tool_records),
            )
            yield await emit(
                "agent_status",
                {
                    "agentId": executor.agent_id,
                    "agentRole": executor.role,
                    "status": "completed",
                    "summary": execution_summary,
                },
            )

            should_finalize = False
            if loop_outcome.made_progress and not getattr(state.execution, "awaiting_confirmation", False):
                if not state.plan or not state.plan.steps:
                    should_finalize = True
                elif state.tool_records:
                    should_finalize = True

            if should_finalize:
                if state.plan and state.plan.steps:
                    state.plan.steps = state.plan.steps[1:]
                break

            if state.no_progress_count >= max(1, int(state.runtime_config.no_progress_rounds or 2)):
                if replan_count >= 1:
                    state.stop_reason = "no_progress"
                    break
                state.plan = None
                replan_count += 1
                continue

            if round_index >= max_rounds:
                state.stop_reason = "max_rounds"
                break

        try:
            final_answer = await summarizer.run_text(
                system_prompt=(
                    "你是 Crater 的 Plan-and-Execute 总结器。"
                    "请基于规划与执行证据，用中文给出最终答复。结论在前，证据在后，建议最后。"
                    "不要编造未验证事实；如果证据不足，要明确说明仍缺什么。"
                ),
                user_prompt=(
                    f"用户请求:\n{goal_message}\n\n"
                    f"计划摘要:\n{state.plan.summary if state.plan else '(empty)'}\n\n"
                    f"计划步骤:\n{state.plan.steps if state.plan else []}\n\n"
                    f"执行摘要:\n{state.execution.summary if state.execution else '(empty)'}\n\n"
                    f"证据摘要:\n{_build_evidence_summary(state) or '(empty)'}\n\n"
                    f"原始紧凑证据:\n{_compact_evidence_for_prompt(state)}\n\n"
                    "请输出面向用户的最终答复。"
                ),
                history_messages=history_messages,
            )
            _record_agent_usage(state, summarizer)
            yield await emit_llm_completed(
                agent_id=summarizer.agent_id,
                agent_role="plan_execute_summarizer",
                usage=summarizer.last_usage,
                latency_ms=summarizer.last_latency_ms,
            )
            state.final_answer = final_answer or state.execution.summary or "执行完成。"
        except Exception:
            logger.exception("PlanExecute summarizer failed")
            state.final_answer = (
                (state.execution.summary if state.execution else "")
                or (_build_evidence_summary(state))
                or "执行完成，但生成最终答复时出错。"
            )

        yield await emit_final_answer(state.final_answer)
        yield {"event": "done", "data": {"usageSummary": state.usage_summary.to_dict()}}
