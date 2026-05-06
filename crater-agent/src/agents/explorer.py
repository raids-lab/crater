"""Explorer agent for multi-agent orchestration."""

from __future__ import annotations

import json
import logging

from agents._domain_rules import render_rules
from agents.base import BaseRoleAgent, RoleExecutionResult
from tools.definitions import READ_ONLY_TOOL_NAMES

logger = logging.getLogger(__name__)


# Stable role principles — apply regardless of available tools or user message.
# Domain-specific advice lives in _domain_rules.py and is injected on demand.
_EXPLORER_ROLE_PRINCIPLES = (
    "你是 Crater 的 Explorer Agent，负责用只读工具收集最小必要证据并总结。\n"
    "你只能使用只读工具，绝对不能执行写操作。可以连续多轮调用工具，每次拿到结果后基于结果判断下一步。\n\n"
    "## 调用原则\n"
    "- 每次调用必须服务于一个未决字段、Planner 计划项或最新证据缺口；相同参数会被系统拦截，"
    "也不要用不同工具/参数反复验证同一已满足事实。\n"
    "- 如果已有工具结果、resume_after_confirmation.result 或 source_turn_context.tool_calls 已覆盖根因/状态/建议，"
    "先消费这些证据；除非有明确新缺口，不要补读同义工具。\n"
    "- 直接工具存在时不要退化成无关列表或泛化建议；核心问题先取一个最能回答的证据，必要时最多补一个旁证。\n"
    "- 证据覆盖根因/状态/建议后立即停止探索并交给 Coordinator finalize。\n"
    "- 如果 Planner 工具计划非空，优先按其工具名、参数和停止条件推进；明显不匹配最新证据时才自行改写。\n\n"
    "## 总结要求\n"
    "- 最终总结要明确：已确认事实、仍缺失的信息、建议下一步。\n"
    "- 保留对象名、状态值、数值、单位、错误码、关键动作短语；不要把 degraded/healthy、Pending/Running、"
    "OOM、FileNotFoundError 等关键信号改写丢失。\n"
)


def _format_compact_evidence(compact_evidence: list[dict]) -> str:
    """Format structured evidence into a readable block for the LLM prompt."""
    if not compact_evidence:
        return "(无)"
    lines = []
    for i, item in enumerate(compact_evidence, 1):
        tool_name = item.get("tool_name", "unknown")
        result = item.get("result", {})
        # Truncate result to avoid excessive prompt size
        result_str = json.dumps(result, ensure_ascii=False, default=str)
        if len(result_str) > 400:
            result_str = result_str[:400] + "...(truncated)"
        lines.append(f"  {i}. [{tool_name}] {result_str}")
    return "\n".join(lines)


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


# Removed: _compact_guidance_for_tools — replaced by render_rules() from
# agents._domain_rules, which centralizes tool/scenario gating across agents.


class ExplorerAgent(BaseRoleAgent):
    def build_tool_loop_prompts(
        self,
        *,
        user_message: str,
        page_context: dict,
        plan_candidate_tools: list[str],
        plan_steps: list[str],
        enabled_tools: list[str],
        evidence_summary: str = "",
        attempted_tool_signatures: list[str] | None = None,
        compact_evidence: list[dict] | None = None,
        plan_tool_hints: list[dict] | None = None,
    ) -> tuple[str, str]:
        allowed = sorted({t for t in enabled_tools if t in READ_ONLY_TOOL_NAMES})
        visible_tools = list(allowed)
        candidates_hint = ""
        valid_candidates: list[str] = []
        if plan_candidate_tools:
            valid_candidates = [t for t in plan_candidate_tools if t in allowed]
            if valid_candidates:
                candidates_hint = f"\nPlanner 推荐的候选工具: {', '.join(valid_candidates)}"
                visible_tools = valid_candidates + [t for t in visible_tools if t not in valid_candidates]
        recent_attempts = list((attempted_tool_signatures or [])[-8:])

        steps_hint = ""
        if plan_steps:
            steps_hint = "\nPlanner 的调查步骤:\n" + "\n".join(
                f"  {i + 1}. {step}" for i, step in enumerate(plan_steps)
            )

        # Format structured evidence so LLM can see actual results, not just summaries
        evidence_detail = _format_compact_evidence(compact_evidence or [])
        plan_tool_detail = _format_plan_tool_hints(plan_tool_hints)

        domain_rules_block = render_rules(
            agent="explorer",
            tools=visible_tools,
            message=user_message,
            stage="observe",
        )
        # Quick triage rules that depend on tool count vs full prompt
        if len(visible_tools) <= 5:
            tail_advice = (
                "- 只调用能回答当前未决字段的只读工具；证据足够后立即总结，不要为了完整性继续横向查询。\n"
                "- 如果只有一个可用工具且它能直接回答当前问题，优先调用它，不要先输出泛化建议。\n"
            )
        else:
            tail_advice = (
                "- 对存储/网络/节点/作业问题，优先调用平台内只读工具。\n"
                "- 如果用户只是确认作业现在是否正常，默认先看 get_job_detail；需要补健康佐证时在 query_job_metrics 和 get_job_events 中二选一，不要两个都查满。\n"
                "- 已有的工具签名见下方，已获取的工具不要重复调用。\n"
            )
        system_prompt = (
            _EXPLORER_ROLE_PRINCIPLES
            + f"\n## 当前可用只读工具\n{', '.join(visible_tools) or '(empty)'}\n"
            + (f"{candidates_hint}\n" if candidates_hint else "")
            + (f"{steps_hint}\n" if steps_hint else "")
            + "\n## 当前轮额外要求\n"
            + tail_advice
            + (f"\n## 场景规则\n{domain_rules_block}\n" if domain_rules_block else "")
        )
        user_prompt = (
            f"用户请求:\n{user_message}\n\n"
            f"页面上下文:\n{page_context}\n\n"
            f"Planner 工具计划（如需取证，优先照此推进）:\n{plan_tool_detail}\n\n"
            f"已获取的工具结果（不要重复调用这些）:\n{evidence_detail}\n\n"
            f"证据文本摘要:\n{evidence_summary or '(empty)'}\n\n"
            f"最近已执行工具签名:\n{recent_attempts or []}\n\n"
            "请调用必要工具；若已有证据足够，就直接给出中文总结。"
        )
        return system_prompt, user_prompt

    async def select_tools_with_llm(
        self,
        *,
        user_message: str,
        page_context: dict,
        plan_candidate_tools: list[str],
        plan_steps: list[str],
        enabled_tools: list[str],
        evidence_summary: str = "",
        attempted_tool_signatures: list[str] | None = None,
        plan_tool_hints: list[dict] | None = None,
        history_messages: list | None = None,
    ) -> list[tuple[str, dict]]:
        """Use LLM to select read-only tools and generate arguments.

        Receives candidate_tools from Planner's PlanOutput as suggestions,
        but the LLM makes the final decision from the enabled_tools set.
        Only READ_ONLY tools are allowed (hard check).
        """
        allowed = sorted({t for t in enabled_tools if t in READ_ONLY_TOOL_NAMES})
        if not allowed:
            logger.warning("No read-only tools available for Explorer")
            return []

        # Build tool catalog description for LLM
        candidates_hint = ""
        valid_candidates: list[str] = []
        if plan_candidate_tools:
            valid_candidates = [t for t in plan_candidate_tools if t in allowed]
            if valid_candidates:
                candidates_hint = f"\nPlanner 推荐的候选工具: {', '.join(valid_candidates)}"
        visible_tools = valid_candidates + [t for t in allowed if t not in valid_candidates]
        recent_attempts = list((attempted_tool_signatures or [])[-8:])

        steps_hint = ""
        if plan_steps:
            steps_hint = "\nPlanner 的调查步骤:\n" + "\n".join(f"  {i+1}. {s}" for i, s in enumerate(plan_steps))
        plan_tool_detail = _format_plan_tool_hints(plan_tool_hints)

        domain_rules_block = render_rules(
            agent="explorer",
            tools=visible_tools,
            message=user_message,
            stage="observe",
        )
        select_system = (
            "你是 Crater 的 Explorer Agent。你的职责是选择只读工具来收集证据。\n"
            '你必须输出 JSON 数组，每个元素是 {"tool": "工具名", "args": {参数}}。'
            "只能从可用工具列表中选择。如果已有证据足够或没有合适工具，输出空数组 []。\n\n"
            f"## 可用只读工具\n{', '.join(visible_tools) or '(empty)'}\n"
            + (f"{candidates_hint}\n" if candidates_hint else "")
            + (f"{steps_hint}\n" if steps_hint else "")
            + f"\n## Planner 工具计划\n{plan_tool_detail}\n\n"
            "## 选择原则\n"
            "- Planner 工具计划非空时，优先选择其中与未决事实直接相关的工具和参数；已满足停止条件时不再选。\n"
            "- 不要重复调用已经以相同参数执行过的工具，除非世界状态明显变化。\n"
            "- 写操作禁止；本阶段只能选只读工具。\n"
            + (f"\n## 场景规则\n{domain_rules_block}\n" if domain_rules_block else "")
            + "\n## 输出示例\n"
            '[{"tool": "get_job_detail", "args": {"job_name": "xxx"}}]\n'
        )
        result = await self.run_json(
            system_prompt=select_system,
            user_prompt=(
                f"用户请求:\n{user_message}\n\n"
                f"页面上下文:\n{page_context}\n\n"
                f"已有证据摘要:\n{evidence_summary or '(empty)'}\n\n"
                f"最近已执行工具签名:\n{recent_attempts or []}\n\n"
                "请选择需要调用的工具。"
            ),
            history_messages=history_messages,
        )

        return self._parse_tool_selections(result, allowed)

    @staticmethod
    def _parse_tool_selections(
        result: dict | list, allowed: set[str] | list[str]
    ) -> list[tuple[str, dict]]:
        """Parse LLM output into validated tool selections."""
        allowed_set = set(allowed)
        selections: list[tuple[str, dict]] = []

        # Handle case where run_json returned a dict with "raw" key
        if isinstance(result, dict):
            if "raw" in result and len(result) == 1:
                logger.warning("Explorer tool selection was not valid JSON")
                return []
            # Maybe the LLM returned a single tool as a dict
            tool_list = [result]
        elif isinstance(result, list):
            tool_list = result
        else:
            return []

        for item in tool_list:
            if not isinstance(item, dict):
                continue
            tool_name = str(item.get("tool") or item.get("name") or "").strip()
            tool_args = item.get("args") or item.get("arguments") or {}
            if not isinstance(tool_args, dict):
                tool_args = {}

            if not tool_name:
                continue

            # Hard check: only read-only tools allowed
            if tool_name not in allowed_set:
                logger.warning(
                    "Explorer LLM selected non-allowed tool '%s', skipping", tool_name
                )
                continue

            # Deduplicate
            if any(t == tool_name and a == tool_args for t, a in selections):
                continue

            selections.append((tool_name, tool_args))

        return selections

    async def summarize_evidence(
        self,
        *,
        user_message: str,
        plan_summary: str,
        evidence: list[dict],
    ) -> RoleExecutionResult:
        summary = await self.run_text(
            system_prompt=(
                "你是 Crater 的 Explore Agent。你只负责整理已获取的只读证据，"
                "不要提出写操作。用中文总结最关键发现。"
            ),
            user_prompt=(
                f"用户请求:\n{user_message}\n\n"
                f"规划摘要:\n{plan_summary}\n\n"
                f"证据:\n{evidence}\n\n"
                "请输出证据总结，突出已确认事实与仍缺失的信息。"
            ),
        )
        return RoleExecutionResult(
            summary=summary or "已整理探索证据。",
            metadata={"evidence_count": len(evidence)},
        )
