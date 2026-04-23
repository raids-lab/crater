"""ChainJudge: uses Qwen (coordinator role) for technical reasoning chain evaluation."""
from __future__ import annotations

import json
import logging

from crater_agent.llm.client import ModelClientFactory

logger = logging.getLogger(__name__)

DEFAULT_SCORES = {
    "tool_relevance": 0,
    "diagnosis_accuracy": 0,
    "suggestion_quality": 0,
    "coherence": 0,
    "reasoning": "eval_failed",
}


class ChainJudge:
    def __init__(self, model_role: str = "coordinator"):
        try:
            self.client = ModelClientFactory().create(model_role)
            self.model_name = model_role
        except Exception:
            self.client = None
            self.model_name = model_role
            logger.warning(
                "ChainJudge: model role '%s' not found in config", model_role
            )

    async def judge(self, user_query: str, tool_calls: list[dict], final_response: str) -> dict:
        if not self.client or not user_query.strip():
            return DEFAULT_SCORES

        from crater_agent.quality.prompts import CHAIN_JUDGE_SYSTEM, CHAIN_JUDGE_USER_TEMPLATE
        from langchain_core.messages import SystemMessage, HumanMessage

        # Summarize tool calls (limit to avoid token overflow)
        tc_lines = []
        for tc in tool_calls[:20]:
            name = tc.get("tool_name", "unknown")
            status = tc.get("result_status", "")
            tc_lines.append(f"- {name} [{status}]")
        tool_calls_summary = "\n".join(tc_lines) if tc_lines else "(no tool calls)"

        messages = [
            SystemMessage(content=CHAIN_JUDGE_SYSTEM),
            HumanMessage(
                content=CHAIN_JUDGE_USER_TEMPLATE.format(
                    user_query=user_query[:500],
                    tool_calls_summary=tool_calls_summary,
                    final_response=final_response[:800],
                )
            ),
        ]
        try:
            response = await self.client.ainvoke(messages)
            text = response.content.strip()
            if text.startswith("```"):
                text = text.split("```")[1]
                if text.startswith("json"):
                    text = text[4:]
            return json.loads(text)
        except Exception as e:
            logger.warning("ChainJudge failed: %s", e)
            return {**DEFAULT_SCORES, "reasoning": f"eval_error: {e}"}
