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

from fastapi import FastAPI, HTTPException

logger = logging.getLogger(__name__)
from pydantic import BaseModel, Field
from sse_starlette.sse import EventSourceResponse

from crater_agent.agents.approval import ApprovalAgent, ApprovalEvalRequest, ApprovalEvalResponse
from crater_agent.app_quality import router as quality_router
from crater_agent.config import settings
from crater_agent.internal_tools_router import internal_tools_router
from crater_agent.llm.client import ModelClientFactory
from crater_agent.orchestrators.multi import MultiAgentOrchestrator
from crater_agent.orchestrators.single import SingleAgentOrchestrator
from crater_agent.pipeline.router import pipeline_router

app = FastAPI(title="Crater Agent Service", version="0.1.0")
app.include_router(pipeline_router, prefix="/pipeline", tags=["pipeline"])
app.include_router(quality_router)
app.include_router(internal_tools_router, prefix="/internal", tags=["internal-tools"])

logger.info(
    "agent service starting: backend=%s eval_artifacts=%s eval_artifact_dir=%s",
    settings.crater_backend_url,
    settings.quality_eval_write_artifacts,
    settings.resolve_quality_eval_output_dir(),
)

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
        default_model = ModelClientFactory().create("default").model_name
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
        request.context = request_context
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

    return EventSourceResponse(event_generator(), ping=0)


# ---------------------------------------------------------------------------
# Approval evaluation endpoint (synchronous, non-streaming)
# ---------------------------------------------------------------------------
import asyncio

_approval_semaphore = asyncio.Semaphore(3)
_approval_agent: ApprovalAgent | None = None


def _get_approval_agent() -> ApprovalAgent:
    global _approval_agent
    if _approval_agent is None:
        _approval_agent = ApprovalAgent()
    return _approval_agent


@app.post("/evaluate/approval", response_model=ApprovalEvalResponse)
async def evaluate_approval(request: ApprovalEvalRequest):
    """Evaluate a job lock approval order.

    Called by Go backend as a synchronous hook during order creation.
    Returns structured verdict (approve/escalate). Never returns 5xx
    for evaluation logic failures — those are wrapped in an escalate verdict.
    """
    logger.info(
        "[approval] received evaluation request: order=%d job=%s hours=%d user=%s session=%s",
        request.order_id, request.job_name, request.extension_hours,
        request.username, request.session_id,
    )
    if not _approval_semaphore.locked() and _approval_semaphore._value <= 0:
        logger.warning("[approval] rejected: semaphore full (429)")
        raise HTTPException(status_code=429, detail="approval evaluation busy")

    async with _approval_semaphore:
        agent = _get_approval_agent()
        try:
            result = await asyncio.wait_for(
                agent.evaluate(request),
                timeout=120.0,
            )
            logger.info(
                "[approval] completed: order=%d verdict=%s confidence=%.2f reason=%.100s",
                request.order_id, result.verdict, result.confidence, result.reason,
            )
            return result
        except asyncio.TimeoutError:
            logger.warning("[approval] timed out after 120s for order %d", request.order_id)
            return ApprovalEvalResponse(
                verdict="escalate",
                confidence=0.1,
                reason="Agent 评估超时",
            )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host=settings.host, port=settings.port)
