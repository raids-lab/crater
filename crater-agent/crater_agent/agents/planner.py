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

        result = await self.run_json(
            system_prompt=(
                "你是 Crater 的 Planner Agent。你负责分析用户请求并制定执行计划。\n\n"
                "你的计划会被 Coordinator 协调者审查，由 Explorer（只读工具收集证据）和 "
                "Executor（读+写工具执行操作）分别执行。\n\n"
                "## 规划原则\n"
                "- 先理解用户到底要什么，再规划步骤\n"
                "- steps 描述要做什么，不需要关心谁来执行\n"
                "- 如果已有证据足够回答用户，可以只输出一步「总结回复用户」\n"
                "- candidate_tools 只能从当前可用工具中选择，不能编造工具名\n"
                "- 优先遵守当前页面范围：普通用户页优先用户/当前账户范围，不要主动规划管理员报告或全局巡检工具\n"
                "- 只有当页面就是 admin 场景，或用户明确要求全局/集群/所有用户视角时，才考虑管理员级集群工具\n"
                "- 如果本轮是在追问上一轮回答或质疑上一轮结论，必须结合近期对话上下文，不要脱离上下文重新编例子\n"
                "- 不要过度规划，Coordinator 会在每步执行后审查进展\n"
                "- 若用户在问“为什么失败 / 卡在哪 / 现在正常吗 / 有没有 / 用什么配置 / 能不能提交 / 给我完整配置”，且相关只读工具可用，计划里必须直接安排取证，不要把回答退化成帮助说明或工具清单\n"
                "- 若用户在问排队公平性、优先级或“为什么他比我晚却先跑/这不公平”，计划里优先安排 analyze_queue_status；如果还需要解释账户是否被公平调度降权，再补 check_quota。\n"
                "- 若用户在新建页咨询训练/分布式作业配置，默认至少覆盖这些关键事实桶：配置建议或模板、配额、镜像；只有当用户明确关心当前是否能立即调度或哪种 GPU 现在有空时，再补实时容量\n"
                "- 若用户是在做提交前参数校验，计划里不要为了凑建议而引入异构混搭 GPU 方案；默认优先验证配额、镜像存在性，并给“减少数量”或“整体切换 GPU 型号”的建议。\n"
                "- 若用户明确提到 8 张 A100、分布式训练、完整提交配置这类高资源训练场景，计划中应优先包含 get_job_templates、check_quota，以及 get_realtime_capacity 或 get_resource_recommendation 中与当前问题直接相关的工具；不要只规划页面导航说明\n"
                "- 若用户是在创建 Jupyter / WebIDE，默认最小核验集合是 get_job_templates、check_quota、list_available_images；在这些关键事实未覆盖前，不要把计划收束为“直接回复用户”\n"
                "- 若用户只说“为什么我的作业失败了”且没有指明 jobName，而 list_user_jobs 可用，优先先列出最近失败作业并要求用户确认对象；不要直接规划成对某一条失败作业下结论。\n"
                "- 只有关键事实已经齐全，或当前问题本身就是纯帮助/纯说明时，才允许把计划压缩成一步“总结回复用户”\n\n"
                "请输出 JSON 格式：\n"
                '{\n'
                '  "goal": "本次目标（一句话）",\n'
                '  "steps": ["步骤1", "步骤2", ...],\n'
                '  "candidate_tools": ["tool_name1", "tool_name2"],\n'
                '  "risk": "low|medium|high",\n'
                '  "raw_summary": "面向 Coordinator 的自然语言摘要"\n'
                '}\n\n'
                "使用中文。"
            ),
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
