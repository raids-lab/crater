"""Configuration for Crater Agent Service."""

import json
from pathlib import Path
from typing import Any

from pydantic import Field
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Agent service configuration, loaded from environment variables."""

    # LLM config mode:
    # - simple:  use the single model settings below
    # - map:     use llm_clients_config_json/path as a direct role -> client map
    # - preset:  use a built-in preset
    # - auto:    map > preset > simple
    llm_config_mode: str = Field(
        default="auto",
        description="LLM config mode: auto | simple | map | preset",
    )

    # Simple mode: one model for all roles.
    llm_base_url: str = Field(
        default="https://87ba3.gpu.act.buaa.edu.cn/v1",
        description="LLM API base URL",
    )
    llm_api_key: str = Field(default="", description="Legacy fallback LLM API key")
    llm_api_key_env: str = Field(
        default="",
        description="Optional env var name that stores the LLM API key",
    )
    llm_model_name: str = Field(default="qwen3.5", description="LLM model name")
    llm_temperature: float = Field(default=0.1, description="LLM temperature for tool calling")
    llm_max_tokens: int = Field(default=4096, description="Max tokens per LLM response")
    llm_timeout: int = Field(default=90, description="LLM request timeout (seconds)")
    default_orchestration_mode: str = Field(
        default="single_agent", description="Default orchestration mode for agent chat"
    )

    # Map mode: direct role/purpose -> client config JSON map.
    llm_clients_config_json: str = Field(
        default="",
        description="JSON-encoded role/purpose -> client config map",
    )
    llm_clients_config_path: str = Field(
        default="",
        description="Optional path to a JSON file containing the client map",
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

    def normalized_llm_config_mode(self) -> str:
        normalized = self.llm_config_mode.strip().lower()
        if normalized == "profiles":
            return "map"
        if normalized in {"simple", "map", "preset"}:
            return normalized
        return "auto"

    def resolved_llm_config_mode(self) -> str:
        mode = self.normalized_llm_config_mode()
        if mode != "auto":
            return mode
        if self._load_raw_llm_clients_config():
            return "map"
        if self.llm_clients_preset.strip():
            return "preset"
        return "simple"

    @staticmethod
    def _read_optional_json(path: str) -> str:
        normalized = path.strip()
        if not normalized:
            return ""
        return Path(normalized).read_text(encoding="utf-8").strip()

    def _load_raw_llm_clients_config(self) -> str:
        raw = self.llm_clients_config_json.strip()
        if raw:
            return raw
        return self._read_optional_json(self.llm_clients_config_path)

    def _resolve_preset_base_url(self, preset_name: str) -> str:
        normalized = preset_name.strip().lower()
        if normalized.startswith("dashscope"):
            return "https://dashscope.aliyuncs.com/compatible-mode/v1"
        return self.llm_base_url.strip()

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

    def build_simple_llm_config(self) -> dict[str, Any]:
        return {
            "default": self._build_client_entry(
                base_url=self.llm_base_url.strip() or "https://87ba3.gpu.act.buaa.edu.cn/v1",
                model=self.llm_model_name.strip() or "qwen3.5",
                temperature=self.llm_temperature,
                max_tokens=self.llm_max_tokens,
                timeout=self.llm_timeout,
                api_key_env=self.llm_api_key_env.strip(),
            )
        }

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
        fast_timeout = max(self.llm_timeout, 45)
        strong_timeout = max(self.llm_timeout, 90)
        planner_timeout = max(self.llm_timeout, 120)

        return {
            "default": self._build_client_entry(
                base_url=base_url,
                model="qwen3.6-plus",
                temperature=0.1,
                max_tokens=max(self.llm_max_tokens, 4096),
                timeout=strong_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "coordinator": self._build_client_entry(
                base_url=base_url,
                model="qwen3.6-plus",
                temperature=0.05,
                max_tokens=max(self.llm_max_tokens, 4096),
                timeout=strong_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "planner": self._build_client_entry(
                base_url=base_url,
                model="qwen3.5-122b-a10b",
                temperature=0.05,
                max_tokens=max(self.llm_max_tokens, 6144),
                timeout=planner_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "explorer": self._build_client_entry(
                base_url=base_url,
                model="qwen3.5-flash",
                temperature=0.1,
                max_tokens=min(max(self.llm_max_tokens, 2048), 4096),
                timeout=fast_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "executor": self._build_client_entry(
                base_url=base_url,
                model="qwen3.6-plus",
                temperature=0.05,
                max_tokens=max(self.llm_max_tokens, 4096),
                timeout=strong_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "verifier": self._build_client_entry(
                base_url=base_url,
                model="qwen3.5-122b-a10b",
                temperature=0.0,
                max_tokens=min(max(self.llm_max_tokens, 2048), 4096),
                timeout=strong_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "guide": self._build_client_entry(
                base_url=base_url,
                model="qwen3.5-flash",
                temperature=0.2,
                max_tokens=min(max(self.llm_max_tokens, 2048), 4096),
                timeout=fast_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            ),
            "general": self._build_client_entry(
                base_url=base_url,
                model="qwen3.5-flash",
                temperature=0.15,
                max_tokens=min(max(self.llm_max_tokens, 2048), 4096),
                timeout=fast_timeout,
                api_key_env="DASHSCOPE_API_KEY",
            )
        }

    def load_llm_clients_config(self) -> dict[str, Any]:
        raw = self._load_raw_llm_clients_config()
        mode = self.resolved_llm_config_mode()

        if mode == "simple":
            return self.build_simple_llm_config()

        if mode == "preset":
            preset_name = self.llm_clients_preset.strip()
            if not preset_name:
                raise ValueError("CRATER_AGENT_LLM_CONFIG_MODE=preset requires CRATER_AGENT_LLM_CLIENTS_PRESET")
            config = self._build_preset_config(preset_name)
            if not config:
                raise ValueError(f"Unsupported CRATER_AGENT_LLM_CLIENTS_PRESET: {preset_name}")
            return config

        if not raw:
            raise ValueError(
                "CRATER_AGENT_LLM_CONFIG_MODE=map requires CRATER_AGENT_LLM_CLIENTS_CONFIG_JSON or "
                "CRATER_AGENT_LLM_CLIENTS_CONFIG_PATH"
            )
        loaded = json.loads(raw)
        if not isinstance(loaded, dict):
            raise ValueError("CRATER_AGENT_LLM_CLIENTS_CONFIG must decode to a JSON object")
        return loaded

    def resolved_llm_config_source(self) -> str:
        mode = self.resolved_llm_config_mode()
        if mode == "map":
            if self.llm_clients_config_json.strip():
                return "llm_client_map_json"
            return f"llm_client_map_path:{self.llm_clients_config_path.strip()}"
        if mode == "preset":
            return f"preset:{self.llm_clients_preset.strip()}"
        return "simple_env"

    def public_agent_config_summary(self) -> dict[str, Any]:
        return {
            "defaultOrchestrationMode": self.normalized_default_orchestration_mode(),
            "availableModes": ["single_agent", "multi_agent"],
            "llmConfigMode": self.normalized_llm_config_mode(),
            "llmResolvedConfigMode": self.resolved_llm_config_mode(),
            "llmConfigSource": self.resolved_llm_config_source(),
        }


settings = Settings()
