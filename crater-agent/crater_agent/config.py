"""Configuration for Crater Agent Service."""

import json
from pathlib import Path
from typing import Any

from pydantic import Field
from pydantic_settings import BaseSettings


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

    def resolve_llm_clients_config_path(self) -> Path:
        configured = self.llm_clients_config_path.strip() or "./config/llm-clients.json"
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

    def public_agent_config_summary(self) -> dict[str, Any]:
        configs = self.load_llm_client_configs()
        return {
            "defaultOrchestrationMode": self.normalized_default_orchestration_mode(),
            "availableModes": ["single_agent", "multi_agent"],
            "llmConfigPath": str(self.resolve_llm_clients_config_path()),
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
