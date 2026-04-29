"""Executor agent for multi-agent orchestration."""

from __future__ import annotations

import logging

from crater_agent.agents.base import BaseRoleAgent, RoleExecutionResult
from crater_agent.tools.definitions import CONFIRM_TOOL_NAMES

logger = logging.getLogger(__name__)


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
    ) -> tuple[str, str]:
        allowed_tools = sorted(set(enabled_tools))
        visible_tools = allowed_tools[:10]
        hidden_tool_count = max(0, len(allowed_tools) - len(visible_tools))
        system_prompt = (
            "你是 Crater 的 Executor Agent。你的职责是根据用户请求和现有证据，直接推进下一步工具执行。\n"
            "你可以连续多轮调用工具，并且每拿到一次工具结果，都要继续基于结果判断下一步。\n"
            "如果需要进一步确认目标对象，可以先调用只读工具；如果用户已经明确要求操作，可以调用写工具。\n"
            "当你认为无需再调用工具时，请直接给出中文执行总结。\n\n"
            f"可用工具共 {len(allowed_tools)} 个；当前聚焦: {', '.join(visible_tools) or '(empty)'}"
            + (f"；其余 {hidden_tool_count} 个暂不展开" if hidden_tool_count > 0 else "")
            + "\n\n"
            "执行要求：\n"
            "- 当前用户角色不是管理员时，不要尝试执行管理员专属写操作；应说明权限限制并建议联系管理员。\n"
            "- 纯查询/诊断请求不要擅自执行写操作。\n"
            "- 若仅需补充事实证据，优先继续用只读工具，不要直接进入写操作。\n"
            "- 写操作前应确保目标对象明确；如果不明确，先补最小只读核验。\n"
            "- 用户自述、页面状态或上一轮口头结论都不是已验证事实；对 restart/stop/scale/uncordon/create 这类写操作，应先补一到两条与当前动作直接相关的事实。\n"
            "- 每次只读调用都必须服务于一个未决字段或风险点；不要为了求稳扩散到无关 read。\n"
            "- 对 create_jupyter_job / create_webide_job：如果模板默认值、配额、镜像匹配关系还没核实，不要猜 CPU/内存/GPU 型号，优先补 get_job_templates / check_quota / list_available_images 这类相关证据；若用户给的参数已基本齐全，最小合格核验就是这三项，不要省略 get_job_templates。\n"
            "- 对 uncordon_node / restart_workload / k8s_scale_workload：至少先补一次与节点或工作负载健康直接相关的核验，不要只根据用户一句“已经恢复”就进入确认。\n"
            "- run_kubectl / execute_admin_command 属于受控高风险动作：应先说明触发原因，并等待确认流，不得构造任意 shell 文本。\n"
            "- 如果同一轮明确需要多个互相独立的确认型写动作，可以一次性调用这些写工具生成多张确认卡；系统会统一暂停等待用户确认。\n"
            "- 这不是强制多确认：只有一个必要写动作时，正常生成一张确认卡即可。\n"
            "- 动作存在明确依赖（例如先创建再修改同一新对象）时，不要并列发确认；先推进当前可执行的最小动作，等待后续轮次。\n"
            "- 避免重复调用相同参数的工具，除非世界状态明显变化。\n"
        )
        user_prompt = (
            f"用户请求:\n{user_message}\n\n"
            f"页面上下文:\n{page_context}\n\n"
            f"规划摘要:\n{plan_summary or '(empty)'}\n\n"
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
    ) -> list[dict]:
        """Use LLM to decide whether write actions are needed, and if so, which ones."""
        write_tools = sorted({t for t in enabled_tools if t in CONFIRM_TOOL_NAMES})
        if not write_tools:
            return []

        result = await self.run_json(
            system_prompt=(
                "你是 Crater 的 Executor Agent。你的职责是基于用户请求和证据决定下一批原子写操作。\n"
                "你只能输出写操作计划，不能执行工具，也不要总结。\n\n"
                f"可用写操作工具: {', '.join(write_tools)}\n\n"
                "输出 JSON:\n"
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
                '}\n\n'
                "注意：\n"
                "- actions 里每个元素都必须是原子动作。\n"
                "- depends_on 使用 1-based 下标，表示依赖本次输出中更早的动作；没有依赖就输出空数组。\n"
                "- 当前用户角色不是管理员时，不要输出管理员专属写操作；应返回空 actions。\n"
                "- 只有用户明确要求操作时才选择写操作。\n"
                "- 纯诊断/查询请求不应产生写操作。\n"
                '- "帮我看看这个作业" ≠ "帮我停止这个作业"。\n'
                "- 如果用户明确要求多个独立写动作（例如同时 label 和 taint 同一节点），可以在 actions 中同时列出这些动作，depends_on 为空。\n"
                "- 这不是强制多动作：只有一个必要写动作时，actions 只输出一个元素。\n"
                "- 如果动作有先后依赖，不要并列输出为独立动作；用 depends_on 标出依赖，或只输出当前可执行的最小动作。\n"
                "- 不要把用户自述状态当作已验证证据；如果写操作还依赖关键事实未核实，应先返回空 actions，把机会留给前面的只读探索。\n"
                "- create_jupyter_job / create_webide_job 在模板默认值、配额或镜像匹配关系未核实时，不要凭空猜 CPU/内存/GPU 型号后直接创建。\n"
                "- uncordon_node / restart_workload / k8s_scale_workload 在节点或工作负载健康未核实时，不要直接输出写动作。\n"
                "- 当选择 run_kubectl / execute_admin_command 时，reason 必须体现“结构化工具和只读证据不足，必须执行受控命令”。\n"
                "- 如果已经有 pending_actions，不要重复生成等价动作。\n"
                "- 如果 requested_scope=all 且证据里已经列出候选对象，可以输出多条动作。"
            ),
            user_prompt=(
                f"用户请求:\n{user_message}\n\n"
                f"页面上下文:\n{page_context}\n\n"
                f"规划摘要:\n{plan_summary or '(empty)'}\n\n"
                f"Explorer 证据总结:\n{evidence_summary}\n\n"
                f"紧凑证据:\n{compact_evidence}\n\n"
                f"结构化意图:\n"
                f"- action_intent={action_intent}\n"
                f"- selected_job_name={selected_job_name}\n"
                f"- requested_scope={requested_scope}\n\n"
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
