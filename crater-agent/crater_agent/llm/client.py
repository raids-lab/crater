"""LLM client factory for Crater Agent."""

from __future__ import annotations

import os
from dataclasses import dataclass, field
from typing import Any

from langchain_openai import ChatOpenAI

from crater_agent.config import settings

NO_AUTH_API_KEY_PLACEHOLDER = "sk-no-auth-required"


@dataclass
class ClientConfig:
    """Direct LLM client config for one role or purpose."""

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

    @classmethod
    def from_dict(cls, name: str, raw: dict[str, Any]) -> "ClientConfig":
        return cls(
            name=name,
            provider=str(raw.get("provider") or "openai_compatible"),
            base_url=str(raw.get("base_url") or ""),
            api_key=str(raw.get("api_key") or ""),
            api_key_env=str(raw.get("api_key_env") or ""),
            model=str(raw.get("model") or ""),
            temperature=float(raw.get("temperature") or 0.1),
            max_tokens=int(raw.get("max_tokens") or 4096),
            timeout=int(raw.get("timeout") or 60),
            max_retries=int(raw.get("max_retries") or 2),
            headers={str(key): str(value) for key, value in (raw.get("headers") or {}).items()},
            model_kwargs={str(key): value for key, value in (raw.get("model_kwargs") or {}).items()},
        )

    def resolved_api_key(self) -> str:
        if self.api_key_env:
            env_value = os.getenv(self.api_key_env, "").strip()
            if env_value:
                return env_value
        if self.api_key.strip():
            return self.api_key.strip()
        return NO_AUTH_API_KEY_PLACEHOLDER


def normalize_client_map(raw: dict[str, Any]) -> dict[str, ClientConfig]:
    """Normalize the direct purpose/role -> client config map."""

    lookup: dict[str, ClientConfig] = {}
    for name, cfg in raw.items():
        if isinstance(cfg, dict):
            lookup[str(name)] = ClientConfig.from_dict(str(name), cfg)

    fallback = lookup.get("default")
    if fallback is None:
        raise ValueError("LLM client config must define a 'default' client")

    for name, cfg in list(lookup.items()):
        if not cfg.base_url:
            cfg.base_url = fallback.base_url
        if not cfg.model:
            cfg.model = fallback.model
        if not cfg.max_tokens:
            cfg.max_tokens = fallback.max_tokens
        if not cfg.timeout:
            cfg.timeout = fallback.timeout
        lookup[name] = cfg

    return lookup


class ModelClientFactory:
    """Builds ChatOpenAI clients from a direct role/purpose -> client map."""

    def __init__(self, raw_clients: dict[str, Any] | None = None):
        if raw_clients is None:
            raw_clients = settings.load_llm_client_configs()
        self._clients = normalize_client_map(raw_clients)

    @property
    def client_map(self) -> dict[str, ClientConfig]:
        return self._clients

    def resolve_client_name(self, purpose: str, orchestration_mode: str) -> str:
        normalized_purpose = str(purpose or "").strip()
        if normalized_purpose and normalized_purpose in self._clients:
            return normalized_purpose
        if orchestration_mode == "single_agent" and "single_agent" in self._clients:
            return "single_agent"
        return "default"

    def create(self, *, purpose: str, orchestration_mode: str) -> ChatOpenAI:
        client_name = self.resolve_client_name(purpose, orchestration_mode)
        config = self._clients[client_name]
        client_kwargs: dict[str, Any] = {
            "base_url": config.base_url,
            "api_key": config.resolved_api_key(),
            "model": config.model,
            "temperature": config.temperature,
            "max_tokens": config.max_tokens,
            "timeout": config.timeout,
            "max_retries": config.max_retries,
        }
        if config.headers:
            client_kwargs["default_headers"] = config.headers
        if config.model_kwargs:
            client_kwargs["model_kwargs"] = config.model_kwargs
        return ChatOpenAI(**client_kwargs)
