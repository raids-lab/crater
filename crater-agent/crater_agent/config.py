"""Configuration for Crater Agent Service."""

import json
import os
from pathlib import Path
from typing import Any

from pydantic import Field
from pydantic_settings import BaseSettings


def _resolve_profile_relative_path(raw_path: str, *, base_dir: Path) -> Path:
    candidate = Path(raw_path).expanduser()
    if candidate.is_absolute():
        return candidate
    return (base_dir / candidate).resolve()


class Settings(BaseSettings):
    """Agent service configuration, loaded from environment variables."""

    default_orchestration_mode: str = Field(
        default="single_agent",
        description="Default orchestration mode for agent chat",
    )

    # Single source of truth for LLM routing.
    # This file contains the direct purpose/role -> client config map.
    llm_clients_config_path: str = Field(
        default="./config/llm-clients.json",
        description="Path to the LLM client map JSON file",
    )

    # MAS runtime guardrails (optional, has sensible defaults).
    agent_runtime_config_path: str = Field(
        default="./config/agent-runtime.json",
        description="Path to the MAS runtime config JSON file",
    )
    agent_profile_path: str = Field(
        default="",
        description="Optional path to an agent-owned JSON profile for standalone/offline deployment",
    )
    platform_runtime_config_path: str = Field(
        default="",
        description="Optional path to an agent-side platform/runtime override config (prefer YAML)",
    )
    backend_debug_config_path: str = Field(
        default="",
        description="Optional path to backend debug config YAML for live platform discovery",
    )

    # Crater Go Backend
    crater_backend_url: str = Field(
        default="http://localhost:8080", description="Crater Go backend URL"
    )
    crater_backend_internal_token: str = Field(
        default="", description="Shared token for Python Agent -> Go internal tool execution"
    )
    agent_internal_token: str = Field(
        default="dev-agent-internal-token",
        description="Internal token for Go backend authentication",
    )

    # Agent Behavior
    max_tool_calls_per_turn: int = Field(
        default=15, description="Max tool calls in a single ReAct loop"
    )
    tool_execution_timeout: int = Field(default=30, description="Tool execution timeout (seconds)")
    history_max_tokens: int = Field(
        default=4000, description="Max tokens for conversation history"
    )
    max_context_tokens: int = Field(
        default=30000, description="Estimated LLM context window budget for proactive compaction"
    )
    tokenizer_encoding: str = Field(
        default="cl100k_base", description="tiktoken encoding name for token counting"
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

    def model_post_init(self, __context: Any) -> None:
        del __context
        self._apply_agent_profile_overrides()

    def normalized_default_orchestration_mode(self) -> str:
        return (
            "multi_agent"
            if self.default_orchestration_mode.strip().lower() == "multi_agent"
            else "single_agent"
        )

    def resolve_llm_clients_config_path(self) -> Path:
        configured = self.llm_clients_config_path.strip() or "./config/llm-clients.json"
        return self._resolve_config_path(configured)

    def resolve_agent_runtime_config_path(self) -> Path:
        configured = self.agent_runtime_config_path.strip() or "./config/agent-runtime.json"
        return self._resolve_config_path(configured)

    def resolve_agent_profile_path(self) -> Path | None:
        configured = self.agent_profile_path.strip()
        if not configured:
            configured = str(os.getenv("CRATER_AGENT_PROFILE_PATH") or "").strip()
        if configured:
            return self._resolve_config_path(configured)

        default_path = self._resolve_config_path("./config/agent-profile.json")
        if default_path.exists():
            return default_path
        return None

    def resolve_platform_runtime_config_path(self) -> Path | None:
        configured = self.platform_runtime_config_path.strip()
        if configured:
            return self._resolve_config_path(configured)

        for default_path in (
            "./config/platform-runtime.yaml",
            "./config/platform-runtime.yml",
        ):
            resolved = self._resolve_config_path(default_path)
            if resolved.exists():
                return resolved
        return None

    def resolve_backend_debug_config_path(self) -> Path | None:
        configured = self.backend_debug_config_path.strip()
        if not configured:
            return None
        return self._resolve_config_path(configured)

    def _resolve_config_path(self, configured: str) -> Path:
        raw_path = Path(configured).expanduser()
        if raw_path.is_absolute():
            return raw_path

        cwd_candidate = Path.cwd() / raw_path
        if cwd_candidate.exists():
            return cwd_candidate

        project_root = Path(__file__).resolve().parents[1]
        return project_root / raw_path

    def load_llm_client_configs(self) -> dict[str, dict[str, Any]]:
        path = self.resolve_llm_clients_config_path()
        if not path.exists():
            raise ValueError(f"LLM client config file not found: {path}")

        raw = path.read_text(encoding="utf-8").strip()
        if not raw:
            raise ValueError(f"LLM client config file is empty: {path}")

        loaded = json.loads(raw)
        if not isinstance(loaded, dict):
            raise ValueError("LLM client config must decode to a JSON object")

        configs = {
            str(name): dict(config)
            for name, config in loaded.items()
            if isinstance(name, str) and isinstance(config, dict)
        }
        if "default" not in configs:
            raise ValueError("LLM client config must define a 'default' client")
        return configs

    def get_llm_client_config(self, client_key: str = "default") -> dict[str, Any]:
        configs = self.load_llm_client_configs()
        normalized = client_key.strip()
        if normalized and normalized in configs:
            return dict(configs[normalized])
        return dict(configs["default"])

    def load_agent_runtime_config(self) -> dict[str, Any]:
        """Load MAS runtime config. Returns empty dict if file missing (defaults used)."""
        raw_path = self.resolve_agent_runtime_config_path()
        if not raw_path.exists():
            return {}
        try:
            raw = raw_path.read_text(encoding="utf-8").strip()
            loaded = json.loads(raw) if raw else {}
            return dict(loaded) if isinstance(loaded, dict) else {}
        except Exception:
            return {}

    def _load_agent_profile(self) -> dict[str, Any]:
        path = self.resolve_agent_profile_path()
        if path is None or not path.exists():
            return {}
        try:
            raw = path.read_text(encoding="utf-8").strip()
            loaded = json.loads(raw) if raw else {}
            return dict(loaded) if isinstance(loaded, dict) else {}
        except Exception:
            return {}

    def _apply_agent_profile_overrides(self) -> None:
        profile = self._load_agent_profile()
        if not profile:
            return
        profile_path = self.resolve_agent_profile_path()
        profile_base = profile_path.parent if profile_path is not None else Path.cwd()

        override_specs = {
            "defaultOrchestrationMode": (
                "default_orchestration_mode",
                "CRATER_AGENT_DEFAULT_ORCHESTRATION_MODE",
            ),
            "llmClientsConfigPath": (
                "llm_clients_config_path",
                "CRATER_AGENT_LLM_CLIENTS_CONFIG_PATH",
            ),
            "agentRuntimeConfigPath": (
                "agent_runtime_config_path",
                "CRATER_AGENT_AGENT_RUNTIME_CONFIG_PATH",
            ),
            "platformRuntimeConfigPath": (
                "platform_runtime_config_path",
                "CRATER_AGENT_PLATFORM_RUNTIME_CONFIG_PATH",
            ),
            "backendDebugConfigPath": (
                "backend_debug_config_path",
                "CRATER_AGENT_BACKEND_DEBUG_CONFIG_PATH",
            ),
            "craterBackendUrl": (
                "crater_backend_url",
                "CRATER_AGENT_CRATER_BACKEND_URL",
            ),
            "craterBackendInternalToken": (
                "crater_backend_internal_token",
                "CRATER_AGENT_CRATER_BACKEND_INTERNAL_TOKEN",
            ),
            "host": ("host", "CRATER_AGENT_HOST"),
            "port": ("port", "CRATER_AGENT_PORT"),
            "debug": ("debug", "CRATER_AGENT_DEBUG"),
        }
        path_like_fields = {
            "llm_clients_config_path",
            "agent_runtime_config_path",
            "platform_runtime_config_path",
            "backend_debug_config_path",
        }
        for profile_key, (field_name, env_name) in override_specs.items():
            if os.getenv(env_name) not in (None, ""):
                continue
            value = profile.get(profile_key)
            if value in (None, ""):
                continue
            if field_name in path_like_fields:
                value = str(_resolve_profile_relative_path(str(value), base_dir=profile_base))
            setattr(self, field_name, value)

    def public_agent_config_summary(self) -> dict[str, Any]:
        configs = self.load_llm_client_configs()
        return {
            "defaultOrchestrationMode": self.normalized_default_orchestration_mode(),
            "availableModes": ["single_agent", "multi_agent"],
            "agentProfilePath": (
                str(self.resolve_agent_profile_path())
                if self.resolve_agent_profile_path() is not None
                else ""
            ),
            "llmConfigPath": str(self.resolve_llm_clients_config_path()),
            "agentRuntimeConfigPath": str(self.resolve_agent_runtime_config_path()),
            "platformRuntimeConfigPath": (
                str(self.resolve_platform_runtime_config_path())
                if self.resolve_platform_runtime_config_path() is not None
                else ""
            ),
            "backendDebugConfigPath": (
                str(self.resolve_backend_debug_config_path())
                if self.resolve_backend_debug_config_path() is not None
                else ""
            ),
            "llmClientKeys": list(configs.keys()),
            "llmClients": {
                name: {
                    "baseUrl": str(config.get("base_url") or ""),
                    "model": str(config.get("model") or ""),
                    "temperature": float(config.get("temperature") or 0.0),
                    "maxTokens": int(config.get("max_tokens") or 0),
                    "timeout": int(config.get("timeout") or 0),
                    "apiKeyEnv": str(config.get("api_key_env") or ""),
                    "hasInlineApiKey": bool(str(config.get("api_key") or "").strip()),
                }
                for name, config in configs.items()
            },
        }

settings = Settings()
