"""Minimal tool definitions for Crater Agent chat."""

from typing import Optional

from langchain_core.tools import tool


@tool
def get_job_detail(job_name: str) -> dict:
    """获取指定作业的详细状态、资源、时间线和终止信息。"""
    pass


@tool
def get_job_events(job_name: str) -> dict:
    """获取作业关联事件。"""
    pass


@tool
def get_job_logs(job_name: str, tail: int = 200, keyword: Optional[str] = None) -> dict:
    """获取作业日志尾部，可选关键词过滤。"""
    pass


@tool
def diagnose_job(job_name: str) -> dict:
    """对作业进行规则诊断，返回故障分类、证据和建议。"""
    pass


@tool
def get_diagnostic_context(job_name: str, include_log: bool = True, tail: int = 120) -> dict:
    """获取作业诊断上下文，包括元数据、事件、终止状态和可选日志。"""
    pass


@tool
def search_similar_failures(job_name: str, days: int = 30, limit: int = 5) -> dict:
    """搜索与指定作业相似的历史失败案例。"""
    pass


@tool
def query_job_metrics(job_name: str, metrics: Optional[list[str]] = None, time_range: str = "last_2h") -> dict:
    """查询作业 GPU/CPU/内存等监控指标。"""
    pass


@tool
def analyze_queue_status(job_name: str) -> dict:
    """分析 Pending 作业排队或调度原因。"""
    pass


@tool
def get_realtime_capacity() -> dict:
    """读取实时资源容量概览。"""
    pass


@tool
def list_available_images(limit: int = 30) -> dict:
    """列出当前用户可见镜像。"""
    pass


@tool
def list_available_gpu_models() -> dict:
    """列出可选 GPU 型号。"""
    pass


@tool
def check_quota() -> dict:
    """查看当前账户配额摘要。"""
    pass


@tool
def list_user_jobs(limit: int = 20) -> dict:
    """列出当前用户近期作业。"""
    pass


@tool
def get_job_templates() -> dict:
    """列出可创建的作业模板。"""
    pass


@tool
def get_resource_recommendation(task_type: Optional[str] = None) -> dict:
    """根据任务描述推荐基础资源配置。"""
    pass


@tool
def resubmit_job(job_name: str) -> dict:
    """重新提交已有作业；不要先文字询问确认，调用本工具后系统会弹确认卡片。"""
    pass


@tool
def stop_job(job_name: str) -> dict:
    """停止作业；不要先文字询问确认，调用本工具后系统会弹确认卡片。"""
    pass


@tool
def delete_job(job_name: str) -> dict:
    """删除作业；不要先文字询问确认，调用本工具后系统会弹确认卡片。"""
    pass


@tool
def create_jupyter_job(
    name: str,
    image_link: str,
    cpu: Optional[str] = None,
    memory: Optional[str] = None,
    gpu_count: Optional[int] = None,
    gpu_model: Optional[str] = None,
) -> dict:
    """创建 Jupyter 作业；不要先文字询问确认，调用本工具后系统会弹确认卡片。"""
    pass


@tool
def create_webide_job(
    name: str,
    image_link: str,
    cpu: Optional[str] = None,
    memory: Optional[str] = None,
    gpu_count: Optional[int] = None,
    gpu_model: Optional[str] = None,
) -> dict:
    """创建 WebIDE 作业；不要先文字询问确认，调用本工具后系统会弹确认卡片。"""
    pass


@tool
def create_custom_job(
    name: str,
    image_link: str,
    command: str,
    cpu: Optional[str] = None,
    memory: Optional[str] = None,
    gpu_count: Optional[int] = None,
    gpu_model: Optional[str] = None,
) -> dict:
    """创建自定义作业；不要先文字询问确认，调用本工具后系统会弹确认卡片。"""
    pass


@tool
def create_pytorch_job(
    name: str,
    image_link: str,
    command: Optional[str] = None,
    cpu: Optional[str] = None,
    memory: Optional[str] = None,
    gpu_count: Optional[int] = None,
    gpu_model: Optional[str] = None,
) -> dict:
    """创建 PyTorch 作业；不要先文字询问确认，调用本工具后系统会弹确认卡片。"""
    pass


@tool
def create_tensorflow_job(
    name: str,
    image_link: str,
    command: Optional[str] = None,
    cpu: Optional[str] = None,
    memory: Optional[str] = None,
    gpu_count: Optional[int] = None,
    gpu_model: Optional[str] = None,
) -> dict:
    """创建 TensorFlow 作业；不要先文字询问确认，调用本工具后系统会弹确认卡片。"""
    pass


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
    list_available_gpu_models,
    check_quota,
    list_user_jobs,
    get_job_templates,
    get_resource_recommendation,
]

AUTO_ACTION_TOOLS = []

CONFIRM_TOOLS = [
    resubmit_job,
    stop_job,
    delete_job,
    create_jupyter_job,
    create_webide_job,
    create_custom_job,
    create_pytorch_job,
    create_tensorflow_job,
]

ALL_TOOLS = AUTO_TOOLS + AUTO_ACTION_TOOLS + CONFIRM_TOOLS

DEPRECATED_TOOL_NAMES: set[str] = set()
INTERNAL_TOOLS = []
INTERNAL_TOOL_NAMES = set()
READ_ONLY_TOOL_NAMES = {tool_item.name for tool_item in AUTO_TOOLS}
AUTO_ACTION_TOOL_NAMES = set()
CONFIRM_TOOL_NAMES = {tool_item.name for tool_item in CONFIRM_TOOLS}
WRITE_TOOL_NAMES = CONFIRM_TOOL_NAMES
ADMIN_ONLY_TOOL_NAMES = set()

ROLE_ALLOWED_TOOL_NAMES = {
    "planner": READ_ONLY_TOOL_NAMES,
    "coordinator": READ_ONLY_TOOL_NAMES,
    "explorer": READ_ONLY_TOOL_NAMES,
    "executor": {tool_item.name for tool_item in ALL_TOOLS},
    "verifier": READ_ONLY_TOOL_NAMES,
    "guide": set(),
    "general": READ_ONLY_TOOL_NAMES,
    "single_agent": {tool_item.name for tool_item in ALL_TOOLS},
}


def is_tool_allowed_for_role(role: Optional[str], tool_name: str) -> bool:
    normalized_role = (role or "single_agent").strip().lower() or "single_agent"
    allowed = ROLE_ALLOWED_TOOL_NAMES.get(normalized_role, ROLE_ALLOWED_TOOL_NAMES["single_agent"])
    return tool_name in allowed


def is_actor_allowed_for_tool(actor_role: Optional[str], tool_name: str) -> bool:
    return True
