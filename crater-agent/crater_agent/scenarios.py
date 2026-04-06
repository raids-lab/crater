"""Shared scenario / entrypoint inference for Crater Agent."""

from __future__ import annotations

import re
from typing import Any

DEFAULT_ENTRYPOINT = "default"
NODE_ANALYSIS_ENTRYPOINT = "node_analysis"

_NODE_ROUTE_RE = re.compile(r"/nodes/([^/?#]+)", re.IGNORECASE)
_NODE_TOKEN_RE = re.compile(r"\b[a-z0-9]+(?:-[a-z0-9]+)*node(?:-[a-z0-9]+)*\b", re.IGNORECASE)


def _normalized_text(value: Any) -> str:
    return str(value or "").strip().lower()


def normalize_entrypoint(value: Any) -> str:
    normalized = _normalized_text(value)
    if normalized == NODE_ANALYSIS_ENTRYPOINT:
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

def infer_entrypoint(page_context: dict[str, Any], user_message: str = "") -> str:
    explicit = normalize_entrypoint(page_context.get("entrypoint") or page_context.get("entryPoint"))
    if explicit != DEFAULT_ENTRYPOINT:
        return explicit

    if is_admin_page(page_context) and extract_node_name(page_context, user_message):
        return NODE_ANALYSIS_ENTRYPOINT
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

    return ""
