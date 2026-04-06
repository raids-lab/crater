"""Single-agent orchestrator wrapper."""

from __future__ import annotations

import json
from typing import Any, AsyncIterator

from langchain_core.messages import AIMessage, HumanMessage

from crater_agent.agent.graph import create_agent_graph
from crater_agent.config import settings
from crater_agent.llm.client import ModelClientFactory
from crater_agent.memory.session import build_history_messages
from crater_agent.report_utils import build_pipeline_report_payload
from crater_agent.tools.executor import GoBackendToolExecutor, ToolExecutorProtocol


class SingleAgentOrchestrator:
    def __init__(self, tool_executor: ToolExecutorProtocol | None = None):
        self.tool_executor = tool_executor or GoBackendToolExecutor()

    async def stream(self, *, request: Any, model_factory: ModelClientFactory) -> AsyncIterator[dict]:
        llm = model_factory.create("default")
        graph = create_agent_graph(tool_executor=self.tool_executor, llm=llm)
        context = dict(request.context or {})
        pending_tool_calls: list[dict[str, Any]] = []
        tool_result_summaries: list[str] = []
        pending_final_content = ""
        emitted_final_answer = False
        emitted_confirmation = False
        initial_state = {
            "messages": [HumanMessage(content=request.message)],
            "context": {
                **context,
                "session_id": request.session_id,
                "turn_id": request.turn_id,
            },
            "tool_call_count": 0,
            "attempted_tool_calls": {},
            "pending_confirmation": None,
            "trace": [],
        }
        history = context.get("history", [])
        if history:
            initial_state["messages"] = build_history_messages(
                history=history,
                max_tokens=settings.history_max_tokens,
                tool_result_max_chars=160,
            ) + initial_state["messages"]

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
                    emitted_confirmation = True
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
                        },
                    }
                    continue

                tool_result_summaries.append(f"{tool_name}: {str(raw_output)[:160]}")
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
                pending = pending_tool_calls.pop(0)
                tool_name = pending.get("name", "unknown")
                tool_call_id = pending.get("id")
                pending_confirmation = (
                    output.get("pending_confirmation", {}) if isinstance(output, dict) else {}
                )
                if pending_confirmation:
                    confirmation = pending_confirmation.get("confirmation", {})
                    emitted_confirmation = True
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
                        },
                    }
                    continue

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
                },
            }

        if not emitted_final_answer and not emitted_confirmation:
            if tool_result_summaries:
                details = "\n".join(f"- {item}" for item in tool_result_summaries[-3:])
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
                },
            }

        yield {"event": "done", "data": {}}
