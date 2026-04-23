"""Base class for ticket/order evaluation agents.

All ticket agents share the same execution pattern:
  1. Run a ReAct loop with a restricted tool whitelist
  2. Extract a structured verdict from the agent's output
  3. Fall back to BaseRoleAgent.run_json if extraction fails
  4. Never raise exceptions — always return a safe default verdict

Subclasses only define domain-specific parts:
  - Which tools to use
  - What system prompt to inject
  - How to parse the verdict
  - What the safe default verdict is

This enables different ticket types (approval, quota, dataset access,
node maintenance, ...) to reuse the same evaluation infrastructure.
"""

from __future__ import annotations

import asyncio
import json
import logging
import re
from abc import ABC, abstractmethod
from typing import Any, Generic, TypeVar

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from pydantic import BaseModel

from crater_agent.agent.graph import create_agent_graph
from crater_agent.agents.base import BaseRoleAgent
from crater_agent.config import settings
from crater_agent.llm.client import ModelClientFactory
from crater_agent.tools.executor import GoBackendToolExecutor

logger = logging.getLogger(__name__)

# Type variables for request/response generics
TRequest = TypeVar("TRequest", bound=BaseModel)
TVerdict = TypeVar("TVerdict", bound=BaseModel)


def _collect_trace(messages: list) -> list[dict[str, Any]]:
    """Extract tool call trace from message history."""
    trace = []
    for msg in messages:
        if isinstance(msg, AIMessage) and msg.tool_calls:
            for tc in msg.tool_calls:
                trace.append({
                    "tool": tc.get("name", ""),
                    "args_summary": str(tc.get("args", {}))[:200],
                })
    return trace


def _collect_evidence(messages: list, max_chars: int = 400) -> str:
    """Build evidence summary from tool results in message history."""
    parts = []
    for msg in messages:
        if hasattr(msg, "name") and hasattr(msg, "content"):
            tool_name = getattr(msg, "name", "unknown")
            content = str(msg.content)[:max_chars]
            parts.append(f"[{tool_name}] {content}")
    return "\n".join(parts) if parts else "(no tool results)"


class TicketAgent(ABC, Generic[TRequest, TVerdict]):
    """Base class for all ticket/order evaluation agents.

    Subclass this to create a new ticket type. You need to implement
    five methods — everything else (ReAct loop, fallback, error handling,
    trace collection) is handled by the base class.

    Example:
        class QuotaAgent(TicketAgent[QuotaRequest, QuotaVerdict]):
            def allowed_tools(self) -> list[str]:
                return ["check_quota", "get_realtime_capacity", ...]

            def system_prompt(self) -> str:
                return "你是配额审批助手..."

            def build_user_message(self, request: QuotaRequest) -> str:
                return f"请评估配额申请：{request.account_name}..."

            def extract_verdict(self, text: str) -> QuotaVerdict | None:
                # parse JSON from LLM output
                ...

            def default_verdict(self) -> QuotaVerdict:
                return QuotaVerdict(verdict="escalate", confidence=0.1)
    """

    def __init__(
        self,
        *,
        agent_id: str = "ticket",
        tool_executor: GoBackendToolExecutor | None = None,
        llm=None,
        llm_purpose: str = "default",
    ):
        self.agent_id = agent_id
        self.tool_executor = tool_executor or GoBackendToolExecutor()
        if llm is None:
            try:
                llm = ModelClientFactory().create(
                    purpose=llm_purpose, orchestration_mode="single_agent"
                )
            except (TypeError, KeyError):
                llm = ModelClientFactory().create("default")
        self.llm = llm
        self._fallback_agent = BaseRoleAgent(
            agent_id=f"{agent_id}-fallback",
            role=agent_id,
            llm=llm,
        )

    # ---------------------------------------------------------------
    # Abstract methods — subclasses must implement these
    # ---------------------------------------------------------------

    @abstractmethod
    def allowed_tools(self) -> list[str]:
        """Return the tool whitelist for this ticket type."""
        ...

    @abstractmethod
    def system_prompt(self) -> str:
        """Return the system prompt for this ticket type."""
        ...

    @abstractmethod
    def build_user_message(self, request: TRequest) -> str:
        """Convert a request into the user message for the LLM."""
        ...

    @abstractmethod
    def extract_verdict(self, text: str) -> TVerdict | None:
        """Try to extract a structured verdict from the LLM's response.

        Return None if the text doesn't contain a valid verdict.
        """
        ...

    @abstractmethod
    def default_verdict(self, *, reason: str = "") -> TVerdict:
        """Return the safe default verdict when evaluation fails.

        This is used when the agent times out, crashes, or cannot
        produce a valid verdict. It should always be the safest option
        (e.g., escalate to human).
        """
        ...

    # ---------------------------------------------------------------
    # Optional hooks — subclasses may override
    # ---------------------------------------------------------------

    def build_context(self, request: TRequest) -> dict[str, Any]:
        """Build the context dict for the ReAct graph.

        Override to add request-specific context (e.g., user_id, session_id).
        """
        return {
            "capabilities": {
                "enabled_tools": self.allowed_tools(),
            },
            "actor": {"role": "system"},
        }

    def fallback_prompt(self) -> str:
        """System prompt for the fallback LLM call (no tools).

        Override to customize the fallback verdict extraction prompt.
        """
        return (
            "你是工单评估助手。基于以下工具调查结果，直接输出评估结论 JSON。"
        )

    def set_trace(self, verdict: TVerdict, trace: list[dict[str, Any]]) -> TVerdict:
        """Attach trace data to the verdict. Override if your verdict model
        has a different trace field name."""
        if hasattr(verdict, "trace"):
            verdict.trace = trace  # type: ignore[attr-defined]
        return verdict

    # ---------------------------------------------------------------
    # Shared implementation — do not override
    # ---------------------------------------------------------------

    async def evaluate(self, request: TRequest) -> TVerdict:
        """Run the full evaluation pipeline. Never raises."""
        logger.info("[%s] starting evaluation", self.agent_id)
        try:
            result = await self._do_evaluate(request)
            logger.info("[%s] evaluation completed successfully", self.agent_id)
            return result
        except asyncio.TimeoutError:
            logger.warning("[%s] evaluation timed out", self.agent_id)
            return self.default_verdict(reason="Agent 评估超时，转交人工处理")
        except Exception:
            logger.exception("[%s] evaluation failed", self.agent_id)
            return self.default_verdict(reason="Agent 评估异常，转交人工处理")

    async def _do_evaluate(self, request: TRequest) -> TVerdict:
        """Internal evaluation using ReAct graph."""
        context = self.build_context(request)
        session_id = context.get("session_id", "unknown")
        logger.info("[%s] building ReAct graph, session=%s, tools=%s", self.agent_id, session_id, self.allowed_tools())

        graph = create_agent_graph(
            tool_executor=self.tool_executor,
            llm=self.llm,
        )

        initial_state = {
            "messages": [
                SystemMessage(content=self.system_prompt()),
                HumanMessage(content=self.build_user_message(request)),
            ],
            "context": context,
            "tool_call_count": 0,
            "attempted_tool_calls": {},
            "pending_confirmation": None,
            "force_limit_reached": False,
            "trace": [],
        }

        logger.info("[%s] invoking ReAct graph...", self.agent_id)
        final_state = await graph.ainvoke(initial_state)

        messages = final_state.get("messages", [])
        tool_call_count = final_state.get("tool_call_count", 0)
        trace = _collect_trace(messages)
        logger.info("[%s] ReAct graph done: %d messages, %d tool calls", self.agent_id, len(messages), tool_call_count)

        # Extract verdict from last AI message
        last_ai_content = ""
        for msg in reversed(messages):
            if isinstance(msg, AIMessage) and msg.content:
                last_ai_content = msg.content
                break

        verdict = self.extract_verdict(last_ai_content)
        if verdict is not None:
            logger.info("[%s] verdict extracted directly: %s", self.agent_id, getattr(verdict, "verdict", "?"))
            return self.set_trace(verdict, trace)

        # Fallback: BaseRoleAgent.run_json with no tools
        logger.info("[%s] no verdict in graph output (last_ai_content=%.200s), using fallback", self.agent_id, last_ai_content)
        return await self._fallback_conclude(messages, trace)

    async def _fallback_conclude(
        self,
        messages: list,
        trace: list[dict[str, Any]],
    ) -> TVerdict:
        """Use BaseRoleAgent.run_json to force a verdict from evidence."""
        evidence = _collect_evidence(messages)

        result = await self._fallback_agent.run_json(
            system_prompt=self.fallback_prompt(),
            user_prompt=f"工具调查结果：\n{evidence}\n\n请输出评估结论 JSON。",
        )

        if isinstance(result, dict) and not result.get("raw"):
            try:
                verdict = self._parse_fallback_result(result)
                if verdict is not None:
                    return self.set_trace(verdict, trace)
            except (TypeError, ValueError):
                pass

        default = self.default_verdict(reason=f"Agent 收集了 {len(trace)} 条工具记录但未能得出结论")
        return self.set_trace(default, trace)

    def _parse_fallback_result(self, result: dict) -> TVerdict | None:
        """Try to construct a verdict from the fallback JSON result.

        Override if your verdict model needs special parsing.
        """
        # Default: try to extract verdict from the raw JSON text
        return self.extract_verdict(json.dumps(result, ensure_ascii=False))
