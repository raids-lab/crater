"""Core quality analyzer: orchestrates ChatJudge + ChainJudge, saves results."""
from __future__ import annotations

import logging
from datetime import datetime
from typing import Optional

import httpx

from crater_agent.quality.chat_judge import ChatJudge
from crater_agent.quality.chain_judge import ChainJudge
from crater_agent.quality.session_loader import SessionLoader
from crater_agent.quality.reporter import write_md_report
from crater_agent.quality.paths import feedback_artifact_path, manual_artifact_path

logger = logging.getLogger(__name__)


class QualityAnalyzer:
    def __init__(
        self,
        backend_url: str,
        internal_token: str,
        chat_model_role: str = "dialogue_eval_flash",
        chain_model_role: str = "coordinator",
    ):
        self.backend_url = backend_url.rstrip("/")
        self.internal_token = internal_token
        self.loader = SessionLoader(backend_url, internal_token)
        self.chat_judge = ChatJudge(chat_model_role)
        self.chain_judge = ChainJudge(chain_model_role)

    async def analyze(
        self,
        eval_id: int,
        session_id: str,
        turn_id: Optional[str],
        trigger_source: str = "feedback",
        rating: Optional[int] = None,
        feedback_id: Optional[int] = None,
    ) -> None:
        """Run quality analysis and write results back to Go backend + artifact files."""
        try:
            data = await self.loader.load(session_id, turn_id)
        except Exception as e:
            logger.error("QualityAnalyzer: failed to load session %s: %s", session_id, e)
            await self._report_failure(eval_id, str(e))
            return

        # Run both judges concurrently
        import asyncio

        chat_task = asyncio.create_task(self.chat_judge.judge(data.dialogue_text))
        chain_task = asyncio.create_task(
            self.chain_judge.judge(data.user_query, data.tool_calls, data.final_response)
        )
        chat_scores, chain_scores = await asyncio.gather(chat_task, chain_task)

        # Write artifact file
        if trigger_source == "feedback":
            artifact = feedback_artifact_path(session_id, turn_id)
        else:
            artifact = manual_artifact_path(session_id)

        write_md_report(artifact, session_id, turn_id, chat_scores, chain_scores, trigger_source, rating)

        # Write result back to Go backend
        await self._write_result(eval_id, chat_scores, chain_scores, str(artifact))

    async def _write_result(
        self, eval_id: int, chat_scores: dict, chain_scores: dict, artifact_path: str
    ) -> None:
        payload = {
            "evalId": eval_id,
            "chatScores": chat_scores,
            "chainScores": chain_scores,
            "chatModel": self.chat_judge.model_name,
            "chainModel": self.chain_judge.model_name,
            "summary": chat_scores.get("reasoning", "") + " | " + chain_scores.get("reasoning", ""),
            "rawChatResp": chat_scores,
            "rawChainResp": chain_scores,
            "artifactPath": artifact_path,
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
