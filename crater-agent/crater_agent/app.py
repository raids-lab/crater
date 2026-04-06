"""FastAPI application for Crater Agent Service.

Exposes:
- POST /chat  — accepts user message + context, returns SSE stream of agent events
- GET /health — health check
"""

from __future__ import annotations

import json
import logging
from typing import Any

from fastapi import FastAPI

logger = logging.getLogger(__name__)
from pydantic import BaseModel, Field
from sse_starlette.sse import EventSourceResponse

from crater_agent.config import settings
from crater_agent.llm.client import ModelClientFactory
from crater_agent.orchestrators.multi import MultiAgentOrchestrator
from crater_agent.orchestrators.single import SingleAgentOrchestrator
from crater_agent.pipeline.router import pipeline_router

app = FastAPI(title="Crater Agent Service", version="0.1.0")
app.include_router(pipeline_router, prefix="/pipeline", tags=["pipeline"])

single_orchestrator = SingleAgentOrchestrator()
multi_orchestrator = MultiAgentOrchestrator()


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
    orchestration = context.get("orchestration") or {}
    mode = orchestration.get("mode") or settings.normalized_default_orchestration_mode()
    if mode == "multi_agent":
        return "multi_agent"
    return "single_agent"


@app.get("/health")
async def health():
    default_model = ""
    try:
        default_model = str(settings.get_llm_client_config("default").get("model") or "")
    except Exception:
        pass
    try:
        default_model = ModelClientFactory().create(
            purpose="default", orchestration_mode="single_agent"
        ).model_name
    except Exception:
        pass
    return {"status": "ok", "model": default_model}


@app.get("/config-summary")
async def config_summary():
    return settings.public_agent_config_summary()


@app.post("/chat")
async def chat(request: ChatRequest):
    """Process a chat message and return SSE stream.

    The Go backend calls this endpoint with the user message and context.
    Returns Server-Sent Events with thinking, tool_call, tool_result, message events.
    """
    async def event_generator():
        request_context = build_request_context(request)
        orchestration_mode = get_orchestration_mode(request_context)
        model_factory = ModelClientFactory()
        orchestrator = (
            multi_orchestrator if orchestration_mode == "multi_agent" else single_orchestrator
        )

        try:
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

    return EventSourceResponse(event_generator())


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host=settings.host, port=settings.port)
