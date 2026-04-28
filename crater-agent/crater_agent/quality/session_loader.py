"""Load agent session data from Go backend for quality evaluation."""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Optional

import httpx


@dataclass
class SessionData:
    session_id: str
    turn_id: Optional[str]
    eval_scope: str
    messages: list[dict]          # [{role, content, tool_calls?, tool_name?}]
    tool_calls: list[dict]        # [{tool_name, tool_args, tool_result, result_status}]
    user_query: str               # first user message content
    final_response: str           # last assistant message content
    dialogue_text: str            # formatted for xiaomi model: "[N] role: content\n..."
    turns: list[dict] = field(default_factory=list)


class SessionLoader:
    def __init__(self, backend_url: str, internal_token: str):
        self.backend_url = backend_url.rstrip("/")
        self.headers = {"X-Agent-Internal-Token": internal_token}

    async def load(
        self,
        session_id: str,
        turn_id: str | None = None,
        eval_scope: str = "session",
    ) -> SessionData:
        scope = "turn" if eval_scope == "turn" and turn_id else "session"
        async with httpx.AsyncClient(timeout=30) as client:
            msgs_resp = await client.get(
                f"{self.backend_url}/internal/agent/sessions/{session_id}/messages",
                headers=self.headers,
            )
            msgs_resp.raise_for_status()
            messages = msgs_resp.json().get("data", [])

            turns_resp = await client.get(
                f"{self.backend_url}/internal/agent/sessions/{session_id}/turns",
                headers=self.headers,
            )
            turns_resp.raise_for_status()
            turns = turns_resp.json().get("data", [])

            if scope == "turn":
                tc_resp = await client.get(
                    f"{self.backend_url}/internal/agent/turns/{turn_id}/tool-calls",
                    headers=self.headers,
                )
                tc_resp.raise_for_status()
                tool_calls = tc_resp.json().get("data", [])
            else:
                tc_resp = await client.get(
                    f"{self.backend_url}/internal/agent/sessions/{session_id}/tool-calls",
                    headers=self.headers,
                )
                tc_resp.raise_for_status()
                tool_calls = tc_resp.json().get("data", [])

        target_messages = messages
        if scope == "turn" and turn_id:
            target_turn = next((turn for turn in turns if turn.get("turnId") == turn_id), None)
            final_message_id = target_turn.get("finalMessageId") if target_turn else None
            if final_message_id:
                final_index = next(
                    (
                        index
                        for index, msg in enumerate(messages)
                        if str(msg.get("id")) == str(final_message_id)
                    ),
                    -1,
                )
                if final_index >= 0:
                    start_index = 0
                    for index in range(final_index - 1, -1, -1):
                        if messages[index].get("role") == "user":
                            start_index = index
                            break
                    target_messages = messages[start_index : final_index + 1]

        # Build dialogue text in [N] role: content format for xiaomi
        dialogue_lines = []
        user_query = ""
        final_response = ""
        n = 0
        for msg in target_messages:
            role = msg.get("role", "")
            content = msg.get("content", "") or ""
            if role == "user":
                n += 1
                role_label = "用户"
                if not user_query:
                    user_query = content
                dialogue_lines.append(f"[{n}] {role_label}：{content[:500]}")
            elif role == "assistant" and content:
                n += 1
                role_label = "助手"
                final_response = content
                dialogue_lines.append(f"[{n}] {role_label}：{content[:500]}")

        return SessionData(
            session_id=session_id,
            turn_id=turn_id,
            eval_scope=scope,
            messages=target_messages,
            tool_calls=tool_calls,
            turns=turns,
            user_query=user_query,
            final_response=final_response,
            dialogue_text="\n".join(dialogue_lines),
        )
