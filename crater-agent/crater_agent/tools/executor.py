"""Tool executor that calls the Crater Go backend.

All tool executions are proxied through Go's /v1/agent/tools/execute endpoint,
which handles permission checks, data access, and audit logging.

For benchmarking, a MockToolExecutor is provided that returns pre-recorded responses.
"""

from __future__ import annotations

import json
import time
from typing import Any, Protocol

import httpx

from crater_agent.config import settings
from crater_agent.tools.definitions import CONFIRM_TOOL_NAMES, is_tool_allowed_for_role


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
    ) -> dict[str, Any]:
        """Execute a tool via Go backend HTTP call.

        Returns:
            For auto tools: {"status": "success", "result": {...}}
            For confirm tools: {"status": "confirmation_required", "confirm_id": "...", ...}
            On error: {"status": "error", "message": "..."}
        """
        start_time = time.monotonic()
        normalized_role = (agent_role or "single_agent").strip().lower() or "single_agent"
        if not is_tool_allowed_for_role(normalized_role, tool_name):
            return {
                "status": "error",
                "error_type": "tool_policy",
                "retryable": False,
                "message": f"Tool {tool_name} is not allowed for agent role {normalized_role}",
                "_latency_ms": int((time.monotonic() - start_time) * 1000),
            }
        try:
            resp = await self.client.post(
                "/api/agent/tools/execute",
                headers={
                    "X-Agent-Internal-Token": settings.crater_backend_internal_token,
                },
                json={
                    "tool_name": tool_name,
                    "tool_args": tool_args,
                    "session_id": session_id,
                    "turn_id": turn_id,
                    "tool_call_id": tool_call_id,
                    "agent_id": agent_id,
                    "agent_role": normalized_role,
                },
            )
            resp.raise_for_status()
            payload = resp.json()
            result = payload.get("data", payload)
            if not isinstance(result, dict):
                result = {"status": "success", "result": result}
            result["_latency_ms"] = int((time.monotonic() - start_time) * 1000)
            return result
        except httpx.TimeoutException:
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


class MockToolExecutor:
    """Mock executor for benchmarking. Returns pre-recorded tool responses."""

    def __init__(self, snapshots: dict[str, Any]):
        """
        Args:
            snapshots: mapping from tool_name to pre-recorded response data.
                       Supports arg-based lookup: {"tool_name": response} or
                       {"tool_name": {arg_key: {arg_val: response}}}
        """
        self.snapshots = snapshots
        self.call_log: list[dict[str, Any]] = []

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
    ) -> dict[str, Any]:
        start_time = time.monotonic()

        # Record the call for evaluation
        call_record = {
            "tool_name": tool_name,
            "tool_args": tool_args,
            "timestamp": time.time(),
        }

        # Check if this is a confirm tool
        if tool_name in CONFIRM_TOOL_NAMES:
            result = {
                "tool_call_id": tool_call_id,
                "status": "confirmation_required",
                "confirmation": {
                    "confirm_id": f"mock_cf_{len(self.call_log)}",
                    "tool_name": tool_name,
                    "description": f"模拟确认请求: {tool_name}({tool_args})",
                    "risk_level": "high",
                },
            }
        elif tool_name in self.snapshots:
            snapshot = self.snapshots[tool_name]
            result = {"tool_call_id": tool_call_id, "status": "success", "result": snapshot}
        else:
            result = {
                "tool_call_id": tool_call_id,
                "status": "error",
                "message": f"Mock: no snapshot for tool '{tool_name}'",
            }

        result["_latency_ms"] = int((time.monotonic() - start_time) * 1000)
        call_record["result"] = result
        self.call_log.append(call_record)
        return result
