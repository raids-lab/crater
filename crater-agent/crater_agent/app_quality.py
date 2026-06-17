"""Quality eval FastAPI routes (internal + manual trigger)."""
from __future__ import annotations

import logging
from typing import Optional

from fastapi import APIRouter, BackgroundTasks, Header, HTTPException
from pydantic import BaseModel

from crater_agent.config import settings
from crater_agent.quality.analyzer import QualityAnalyzer

logger = logging.getLogger(__name__)
router = APIRouter()


def _get_analyzer() -> QualityAnalyzer:
    return QualityAnalyzer(
        backend_url=str(settings.crater_backend_url),
        internal_token=settings.agent_internal_token,
    )


class FeedbackTriggerRequest(BaseModel):
    eval_id: int
    session_id: str
    turn_id: Optional[str] = None
    eval_scope: str = "session"
    eval_type: str = "full"
    feedback_id: Optional[int] = None
    rating: Optional[int] = None
    dialogue_model_role: Optional[str] = None
    task_model_role: Optional[str] = None


class ManualTriggerRequest(BaseModel):
    session_id: str
    turn_id: Optional[str] = None
    eval_id: Optional[int] = None
    eval_scope: str = "session"
    eval_type: str = "full"
    dialogue_model_role: Optional[str] = None
    task_model_role: Optional[str] = None


@router.post("/eval/quality/feedback")
async def trigger_quality_eval_from_feedback(
    req: FeedbackTriggerRequest,
    background_tasks: BackgroundTasks,
    x_agent_internal_token: Optional[str] = Header(default=None),
):
    """Internal endpoint: triggered by Go backend after user submits feedback."""
    if x_agent_internal_token != settings.agent_internal_token:
        raise HTTPException(status_code=401, detail="unauthorized")

    analyzer = _get_analyzer()
    background_tasks.add_task(
        analyzer.analyze,
        eval_id=req.eval_id,
        session_id=req.session_id,
        turn_id=req.turn_id,
        eval_scope=req.eval_scope,
        eval_type=req.eval_type,
        trigger_source="feedback",
        rating=req.rating,
        feedback_id=req.feedback_id,
        dialogue_model_role=req.dialogue_model_role,
        task_model_role=req.task_model_role,
    )
    return {"status": "accepted"}


@router.post("/eval/quality/session")
async def trigger_quality_eval_manual(
    req: ManualTriggerRequest,
    background_tasks: BackgroundTasks,
    x_agent_internal_token: Optional[str] = Header(default=None),
):
    """Manual trigger for single session quality evaluation."""
    if x_agent_internal_token != settings.agent_internal_token:
        raise HTTPException(status_code=401, detail="unauthorized")

    # For manual trigger, eval_id may be provided or we use 0 as sentinel
    eval_id = req.eval_id or 0
    analyzer = _get_analyzer()
    background_tasks.add_task(
        analyzer.analyze,
        eval_id=eval_id,
        session_id=req.session_id,
        turn_id=req.turn_id,
        eval_scope=req.eval_scope,
        eval_type=req.eval_type,
        trigger_source="manual",
        dialogue_model_role=req.dialogue_model_role,
        task_model_role=req.task_model_role,
    )
    return {"status": "accepted", "session_id": req.session_id}
