"""Shared scenario / entrypoint inference for Crater Agent."""

from __future__ import annotations

import re
from typing import Any

DEFAULT_ENTRYPOINT = "default"
NODE_ANALYSIS_ENTRYPOINT = "node_analysis"
OPS_REPORT_ENTRYPOINT = "ops_report"

_NODE_ROUTE_RE = re.compile(r"/nodes/([^/?#]+)", re.IGNORECASE)
_NODE_TOKEN_RE = re.compile(r"\b[a-z0-9]+(?:-[a-z0-9]+)*node(?:-[a-z0-9]+)*\b", re.IGNORECASE)


def _normalized_text(value: Any) -> str:
    return str(value or "").strip().lower()


def normalize_entrypoint(value: Any) -> str:
    normalized = _normalized_text(value)
    if normalized in {NODE_ANALYSIS_ENTRYPOINT, OPS_REPORT_ENTRYPOINT}:
        return normalized
    return DEFAULT_ENTRYPOINT


def is_admin_page(page_context: dict[str, Any]) -> bool:
    route = _normalized_text(page_context.get("route"))
    url = _normalized_text(page_context.get("url"))
    return route.startswith("/admin") or "/admin/" in route or url.startswith("/admin") or "/admin/" in url


def extract_node_name(page_context: dict[str, Any], user_message: str = "") -> str:
    explicit = str(page_context.get("node_name") or page_context.get("nodeName") or "").strip()
    if explicit:
        return explicit

    for candidate in (page_context.get("route"), page_context.get("url")):
        match = _NODE_ROUTE_RE.search(str(candidate or ""))
        if match:
            return match.group(1).strip()

    token_match = _NODE_TOKEN_RE.search(user_message or "")
    if token_match:
        return token_match.group(0).strip()
    return ""


def is_ops_report_request(page_context: dict[str, Any], user_message: str) -> bool:
    if not is_admin_page(page_context):
        return False
    normalized = _normalized_text(user_message)
    keywords = (
        "分析报告",
        "运维报告",
        "智能运维",
        "巡检",
        "统计",
        "成功作业",
        "失败作业",
        "最近一次报告",
        "最新报告",
        "管理员报告",
        "审计报告",
    )
    return any(keyword in normalized for keyword in keywords)


def infer_entrypoint(page_context: dict[str, Any], user_message: str = "") -> str:
    explicit = normalize_entrypoint(page_context.get("entrypoint") or page_context.get("entryPoint"))
    if explicit != DEFAULT_ENTRYPOINT:
        return explicit

    if is_admin_page(page_context) and extract_node_name(page_context, user_message):
        return NODE_ANALYSIS_ENTRYPOINT
    if is_ops_report_request(page_context, user_message):
        return OPS_REPORT_ENTRYPOINT
    return DEFAULT_ENTRYPOINT


def build_scenario_prompt_hint(page_context: dict[str, Any], user_message: str = "") -> str:
    entrypoint = infer_entrypoint(page_context, user_message)
    node_name = extract_node_name(page_context, user_message)

    if entrypoint == NODE_ANALYSIS_ENTRYPOINT:
        node_line = f"- 当前关注节点: {node_name}" if node_name else "- 当前关注节点: 未明确"
        return "\n".join(
            [
                "## 场景入口",
                "- 当前入口: 管理员节点分析",
                node_line,
                "- 优先调用 get_node_detail，先确认节点状态、running_jobs、alerts、资源占用，再下结论。",
                "- 如果节点存在资源压力、taint、作业集中或异常告警，要明确指出风险范围和建议动作。",
            ]
        )

    if entrypoint == OPS_REPORT_ENTRYPOINT:
        return "\n".join(
            [
                "## 场景入口",
                "- 当前入口: 智能运维分析报告",
                "- 优先调用 get_admin_ops_report 获取管理员视角的成功/失败/闲置作业汇总。",
                "- 如果用户明确问“最近一次/定时/上次报告”，再优先考虑 get_latest_audit_report 和 list_audit_items。",
                "- 输出时先给结论，再给关键统计和建议动作，不要只罗列原始字段。",
            ]
        )

    return ""
