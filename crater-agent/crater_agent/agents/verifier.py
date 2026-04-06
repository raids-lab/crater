"""Verifier agent for multi-agent orchestration."""

from __future__ import annotations

import logging

from crater_agent.agents.base import BaseRoleAgent, RoleExecutionResult

logger = logging.getLogger(__name__)

_VALID_VERDICTS = {"pass", "risk", "missing_evidence"}


class VerifierAgent(BaseRoleAgent):
    async def verify(
        self,
        *,
        user_message: str,
        plan_summary: str,
        evidence_summary: str,
        compact_evidence: list[dict] | None = None,
        executor_summary: str,
    ) -> RoleExecutionResult:
        result = await self.run_json(
            system_prompt=(
                "你是 Crater 的 Verifier Agent。你的职责不是确认实现没问题，"
                "而是指出证据缺口、潜在风险或验证结论。\n\n"
                "你必须优先基于实际证据判断，而不是只重复 Explorer/Executor 的摘要。\n"
                "如果证据不足，要明确指出还缺哪类证据；如果发现风险，要指出冲突点或不一致点。\n\n"
                "你必须输出严格的 JSON 格式:\n"
                '{"verdict": "pass|risk|missing_evidence", "note": "验证说明"}\n\n'
                "verdict 只允许三个值之一: pass, risk, missing_evidence\n"
                "- pass: 证据充分，结论可信\n"
                "- risk: 发现潜在风险或不一致\n"
                "- missing_evidence: 证据不足，需要更多信息\n\n"
                "note 必须用中文说明验证结论。"
            ),
            user_prompt=(
                f"用户请求:\n{user_message}\n\n"
                f"计划摘要:\n{plan_summary or '(empty)'}\n\n"
                f"探索结论:\n{evidence_summary}\n\n"
                f"紧凑证据:\n{compact_evidence or []}\n\n"
                f"执行阶段:\n{executor_summary}\n\n"
                "请给出验证结论。"
            ),
        )

        # Parse with strict validation
        verdict, note = self._parse_verdict(result)

        return RoleExecutionResult(
            summary=note,
            status=verdict,
            metadata={"verification_result": verdict},
        )

    @staticmethod
    def _parse_verdict(result: dict) -> tuple[str, str]:
        """Parse and validate verifier output with safe defaults."""
        # If run_json failed (got raw text fallback)
        if "raw" in result and len(result) == 1:
            logger.warning("Verifier output was not valid JSON, defaulting to missing_evidence")
            return "missing_evidence", "Verifier 输出格式异常，无法解析验证结论。"

        verdict = str(result.get("verdict") or "").strip().lower()
        note = str(result.get("note") or "").strip()

        # Validate verdict is one of the allowed values
        if verdict not in _VALID_VERDICTS:
            logger.warning(
                "Verifier returned invalid verdict '%s', defaulting to missing_evidence",
                verdict,
            )
            verdict = "missing_evidence"
            if not note:
                note = "Verifier 返回了无效的 verdict 值，视为证据不足。"

        if not note:
            note = "验证已完成。"

        return verdict, note
