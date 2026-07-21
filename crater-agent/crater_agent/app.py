"""FastAPI application for Crater Agent Service.

Exposes:
- POST /chat  — accepts user message + context, returns SSE stream of agent events
- GET /health — health check
"""

from __future__ import annotations

import json
import logging
from typing import Any

# Configure logging before anything else — without this, logger.info() in agent
# modules is silently swallowed because the root logger defaults to WARNING.
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    datefmt="%H:%M:%S",
)

from fastapi import FastAPI

logger = logging.getLogger(__name__)
from pydantic import BaseModel, Field
from sse_starlette.sse import EventSourceResponse

from crater_agent.config import settings
from crater_agent.llm.client import (
    ModelClientFactory,
    local_llm_config_fallback_enabled,
    normalize_runtime_llm_client_configs,
    reset_runtime_llm_client_configs,
    set_runtime_llm_client_configs,
)
from crater_agent.orchestrators.single import SingleAgentOrchestrator

app = FastAPI(title="Crater Agent Service", version="0.1.0")

logger.info(
    "agent service starting: backend=%s",
    settings.crater_backend_url,
)

single_orchestrator = SingleAgentOrchestrator()


class ChatRequest(BaseModel):
    """Request from Go backend."""

    session_id: str
    message: str
    turn_id: str | None = None
    context: dict[str, Any] = Field(default_factory=dict)
    user_id: int | None = None
    account_id: int | None = None
    username: str | None = None
    page_context: dict[str, Any] | None = None


def build_request_context(request: ChatRequest) -> dict[str, Any]:
    context = dict(request.context or {})
    actor = dict(context.get("actor") or {})
    if request.user_id is not None and "user_id" not in actor:
        actor["user_id"] = request.user_id
    if request.account_id is not None and "account_id" not in actor:
        actor["account_id"] = request.account_id
    if request.username is not None and "username" not in actor:
        actor["username"] = request.username
    if actor:
        context["actor"] = actor
    if request.page_context and "page" not in context:
        context["page"] = request.page_context
    return context


def get_orchestration_mode(context: dict[str, Any]) -> str:
    return "single_agent"


def get_request_model_factory(context: dict[str, Any]) -> ModelClientFactory:
    runtime_clients = get_request_llm_client_configs(context)
    if runtime_clients:
        return ModelClientFactory(raw_clients=runtime_clients)
    return ModelClientFactory()


def get_request_llm_client_configs(context: dict[str, Any]) -> dict[str, Any] | None:
    llm_context = context.get("llm") if isinstance(context, dict) else {}
    if isinstance(llm_context, dict):
        raw_clients = llm_context.get("client_config")
        return normalize_runtime_llm_client_configs(raw_clients)
    return None


@app.get("/health")
async def health():
    default_model = ""
    if local_llm_config_fallback_enabled():
        try:
            default_model = str(settings.get_llm_client_config("default").get("model") or "")
        except Exception:
            pass
        try:
            default_model = ModelClientFactory().create("default").model_name
        except Exception:
            pass
    return {"status": "ok", "model": default_model}


@app.get("/config-summary")
async def config_summary():
    if not local_llm_config_fallback_enabled():
        return {
            "defaultOrchestrationMode": settings.normalized_default_orchestration_mode(),
            "availableModes": ["single_agent"],
            "localLLMConfigFallbackEnabled": False,
        }
    return settings.public_agent_config_summary()


@app.post("/chat")
async def chat(request: ChatRequest):
    """Process a chat message and return SSE stream.

    The Go backend calls this endpoint with the user message and context.
    Returns Server-Sent Events with thinking, tool_call, tool_result, message events.
    """
    async def event_generator():
        request_context = build_request_context(request)
        request.context = request_context
        orchestration_mode = get_orchestration_mode(request_context)
        runtime_llm_clients = get_request_llm_client_configs(request_context)

        runtime_token = set_runtime_llm_client_configs(runtime_llm_clients)
        try:
            model_factory = get_request_model_factory(request_context)
            orchestrator = single_orchestrator
            async for event in orchestrator.stream(request=request, model_factory=model_factory):
                yield {
                    "event": event["event"],
                    "data": json.dumps(event.get("data", {}), ensure_ascii=False),
                }
        except Exception as e:
            logger.exception("Orchestrator stream error")
            yield {
                "event": "error",
                "data": json.dumps(
                    {"code": "agent_error", "message": str(e)}, ensure_ascii=False
                ),
            }
            yield {
                "event": "done",
                "data": json.dumps({}, ensure_ascii=False),
            }
        finally:
            reset_runtime_llm_client_configs(runtime_token)

    return EventSourceResponse(event_generator(), ping=0)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host=settings.host, port=settings.port)
