"""Base helpers for specialized Crater Agent roles."""

from __future__ import annotations

import json
import logging
import math
from dataclasses import dataclass

from langchain_core.messages import HumanMessage, SystemMessage
from langchain_openai import ChatOpenAI
from openai import APIConnectionError, APITimeoutError, InternalServerError, RateLimitError
from tenacity import retry, retry_if_exception_type, stop_after_attempt, wait_exponential

logger = logging.getLogger(__name__)

_RETRYABLE_LLM_ERRORS = (APITimeoutError, APIConnectionError, InternalServerError, RateLimitError)


@dataclass
class RoleExecutionResult:
    """Normalized role output."""

    summary: str
    status: str = "completed"
    metadata: dict | None = None


class BaseRoleAgent:
    """Minimal wrapper around a role-specific LLM client."""

    def __init__(
        self,
        *,
        agent_id: str,
        role: str,
        llm: ChatOpenAI,
        json_retry_limit: int = 1,
    ):
        self.agent_id = agent_id
        self.role = role
        self.llm = llm
        self.json_retry_limit = max(0, int(json_retry_limit))
        self.last_usage: dict[str, int] = {
            "llm_calls": 0,
            "input_tokens": 0,
            "output_tokens": 0,
        }
        self.last_content = ""
        self.last_reasoning_content = ""
        self.last_selected_text = ""

    async def run_text(self, *, system_prompt: str, user_prompt: str) -> str:
        agent_ref = f"{self.role}/{self.agent_id}"
        self.last_usage = {"llm_calls": 0, "input_tokens": 0, "output_tokens": 0}
        self.last_content = ""
        self.last_reasoning_content = ""
        self.last_selected_text = ""

        @retry(
            stop=stop_after_attempt(3),
            wait=wait_exponential(multiplier=1, min=1, max=8),
            retry=retry_if_exception_type(_RETRYABLE_LLM_ERRORS),
            before_sleep=lambda rs: logger.warning(
                "[%s] LLM call retry #%d after %s: %s",
                agent_ref,
                rs.attempt_number,
                type(rs.outcome.exception()).__name__,
                rs.outcome.exception(),
            ),
        )
        async def _invoke():
            return await self.llm.ainvoke(
                [
                    SystemMessage(content=system_prompt),
                    HumanMessage(content=user_prompt),
                ]
            )

        try:
            response = await _invoke()
            content, reasoning = self._extract_response_texts(response)
            selected = self._select_response_text(content=content, reasoning=reasoning)
            self.last_content = content
            self.last_reasoning_content = reasoning
            self.last_selected_text = selected
            self.last_usage = self._extract_usage(
                response=response,
                system_prompt=system_prompt,
                user_prompt=user_prompt,
                output_text=selected,
            )
            return selected
        except Exception:
            logger.exception("[%s] LLM call failed after retries", agent_ref)
            raise

    async def run_json(self, *, system_prompt: str, user_prompt: str) -> dict | list:
        content = await self.run_text(system_prompt=system_prompt, user_prompt=user_prompt)
        original_content = self.last_content
        original_reasoning = self.last_reasoning_content
        original_selected = self.last_selected_text or content
        aggregate_usage = dict(self.last_usage)
        parsed = self._parse_json_candidates(
            original_content,
            original_reasoning,
            original_selected,
        )
        if parsed is not None:
            return parsed

        repair_input = original_selected or original_content or original_reasoning or content
        for _ in range(self.json_retry_limit):
            repair_content = await self.run_text(
                system_prompt=(
                    "你是 JSON 修复器。"
                    "请把输入内容修复成严格合法的 JSON。"
                    "只输出 JSON，本轮不要输出解释、注释或 Markdown。"
                ),
                user_prompt=(
                    "请尽量保留原字段和原结构，将下面内容修复为合法 JSON：\n\n"
                    f"{repair_input}"
                ),
            )
            aggregate_usage = self._merge_usage(aggregate_usage, self.last_usage)
            parsed = self._parse_json_candidates(
                self.last_content,
                self.last_reasoning_content,
                self.last_selected_text,
                repair_content,
            )
            if parsed is not None:
                self.last_usage = aggregate_usage
                self.last_content = original_content
                self.last_reasoning_content = original_reasoning
                self.last_selected_text = original_selected
                return parsed
            repair_input = self.last_selected_text or repair_content

        self.last_usage = aggregate_usage
        self.last_content = original_content
        self.last_reasoning_content = original_reasoning
        self.last_selected_text = original_selected
        return {"raw": original_selected or original_content or original_reasoning or content}

    @staticmethod
    def _merge_usage(base: dict[str, int], delta: dict[str, int] | None) -> dict[str, int]:
        merged = {
            "llm_calls": int(base.get("llm_calls") or 0),
            "input_tokens": int(base.get("input_tokens") or 0),
            "output_tokens": int(base.get("output_tokens") or 0),
        }
        if isinstance(delta, dict):
            merged["llm_calls"] += int(delta.get("llm_calls") or 0)
            merged["input_tokens"] += int(delta.get("input_tokens") or 0)
            merged["output_tokens"] += int(delta.get("output_tokens") or 0)
        return merged

    @staticmethod
    def _estimate_tokens(text: str) -> int:
        stripped = str(text or "").strip()
        if not stripped:
            return 0
        return max(1, math.ceil(len(stripped) / 4))

    @staticmethod
    def _coerce_text(value: object) -> str:
        if value is None:
            return ""
        if isinstance(value, str):
            return value.strip()
        if isinstance(value, list):
            parts: list[str] = []
            for item in value:
                if isinstance(item, str):
                    text = item
                elif isinstance(item, dict):
                    text = item.get("text") or item.get("content") or ""
                else:
                    text = str(item)
                text = str(text or "").strip()
                if text:
                    parts.append(text)
            return "\n".join(parts).strip()
        return str(value).strip()

    def _extract_response_texts(self, response: object) -> tuple[str, str]:
        additional_kwargs = getattr(response, "additional_kwargs", None) or {}
        content = self._coerce_text(getattr(response, "content", ""))
        reasoning = self._coerce_text(
            getattr(response, "reasoning_content", "")
            or additional_kwargs.get("reasoning_content", "")
        )
        return content, reasoning

    @classmethod
    def _looks_like_json(cls, text: str) -> bool:
        stripped = cls._strip_code_fences(text)
        return (stripped.startswith("{") and stripped.endswith("}")) or (
            stripped.startswith("[") and stripped.endswith("]")
        )

    def _prefers_reasoning_output(self) -> bool:
        return self.role == "planner"

    def _merge_reasoning_and_content(self, *, content: str, reasoning: str) -> str:
        content = str(content or "").strip()
        reasoning = str(reasoning or "").strip()
        if not reasoning:
            return content
        if not content:
            return reasoning
        if reasoning == content:
            return reasoning
        if content in reasoning:
            return reasoning
        if reasoning in content:
            return content
        if self._looks_like_json(content):
            return reasoning
        return f"{reasoning}\n\n补充输出：\n{content}"

    def _select_response_text(self, *, content: str, reasoning: str) -> str:
        if self._prefers_reasoning_output():
            return self._merge_reasoning_and_content(content=content, reasoning=reasoning)
        return content or reasoning

    def latest_reasoning_summary(self) -> str:
        summary = self._merge_reasoning_and_content(
            content=self.last_content,
            reasoning=self.last_reasoning_content,
        )
        return summary or self.last_selected_text

    def _extract_usage(
        self,
        *,
        response: object,
        system_prompt: str,
        user_prompt: str,
        output_text: str,
    ) -> dict[str, int]:
        usage = getattr(response, "usage_metadata", None) or {}
        response_metadata = getattr(response, "response_metadata", None) or {}
        token_usage = (
            response_metadata.get("token_usage") if isinstance(response_metadata, dict) else {}
        ) or {}
        input_tokens = (
            usage.get("input_tokens")
            or usage.get("prompt_tokens")
            or token_usage.get("prompt_tokens")
            or token_usage.get("input_tokens")
            or 0
        )
        output_tokens = (
            usage.get("output_tokens")
            or usage.get("completion_tokens")
            or token_usage.get("completion_tokens")
            or token_usage.get("output_tokens")
            or 0
        )
        if not input_tokens:
            input_tokens = self._estimate_tokens(system_prompt) + self._estimate_tokens(user_prompt)
        if not output_tokens:
            output_tokens = self._estimate_tokens(output_text)
        return {
            "llm_calls": 1,
            "input_tokens": int(input_tokens),
            "output_tokens": int(output_tokens),
        }

    @staticmethod
    def _strip_code_fences(content: str) -> str:
        stripped = str(content or "").strip()
        if not stripped.startswith("```"):
            return stripped
        lines = stripped.splitlines()
        if not lines:
            return stripped
        if lines[-1].strip() == "```":
            lines = lines[1:-1]
        else:
            lines = lines[1:]
        return "\n".join(lines).strip()

    @classmethod
    def _extract_json_fragment(cls, content: str) -> str | None:
        text = cls._strip_code_fences(content)
        start = -1
        for index, char in enumerate(text):
            if char in "{[":
                start = index
                break
        if start < 0:
            return None

        stack: list[str] = []
        in_string = False
        escaped = False
        for index in range(start, len(text)):
            char = text[index]
            if in_string:
                if escaped:
                    escaped = False
                elif char == "\\":
                    escaped = True
                elif char == '"':
                    in_string = False
                continue
            if char == '"':
                in_string = True
                continue
            if char == "{":
                stack.append("}")
                continue
            if char == "[":
                stack.append("]")
                continue
            if char in "}]":
                if not stack or char != stack[-1]:
                    return None
                stack.pop()
                if not stack:
                    return text[start : index + 1]
        return None

    @classmethod
    def _parse_json_candidate(cls, content: str) -> dict | list | None:
        candidates = [cls._strip_code_fences(content)]
        fragment = cls._extract_json_fragment(content)
        if fragment and fragment not in candidates:
            candidates.append(fragment)
        for candidate in candidates:
            if not candidate:
                continue
            try:
                parsed = json.loads(candidate)
            except json.JSONDecodeError:
                continue
            if isinstance(parsed, (dict, list)):
                return parsed
        return None

    @classmethod
    def _parse_json_candidates(cls, *contents: str) -> dict | list | None:
        seen: set[str] = set()
        for content in contents:
            candidate = str(content or "").strip()
            if not candidate or candidate in seen:
                continue
            seen.add(candidate)
            parsed = cls._parse_json_candidate(candidate)
            if parsed is not None:
                return parsed
        return None

    @staticmethod
    def summarize_capabilities(capabilities: dict | None) -> str:
        capabilities = capabilities or {}
        tool_catalog = capabilities.get("tool_catalog") or []
        tool_lines: list[str] = []
        for item in tool_catalog[:12]:
            if not isinstance(item, dict):
                continue
            name = str(item.get("name") or "").strip()
            description = str(item.get("description") or "").strip()
            mode = str(item.get("mode") or "").strip()
            if not name:
                continue
            suffix = f" [{mode}]" if mode else ""
            if description:
                tool_lines.append(f"- {name}{suffix}: {description}")
            else:
                tool_lines.append(f"- {name}{suffix}")

        role_policies = capabilities.get("role_policies") or {}
        policy_lines: list[str] = []
        if isinstance(role_policies, dict):
            for role_name in ("coordinator", "planner", "explorer", "executor", "verifier", "guide", "general"):
                description = str(role_policies.get(role_name) or "").strip()
                if description:
                    policy_lines.append(f"- {role_name}: {description}")

        sections: list[str] = []
        if tool_lines:
            sections.append("可用工具摘要:\n" + "\n".join(tool_lines))
        if policy_lines:
            sections.append("角色约束:\n" + "\n".join(policy_lines))
        return "\n\n".join(sections) if sections else "无额外能力摘要"
