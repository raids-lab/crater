"""Coordinator agent for multi-agent orchestration."""

from __future__ import annotations

import logging
from dataclasses import dataclass
from typing import Any

from crater_agent.agents.base import BaseRoleAgent, RoleExecutionResult

logger = logging.getLogger(__name__)


def _looks_like_system_job_name(value: str) -> bool:
    normalized = value.strip().lower()
    if not normalized:
        return False
    if normalized.count("-") < 2:
        return False
    return any(ch.isdigit() for ch in normalized)


@dataclass
class TurnContextDecision:
    route: str = "diagnostic"
    action_intent: str | None = None
    selected_job_name: str | None = None
    requested_scope: str = "unspecified"
    rationale: str = ""


@dataclass
class LoopDecision:
    step: str = "finalize"
    rationale: str = ""


class CoordinatorAgent(BaseRoleAgent):
    async def decide_turn_context(
        self,
        *,
        user_message: str,
        page_context: dict[str, Any],
        continuation: dict[str, Any] | None = None,
        recent_history_excerpt: str = "",
        capabilities: dict[str, Any] | None = None,
    ) -> TurnContextDecision:
        capability_summary = self.summarize_capabilities(
            capabilities,
            max_tools=6,
            include_descriptions=False,
            include_role_policies=False,
        )

        result = await self.run_json(
            system_prompt=(
                "你是 Crater 的 Coordinator Agent。你负责解释“当前这一轮”用户输入，"
                "然后决定 multi-agent 的路由与上下文绑定。\n\n"
                "你会拿到四类信息：\n"
                "1. 当前用户输入：这是本轮真正要处理的内容，优先级最高。\n"
                "2. 页面上下文：帮助判断用户当前关注的页面或作业。\n"
                "3. continuation：由后端提供的结构化续接上下文，表示上一轮是否还在等待用户补充信息或确认。\n"
                "4. recent_history_excerpt：最近对话摘录，只能辅助理解省略表达，不能盖过当前输入。\n\n"
                "工作原则：\n"
                "- 必须以当前用户输入为主；不要因为历史里提到某个动作，就忽略当前输入的新意图。\n"
                "- 如果 continuation.clarification 表示上一轮在等待用户从候选作业中选择，"
                "而当前输入只是“第一个/这个/全部/某个 jobName”，你应结合 continuation 解析。\n"
                "- 如果当前输入已经改变主题，例如从“重提失败作业”切换到“为什么失败/怎么看日志/怎么创建作业”，"
                "就不要机械沿用旧动作。\n"
                "- guide: 帮助、文档、概念解释、使用说明。\n"
                "- general: 普通平台问答，不需要具体作业诊断或写操作。\n"
                "- diagnostic: 具体作业排障、作业列表查询、资源分析、以及任何写操作意图。\n"
                "- action_intent 仅在你确认当前输入仍是 stop/delete/resubmit 其中之一时填写，否则填 null。\n"
                "- selected_job_name 只在你能明确绑定到单个系统 jobName 时填写；无法确定就填 null。\n"
                "- requested_scope=all 只在当前输入明确表达“全部/所有/all/every one of them”等整体范围时填写；"
                "不要因为 failed jobs / 失败作业 这类泛指复数表达就自动推断为 all。\n"
                '- requested_scope 只能是 "single"、"all"、"unspecified"。\n\n'
                "输出 JSON：\n"
                "{\n"
                '  "route": "guide|general|diagnostic",\n'
                '  "action_intent": "resubmit|stop|delete|null",\n'
                '  "selected_job_name": "sg-xxx|jpt-xxx|null",\n'
                '  "requested_scope": "single|all|unspecified",\n'
                '  "rationale": "简短理由"\n'
                "}\n"
            ),
            user_prompt=(
                f"当前用户输入:\n{user_message}\n\n"
                f"页面上下文:\n{page_context}\n\n"
                f"continuation:\n{continuation or {}}\n\n"
                f"recent_history_excerpt:\n{recent_history_excerpt or '(empty)'}\n\n"
                f"能力摘要:\n{capability_summary}\n\n"
                "请输出结构化 JSON。"
            ),
        )

        return self._parse_turn_context(result)

    @staticmethod
    def _parse_turn_context(result: dict[str, Any] | list[Any]) -> TurnContextDecision:
        if not isinstance(result, dict) or ("raw" in result and len(result) == 1):
            logger.warning("Coordinator turn-context decision was invalid: %s", result)
            return TurnContextDecision()

        route = str(result.get("route") or "").strip().lower()
        if route not in {"guide", "general", "diagnostic"}:
            route = "diagnostic"

        action_intent = str(result.get("action_intent") or "").strip().lower() or None
        if action_intent in {"null", "none"}:
            action_intent = None
        if action_intent not in {None, "resubmit", "stop", "delete"}:
            action_intent = None

        selected_job_name = (
            str(result.get("selected_job_name") or result.get("job_name") or "").strip().lower()
            or None
        )
        if selected_job_name in {"null", "none"}:
            selected_job_name = None
        if selected_job_name and not _looks_like_system_job_name(selected_job_name):
            selected_job_name = None

        requested_scope = str(result.get("requested_scope") or "").strip().lower()
        if requested_scope not in {"single", "all", "unspecified"}:
            requested_scope = "unspecified"

        rationale = str(result.get("rationale") or "").strip()

        return TurnContextDecision(
            route=route,
            action_intent=action_intent,
            selected_job_name=selected_job_name,
            requested_scope=requested_scope,
            rationale=rationale,
        )

    async def decide_next_step(
        self,
        *,
        user_message: str,
        page_context: dict[str, Any],
        plan_summary: str,
        evidence_summary: str,
        action_history_summary: str,
        pending_actions: list[dict[str, Any]],
        continuation: dict[str, Any] | None = None,
        loop_iteration: int = 1,
        replan_count: int = 0,
        verification_summary: str = "",
    ) -> LoopDecision:
        result = await self.run_json(
            system_prompt=(
                "你是 Crater 的 Coordinator Agent。你现在不负责高层路由，"
                "而是负责当前 turn 内的 controller loop 决策。\n\n"
                "你必须在下面步骤中选一个：\n"
                '- "explore": 继续收集只读证据\n'
                '- "execute": 执行当前已准备好的写操作，或让 Executor 产出下一批动作\n'
                '- "verify": 对现有结论做挑战式验证\n'
                '- "replan": 当前计划失效，需要 Planner 基于新信息重规划\n'
                '- "finalize": 信息已足够，可以直接输出最终答复\n\n'
                "决策原则：\n"
                "- 如果已经存在 pending_actions，优先 execute，不要重复 explore。\n"
                "- 如果证据明显不够、参数不明确或目标对象未定位，优先 explore。\n"
                "- 只有在证据或世界状态变化导致原计划不再成立时才 replan。\n"
                "- verify 通常在已经有结论或动作结果之后再做。\n"
                "- 不要因为历史动作而忽略当前输入，但 continuation 可以表示这是确认后的继续执行。\n"
                "输出 JSON:\n"
                '{\n'
                '  "step": "explore|execute|verify|replan|finalize",\n'
                '  "rationale": "简短理由"\n'
                '}\n'
            ),
            user_prompt=(
                f"当前用户输入:\n{user_message}\n\n"
                f"页面上下文:\n{page_context}\n\n"
                f"当前计划摘要:\n{plan_summary or '(empty)'}\n\n"
                f"探索摘要:\n{evidence_summary or '(empty)'}\n\n"
                f"执行历史摘要:\n{action_history_summary or '(empty)'}\n\n"
                f"待执行动作:\n{pending_actions or []}\n\n"
                f"验证摘要:\n{verification_summary or '(empty)'}\n\n"
                f"continuation:\n{continuation or {}}\n\n"
                f"loop_iteration={loop_iteration}, replan_count={replan_count}\n\n"
                "请输出结构化 JSON。"
            ),
        )
        return self._parse_loop_decision(result)

    @staticmethod
    def _parse_loop_decision(result: dict[str, Any] | list[Any]) -> LoopDecision:
        if not isinstance(result, dict) or ("raw" in result and len(result) == 1):
            logger.warning("Coordinator loop decision was invalid: %s", result)
            return LoopDecision(step="finalize")

        step = str(result.get("step") or "").strip().lower()
        if step not in {"explore", "execute", "verify", "replan", "finalize"}:
            step = "finalize"
        rationale = str(result.get("rationale") or "").strip()
        return LoopDecision(step=step, rationale=rationale)

    async def summarize(
        self,
        *,
        user_message: str,
        plan_summary: str,
        evidence_summary: str,
        compact_evidence: list[dict[str, Any]] | None = None,
        executor_summary: str,
        verifier_summary: str,
    ) -> RoleExecutionResult:
        summary = await self.run_text(
            system_prompt=(
                "你是 Crater 的 Coordinator Agent。你负责整合 Planner、Explorer、Executor、Verifier "
                "的输出，向用户给出最终答复。要求中文、结论在前、证据在后、建议最后。\n"
                "请优先基于实际证据作答，不要只复述其他 agent 的摘要。"
            ),
            user_prompt=(
                f"用户请求:\n{user_message}\n\n"
                f"Planner:\n{plan_summary}\n\n"
                f"Explorer:\n{evidence_summary}\n\n"
                f"紧凑证据:\n{compact_evidence or []}\n\n"
                f"Executor:\n{executor_summary}\n\n"
                f"Verifier:\n{verifier_summary}\n\n"
                "请输出最终面向用户的自然语言回复。"
            ),
        )
        return RoleExecutionResult(summary=summary or "已完成最终总结。")
