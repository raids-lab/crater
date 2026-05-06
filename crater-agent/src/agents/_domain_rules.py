"""Shared domain-specific guidance rules for MAS sub-agents.

Each rule is conditionally injected into a sub-agent prompt only when its
preconditions match (relevant tools available, scenario keywords present,
or stage-specific context). This keeps individual agent prompts focused on
their role and avoids the bloat of duplicating the same guidance across
planner/explorer/executor/coordinator.

Usage:
    from agents._domain_rules import render_rules
    rules_block = render_rules(
        agent="explorer",
        tools=visible_tools,
        message=user_message,
        stage="observe",
    )
    system_prompt = ROLE_HEADER + ROLE_PRINCIPLES + rules_block

Design notes:
- Rules are *advisory* and meant to bias the LLM toward sound choices on
  recurring scenarios. They do NOT replace the orchestrator's hard safety
  rules (permission checks, dedup, awaiting_confirmation, companion taint).
- Each rule has a unique ID for traceability. Logging the activated rule IDs
  helps debug why an LLM picked a particular tool path.
- Adding a rule: append a DomainRule entry; do NOT inline new guidance into
  individual agent system prompts.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Callable


@dataclass(frozen=True)
class DomainRule:
    """A single piece of conditionally-injected guidance.

    Attributes:
        id: Stable identifier for traceability and rule deduplication.
        text: Single-line guidance string injected into the system prompt.
        applies_to_agents: Which agent roles this rule is relevant for.
        requires_tools: Rule fires only when ALL listed tools are available.
        any_of_tools: Rule fires when AT LEAST ONE listed tool is available.
        message_match: Optional callable(normalized_message) -> bool for
            scenario-specific gating.
        stages: Optional set of stages where the rule applies (e.g. {"observe"}).
    """

    id: str
    text: str
    applies_to_agents: frozenset[str] = field(default_factory=frozenset)
    requires_tools: frozenset[str] = field(default_factory=frozenset)
    any_of_tools: frozenset[str] = field(default_factory=frozenset)
    message_match: Callable[[str], bool] | None = None
    stages: frozenset[str] = field(default_factory=frozenset)


def _msg_contains(*tokens: str) -> Callable[[str], bool]:
    def _check(message: str) -> bool:
        return any(token in message for token in tokens)

    return _check


# ---------------------------------------------------------------------------
# Rule catalog — single source of truth for cross-agent domain guidance.
# Group by intent so it's easy to audit which rules govern which scenarios.
# ---------------------------------------------------------------------------

DOMAIN_RULES: tuple[DomainRule, ...] = (
    # --- Semantic safety: label vs taint ---
    DomainRule(
        id="label_taint_isolation",
        text=(
            "节点隔离场景：label 只是审计标记，NoSchedule taint 才阻止新 Pod 调度。"
            "用户同时要求隔离标签和 NoSchedule 时，k8s_label_node 与 k8s_taint_node 必须并列发起，"
            "depends_on 均为空；只发 label 会遗漏阻止调度的核心动作。"
        ),
        applies_to_agents=frozenset({"executor", "planner"}),
        requires_tools=frozenset({"k8s_label_node", "k8s_taint_node"}),
        message_match=_msg_contains("隔离", "noschedule", "taint", "isolate"),
    ),
    # --- Semantic safety: write-after verification ---
    DomainRule(
        id="post_write_target_state_first",
        text=(
            "写操作完成后用户要求执行后检查时，先用只读工具核验目标对象状态（Pod/Endpoints/工作负载副本），"
            "再补症状证据（Prometheus 指标）；症状指标不能单独证明状态已落地。"
        ),
        applies_to_agents=frozenset({"explorer", "planner", "executor"}),
        any_of_tools=frozenset({"k8s_list_pods", "k8s_get_endpoints", "k8s_describe_resource"}),
    ),
    # --- Fact validation: empty Prometheus series ---
    DomainRule(
        id="prometheus_empty_series",
        text=(
            "prometheus_query 空 series 只能表示指标未命中，不能写成流量稳定/容量正常/异常已验证。"
            "首条空 series 后不要重复同义 PromQL，转用 Kubernetes 事件/PVC/Pod 工具做最多一个旁证。"
        ),
        applies_to_agents=frozenset({"explorer", "planner"}),
        any_of_tools=frozenset({"prometheus_query"}),
    ),
    DomainRule(
        id="prometheus_narrow_query",
        text=(
            "prometheus_query 优先选 1 条最贴近问题的窄查询，带上 namespace/job/pod/PVC 等范围；"
            "具体 PromQL 表达式参考工具描述，不要使用全局 sum 聚合吞掉对象标签。"
        ),
        applies_to_agents=frozenset({"explorer", "planner"}),
        any_of_tools=frozenset({"prometheus_query"}),
    ),
    # --- Fact validation: Pending != Running ---
    DomainRule(
        id="pending_not_running",
        text=(
            '新建作业是 Pending 时只能说「已提交/已创建且当前 Pending」，不能写成 Running 或运行正常；'
            "若证据解释了 Pending 原因（排队/资源不足等），直接说出根因。"
        ),
        applies_to_agents=frozenset({"explorer", "executor", "coordinator"}),
    ),
    # --- Diagnostic priority: queue fairness ---
    DomainRule(
        id="queue_fairness",
        text=(
            '排队公平性、优先级或「为什么他先跑」类问题：优先用 analyze_queue_status 解释调度原因，'
            "需要时再补 check_quota；analyze_queue_status 已给出双方优先级和 fairness_notes 时不要再分别 get_job_detail。"
        ),
        applies_to_agents=frozenset({"explorer", "planner"}),
        any_of_tools=frozenset({"analyze_queue_status"}),
        message_match=_msg_contains("公平", "优先级", "排队", "为什么他", "插队"),
    ),
    # --- Diagnostic priority: clear root cause ---
    DomainRule(
        id="clear_root_cause_short_circuit",
        text=(
            "日志已明确给出 FileNotFoundError、OOMKilled、ImagePullBackOff 等直接根因时，"
            "基于日志和最少旁证总结，不要把 detail/logs/events/diagnose_job 全部查满。"
        ),
        applies_to_agents=frozenset({"explorer", "planner"}),
        any_of_tools=frozenset({"get_job_logs", "diagnose_job"}),
    ),
    # --- Storage root cause sequence ---
    DomainRule(
        id="storage_root_cause_sequence",
        text=(
            "已确认根因（no space left on device、PVC 使用率超 95%、TSDB 写失败、chunks_head/WAL 写失败）后，"
            "停止横向探索，总结为已验证根因。建议顺序：先恢复或扩容存储 → 校验数据写入 → 谨慎重启。"
            '不要把「清理用户作业」或「直接重启」作为首要建议。'
        ),
        applies_to_agents=frozenset({"explorer", "planner"}),
    ),
    # --- Cluster health framing ---
    DomainRule(
        id="cluster_health_summary",
        text=(
            "集群 healthy + 无告警时直接说不需要立即处置，给出后续观察建议。"
            "degraded 时保留 NotReady/维护节点名、GPU 分配率/利用率、近 1 小时失败作业数，"
            "区分紧急关注项和计划维护项。"
        ),
        applies_to_agents=frozenset({"explorer"}),
        any_of_tools=frozenset({"get_cluster_health_report", "get_cluster_health_overview", "get_health_overview"}),
    ),
    # --- Submission-config minimal facts ---
    DomainRule(
        id="submission_config_facts",
        text=(
            "新建页询问镜像/推荐配置/提交可行性时，覆盖三类事实：配置建议或模板、配额、可用镜像。"
            "可用镜像必须由 list_available_images 核实，不能用 get_resource_recommendation 替代。"
            "只有用户明确关心立即调度时才补 get_realtime_capacity。"
        ),
        applies_to_agents=frozenset({"explorer", "planner"}),
        any_of_tools=frozenset({"get_job_templates", "list_available_images", "check_quota"}),
    ),
    # --- Bulk admin ops ---
    DomainRule(
        id="bulk_admin_stop",
        text=(
            '管理员"低利用率/闲置/可清理/释放 GPU/批量停机"治理：先用 list_audit_items(action_type="stop", handled="false") '
            "获取候选，对象范围明确后让 Executor 用 batch_stop_jobs 一次发起确认；不要逐个停或停留在口头建议。"
        ),
        applies_to_agents=frozenset({"explorer", "planner", "executor"}),
        any_of_tools=frozenset({"list_audit_items", "batch_stop_jobs"}),
        message_match=_msg_contains("低利用率", "闲置", "可清理", "释放 gpu", "批量停"),
    ),
    # --- Multi-job metric comparison ---
    DomainRule(
        id="metric_comparison_preserve_numbers",
        text=(
            "两个或多个具名作业指标对比：优先一次聚合查询覆盖所有对象，不要重复对每个对象调用同类工具。"
            "总结必须保留双方核心数值、差异倍数/方向、winner，以及低效一侧可能的 I/O/dataloader/batch size/同步等待瓶颈。"
            "throughput/samples/sec/吞吐量/显存占用字段必须保留原始数值，不要只摘要 GPU 利用率。"
        ),
        applies_to_agents=frozenset({"explorer", "planner"}),
        any_of_tools=frozenset({"query_job_metrics"}),
    ),
    # --- Rollout stuck preservation ---
    DomainRule(
        id="rollout_stuck_signal",
        text=(
            "Kubernetes rollout / workload 发布卡住且镜像拉取错误或发布超时时，"
            '保留原始错误信号，明确写出「rollout 卡住」。建议覆盖修镜像/重发布/回滚/确认新 Pod 拉取成功。'
        ),
        applies_to_agents=frozenset({"explorer", "planner"}),
        any_of_tools=frozenset({"k8s_rollout_status"}),
    ),
    # --- Failure-rate vs ops-report ---
    DomainRule(
        id="failure_rate_vs_ops_report",
        text=(
            "失败率/账户失败率/主要失败原因统计 → get_failure_statistics。"
            "get_admin_ops_report 仅用于运维治理周报、资源浪费综述或成功/失败/闲置样本汇总。"
        ),
        applies_to_agents=frozenset({"planner", "explorer"}),
        any_of_tools=frozenset({"get_failure_statistics", "get_admin_ops_report"}),
        message_match=_msg_contains("失败率", "失败原因"),
    ),
    # --- Followup object resolution ---
    DomainRule(
        id="followup_object_resolution",
        text=(
            '多轮追问"第一个/最新/刚才那个/这个为什么失败"时，从最近历史或已获取列表解析具体 jobName，'
            "再调用 get_job_detail/get_job_events/get_job_logs/diagnose_job 中最直接的只读工具；"
            '不要只输出"需要获取证据"。'
        ),
        applies_to_agents=frozenset({"explorer", "planner"}),
        any_of_tools=frozenset({"get_job_detail", "get_job_events", "get_job_logs", "diagnose_job"}),
        message_match=_msg_contains("第一个", "最新", "刚才", "这个为什么"),
    ),
    # --- No-data investigation order ---
    DomainRule(
        id="no_data_pod_first",
        text=(
            "首页 no data / Prometheus 停写 / 监控看不到数据时，先 k8s_list_pods 再 k8s_get_events；"
            "Pod Pending/CrashLoopBackOff 或状态不清时再补 k8s_describe_resource。"
            "不要跳过 list_pods 直接看 events 或 describe。"
        ),
        applies_to_agents=frozenset({"explorer", "planner"}),
        requires_tools=frozenset({"k8s_list_pods", "k8s_get_events"}),
        message_match=_msg_contains("no data", "停写", "看不到数据", "监控空"),
    ),
    # --- GPU heterogeneous mix avoidance ---
    DomainRule(
        id="no_gpu_heterogeneous_mix",
        text=(
            "提交前参数校验时不要臆造异构混搭 GPU 方案，除非证据明确支持；"
            '默认建议「减少数量」或「整体切换到另一种单一 GPU 型号」。'
        ),
        applies_to_agents=frozenset({"explorer", "planner"}),
    ),
)


def _matches(rule: DomainRule, *, tools: set[str], message: str) -> bool:
    if rule.requires_tools and not rule.requires_tools.issubset(tools):
        return False
    if rule.any_of_tools and not (rule.any_of_tools & tools):
        return False
    if rule.message_match and not rule.message_match(message):
        return False
    return True


def render_rules(
    *,
    agent: str,
    tools: list[str] | set[str],
    message: str = "",
    stage: str = "",
) -> str:
    """Build the dynamic rule block for an agent's system prompt.

    Returns an empty string if no rules match (caller can skip the section).
    """
    tool_set = {str(t) for t in (tools or [])}
    normalized = str(message or "").lower()
    matched: list[str] = []
    for rule in DOMAIN_RULES:
        if rule.applies_to_agents and agent not in rule.applies_to_agents:
            continue
        if rule.stages and stage and stage not in rule.stages:
            continue
        if not _matches(rule, tools=tool_set, message=normalized):
            continue
        matched.append(f"- {rule.text}")
    return "\n".join(matched)


def matched_rule_ids(
    *,
    agent: str,
    tools: list[str] | set[str],
    message: str = "",
    stage: str = "",
) -> list[str]:
    """Return rule IDs that would fire — useful for tracing/logging."""
    tool_set = {str(t) for t in (tools or [])}
    normalized = str(message or "").lower()
    ids: list[str] = []
    for rule in DOMAIN_RULES:
        if rule.applies_to_agents and agent not in rule.applies_to_agents:
            continue
        if rule.stages and stage and stage not in rule.stages:
            continue
        if not _matches(rule, tools=tool_set, message=normalized):
            continue
        ids.append(rule.id)
    return ids
