from __future__ import annotations

import os
from typing import Any

from fastapi import APIRouter, Header, HTTPException
from pydantic import BaseModel, Field

from crater_agent.config import settings
from crater_agent.tools.local_executor import LocalToolExecutor

internal_tools_router = APIRouter()
_local_write_executor = LocalToolExecutor()


def _expected_internal_token() -> str:
    explicit = str(os.getenv("CRATER_AGENT_AGENT_INTERNAL_TOKEN") or "").strip()
    if explicit:
        return explicit
    shared = str(os.getenv("CRATER_AGENT_INTERNAL_TOKEN") or "").strip()
    if shared:
        return shared
    backend_token = str(settings.crater_backend_internal_token or "").strip()
    if backend_token:
        return backend_token
    return str(settings.agent_internal_token or "").strip()


def _verify_internal_token(header_value: str) -> None:
    expected = _expected_internal_token()
    if not expected or header_value != expected:
        raise HTTPException(status_code=403, detail="Invalid internal token")


class LocalWriteExecutionRequest(BaseModel):
    tool_name: str = Field(..., alias="tool_name")
    tool_args: dict[str, Any] = Field(default_factory=dict, alias="tool_args")
    session_id: str = Field(default="", alias="session_id")
    turn_id: str | None = Field(default=None, alias="turn_id")
    tool_call_id: str | None = Field(default=None, alias="tool_call_id")
    confirm_id: str | None = Field(default=None, alias="confirm_id")
    agent_id: str | None = Field(default=None, alias="agent_id")
    agent_role: str | None = Field(default=None, alias="agent_role")
    actor_role: str | None = Field(default=None, alias="actor_role")
    user_id: int = 0
    execution_backend: str = Field(default="python_local", alias="execution_backend")

    model_config = {"populate_by_name": True}


@internal_tools_router.get("/tools/catalog")
async def get_local_tool_catalog(
    x_agent_internal_token: str = Header(..., alias="X-Agent-Internal-Token"),
) -> dict[str, Any]:
    _verify_internal_token(x_agent_internal_token)
    return {"tools": _local_write_executor.build_tool_catalog()}


@internal_tools_router.post("/tools/execute-local-write")
async def execute_local_write(
    request: LocalWriteExecutionRequest,
    x_agent_internal_token: str = Header(..., alias="X-Agent-Internal-Token"),
) -> dict[str, Any]:
    _verify_internal_token(x_agent_internal_token)
    return await _local_write_executor.execute(
        tool_name=request.tool_name,
        tool_args=dict(request.tool_args or {}),
        session_id=request.session_id,
        user_id=int(request.user_id or 0),
        turn_id=request.turn_id,
        tool_call_id=request.tool_call_id,
        agent_id=request.agent_id,
        agent_role=request.agent_role,
        actor_role=request.actor_role,
        execution_backend=request.execution_backend,
        confirmed_dispatch=True,
        confirm_id=request.confirm_id,
    )
