from __future__ import annotations

import os
import hmac

from config import settings


def expected_internal_token() -> str:
    for key in (
        "CRATER_AGENT_INTERNAL_TOKEN",
        "CRATER_AGENT_CRATER_BACKEND_INTERNAL_TOKEN",
        "CRATER_AGENT_AGENT_INTERNAL_TOKEN",
    ):
        value = str(os.getenv(key) or "").strip()
        if value:
            return value

    for value in (
        settings.crater_backend_internal_token,
        settings.agent_internal_token,
    ):
        normalized = str(value or "").strip()
        if normalized:
            return normalized

    return ""


def verify_internal_token(header_value: str | None) -> bool:
    expected = expected_internal_token()
    provided = str(header_value or "").strip()
    return bool(expected and provided and hmac.compare_digest(provided, expected))
