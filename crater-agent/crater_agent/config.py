"""Configuration for Crater Agent Service."""

import json
from pathlib import Path
from typing import Any

from pydantic import Field
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Agent service configuration, loaded from environment variables."""

    # Legacy single-client fallback. Used only when no llm_clients config is provided.
    llm_base_url: str = Field(default="http://localhost:11434/v1", description="LLM API base URL")
    llm_api_key: str = Field(default="", description="Legacy fallback LLM API key")
    llm_model_name: str = Field(default="qwen2.5:14b", description="LLM model name")
    llm_temperature: float = Field(default=0.1, description="LLM temperature for tool calling")
    llm_max_tokens: int = Field(default=4096, description="Max tokens per LLM response")
    default_orchestration_mode: str = Field(
        default="single_agent", description="Default orchestration mode for agent chat"
    )

    # Preferred runtime config: simple llm_clients map + role routing.
    llm_clients_config_json: str = Field(
        default="",
        description="JSON-encoded llm_clients config for single-agent and multi-agent routing",
    )
    llm_clients_config_path: str = Field(
        default="",
        description="Optional path to a JSON file containing llm_clients config",
    )
    llm_clients_preset: str = Field(
        default="",
        description="Optional built-in role-aware LLM preset used when no explicit client config is provided",
    )

    # Crater Go Backend
    crater_backend_url: str = Field(
        default="http://localhost:8080", description="Crater Go backend URL"
    )
    crater_backend_internal_token: str = Field(
        default="", description="Shared token for Python Agent -> Go internal tool execution"
    )

    # Agent Behavior
    max_tool_calls_per_turn: int = Field(
        default=10, description="Max tool calls in a single ReAct loop"
    )
    tool_execution_timeout: int = Field(default=30, description="Tool execution timeout (seconds)")
    history_max_tokens: int = Field(
        default=4000, description="Max tokens for conversation history"
    )

    # Service
    host: str = Field(default="0.0.0.0")
    port: int = Field(default=8000)
    debug: bool = Field(default=False)

    model_config = {
        "env_prefix": "CRATER_AGENT_",
        "env_file": ".env",
        # .env may also contain non-prefixed secret variables such as DASHSCOPE_API_KEY
        # referenced indirectly by llm client configs. They should not fail Settings init.
        "extra": "ignore",
    }

    def normalized_default_orchestration_mode(self) -> str:
        return (
            "multi_agent"
            if self.default_orchestration_mode.strip().lower() == "multi_agent"
            else "single_agent"
        )

    @staticmethod
    def _read_optional_json(path: str) -> str:
        normalized = path.strip()
        if not normalized:
            return ""
        return Path(normalized).read_text(encoding="utf-8").strip()

    @staticmethod
    def _is_local_legacy_base_url(base_url: str) -> bool:
        normalized = base_url.strip().rstrip("/")
        return normalized in {"", "http://localhost:11434/v1"}

    def _resolve_preset_base_url(self, preset_name: str) -> str:
        base_url = self.llm_base_url.strip()
        normalized = preset_name.strip().lower()
        if normalized.startswith("dashscope") and self._is_local_legacy_base_url(base_url):
            return "https://dashscope.aliyuncs.com/compatible-mode/v1"
        return base_url

    def _build_client_entry(
        self,
        *,
        base_url: str,
        model: str,
        temperature: float,
        max_tokens: int,
        timeout: int,
        api_key_env: str = "",
    ) -> dict[str, Any]:
        entry = {
            "provider": "openai_compatible",
            "base_url": base_url,
            "model": model,
            "temperature": temperature,
            "max_tokens": max_tokens,
            "timeout": timeout,
        }
        if api_key_env:
            entry["api_key_env"] = api_key_env
        elif self.llm_api_key.strip():
            entry["api_key"] = self.llm_api_key.strip()
        return entry

    def _build_preset_config(self, preset_name: str) -> dict[str, Any]:
        normalized = preset_name.strip().lower()
        if normalized not in {
            "dashscope_multi_agent_v1",
            "dashscope_role_aware_v1",
            "dashscope_role_aware",
            "dashscope",
        }:
            return {}

        base_url = self._resolve_preset_base_url(normalized)
        fast_timeout = max(self.tool_execution_timeout, 45)
        strong_timeout = max(self.tool_execution_timeout, 90)
        planner_timeout = max(self.tool_execution_timeout, 120)

        llm_clients = {
            "single_default": self._build_client_entry(
                base_url=base_url,
                model="qwen3.6-plus",
                temperature=0.1,
                max_tokens=max(self.llm_max_tokens, 4096),
                timeout=strong_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "coordinator_router": self._build_client_entry(
                base_url=base_url,
                model="qwen3.6-plus",
                temperature=0.05,
                max_tokens=max(self.llm_max_tokens, 4096),
                timeout=strong_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "planner_reasoner": self._build_client_entry(
                base_url=base_url,
                model="qwen3.5-122b-a10b",
                temperature=0.05,
                max_tokens=max(self.llm_max_tokens, 6144),
                timeout=planner_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "explorer_fast": self._build_client_entry(
                base_url=base_url,
                model="qwen3.5-flash",
                temperature=0.1,
                max_tokens=min(max(self.llm_max_tokens, 2048), 4096),
                timeout=fast_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "executor_careful": self._build_client_entry(
                base_url=base_url,
                model="qwen3.6-plus",
                temperature=0.05,
                max_tokens=max(self.llm_max_tokens, 4096),
                timeout=strong_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "verifier_strict": self._build_client_entry(
                base_url=base_url,
                model="qwen3.5-122b-a10b",
                temperature=0.0,
                max_tokens=min(max(self.llm_max_tokens, 2048), 4096),
                timeout=strong_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "guide_fast": self._build_client_entry(
                base_url=base_url,
                model="qwen3.5-flash",
                temperature=0.2,
                max_tokens=min(max(self.llm_max_tokens, 2048), 4096),
                timeout=fast_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "general_fast": self._build_client_entry(
                base_url=base_url,
                model="qwen3.5-flash",
                temperature=0.15,
                max_tokens=min(max(self.llm_max_tokens, 2048), 4096),
                timeout=fast_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
        }

        return {
            "default_client_key": "single_default",
            "llm_clients": llm_clients,
            "single_agent": {"client_key": "single_default"},
            "multi_agent_roles": {
                "coordinator": "coordinator_router",
                "planner": "planner_reasoner",
                "explorer": "explorer_fast",
                "executor": "executor_careful",
                "verifier": "verifier_strict",
                "guide": "guide_fast",
                "general": "general_fast",
            },
        }

    def load_llm_clients_config(self) -> dict[str, Any]:
        raw = self.llm_clients_config_json.strip()
        if not raw:
            raw = self._read_optional_json(self.llm_clients_config_path)
        if not raw:
            preset_name = self.llm_clients_preset.strip()
            if preset_name:
                return self._build_preset_config(preset_name)
            return {}
        loaded = json.loads(raw)
        if not isinstance(loaded, dict):
            raise ValueError("CRATER_AGENT_LLM_CLIENTS_CONFIG must decode to a JSON object")
        return loaded

    def public_agent_config_summary(self) -> dict[str, Any]:
        return {
            "defaultOrchestrationMode": self.normalized_default_orchestration_mode(),
            "availableModes": ["single_agent", "multi_agent"],
        }


settings = Settings()
