"""Load agent session data from Go backend for quality evaluation."""
from __future__ import annotations

import httpx
from dataclasses import dataclass, field
from typing import Optional


@dataclass
class SessionData:
    session_id: str
    turn_id: Optional[str]
    messages: list[dict]          # [{role, content, tool_calls?, tool_name?}]
    tool_calls: list[dict]        # [{tool_name, tool_args, tool_result, result_status}]
    user_query: str               # first user message content
    final_response: str           # last assistant message content
    dialogue_text: str            # formatted for xiaomi model: "[N] role: content\n..."


class SessionLoader:
    def __init__(self, backend_url: str, internal_token: str):
        self.backend_url = backend_url.rstrip("/")
        self.headers = {"X-Agent-Internal-Token": internal_token}

    async def load(self, session_id: str, turn_id: str | None = None) -> SessionData:
        async with httpx.AsyncClient(timeout=30) as client:
            msgs_resp = await client.get(
                f"{self.backend_url}/internal/agent/sessions/{session_id}/messages",
                headers=self.headers,
            )
            msgs_resp.raise_for_status()
            messages = msgs_resp.json().get("data", [])

            tool_calls: list[dict] = []
            if turn_id:
                tc_resp = await client.get(
                    f"{self.backend_url}/internal/agent/turns/{turn_id}/tool-calls",
                    headers=self.headers,
                )
                if tc_resp.status_code == 200:
                    tool_calls = tc_resp.json().get("data", [])

        # Build dialogue text in [N] role: content format for xiaomi
        dialogue_lines = []
        user_query = ""
        final_response = ""
        n = 0
        for msg in messages:
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
            messages=messages,
            tool_calls=tool_calls,
            user_query=user_query,
            final_response=final_response,
            dialogue_text="\n".join(dialogue_lines),
        )
