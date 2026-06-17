from __future__ import annotations

import shlex
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from crater_agent.runtime.platform import PlatformRuntimeConfig


_READ_ONLY_KUBECTL_PATTERNS = {
    ("api-resources",),
    ("api-versions",),
    ("auth", "can-i"),
    ("cluster-info",),
    ("config", "view"),
    ("describe",),
    ("explain",),
    ("get",),
    ("logs",),
    ("rollout", "status"),
    ("top",),
    ("version",),
    ("wait",),
}

_CLUSTER_SCOPED_RESOURCES = {
    "clusterrole",
    "clusterrolebinding",
    "clusterroles",
    "clusterrolebindings",
    "crd",
    "crds",
    "customresourcedefinition",
    "customresourcedefinitions",
    "namespace",
    "namespaces",
    "node",
    "nodes",
    "persistentvolume",
    "persistentvolumes",
    "pv",
    "storageclass",
    "storageclasses",
}

_MUTATING_KUBECTL_VERBS = {
    "annotate",
    "apply",
    "cordon",
    "create",
    "delete",
    "drain",
    "edit",
    "label",
    "patch",
    "replace",
    "rollout",
    "scale",
    "set",
    "taint",
    "uncordon",
}


def _normalized_text(value: str) -> str:
    return " ".join(str(value or "").strip().lower().split())


def _protected_namespaces(runtime: PlatformRuntimeConfig) -> set[str]:
    return {
        str(item).strip().lower()
        for item in runtime.k8s_protected_namespaces
        if str(item).strip()
    }


def _protected_nodes(runtime: PlatformRuntimeConfig) -> set[str]:
    return {
        str(item).strip().lower()
        for item in runtime.k8s_protected_node_names
        if str(item).strip()
    }


def parse_shell_command(command: str) -> list[str]:
    try:
        parts = shlex.split(str(command or "").strip())
    except ValueError as exc:
        raise ValueError(f"invalid shell command: {exc}") from exc
    if not parts:
        raise ValueError("command cannot be empty")
    return parts


def enforce_namespace_policy(
    runtime: PlatformRuntimeConfig,
    namespace: str | None,
    *,
    operation: str,
) -> None:
    normalized_namespace = str(namespace or "").strip().lower()
    if not normalized_namespace:
        return
    if normalized_namespace in _protected_namespaces(runtime):
        raise ValueError(
            f"{operation} is blocked for protected namespace {normalized_namespace!r}"
        )


def enforce_node_policy(
    runtime: PlatformRuntimeConfig,
    node_name: str | None,
    *,
    operation: str,
) -> None:
    normalized_node = str(node_name or "").strip().lower()
    if not normalized_node:
        return
    if normalized_node in _protected_nodes(runtime):
        raise ValueError(f"{operation} is blocked for protected node {normalized_node!r}")


def _read_only_pattern_matches(tokens: list[str]) -> bool:
    if not tokens or tokens[0] != "kubectl":
        return False
    non_flag = [token for token in tokens[1:] if not token.startswith("-")]
    for pattern in _READ_ONLY_KUBECTL_PATTERNS:
        if tuple(non_flag[: len(pattern)]) == pattern:
            return True
    return False


def _find_flag_value(tokens: list[str], *flags: str) -> str | None:
    for idx, token in enumerate(tokens):
        if token in flags and idx+1 < len(tokens):
            return tokens[idx+1]
        for flag in flags:
            prefix = f"{flag}="
            if token.startswith(prefix):
                return token[len(prefix):]
    return None


def _resource_and_name(token: str) -> tuple[str, str]:
    parts = [part.strip() for part in token.split("/", 1)]
    if len(parts) == 2:
        return parts[0].lower(), parts[1]
    return token.strip().lower(), ""


def _iter_non_flag_tokens(tokens: list[str], *, start: int = 1) -> list[str]:
    return [token for token in tokens[start:] if not token.startswith("-")]


def _effective_namespace(runtime: PlatformRuntimeConfig, tokens: list[str]) -> str:
    explicit = _find_flag_value(tokens, "-n", "--namespace")
    if explicit:
        return explicit.strip()
    return (
        str(runtime.kube_namespace or "").strip()
        or str(runtime.namespaces.get("job") or "").strip()
    )


def validate_kubectl_write_command(
    runtime: PlatformRuntimeConfig,
    command: str,
) -> list[str]:
    tokens = parse_shell_command(command)
    if tokens[0] != "kubectl":
        raise ValueError("run_kubectl only accepts commands that start with 'kubectl'")

    normalized_command = _normalized_text(command)
    for pattern in runtime.k8s_blocked_command_patterns:
        normalized_pattern = _normalized_text(pattern)
        if normalized_pattern and normalized_pattern in normalized_command:
            raise ValueError(f"kubectl command contains blocked pattern: {pattern!r}")

    if _read_only_pattern_matches(tokens):
        raise ValueError(
            "read-only kubectl commands are blocked here; use the dedicated read-only tools instead"
        )

    non_flag = _iter_non_flag_tokens(tokens)
    if not non_flag:
        raise ValueError("kubectl command is missing an operation")

    operation = non_flag[0].lower()
    if operation not in _MUTATING_KUBECTL_VERBS:
        raise ValueError(
            f"kubectl operation {operation!r} is not allowed via run_kubectl"
        )

    effective_namespace = _effective_namespace(runtime, tokens)
    if operation in {"apply", "create", "replace", "edit", "set"}:
        enforce_namespace_policy(runtime, effective_namespace, operation=operation)

    if operation in {"cordon", "uncordon", "drain"}:
        node_name = non_flag[1] if len(non_flag) > 1 else ""
        enforce_node_policy(runtime, node_name, operation=operation)
        return tokens

    if operation == "rollout":
        action = non_flag[1].lower() if len(non_flag) > 1 else ""
        if action == "status":
            raise ValueError(
                "kubectl rollout status is read-only; use k8s_rollout_status instead"
            )
        enforce_namespace_policy(runtime, effective_namespace, operation=f"rollout {action}".strip())
        return tokens

    resource_token = non_flag[1] if len(non_flag) > 1 else ""
    resource, inline_name = _resource_and_name(resource_token)
    name = inline_name or (non_flag[2] if len(non_flag) > 2 else "")

    if resource in {"namespace", "namespaces", "ns"}:
        enforce_namespace_policy(runtime, name, operation=operation)
        return tokens
    if resource in {"node", "nodes"}:
        enforce_node_policy(runtime, name, operation=operation)
        return tokens

    if resource and resource not in _CLUSTER_SCOPED_RESOURCES:
        enforce_namespace_policy(runtime, effective_namespace, operation=operation)
    return tokens
