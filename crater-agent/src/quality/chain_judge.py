"""ChainJudge: uses a task-eval model for technical reasoning chain evaluation."""
from __future__ import annotations

import asyncio
import json
import logging

from agents.base import BaseRoleAgent
from llm.client import ModelClientFactory

logger = logging.getLogger(__name__)

DEFAULT_SCORES = {
    "tool_relevance": 0,
    "diagnosis_accuracy": 0,
    "suggestion_quality": 0,
    "coherence": 0,
    "reasoning": "eval_failed",
}

MAX_JUDGE_PARSE_ATTEMPTS = 3


def _preview_text(value: str, limit: int = 160) -> str:
    compact = " ".join(str(value or "").split())
    if len(compact) <= limit:
        return compact
    return f"{compact[:limit]}..."


class ChainJudge:
    def __init__(self, model_role: str = "task_eval"):
        try:
            factory = ModelClientFactory()
            config = factory.client_map.get(model_role) or factory.client_map["default"]
            self.client = factory.create(model_role)
            self.model_name = config.model or model_role
            self.model_role = model_role
        except Exception:
            self.client = None
            self.model_name = model_role
            self.model_role = model_role
            logger.warning(
                "ChainJudge: model role '%s' not found in config", model_role
            )

    async def judge(
        self,
        user_query: str,
        tool_calls: list[dict],
        final_response: str,
        *,
        ground_truth_summary: str = "",
    ) -> dict:
        if not self.client or not user_query.strip():
            return DEFAULT_SCORES

        from langchain_core.messages import HumanMessage, SystemMessage

        from quality.prompts import CHAIN_JUDGE_SYSTEM, CHAIN_JUDGE_USER_TEMPLATE

        # Summarize tool calls (limit to avoid token overflow)
        tc_lines = []
        for tc in tool_calls[:20]:
            name = tc.get("toolName") or tc.get("tool_name") or "unknown"
            status = tc.get("resultStatus") or tc.get("result_status") or ""
            preview = str(
                tc.get("toolResultPreview")
                or tc.get("result_preview")
                or ""
            ).strip()
            if len(preview) > 260:
                preview = f"{preview[:260]}..."
            suffix = f": {preview}" if preview else ""
            tc_lines.append(f"- {name} [{status}]{suffix}")
        tool_calls_summary = "\n".join(tc_lines) if tc_lines else "(no tool calls)"

        messages = [
            SystemMessage(content=CHAIN_JUDGE_SYSTEM),
            HumanMessage(
                content=CHAIN_JUDGE_USER_TEMPLATE.format(
                    user_query=user_query[:500],
                    ground_truth_summary=(ground_truth_summary or "(not provided)")[:1200],
                    tool_calls_summary=tool_calls_summary,
                    final_response=final_response[:1200],
                )
            ),
        ]
        last_error: Exception | None = None
        last_content = ""
        last_reasoning = ""
        for attempt in range(1, MAX_JUDGE_PARSE_ATTEMPTS + 1):
            try:
                response = await self.client.ainvoke(messages)
                content = BaseRoleAgent._coerce_text(getattr(response, "content", ""))
                reasoning = BaseRoleAgent._coerce_text(
                    getattr(response, "reasoning_content", "")
                    or (getattr(response, "additional_kwargs", None) or {}).get("reasoning_content", "")
                )
                last_content = content
                last_reasoning = reasoning
                parsed = BaseRoleAgent._parse_json_candidates(content, reasoning)
                if parsed is None:
                    raise json.JSONDecodeError("empty or invalid judge json", content or reasoning or "", 0)
                if not isinstance(parsed, dict):
                    raise json.JSONDecodeError("judge output is not a json object", str(parsed), 0)
                return parsed
            except Exception as e:
                last_error = e
                if attempt >= MAX_JUDGE_PARSE_ATTEMPTS:
                    break
                logger.warning(
                    "ChainJudge retrying after invalid judge output "
                    "(attempt %s/%s): %s; content=%r reasoning=%r",
                    attempt,
                    MAX_JUDGE_PARSE_ATTEMPTS,
                    e,
                    _preview_text(last_content),
                    _preview_text(last_reasoning),
                )
                await asyncio.sleep(0.4 * attempt)

        logger.warning(
            "ChainJudge failed after %s attempts: %s; content=%r reasoning=%r",
            MAX_JUDGE_PARSE_ATTEMPTS,
            last_error,
            _preview_text(last_content),
            _preview_text(last_reasoning),
        )
        return {**DEFAULT_SCORES, "reasoning": f"eval_error: {last_error}"}
