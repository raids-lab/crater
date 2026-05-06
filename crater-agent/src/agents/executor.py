"""Executor agent for multi-agent orchestration."""

from __future__ import annotations

import json
import logging

from agents._domain_rules import render_rules
from agents.base import BaseRoleAgent, RoleExecutionResult
from tools.definitions import CONFIRM_TOOL_NAMES

logger = logging.getLogger(__name__)


# Stable role principles. Domain-specific rules are injected via render_rules().
_EXECUTOR_TOOL_LOOP_PRINCIPLES = (
    "你是 Crater 的 Executor Agent。你的职责是根据用户请求和现有证据，直接推进下一步工具执行。\n"
    "可以连续多轮调用工具，每次拿到结果后基于结果判断下一步。无需再调用工具时直接给出中文执行总结。\n\n"
    "## 执行原则\n"
    "- actor_role 不是管理员时，不要尝试执行管理员专属写操作；说明权限限制并建议联系管理员。\n"
    "- 纯查询/诊断请求不要擅自执行写操作；写操作前确保目标对象明确，不明确就先补最小只读核验。\n"
    "- 用户自述、页面状态或上一轮口头结论都不是已验证事实；restart/stop/scale/uncordon/create 这类写操作前应先补 1-2 条直接相关事实。\n"
    "- 写意图、目标对象和最小风险事实明确时，必须调用真实确认工具进入 confirmation_required；不要输出伪工具文本或仅给口头建议。\n"
    "- run_kubectl / execute_admin_command 属于受控高风险动作：先说明触发原因再等待确认流，不得构造任意 shell 文本。\n"
    "- 同一轮多个互相独立的确认型写动作可以一次性发起多张确认卡；有依赖的动作必须用 depends_on 标出或只输出当前可执行的最小动作。\n"
    "- 确认型写操作完成后，用户要求继续验证时不要再次发起同一个写操作；先用只读工具核验目标状态，再核验流量/错误率/队列等症状。\n"
    "- continuation.resume_after_confirmation、source_turn_context.tool_calls 或 action_history 表明同一写工具已完成时，禁止再次执行。\n"
    "- 不要重复调用相同参数的工具，除非世界状态明显变化。\n"
)

_EXECUTOR_DECIDE_PRINCIPLES = (
    "你是 Crater 的 Executor Agent。你的职责是基于用户请求和证据决定下一批原子写操作。\n"
    "你只能输出写操作计划，不能执行工具，也不要总结。\n\n"
    "## 输出原则\n"
    "- 只有用户明确要求操作时才选择写操作；纯诊断/查询请求返回空 actions。\n"
    "- actor_role 不是管理员时不要输出管理员专属写操作。\n"
    "- 每个 action 是原子动作；depends_on 用 1-based 下标表示依赖更早的动作，无依赖填空数组。\n"
    "- 多个互相独立的确认型写动作可以并列输出（depends_on 为空）；有依赖时不要并列。\n"
    "- 不要把用户自述当作已验证证据；写操作仍依赖关键事实未核实时返回空 actions，让前面的只读阶段补证据。\n"
    "- 已有 pending_actions 不要重复生成等价动作；同一写工具已 confirmed/completed 时返回空 actions。\n"
    "- args 必须有效：k8s_scale_workload 至少含 kind/name/replicas，namespace 不确定时返回空 actions；create 类工具尽量复用模板/镜像/配额证据中的字段，缺关键字段不要编造。\n"
    "- requested_scope=all 且证据中有候选对象时优先使用批量确认工具（如 batch_stop_jobs）；没有批量工具再输出多条原子动作。\n"
    '- 选择 run_kubectl / execute_admin_command 时，reason 必须体现「结构化工具和只读证据不足，必须执行受控命令」。\n'
)


def _format_plan_tool_hints(plan_tool_hints: list[dict] | None) -> str:
    if not plan_tool_hints:
        return "(无)"
    lines: list[str] = []
    for index, item in enumerate(plan_tool_hints, start=1):
        if not isinstance(item, dict):
            continue
        tool_name = str(item.get("tool") or item.get("tool_name") or "").strip()
        if not tool_name:
            continue
        args = item.get("args") or item.get("tool_args") or {}
        if not isinstance(args, dict):
            args = {}
        purpose = str(item.get("purpose") or "").strip()
        stop_condition = str(item.get("stop_condition") or item.get("stopCondition") or "").strip()
        detail = f"{tool_name}({json.dumps(args, ensure_ascii=False, sort_keys=True)})"
        if purpose:
            detail += f"；目的：{purpose}"
        if stop_condition:
            detail += f"；停止条件：{stop_condition}"
        lines.append(f"  {index}. {detail}")
    return "\n".join(lines) or "(无)"


class ExecutorAgent(BaseRoleAgent):
    def build_tool_loop_prompts(
        self,
        *,
        user_message: str,
        page_context: dict,
        plan_summary: str,
        evidence_summary: str,
        compact_evidence: list[dict],
        action_intent: str | None,
        selected_job_name: str | None,
        requested_scope: str,
        action_history: list[dict],
        pending_actions: list[dict],
        enabled_tools: list[str],
        actor_role: str = "user",
        plan_tool_hints: list[dict] | None = None,
    ) -> tuple[str, str]:
        allowed_tools = sorted(set(enabled_tools))
        visible_tools = allowed_tools[:10]
        hidden_tool_count = max(0, len(allowed_tools) - len(visible_tools))
        plan_tool_detail = _format_plan_tool_hints(plan_tool_hints)
        domain_rules_block = render_rules(
            agent="executor",
            tools=allowed_tools,
            message=user_message,
            stage="act",
        )
        system_prompt = (
            _EXECUTOR_TOOL_LOOP_PRINCIPLES
            + f"\n## 当前可用工具（共 {len(allowed_tools)} 个；当前聚焦）\n"
            f"{', '.join(visible_tools) or '(empty)'}"
            + (f"；其余 {hidden_tool_count} 个暂不展开" if hidden_tool_count > 0 else "")
            + (f"\n\n## 场景规则\n{domain_rules_block}\n" if domain_rules_block else "\n")
        )
        user_prompt = (
            f"用户请求:\n{user_message}\n\n"
            f"页面上下文:\n{page_context}\n\n"
            f"规划摘要:\n{plan_summary or '(empty)'}\n\n"
            f"Planner 工具计划（非强制，但用于检查并列动作完整性）:\n{plan_tool_detail}\n\n"
            f"Explorer 证据摘要:\n{evidence_summary or '(empty)'}\n\n"
            f"紧凑证据:\n{compact_evidence}\n\n"
            f"结构化意图:\n"
            f"- action_intent={action_intent}\n"
            f"- selected_job_name={selected_job_name}\n"
            f"- requested_scope={requested_scope}\n\n"
            f"- actor_role={actor_role}\n\n"
            f"已有执行历史:\n{action_history or []}\n\n"
            f"待执行动作:\n{pending_actions or []}\n\n"
            "请直接推进执行；若需要工具就调用工具，否则直接总结。"
        )
        return system_prompt, user_prompt

    async def decide_actions_with_llm(
        self,
        *,
        user_message: str,
        page_context: dict,
        plan_summary: str,
        evidence_summary: str,
        compact_evidence: list[dict],
        action_intent: str | None,
        selected_job_name: str | None,
        requested_scope: str,
        action_history: list[dict],
        pending_actions: list[dict],
        enabled_tools: list[str],
        history_messages: list | None = None,
        actor_role: str = "user",
        plan_tool_hints: list[dict] | None = None,
    ) -> list[dict]:
        """Use LLM to decide whether write actions are needed, and if so, which ones."""
        write_tools = sorted({t for t in enabled_tools if t in CONFIRM_TOOL_NAMES})
        if not write_tools:
            return []
        plan_tool_detail = _format_plan_tool_hints(plan_tool_hints)

        domain_rules_block = render_rules(
            agent="executor",
            tools=enabled_tools,
            message=user_message,
            stage="act",
        )
        decide_system = (
            _EXECUTOR_DECIDE_PRINCIPLES
            + f"\n## 可用写操作工具\n{', '.join(write_tools)}\n"
            + (f"\n## 场景规则\n{domain_rules_block}\n" if domain_rules_block else "")
            + "\n## 输出格式\n"
            '{\n'
            '  "actions": [\n'
            '    {\n'
            '      "tool": "tool_name",\n'
            '      "args": {参数},\n'
            '      "title": "动作标题",\n'
            '      "reason": "执行原因",\n'
            '      "depends_on": [1]\n'
            '    }\n'
            '  ],\n'
            '  "reason": "整体执行理由"\n'
            '}\n'
        )
        result = await self.run_json(
            system_prompt=decide_system,
            user_prompt=(
                f"用户请求:\n{user_message}\n\n"
                f"页面上下文:\n{page_context}\n\n"
                f"规划摘要:\n{plan_summary or '(empty)'}\n\n"
                f"Planner 工具计划（非强制，但用于检查并列动作完整性）:\n{plan_tool_detail}\n\n"
                f"Explorer 证据总结:\n{evidence_summary}\n\n"
                f"紧凑证据:\n{compact_evidence}\n\n"
                f"结构化意图:\n"
                f"- action_intent={action_intent}\n"
                f"- selected_job_name={selected_job_name}\n"
                f"- requested_scope={requested_scope}\n"
                f"- actor_role={actor_role}\n\n"
                f"已有执行历史:\n{action_history or []}\n\n"
                f"待执行动作:\n{pending_actions or []}\n\n"
                "请输出下一批写操作计划。"
            ),
            history_messages=history_messages,
        )

        return self._parse_action_plan(result, write_tools)

    @staticmethod
    def _parse_action_plan(result: dict, write_tools: list[str]) -> list[dict]:
        """Parse LLM output into validated action plan items."""
        if "raw" in result and len(result) == 1:
            logger.warning("Executor action decision was not valid JSON")
            return []

        raw_actions = result.get("actions") or []
        if not isinstance(raw_actions, list):
            return []

        parsed: list[dict] = []
        for item in raw_actions[:6]:
            if not isinstance(item, dict):
                continue
            tool_name = str(item.get("tool") or item.get("action") or "").strip()
            if not tool_name:
                continue
            if tool_name not in CONFIRM_TOOL_NAMES:
                logger.warning("Executor LLM selected non-write tool '%s', rejecting", tool_name)
                continue
            if tool_name not in write_tools:
                logger.warning("Executor LLM selected disabled write tool '%s', rejecting", tool_name)
                continue

            args = item.get("args") or item.get("arguments") or {}
            if not isinstance(args, dict):
                args = {}

            depends_on_raw = item.get("depends_on") or []
            depends_on: list[int] = []
            if isinstance(depends_on_raw, list):
                for dep in depends_on_raw:
                    try:
                        index = int(dep)
                    except (TypeError, ValueError):
                        continue
                    if index > 0:
                        depends_on.append(index)

            parsed.append(
                {
                    "tool_name": tool_name,
                    "tool_args": args,
                    "title": str(item.get("title") or "").strip(),
                    "reason": str(item.get("reason") or "").strip(),
                    "depends_on_indexes": depends_on,
                }
            )

        return parsed

    async def summarize_action(
        self,
        *,
        user_message: str,
        plan_summary: str,
        action_result: dict | None,
        history_messages: list | None = None,
    ) -> RoleExecutionResult:
        if not action_result:
            return RoleExecutionResult(summary="无需执行写操作，继续进入验证。", metadata={})

        summary = await self.run_text(
            system_prompt=(
                "你是 Crater 的 Executor Agent。你负责解释执行阶段的结果。"
                "用中文说明执行结果或为何进入确认。"
            ),
            user_prompt=(
                f"用户请求:\n{user_message}\n\n"
                f"规划摘要:\n{plan_summary}\n\n"
                f"执行结果:\n{action_result}\n\n"
                "请给出执行阶段的简短总结。"
            ),
            history_messages=history_messages,
        )
        return RoleExecutionResult(summary=summary or "已完成执行阶段总结。")
