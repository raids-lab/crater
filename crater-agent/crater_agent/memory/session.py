"""Session memory management.

Handles conversation history loading and token budget management.
History is provided by the Go backend (loaded from PostgreSQL) and
passed in the request context.
"""

from __future__ import annotations

from langchain_core.messages import AIMessage, HumanMessage, ToolMessage


def estimate_tokens(text: str) -> int:
    """Rough token estimate: ~1 token per 2 Chinese chars or 4 English chars."""
    cn_chars = sum(1 for c in text if "\u4e00" <= c <= "\u9fff")
    en_chars = len(text) - cn_chars
    return cn_chars // 2 + en_chars // 4 + 1


def build_history_messages(
    history: list[dict],
    max_tokens: int = 4000,
    tool_result_max_chars: int = 200,
) -> list:
    """Build LangChain message objects from Go-provided history.

    Loads messages from most recent backwards until token budget is exhausted.
    Tool results are truncated to save tokens.

    Args:
        history: List of message dicts from Go backend
                 [{"role": "user", "content": "..."}, ...]
        max_tokens: Maximum token budget for history
        tool_result_max_chars: Max chars for tool result content

    Returns:
        List of LangChain message objects, in chronological order
    """
    if not history:
        return []

    selected = []
    token_count = 0

    for msg in reversed(history):
        role = msg.get("role", "")
        content = msg.get("content", "")

        # Truncate tool results to save tokens
        if role == "tool" and len(content) > tool_result_max_chars:
            content = content[:tool_result_max_chars] + "... (truncated)"

        msg_tokens = estimate_tokens(content)
        if token_count + msg_tokens > max_tokens:
            break

        if role == "user":
            selected.append(HumanMessage(content=content))
        elif role == "assistant":
            selected.append(AIMessage(content=content))
        elif role == "tool":
            selected.append(
                ToolMessage(
                    content=content,
                    tool_call_id=msg.get("tool_call_id", "unknown"),
                )
            )

        token_count += msg_tokens

    return list(reversed(selected))
