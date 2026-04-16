from __future__ import annotations

from crater_agent.tools.definitions import (
    ADMIN_ONLY_TOOL_NAMES,
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
    "sandbox_grep",
    "sandbox_list_dir",
    "sandbox_read_file",
    "get_agent_runtime_summary",
    "k8s_list_nodes",
    "k8s_list_pods",
    "k8s_get_events",
    "k8s_describe_resource",
}


def test_new_read_only_tools_are_registered():
    assert NEW_READ_ONLY_TOOLS.issubset(READ_ONLY_TOOL_NAMES)


def test_run_ops_script_is_confirmation_tool():
    assert "run_ops_script" in CONFIRM_TOOL_NAMES


def test_planner_coordinator_explorer_can_use_new_read_only_tools():
    for role in ("planner", "coordinator", "explorer"):
        for tool_name in NEW_READ_ONLY_TOOLS:
            assert is_tool_allowed_for_role(role, tool_name), f"{role} should allow {tool_name}"


def test_non_executor_roles_cannot_use_run_ops_script():
    for role in ("planner", "coordinator", "explorer", "verifier", "general", "guide"):
        assert not is_tool_allowed_for_role(role, "run_ops_script")


def test_executor_and_single_agent_can_use_run_ops_script():
    assert is_tool_allowed_for_role("executor", "run_ops_script")
    assert is_tool_allowed_for_role("single_agent", "run_ops_script")


def test_k8s_read_tools_accessible_to_user():
    """k8s_get_events/describe/pod_logs no longer admin-only."""
    for tool in ("k8s_get_events", "k8s_describe_resource", "k8s_get_pod_logs"):
        assert is_actor_allowed_for_tool("user", tool) is True, f"{tool} should be user-accessible"
        assert tool not in ADMIN_ONLY_TOOL_NAMES, f"{tool} should not be in ADMIN_ONLY_TOOL_NAMES"


def test_k8s_cluster_tools_remain_admin_only():
    """k8s_list_nodes/list_pods remain admin-only."""
    for tool in ("k8s_list_nodes", "k8s_list_pods"):
        assert is_actor_allowed_for_tool("user", tool) is False
        assert is_actor_allowed_for_tool("admin", tool) is True


def test_prometheus_query_remains_admin_only():
    assert is_actor_allowed_for_tool("user", "prometheus_query") is False
    assert is_actor_allowed_for_tool("admin", "prometheus_query") is True
