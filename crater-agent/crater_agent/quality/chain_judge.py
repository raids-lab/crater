"""ChainJudge: uses a task-eval model for technical reasoning chain evaluation."""
from __future__ import annotations

import json
import logging

from crater_agent.agents.base import BaseRoleAgent
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

    async def judge(self, user_query: str, tool_calls: list[dict], final_response: str) -> dict:
        if not self.client or not user_query.strip():
            return DEFAULT_SCORES

        from langchain_core.messages import HumanMessage, SystemMessage

        from crater_agent.quality.prompts import CHAIN_JUDGE_SYSTEM, CHAIN_JUDGE_USER_TEMPLATE

        # Summarize tool calls (limit to avoid token overflow)
        tc_lines = []
        for tc in tool_calls[:20]:
            name = tc.get("toolName") or tc.get("tool_name") or "unknown"
            status = tc.get("resultStatus") or tc.get("result_status") or ""
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
            content = BaseRoleAgent._coerce_text(getattr(response, "content", ""))
            reasoning = BaseRoleAgent._coerce_text(
                getattr(response, "reasoning_content", "")
                or (getattr(response, "additional_kwargs", None) or {}).get("reasoning_content", "")
            )
            parsed = BaseRoleAgent._parse_json_candidates(content, reasoning)
            if parsed is None:
                raise json.JSONDecodeError("empty or invalid judge json", content or reasoning or "", 0)
            if not isinstance(parsed, dict):
                raise json.JSONDecodeError("judge output is not a json object", str(parsed), 0)
            return parsed
        except Exception as e:
            logger.warning("ChainJudge failed: %s", e)
            return {**DEFAULT_SCORES, "reasoning": f"eval_error: {e}"}
