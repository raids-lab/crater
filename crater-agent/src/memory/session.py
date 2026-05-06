"""Session memory management.

Handles conversation history loading and token budget management.
History is provided by the Go backend (loaded from PostgreSQL) and
passed in the request context.
"""

from __future__ import annotations

from langchain_core.messages import AIMessage, HumanMessage

from llm.tokenizer import count_tokens


def estimate_tokens(text: str) -> int:
    """Count tokens in text using tiktoken (with heuristic fallback)."""
    return count_tokens(text)


def _truncate_head_tail(text: str, max_chars: int) -> str:
    """Truncate keeping head and tail for better context preservation."""
    if len(text) <= max_chars:
        return text
    half = max_chars // 2
    return f"{text[:half]}\n\n...(内容过长，已截断)...\n\n{text[-half:]}"


def build_history_messages(
    history: list[dict],
    max_tokens: int = 4000,
    tool_result_max_chars: int = 1200,
    tool_error_max_chars: int = 1600,
) -> list:
    """Build LangChain message objects from Go-provided history.

    Loads messages from most recent backwards until token budget is exhausted.
    Tool results are truncated (head+tail) to save tokens while preserving
    key information at both ends of the output.

    Args:
        history: List of message dicts from Go backend
                 [{"role": "user", "content": "..."}, ...]
        max_tokens: Maximum token budget for history
        tool_result_max_chars: Max chars for tool result content
        tool_error_max_chars: Max chars for tool error content

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

        # Truncate tool results with head+tail to preserve key info
        if role == "tool":
            is_error = any(kw in content for kw in ("error", "Error", "failed", "Failed", "错误", "失败"))
            limit = tool_error_max_chars if is_error else tool_result_max_chars
            content = _truncate_head_tail(content, limit)

        msg_tokens = estimate_tokens(content)
        if token_count + msg_tokens > max_tokens:
            break

        if role == "user":
            selected.append(HumanMessage(content=content))
        elif role == "assistant":
            selected.append(AIMessage(content=content))
        elif role == "tool":
            tool_call_id = str(msg.get("tool_call_id", "unknown") or "unknown").strip()
            selected.append(
                AIMessage(
                    content=f"【历史工具结果 {tool_call_id}】{content}",
                )
            )

        token_count += msg_tokens

    return list(reversed(selected))
