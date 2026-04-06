"""Explorer agent for multi-agent orchestration."""

from __future__ import annotations

import logging

from crater_agent.agents.base import BaseRoleAgent, RoleExecutionResult
from crater_agent.tools.definitions import READ_ONLY_TOOL_NAMES

logger = logging.getLogger(__name__)


class ExplorerAgent(BaseRoleAgent):
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
        if plan_candidate_tools:
            valid_candidates = [t for t in plan_candidate_tools if t in allowed]
            if valid_candidates:
                candidates_hint = f"\nPlanner 推荐的候选工具: {', '.join(valid_candidates)}"

        steps_hint = ""
        if plan_steps:
            steps_hint = "\nPlanner 的调查步骤:\n" + "\n".join(f"  {i+1}. {s}" for i, s in enumerate(plan_steps))

        result = await self.run_json(
            system_prompt=(
                "你是 Crater 的 Explorer Agent。你的职责是选择只读工具来收集证据。\n"
                "你必须输出 JSON 数组，每个元素是 {\"tool\": \"工具名\", \"args\": {参数}}。\n"
                "最多选择 3 个工具。只能从可用工具列表中选择。\n\n"
                f"可用只读工具: {', '.join(allowed)}\n"
                f"{candidates_hint}"
                f"{steps_hint}\n\n"
                "如果已有证据已经足够，请输出空数组 []。\n"
                "不要重复调用已经以相同参数执行过的工具，除非页面上下文或世界状态明确变化。\n\n"
                "输出格式示例:\n"
                '[{"tool": "get_job_detail", "args": {"job_name": "xxx"}}]\n\n'
                "如果无需调用工具，输出空数组 []。"
            ),
            user_prompt=(
                f"用户请求:\n{user_message}\n\n"
                f"页面上下文:\n{page_context}\n\n"
                f"已有证据摘要:\n{evidence_summary or '(empty)'}\n\n"
                f"本轮已执行工具签名:\n{attempted_tool_signatures or []}\n\n"
                "请选择需要调用的工具。"
            ),
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

        for item in tool_list[:3]:
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
