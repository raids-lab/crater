"""Token counting with tiktoken backend and heuristic fallback.

Provides accurate token estimation for context window management.
Uses tiktoken cl100k_base encoding (compatible with Qwen/GPT-4 family).
Falls back to character-based heuristic if tiktoken is unavailable.

All counting is local — zero network requests, zero token consumption.
"""

from __future__ import annotations

import logging
from typing import Any

logger = logging.getLogger(__name__)

_MESSAGE_OVERHEAD_TOKENS = 4  # per-message framing (role, separators)


class TokenCounter:
    """Token counter with tiktoken backend and heuristic fallback."""

    def __init__(self, encoding_name: str = "cl100k_base"):
        self._encoding = None
        try:
            import tiktoken

            self._encoding = tiktoken.get_encoding(encoding_name)
        except Exception:
            logger.info("tiktoken unavailable, using heuristic token estimation")

    def count_text(self, text: str) -> int:
        """Count tokens in a text string."""
        if not text:
            return 0
        if self._encoding is not None:
            return len(self._encoding.encode(text))
        return _heuristic_count(text)

    def count_messages(self, messages: list[Any]) -> int:
        """Count tokens across a list of LangChain messages."""
        total = 0
        for msg in messages:
            content = str(getattr(msg, "content", "") or "")
            total += self.count_text(content) + _MESSAGE_OVERHEAD_TOKENS
        return total


def _heuristic_count(text: str) -> int:
    """Fallback heuristic: ~1 token per 2 CJK chars, ~1 token per 4 Latin chars."""
    cjk = sum(1 for c in text if "\u4e00" <= c <= "\u9fff")
    latin = len(text) - cjk
    return cjk // 2 + latin // 4 + 1


# Module-level singleton, lazily initialized.
_default_counter: TokenCounter | None = None


def get_token_counter() -> TokenCounter:
    """Get the module-level TokenCounter singleton."""
    global _default_counter
    if _default_counter is None:
        from crater_agent.config import settings

        _default_counter = TokenCounter(encoding_name=settings.tokenizer_encoding)
    return _default_counter


def count_tokens(text: str) -> int:
    """Convenience function: count tokens in text using the default counter."""
    return get_token_counter().count_text(text)


def count_message_tokens(messages: list[Any]) -> int:
    """Convenience function: count tokens across messages using the default counter."""
    return get_token_counter().count_messages(messages)
