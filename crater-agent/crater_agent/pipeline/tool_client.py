"""Shared backend tool client for internal pipeline runs."""

from __future__ import annotations

import logging
from typing import Any

import httpx

from crater_agent.config import settings

logger = logging.getLogger(__name__)

PIPELINE_SESSION_ID = "00000000-0000-0000-0000-000000000000"
PIPELINE_INTERNAL_CONTEXT = {
    "role": "admin",
    "username": "agent-pipeline",
    "account_name": "system",
}


class PipelineToolClient:
    def __init__(self, *, timeout: int = 60):
        self._client = httpx.AsyncClient(timeout=timeout)

    async def __aenter__(self) -> "PipelineToolClient":
        return self

    async def __aexit__(self, exc_type, exc, tb) -> None:
        await self.close()

    async def close(self) -> None:
        await self._client.aclose()

    async def execute(self, tool_name: str, tool_args: dict[str, Any]) -> dict[str, Any]:
        try:
            response = await self._client.post(
                f"{settings.crater_backend_url}/api/agent/tools/execute",
                json={
                    "tool_name": tool_name,
                    "tool_args": tool_args,
                    "session_id": PIPELINE_SESSION_ID,
                    "turn_id": "pipeline",
                    "tool_call_id": f"pipeline-{tool_name}",
                    "agent_id": "pipeline",
                    "agent_role": "single_agent",
                    "internal_context": PIPELINE_INTERNAL_CONTEXT,
                },
                headers={
                    "X-Agent-Internal-Token": settings.crater_backend_internal_token,
                    "Content-Type": "application/json",
                },
            )
            if response.status_code == 200:
                payload = response.json()
                return payload.get("data", payload)
            return {"status": "error", "message": f"HTTP {response.status_code}"}
        except Exception as exc:
            logger.error("Pipeline tool execution failed: %s - %s", tool_name, exc)
            return {"status": "error", "message": str(exc)}
