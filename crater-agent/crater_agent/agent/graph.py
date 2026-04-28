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
from crater_agent.tools.tool_selector import (
    _resolve_actor_role,
    sanitize_capabilities_for_context,
    select_tools_for_context,
)

logger = logging.getLogger(__name__)


def _truncate_text(value: str, max_chars: int = 2400) -> str:
    if len(value) <= max_chars:
        return value
    head = value[: max_chars // 2]
    tail = value[-max_chars // 2 :]
    return f"{head}\n\n...(内容过长，已截断)...\n\n{tail}"


_DEFAULT_TOOL_TOKEN_BUDGET = 3000

_TOOL_TOKEN_BUDGETS: dict[str, int] = {
    "get_job_logs": 4000,
    "diagnose_job": 4000,
    "get_diagnostic_context": 4000,
    "get_job_detail": 3000,
    "prometheus_query": 2000,
    "query_job_metrics": 2000,
}

_TOOL_EXTRACT_PROMPT = """\
从以下工具输出中提取与用户问题最相关的关键信息。

工具: {tool_name}
用户问题: {user_question}

工具完整输出:
{tool_output}

要求：
- 保留所有错误信息、异常堆栈、关键状态码
- 保留与用户问题直接相关的数据
- 保留资源名、命名空间、时间戳等关键标识
- 删除重复的正常日志行、冗余的健康检查输出
- 用紧凑格式输出，不要添加额外解释"""


def _truncate_to_token_budget(text: str, budget_tokens: int) -> str:
    """Truncate text to fit within a token budget, keeping head + tail."""
    from crater_agent.llm.tokenizer import count_tokens

    if count_tokens(text) <= budget_tokens:
        return text
    max_chars = budget_tokens * 3  # conservative: ~3 chars/token
    return _truncate_text(text, max_chars=max_chars)


async def _extract_with_llm(
    tool_name: str,
    raw_text: str,
    budget_tokens: int,
    llm: Any,
    user_question: str,
) -> str | None:
    """Use LLM to intelligently extract key info from oversized tool results.

    Returns extracted text on success, None on failure (caller falls back to
    hard truncation).
    """
    import asyncio

    prompt = _TOOL_EXTRACT_PROMPT.format(
        tool_name=tool_name,
        user_question=user_question or "（未知）",
        tool_output=raw_text,
    )
    try:
        response = await asyncio.wait_for(
            llm.ainvoke([HumanMessage(content=prompt)]),
            timeout=10.0,
        )
        extracted = str(getattr(response, "content", "") or "").strip()
        if not extracted:
            return None
        logger.info(
            "LLM extract for %s: %d chars -> %d chars",
            tool_name, len(raw_text), len(extracted),
        )
        return extracted
    except asyncio.TimeoutError:
        logger.warning("LLM extract for %s timed out", tool_name)
        return None
    except Exception:
        logger.warning("LLM extract for %s failed", tool_name, exc_info=True)
        return None


def _build_tool_observation_sync(tool_name: str, result: dict[str, Any]) -> str:
    """Build tool observation string (synchronous, hard truncation only)."""
    if result.get("status") == "confirmation_required":
        return json.dumps(result, ensure_ascii=False)

    budget = _TOOL_TOKEN_BUDGETS.get(tool_name, _DEFAULT_TOOL_TOKEN_BUDGET)

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
        return _truncate_to_token_budget(
            json.dumps(error_payload, ensure_ascii=False), budget
        )

    result_content = result.get("result", result.get("message", ""))
    if isinstance(result_content, dict):
        return _truncate_to_token_budget(
            json.dumps(result_content, ensure_ascii=False), budget
        )

    return _truncate_to_token_budget(str(result_content), budget)


async def _build_tool_observation(
    tool_name: str,
    result: dict[str, Any],
    llm: Any = None,
    user_question: str = "",
) -> str:
    """Build tool observation, using LLM extract for oversized results.

    Flow:
    1. Serialize the raw result
    2. If within token budget → return as-is
    3. If over budget AND llm provided → LLM extract (semantic compression)
    4. If LLM extract fails or no llm → fallback to head+tail hard truncation
    """
    if result.get("status") == "confirmation_required":
        return json.dumps(result, ensure_ascii=False)

    budget = _TOOL_TOKEN_BUDGETS.get(tool_name, _DEFAULT_TOOL_TOKEN_BUDGET)
    from crater_agent.llm.tokenizer import count_tokens

    # Serialize the raw result
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
        raw_text = json.dumps(error_payload, ensure_ascii=False)
    else:
        result_content = result.get("result", result.get("message", ""))
        if isinstance(result_content, dict):
            raw_text = json.dumps(result_content, ensure_ascii=False)
        else:
            raw_text = str(result_content)

    # Fast path: within budget, return as-is
    if count_tokens(raw_text) <= budget:
        return raw_text

    # Slow path: over budget → try LLM extract, then fallback to hard truncation
    if llm is not None:
        extracted = await _extract_with_llm(
            tool_name, raw_text, budget, llm, user_question,
        )
        if extracted is not None:
            # Ensure extracted result fits the budget (LLM may still overshoot)
            return _truncate_to_token_budget(extracted, budget)

    return _truncate_to_token_budget(raw_text, budget)


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


def _estimate_message_tokens(messages: list[Any]) -> int:
    """Count tokens across messages using tiktoken (with heuristic fallback)."""
    from crater_agent.llm.tokenizer import count_message_tokens

    return count_message_tokens(messages)


def _extract_current_query(messages: list[Any]) -> str:
    """Find the most recent user question from the message list."""
    for msg in reversed(messages):
        if isinstance(msg, HumanMessage):
            return str(msg.content or "")[:300]
    return ""


async def _proactive_compact(
    messages: list[Any], max_context: int, llm: Any = None,
) -> list[Any]:
    """Proactively compact messages before hitting the API context limit.

    Reserves budget for tool schemas (~8000 tokens) and LLM response (~4000 tokens).
    First attempts LLM-based summarisation; falls back to hard truncation on failure.
    """
    tool_schema_budget = 8000
    response_budget = 4000
    available = max_context - tool_schema_budget - response_budget
    if available <= 0:
        return messages
    estimated = _estimate_message_tokens(messages)
    if estimated <= available:
        return messages
    logger.info(
        "Proactive compaction: estimated %d tokens > available %d, compacting",
        estimated, available,
    )
    # Try LLM summarisation first
    if llm is not None:
        from crater_agent.agent.compaction import compact_messages_with_llm

        compacted = await compact_messages_with_llm(
            messages, llm, current_query=_extract_current_query(messages),
        )
        if compacted is not None:
            logger.info(
                "LLM compaction succeeded: %d -> %d messages",
                len(messages), len(compacted),
            )
            return compacted
    # Fallback to hard truncation
    return _compact_messages_for_retry(messages)


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
        capabilities = sanitize_capabilities_for_context(context, context.get("capabilities"))
        enabled_tool_names = capabilities.get("enabled_tools") or []
        if enabled_tool_names:
            enabled_set = set(enabled_tool_names)
            base_tools = [tool for tool in ALL_TOOLS if tool.name in enabled_set]
        else:
            base_tools = ALL_TOOLS
        return select_tools_for_context(context, base_tools)

    # ----- Node: agent (LLM reasoning) -----
    async def agent_node(state: CraterAgentState) -> dict:
        """LLM thinks about the current state and decides next action."""
        messages = state["messages"]
        context = dict(state.get("context", {}) or {})
        context["capabilities"] = sanitize_capabilities_for_context(context, context.get("capabilities"))
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
            messages = await _proactive_compact(
                messages, max_context=settings.max_context_tokens, llm=llm,
            )
            llm_start = time.time()
            response = await llm_with_tools.ainvoke(messages)
            llm_ms = int((time.time() - llm_start) * 1000)
            logger.info(
                "agent_node: LLM responded in %dms, tool_calls=%d, content_len=%d",
                llm_ms, len(response.tool_calls) if response.tool_calls else 0,
                len(response.content) if response.content else 0,
            )
        except BadRequestError as exc:
            if not _is_context_limit_error(exc):
                raise
            # Try LLM compaction on context limit error before hard fallback
            from crater_agent.agent.compaction import compact_messages_with_llm

            compact_messages = await compact_messages_with_llm(
                messages, llm, current_query=_extract_current_query(messages),
            )
            if compact_messages is None:
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
        context = dict(state.get("context", {}) or {})
        context["capabilities"] = sanitize_capabilities_for_context(context, context.get("capabilities"))
        # Extract user question for LLM tool result extraction context
        _user_question = ""
        for msg in reversed(messages):
            if isinstance(msg, HumanMessage):
                _user_question = str(msg.content or "")[:200]
                break
        session_id = context.get("session_id", "unknown")
        actor = context.get("actor", {})
        user_id = actor.get("user_id", 0)
        actor_role = _resolve_actor_role(context)
        turn_id = context.get("turn_id")

        tool_messages = []
        trace_entries = []
        new_tool_call_count = state.get("tool_call_count", 0)
        pending_confirmations: list[dict[str, Any]] = []
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
            tool_started_at = time.perf_counter()
            result = await tool_executor.execute(
                tool_name=tool_name,
                tool_args=tool_args,
                session_id=session_id,
                user_id=user_id,
                turn_id=turn_id,
                tool_call_id=tc["id"],
                agent_id="single-agent",
                agent_role="single_agent",
                actor_role=actor_role,
            )
            measured_tool_latency_ms = max(1, int((time.perf_counter() - tool_started_at) * 1000))
            if not isinstance(result, dict):
                result = {"status": "error", "message": str(result)}
            if not result.get("_latency_ms"):
                result["_latency_ms"] = measured_tool_latency_ms

            # Record trace
            trace_entries.append({
                "node": "tools",
                "timestamp": time.time(),
                "tool_name": tool_name,
                "tool_args": tool_args,
                "result_status": result.get("status", "unknown"),
                "latency_ms": result.get("_latency_ms", 0),
            })

            # Collect ALL confirmation-required results, tagged with tool_call_id
            # for correct matching in the orchestrator.
            if result.get("status") == "confirmation_required":
                pending_confirmations.append({**result, "_tool_call_id": tc["id"]})
                tool_messages.append(
                    ToolMessage(
                        content=json.dumps(result, ensure_ascii=False),
                        tool_call_id=tc["id"],
                    )
                )
            else:
                observation = await _build_tool_observation(
                    tool_name, result, llm=llm, user_question=_user_question,
                )
                tool_messages.append(
                    ToolMessage(content=observation, tool_call_id=tc["id"])
                )

        executed = [e.get("tool_name", "?") for e in trace_entries if e.get("node") == "tools"]
        if executed:
            logger.info("tools_node: executed %s, total_count=%d", executed, new_tool_call_count)

        return {
            "messages": tool_messages,
            "tool_call_count": new_tool_call_count,
            "attempted_tool_calls": attempted_tool_calls,
            "pending_confirmations": pending_confirmations,
            "trace": trace_entries,
        }

    # ----- Node: summarize (forced synthesis when tool limit reached) -----
    async def summarize_node(state: CraterAgentState) -> dict:
        """Force LLM to synthesize existing evidence when tool call limit is reached."""
        messages = list(state["messages"])

        # Inject a nudge so the LLM knows it must summarize now
        messages.append(
            HumanMessage(
                content=(
                    "[系统提示] 你已达到本轮工具调用上限，无法继续调用工具。"
                    "请基于已收集到的所有工具返回结果，直接给出完整的综合分析回答。"
                )
            )
        )

        try:
            # No tools bound — LLM can only produce text
            response = await llm.ainvoke(messages)
        except BadRequestError as exc:
            if not _is_context_limit_error(exc):
                raise
            from crater_agent.agent.compaction import compact_messages_with_llm

            compact_messages = await compact_messages_with_llm(
                messages, llm, current_query=_extract_current_query(messages),
            )

            if compact_messages is None:
                compact_messages = _compact_messages_for_retry(messages)
            logger.warning(
                "Summarize node context limit hit, retrying with compacted messages: %d -> %d",
                len(messages),
                len(compact_messages),
            )
            response = await llm.ainvoke(compact_messages)

        # Handle qwen thinking mode
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

        trace_entry = {
            "node": "summarize",
            "timestamp": time.time(),
            "response_length": len(response.content) if response.content else 0,
        }

        return {
            "messages": [response],
            "trace": [trace_entry],
        }

    # ----- Conditional edge: should the agent continue? -----
    def should_continue(state: CraterAgentState) -> str:
        """Determine if the agent should call tools, wait for confirmation, or respond."""
        messages = state["messages"]
        last_message = messages[-1]
        tool_call_count = state.get("tool_call_count", 0)

        # Safety: max tool calls reached
        if tool_call_count >= settings.max_tool_calls_per_turn:
            # If the LLM still wanted to call tools, route to summarize node
            # so it can synthesize existing evidence instead of a raw fallback.
            if isinstance(last_message, AIMessage) and last_message.tool_calls:
                return "summarize"
            return "respond"

        # If there are pending confirmations → pause and respond
        if state.get("pending_confirmations"):
            return "respond"

        # If the LLM produced tool_calls → execute them
        if isinstance(last_message, AIMessage) and last_message.tool_calls:
            return "tools"

        # Otherwise, LLM decided to respond directly → end
        return "respond"

    def after_tools(state: CraterAgentState) -> str:
        if state.get("pending_confirmations"):
            return "respond"
        return "agent"

    # ----- Build the graph -----
    graph = StateGraph(CraterAgentState)
    graph.add_node("agent", agent_node)
    graph.add_node("tools", tools_node)
    graph.add_node("summarize", summarize_node)

    graph.set_entry_point("agent")
    graph.add_conditional_edges(
        "agent",
        should_continue,
        {
            "tools": "tools",
            "summarize": "summarize",
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
    # summarize always ends the graph
    graph.add_edge("summarize", END)

    return graph.compile()
