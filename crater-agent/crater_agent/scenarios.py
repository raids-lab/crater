"""Lightweight context extraction helpers for Crater Agent.

This module intentionally avoids scenario-specific prompt templates or rigid
entrypoint routing. It only extracts soft hints from page context and the
current user message so the agent can decide which tools to call.
"""

from __future__ import annotations

import re
from typing import Any

_NODE_ROUTE_RE = re.compile(r"/nodes/([^/?#]+)", re.IGNORECASE)
_JOB_ROUTE_RE = re.compile(r"/jobs/([^/?#]+)", re.IGNORECASE)
_PVC_ROUTE_RE = re.compile(r"/pvcs?/([^/?#]+)", re.IGNORECASE)
_SYSTEM_JOB_RE = re.compile(r"\b[a-z0-9]+(?:-[a-z0-9]+){2,}\b", re.IGNORECASE)
_GENERIC_TOKEN_RE = re.compile(r"\b[a-z0-9][a-z0-9.-]*[a-z0-9]\b", re.IGNORECASE)
_DISTRIBUTED_NETWORK_HINT_RE = re.compile(
    r"(nccl|rdma|roce|infiniband|allreduce|all-reduce|distributed|多机|多卡|网络|通信)",
    re.IGNORECASE,
)
_STORAGE_HINT_RE = re.compile(
    r"(pvc|pv|存储|挂载|mount|volume|ceph|文件系统|storage)",
    re.IGNORECASE,
)
_NODE_NEARBY_HINT_RE = re.compile(
    r"(节点|node|notready|not ready|ready|cordon|uncordon|drain|重启|升级|宕机|挂了|故障)",
    re.IGNORECASE,
)
_PVC_TOKEN_RE = re.compile(r"\b(?:pvc[-a-z0-9]*|[-a-z0-9]*pvc[-a-z0-9]*)\b", re.IGNORECASE)

_JOB_PREFIXES = ("sg-", "jpt-", "wc-", "job-", "train-", "task-", "exp-", "ray-")


def _normalized_text(value: Any) -> str:
    return str(value or "").strip().lower()


def is_admin_page(page_context: dict[str, Any]) -> bool:
    route = _normalized_text(page_context.get("route"))
    url = _normalized_text(page_context.get("url"))
    return route.startswith("/admin") or "/admin/" in route or url.startswith("/admin") or "/admin/" in url


def _looks_like_system_job_name(value: str) -> bool:
    normalized = value.strip().lower()
    if not normalized:
        return False
    if not _SYSTEM_JOB_RE.fullmatch(normalized):
        return False
    if normalized.startswith(_JOB_PREFIXES):
        return True
    return normalized.count("-") >= 3 and any(ch.isdigit() for ch in normalized)


def _looks_like_node_name(value: str) -> bool:
    normalized = value.strip().lower()
    if not normalized:
        return False
    if _looks_like_system_job_name(normalized):
        return False
    if "." in normalized:
        return False
    if normalized.startswith(("pvc-", "pod/", "svc/")):
        return False
    if normalized in {"node", "nodes"}:
        return False
    parts = [part for part in re.split(r"[-.]", normalized) if part]
    if len(parts) < 2:
        return False
    return any(ch.isdigit() for ch in normalized) or any(len(part) > 4 for part in parts)


def extract_job_name(page_context: dict[str, Any], user_message: str = "") -> str:
    for key in ("job_name", "jobName"):
        explicit = str(page_context.get(key) or "").strip()
        if explicit:
            return explicit

    for candidate in (page_context.get("route"), page_context.get("url")):
        match = _JOB_ROUTE_RE.search(str(candidate or ""))
        if match:
            return match.group(1).strip()

    for candidate in _SYSTEM_JOB_RE.findall(str(user_message or "")):
        if _looks_like_system_job_name(candidate):
            return candidate.strip()
    return ""


def extract_node_name(page_context: dict[str, Any], user_message: str = "") -> str:
    for key in ("node_name", "nodeName"):
        explicit = str(page_context.get(key) or "").strip()
        if explicit:
            return explicit

    for candidate in (page_context.get("route"), page_context.get("url")):
        match = _NODE_ROUTE_RE.search(str(candidate or ""))
        if match:
            return match.group(1).strip()

    message = str(user_message or "")
    if not _NODE_NEARBY_HINT_RE.search(message):
        return ""

    for candidate in _GENERIC_TOKEN_RE.findall(message):
        if _looks_like_node_name(candidate):
            return candidate.strip()
    return ""


def extract_pvc_name(page_context: dict[str, Any], user_message: str = "") -> str:
    for key in ("pvc_name", "pvcName"):
        explicit = str(page_context.get(key) or "").strip()
        if explicit:
            return explicit

    for candidate in (page_context.get("route"), page_context.get("url")):
        match = _PVC_ROUTE_RE.search(str(candidate or ""))
        if match:
            return match.group(1).strip()

    token_match = _PVC_TOKEN_RE.search(str(user_message or ""))
    if token_match:
        return token_match.group(0).strip()
    return ""


def mentions_storage(page_context: dict[str, Any], user_message: str = "") -> bool:
    if extract_pvc_name(page_context, user_message):
        return True
    route = str(page_context.get("route") or "")
    url = str(page_context.get("url") or "")
    return bool(_STORAGE_HINT_RE.search(route) or _STORAGE_HINT_RE.search(url) or _STORAGE_HINT_RE.search(str(user_message or "")))


def mentions_distributed_network(page_context: dict[str, Any], user_message: str = "") -> bool:
    route = str(page_context.get("route") or "")
    url = str(page_context.get("url") or "")
    text = " ".join((route, url, str(user_message or "")))
    return bool(_DISTRIBUTED_NETWORK_HINT_RE.search(text))


def extract_focus_hints(page_context: dict[str, Any], user_message: str = "") -> dict[str, Any]:
    job_name = extract_job_name(page_context, user_message)
    node_name = extract_node_name(page_context, user_message)
    pvc_name = extract_pvc_name(page_context, user_message)
    return {
        "job_name": job_name,
        "node_name": node_name,
        "pvc_name": pvc_name,
        "is_admin_page": is_admin_page(page_context),
        "mentions_storage": mentions_storage(page_context, user_message),
        "mentions_distributed_network": mentions_distributed_network(page_context, user_message),
    }
