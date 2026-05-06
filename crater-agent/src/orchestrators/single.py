"""Single-agent orchestrator wrapper."""

from __future__ import annotations

import json
import re
import time
from typing import Any, AsyncIterator

from langchain_core.messages import AIMessage, HumanMessage

from agent.graph import create_agent_graph
from config import settings
from llm.client import ModelClientFactory
from memory.session import build_history_messages
from report_utils import build_pipeline_report_payload
from tools.executor import CompositeToolExecutor, ToolExecutorProtocol
from tools.tool_selector import sanitize_capabilities_for_context


def _extract_llm_usage(output: Any) -> dict[str, int]:
    usage = getattr(output, "usage_metadata", None) or {}
    response_metadata = getattr(output, "response_metadata", None) or {}
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
        "llm_input_tokens": int(input_tokens or 0),
        "llm_output_tokens": int(output_tokens or 0),
        "total_tokens": int(total_tokens or 0),
        "llm_reported_token_calls": 1 if has_reported_usage else 0,
        "llm_missing_token_calls": 0 if has_reported_usage else 1,
    }


def _looks_like_continuation_reply(user_message: str) -> bool:
    normalized = str(user_message or "").strip().lower()
    if not normalized:
        return False
    if normalized in {"确认", "继续", "这个", "就这个", "好的", "ok", "yes", "1", "2", "3"}:
        return True
    if re.fullmatch(r"[A-Za-z0-9]+(?:-[A-Za-z0-9]+){2,}", normalized):
        return True
    return any(
        token in normalized
        for token in (
            "第一个",
            "第1个",
            "第二个",
            "第2个",
            "改成",
            "名字叫",
            "重新来",
            "继续刚才",
            "按刚才",
            "全部",
            "所有",
        )
    )


class SingleAgentOrchestrator:
    def __init__(self, tool_executor: ToolExecutorProtocol | None = None):
        self.tool_executor = tool_executor or CompositeToolExecutor()

    async def stream(self, *, request: Any, model_factory: ModelClientFactory) -> AsyncIterator[dict]:
        # Support both:
        # - ModelClientFactory.create(client_key: str)
        # - role-aware factories with create(purpose=..., orchestration_mode=...)
        try:
            llm = model_factory.create(purpose="default", orchestration_mode="single_agent")
        except TypeError:
            llm = model_factory.create("default")
        graph = create_agent_graph(tool_executor=self.tool_executor, llm=llm)
        context = dict(request.context or {})
        context["capabilities"] = sanitize_capabilities_for_context(context, context.get("capabilities"))
        pending_tool_calls: list[dict[str, Any]] = []
        tool_result_summaries: list[str] = []
        pending_final_content = ""
        emitted_final_answer = False
        emitted_confirmation = False
        llm_started_at: float | None = None
        usage_summary: dict[str, int] = {
            "llm_calls": 0,
            "llm_input_tokens": 0,
            "llm_output_tokens": 0,
            "total_tokens": 0,
            "llm_reported_token_calls": 0,
            "llm_missing_token_calls": 0,
            "reported_token_coverage": 0,
            "llm_latency_ms": 0,
            "tool_latency_ms": 0,
            "tool_calls": 0,
            "read_tool_calls": 0,
            "write_tool_calls": 0,
            "evidence_items": 0,
        }
        initial_state = {
            "messages": [HumanMessage(content=request.message)],
            "context": {
                **context,
                "session_id": request.session_id,
                "turn_id": request.turn_id,
            },
            "tool_call_count": 0,
            "attempted_tool_calls": {},
            "pending_confirmations": [],
            "trace": [],
        }
        history = context.get("history", [])
        if history:
            initial_state["messages"] = build_history_messages(
                history=history,
                max_tokens=settings.history_max_tokens,
                tool_result_max_chars=160,
            ) + initial_state["messages"]
        continuation = context.get("continuation") or {}
        current_request_is_follow_up = _looks_like_continuation_reply(request.message)
        continuation_messages: list[HumanMessage] = []
        pending_confirmation = continuation.get("pending_confirmation") or {}
        if isinstance(pending_confirmation, dict) and pending_confirmation and current_request_is_follow_up:
            pending_summary = {
                "tool_name": pending_confirmation.get("tool_name", ""),
                "result_status": pending_confirmation.get("result_status", ""),
                "tool_args": pending_confirmation.get("tool_args", {}),
            }
            continuation_messages.append(
                HumanMessage(
                    content=(
                        "[系统续接上下文] 上一轮仍有待确认操作："
                        f"{json.dumps(pending_summary, ensure_ascii=False)}。"
                        "只有当用户当前输入明显是在继续这件事时，才沿用该上下文；否则按新请求处理。"
                    )
                )
            )
        elif isinstance(pending_confirmation, dict) and pending_confirmation:
            continuation_messages.append(
                HumanMessage(
                    content=(
                        "[系统续接上下文] 上一轮仍有待确认操作，但当前用户输入看起来是新的独立请求。"
                        "除非用户明确说“确认/继续/这个/第一个/具体名称”，否则不要延续上一轮创建或修改计划；"
                        "优先回答本轮问题。"
                    )
                )
            )
        resume_after_confirmation = continuation.get("resume_after_confirmation") or {}
        if isinstance(resume_after_confirmation, dict) and resume_after_confirmation and current_request_is_follow_up:
            resume_summary = {
                "tool_name": resume_after_confirmation.get("tool_name", ""),
                "result_status": resume_after_confirmation.get("result_status", ""),
                "confirmed": resume_after_confirmation.get("confirmed"),
                "tool_args": resume_after_confirmation.get("tool_args", {}),
                "result": resume_after_confirmation.get("result", {}),
            }
            continuation_messages.append(
                HumanMessage(
                    content=(
                        "[系统续接上下文] 上一轮确认结果："
                        f"{json.dumps(resume_summary, ensure_ascii=False)}。"
                        "当前轮应基于这个结果理解用户后续输入，但不要忽略本轮新的完整请求。"
                    )
                )
            )
        elif isinstance(resume_after_confirmation, dict) and resume_after_confirmation:
            continuation_messages.append(
                HumanMessage(
                    content=(
                        "[系统续接上下文] 上一轮确认流程已经结束，但当前用户输入看起来不是续接语句。"
                        "不要把上一轮写操作当作当前默认目标；若本轮是在问失败原因、状态或教程，"
                        "就按新的诊断/检索请求处理。"
                    )
                )
            )
        if continuation_messages:
            initial_state["messages"] = (
                initial_state["messages"][:-1]
                + continuation_messages
                + initial_state["messages"][-1:]
            )

        yield {
            "event": "agent_run_started",
            "data": {
                "turnId": request.turn_id,
                "sessionId": request.session_id,
                "agentId": "single-agent",
                "agentRole": "single_agent",
                "status": "started",
                "summary": "单核 Agent 已启动",
            },
        }

        async for event in graph.astream_events(initial_state, version="v2"):
            kind = event["event"]
            if kind == "on_chat_model_start":
                llm_started_at = time.monotonic()
                yield {
                    "event": "agent_status",
                    "data": {
                        "turnId": request.turn_id,
                        "agentId": "single-agent",
                        "agentRole": "single_agent",
                        "status": "running",
                        "summary": "Agent 思考中",
                    },
                }
                continue

            if kind == "on_chat_model_end":
                output = event["data"]["output"]
                latency_ms = (
                    int((time.monotonic() - llm_started_at) * 1000)
                    if llm_started_at is not None
                    else 0
                )
                usage = _extract_llm_usage(output)
                usage_summary["llm_calls"] += 1
                usage_summary["llm_input_tokens"] += usage["llm_input_tokens"]
                usage_summary["llm_output_tokens"] += usage["llm_output_tokens"]
                usage_summary["total_tokens"] += usage["total_tokens"]
                usage_summary["llm_reported_token_calls"] += usage["llm_reported_token_calls"]
                usage_summary["llm_missing_token_calls"] += usage["llm_missing_token_calls"]
                usage_summary["reported_token_coverage"] = (
                    usage_summary["llm_reported_token_calls"] / usage_summary["llm_calls"]
                    if usage_summary["llm_calls"]
                    else 0
                )
                usage_summary["llm_latency_ms"] += latency_ms
                yield {
                    "event": "llm_call_completed",
                    "data": {
                        "turnId": request.turn_id,
                        "agentId": "single-agent",
                        "agentRole": "single_agent",
                        "latencyMs": latency_ms,
                        "usage": usage,
                    },
                }
                llm_started_at = None
                if isinstance(output, AIMessage):
                    if output.tool_calls:
                        for tc in output.tool_calls:
                            pending_tool_calls.append(tc)
                            yield {
                                "event": "tool_call_started",
                                "data": {
                                    "turnId": request.turn_id,
                                    "agentId": "single-agent",
                                    "agentRole": "single_agent",
                                    "toolCallId": tc.get("id"),
                                    "toolName": tc["name"],
                                    "toolArgs": tc["args"],
                                    "status": "executing",
                                },
                            }
                    else:
                        # Extract content — handle qwen thinking mode where content
                        # may be empty but reasoning_content holds the actual reply.
                        final_content = output.content or ""
                        if not final_content:
                            # qwen3 thinking mode: content in additional_kwargs
                            final_content = (
                                getattr(output, "reasoning_content", "")
                                or (output.additional_kwargs or {}).get("reasoning_content", "")
                            )
                        if final_content:
                            pending_final_content = final_content
                continue

            if kind == "on_tool_end":
                output = event["data"].get("output", "")
                raw_output = getattr(output, "content", output)
                pending = pending_tool_calls.pop(0) if pending_tool_calls else {}
                tool_name = pending.get("name") or event.get("name", "unknown")
                tool_call_id = pending.get("id")

                try:
                    result_data = raw_output if isinstance(raw_output, dict) else json.loads(str(raw_output))
                except (json.JSONDecodeError, TypeError):
                    result_data = None

                if isinstance(result_data, dict) and result_data.get("status") == "confirmation_required":
                    confirmation = result_data.get("confirmation", {})
                    tool_latency_ms = int(result_data.get("_latency_ms") or 0)
                    emitted_confirmation = True
                    usage_summary["tool_calls"] += 1
                    usage_summary["write_tool_calls"] += 1
                    usage_summary["evidence_items"] += 1
                    usage_summary["tool_latency_ms"] += tool_latency_ms
                    yield {
                        "event": "tool_call_confirmation_required",
                        "data": {
                            "turnId": request.turn_id,
                            "agentId": "single-agent",
                            "agentRole": "single_agent",
                            "toolCallId": tool_call_id,
                            "confirmId": confirmation.get("confirm_id", ""),
                            "action": confirmation.get("tool_name", tool_name),
                            "description": confirmation.get("description", ""),
                            "interaction": confirmation.get("interaction", "approval"),
                            "form": confirmation.get("form"),
                            "status": "awaiting_confirmation",
                            "latencyMs": tool_latency_ms,
                        },
                    }
                    continue

                tool_result_summaries.append(f"{tool_name}: {str(raw_output)[:300]}")
                tool_latency_ms = (
                    int(result_data.get("_latency_ms") or 0)
                    if isinstance(result_data, dict)
                    else 0
                )
                usage_summary["tool_calls"] += 1
                usage_summary["read_tool_calls"] += 1
                usage_summary["evidence_items"] += 1
                usage_summary["tool_latency_ms"] += tool_latency_ms
                yield {
                    "event": "tool_call_completed",
                    "data": {
                        "turnId": request.turn_id,
                        "agentId": "single-agent",
                        "agentRole": "single_agent",
                        "toolCallId": tool_call_id,
                        "toolName": tool_name,
                        "result": raw_output,
                        "resultSummary": str(raw_output)[:500],
                        "status": "error" if isinstance(result_data, dict) and result_data.get("status") == "error" else "done",
                        "isError": isinstance(result_data, dict) and result_data.get("status") == "error",
                        "latencyMs": tool_latency_ms,
                    },
                }
                report_payload = (
                    build_pipeline_report_payload(tool_name, result_data)
                    if isinstance(result_data, dict)
                    else None
                )
                if report_payload:
                    yield {
                        "event": "pipeline_report",
                        "data": report_payload,
                    }
                continue

            if kind == "on_chain_end" and event.get("name") == "tools":
                output = event["data"].get("output", {})
                if not pending_tool_calls:
                    continue
                if not isinstance(output, dict):
                    continue

                # Build a lookup of confirmation results keyed by tool_call_id
                # so we can match each confirmation to the correct pending_tool_call.
                pending_confs = output.get("pending_confirmations") or []
                conf_by_tc_id: dict[str, dict] = {}
                for conf in pending_confs:
                    tc_id = conf.get("_tool_call_id")
                    if tc_id:
                        conf_by_tc_id[tc_id] = conf

                tool_trace = [
                    entry for entry in (output.get("trace") or [])
                    if isinstance(entry, dict) and entry.get("node") == "tools"
                ]
                tool_messages = list(output.get("messages") or [])

                for idx, entry in enumerate(tool_trace):
                    pending = pending_tool_calls.pop(0) if pending_tool_calls else {}
                    tool_call_id = pending.get("id")
                    tool_name = entry.get("tool_name") or pending.get("name") or "unknown"
                    tool_args = (
                        entry.get("tool_args")
                        if isinstance(entry.get("tool_args"), dict)
                        else pending.get("args") or {}
                    )

                    # Check if this tool_call is a confirmation
                    if tool_call_id and tool_call_id in conf_by_tc_id:
                        conf = conf_by_tc_id[tool_call_id]
                        confirmation = conf.get("confirmation", {})
                        emitted_confirmation = True
                        usage_summary["tool_calls"] += 1
                        usage_summary["write_tool_calls"] += 1
                        usage_summary["evidence_items"] += 1
                        tool_latency_ms = int(entry.get("latency_ms", 0) or 0)
                        usage_summary["tool_latency_ms"] += tool_latency_ms
                        yield {
                            "event": "tool_call_confirmation_required",
                            "data": {
                                "turnId": request.turn_id,
                                "agentId": "single-agent",
                                "agentRole": "single_agent",
                                "toolCallId": tool_call_id,
                                "confirmId": confirmation.get("confirm_id", ""),
                                "action": confirmation.get("tool_name", tool_name),
                                "description": confirmation.get("description", ""),
                                "interaction": confirmation.get("interaction", "approval"),
                                "form": confirmation.get("form"),
                                "status": "awaiting_confirmation",
                                "latencyMs": tool_latency_ms,
                            },
                        }
                    else:
                        # Normal tool completion
                        result_status = str(entry.get("result_status") or "unknown").strip().lower()
                        is_error = result_status == "error"
                        raw_output = ""
                        if idx < len(tool_messages):
                            raw_output = str(getattr(tool_messages[idx], "content", "") or "")
                        tool_result_summaries.append(f"{tool_name}: {raw_output[:300]}")
                        usage_summary["tool_calls"] += 1
                        usage_summary["read_tool_calls"] += 1
                        usage_summary["evidence_items"] += 1
                        usage_summary["tool_latency_ms"] += int(entry.get("latency_ms", 0) or 0)
                        yield {
                            "event": "tool_call_completed",
                            "data": {
                                "turnId": request.turn_id,
                                "agentId": "single-agent",
                                "agentRole": "single_agent",
                                "toolCallId": tool_call_id,
                                "toolName": tool_name,
                                "toolArgs": tool_args,
                                "result": raw_output,
                                "resultSummary": raw_output[:500],
                                "status": "error" if is_error else "done",
                                "isError": is_error,
                                "latencyMs": entry.get("latency_ms", 0),
                            },
                        }
                continue

        # Cancel any orphaned tool_call_started events that never got executed
        # (e.g., LLM requested tools but limit was hit before tools_node ran)
        for tc in pending_tool_calls:
            yield {
                "event": "tool_call_completed",
                "data": {
                    "turnId": request.turn_id,
                    "agentId": "single-agent",
                    "agentRole": "single_agent",
                    "toolCallId": tc.get("id"),
                    "toolName": tc.get("name", "unknown"),
                    "toolArgs": tc.get("args", {}),
                    "result": "",
                    "resultSummary": "已超过单轮工具调用上限，本次调用已取消",
                    "status": "cancelled",
                    "isError": False,
                },
            }
        pending_tool_calls.clear()

        if pending_final_content and not emitted_confirmation:
            emitted_final_answer = True
            yield {
                "event": "final_answer",
                "data": {
                    "turnId": request.turn_id,
                    "sessionId": request.session_id,
                    "agentId": "single-agent",
                    "agentRole": "single_agent",
                    "content": pending_final_content,
                    "usageSummary": dict(usage_summary),
                },
            }

        if not emitted_final_answer and not emitted_confirmation:
            if tool_result_summaries:
                details = "\n".join(f"- {item}" for item in tool_result_summaries)
                fallback_content = (
                    "我已经完成了本轮工具调用，但模型没有正常产出最终答复。"
                    "已拿到的结果如下：\n"
                    f"{details}\n\n"
                    "你可以基于这些结果继续追问，或者重试一次，我会继续推进。"
                )
            else:
                fallback_content = (
                    "本轮执行已结束，但模型没有正常产出最终答复。"
                    "请直接重试一次；如果问题持续出现，我可以继续帮你定位具体是哪一步卡住了。"
                )
            yield {
                "event": "final_answer",
                "data": {
                    "turnId": request.turn_id,
                    "sessionId": request.session_id,
                    "agentId": "single-agent",
                    "agentRole": "single_agent",
                    "content": fallback_content,
                    "usageSummary": dict(usage_summary),
                },
            }

        yield {"event": "done", "data": {"usageSummary": dict(usage_summary)}}
