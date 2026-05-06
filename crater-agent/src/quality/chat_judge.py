"""ChatJudge: uses tongyi-xiaomi-analysis-flash for dialogue quality evaluation."""
from __future__ import annotations

import json
import logging

from agents.base import BaseRoleAgent
from llm.client import ModelClientFactory

logger = logging.getLogger(__name__)

DEFAULT_SCORES = {
    "intent_understanding": 0,
    "completeness": 0,
    "satisfaction_pred": 0,
    "reasoning": "eval_failed",
}


class ChatJudge:
    def __init__(self, model_role: str = "dialogue_eval_flash"):
        # Falls back to default if not configured
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
                "ChatJudge: model role '%s' not found in config, eval disabled", model_role
            )

    async def judge(self, dialogue_text: str) -> dict:
        """Return dialogue scores, or defaults when the judge cannot run."""
        if not self.client or not dialogue_text.strip():
            return DEFAULT_SCORES

        from langchain_core.messages import HumanMessage, SystemMessage

        from quality.prompts import CHAT_JUDGE_SYSTEM, CHAT_JUDGE_USER_TEMPLATE

        messages = [
            SystemMessage(content=CHAT_JUDGE_SYSTEM),
            HumanMessage(content=CHAT_JUDGE_USER_TEMPLATE.format(dialogue=dialogue_text)),
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
            logger.warning("ChatJudge failed: %s", e)
            return {**DEFAULT_SCORES, "reasoning": f"eval_error: {e}"}
