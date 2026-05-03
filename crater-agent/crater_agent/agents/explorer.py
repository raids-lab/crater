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
            "- 如果用户只是在确认“当前这个作业现在正常吗、还需不需要处理”，默认先看 get_job_detail；只有在需要补健康佐证时，再二选一补 query_job_metrics 或 get_job_events，不要把 metrics 和 events 都查满。若最终结论健康，探索总结要直接说明作业当前状态、指标是否正常、是否需要额外处理，并补一句具体 watchpoint（如状态变为 Failed/Pending、GPU 利用率明显下跌、显存异常飙升时再排查）。\n"
            "- 如果用户在问“为什么他比我晚排队却先跑了 / 这不公平 / 为什么优先级更低却先调度”，优先用 analyze_queue_status 解释调度原因，再用 check_quota 补充是否存在配额超额或 WeightedFairShare 影响；analyze_queue_status 已经能给出双方优先级、公平调度和配额线索时，不要再额外对两个作业都调用 get_job_detail。\n"
            "- Pending/FailedScheduling 且问题涉及 PVC、StorageClass、配额或队列时，analyze_queue_status 是优先工具；它已经返回 PVC/StorageClass 根因、可用资源或配额排除时，不要再补同义事件查询。\n"
            "- 日志已经明确给出 FileNotFoundError、OOMKilled、ImagePullBackOff 等根因时，应基于日志和最少旁证总结；diagnose_job 只在原始信号不够确定时使用，避免 detail/logs/events/diagnose 全量堆叠。\n"
            "- 如果用户明确要求批量 stop/delete/resubmit，先用只读工具明确对象范围；审计/待处理项工具比普通作业列表更适合批量治理场景。管理员要求“低利用率/闲置/可清理/释放 GPU/都停掉/批量停机”时，优先用 list_audit_items(action_type=\"stop\", handled=\"false\") 获取候选；对象范围明确后停止探索，让 Executor 发起确认。\n"
            "- 如果用户明确要求节点隔离、扩缩容、重启或解除隔离，探索只负责补动作直接相关的最小事实；工作负载目标可来自页面上下文的 workload/name/resource_name。证据足够后停止，不要把写操作替换成口头建议。\n"
            "- 如果已有确认型写操作结果，并且用户要求“执行后/确认后/顺便/然后”检查状态，探索必须优先补目标状态的直接只读证据，再补用户关心的症状证据。Kubernetes 工作负载 scale/restart 后，目标状态证据通常来自工作负载、Pod 或 Endpoints；流量、错误率、延迟等 Prometheus 指标只能作为症状证据，不能单独证明副本数、endpoint 数或调度状态已经落地。\n"
            "- 如果用户询问失败率、哪个账户/团队失败率最高、或主要失败原因统计，优先选择 get_failure_statistics；get_admin_ops_report 只用于运维治理综述、资源浪费、成功/失败/闲置样本汇总，不要用它替代精确失败率排名。\n"
            "- 如果用户说首页 no data、Prometheus 停写、监控看不到数据，且可用工具里同时有 k8s_list_pods 和 k8s_get_events，优先先选 k8s_list_pods 再选 k8s_get_events；如果 Pod 已经显示 Pending/CrashLoopBackOff 或状态不清，再补 k8s_describe_resource。不要跳过 list_pods 直接只看 events 或只看 describe。\n"
            "- 如果可用工具列表里真的包含 prometheus_query，Prometheus 直接核实类问题才先执行 1 条最贴近问题的窄 prometheus_query，必须尽量带上 namespace/job/pod/PVC 等范围；"
            "PVC/存储容量类优先查 `kubelet_volume_stats_used_bytes{...} / kubelet_volume_stats_capacity_bytes{...}` 的逐 PVC 比值，保留 persistentvolumeclaim 标签；不要先查 prometheus_tsdb_*、WAL/head/retention 等间接指标。"
            "服务/Ingress 低流量核实时优先查服务维度请求速率，例如 `sum(rate(nginx_ingress_controller_requests{service=\"...\"}[30m]))`；没有 ingress 指标时再退到 `http_requests_total`。"
            "如果可用工具列表里没有 prometheus_query，就不要在解释里暗示它可用，而应改用 Kubernetes 事件、Pod、describe 或存储工具做最多一个旁证。"
            "只要结果已经能回答，就立刻停止，不要横向扫无关 exporter、TSDB 细项或等价 PromQL 变体。"
            "如果首条 PromQL 返回空 series，不要连续尝试多个同义 PromQL；应总结指标缺口，或在可用时换 Kubernetes 事件/PVC/Pod 只读工具做最多一个旁证。"
            "空 series 只能表示指标未命中，不能写成流量稳定、容量正常或异常已被验证。\n"
            "- 如果 Kubernetes 事件、Pod 状态、describe 或 Prometheus 指标已经出现 no space left on device、PVC 使用率超过 95%、TSDB 写入失败、chunks_head/WAL 写失败，探索总结必须把它写成已验证的 Prometheus 存储/PVC 根因；低风险顺序应是先恢复或扩容存储，再校验 TSDB/数据写入恢复，最后才考虑谨慎重启。不要说“尚未收集到具体错误日志”，也不要把清理用户作业或直接重启作为首要建议。\n"
            "- 如果当前证据显示集群整体 healthy、没有告警、且用户只是在问是否需要马上处理，探索总结要明确整体健康、没有告警、不需要立即处置，并给出后续例行巡检或观察建议。\n"
            "- 如果集群健康报告返回 degraded，要在探索总结里保留 degraded、NotReady/维护节点名、GPU 分配率/利用率、近 1 小时失败作业数，并明确哪些异常需要关注、哪些是计划维护。\n"
            "- 如果作业是 Evicted/DiskPressure/ephemeral-storage，必须优先找事件和日志中是否有 `/tmp`、debug 输出、日志或 checkpoint 写爆节点；总结建议要覆盖关闭 debug 模式、减少日志输出、将输出写入 PVC、清理节点临时盘或设置 ephemeral-storage request/limit 中的相关动作。\n"
            "- 如果节点 reboot 后仍 NotReady，且用户提到 RDMA/GPU 驱动死锁或是否需要机房介入，探索总结必须覆盖：当前 Ready/NotReady、节点上的作业影响、是否应先 cordon、是否评估 drain、带外 reset 与机房人工介入边界。\n"
            "- 如果是两个或多个具名作业指标对比，优先直接查询聚合指标；若一次工具结果已经覆盖所有对象，就不要再对每个对象重复调用同类工具或详情工具。探索总结要保留双方核心数值、差异倍数/方向、哪个更好，以及低效一侧可能的 I/O、dataloader worker、batch size 或同步等待瓶颈。\n"
            "- 指标对比工具结果里如果包含 throughput、samples/sec、吞吐量、显存占用或效率字段，探索总结必须保留这些直接数值；不要只摘要 GPU 利用率。比较倍数要按原始数值估算，避免把吞吐量差异漏掉。\n"
            "- 如果用户在新建页做参数校验或问“这个配置能不能提交”，不要臆造异构混搭 GPU 方案；除非 get_resource_recommendation、模板或平台规则明确支持，否则只建议“减少数量”或“整体切换到另一种单一 GPU 型号”。\n"
            "- 如果 list_user_jobs 已经列出多个失败作业，而用户问题本身没有指明 jobName，应尽快停在澄清上；可以给每个失败作业补一句基于 exit_code / failure_reason 的初步线索，但不要直接把所有作业说成同一个根因。\n"
            "- 多轮追问里，如果用户说“第一个/最新/刚才那个/这个为什么失败”，必须从最近历史或已获取列表中解析具体 jobName，再调用 get_job_detail、get_job_events、get_job_logs 或 diagnose_job 中最直接的只读工具；不要只说“需要获取证据”却不调用工具。\n"
            "- 如果是 Kubernetes rollout / workload 发布卡住，且证据显示镜像拉取错误或发布超时，探索总结里要保留原始错误信号，并明确写出这是 rollout 卡住；建议覆盖修正镜像、重新发布或回滚、确认新 Pod 拉取成功。\n"
            "- 如果用户在新建页询问“有没有某种镜像 / 应该用什么配置 / 这个配置能不能提交 / 最好给我完整提交配置”，至少覆盖与问题直接相关的三类事实：配置建议或模板、配额、可用镜像；只有在用户明确关心是否能立即调度或节点可用性时，再补 get_realtime_capacity。\n"
            "- 如果用户明确提到“8 张 A100 / 8 卡 / 分布式训练 / 完整提交配置”这类高资源训练规划，不要停在模板+配额+镜像三件套；应继续补 get_realtime_capacity，并在可用时补 get_resource_recommendation，用来回答能否立即调度、单节点还是多节点、以及 A100-40G / A100-80G 的取舍。\n"
            "- 对“8 张 A100 / 8 卡 / 分布式训练 / 完整提交配置”这类场景，优先使用 4 个事实源：get_job_templates、check_quota、get_realtime_capacity、get_resource_recommendation。只有当用户明确追问镜像选择、或模板/推荐里没有可直接使用的镜像时，再额外调用 list_available_images。\n"
            "- 如果当前证据表明单个节点已经有 8 张 GPU 全空闲，默认先给“单节点 8 卡 DDP”方案，并补一句“如果你实际需要跨节点分布式，再继续补多节点参数”。\n"
            "- 当示例 entrypoint 使用 `torchrun --nproc_per_node=8` 这类 launcher 时，除非平台模板明确要求手动绑卡，否则不要额外写 `CUDA_VISIBLE_DEVICES`；保留 NCCL/OMP 等必要环境变量即可。\n"
            "- 联网搜索能力由模型内置提供，仅用于厂商文档、公告、CVE 对照，不能替代平台实时状态查询。\n"
            "- 优先使用最少的工具获得足够证据。\n"
            "- 如果 Planner 工具计划非空，优先按其中的工具名、参数和停止条件推进；只有这些参数与用户目标或最新证据明显不匹配时才自行改写。\n"
            "- 下方“已获取的工具结果”列出了之前已经调用过的工具和结果，不要重复调用这些工具。\n"
            "- 如果已获取的结果已经足以回答用户问题，直接给出总结，不要再调工具。\n"
            "- 最终总结要明确：已确认事实、仍缺失的信息、建议下一步。\n"
        )
        user_prompt = (
            f"用户请求:\n{user_message}\n\n"
            f"页面上下文:\n{page_context}\n\n"
            f"Planner 工具计划（如需取证，优先照此推进）:\n{plan_tool_detail}\n\n"
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

        result = await self.run_json(
            system_prompt=(
                "你是 Crater 的 Explorer Agent。你的职责是选择只读工具来收集证据。\n"
                "你必须输出 JSON 数组，每个元素是 {\"tool\": \"工具名\", \"args\": {参数}}。\n"
                "只能从可用工具列表中选择。\n\n"
                f"可用只读工具: {', '.join(visible_tools) or '(empty)'}\n"
                f"{candidates_hint}"
                f"{steps_hint}\n\n"
                f"Planner 工具计划:\n{plan_tool_detail}\n\n"
                "如果已有证据已经足够，请输出空数组 []。\n"
                "如果 Planner 工具计划非空，优先选择其中与当前未决事实直接相关的工具和参数；如果已有证据已满足其停止条件，则不要再选。\n"
                "如果用户只是在确认当前单个作业是否正常、是否需要处理，默认先选 get_job_detail；若需要补健康佐证，只能在 query_job_metrics 和 get_job_events 里再补一个，不要两个都选。健康结论必须直接说明作业状态、指标是否正常、是否需要额外处理，并补一句具体 watchpoint。\n"
                "如果用户在问排队公平性、优先级为什么被别人插队、或直接说“这不公平”，优先选择 analyze_queue_status；若问题涉及账户今日是否超额，再补 check_quota。若 analyze_queue_status 已提供双方优先级和 fairness_notes，就不要再为两个作业分别补 get_job_detail。\n"
                "Pending/FailedScheduling 且问题涉及 PVC、StorageClass、配额或队列时，优先选择 analyze_queue_status；如果它已经返回 PVC/StorageClass 根因、可用资源或配额排除，不要再补同义事件查询。\n"
                "日志已经明确给出 FileNotFoundError、OOMKilled、ImagePullBackOff 等根因时，优先基于日志和最少旁证总结；diagnose_job 只在原始信号不够确定时使用。\n"
                "如果用户明确要求批量 stop/delete/resubmit，优先选择能明确待处理对象或候选对象范围的只读工具；管理员要求“低利用率/闲置/可清理/释放 GPU/都停掉/批量停机”时，优先选择 list_audit_items(action_type=\"stop\", handled=\"false\")；对象范围已经明确时输出空数组，让 Executor 进入确认。\n"
                "如果用户明确要求节点隔离、扩缩容、重启或解除隔离，只选择与目标对象健康/风险直接相关的最小只读证据；工作负载目标可来自页面上下文的 workload/name/resource_name。不要扩散成全量巡检。\n"
                "如果已有确认型写操作结果，并且用户要求执行后检查状态，优先选择能直接验证目标状态的只读工具，再选择用户额外关心的症状工具。Kubernetes 工作负载 scale/restart 后，Pod/Endpoints/工作负载状态是目标状态证据；Prometheus 流量或错误率是症状证据，不能单独替代目标状态核验。\n"
                "如果用户询问失败率、哪个账户/团队失败率最高、或主要失败原因统计，优先选择 get_failure_statistics；get_admin_ops_report 只用于运维治理综述、资源浪费、成功/失败/闲置样本汇总，不要用它替代精确失败率排名。\n"
                "如果用户说首页 no data、Prometheus 停写、监控看不到数据，且可用工具里同时有 k8s_list_pods 和 k8s_get_events，优先先选 k8s_list_pods 再选 k8s_get_events；如果 Pod 已经显示 Pending/CrashLoopBackOff 或状态不清，再补 k8s_describe_resource。不要跳过 list_pods 直接只看 events 或只看 describe。\n"
                "只有可用工具列表里真的包含 prometheus_query 时，Prometheus 直接核实类问题才优先选择 1 条最贴近问题的 prometheus_query，必须尽量带上 namespace/job/pod/PVC 等范围；PVC/存储容量类优先选择 kubelet_volume_stats_used_bytes / kubelet_volume_stats_capacity_bytes 的逐 PVC 比值；服务/Ingress 低流量优先选择服务维度请求速率，如 nginx_ingress_controller_requests，没有 ingress 指标时再退到 http_requests_total。不要先选 prometheus_tsdb_*、WAL/head/retention 等间接指标。如果可用工具列表里没有 prometheus_query，禁止选择或描述该工具不可用，应改用 Kubernetes 事件、Pod、describe 或存储工具做最多一个旁证。只要首条查询已经返回能回答问题的 series，就不要再继续选择等价或旁路 PromQL。若首条返回空 series，不要连续选择多个同义 PromQL；应停止并总结指标缺口，或在可用时改选 Kubernetes 事件/PVC/Pod 只读工具做最多一个旁证。空 series 不能被当成稳定或正常的证明。\n"
                "如果 Kubernetes 事件、Pod 状态、describe 或 Prometheus 指标已经出现 no space left on device、PVC 使用率超过 95%、TSDB 写入失败、chunks_head/WAL 写失败，必须停止继续横向探索并总结为已验证的 Prometheus 存储/PVC 根因；建议顺序是先恢复或扩容存储，再校验 TSDB/数据写入恢复，最后才考虑谨慎重启。不要说证据不足，也不要把直接重启放在第一步。\n"
                "如果用户只是在确认集群整体是否 healthy、是否需要立即处理，而 get_cluster_health_report 可用，得到 healthy + 无告警后就可以停止；总结要直接说明整体健康、没有告警、不需要立即处置，并给出后续例行巡检或观察建议。\n"
                "如果 get_cluster_health_report 返回 degraded，输出必须保留 degraded、NotReady/维护节点名、GPU 分配率/利用率、近 1 小时失败作业数，并区分紧急关注项和计划维护项。\n"
                "如果作业是 Evicted/DiskPressure/ephemeral-storage，优先选 get_job_events；若日志工具可用且还没确认是否写爆 /tmp，补 get_job_logs 或 diagnose_job。总结建议要包含关闭 debug 模式、减少日志输出、输出写入 PVC、磁盘清理或 ephemeral-storage request/limit 等相关动作。\n"
                "如果节点 reboot 后仍 NotReady，且用户提到 RDMA/GPU 驱动死锁或机房介入，优先用节点和 Pod 只读工具确认影响；总结必须覆盖先 cordon、评估 drain、确认节点上的作业影响、带外 reset/机房人工介入边界。\n"
                "如果是两个或多个具名作业指标对比，优先直接查询聚合指标；若一次工具结果已经覆盖所有对象，就不要再对每个对象重复调用同类工具或详情工具。总结必须保留双方核心数值、差异倍数/方向、winner，以及低效一侧可能的 I/O/dataloader/batch size/同步等待瓶颈。\n"
                "指标对比工具结果里如果包含 throughput、samples/sec、吞吐量、显存占用或效率字段，最终总结必须保留这些直接数值；不要只摘要 GPU 利用率。比较倍数要按原始数值估算，避免漏掉吞吐量差异。\n"
                "如果是 Kubernetes rollout / workload 发布卡住，且 rollout 或事件里出现镜像拉取错误或发布超时，回答里要保留原始错误信号，并明确写出 rollout 卡住；建议覆盖修正镜像、重新发布或回滚、确认新 Pod 拉取成功。\n"
                "如果用户在新建页询问镜像、推荐配置、提交可行性或完整提交配置，默认优先覆盖配置建议/模板、配额、可用镜像三类事实；若问题未要求立即调度，不要额外选择 get_realtime_capacity。\n"
                "如果用户是在做提交前参数校验，不要产出异构混搭 GPU 的替代方案，除非证据明确显示平台支持这种混搭；默认优先给“减少数量”或“整体切换到另一种单一 GPU 型号”的建议。\n"
                "多轮追问里，如果用户说“第一个/最新/刚才那个/这个为什么失败”，必须从最近历史或已获取列表中解析具体 jobName，再调用 get_job_detail、get_job_events、get_job_logs 或 diagnose_job 中最直接的只读工具；不要只输出空数组或只说需要继续取证。\n"
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
