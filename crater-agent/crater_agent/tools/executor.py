"""Tool executor that calls the Crater Go backend.

All tool executions are proxied through Go's /v1/agent/tools/execute endpoint,
which handles permission checks, data access, and audit logging.

For benchmarking, a MockToolExecutor is provided that returns pre-recorded responses.
"""

from __future__ import annotations

import json
import logging
import time
from typing import Any, Protocol

logger = logging.getLogger(__name__)

import httpx

from crater_agent.config import settings
from crater_agent.runtime.platform import route_for_tool
from crater_agent.tools.definitions import (
    CONFIRM_TOOL_NAMES,
    READ_ONLY_TOOL_NAMES,
    is_actor_allowed_for_tool,
    is_tool_allowed_for_role,
)
from crater_agent.tools.local_executor import LocalToolExecutor


class ToolExecutorProtocol(Protocol):
    """Protocol for tool executors (real and mock)."""

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
                "message": f"Tool {tool_name} requires admin privileges",
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


class LocalCoreToolExecutor(LocalToolExecutor):
    """Backward-compatible alias for the agent-local core tool executor.

    Older code referenced LocalCoreToolExecutor; the implementation now lives in
    crater_agent.tools.local_executor.LocalToolExecutor.
    """


class CompositeToolExecutor:
    """Routes tool executions between local core tools and backend delegation."""

    def __init__(
        self,
        *,
        local: LocalToolExecutor | None = None,
        backend: ToolExecutorProtocol | None = None,
        backend_url: str | None = None,
    ):
        self.local = local or LocalToolExecutor()
        self.backend: ToolExecutorProtocol = backend or GoBackendToolExecutor(backend_url=backend_url)
        self._runtime = self.local.runtime

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
        del execution_backend
        default_target = (
            "local"
            if (
                self.local.supports(tool_name)
                and self._runtime.default_route_for_tool(tool_name) == "local"
            )
            else "backend"
        )
        target = route_for_tool(self._runtime, tool_name, default=default_target)
        if target == "local":
            local_route_allowed = self.local.supports(tool_name)
            if tool_name in CONFIRM_TOOL_NAMES:
                local_route_allowed = local_route_allowed and self._runtime.is_local_write_tool(tool_name)
            else:
                local_route_allowed = local_route_allowed and self._runtime.is_local_core_tool(tool_name)
            if not local_route_allowed:
                logger.warning(
                    "[tool] route override ignored for %s: local execution is not enabled for this tool",
                    tool_name,
                )
                target = "backend"
        if tool_name in CONFIRM_TOOL_NAMES:
            execution_backend_name = "python_local" if target == "local" else "backend"
            logger.info(
                "[tool] route: %s target=%s execution_backend=%s confirm=true",
                tool_name,
                target,
                execution_backend_name,
            )
            return await self.backend.execute(
                tool_name=tool_name,
                tool_args=tool_args,
                session_id=session_id,
                user_id=user_id,
                turn_id=turn_id,
                tool_call_id=tool_call_id,
                agent_id=agent_id,
                agent_role=agent_role,
                actor_role=actor_role,
                execution_backend=execution_backend_name,
            )
        if target == "local":
            logger.info("[tool] route: %s target=local execution_backend=python_local confirm=false", tool_name)
            return await self.local.execute(
                tool_name=tool_name,
                tool_args=tool_args,
                session_id=session_id,
                user_id=user_id,
                turn_id=turn_id,
                tool_call_id=tool_call_id,
                agent_id=agent_id,
                agent_role=agent_role,
                actor_role=actor_role,
                execution_backend="python_local",
            )
        logger.info("[tool] route: %s target=backend execution_backend=%s confirm=false", tool_name, "backend")
        return await self.backend.execute(
            tool_name=tool_name,
            tool_args=tool_args,
            session_id=session_id,
            user_id=user_id,
            turn_id=turn_id,
            tool_call_id=tool_call_id,
            agent_id=agent_id,
            agent_role=agent_role,
            actor_role=actor_role,
            execution_backend="backend" if tool_name in CONFIRM_TOOL_NAMES else None,
        )

    async def close(self) -> None:
        local_close = getattr(self.local, "close", None)
        if callable(local_close):
            await local_close()
        close = getattr(self.backend, "close", None)
        if callable(close):
            await close()


class ReadOnlyToolExecutor:
    """Proxy executor that denies any non-read-only tools.

    Intended for eval harness live-readonly mode to prevent accidental mutations.
    """

    def __init__(self, inner: ToolExecutorProtocol, *, allowed_tools: set[str] | None = None):
        self.inner = inner
        self.allowed_tools = allowed_tools or set(READ_ONLY_TOOL_NAMES)

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
        del execution_backend
        start_time = time.monotonic()
        if tool_name in CONFIRM_TOOL_NAMES:
            return {
                "tool_call_id": tool_call_id,
                "status": "confirmation_required",
                "confirmation": {
                    "confirm_id": f"eval_ro_{tool_name}_{tool_call_id or 'pending'}",
                    "tool_name": tool_name,
                    "description": (
                        f"live-readonly evaluation blocks write tool '{tool_name}' "
                        f"with args {tool_args}"
                    ),
                    "interaction": "approval",
                    "risk_level": "high",
                },
                "_latency_ms": int((time.monotonic() - start_time) * 1000),
            }
        if tool_name not in self.allowed_tools:
            return {
                "status": "error",
                "error_type": "tool_policy",
                "retryable": False,
                "message": f"Tool {tool_name} is not allowed in live-readonly mode",
                "_latency_ms": int((time.monotonic() - start_time) * 1000),
            }
        return await self.inner.execute(
            tool_name=tool_name,
            tool_args=tool_args,
            session_id=session_id,
            user_id=user_id,
            turn_id=turn_id,
            tool_call_id=tool_call_id,
            agent_id=agent_id,
            agent_role=agent_role,
            actor_role=actor_role,
        )

    async def close(self) -> None:
        close = getattr(self.inner, "close", None)
        if callable(close):
            await close()


# Backward-compatible import path for older tests and debug scripts. The
# benchmark implementation lives under crater_agent.eval to keep mock-only
# behavior out of production executors.
from crater_agent.eval.mock_executor import MockToolExecutor  # noqa: E402
