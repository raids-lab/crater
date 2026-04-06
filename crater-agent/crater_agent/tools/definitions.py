"""Tool definitions for Crater Agent.

Each tool is declared here with its schema. Actual execution happens via
HTTP callback to the Go backend's /v1/agent/tools/execute endpoint.
"""

from typing import List, Optional

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
def get_job_events(job_name: str) -> dict:
    """获取作业的 Kubernetes 事件列表，包括调度、镜像拉取、容器状态变更等事件。

    Args:
        job_name: 作业的系统唯一名（Job.JobName）
    """
    pass


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
    days: int = 30,
    limit: int = 20,
) -> dict:
    """列出当前用户在当前账户下的作业，支持按状态筛选。

    Args:
        statuses: 可选，按状态过滤，如 running、failed、deleted、completed
        days: 查询最近 N 天，默认 30
        limit: 最多返回 N 条，默认 20
    """
    pass


@tool
def get_cluster_health_overview(days: int = 7) -> dict:
    """获取管理员视角下整个集群的作业健康概览：全局作业总量、运行中、失败、排队、涉及用户数等。

    Args:
        days: 统计最近 N 天，默认 7
    """
    pass


@tool
def list_cluster_jobs(
    statuses: Optional[List[str]] = None,
    days: int = 7,
    limit: int = 30,
) -> dict:
    """列出管理员视角下整个集群最近的作业，支持按状态筛选。

    Args:
        statuses: 可选，按状态过滤，如 running、failed、pending
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
    """查看当前账户的资源配额使用情况（CPU/GPU/Memory 的配额和已使用量）。

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
# C. Action Tools (require user confirmation)
# ============================================================


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


# ============================================================
# Tool Registry
# ============================================================

# Tools that execute automatically (read-only queries)
AUTO_TOOLS = [
    get_job_detail,
    get_job_events,
    get_job_logs,
    diagnose_job,
    get_diagnostic_context,
    search_similar_failures,
    query_job_metrics,
    analyze_queue_status,
    get_realtime_capacity,
    list_available_images,
    list_cuda_base_images,
    list_available_gpu_models,
    recommend_training_images,
    get_health_overview,
    list_user_jobs,
    get_cluster_health_overview,
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
]

# Tools that require user confirmation before execution (write operations)
CONFIRM_TOOLS = [
    resubmit_job,
    stop_job,
    delete_job,
    create_jupyter_job,
    create_training_job,
    mark_audit_handled,
    batch_stop_jobs,
    notify_job_owner,
]

ALL_TOOLS = AUTO_TOOLS + CONFIRM_TOOLS

READ_ONLY_TOOL_NAMES = {t.name for t in AUTO_TOOLS}
CONFIRM_TOOL_NAMES = {t.name for t in CONFIRM_TOOLS}
WRITE_TOOL_NAMES = CONFIRM_TOOL_NAMES  # Alias used by spec

ROLE_ALLOWED_TOOL_NAMES = {
    "planner": set(READ_ONLY_TOOL_NAMES),
    "coordinator": set(READ_ONLY_TOOL_NAMES),
    "explorer": set(READ_ONLY_TOOL_NAMES),
    "executor": {tool.name for tool in ALL_TOOLS},
    "verifier": set(READ_ONLY_TOOL_NAMES),
    "guide": set(),
    "general": set(READ_ONLY_TOOL_NAMES),
    "single_agent": {tool.name for tool in ALL_TOOLS},
}


def is_tool_allowed_for_role(role: Optional[str], tool_name: str) -> bool:
    normalized_role = (role or "single_agent").strip().lower() or "single_agent"
    allowed = ROLE_ALLOWED_TOOL_NAMES.get(normalized_role, ROLE_ALLOWED_TOOL_NAMES["single_agent"])
    return tool_name in allowed
