"""Model profile parsing helpers for Crater Agent."""

from __future__ import annotations

import logging
import os
from dataclasses import dataclass, field
from typing import Any

from crater_agent.config import settings

logger = logging.getLogger(__name__)

NO_AUTH_API_KEY_PLACEHOLDER = "sk-no-auth-required"


@dataclass
class ModelProfile:
    """Normalized model profile used by orchestrators."""

    name: str
    provider: str = "openai_compatible"
    base_url: str = ""
    api_key: str = ""
    api_key_env: str = ""
    model: str = ""
    temperature: float = 0.1
    max_tokens: int = 4096
    timeout: int = 60
    max_retries: int = 2
    headers: dict[str, str] = field(default_factory=dict)
    model_kwargs: dict[str, Any] = field(default_factory=dict)
    purpose: str = "default"

    @classmethod
    def from_dict(cls, raw: dict[str, Any], fallback_name: str) -> "ModelProfile":
        return cls(
            name=str(raw.get("name") or fallback_name),
            provider=str(raw.get("provider") or "openai_compatible"),
            base_url=str(raw.get("base_url") or ""),
            api_key=str(raw.get("api_key") or ""),
            api_key_env=str(raw.get("api_key_env") or ""),
            model=str(raw.get("model") or ""),
            temperature=float(raw.get("temperature") or 0.1),
            max_tokens=int(raw.get("max_tokens") or 4096),
            timeout=int(raw.get("timeout") or 60),
            max_retries=int(raw.get("max_retries") or 2),
            headers={
                str(key): str(value)
                for key, value in (raw.get("headers") or {}).items()
            },
            model_kwargs={
                str(key): value
                for key, value in (raw.get("model_kwargs") or {}).items()
            },
            purpose=str(raw.get("purpose") or "default"),
        )

    def resolved_api_key(self) -> str:
        if self.api_key_env:
            env_value = os.getenv(self.api_key_env, "").strip()
            if env_value:
                return env_value
        if self.api_key.strip():
            return self.api_key.strip()
        # LangChain/OpenAI client often expects an api_key field even for local
        # no-auth compatible endpoints such as Ollama / gateway shims.
        return NO_AUTH_API_KEY_PLACEHOLDER


def default_fallback_profile(name: str = "default") -> ModelProfile:
    return ModelProfile(
        name=name,
        provider="openai_compatible",
        base_url=settings.llm_base_url,
        api_key=settings.llm_api_key,
        api_key_env="",
        model=settings.llm_model_name,
        temperature=settings.llm_temperature,
        max_tokens=settings.llm_max_tokens,
        timeout=settings.tool_execution_timeout,
        max_retries=2,
        purpose="default",
    )


def _explicit_runtime_profile_overrides() -> dict[str, Any]:
    overrides: dict[str, Any] = {}

    base_url = os.getenv("CRATER_AGENT_LLM_BASE_URL", "").strip()
    if base_url:
        overrides["base_url"] = base_url

    model_name = os.getenv("CRATER_AGENT_LLM_MODEL_NAME", "").strip()
    if model_name:
        overrides["model"] = model_name

    api_key = os.getenv("CRATER_AGENT_LLM_API_KEY", "").strip()
    if api_key:
        overrides["api_key"] = api_key

    temperature = os.getenv("CRATER_AGENT_LLM_TEMPERATURE", "").strip()
    if temperature:
        try:
            overrides["temperature"] = float(temperature)
        except ValueError:
            pass

    max_tokens = os.getenv("CRATER_AGENT_LLM_MAX_TOKENS", "").strip()
    if max_tokens:
        try:
            overrides["max_tokens"] = int(max_tokens)
        except ValueError:
            pass

    return overrides


def _apply_runtime_profile_overrides(
    lookup: dict[str, ModelProfile],
    overrides: dict[str, Any],
) -> dict[str, ModelProfile]:
    if not overrides:
        return lookup

    logger.warning(
        "Runtime env overrides active for ALL LLM profiles: %s",
        ", ".join(f"{k}={v}" for k, v in overrides.items()),
    )

    for profile_name, profile in list(lookup.items()):
        if "base_url" in overrides:
            profile.base_url = str(overrides["base_url"])
        if "model" in overrides:
            profile.model = str(overrides["model"])
        if "api_key" in overrides:
            profile.api_key_env = ""
            profile.api_key = str(overrides["api_key"])
        if "temperature" in overrides:
            profile.temperature = float(overrides["temperature"])
        if "max_tokens" in overrides:
            profile.max_tokens = int(overrides["max_tokens"])
        lookup[profile_name] = profile
    return lookup


_ALL_ROLES = ("coordinator", "planner", "explorer", "executor", "verifier", "guide", "general")


def coerce_supported_config(raw: dict[str, Any] | None) -> dict[str, Any]:
    """Convert standard llm_clients JSON to normalized internal format."""
    if not raw:
        return {}

    candidate_configs = raw.get("llm_clients")
    if not isinstance(candidate_configs, dict):
        return {}

    default_profile = str(
        raw.get("default_client_key")
        or ("default" if "default" in candidate_configs else next(iter(candidate_configs), "default"))
    )

    profiles: list[dict[str, Any]] = []
    for name, cfg in candidate_configs.items():
        if not isinstance(cfg, dict):
            continue
        profiles.append(
            {
                "name": name,
                "provider": cfg.get("provider", "openai_compatible"),
                "base_url": cfg.get("base_url") or settings.llm_base_url,
                "api_key": cfg.get("api_key") or "",
                "api_key_env": cfg.get("api_key_env") or "",
                "model": cfg.get("model") or settings.llm_model_name,
                "temperature": cfg.get("temperature", settings.llm_temperature),
                "max_tokens": cfg.get("max_tokens") or settings.llm_max_tokens,
                "timeout": cfg.get("timeout", settings.tool_execution_timeout),
                "headers": cfg.get("headers") or {},
                "model_kwargs": cfg.get("model_kwargs") or {},
                "purpose": cfg.get("purpose") or name,
            }
        )

    single_agent_profile = str(
        (raw.get("single_agent") or {}).get("client_key") or default_profile
    )

    role_source = raw.get("multi_agent_roles")
    if not isinstance(role_source, dict):
        role_source = {}

    return {
        "default_profile": default_profile,
        "profiles": profiles,
        "single_agent": {"profile": single_agent_profile},
        "multi_agent": {
            "roles": {
                role: role_source.get(role, default_profile) for role in _ALL_ROLES
            }
        },
    }


def normalize_profiles(raw: dict[str, Any] | None) -> dict[str, Any]:
    profiles = coerce_supported_config(raw)
    profiles.setdefault("default_profile", "default")
    profiles.setdefault("profiles", [])
    profiles.setdefault("single_agent", {"profile": profiles["default_profile"]})
    profiles.setdefault(
        "multi_agent",
        {
            "roles": {role: profiles["default_profile"] for role in _ALL_ROLES}
        },
    )
    return profiles


def build_profile_lookup(raw: dict[str, Any] | None) -> tuple[dict[str, ModelProfile], dict[str, Any]]:
    normalized = normalize_profiles(raw)
    lookup: dict[str, ModelProfile] = {}
    for entry in normalized.get("profiles", []):
        if not isinstance(entry, dict):
            continue
        profile = ModelProfile.from_dict(entry, fallback_name=str(entry.get("name") or "default"))
        lookup[profile.name] = profile

    default_name = str(normalized.get("default_profile") or "default")
    if default_name not in lookup:
        lookup[default_name] = default_fallback_profile(default_name)

    fallback = default_fallback_profile(default_name)
    for profile_name, profile in list(lookup.items()):
        if not profile.base_url:
            profile.base_url = fallback.base_url
        if not profile.model:
            profile.model = fallback.model
        if not profile.max_tokens:
            profile.max_tokens = fallback.max_tokens
        if not profile.timeout:
            profile.timeout = fallback.timeout
        lookup[profile_name] = profile

    lookup = _apply_runtime_profile_overrides(lookup, _explicit_runtime_profile_overrides())
    return lookup, normalized
