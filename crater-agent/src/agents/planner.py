"""Planner agent for multi-agent orchestration."""

from __future__ import annotations

import logging
from dataclasses import asdict, dataclass

from agents._domain_rules import render_rules
from agents.base import BaseRoleAgent, RoleExecutionResult

logger = logging.getLogger(__name__)


# Stable role principles. Domain-specific scenario rules are injected
# dynamically via _domain_rules.render_rules().
_PLANNER_ROLE_PRINCIPLES = (
    "你是 Crater 的 Planner Agent。你负责分析用户请求并制定执行计划。\n"
    "你的计划会被 Coordinator 协调者审查，由 Explorer（只读工具）和 Executor（读+写工具）分别执行。\n\n"
    "## 规划原则\n"
    "- 先理解用户到底要什么，再规划步骤；steps 描述要做什么，不需要关心谁来执行。\n"
    "- 已有证据足够回答用户时，可只输出一步「总结回复用户」。\n"
    "- candidate_tools 只能从当前可用工具中选择，不能编造工具名。\n"
    "- 每个非总结计划项必须绑定一个未决字段、一个具体相关工具或真实确认工具、选择该工具的理由、以及停止条件。\n"
    "- tool_hints 是非强制建议；每项只引用当前可用工具，参数只填能从用户请求/页面上下文/已有证据确定的字段，"
    "不确定就留空。tool_hints 每项包含 tool、args、purpose、stop_condition。\n"
    "- 优先窄而直接的工具，避免用宽泛 overview 替代具名对象查询；每轮控制在最小必要步骤。\n"
    "- 不要把场景 id、测试期望、评分标准或固定答案写进计划。\n"
    "- 优先遵守当前页面范围：普通用户页只用用户/当前账户范围工具；admin 场景或用户明确要求全局视角才用集群级工具。\n"
    "- 候选对象歧义时先列候选或请求确认，不要一次性诊断多个互斥对象。\n"
    "- 续接确认结果时先读 workflow/source_turn_context 中已执行的计划/工具/action_history，"
    "禁止把已完成写操作重新规划为待执行。\n"
    "- 写操作计划必须标出目标对象、权限/风险最小核验、最终应调用的确认工具；多个写动作互相独立时可列为并列动作，有依赖时标出先后关系。\n"
    "- 写操作后的验证计划要同时覆盖目标状态和症状/指标两类事实。\n"
)


@dataclass
class PlanOutput:
    """Structured planner output passed to downstream agents."""

    goal: str
    steps: list[str]
    candidate_tools: list[str]
    tool_hints: list[dict]
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
        history_messages: list | None = None,
    ) -> RoleExecutionResult:
        page_summary = page_context or {}
        enabled_tools = list((capabilities or {}).get("enabled_tools") or [])
        surface = dict((capabilities or {}).get("surface") or {})
        all_tool_names = enabled_tools
        capability_summary = self.summarize_capabilities(
            capabilities,
            allowed_tool_names=all_tool_names,
            max_tools=12,
            include_descriptions=True,
            include_role_policies=False,
        )

        is_replan = bool(replan_reason)
        replan_section = f"重规划原因:\n{replan_reason}" if is_replan else "（首次规划）"
        domain_rules_block = render_rules(
            agent="planner",
            tools=enabled_tools,
            message=user_message,
            stage="plan",
        )
        system_prompt = (
            _PLANNER_ROLE_PRINCIPLES
            + (f"\n## 场景规则\n{domain_rules_block}\n" if domain_rules_block else "")
            + "\n## 输出格式\n"
            '{\n'
            '  "goal": "本次目标（一句话）",\n'
            '  "steps": ["步骤1", "步骤2", ...],\n'
            '  "candidate_tools": ["tool_name1", "tool_name2"],\n'
            '  "tool_hints": [\n'
            '    {"tool": "tool_name1", "args": {"key": "value"}, "purpose": "为什么", "stop_condition": "何时停止"}\n'
            '  ],\n'
            '  "risk": "low|medium|high",\n'
            '  "raw_summary": "面向 Coordinator 的自然语言摘要"\n'
            '}\n\n'
            "使用中文。\n"
        )

        result = await self.run_json(
            system_prompt=system_prompt,
            user_prompt=(
                f"用户请求:\n{user_message}\n\n"
                f"当前用户角色:\n{actor_role}\n\n"
                f"页面上下文:\n{page_summary}\n\n"
                f"页面边界:\n{surface or {}}\n\n"
                f"已有证据摘要:\n{evidence_summary or '(empty)'}\n\n"
                f"已有执行历史摘要:\n{action_history_summary or '(empty)'}\n\n"
                f"continuation:\n{continuation or {}}\n\n"
                f"{replan_section}\n\n"
                f"能力摘要:\n{capability_summary}\n\n"
                "请输出结构化 JSON 计划。"
            ),
            history_messages=history_messages,
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
                tool_hints=[],
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
                tool_hints=[],
                risk="low",
                raw_summary=fallback_summary or str(result["raw"]),
            )

        goal = str(result.get("goal") or "").strip()
        steps_raw = result.get("steps") or []
        steps = [str(s) for s in steps_raw] if isinstance(steps_raw, list) else []

        tools_raw = result.get("candidate_tools") or []
        candidate_tools = [str(t) for t in tools_raw] if isinstance(tools_raw, list) else []
        allowed_tools = set(candidate_tools)

        tool_hints_raw = result.get("tool_hints") or result.get("toolHints") or []
        tool_hints: list[dict] = []
        if isinstance(tool_hints_raw, list):
            for item in tool_hints_raw:
                if not isinstance(item, dict):
                    continue
                tool_name = str(item.get("tool") or item.get("tool_name") or item.get("name") or "").strip()
                if not tool_name:
                    continue
                if allowed_tools and tool_name not in allowed_tools:
                    continue
                args = item.get("args") or item.get("tool_args") or {}
                if not isinstance(args, dict):
                    args = {}
                purpose = str(item.get("purpose") or item.get("reason") or "").strip()
                stop_condition = str(item.get("stop_condition") or item.get("stopCondition") or "").strip()
                normalized = {"tool": tool_name, "args": args}
                if purpose:
                    normalized["purpose"] = purpose
                if stop_condition:
                    normalized["stop_condition"] = stop_condition
                if normalized not in tool_hints:
                    tool_hints.append(normalized)

        for hint in tool_hints:
            tool_name = str(hint.get("tool") or "").strip()
            if tool_name and tool_name not in candidate_tools:
                candidate_tools.append(tool_name)

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
            tool_hints=tool_hints,
            risk=risk,
            raw_summary=raw_summary,
        )
