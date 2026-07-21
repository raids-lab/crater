"""Tool executor that calls the Crater Go backend.

All tool executions are proxied through Go's /v1/agent/tools/execute endpoint,
which handles permission checks, data access, and audit logging.
"""

from __future__ import annotations

import json
import logging
import time
from typing import Any, Protocol

logger = logging.getLogger(__name__)

import httpx

from crater_agent.config import settings
from crater_agent.tools.definitions import (
    is_actor_allowed_for_tool,
    is_tool_allowed_for_role,
)


def normalize_tool_args_for_backend(tool_name: str, tool_args: dict[str, Any]) -> dict[str, Any]:
    """Normalize provider-specific argument variants before backend dispatch."""
    normalized = dict(tool_args or {})
    if (
        tool_name == "notify_job_owner"
        and "job_names" not in normalized
        and normalized.get("job_name")
    ):
        normalized["job_names"] = normalized["job_name"]
    return normalized


class ToolExecutorProtocol(Protocol):
    """Protocol for production tool executors."""

    async def execute(
        self,
        tool_name: str,
        tool_args: dict[str, Any],
        session_id: str,
        user_id: int,
        turn_id: str | None = None,
        tool_call_id: str | None = None,
        agent_id: str | None = None,
        agent_role: str | None = None,
        actor_role: str | None = None,
        execution_backend: str | None = None,
    ) -> dict[str, Any]: ...


class GoBackendToolExecutor:
    """Executes tools by calling the Crater Go backend."""

    def __init__(self, backend_url: str | None = None):
        self.backend_url = backend_url or settings.crater_backend_url
        self.client = httpx.AsyncClient(
            base_url=self.backend_url,
            timeout=settings.tool_execution_timeout,
        )

    async def execute(
        self,
        tool_name: str,
        tool_args: dict[str, Any],
        session_id: str,
        user_id: int,
        turn_id: str | None = None,
        tool_call_id: str | None = None,
        agent_id: str | None = None,
        agent_role: str | None = None,
        actor_role: str | None = None,
        execution_backend: str | None = None,
    ) -> dict[str, Any]:
        """Execute a tool via Go backend HTTP call.

        Returns:
            For auto tools: {"status": "success", "result": {...}}
            For confirm tools: {"status": "confirmation_required", "confirm_id": "...", ...}
            On error: {"status": "error", "message": "..."}
        """
        start_time = time.monotonic()
        tool_args = normalize_tool_args_for_backend(tool_name, tool_args)
        logger.info("[tool] execute: %s args=%s session=%s actor=%s", tool_name, tool_args, session_id, actor_role)
        normalized_role = (agent_role or "single_agent").strip().lower() or "single_agent"
        if not is_tool_allowed_for_role(normalized_role, tool_name):
            return {
                "status": "error",
                "error_type": "tool_policy",
                "retryable": False,
                "message": f"Tool {tool_name} is not allowed for agent role {normalized_role}",
                "_latency_ms": int((time.monotonic() - start_time) * 1000),
            }
        if not is_actor_allowed_for_tool(actor_role, tool_name):
            return {
                "status": "error",
                "error_type": "tool_policy",
                "retryable": False,
                "message": f"你当前没有管理员权限，不能执行 {tool_name}；如确需处理，请联系平台管理员或切换到管理员页面后再操作。",
                "_latency_ms": int((time.monotonic() - start_time) * 1000),
            }
        try:
            request_body: dict[str, Any] = {
                "tool_name": tool_name,
                "tool_args": tool_args,
                "session_id": session_id,
                "turn_id": turn_id,
                "tool_call_id": tool_call_id,
                "agent_id": agent_id,
                "agent_role": normalized_role,
            }
            if execution_backend:
                request_body["execution_backend"] = execution_backend
            # System-level agents (e.g. approval evaluator) don't have a real
            # AgentSession in the database.  Pass internal_context so the Go
            # backend resolves an admin token directly instead of doing a
            # session lookup that would fail with "session not found".
            if actor_role == "system":
                request_body["internal_context"] = {
                    "role": "admin",
                    "username": "agent-approval",
                }

            resp = await self.client.post(
                "/api/agent/tools/execute",
                headers={
                    "X-Agent-Internal-Token": settings.crater_backend_internal_token,
                },
                json=request_body,
            )
            resp.raise_for_status()
            payload = resp.json()
            result = payload.get("data", payload)
            if not isinstance(result, dict):
                result = {"status": "success", "result": result}
            result["_latency_ms"] = int((time.monotonic() - start_time) * 1000)
            status = result.get("status", "ok")
            if status == "error":
                logger.warning("[tool] %s failed: %s (latency=%dms)", tool_name, result.get("message") or result.get("error", ""), result["_latency_ms"])
            else:
                # Truncate result summary to avoid flooding logs
                brief = str(result.get("result", ""))[:200]
                logger.info("[tool] %s ok: %s (latency=%dms)", tool_name, brief, result["_latency_ms"])
            return result
        except httpx.TimeoutException:
            logger.warning("[tool] %s timed out after %ds", tool_name, settings.tool_execution_timeout)
            return {
                "status": "error",
                "error_type": "timeout",
                "retryable": True,
                "message": f"Tool {tool_name} 执行超时 ({settings.tool_execution_timeout}s)",
                "_latency_ms": int((time.monotonic() - start_time) * 1000),
            }
        except httpx.HTTPStatusError as e:
            detail = ""
            try:
                body = e.response.json()
                if isinstance(body, dict):
                    detail = str(body.get("msg") or body.get("message") or body.get("error") or "")
            except (json.JSONDecodeError, ValueError, TypeError):
                detail = ""

            status_code = e.response.status_code
            if status_code in (401, 403):
                error_type = "auth"
                retryable = False
            elif status_code == 404:
                error_type = "not_found"
                retryable = False
            elif status_code == 429:
                error_type = "rate_limit"
                retryable = True
            elif 500 <= status_code < 600:
                error_type = "server"
                retryable = True
            else:
                error_type = "http"
                retryable = False

            logger.warning("[tool] %s HTTP error: %d %s", tool_name, status_code, detail)
            return {
                "status": "error",
                "error_type": error_type,
                "retryable": retryable,
                "status_code": status_code,
                "message": (
                    f"Tool {tool_name} 执行失败: HTTP {status_code}"
                    + (f" - {detail}" if detail else "")
                ),
                "_latency_ms": int((time.monotonic() - start_time) * 1000),
            }
        except httpx.RequestError as e:
            logger.warning("[tool] %s network error: %s", tool_name, e)
            return {
                "status": "error",
                "error_type": "network",
                "retryable": True,
                "message": f"Tool {tool_name} 网络异常: {e}",
                "_latency_ms": int((time.monotonic() - start_time) * 1000),
            }
        except Exception as e:
            return {
                "status": "error",
                "error_type": "unexpected",
                "retryable": False,
                "message": f"Tool {tool_name} 执行异常: {e}",
                "_latency_ms": int((time.monotonic() - start_time) * 1000),
            }

    async def close(self):
        await self.client.aclose()


class CompositeToolExecutor(GoBackendToolExecutor):
    """Compatibility alias: all tool execution now goes through Go backend."""
