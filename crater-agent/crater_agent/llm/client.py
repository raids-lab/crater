"""LLM client factory for Crater Agent."""

from __future__ import annotations

from typing import Any

from langchain_openai import ChatOpenAI

from crater_agent.config import settings
from crater_agent.llm.profiles import build_profile_lookup, default_fallback_profile


class ModelClientFactory:
    """Builds ChatOpenAI clients from request-scoped profile config."""

    def __init__(self, raw_profiles: dict[str, Any] | None = None):
        if raw_profiles is None:
            raw_profiles = settings.load_llm_clients_config()
        self._lookup, self._normalized = build_profile_lookup(raw_profiles)

    @property
    def normalized_profiles(self) -> dict[str, Any]:
        return self._normalized

    def resolve_profile_name(self, purpose: str, orchestration_mode: str) -> str:
        default_name = str(self._normalized.get("default_profile") or "default")
        if orchestration_mode == "multi_agent":
            roles = ((self._normalized.get("multi_agent") or {}).get("roles") or {})
            return str(roles.get(purpose) or default_name)
        if orchestration_mode == "single_agent":
            single = self._normalized.get("single_agent") or {}
            return str(single.get("profile") or default_name)
        return default_name

    def create(self, *, purpose: str, orchestration_mode: str) -> ChatOpenAI:
        profile_name = self.resolve_profile_name(purpose, orchestration_mode)
        profile = self._lookup.get(profile_name) or default_fallback_profile(profile_name)
        client_kwargs: dict[str, Any] = {
            "base_url": profile.base_url,
            "api_key": profile.resolved_api_key(),
            "model": profile.model,
            "temperature": profile.temperature,
            "max_tokens": profile.max_tokens,
            "timeout": profile.timeout,
            "max_retries": profile.max_retries,
        }
        if profile.headers:
            client_kwargs["default_headers"] = profile.headers
        if profile.model_kwargs:
            client_kwargs["model_kwargs"] = profile.model_kwargs
        return ChatOpenAI(**client_kwargs)
