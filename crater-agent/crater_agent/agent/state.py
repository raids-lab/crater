"""LangGraph State definition for Crater Agent."""

from __future__ import annotations

import operator
from typing import Annotated, Any

from langgraph.graph import MessagesState


class CraterAgentState(MessagesState):
    """State that persists through the entire ReAct loop.

    Inherits `messages` from MessagesState (list of BaseMessage with add_messages reducer).
    """

    # User and page context injected by Go backend
    context: dict[str, Any]

    # Track tool call count to prevent infinite loops
    tool_call_count: int

    # Track same-turn tool invocations to avoid repeated calls with identical args
    attempted_tool_calls: dict[str, int]

    # When a write operation needs user confirmation, store it here
    # The ReAct loop pauses and returns this to the frontend
    pending_confirmation: dict[str, Any] | None

    # Accumulated trace records for evaluation/auditing
    # Uses operator.add reducer so each node's trace entries are appended, not replaced
    trace: Annotated[list[dict[str, Any]], operator.add]
