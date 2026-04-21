"""LLM-based conversation history compaction.

When the context window is nearing its limit, this module summarises older
conversation messages using the LLM, preserving key investigation context
(tool calls, findings, conclusions) while drastically reducing token count.

Design follows HolmesGPT's compaction approach:
- Tool schemas (bind_tools) are NEVER compressed — they're a separate API param.
- SystemMessage is always preserved uncompacted.
- The most recent UserMessage and the last N messages are preserved.
- Everything else is summarised into a single AIMessage.
"""

from __future__ import annotations

import asyncio
import logging
from typing import Any

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage, ToolMessage

logger = logging.getLogger(__name__)

COMPACTION_PROMPT = """\
请将以下对话历史压缩为简洁的中文摘要。

## 当前用户正在问的问题：
{current_query}

## 历史对话中的用户消息：
{user_messages}

## 历史对话中的助手回复和工具调用结果：
{compressible_messages}

## 压缩规则：
1. **与当前问题相关的历史**：保留关键结论、工具名+参数+结果、技术细节（资源名/命名空间/错误码）
2. **与当前问题无关的历史**：一句话概括即可（如"之前用户询问过 job-X 的状态，已解决"），不需要保留细节
3. 用户的原始意图用简短引用保留，不要改写用户的核心诉求
4. 助手的冗长分析压缩为 1-2 句结论
5. 如果有待完成的操作或未解决的问题，必须保留

注意：历史消息不一定都和当前问题相关，请根据相关性灵活压缩，不需要平均对待每条历史。

请用中文输出紧凑摘要。"""


def _format_messages_for_summary(messages: list[Any]) -> tuple[str, str]:
    """Split messages into user messages (preserve) and compressible messages.

    Returns:
        (user_messages_text, compressible_messages_text)
    """
    user_parts: list[str] = []
    compress_parts: list[str] = []
    for msg in messages:
        content = str(getattr(msg, "content", "") or "")
        if not content:
            continue
        if isinstance(msg, HumanMessage):
            user_parts.append(f"- {content}")
        elif isinstance(msg, AIMessage):
            tool_calls = getattr(msg, "tool_calls", None) or []
            if tool_calls:
                tc_names = ", ".join(tc.get("name", "?") for tc in tool_calls)
                compress_parts.append(f"[助手→调用工具: {tc_names}] {content}")
            else:
                compress_parts.append(f"[助手] {content}")
        elif isinstance(msg, ToolMessage):
            tool_call_id = getattr(msg, "tool_call_id", "")
            compress_parts.append(f"[工具结果 {tool_call_id}] {content}")
        elif isinstance(msg, SystemMessage):
            continue
        else:
            compress_parts.append(f"[{type(msg).__name__}] {content}")
    return "\n".join(user_parts) or "（无）", "\n".join(compress_parts) or "（无）"


async def compact_messages_with_llm(
    messages: list[Any],
    llm: Any,
    preserve_tail: int = 4,
    timeout: float = 15.0,
    current_query: str = "",
) -> list[Any] | None:
    """Summarise older messages using the LLM to reduce context window usage.

    Args:
        messages: Full message list (including SystemMessage at index 0).
        llm: A ChatOpenAI (or compatible) LLM instance.
        preserve_tail: Number of most-recent messages to keep uncompacted.
        timeout: Maximum seconds to wait for the LLM summary call.
        current_query: The current user question — used as anchor for
                relevance-based compression. History unrelated to this
                query is compressed more aggressively.

    Returns:
        Compacted message list on success, or ``None`` on failure (caller
        should fall back to the existing hard-truncation strategy).
    """
    if len(messages) <= preserve_tail + 2:
        return None  # Not enough messages to compact

    # Split: system message + body
    system_msg = messages[0] if isinstance(messages[0], SystemMessage) else None
    body = messages[1:] if system_msg else list(messages)

    if len(body) <= preserve_tail:
        return None

    # Partition into compactable and preserved sections
    to_compact = body[:-preserve_tail]
    tail = body[-preserve_tail:]

    # Ensure the last user message is in the preserved section
    last_human = next(
        (m for m in reversed(body) if isinstance(m, HumanMessage)), None
    )
    tail_has_human = any(isinstance(m, HumanMessage) for m in tail)

    conversation_text = _format_messages_for_summary(to_compact)
    user_text, compress_text = conversation_text
    if not user_text.strip() and not compress_text.strip():
        return None

    # Derive current query from preserved tail if not explicitly provided
    if not current_query:
        for m in reversed(tail):
            if isinstance(m, HumanMessage):
                current_query = str(m.content or "")[:300]
                break

    prompt = COMPACTION_PROMPT.format(
        current_query=current_query or "（未提供）",
        user_messages=user_text,
        compressible_messages=compress_text,
    )

    try:
        response = await asyncio.wait_for(
            llm.ainvoke([HumanMessage(content=prompt)]),
            timeout=timeout,
        )
        summary = str(getattr(response, "content", "") or "").strip()
        if not summary:
            logger.warning("LLM compaction returned empty summary, falling back")
            return None
    except asyncio.TimeoutError:
        logger.warning("LLM compaction timed out after %.1fs, falling back", timeout)
        return None
    except Exception:
        logger.warning("LLM compaction failed, falling back", exc_info=True)
        return None

    # Assemble compacted messages
    compacted: list[Any] = []
    if system_msg:
        compacted.append(system_msg)
    compacted.append(AIMessage(content=f"[对话摘要] {summary}"))
    if last_human and not tail_has_human:
        compacted.append(last_human)
    compacted.extend(tail)
    return compacted
