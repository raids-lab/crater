"""Tool definitions for Crater Agent.

Each tool is declared here with its schema. Actual execution happens via
HTTP callback to the Go backend's /v1/agent/tools/execute endpoint.
"""

from typing import Any, Dict, List, Optional

from langchain_core.tools import tool


# ============================================================
# A. Diagnosis Tools (auto-execute)
# ============================================================


@tool
def get_job_detail(job_name: str) -> dict:
    """获取指定作业的详细信息，包括状态、资源配置、时间线、节点信息、退出码。

    Args:
        job_name: 作业的系统唯一名（对应后端 Job.JobName，如 sg-xxx / jpt-xxx）
    """
    pass  # Executed by Go backend



@tool
def get_job_logs(job_name: str, tail: int = 200, keyword: Optional[str] = None) -> dict:
    """获取作业容器的日志输出。可选按关键词过滤。

    Args:
        job_name: 作业的系统唯一名（Job.JobName）
        tail: 返回最后 N 行日志，默认 200
        keyword: 可选，日志中搜索的关键词（正则表达式）
    """
    pass


@tool
def diagnose_job(job_name: str) -> dict:
    """运行规则诊断引擎，对作业进行故障分类和根因分析。
    返回结构化诊断结果，包括故障类别、严重程度、置信度、解决方案。

    Args:
        job_name: 作业的系统唯一名（Job.JobName）
    """
    pass


@tool
def get_diagnostic_context(job_name: str) -> dict:
    """获取完整的诊断上下文：元数据 + 事件 + 终止状态 + 性能指标 + 调度信息。
    信息量较大，适合需要深度分析时使用。

    Args:
        job_name: 作业的系统唯一名（Job.JobName）
    """
    pass


@tool
def search_similar_failures(
    job_name: str, days: int = 30, limit: int = 5
) -> dict:
    """搜索与指定作业相似的历史失败案例（基于退出码、镜像、故障类别匹配）。

    Args:
        job_name: 当前作业的系统唯一名（Job.JobName，用于提取特征进行匹配）
        days: 搜索最近 N 天，默认 30
        limit: 最多返回 N 条，默认 5
    """
    pass


# ============================================================
# B. Metrics / Queue Analysis Tools (auto-execute)
# ============================================================


@tool
def query_job_metrics(
    job_name: str,
    metrics: Optional[List[str]] = None,
    time_range: str = "last_2h",
) -> dict:
    """查询作业的 GPU/CPU/Memory 性能指标。返回聚合值（avg, max, stddev）。

    Args:
        job_name: 作业的系统唯一名（Job.JobName）
        metrics: 要查询的指标列表，可选值: gpu_util, gpu_mem, cpu_usage, mem_usage。默认全部
        time_range: 时间范围，如 "last_1h", "last_2h", "last_24h"
    """
    pass


@tool
def analyze_queue_status(job_name: str) -> dict:
    """分析指定 Pending 作业的排队原因，综合调度事件、配额和实时容量给出解释。

    Args:
        job_name: 作业的系统唯一名（Job.JobName）
    """
    pass


@tool
def get_realtime_capacity() -> dict:
    """查询当前集群实时容量概览，返回节点级资源使用和状态摘要。"""
    pass


@tool
def list_available_images(
    job_type: Optional[str] = None,
    keyword: Optional[str] = None,
    limit: int = 20,
) -> dict:
    """列出当前用户可见的镜像列表，可按任务类型或关键词过滤。

    Args:
        job_type: 可选，镜像任务类型，如 custom、pytorch、jupyter、training
        keyword: 可选，按镜像链接、描述、标签搜索
        limit: 最多返回 N 条，默认 20
    """
    pass


@tool
def list_cuda_base_images(limit: int = 20) -> dict:
    """列出平台登记的 CUDA 基础镜像。

    Args:
        limit: 最多返回 N 条，默认 20
    """
    pass


@tool
def list_available_gpu_models(limit: int = 20) -> dict:
    """列出当前集群可见的 GPU 资源型号及总量/已用/剩余摘要。

    Args:
        limit: 最多返回 N 条，默认 20
    """
    pass


@tool
def recommend_training_images(
    task_description: str,
    framework: Optional[str] = None,
    limit: int = 5,
) -> dict:
    """基于当前真实可见镜像，为训练任务推荐候选镜像。

    Args:
        task_description: 任务描述，如“意图识别小模型训练”
        framework: 可选，框架偏好，如 pytorch / tensorflow
        limit: 最多返回 N 条推荐，默认 5
    """
    pass


@tool
def get_health_overview(days: int = 7) -> dict:
    """获取当前用户在当前账户下的作业健康概览：总作业数、失败数、运行中数、失败率等。

    Args:
        days: 统计最近 N 天，默认 7
    """
    pass


@tool
def list_user_jobs(
    statuses: Optional[List[str]] = None,
    job_types: Optional[List[str]] = None,
    days: int = 30,
    limit: int = 20,
) -> dict:
    """列出当前用户在当前账户下的作业，支持按状态和作业类型筛选。

    Args:
        statuses: 可选，按状态过滤，如 running、failed、deleted、completed
        job_types: 可选，按作业类型过滤，如 custom、jupyter、webide、pytorch
        days: 查询最近 N 天，默认 30
        limit: 最多返回 N 条，默认 20
    """
    pass



@tool
def list_cluster_jobs(
    statuses: Optional[List[str]] = None,
    job_types: Optional[List[str]] = None,
    days: int = 7,
    limit: int = 30,
) -> dict:
    """列出管理员视角下整个集群最近的作业，支持按状态和作业类型筛选。

    Args:
        statuses: 可选，按状态过滤，如 running、failed、pending
        job_types: 可选，按作业类型过滤，如 custom、jupyter、webide、pytorch
        days: 查询最近 N 天，默认 7
        limit: 最多返回 N 条，默认 30
    """
    pass


@tool
def list_cluster_nodes() -> dict:
    """列出集群节点摘要，包括节点状态、工作负载数、供应商和节点数量概览。
    适合管理员判断是否存在资源紧张、节点异常或调度热点。
    """
    pass


@tool
def check_quota(account_id: Optional[int] = None) -> dict:
    """查看当前账户的资源配额与使用情况。

    返回可能包含配额上限、已使用量、活跃作业数等信息。
    若某项字段缺失，说明平台暂未提供该维度数据，不代表无限制或零使用。

    Args:
        account_id: 账户 ID，默认使用当前用户的账户
    """
    pass


# ============================================================
# B2. Newly Added Read-only Tools (auto-execute)
# ============================================================


@tool
def detect_idle_jobs(gpu_threshold: int = 5, hours: int = 24) -> dict:
    """检测当前运行中但 GPU 利用率较低的闲置作业。
    可用于发现资源浪费、建议用户释放 GPU。

    Args:
        gpu_threshold: GPU 利用率阈值（百分比），低于该值视为闲置，默认 5
        hours: 回溯时间窗口（小时），默认 24
    """
    pass


@tool
def get_job_templates(limit: int = 20) -> dict:
    """列出平台可用的作业模板，包括模板名称、描述、文档和配置内容。

    Args:
        limit: 最多返回 N 条，默认 20
    """
    pass


@tool
def get_failure_statistics(days: int = 7, limit: int = 10) -> dict:
    """统计近期失败作业的故障类别分布，返回按频次排序的故障分类及示例作业。

    Args:
        days: 统计最近 N 天，默认 7
        limit: 最多返回 N 个分类，默认 10
    """
    pass


@tool
def get_cluster_health_report(days: int = 7) -> dict:
    """获取集群整体健康报告，聚合作业概览、节点容量、GPU 型号分布和故障统计。
    需要管理员权限。

    Args:
        days: 统计最近 N 天，默认 7
    """
    pass


@tool
def get_resource_recommendation(
    task_description: str,
    framework: Optional[str] = None,
    gpu_required: bool = True,
) -> dict:
    """根据任务描述和框架需求，基于集群当前可用资源推荐 CPU/GPU/内存配置。

    Args:
        task_description: 任务描述，如"BERT 意图识别微调"
        framework: 可选，框架偏好，如 pytorch / tensorflow
        gpu_required: 是否需要 GPU，默认 True
    """
    pass


@tool
def get_node_detail(node_name: str) -> dict:
    """获取单个集群节点的详细信息，包括硬件配置、资源用量和 GPU 信息。
    需要管理员权限。

    Args:
        node_name: 节点名称
    """
    pass


@tool
def get_admin_ops_report(
    days: int = 7,
    success_limit: int = 5,
    failure_limit: int = 5,
    gpu_threshold: int = 5,
    idle_hours: int = 24,
) -> dict:
    """获取管理员视角的智能运维分析报告，聚合成功/失败/闲置作业及资源差异。
    需要管理员权限。

    Args:
        days: 回看天数，默认 7
        success_limit: 展示的成功作业样本数，默认 5
        failure_limit: 展示的失败作业样本数，默认 5
        gpu_threshold: 低利用率阈值，默认 5
        idle_hours: 闲置判定窗口（小时），默认 24
    """
    pass


# ============================================================
# B3. Audit / AIOps Read-only Tools (admin, auto-execute)
# ============================================================


@tool
def get_latest_audit_report(report_type: str = "gpu_audit") -> str:
    """获取最近一次运维审计报告摘要。返回报告ID、状态、汇总数据（扫描数、检测数、GPU浪费小时数、建议分类）。
    仅管理员可用。"""
    pass


@tool
def list_audit_items(
    report_id: str = "",
    action_type: str = "",
    severity: str = "",
    handled: str = "",
    limit: int = 20,
) -> str:
    """筛选审计条目列表。可按 action_type(stop/notify/downscale)、severity(critical/warning/info)、handled(true/false) 过滤。
    返回作业名、用户、GPU利用率、建议操作等。仅管理员可用。"""
    pass


@tool
def save_audit_report(
    report_type: str = "gpu_audit",
    trigger_source: str = "cron",
    summary: str = "{}",
    items: str = "[]",
) -> str:
    """保存审计报告到数据库。由 Pipeline 内部调用，不直接暴露给用户。"""
    pass


# ============================================================
# B4. Storage / Network / Retrieval Tools (admin, auto-execute)
# ============================================================


@tool
def list_storage_pvcs(
    namespace: Optional[str] = None,
    status: Optional[str] = None,
    limit: int = 30,
) -> dict:
    """列出存储 PVC 摘要（容量、状态、命名空间、绑定关系）。
    仅管理员可用。

    Args:
        namespace: 可选，按命名空间过滤
        status: 可选，按 PVC 状态过滤（如 Bound / Pending）
        limit: 最多返回 N 条，默认 30
    """
    pass


@tool
def get_pvc_detail(pvc_name: str, namespace: Optional[str] = None) -> dict:
    """获取单个 PVC 详情，包括容量、访问模式、存储类、挂载引用等。
    仅管理员可用。

    Args:
        pvc_name: PVC 名称
        namespace: 可选，PVC 所在命名空间
    """
    pass


@tool
def get_pvc_events(
    pvc_name: str,
    namespace: Optional[str] = None,
    limit: int = 50,
) -> dict:
    """获取 PVC 相关事件（调度、挂载、绑定、扩容失败等）。
    仅管理员可用。

    Args:
        pvc_name: PVC 名称
        namespace: 可选，PVC 所在命名空间
        limit: 最多返回 N 条事件，默认 50
    """
    pass


@tool
def inspect_job_storage(job_name: str) -> dict:
    """检查指定作业的存储挂载与卷声明情况，辅助定位挂载失败或存储不可用问题。
    仅管理员可用。

    Args:
        job_name: 作业系统唯一名（Job.JobName）
    """
    pass


@tool
def get_storage_capacity_overview(namespace: Optional[str] = None) -> dict:
    """获取存储容量总览，包含已用/可用/异常 PVC 摘要。
    仅管理员可用。

    Args:
        namespace: 可选，按命名空间聚合
    """
    pass


@tool
def get_node_network_summary(
    node_name: Optional[str] = None,
    include_addresses: bool = False,
    limit: Optional[int] = None,
) -> dict:
    """获取节点网络状态摘要（节点网络可用性、关键告警、基础网卡状态等）。
    仅管理员可用。

    Args:
        node_name: 可选，指定节点；为空则返回集群级网络摘要
        include_addresses: 是否附带节点地址列表
        limit: 可选，返回节点数量上限
    """
    pass


@tool
def diagnose_distributed_job_network(
    job_name: Optional[str] = None,
    tail_lines: int = 200,
    max_log_matches: int = 50,
    keyword: Optional[str] = None,
    lookback_hours: Optional[int] = None,
    limit: Optional[int] = None,
) -> dict:
    """诊断分布式作业网络通信问题（如 NCCL/RDMA 相关异常）。
    仅管理员可用。

    Args:
        job_name: 作业系统唯一名（Job.JobName）；为空时返回近期分布式作业的聚合诊断
        tail_lines: 日志回看行数，默认 200
        max_log_matches: 最多保留多少条日志命中
        keyword: 可选，额外关键词/正则过滤
        lookback_hours: 聚合诊断时的时间窗口（小时）
        limit: 聚合诊断时最多检查多少个作业
    """
    pass


@tool
def web_search(
    query: str,
    limit: int = 5,
) -> dict:
    """用 DuckDuckGo 检索外部公开信息，返回搜索结果列表（标题、摘要、链接）。
    仅管理员可用。

    Args:
        query: 检索关键词
        limit: 最多返回 N 条结果，默认 5
    """
    pass


@tool
def fetch_url(
    url: str,
    max_chars: int = 4000,
) -> dict:
    """抓取指定 URL 的页面正文（去除脚本/样式后的可读文本）。
    配合 web_search 使用：先搜索拿到链接，再用此工具读取具体内容。
    仅支持 platform 白名单域名。仅管理员可用。

    Args:
        url: 要读取的完整 URL（https://...）
        max_chars: 返回正文的最大字符数，默认 4000
    """
    pass



@tool
def get_agent_runtime_summary() -> dict:
    """返回 agent 侧运行时配置摘要（backend/k8s/prometheus、本地工具沙箱规则等）。

    该工具为平台无关工具，本地执行，不依赖 Crater 后端。
    """
    pass


@tool
def k8s_list_nodes(
    label_selector: Optional[str] = None,
    field_selector: Optional[str] = None,
    limit: int = 100,
) -> dict:
    """通过 agent 侧 kubeconfig 直接列出 Kubernetes 节点摘要。

    适用于节点 NotReady、调度异常、节点筛选等排查。
    """
    pass


@tool
def k8s_list_pods(
    namespace: Optional[str] = None,
    label_selector: Optional[str] = None,
    field_selector: Optional[str] = None,
    node_name: Optional[str] = None,
    limit: int = 200,
) -> dict:
    """通过 agent 侧 kubeconfig 直接列出 Pod 摘要。

    适用于按 namespace / label / node 聚合排查 Pod 与作业状态。
    """
    pass


@tool
def k8s_get_events(
    namespace: Optional[str] = None,
    field_selector: Optional[str] = None,
    limit: int = 100,
) -> dict:
    """通过 agent 侧 kubeconfig 查询 Kubernetes 事件。

    适用于镜像拉取失败、调度失败、节点异常、PVC 挂载失败等问题排查。
    """
    pass


@tool
def k8s_describe_resource(
    kind: str,
    name: str,
    namespace: Optional[str] = None,
) -> dict:
    """通过 agent 侧 kubeconfig 执行 kubectl describe。

    适用于节点、Pod、PVC、Deployment、DaemonSet 等资源的详细排查。
    """
    pass


@tool
def k8s_get_pod_logs(
    pod_name: str,
    namespace: Optional[str] = None,
    container: Optional[str] = None,
    tail: int = 200,
    since_seconds: Optional[int] = None,
    previous: bool = False,
) -> dict:
    """通过 agent 侧 kubeconfig 读取 Pod 日志。

    适用于镜像拉取失败后容器错误、CrashLoopBackOff、监控组件异常、训练容器报错等排查。
    """
    pass


@tool
def prometheus_query(
    query: str,
    query_type: str = "instant",
    time: Optional[str] = None,
    start: Optional[str] = None,
    end: Optional[str] = None,
    step: Optional[str] = None,
    timeout_seconds: Optional[int] = None,
    max_series: int = 20,
    max_points_per_series: int = 120,
) -> dict:
    """通过 agent 侧 Prometheus API 直接执行 instant/range 查询。

    适用于 GPU/节点/Pod 指标、Prometheus 自身健康、node-exporter 重启频繁等问题排查。
    """
    pass


@tool
def harbor_check(
    server: Optional[str] = None,
    image: Optional[str] = None,
    repository: Optional[str] = None,
    reference: Optional[str] = None,
    timeout_seconds: Optional[int] = None,
) -> dict:
    """检查 Harbor/OCI Registry 健康状态，以及指定镜像是否存在。

    适用于镜像拉取失败、构建镜像失败、Harbor 迁移后地址异常等问题排查。
    """
    pass


# ============================================================
# B6. Infrastructure Query Tools (admin, local kubectl)
# ============================================================


@tool
def k8s_get_service(
    namespace: Optional[str] = None,
    label_selector: Optional[str] = None,
    field_selector: Optional[str] = None,
    name: Optional[str] = None,
    limit: int = 50,
) -> dict:
    """通过 agent 侧 kubeconfig 查询 Kubernetes Service 资源。

    适用于 Jupyter/WebIDE 不可达时检查 Service 是否存在、端口映射是否正确、ClusterIP 是否分配。
    仅管理员可用。

    Args:
        namespace: 可选，按命名空间过滤；为空则查询默认命名空间
        label_selector: 可选，标签选择器
        field_selector: 可选，字段选择器
        name: 可选，指定单个 Service 名称
        limit: 最多返回 N 条，默认 50
    """
    pass


@tool
def k8s_get_endpoints(
    namespace: Optional[str] = None,
    name: Optional[str] = None,
    limit: int = 50,
) -> dict:
    """通过 agent 侧 kubeconfig 查询 Kubernetes Endpoints 资源。

    适用于排查 Service 后端是否有就绪 Pod、地址是否正确、端口是否匹配。
    仅管理员可用。

    Args:
        namespace: 可选，按命名空间过滤
        name: 可选，指定单个 Endpoints 名称（通常与 Service 同名）
        limit: 最多返回 N 条，默认 50
    """
    pass


@tool
def k8s_get_ingress(
    namespace: Optional[str] = None,
    label_selector: Optional[str] = None,
    name: Optional[str] = None,
    limit: int = 50,
) -> dict:
    """通过 agent 侧 kubeconfig 查询 Kubernetes Ingress 资源。

    适用于 Jupyter/WebIDE 外部访问不可达时检查 Ingress 规则、TLS 配置、后端 Service 绑定。
    仅管理员可用。

    Args:
        namespace: 可选，按命名空间过滤
        label_selector: 可选，标签选择器
        name: 可选，指定单个 Ingress 名称
        limit: 最多返回 N 条，默认 50
    """
    pass


@tool
def get_volcano_queue_state(
    name: Optional[str] = None,
    limit: int = 50,
) -> dict:
    """查询 Volcano Queue 调度队列状态（集群级资源，无需指定命名空间）。

    适用于排查队列公平性、容量分配、Pending 作业排队原因、资源配额是否匹配。
    仅管理员可用。

    Args:
        name: 可选，指定单个队列名称；为空则列出所有队列
        limit: 最多返回 N 条，默认 50
    """
    pass


@tool
def k8s_get_configmap(
    namespace: Optional[str] = None,
    name: Optional[str] = None,
    label_selector: Optional[str] = None,
    limit: int = 50,
) -> dict:
    """通过 agent 侧 kubeconfig 查询 Kubernetes ConfigMap 资源。

    适用于检查服务配置、构建参数、代理配置、镜像源配置等。
    仅管理员可用。

    Args:
        namespace: 可选，按命名空间过滤
        name: 可选，指定单个 ConfigMap 名称
        label_selector: 可选，标签选择器
        limit: 最多返回 N 条，默认 50
    """
    pass


@tool
def k8s_get_networkpolicy(
    namespace: Optional[str] = None,
    name: Optional[str] = None,
    limit: int = 50,
) -> dict:
    """通过 agent 侧 kubeconfig 查询 Kubernetes NetworkPolicy 资源。

    适用于排查出网限制（如构建 Pod 无法访问外部 apt 源）、Pod 间通信隔离（如 NCCL 端口阻断）等网络策略问题。
    仅管理员可用。

    Args:
        namespace: 可选，按命名空间过滤
        name: 可选，指定单个 NetworkPolicy 名称
        limit: 最多返回 N 条，默认 50
    """
    pass


@tool
def aggregate_image_pull_errors(
    namespace: Optional[str] = None,
    time_window_minutes: int = 60,
    limit: int = 100,
) -> dict:
    """聚合集群中镜像拉取失败事件，按镜像名分组统计。

    适用于 Harbor 故障、镜像仓库不可用、网络问题导致的批量镜像拉取失败排查。
    仅管理员可用。

    Args:
        namespace: 可选，按命名空间过滤；为空则查询全部命名空间
        time_window_minutes: 时间窗口（分钟），默认 60
        limit: 最多返回 N 条聚合结果，默认 100
    """
    pass


@tool
def detect_zombie_jobs(
    running_hours_threshold: int = 72,
    limit: int = 50,
) -> dict:
    """检测可能已僵死但仍处于 Running 状态的作业 Pod。

    基于运行时长判定：超过阈值小时数的 Running Pod 视为疑似僵尸作业。
    适用于批量清理长期占用资源的无效作业。
    仅管理员可用。

    Args:
        running_hours_threshold: 运行时长阈值（小时），默认 72
        limit: 最多返回 N 条，默认 50
    """
    pass


@tool
def get_ddp_rank_mapping(job_name: str) -> dict:
    """获取分布式训练作业（Volcano Job）的 rank→Pod 映射关系。

    适用于 DDP/NCCL 通信超时时快速定位哪个 rank 在哪个节点、IP 地址是否正确。
    仅管理员可用。

    Args:
        job_name: Volcano Job 名称（volcano.sh/job-name 标签值）
    """
    pass


@tool
def get_node_kernel_diagnostics(
    node_name: str,
    dmesg_lines: int = 200,
    check_d_state: bool = True,
) -> dict:
    """通过 kubectl debug 获取节点内核级诊断信息。

    收集 dmesg 中 GPU/NCCL/RDMA 相关日志、D 状态进程、负载均值等。
    适用于 RDMA 驱动死锁、GPU Xid 错误、节点内核卡死等硬件级故障排查。
    仅管理员可用。需要集群支持 kubectl debug node（K8s 1.23+）。

    Args:
        node_name: 目标节点名称
        dmesg_lines: dmesg 回看行数，默认 200
        check_d_state: 是否检查 D 状态进程，默认 True
    """
    pass


@tool
def get_rdma_interface_status(node_name: str) -> dict:
    """通过 kubectl debug 获取节点 RDMA/InfiniBand 接口状态。

    收集 ibstat、rdma link、IB 端口状态、相关内核模块加载情况。
    适用于 RDMA 通信异常、IB 端口 down、驱动未加载等智算网络故障排查。
    仅管理员可用。需要集群支持 kubectl debug node（K8s 1.23+）。

    Args:
        node_name: 目标节点名称
    """
    pass


# ============================================================
# B7. Extended K8s Read-only Tools (admin, local kubectl)
# ============================================================


@tool
def k8s_top_nodes(node_name: Optional[str] = None) -> dict:
    """查询节点实时 CPU/Memory 使用率（需要 metrics-server 已部署）。
    仅管理员可用。

    Args:
        node_name: 可选，指定单个节点；为空则返回全部节点
    """
    pass


@tool
def k8s_top_pods(
    namespace: Optional[str] = None,
    label_selector: Optional[str] = None,
    limit: int = 50,
) -> dict:
    """查询 Pod 实时 CPU/Memory 使用率（需要 metrics-server 已部署）。
    仅管理员可用。

    Args:
        namespace: 可选，按命名空间过滤
        label_selector: 可选，按标签过滤
        limit: 最多返回 N 条，默认 50
    """
    pass


@tool
def k8s_rollout_status(
    kind: str,
    name: str,
    namespace: Optional[str] = None,
) -> dict:
    """查看 Deployment/StatefulSet/DaemonSet 的滚动发布状态。
    仅管理员可用。

    Args:
        kind: 资源类型（Deployment / StatefulSet / DaemonSet）
        name: 资源名称
        namespace: 可选，命名空间
    """
    pass


# ============================================================
# B5. Local Code Execution Tool (admin, local CAMEL sandbox)
# ============================================================


@tool
def execute_code(
    code: str,
    language: str = "python",
    timeout: int = 30,
) -> dict:
    """在受控沙箱中执行代码片段，用于日志分析、指标计算、数据处理等。
    使用 CAMEL CodeExecutionToolkit，沙箱类型由 CRATER_AGENT_CODE_SANDBOX 控制
    （默认 subprocess；Docker 模式需要宿主机运行 Docker daemon）。
    仅管理员可用。

    Args:
        code: 要执行的代码内容（Python）
        language: 编程语言，默认 python（当前仅支持 python）
        timeout: 超时秒数，默认 30
    """
    pass  # Executed by LocalToolExecutor._handle_execute_code


# ============================================================
# C. Action Tools (require user confirmation)
# ============================================================


@tool
def cordon_node(node_name: str, reason: Optional[str] = None) -> dict:
    """将节点标记为不可调度。需要管理员确认。"""
    pass


@tool
def uncordon_node(node_name: str, reason: Optional[str] = None) -> dict:
    """恢复节点调度。需要管理员确认。"""
    pass


@tool
def drain_node(node_name: str, reason: Optional[str] = None) -> dict:
    """排空节点并禁止新调度。需要管理员确认。"""
    pass


@tool
def delete_pod(
    name: str,
    namespace: Optional[str] = None,
    force: bool = False,
    grace_period_seconds: Optional[int] = None,
) -> dict:
    """删除 Pod 以触发重建或清理卡死实例。需要管理员确认。"""
    pass


@tool
def restart_workload(
    kind: str,
    name: str,
    namespace: Optional[str] = None,
) -> dict:
    """对 Deployment/StatefulSet/DaemonSet 执行滚动重启。需要管理员确认。"""
    pass


@tool
def resubmit_job(
    job_name: str,
    name: Optional[str] = None,
    cpu: Optional[str] = None,
    memory: Optional[str] = None,
    gpu_count: Optional[int] = None,
    gpu_model: Optional[str] = None,
) -> dict:
    """基于原有配置重新提交作业，可调整部分参数。需要用户确认。

    Args:
        job_name: 原作业的系统唯一名（Job.JobName）
        name: 可选，新作业的显示名称（系统内部 jobName 仍自动生成）
        cpu: 可选，调整后的 CPU 配置（如 "4" 或 "8000m"）
        memory: 可选，调整后的内存配置（如 "32Gi"）
        gpu_count: 可选，调整后的 GPU 数量
        gpu_model: 可选，调整后的 GPU 型号
    """
    pass


@tool
def stop_job(job_name: str) -> dict:
    """停止运行中的作业。需要用户确认。

    Args:
        job_name: 要停止作业的系统唯一名（Job.JobName）
    """
    pass


@tool
def delete_job(job_name: str) -> dict:
    """删除作业记录。需要用户确认。

    Args:
        job_name: 要删除作业的系统唯一名（Job.JobName）
    """
    pass


@tool
def create_jupyter_job(
    name: str = "",
    image_link: str = "",
    cpu: str = "2",
    memory: str = "8Gi",
    gpu_count: Optional[int] = None,
    gpu_model: Optional[str] = None,
) -> dict:
    """创建一个 Jupyter 交互式作业，需要用户确认。
    若部分字段缺失，系统可通过确认表单让用户补全。

    Args:
        name: 作业显示名称（对应后端 Job.Name），可先留空交给确认表单补全
        image_link: 容器镜像地址，可先留空交给确认表单补全
        cpu: CPU 请求量，如 "2"
        memory: 内存请求量，如 "8Gi"
        gpu_count: 可选，GPU 数量
        gpu_model: 可选，GPU 型号，如 v100 / a100
    """
    pass


@tool
def create_training_job(
    name: str = "",
    image_link: str = "",
    command: str = "",
    working_dir: str = "",
    cpu: str = "4",
    memory: str = "16Gi",
    gpu_count: Optional[int] = None,
    gpu_model: Optional[str] = None,
    shell: str = "bash",
) -> dict:
    """创建一个全新的训练作业。Agent 可以先给出草案，用户再在表单中补全并确认。

    Args:
        name: 作业显示名称（对应后端 Job.Name）
        image_link: 容器镜像地址
        command: 启动命令
        working_dir: 容器工作目录
        cpu: CPU 请求量
        memory: 内存请求量
        gpu_count: GPU 数量，可选
        gpu_model: GPU 型号，可选，如 v100 / a100
        shell: 运行命令时使用的 shell，默认 bash
    """
    pass


# ============================================================
# C2. Admin Action Tools (require confirmation, admin only)
# ============================================================


@tool
def mark_audit_handled(item_ids: str = "", handled_by: str = "") -> str:
    """标记审计条目为已处理。需要管理员确认。"""
    pass


@tool
def batch_stop_jobs(job_names: str = "") -> str:
    """批量停止多个作业。传入逗号分隔的作业名列表。需要管理员确认，每个作业都会被停止。"""
    pass


@tool
def notify_job_owner(job_names: str = "", message: str = "") -> str:
    """向作业所有者发送释放资源通知。传入逗号分隔的作业名列表和通知消息。需要管理员确认。"""
    pass


@tool
def run_ops_script(
    script_name: str,
    script_args: Optional[Dict[str, Any]] = None,
    timeout_seconds: Optional[int] = None,
) -> dict:
    """执行受控运维脚本，需要管理员确认。
    script_name 必须是以下白名单值之一，不得自行编造脚本名：
    - inspect_pvc: 检查 PVC 挂载状态和容量
    - inspect_mounts: 检查节点挂载点和 NFS/Lustre 状态
    - collect_events: 收集指定命名空间的 K8s 事件
    - inspect_rdma_node: 检查节点 RDMA/InfiniBand 状态
    - diagnose_nccl_job: 收集分布式训练 NCCL 通信日志

    Args:
        script_name: 必须是白名单中的脚本名（inspect_pvc / inspect_mounts / collect_events / inspect_rdma_node / diagnose_nccl_job），不可传入其他值
        script_args: 结构化参数（JSON 对象），不接受任意 shell 文本
        timeout_seconds: 可选，脚本超时时间（秒）
    """
    pass


# ============================================================
# C3. Extended K8s Write Tools (require confirmation, local kubectl)
# ============================================================


@tool
def k8s_scale_workload(
    kind: str,
    name: str,
    replicas: int,
    namespace: Optional[str] = None,
) -> dict:
    """调整 Deployment/StatefulSet 的副本数。需要管理员确认。

    Args:
        kind: 资源类型（Deployment / StatefulSet）
        name: 资源名称
        replicas: 目标副本数（0-100）
        namespace: 可选，命名空间
    """
    pass


@tool
def k8s_label_node(
    node_name: str,
    key: str,
    value: str,
    overwrite: bool = False,
) -> dict:
    """为节点添加或更新标签。需要管理员确认。

    Args:
        node_name: 节点名称
        key: 标签键
        value: 标签值
        overwrite: 是否覆盖已有标签，默认 False
    """
    pass


@tool
def k8s_taint_node(
    node_name: str,
    key: str,
    value: str = "",
    effect: str = "NoSchedule",
) -> dict:
    """为节点添加 taint。需要管理员确认。

    Args:
        node_name: 节点名称
        key: taint 键
        value: taint 值（可为空）
        effect: taint 效果（NoSchedule / PreferNoSchedule / NoExecute）
    """
    pass


@tool
def execute_admin_command(
    command: str,
    reason: str,
    risk_level: str = "medium",
) -> dict:
    """执行通用管理命令（kubectl/helm 等），由 LLM 自行组合命令内容。需要管理员确认。
    适用于没有专用 tool 覆盖的运维操作，如复杂的 patch、apply、helm upgrade 等。
    命令必须以白名单中的二进制开头（kubectl / helm / velero / istioctl）。
    禁止 delete namespace / delete node / delete pv / exec -it / port-forward 等高危操作。

    Args:
        command: 完整命令字符串，如 "kubectl patch node xxx -p '...'"
        reason: 执行原因说明（必填，用于审计和前端展示）
        risk_level: 风险等级 low/medium/high，默认 medium
    """
    pass


# ============================================================
# B5. Approval / Ticket Tools (auto-execute, read-only)
# ============================================================


@tool
def get_approval_history(user_id: int, days: int = 7) -> dict:
    """查询指定用户近 N 天的审批工单记录，用于评估申请频率和历史模式。

    Args:
        user_id: 用户 ID
        days: 查询最近多少天的工单记录，默认 7
    """
    pass


# ============================================================
# Tool Registry
# ============================================================

# Tools that execute automatically (read-only queries)
AUTO_TOOLS = [
    get_job_detail,
    get_job_logs,
    diagnose_job,
    get_diagnostic_context,
    search_similar_failures,
    query_job_metrics,
    get_realtime_capacity,
    list_available_images,
    list_cuda_base_images,
    list_available_gpu_models,
    recommend_training_images,
    get_health_overview,
    list_user_jobs,
    list_cluster_jobs,
    list_cluster_nodes,
    check_quota,
    detect_idle_jobs,
    get_job_templates,
    get_failure_statistics,
    get_cluster_health_report,
    get_resource_recommendation,
    get_node_detail,
    get_admin_ops_report,
    get_latest_audit_report,
    list_audit_items,
    save_audit_report,
    list_storage_pvcs,
    get_pvc_detail,
    get_pvc_events,
    inspect_job_storage,
    get_storage_capacity_overview,
    get_node_network_summary,
    diagnose_distributed_job_network,
    web_search,
    fetch_url,
    get_agent_runtime_summary,
    k8s_list_nodes,
    k8s_list_pods,
    k8s_get_events,
    k8s_describe_resource,
    k8s_get_pod_logs,
    prometheus_query,
    harbor_check,
    execute_code,  # local CAMEL sandbox execution
    k8s_get_service,
    k8s_get_endpoints,
    k8s_get_ingress,
    get_volcano_queue_state,
    k8s_get_configmap,
    k8s_get_networkpolicy,
    aggregate_image_pull_errors,
    detect_zombie_jobs,
    get_ddp_rank_mapping,
    get_node_kernel_diagnostics,
    get_rdma_interface_status,
    k8s_top_nodes,
    k8s_top_pods,
    k8s_rollout_status,
    get_approval_history,
]

# Tools that require user confirmation before execution (write operations)
CONFIRM_TOOLS = [
    resubmit_job,
    stop_job,
    delete_job,
    create_jupyter_job,
    create_training_job,
    cordon_node,
    uncordon_node,
    drain_node,
    delete_pod,
    restart_workload,
    mark_audit_handled,
    batch_stop_jobs,
    notify_job_owner,
    run_ops_script,
    k8s_scale_workload,
    k8s_label_node,
    k8s_taint_node,
    execute_admin_command,
]

ALL_TOOLS = AUTO_TOOLS + CONFIRM_TOOLS

READ_ONLY_TOOL_NAMES = {t.name for t in AUTO_TOOLS}
CONFIRM_TOOL_NAMES = {t.name for t in CONFIRM_TOOLS}
WRITE_TOOL_NAMES = CONFIRM_TOOL_NAMES  # Alias used by spec

# Tools that should only be accessible to executor/single_agent roles (not explorer/planner)
EXECUTOR_ONLY_TOOL_NAMES = {"execute_code"}

ADMIN_ONLY_TOOL_NAMES = {
    tool.name
    for tool in [
        list_cluster_jobs,
        list_cluster_nodes,
        get_cluster_health_report,
        get_node_detail,
        get_admin_ops_report,
        get_latest_audit_report,
        list_audit_items,
        save_audit_report,
        list_storage_pvcs,
        get_pvc_detail,
        get_pvc_events,
        inspect_job_storage,
        get_storage_capacity_overview,
        get_node_network_summary,
        diagnose_distributed_job_network,
        web_search,
        fetch_url,
        get_agent_runtime_summary,
        k8s_list_nodes,
        k8s_list_pods,
        prometheus_query,
        harbor_check,
        execute_code,  # local code execution, admin-only
        k8s_get_service,
        k8s_get_endpoints,
        k8s_get_ingress,
        get_volcano_queue_state,
        k8s_get_configmap,
        k8s_get_networkpolicy,
        aggregate_image_pull_errors,
        detect_zombie_jobs,
        get_ddp_rank_mapping,
        get_node_kernel_diagnostics,
        get_rdma_interface_status,
        cordon_node,
        uncordon_node,
        drain_node,
        delete_pod,
        restart_workload,
        mark_audit_handled,
        batch_stop_jobs,
        notify_job_owner,
        run_ops_script,
        k8s_top_nodes,
        k8s_top_pods,
        k8s_rollout_status,
        k8s_scale_workload,
        k8s_label_node,
        k8s_taint_node,
        execute_admin_command,
    ]
}

ROLE_ALLOWED_TOOL_NAMES = {
    "planner": set(READ_ONLY_TOOL_NAMES) - EXECUTOR_ONLY_TOOL_NAMES,
    "coordinator": set(READ_ONLY_TOOL_NAMES) - EXECUTOR_ONLY_TOOL_NAMES,
    "explorer": set(READ_ONLY_TOOL_NAMES) - EXECUTOR_ONLY_TOOL_NAMES,
    "executor": {tool.name for tool in ALL_TOOLS},
    "verifier": set(READ_ONLY_TOOL_NAMES) - EXECUTOR_ONLY_TOOL_NAMES,
    "guide": set(),
    "general": set(READ_ONLY_TOOL_NAMES) - EXECUTOR_ONLY_TOOL_NAMES,
    "single_agent": {tool.name for tool in ALL_TOOLS},
}


def is_tool_allowed_for_role(role: Optional[str], tool_name: str) -> bool:
    normalized_role = (role or "single_agent").strip().lower() or "single_agent"
    allowed = ROLE_ALLOWED_TOOL_NAMES.get(normalized_role, ROLE_ALLOWED_TOOL_NAMES["single_agent"])
    return tool_name in allowed


def is_actor_allowed_for_tool(actor_role: Optional[str], tool_name: str) -> bool:
    normalized_role = (actor_role or "user").strip().lower() or "user"
    if tool_name not in ADMIN_ONLY_TOOL_NAMES:
        return True
    # "roleadmin" = strings.ToLower(model.RoleAdmin.String()) from Go backend python_proxy.go
    return normalized_role in {"roleadmin", "admin", "platform_admin", "system_admin"}
