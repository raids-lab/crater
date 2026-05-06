"""Core quality analyzer: orchestrates ChatJudge + ChainJudge, saves results."""
from __future__ import annotations

import logging
from typing import Optional

import httpx

from config import settings
from quality.chain_judge import ChainJudge
from quality.chat_judge import ChatJudge
from quality.paths import feedback_artifact_path, manual_artifact_path
from quality.reporter import write_md_report
from quality.session_loader import SessionLoader

logger = logging.getLogger(__name__)

EVAL_SCOPE_SESSION = "session"
EVAL_SCOPE_TURN = "turn"
EVAL_TYPE_FULL = "full"
EVAL_TYPE_DIALOGUE = "dialogue"
EVAL_TYPE_TASK = "task"


def _normalize_scope(scope: str | None, turn_id: Optional[str]) -> str:
    normalized = (scope or "").strip().lower()
    if normalized == EVAL_SCOPE_TURN and turn_id:
        return EVAL_SCOPE_TURN
    return EVAL_SCOPE_SESSION


def _normalize_eval_type(eval_type: str | None) -> str:
    normalized = (eval_type or "").strip().lower()
    if normalized in {EVAL_TYPE_DIALOGUE, EVAL_TYPE_TASK}:
        return normalized
    return EVAL_TYPE_FULL


class QualityAnalyzer:
    def __init__(
        self,
        backend_url: str,
        internal_token: str,
        chat_model_role: str = "dialogue_eval_flash",
        chain_model_role: str = "task_eval",
    ):
        self.backend_url = backend_url.rstrip("/")
        self.internal_token = internal_token
        self.loader = SessionLoader(backend_url, internal_token)
        self.chat_model_role = chat_model_role
        self.chain_model_role = chain_model_role

    async def analyze(
        self,
        eval_id: int,
        session_id: str,
        turn_id: Optional[str],
        eval_scope: str = EVAL_SCOPE_SESSION,
        eval_type: str = EVAL_TYPE_FULL,
        trigger_source: str = "feedback",
        rating: Optional[int] = None,
        feedback_id: Optional[int] = None,
        dialogue_model_role: Optional[str] = None,
        task_model_role: Optional[str] = None,
    ) -> None:
        """Run quality analysis and write results back to Go backend + artifact files."""
        scope = _normalize_scope(eval_scope, turn_id)
        selected_type = _normalize_eval_type(eval_type)
        try:
            data = await self.loader.load(session_id, turn_id, scope)
        except Exception as e:
            logger.error("QualityAnalyzer: failed to load session %s: %s", session_id, e)
            await self._report_failure(eval_id, str(e))
            return

        import asyncio

        chat_judge: ChatJudge | None = None
        chain_judge: ChainJudge | None = None
        chat_scores: dict = {}
        chain_scores: dict = {}
        tasks = []

        if selected_type in {EVAL_TYPE_FULL, EVAL_TYPE_DIALOGUE}:
            chat_judge = ChatJudge(dialogue_model_role or self.chat_model_role)
            tasks.append(("chat", asyncio.create_task(chat_judge.judge(data.dialogue_text))))
        if selected_type in {EVAL_TYPE_FULL, EVAL_TYPE_TASK}:
            chain_judge = ChainJudge(task_model_role or self.chain_model_role)
            tasks.append(
                (
                    "chain",
                    asyncio.create_task(
                        chain_judge.judge(data.user_query, data.tool_calls, data.final_response)
                    ),
                )
            )

        results = await asyncio.gather(*(task for _, task in tasks))
        for (kind, _), scores in zip(tasks, results):
            if kind == "chat":
                chat_scores = scores
            else:
                chain_scores = scores

        artifact_path = ""
        if settings.quality_eval_write_artifacts:
            if trigger_source == "feedback":
                artifact = feedback_artifact_path(session_id, turn_id)
            else:
                artifact = manual_artifact_path(session_id)

            try:
                write_md_report(
                    artifact,
                    session_id,
                    turn_id,
                    chat_scores,
                    chain_scores,
                    trigger_source,
                    rating,
                )
                artifact_path = str(artifact)
            except Exception as e:
                logger.warning(
                    "QualityAnalyzer: failed to persist artifact for session %s: %s",
                    session_id,
                    e,
                )

        # Write result back to Go backend
        await self._write_result(
            eval_id,
            chat_scores,
            chain_scores,
            artifact_path,
            eval_scope=scope,
            eval_type=selected_type,
            chat_model=chat_judge.model_name if chat_judge else "",
            chain_model=chain_judge.model_name if chain_judge else "",
            metadata={
                "evalScope": scope,
                "evalType": selected_type,
                "dialogueModelRole": chat_judge.model_role if chat_judge else "",
                "taskModelRole": chain_judge.model_role if chain_judge else "",
                "feedbackId": feedback_id,
            },
        )

    async def _write_result(
        self,
        eval_id: int,
        chat_scores: dict,
        chain_scores: dict,
        artifact_path: str,
        eval_scope: str,
        eval_type: str,
        chat_model: str,
        chain_model: str,
        metadata: dict,
    ) -> None:
        summary_parts = [
            str(chat_scores.get("reasoning", "")).strip(),
            str(chain_scores.get("reasoning", "")).strip(),
        ]
        payload = {
            "evalId": eval_id,
            "evalStatus": "completed",
            "chatScores": chat_scores,
            "chainScores": chain_scores,
            "chatModel": chat_model,
            "chainModel": chain_model,
            "summary": " | ".join(part for part in summary_parts if part),
            "rawChatResp": chat_scores,
            "rawChainResp": chain_scores,
            "artifactPath": artifact_path,
            "metadata": metadata,
        }
        try:
            async with httpx.AsyncClient(timeout=10) as client:
                resp = await client.post(
                    f"{self.backend_url}/internal/agent/quality-evals",
                    json=payload,
                    headers={"X-Agent-Internal-Token": self.internal_token},
                )
                resp.raise_for_status()
        except Exception as e:
            logger.error("QualityAnalyzer: failed to write result for eval %d: %s", eval_id, e)

    async def _report_failure(self, eval_id: int, error_msg: str) -> None:
        try:
            async with httpx.AsyncClient(timeout=5) as client:
                await client.post(
                    f"{self.backend_url}/internal/agent/quality-evals",
                    json={"evalId": eval_id, "evalStatus": "failed", "summary": error_msg},
                    headers={"X-Agent-Internal-Token": self.internal_token},
                )
        except Exception:
            pass
