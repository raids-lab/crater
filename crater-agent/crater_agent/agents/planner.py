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
                "- tool_hints 用于给下游 agent 一个可执行但非强制的工具计划；每项只允许引用当前可用工具，参数只填你能从用户请求、页面上下文或已有证据中确定的字段，不确定就留空或省略\n"
                "- tool_hints 每项建议包含 tool、args、purpose、stop_condition；不要把场景 id、测试期望或评分标准写进计划\n"
                "- 优先遵守当前页面范围：普通用户页优先用户/当前账户范围，不要主动规划管理员报告或全局巡检工具\n"
                "- 只有当页面就是 admin 场景，或用户明确要求全局/集群/所有用户视角时，才考虑管理员级集群工具\n"
                "- 如果本轮是在追问上一轮回答或质疑上一轮结论，必须结合近期对话上下文，不要脱离上下文重新编例子\n"
                "- 不要过度规划，Coordinator 会在每步执行后审查进展\n"
                "- 若用户在问“为什么失败 / 卡在哪 / 现在正常吗 / 有没有 / 用什么配置 / 能不能提交 / 给我完整配置”，且相关只读工具可用，计划里必须直接安排取证，不要把回答退化成帮助说明或工具清单\n"
                "- 若用户在问排队公平性、优先级或“为什么他比我晚却先跑/这不公平”，计划里优先安排 analyze_queue_status；如果还需要解释账户是否被公平调度降权，再补 check_quota。\n"
                "- 若 Pending/FailedScheduling 明显涉及 PVC、StorageClass 或队列原因，analyze_queue_status 往往能一次覆盖调度原因、PVC 状态和资源排除；只有该结果缺少事件原因时才补 get_job_events，不要为了完整性重复取证。\n"
                "- 若日志已明确给出 FileNotFoundError、OOMKilled、ImagePullBackOff 等直接根因，diagnose_job 只作为不确定时的辅助；不要把它和 detail/logs/events 全部查满。\n"
                "- 若用户明确要求停止/删除/重提作业，规划应先定位目标对象；单对象必须绑定 jobName，全量/批量操作必须先用审计项或候选列表确认范围，再交给 Executor 选择确认工具。\n"
                "- 管理员要求“低利用率/闲置/可清理/释放 GPU/都停掉/批量停机”这类治理动作时，优先规划 list_audit_items(action_type=\"stop\", handled=\"false\") 获取候选，再由 Executor 使用 batch_stop_jobs 发起确认；不要只给口头建议。\n"
                "- 若用户要求 Kubernetes 扩缩容、重启、节点隔离或解除隔离，规划应包含最小只读核验和后续确认动作，但具体写工具和参数由 Executor 根据工具描述决定；不要把动作降级成自然语言建议。页面上下文里的 workload/name/resource_name 可以作为工作负载目标线索。\n"
                "- 写操作后的验证计划要同时覆盖两类事实：1) 被改变对象的目标状态是否已经落地；2) 用户额外关心的症状或指标是否正常。对于 Kubernetes 工作负载扩缩容/重启，优先把 Deployment/StatefulSet、Pod 或 Endpoints 这类直接状态证据作为目标状态验证，再用 Prometheus/日志等验证流量或症状；不要只用单一指标替代目标状态核验。\n"
                "- 若用户在新建页咨询训练/分布式作业配置，默认至少覆盖这些关键事实桶：配置建议或模板、配额、镜像；只有当用户明确关心当前是否能立即调度或哪种 GPU 现在有空时，再补实时容量\n"
                "- 若用户是在做提交前参数校验，计划里不要为了凑建议而引入异构混搭 GPU 方案；默认优先验证配额、镜像存在性，并给“减少数量”或“整体切换 GPU 型号”的建议。\n"
                "- 若用户明确提到 8 张 A100、分布式训练、完整提交配置这类高资源训练场景，计划中应优先包含 get_job_templates、check_quota，以及 get_realtime_capacity 或 get_resource_recommendation 中与当前问题直接相关的工具；不要只规划页面导航说明\n"
                "- 若用户是在创建 Jupyter / WebIDE，默认最小核验集合是 get_job_templates、check_quota、list_available_images；在这些关键事实未覆盖前，不要把计划收束为“直接回复用户”\n"
                "- 若用户只说“为什么我的作业失败了”且没有指明 jobName，而 list_user_jobs 可用，优先先列出最近失败作业并要求用户确认对象；不要直接规划成对某一条失败作业下结论。\n"
                "- 若用户询问失败率、哪个账户/团队失败率最高、主要失败原因统计，优先把 get_failure_statistics 作为候选工具；只有用户要运维治理周报、资源浪费综述或成功/失败/闲置样本分析时，才优先 get_admin_ops_report。\n"
                "- 两个或多个具名作业做利用率、显存、吞吐或效率对比时，优先规划能直接返回聚合指标的 query_job_metrics；只有用户同时要求确认资源规格、镜像、节点或作业身份时，才补 get_job_detail 或 list_user_jobs。\n"
                "- Prometheus 直接核实类问题应在 tool_hints 中给出 1 条最贴近问题的窄查询，尽量带上 namespace/job/pod/PVC 等范围，并使用 query_type=\"instant\" 或明确 range 窗口；"
                "存储容量/PVC 使用率优先规划 `kubelet_volume_stats_used_bytes{...} / kubelet_volume_stats_capacity_bytes{...}` 的逐 PVC 比值，并保留 persistentvolumeclaim 标签；"
                "服务/Ingress 低流量核实时优先规划服务维度请求速率，例如 `sum(rate(nginx_ingress_controller_requests{service=\"...\"}[30m]))`，没有 ingress 指标时再退到服务自身的 `http_requests_total`；"
                "不要先查 prometheus_tsdb_*、WAL/head/retention 等间接指标，也不要用不可定位的全局 sum 聚合吞掉对象标签。停止条件是得到能回答问题的 series，或确认该指标未命中；"
                "空 series 不能被规划为“稳定/正常”的证明，也不要继续规划多个语义等价 PromQL 变体。\n"
                "- 只有关键事实已经齐全，或当前问题本身就是纯帮助/纯说明时，才允许把计划压缩成一步“总结回复用户”\n\n"
                "请输出 JSON 格式：\n"
                '{\n'
                '  "goal": "本次目标（一句话）",\n'
                '  "steps": ["步骤1", "步骤2", ...],\n'
                '  "candidate_tools": ["tool_name1", "tool_name2"],\n'
                '  "tool_hints": [\n'
                '    {"tool": "tool_name1", "args": {"key": "value"}, "purpose": "为什么需要这个工具", "stop_condition": "什么情况下停止"}\n'
                '  ],\n'
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
