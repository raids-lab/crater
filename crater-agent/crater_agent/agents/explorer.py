"""Explorer agent for multi-agent orchestration."""

from __future__ import annotations

import json
import logging

from crater_agent.agents.base import BaseRoleAgent, RoleExecutionResult
from crater_agent.tools.definitions import READ_ONLY_TOOL_NAMES

logger = logging.getLogger(__name__)


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

        system_prompt = (
            "你是 Crater 的 Explorer Agent。你的职责是通过只读工具主动收集证据，并在证据足够时直接给出探索总结。\n"
            "你可以连续多轮调用工具。每次拿到工具结果后，你必须继续基于结果判断下一步，而不是机械重复同一个工具。\n"
            "如果已有证据足够，请不要再调工具，直接给出中文总结。\n"
            "你只能使用只读工具，绝对不能执行写操作。\n\n"
            f"可用只读工具: {', '.join(visible_tools) or '(empty)'}\n"
            f"{candidates_hint}"
            f"{steps_hint}\n\n"
            "工作要求：\n"
            "- 对存储/网络/节点/作业问题，优先调用平台内只读工具。\n"
            "- 如果用户只是在确认“当前这个作业现在正常吗、还需不需要处理”，默认先看 get_job_detail；只有在需要补健康佐证时，再二选一补 query_job_metrics 或 get_job_events，不要把 metrics 和 events 都查满。若最终结论健康，探索总结里尽量直接写成“作业 <name> 当前运行正常、指标正常、无需额外处理”。不要把“无需额外处理”改写成“无需您立即处理”“无需操作”“无需干预”等同义说法。建议动作尽量原样包含“继续观察”或“如果只是确认状态，则无需额外操作”，并补一句具体 watchpoint（如状态变为 Failed/Pending、GPU 利用率明显下跌、显存异常飙升时再排查）。\n"
            "- 如果用户在问“为什么他比我晚排队却先跑了 / 这不公平 / 为什么优先级更低却先调度”，优先用 analyze_queue_status 解释调度原因，再用 check_quota 补充是否存在配额超额或 WeightedFairShare 影响；analyze_queue_status 已经能给出双方优先级、公平调度和配额线索时，不要再额外对两个作业都调用 get_job_detail。\n"
            "- 如果用户要求“用 Prometheus 指标确认 Prometheus 存储是不是快满了”这类直接指标核实，先执行 1 条最贴近问题的 prometheus_query（例如 PVC used/capacity 比值）；只要结果已经能回答，就立刻停止，不要再横向扫 node filesystem、TSDB 细项或等价 PromQL 变体。\n"
            "- 如果当前证据显示集群整体 healthy、没有告警、且用户只是在问是否需要马上处理，探索总结里尽量原样包含“healthy”“没有告警”“无需立即处理”“暂无紧急动作”“继续例行巡检”。\n"
            "- 如果用户在新建页做参数校验或问“这个配置能不能提交”，不要臆造异构混搭 GPU 方案；除非 get_resource_recommendation、模板或平台规则明确支持，否则只建议“减少数量”或“整体切换到另一种单一 GPU 型号”。\n"
            "- 如果 list_user_jobs 已经列出多个失败作业，而用户问题本身没有指明 jobName，应尽快停在澄清上；可以给每个失败作业补一句基于 exit_code / failure_reason 的初步线索，但不要直接把所有作业说成同一个根因。\n"
            "- 如果是 Kubernetes rollout / workload 发布卡住，且证据显示 ErrImagePull、manifest unknown 或 ProgressDeadlineExceeded，探索总结里要保留这些关键词，并明确写出这是 `rollout` 卡住；建议优先直接列出“修正镜像 tag”“重新发布镜像”“确认新 Pod 拉取成功”，并补一句“镜像修复后 rollout 会继续推进/恢复”；必要时再补“回滚到上一版本”。\n"
            "- 如果用户在新建页询问“有没有某种镜像 / 应该用什么配置 / 这个配置能不能提交 / 最好给我完整提交配置”，至少覆盖与问题直接相关的三类事实：配置建议或模板、配额、可用镜像；只有在用户明确关心是否能立即调度或节点可用性时，再补 get_realtime_capacity。\n"
            "- 如果用户明确提到“8 张 A100 / 8 卡 / 分布式训练 / 完整提交配置”这类高资源训练规划，不要停在模板+配额+镜像三件套；应继续补 get_realtime_capacity，并在可用时补 get_resource_recommendation，用来回答能否立即调度、单节点还是多节点、以及 A100-40G / A100-80G 的取舍。\n"
            "- 对“8 张 A100 / 8 卡 / 分布式训练 / 完整提交配置”这类场景，优先使用 4 个事实源：get_job_templates、check_quota、get_realtime_capacity、get_resource_recommendation。只有当用户明确追问镜像选择、或模板/推荐里没有可直接使用的镜像时，再额外调用 list_available_images。\n"
            "- 如果当前证据表明单个节点已经有 8 张 GPU 全空闲，默认先给“单节点 8 卡 DDP”方案，并补一句“如果你实际需要跨节点分布式，再继续补多节点参数”。\n"
            "- 当示例 entrypoint 使用 `torchrun --nproc_per_node=8` 这类 launcher 时，除非平台模板明确要求手动绑卡，否则不要额外写 `CUDA_VISIBLE_DEVICES`；保留 NCCL/OMP 等必要环境变量即可。\n"
            "- 联网搜索能力由模型内置提供，仅用于厂商文档、公告、CVE 对照，不能替代平台实时状态查询。\n"
            "- 优先使用最少的工具获得足够证据。\n"
            "- 下方“已获取的工具结果”列出了之前已经调用过的工具和结果，不要重复调用这些工具。\n"
            "- 如果已获取的结果已经足以回答用户问题，直接给出总结，不要再调工具。\n"
            "- 最终总结要明确：已确认事实、仍缺失的信息、建议下一步。\n"
        )
        user_prompt = (
            f"用户请求:\n{user_message}\n\n"
            f"页面上下文:\n{page_context}\n\n"
            f"已获取的工具结果（不要重复调用这些）:\n{evidence_detail}\n\n"
            f"证据文本摘要:\n{evidence_summary or '(empty)'}\n\n"
            f"最近已执行工具签名:\n{recent_attempts or []}\n\n"
            "请开始探索；若需要更多信息就调用工具，否则直接总结。"
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

        result = await self.run_json(
            system_prompt=(
                "你是 Crater 的 Explorer Agent。你的职责是选择只读工具来收集证据。\n"
                "你必须输出 JSON 数组，每个元素是 {\"tool\": \"工具名\", \"args\": {参数}}。\n"
                "只能从可用工具列表中选择。\n\n"
                f"可用只读工具: {', '.join(visible_tools) or '(empty)'}\n"
                f"{candidates_hint}"
                f"{steps_hint}\n\n"
                "如果已有证据已经足够，请输出空数组 []。\n"
                "如果用户只是在确认当前单个作业是否正常、是否需要处理，默认先选 get_job_detail；若需要补健康佐证，只能在 query_job_metrics 和 get_job_events 里再补一个，不要两个都选。健康结论里尽量原样包含“运行正常、指标正常、无需额外处理”“如果只是确认状态，则无需额外操作”“继续观察”，并补一句具体 watchpoint；不要把“无需额外处理”改写成“无需操作”“无需您立即处理”等同义表达。\n"
                "如果用户在问排队公平性、优先级为什么被别人插队、或直接说“这不公平”，优先选择 analyze_queue_status；若问题涉及账户今日是否超额，再补 check_quota。若 analyze_queue_status 已提供双方优先级和 fairness_notes，就不要再为两个作业分别补 get_job_detail。\n"
                "如果用户要求“用 Prometheus 指标确认 Prometheus 存储是不是快满了”，优先只选择 1 条最贴近问题的 prometheus_query；只要首条查询已经返回 PVC 使用率，就不要再继续选择等价或旁路 PromQL。\n"
                "如果用户只是在确认集群整体是否 healthy、是否需要立即处理，而 get_cluster_health_report 可用，得到 healthy + 无告警后就可以停止；总结里尽量保留“healthy”“没有告警”“无需立即处理”“暂无紧急动作”“继续例行巡检”。\n"
                "如果是 Kubernetes rollout / workload 发布卡住，且 rollout 或事件里出现 ErrImagePull、manifest unknown、ProgressDeadlineExceeded，回答里尽量保留这些关键词，并明确写出 `rollout` 卡住；建议优先直接写成“修正镜像 tag”“重新发布镜像”“确认新 Pod 拉取成功”，并补一句“镜像修复后 rollout 会继续推进/恢复”，必要时再补“回滚到上一版本”。\n"
                "如果用户在新建页询问镜像、推荐配置、提交可行性或完整提交配置，默认优先覆盖配置建议/模板、配额、可用镜像三类事实；若问题未要求立即调度，不要额外选择 get_realtime_capacity。\n"
                "如果用户是在做提交前参数校验，不要产出异构混搭 GPU 的替代方案，除非证据明确显示平台支持这种混搭；默认优先给“减少数量”或“整体切换到另一种单一 GPU 型号”的建议。\n"
                "如果用户明确提到“8 张 A100 / 8 卡 / 分布式训练 / 完整提交配置”这类高资源训练规划，应继续补 get_realtime_capacity，并在可用时补 get_resource_recommendation；不要只停在模板、配额、镜像三项。\n"
                "对“8 张 A100 / 8 卡 / 分布式训练 / 完整提交配置”这类场景，优先选择 get_job_templates、check_quota、get_realtime_capacity、get_resource_recommendation 这 4 个工具；只有当用户明确追问镜像选择，或前述结果里没有足够镜像信息时，才再加 list_available_images。\n"
                "如果是创建 Jupyter / WebIDE 这类轻量交互作业，优先把 get_job_templates、check_quota、list_available_images 作为最小核验集合；除非用户追问容量，不要主动选 get_realtime_capacity。\n"
                "优先平台内只读工具；联网搜索由模型内置提供，仅用于外部文档/CVE 对照。\n"
                "不要重复调用已经以相同参数执行过的工具，除非页面上下文或世界状态明确变化。\n\n"
                "输出格式示例:\n"
                '[{"tool": "get_job_detail", "args": {"job_name": "xxx"}}]\n\n'
                "如果无需调用工具，输出空数组 []。"
            ),
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
