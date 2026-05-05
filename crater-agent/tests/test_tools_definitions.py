from __future__ import annotations

from crater_agent.tools.definitions import (
    ADMIN_ONLY_TOOL_NAMES,
    AUTO_ACTION_TOOL_NAMES,
    CONFIRM_TOOL_NAMES,
    READ_ONLY_TOOL_NAMES,
    is_actor_allowed_for_tool,
    is_tool_allowed_for_role,
)


NEW_READ_ONLY_TOOLS = {
    "list_storage_pvcs",
    "get_pvc_detail",
    "get_pvc_events",
    "inspect_job_storage",
    "get_storage_capacity_overview",
    "get_node_network_summary",
    "diagnose_distributed_job_network",
    "web_search",
    "fetch_url",
    "get_agent_runtime_summary",
    "k8s_list_nodes",
    "k8s_list_pods",
    "k8s_get_events",
    "k8s_describe_resource",
    "k8s_get_pod_logs",
    "k8s_get_service",
    "k8s_get_endpoints",
    "k8s_get_ingress",
}


def test_new_read_only_tools_are_registered():
    assert NEW_READ_ONLY_TOOLS.issubset(READ_ONLY_TOOL_NAMES)


def test_planner_coordinator_explorer_can_use_new_read_only_tools():
    for role in ("planner", "coordinator", "explorer"):
        for tool_name in NEW_READ_ONLY_TOOLS:
            assert is_tool_allowed_for_role(role, tool_name), f"{role} should allow {tool_name}"


def test_non_executor_roles_cannot_use_run_kubectl():
    for role in ("planner", "coordinator", "explorer", "verifier", "general", "guide"):
        assert not is_tool_allowed_for_role(role, "run_kubectl")


def test_executor_and_single_agent_can_use_confirmed_local_admin_commands():
    assert is_tool_allowed_for_role("executor", "run_kubectl")
    assert is_tool_allowed_for_role("single_agent", "run_kubectl")
    assert is_tool_allowed_for_role("executor", "execute_admin_command")
    assert is_tool_allowed_for_role("single_agent", "execute_admin_command")


def test_k8s_read_tools_accessible_to_user():
    """User-scoped k8s read tools should not require admin."""
    for tool in (
        "k8s_list_pods",
        "k8s_get_events",
        "k8s_describe_resource",
        "k8s_get_pod_logs",
        "k8s_get_service",
        "k8s_get_endpoints",
        "k8s_get_ingress",
    ):
        assert is_actor_allowed_for_tool("user", tool) is True, f"{tool} should be user-accessible"
        assert tool not in ADMIN_ONLY_TOOL_NAMES, f"{tool} should not be in ADMIN_ONLY_TOOL_NAMES"


def test_k8s_cluster_tools_remain_admin_only():
    """Cluster-wide k8s inventory should remain admin-only."""
    for tool in ("k8s_list_nodes",):
        assert is_actor_allowed_for_tool("user", tool) is False
        assert is_actor_allowed_for_tool("admin", tool) is True


def test_web_search_accessible_to_user_and_read_only():
    assert "web_search" in READ_ONLY_TOOL_NAMES
    assert "web_search" not in ADMIN_ONLY_TOOL_NAMES
    assert is_actor_allowed_for_tool("user", "web_search") is True


def test_fetch_url_accessible_to_user_and_read_only():
    assert "fetch_url" in READ_ONLY_TOOL_NAMES
    assert "fetch_url" not in ADMIN_ONLY_TOOL_NAMES
    assert is_actor_allowed_for_tool("user", "fetch_url") is True


def test_new_job_creation_aliases_require_confirmation():
    for tool in (
        "create_webide_job",
        "create_custom_job",
        "create_pytorch_job",
        "create_tensorflow_job",
    ):
        assert tool in CONFIRM_TOOL_NAMES


def test_notify_job_owner_is_admin_confirm_tool():
    assert "notify_job_owner" not in AUTO_ACTION_TOOL_NAMES
    assert "notify_job_owner" in CONFIRM_TOOL_NAMES
    assert "notify_job_owner" not in READ_ONLY_TOOL_NAMES
    assert "notify_job_owner" in ADMIN_ONLY_TOOL_NAMES
    assert is_tool_allowed_for_role("single_agent", "notify_job_owner")
    assert is_tool_allowed_for_role("executor", "notify_job_owner")
    assert not is_tool_allowed_for_role("planner", "notify_job_owner")
    assert not is_actor_allowed_for_tool("user", "notify_job_owner")
    assert is_actor_allowed_for_tool("admin", "notify_job_owner")


def test_prometheus_query_remains_admin_only():
    assert is_actor_allowed_for_tool("user", "prometheus_query") is False
    assert is_actor_allowed_for_tool("admin", "prometheus_query") is True
