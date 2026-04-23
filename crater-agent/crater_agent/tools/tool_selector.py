"""Role-based tool selector for dynamic tool injection.

Instead of binding ALL tools to the LLM (which wastes context window),
select a relevant subset based on actor role (admin vs user).

Admin sees admin tools only when on admin pages; regular user pages
always get user-level tools regardless of JWT role.
"""

from __future__ import annotations

import logging
from typing import Any

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# User-level tool names (available to all roles)
# ---------------------------------------------------------------------------

USER_TOOL_NAMES: frozenset[str] = frozenset({
    # Job diagnosis
    "get_job_detail",
    "get_diagnostic_context",
    "diagnose_job",
    "get_job_logs",
    "search_similar_failures",
    "query_job_metrics",
    # Job management
    "stop_job",
    "delete_job",
    "resubmit_job",
    "create_jupyter_job",
    "create_training_job",
    "list_user_jobs",
    "get_job_templates",
    "get_health_overview",
    "check_quota",
    # Resource query
    "get_realtime_capacity",
    "list_available_images",
    "list_cuda_base_images",
    "list_available_gpu_models",
    "recommend_training_images",
    "get_resource_recommendation",
    # Scoped K8s (ownership-checked for non-admin)
    "k8s_get_events",
    "k8s_describe_resource",
    "k8s_get_pod_logs",
    # Meta
    "get_agent_runtime_summary",
})

_ADMIN_ROLES = frozenset({"roleadmin", "admin", "platform_admin", "system_admin"})


def _resolve_actor_role(context: dict[str, Any]) -> str:
    """Determine the effective actor role from context.

    URL/page route is the primary signal:
      - /admin pages → admin role (admin tools available)
      - non-admin pages → user role (user tools only)
    JWT role is the fallback when no URL info is available.
    """
    page = context.get("page") or {}
    route = str(page.get("route") or "").strip().lower()
    url = str(page.get("url") or "").strip().lower()

    # Primary: URL determines role
    if route.startswith("/admin") or "/admin/" in route or url.startswith("/admin") or "/admin/" in url:
        return "admin"
    if route or url:
        return "user"

    # Fallback: JWT role when no URL info
    actor = context.get("actor") or {}
    role = str(actor.get("role") or "user").strip().lower() or "user"
    return role


def select_tools_for_context(context: dict[str, Any], all_tools: list) -> list:
    """Select tools based on actor role.

    - Admin: returns all tools (no filtering)
    - User: returns only user-level tools

    This replaces the previous URL/route-based filtering.
    """
    role = _resolve_actor_role(context)

    if role in _ADMIN_ROLES:
        logger.debug("Tool selector: admin role %r → all %d tools", role, len(all_tools))
        return all_tools

    # Regular user: filter to user-level tools only
    filtered = [t for t in all_tools if t.name in USER_TOOL_NAMES]
    logger.debug(
        "Tool selector: user role %r → %d/%d tools",
        role, len(filtered), len(all_tools),
    )
    return filtered
