"""Role-based tool selector and capability sanitization.

Security model:
- Privilege elevation must come from trusted actor identity, never from page
  route, URL, or user-provided text.
- Page scope may reduce visibility (for example an admin browsing a user page),
  but it must never expand a normal user into admin scope.
- Capability payloads are sanitized before being exposed to prompts so an
  injected or malformed tool catalog cannot leak admin-only tools.
"""

from __future__ import annotations

import logging
from typing import Any

from crater_agent.tools.definitions import ADMIN_ONLY_TOOL_NAMES, is_actor_allowed_for_tool

logger = logging.getLogger(__name__)

_ADMIN_ROLES = frozenset({"roleadmin", "admin", "platform_admin", "system_admin"})
_SYSTEM_ROLES = frozenset({"system"})


def normalize_actor_role(actor_role: Any) -> str:
    return str(actor_role or "user").strip().lower() or "user"


def is_privileged_actor_role(actor_role: Any) -> bool:
    normalized = normalize_actor_role(actor_role)
    return normalized in _ADMIN_ROLES or normalized in _SYSTEM_ROLES


def _infer_page_scope(context: dict[str, Any]) -> str:
    capabilities = context.get("capabilities") or {}
    surface = capabilities.get("surface") if isinstance(capabilities, dict) else {}
    if isinstance(surface, dict):
        declared_scope = str(surface.get("page_scope") or "").strip().lower()
        if declared_scope in {"admin", "user"}:
            return declared_scope

    page = context.get("page") or {}
    route = str(page.get("route") or "").strip().lower()
    url = str(page.get("url") or "").strip().lower()
    if route.startswith("/admin") or "/admin/" in route or url.startswith("/admin") or "/admin/" in url:
        return "admin"
    if route or url:
        return "user"
    return ""


def _resolve_actor_role(context: dict[str, Any]) -> str:
    """Determine the effective actor role from trusted identity plus page scope.

    Trusted actor identity (JWT/backend context) is the only source that can
    grant admin privileges. Page scope can narrow visibility, but never widen
    it for a normal user.
    """
    actor = context.get("actor") or {}
    trusted_role = normalize_actor_role(actor.get("role"))
    if trusted_role in _SYSTEM_ROLES:
        return trusted_role

    page_scope = _infer_page_scope(context)
    if page_scope == "user":
        return "user"
    if page_scope == "admin" and not is_privileged_actor_role(trusted_role):
        return "user"
    return trusted_role


def sanitize_enabled_tool_names(context: dict[str, Any], enabled_tool_names: list[Any] | None) -> list[str]:
    effective_role = _resolve_actor_role(context)
    sanitized: list[str] = []
    seen: set[str] = set()
    for item in enabled_tool_names or []:
        tool_name = str(item or "").strip()
        if not tool_name or tool_name in seen:
            continue
        if not is_actor_allowed_for_tool(effective_role, tool_name):
            continue
        sanitized.append(tool_name)
        seen.add(tool_name)
    return sanitized


def sanitize_capabilities_for_context(
    context: dict[str, Any],
    capabilities: dict[str, Any] | None,
) -> dict[str, Any]:
    raw = dict(capabilities or {})
    sanitized = dict(raw)

    effective_role = _resolve_actor_role(context)
    sanitized_enabled = sanitize_enabled_tool_names(context, raw.get("enabled_tools"))
    sanitized["enabled_tools"] = sanitized_enabled

    raw_catalog = raw.get("tool_catalog")
    filtered_catalog: list[dict[str, Any]] = []
    enabled_set = set(sanitized_enabled)
    if isinstance(raw_catalog, list):
        for item in raw_catalog:
            if not isinstance(item, dict):
                continue
            name = str(item.get("name") or "").strip()
            if not name:
                continue
            if enabled_set and name not in enabled_set:
                continue
            if not is_actor_allowed_for_tool(effective_role, name):
                continue
            filtered_catalog.append(dict(item))
    sanitized["tool_catalog"] = filtered_catalog

    surface = dict(raw.get("surface") or {})
    page_scope = _infer_page_scope({"capabilities": {"surface": surface}, "page": context.get("page")})
    surface["page_scope"] = "admin" if page_scope == "admin" and is_privileged_actor_role(effective_role) else "user"
    sanitized["surface"] = surface
    return sanitized


def select_tools_for_context(context: dict[str, Any], all_tools: list) -> list:
    """Select tools based on actor role.

    - Admin: returns all tools (no filtering)
    - User: returns only user-level tools

    This replaces the previous URL/route-based filtering.
    """
    role = _resolve_actor_role(context)

    if role in _ADMIN_ROLES or role in _SYSTEM_ROLES:
        logger.debug("Tool selector: admin role %r → all %d tools", role, len(all_tools))
        return all_tools

    # Regular user: expose every tool that is not admin-only.
    filtered = [t for t in all_tools if t.name not in ADMIN_ONLY_TOOL_NAMES]
    logger.debug(
        "Tool selector: user role %r → %d/%d tools",
        role, len(filtered), len(all_tools),
    )
    return filtered
