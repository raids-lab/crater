"""Planner agent for multi-agent orchestration."""

from __future__ import annotations

import logging
from dataclasses import asdict, dataclass

from crater_agent.agents.base import BaseRoleAgent, RoleExecutionResult

logger = logging.getLogger(__name__)


@dataclass
class PlanOutput:
    """Structured planner output passed to downstream agents."""

    goal: str
    steps: list[str]
    candidate_tools: list[str]
    risk: str = "low"
    raw_summary: str = ""

    def to_dict(self) -> dict:
        return asdict(self)


class PlannerAgent(BaseRoleAgent):
    async def plan(
        self,
        *,
        user_message: str,
        page_context: dict,
        capabilities: dict | None = None,
        actor_role: str = "user",
        evidence_summary: str = "",
        action_history_summary: str = "",
        continuation: dict | None = None,
        replan_reason: str = "",
    ) -> RoleExecutionResult:
        page_summary = page_context or {}
        capability_summary = self.summarize_capabilities(capabilities)

        result = await self.run_json(
            system_prompt=(
                "你是 Crater 的 Planner Agent。你负责只读规划。\n"
                "你可以参考页面上下文、能力摘要、既有证据和可用工具目录来决定调查方案，"
                "但不得建议或执行任何写操作。\n\n"
                "请输出 JSON 格式：\n"
                '{\n'
                '  "goal": "诊断目标（一句话）",\n'
                '  "steps": ["调查步骤1", "调查步骤2"],\n'
                '  "candidate_tools": ["tool_name1", "tool_name2"],\n'
                '  "risk": "low|medium|high",\n'
                '  "raw_summary": "面向其他 agent 的自然语言摘要"\n'
                '}\n\n'
                "candidate_tools 必须从可用工具中选择。使用中文。"
            ),
            user_prompt=(
                f"用户请求:\n{user_message}\n\n"
                f"当前用户角色:\n{actor_role}\n\n"
                f"页面上下文:\n{page_summary}\n\n"
                f"已有证据摘要:\n{evidence_summary or '(empty)'}\n\n"
                f"已有执行历史摘要:\n{action_history_summary or '(empty)'}\n\n"
                f"continuation:\n{continuation or {}}\n\n"
                f"重规划原因:\n{replan_reason or '(none)'}\n\n"
                f"能力摘要:\n{capability_summary}\n\n"
                "请输出结构化 JSON 计划。"
            ),
        )

        plan_output = self._parse_plan_output(result)
        return RoleExecutionResult(
            summary=plan_output.raw_summary or plan_output.goal or "已生成只读计划。",
            metadata={"plan_output": plan_output.to_dict()},
        )

    def _parse_plan_output(self, result: dict | list) -> PlanOutput:
        """Parse LLM JSON into PlanOutput, with fallback for malformed output."""
        fallback_summary = self.latest_reasoning_summary()

        if not isinstance(result, dict):
            logger.warning("Planner output was not a JSON object, using fallback summary")
            return PlanOutput(
                goal="",
                steps=[],
                candidate_tools=[],
                risk="low",
                raw_summary=fallback_summary or str(result),
            )

        if "raw" in result and len(result) == 1:
            # run_json failed to parse, we got raw text
            logger.warning("Planner output was not valid JSON, using raw fallback")
            return PlanOutput(
                goal="",
                steps=[],
                candidate_tools=[],
                risk="low",
                raw_summary=fallback_summary or str(result["raw"]),
            )

        goal = str(result.get("goal") or "").strip()
        steps_raw = result.get("steps") or []
        steps = [str(s) for s in steps_raw] if isinstance(steps_raw, list) else []

        tools_raw = result.get("candidate_tools") or []
        candidate_tools = [str(t) for t in tools_raw] if isinstance(tools_raw, list) else []

        risk = str(result.get("risk") or "low").strip().lower()
        if risk not in ("low", "medium", "high"):
            risk = "low"

        raw_summary = str(result.get("raw_summary") or "").strip()
        if not raw_summary:
            raw_summary = fallback_summary

        return PlanOutput(
            goal=goal,
            steps=steps,
            candidate_tools=candidate_tools,
            risk=risk,
            raw_summary=raw_summary,
        )
