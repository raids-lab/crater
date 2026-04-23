"""ChatJudge: uses tongyi-xiaomi-analysis-flash for dialogue quality evaluation."""
from __future__ import annotations

import json
import logging
from typing import Optional

from crater_agent.llm.client import ModelClientFactory

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
            self.client = ModelClientFactory().create(model_role)
            self.model_name = model_role
        except Exception:
            self.client = None
            self.model_name = model_role
            logger.warning(
                "ChatJudge: model role '%s' not found in config, eval disabled", model_role
            )

    async def judge(self, dialogue_text: str) -> dict:
        """Returns {intent_understanding, completeness, satisfaction_pred, reasoning} or defaults on failure."""
        if not self.client or not dialogue_text.strip():
            return DEFAULT_SCORES

        from crater_agent.quality.prompts import CHAT_JUDGE_SYSTEM, CHAT_JUDGE_USER_TEMPLATE
        from langchain_core.messages import SystemMessage, HumanMessage

        messages = [
            SystemMessage(content=CHAT_JUDGE_SYSTEM),
            HumanMessage(content=CHAT_JUDGE_USER_TEMPLATE.format(dialogue=dialogue_text)),
        ]
        try:
            response = await self.client.ainvoke(messages)
            text = response.content.strip()
            # Strip markdown code fences if present
            if text.startswith("```"):
                text = text.split("```")[1]
                if text.startswith("json"):
                    text = text[4:]
            return json.loads(text)
        except Exception as e:
            logger.warning("ChatJudge failed: %s", e)
            return {**DEFAULT_SCORES, "reasoning": f"eval_error: {e}"}
