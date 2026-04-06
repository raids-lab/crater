"""LangGraph ReAct agent graph for Crater.

The agent uses a simple ReAct loop:
  agent (LLM think) → tools (execute) → agent (observe & think again) → ...
  until LLM decides to respond without tool calls → END

Key design: LLM autonomously decides which tools to call and when to stop.
No fixed workflow or intent classification — pure ReAct.
"""

from __future__ import annotations

import json
import logging
import time
from typing import Any

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage, ToolMessage
from langchain_openai import ChatOpenAI
from langgraph.graph import END, StateGraph
from openai import BadRequestError

from crater_agent.agent.prompts import build_system_prompt
from crater_agent.agent.state import CraterAgentState
from crater_agent.config import settings
from crater_agent.llm.client import ModelClientFactory
from crater_agent.skills.loader import load_all_skills
from crater_agent.tools.definitions import ALL_TOOLS
from crater_agent.tools.executor import GoBackendToolExecutor, ToolExecutorProtocol

logger = logging.getLogger(__name__)


def _truncate_text(value: str, max_chars: int = 2400) -> str:
    if len(value) <= max_chars:
        return value
    head = value[: max_chars // 2]
    tail = value[-max_chars // 2 :]
    return f"{head}\n\n...(内容过长，已截断)...\n\n{tail}"


def _build_tool_observation(tool_name: str, result: dict[str, Any]) -> str:
    if result.get("status") == "confirmation_required":
        return json.dumps(result, ensure_ascii=False)

    if result.get("status") == "error":
        error_payload = {
            "status": "error",
            "tool_name": tool_name,
            "message": result.get("message", ""),
            "error_type": result.get("error_type", "unknown"),
            "retryable": bool(result.get("retryable", False)),
        }
        if "status_code" in result:
            error_payload["status_code"] = result["status_code"]
        if isinstance(result.get("result"), dict):
            error_payload["result"] = result["result"]
        return _truncate_text(json.dumps(error_payload, ensure_ascii=False), max_chars=1200)

    result_content = result.get("result", result.get("message", ""))
    if isinstance(result_content, dict):
        compact = dict(result_content)
        if tool_name == "get_job_logs" and isinstance(compact.get("log"), str):
            compact["log"] = _truncate_text(compact["log"], max_chars=1000)
            compact["note"] = "日志内容已截断，只保留部分片段供推理使用"
        return _truncate_text(json.dumps(compact, ensure_ascii=False), max_chars=1400)

    return _truncate_text(str(result_content), max_chars=1400)


def _is_context_limit_error(exc: Exception) -> bool:
    message = str(exc or "")
    return (
        "exceed_context_size_error" in message
        or "available context size" in message
        or "maximum context length" in message
    )


def _compact_message(message: Any) -> Any:
    if isinstance(message, SystemMessage):
        return SystemMessage(content=_truncate_text(str(message.content or ""), max_chars=1600))
    if isinstance(message, HumanMessage):
        return HumanMessage(content=_truncate_text(str(message.content or ""), max_chars=600))
    if isinstance(message, ToolMessage):
        return ToolMessage(
            content=_truncate_text(str(message.content or ""), max_chars=800),
            tool_call_id=getattr(message, "tool_call_id", "unknown"),
        )
    if isinstance(message, AIMessage):
        return AIMessage(
            content=_truncate_text(str(message.content or ""), max_chars=600),
            tool_calls=list(getattr(message, "tool_calls", []) or []),
            additional_kwargs=dict(getattr(message, "additional_kwargs", {}) or {}),
            response_metadata=dict(getattr(message, "response_metadata", {}) or {}),
        )
    return message


def _compact_messages_for_retry(messages: list[Any]) -> list[Any]:
    if not messages:
        return messages

    system_message = messages[0] if isinstance(messages[0], SystemMessage) else None
    body = list(messages[1:] if system_message else messages)
    tail = body[-6:]
    last_human = next((msg for msg in reversed(body) if isinstance(msg, HumanMessage)), None)

    compacted: list[Any] = []
    if system_message:
        compacted.append(_compact_message(system_message))
    if last_human and last_human not in tail:
        compacted.append(_compact_message(last_human))
    compacted.extend(_compact_message(msg) for msg in tail)
    return compacted


def create_llm() -> ChatOpenAI:
    """Create the default single-agent LLM instance."""
    return ModelClientFactory().create("default")


def create_agent_graph(
    tool_executor: ToolExecutorProtocol | None = None,
    skills_dir: str | None = None,
    llm: ChatOpenAI | None = None,
) -> StateGraph:
    """Build the LangGraph StateGraph for the Crater ReAct agent.

    Args:
        tool_executor: Tool executor instance. Defaults to GoBackendToolExecutor.
        skills_dir: Path to skills YAML directory. Defaults to built-in skills.
    """
    if tool_executor is None:
        tool_executor = GoBackendToolExecutor()

    llm = llm or create_llm()

    # Load skills knowledge for system prompt
    skills_context = load_all_skills(skills_dir)

    def get_enabled_tools(context: dict[str, Any]) -> list[Any]:
        capabilities = context.get("capabilities", {})
        enabled_tool_names = capabilities.get("enabled_tools") or []
        if not enabled_tool_names:
            return ALL_TOOLS
        enabled_set = set(enabled_tool_names)
        return [tool for tool in ALL_TOOLS if tool.name in enabled_set]

    # ----- Node: agent (LLM reasoning) -----
    async def agent_node(state: CraterAgentState) -> dict:
        """LLM thinks about the current state and decides next action."""
        messages = state["messages"]
        context = state.get("context", {})
        llm_with_tools = llm.bind_tools(get_enabled_tools(context))

        # Build system prompt on first call (check if system message exists)
        if not messages or not isinstance(messages[0], SystemMessage):
            actor = context.get("actor", {})
            is_first_time = actor.get("is_first_time", False)
            system_prompt = build_system_prompt(
                context=context,
                skills_context=skills_context,
                is_first_time=is_first_time,
                user_message=str(messages[-1].content) if messages else "",
            )
            messages = [SystemMessage(content=system_prompt)] + list(messages)

        try:
            response = await llm_with_tools.ainvoke(messages)
        except BadRequestError as exc:
            if not _is_context_limit_error(exc):
                raise
            compact_messages = _compact_messages_for_retry(messages)
            logger.warning(
                "Single-agent context limit hit, retrying with compacted messages: %d -> %d",
                len(messages),
                len(compact_messages),
            )
            response = await llm_with_tools.ainvoke(compact_messages)

        # Handle qwen thinking mode: content may be empty, actual reply in reasoning_content
        if not response.content and not response.tool_calls:
            reasoning = getattr(response, "reasoning_content", "") or (
                response.additional_kwargs or {}
            ).get("reasoning_content", "")
            if reasoning:
                response = AIMessage(
                    content=reasoning,
                    additional_kwargs=response.additional_kwargs,
                    response_metadata=response.response_metadata,
                )

        # Record trace
        trace_entry = {
            "node": "agent",
            "timestamp": time.time(),
            "has_tool_calls": bool(response.tool_calls),
            "tool_calls_count": len(response.tool_calls) if response.tool_calls else 0,
            "response_length": len(response.content) if response.content else 0,
        }

        return {
            "messages": [response],
            "trace": [trace_entry],
        }

    # ----- Node: tools (execute tool calls) -----
    async def tools_node(state: CraterAgentState) -> dict:
        """Execute the tool calls from the last AI message."""
        messages = state["messages"]
        last_message = messages[-1]
        context = state.get("context", {})
        session_id = context.get("session_id", "unknown")
        actor = context.get("actor", {})
        user_id = actor.get("user_id", 0)
        turn_id = context.get("turn_id")

        tool_messages = []
        trace_entries = []
        new_tool_call_count = state.get("tool_call_count", 0)
        pending_confirmation = None
        attempted_tool_calls = dict(state.get("attempted_tool_calls") or {})

        for tc in last_message.tool_calls:
            tool_name = tc["name"]
            tool_args = tc["args"]
            new_tool_call_count += 1
            tool_signature = json.dumps(
                {
                    "tool_name": tool_name,
                    "tool_args": tool_args,
                },
                ensure_ascii=False,
                sort_keys=True,
            )
            if attempted_tool_calls.get(tool_signature, 0) >= 1:
                tool_messages.append(
                    ToolMessage(
                        content=(
                            f"工具 {tool_name} 在本轮中已用相同参数调用过一次，请不要重复调用，"
                            "应基于现有结果直接回答用户。"
                        ),
                        tool_call_id=tc["id"],
                    )
                )
                trace_entries.append({
                    "node": "tools",
                    "timestamp": time.time(),
                    "tool_name": tool_name,
                    "tool_args": tool_args,
                    "result_status": "duplicate_skipped",
                    "latency_ms": 0,
                })
                continue
            attempted_tool_calls[tool_signature] = attempted_tool_calls.get(tool_signature, 0) + 1

            # Execute via Go backend (or mock)
            result = await tool_executor.execute(
                tool_name=tool_name,
                tool_args=tool_args,
                session_id=session_id,
                user_id=user_id,
                turn_id=turn_id,
                tool_call_id=tc["id"],
                agent_id="single-agent",
                agent_role="single_agent",
            )

            # Record trace
            trace_entries.append({
                "node": "tools",
                "timestamp": time.time(),
                "tool_name": tool_name,
                "tool_args": tool_args,
                "result_status": result.get("status", "unknown"),
                "latency_ms": result.get("_latency_ms", 0),
            })

            # Handle confirmation-required tools
            if result.get("status") == "confirmation_required":
                pending_confirmation = result
                tool_messages.append(
                    ToolMessage(
                        content=json.dumps(result, ensure_ascii=False),
                        tool_call_id=tc["id"],
                    )
                )
            else:
                tool_messages.append(
                    ToolMessage(
                        content=_build_tool_observation(tool_name, result),
                        tool_call_id=tc["id"],
                    )
                )

        return {
            "messages": tool_messages,
            "tool_call_count": new_tool_call_count,
            "attempted_tool_calls": attempted_tool_calls,
            "pending_confirmation": pending_confirmation,
            "trace": trace_entries,
        }

    # ----- Conditional edge: should the agent continue? -----
    def should_continue(state: CraterAgentState) -> str:
        """Determine if the agent should call tools, wait for confirmation, or respond."""
        messages = state["messages"]
        last_message = messages[-1]
        tool_call_count = state.get("tool_call_count", 0)

        # Safety: max tool calls reached → force respond
        if tool_call_count >= settings.max_tool_calls_per_turn:
            return "respond"

        # If there's a pending confirmation → pause and respond
        if state.get("pending_confirmation"):
            return "respond"

        # If the LLM produced tool_calls → execute them
        if isinstance(last_message, AIMessage) and last_message.tool_calls:
            return "tools"

        # Otherwise, LLM decided to respond directly → end
        return "respond"

    def after_tools(state: CraterAgentState) -> str:
        if state.get("pending_confirmation"):
            return "respond"
        return "agent"

    # ----- Build the graph -----
    graph = StateGraph(CraterAgentState)
    graph.add_node("agent", agent_node)
    graph.add_node("tools", tools_node)

    graph.set_entry_point("agent")
    graph.add_conditional_edges(
        "agent",
        should_continue,
        {
            "tools": "tools",
            "respond": END,
        },
    )
    graph.add_conditional_edges(
        "tools",
        after_tools,
        {
            "agent": "agent",
            "respond": END,
        },
    )

    return graph.compile()
