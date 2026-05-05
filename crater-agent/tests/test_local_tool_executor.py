from __future__ import annotations

from pathlib import Path
from typing import Any

import httpx
import pytest

from crater_agent.runtime.platform import PlatformRuntimeConfig
from crater_agent.tools.executor import CompositeToolExecutor
from crater_agent.tools.local_executor import LocalToolExecutor


@pytest.mark.asyncio
async def test_local_executor_sandbox_grep_finds_match(tmp_path: Path):
    target = tmp_path / "hello.txt"
    target.write_text("hello world\nsecond line\n", encoding="utf-8")

    runtime = PlatformRuntimeConfig(
        sandbox_roots=[tmp_path],
        web_search_enabled=False,
    )
    ex = LocalToolExecutor(runtime=runtime)

    out = await ex.execute(
        tool_name="sandbox_grep",
        tool_args={"pattern": "hello", "path": ".", "max_matches": 10},
        session_id="s",
        user_id=1,
        turn_id="t",
        tool_call_id="tc",
        agent_id="a",
        agent_role="single_agent",
    )
    await ex.close()

    assert out["status"] == "success"
    result = out["result"]
    assert result["total_matches"] >= 1
    assert any(str(target) == m["path"] for m in result["matches"])


@pytest.mark.asyncio
async def test_local_executor_sandbox_list_dir_and_read_file(tmp_path: Path):
    target = tmp_path / "a.txt"
    target.write_text("abc\n", encoding="utf-8")

    runtime = PlatformRuntimeConfig(
        sandbox_roots=[tmp_path],
        web_search_enabled=False,
    )
    ex = LocalToolExecutor(runtime=runtime)

    listed = await ex.execute(
        tool_name="sandbox_list_dir",
        tool_args={"path": ".", "max_entries": 50},
        session_id="s",
        user_id=1,
        turn_id="t",
        tool_call_id="tc1",
        agent_id="a",
        agent_role="single_agent",
    )
    assert listed["status"] == "success"
    assert any(entry["name"] == "a.txt" for entry in listed["result"]["entries"])

    read = await ex.execute(
        tool_name="sandbox_read_file",
        tool_args={"path": "a.txt", "max_bytes": 1000},
        session_id="s",
        user_id=1,
        turn_id="t",
        tool_call_id="tc2",
        agent_id="a",
        agent_role="single_agent",
    )
    await ex.close()
    assert read["status"] == "success"
    assert "abc" in read["result"]["content"]


@pytest.mark.asyncio
async def test_local_executor_get_agent_runtime_summary(tmp_path: Path):
    runtime = PlatformRuntimeConfig(
        sandbox_roots=[tmp_path],
        web_search_enabled=False,
    )
    ex = LocalToolExecutor(runtime=runtime)
    out = await ex.execute(
        tool_name="get_agent_runtime_summary",
        tool_args={},
        session_id="s",
        user_id=1,
        turn_id="t",
        tool_call_id="tc",
        agent_id="a",
        agent_role="single_agent",
    )
    await ex.close()
    assert out["status"] == "success"
    assert "agent" in out["result"]
    assert "platformRuntime" in out["result"]


@pytest.mark.asyncio
async def test_local_executor_denies_outside_sandbox(tmp_path: Path):
    runtime = PlatformRuntimeConfig(
        sandbox_roots=[tmp_path],
        web_search_enabled=False,
    )
    ex = LocalToolExecutor(runtime=runtime)

    out = await ex.execute(
        tool_name="sandbox_grep",
        tool_args={"pattern": "x", "path": str(Path("/"))},
        session_id="s",
        user_id=1,
        turn_id="t",
        tool_call_id="tc",
        agent_id="a",
        agent_role="single_agent",
    )
    await ex.close()

    assert out["status"] == "error"
    assert out["error_type"] == "tool_policy"


@pytest.mark.asyncio
async def test_local_executor_web_search_uses_seed_urls():
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.host == "example.com"
        return httpx.Response(
            200,
            text="<html><head><title>GPU Guide</title></head><body>gpu failure diagnosis guide</body></html>",
        )

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler), follow_redirects=True)
    runtime = PlatformRuntimeConfig(
        sandbox_roots=[],
        web_search_enabled=True,
        web_search_allowed_domains=["example.com"],
        web_search_seed_urls=["https://example.com/guide"],
    )
    ex = LocalToolExecutor(runtime=runtime, client=client)

    out = await ex.execute(
        tool_name="web_search",
        tool_args={"query": "gpu diagnosis", "limit": 3},
        session_id="s",
        user_id=1,
        turn_id="t",
        tool_call_id="tc",
        agent_id="a",
        agent_role="single_agent",
    )
    await client.aclose()

    assert out["status"] == "success"
    results = out["result"]["results"]
    assert results
    assert results[0]["url"] == "https://example.com/guide"
    assert "GPU Guide" in results[0]["title"]


class _FakeBackend:
    def __init__(self):
        self.calls: list[dict[str, Any]] = []

    async def execute(self, **kwargs) -> dict[str, Any]:
        self.calls.append(dict(kwargs))
        return {"status": "success", "result": {"delegated": True}}

    async def close(self) -> None:
        return None


@pytest.mark.asyncio
async def test_composite_executor_routes_non_local_to_backend(tmp_path: Path):
    runtime = PlatformRuntimeConfig(
        sandbox_roots=[tmp_path],
        web_search_enabled=False,
    )
    local = LocalToolExecutor(runtime=runtime)
    fake_backend = _FakeBackend()
    ex = CompositeToolExecutor(local=local, backend=fake_backend)  # type: ignore[arg-type]

    out = await ex.execute(
        tool_name="get_job_detail",
        tool_args={"job_name": "jpt-xxx"},
        session_id="s",
        user_id=1,
        turn_id="t",
        tool_call_id="tc",
        agent_id="a",
        agent_role="single_agent",
    )
    await ex.close()

    assert out["status"] == "success"
    assert out["result"]["delegated"] is True
    assert len(fake_backend.calls) == 1
    assert fake_backend.calls[0]["tool_name"] == "get_job_detail"


@pytest.mark.asyncio
async def test_composite_executor_ignores_local_override_for_unsupported_confirm_tool(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
):
    monkeypatch.setenv("CRATER_AGENT_TOOL_ROUTE_cordon_node", "local")
    runtime = PlatformRuntimeConfig(
        sandbox_roots=[tmp_path],
        web_search_enabled=False,
        local_write_tools={"run_kubectl"},
    )
    local = LocalToolExecutor(runtime=runtime)
    fake_backend = _FakeBackend()
    ex = CompositeToolExecutor(local=local, backend=fake_backend)  # type: ignore[arg-type]

    out = await ex.execute(
        tool_name="cordon_node",
        tool_args={"node_name": "node-1", "reason": "maintenance"},
        session_id="s",
        user_id=1,
        turn_id="t",
        tool_call_id="tc",
        agent_id="a",
        agent_role="single_agent",
        actor_role="admin",
    )
    await ex.close()

    assert out["status"] == "success"
    assert out["result"]["delegated"] is True
    assert len(fake_backend.calls) == 1
    assert fake_backend.calls[0]["tool_name"] == "cordon_node"
    assert fake_backend.calls[0]["execution_backend"] == "backend"


respx = pytest.importorskip("respx")
from crater_agent.runtime.platform import PlatformRuntimeConfig


@pytest.mark.asyncio
async def test_k8s_get_events_blocks_user_without_field_selector():
    """Non-admin user without field_selector should get a tool_policy error."""
    runtime = PlatformRuntimeConfig()
    ex = LocalToolExecutor(runtime=runtime)

    result = await ex.execute(
        tool_name="k8s_get_events",
        tool_args={"namespace": "crater-workspace"},
        session_id="s",
        user_id=42,
        agent_role="single_agent",
        actor_role="user",
    )
    await ex.close()

    assert result["status"] == "error"
    assert "involvedObject.name" in result["message"]


@pytest.mark.asyncio
async def test_k8s_get_events_blocks_user_unowned_pod():
    """Non-admin user querying an unowned pod should get a tool_policy error."""
    runtime = PlatformRuntimeConfig()
    ex = LocalToolExecutor(runtime=runtime)

    with respx.mock:
        respx.get("http://localhost:8098/api/agent/k8s-ownership").mock(
            return_value=httpx.Response(200, json={"allowed": False})
        )
        result = await ex.execute(
            tool_name="k8s_get_events",
            tool_args={
                "namespace": "crater-workspace",
                "field_selector": "involvedObject.name=sg-other-worker-0",
            },
            session_id="s",
            user_id=42,
            agent_role="single_agent",
            actor_role="user",
        )
    await ex.close()

    assert result["status"] == "error"
    assert "does not belong" in result["message"]


@pytest.mark.asyncio
async def test_k8s_get_pod_logs_blocks_user_unowned_pod():
    """Non-admin user querying an unowned pod logs should get a tool_policy error."""
    runtime = PlatformRuntimeConfig()
    ex = LocalToolExecutor(runtime=runtime)

    with respx.mock:
        respx.get("http://localhost:8098/api/agent/k8s-ownership").mock(
            return_value=httpx.Response(200, json={"allowed": False})
        )
        result = await ex.execute(
            tool_name="k8s_get_pod_logs",
            tool_args={"pod_name": "sg-other-worker-0"},
            session_id="s",
            user_id=42,
            agent_role="single_agent",
            actor_role="user",
        )
    await ex.close()

    assert result["status"] == "error"
    assert "does not belong" in result["message"]


@pytest.mark.asyncio
async def test_k8s_describe_resource_blocks_non_pod_kind_for_user():
    """Non-admin user can only describe Pod and VCJob."""
    runtime = PlatformRuntimeConfig()
    ex = LocalToolExecutor(runtime=runtime)

    result = await ex.execute(
        tool_name="k8s_describe_resource",
        tool_args={"kind": "Node", "name": "gpu-node-01"},
        session_id="s",
        user_id=42,
        agent_role="single_agent",
        actor_role="user",
    )
    await ex.close()

    assert result["status"] == "error"
    assert "Pod" in result["message"] or "VCJob" in result["message"]


@pytest.mark.asyncio
async def test_k8s_admin_bypasses_ownership_check():
    """Admin user bypasses all ownership checks (may fail with kubectl-not-found, but not tool_policy)."""
    runtime = PlatformRuntimeConfig()
    ex = LocalToolExecutor(runtime=runtime)

    result = await ex.execute(
        tool_name="k8s_get_events",
        tool_args={"namespace": "crater-workspace"},
        session_id="s",
        user_id=1,
        agent_role="single_agent",
        actor_role="admin",
    )
    await ex.close()

    # Admin bypasses ownership checks — message must NOT be an ownership/policy denial
    assert "does not belong" not in result.get("message", "")
    assert "involvedObject.name" not in result.get("message", "")


@pytest.mark.asyncio
async def test_local_executor_blocks_direct_write_execution():
    runtime = PlatformRuntimeConfig(local_write_tools={"k8s_scale_workload"})
    ex = LocalToolExecutor(runtime=runtime)

    result = await ex.execute(
        tool_name="k8s_scale_workload",
        tool_args={"kind": "Deployment", "name": "demo", "replicas": 2},
        session_id="s",
        user_id=1,
        agent_role="single_agent",
        actor_role="admin",
    )
    await ex.close()

    assert result["status"] == "error"
    assert result["error_type"] == "tool_policy"
    assert "requires Go confirmation dispatch" in result["message"]


@pytest.mark.asyncio
async def test_local_executor_allows_confirmed_local_write_dispatch():
    runtime = PlatformRuntimeConfig(local_write_tools={"k8s_scale_workload"})
    ex = LocalToolExecutor(runtime=runtime)

    async def fake_scale_handler(tool_args: dict[str, Any]) -> dict[str, Any]:
        assert tool_args["replicas"] == 3
        return {"status": "success", "result": {"kind": "Deployment", "name": "demo"}}

    ex._handlers["k8s_scale_workload"] = fake_scale_handler  # type: ignore[assignment]

    result = await ex.execute(
        tool_name="k8s_scale_workload",
        tool_args={"kind": "Deployment", "name": "demo", "replicas": 3},
        session_id="s",
        user_id=1,
        agent_role="single_agent",
        actor_role="admin",
        execution_backend="python_local",
        confirmed_dispatch=True,
        confirm_id="42",
    )
    await ex.close()

    assert result["status"] == "success"
    assert result["_audit"]["execution_backend"] == "python_local"
    assert result["result"]["name"] == "demo"


@pytest.mark.asyncio
async def test_composite_executor_routes_local_confirm_tool_via_go_backend():
    class _FakeConfirmBackend(_FakeBackend):
        async def execute(self, **kwargs) -> dict[str, Any]:
            self.calls.append(dict(kwargs))
            return {
                "status": "confirmation_required",
                "confirm_id": "confirm-1",
                "message": "awaiting confirmation",
            }

    runtime = PlatformRuntimeConfig(local_write_tools={"k8s_scale_workload"})
    local = LocalToolExecutor(runtime=runtime)
    fake_backend = _FakeConfirmBackend()
    ex = CompositeToolExecutor(local=local, backend=fake_backend)  # type: ignore[arg-type]

    result = await ex.execute(
        tool_name="k8s_scale_workload",
        tool_args={"kind": "Deployment", "name": "demo", "replicas": 2},
        session_id="s",
        user_id=1,
        turn_id="t",
        tool_call_id="tc-1",
        agent_id="a-1",
        agent_role="single_agent",
        actor_role="admin",
    )
    await ex.close()

    assert result["status"] == "confirmation_required"
    assert len(fake_backend.calls) == 1
    assert fake_backend.calls[0]["execution_backend"] == "python_local"


def test_local_tool_catalog_exposes_enabled_read_and_write_tools():
    runtime = PlatformRuntimeConfig(
        local_core_tools={"get_agent_runtime_summary"},
        local_write_tools={"run_kubectl"},
    )
    ex = LocalToolExecutor(runtime=runtime)

    catalog = ex.build_tool_catalog()
    by_name = {entry["name"]: entry for entry in catalog}

    assert by_name["get_agent_runtime_summary"]["mode"] == "read_only"
    assert by_name["run_kubectl"]["mode"] == "confirm"
    assert by_name["run_kubectl"]["admin_only"] is True
    assert by_name["run_kubectl"]["execution_backend"] == "python_local"


@pytest.mark.asyncio
async def test_run_kubectl_rejects_read_only_and_protected_namespace_commands():
    runtime = PlatformRuntimeConfig(local_write_tools={"run_kubectl"})
    ex = LocalToolExecutor(runtime=runtime)

    read_only = await ex.execute(
        tool_name="run_kubectl",
        tool_args={"command": "kubectl get pods -n crater-workspace", "reason": "debug"},
        session_id="s",
        user_id=1,
        agent_role="single_agent",
        actor_role="admin",
        execution_backend="python_local",
        confirmed_dispatch=True,
    )
    protected = await ex.execute(
        tool_name="run_kubectl",
        tool_args={"command": "kubectl apply -f demo.yaml -n kube-system", "reason": "debug"},
        session_id="s",
        user_id=1,
        agent_role="single_agent",
        actor_role="admin",
        execution_backend="python_local",
        confirmed_dispatch=True,
    )
    await ex.close()

    assert read_only["status"] == "error"
    assert "read-only kubectl commands" in read_only["message"]
    assert protected["status"] == "error"
    assert "protected namespace" in protected["message"]


@pytest.mark.asyncio
async def test_execute_admin_command_rejects_kubectl():
    runtime = PlatformRuntimeConfig(local_write_tools={"execute_admin_command"})
    ex = LocalToolExecutor(runtime=runtime)

    result = await ex.execute(
        tool_name="execute_admin_command",
        tool_args={"command": "kubectl patch node gpu-01 -p '{}'", "reason": "debug"},
        session_id="s",
        user_id=1,
        agent_role="single_agent",
        actor_role="admin",
        execution_backend="python_local",
        confirmed_dispatch=True,
    )
    await ex.close()

    assert result["status"] == "error"
    assert "command must start with one of" in result["message"]


@pytest.mark.asyncio
async def test_execute_admin_command_allows_psql_without_kubeconfig_injection(tmp_path: Path):
    runtime = PlatformRuntimeConfig(
        local_write_tools={"execute_admin_command"},
        kubeconfig_path=str(tmp_path / "kubeconfig"),
    )
    ex = LocalToolExecutor(runtime=runtime)

    observed: dict[str, Any] = {}

    async def fake_run_subprocess(
        cmd: list[str],
        *,
        timeout_seconds: int | None = None,
    ) -> tuple[int, str, str]:
        observed["cmd"] = list(cmd)
        observed["timeout_seconds"] = timeout_seconds
        return 0, "ok", ""

    ex._run_subprocess = fake_run_subprocess  # type: ignore[method-assign]

    result = await ex.execute(
        tool_name="execute_admin_command",
        tool_args={"command": "psql --help", "reason": "validate connectivity tooling"},
        session_id="s",
        user_id=1,
        agent_role="single_agent",
        actor_role="admin",
        execution_backend="python_local",
        confirmed_dispatch=True,
    )
    await ex.close()

    assert result["status"] == "success"
    assert observed["cmd"] == ["psql", "--help"]


@pytest.mark.asyncio
async def test_kubectl_debug_node_runs_without_interactive_flags():
    runtime = PlatformRuntimeConfig(kubeconfig_path="/tmp/kubeconfig")
    ex = LocalToolExecutor(runtime=runtime)

    observed: dict[str, Any] = {}

    async def fake_run_subprocess(
        cmd: list[str],
        *,
        timeout_seconds: int | None = None,
    ) -> tuple[int, str, str]:
        observed["cmd"] = list(cmd)
        observed["timeout_seconds"] = timeout_seconds
        return 0, "ok\n", ""

    ex._run_subprocess = fake_run_subprocess  # type: ignore[method-assign]

    output = await ex._run_kubectl_debug_node("gpu-01", "echo ok", timeout=45)
    await ex.close()

    assert output == "ok\n"
    assert "-it" not in observed["cmd"]
    assert observed["timeout_seconds"] == 45


@pytest.mark.asyncio
async def test_get_rdma_interface_status_exposes_capability_notes():
    runtime = PlatformRuntimeConfig(kubeconfig_path="/tmp/kubeconfig")
    ex = LocalToolExecutor(runtime=runtime)

    async def fake_debug(
        node_name: str,
        commands: str,
        *,
        image: str = "busybox:latest",
        timeout: int = 60,
    ) -> str:
        assert node_name == "gpu-01"
        return """===IB_DEVICES===
(ibstat not available)
===RDMA_LINKS===
(rdma tool not available)
===IB_PORT_STATE===
(no IB ports found)
===IB_LINK===
(no IB interfaces)
===KERNEL_MODULES===
(no relevant modules)
"""

    ex._run_kubectl_debug_node = fake_debug  # type: ignore[method-assign]

    result = await ex.execute(
        tool_name="get_rdma_interface_status",
        tool_args={"node_name": "gpu-01"},
        session_id="s",
        user_id=1,
        agent_role="single_agent",
        actor_role="admin",
    )
    await ex.close()

    assert result["status"] == "success"
    payload = result["result"]
    assert payload["rdma_detected"] is False
    assert "ibstat not available" in payload["capability_notes"]
    assert "rdma tool not available" in payload["capability_notes"]


@pytest.mark.asyncio
async def test_get_node_accelerator_info_exposes_capability_notes():
    runtime = PlatformRuntimeConfig(kubeconfig_path="/tmp/kubeconfig")
    ex = LocalToolExecutor(runtime=runtime)

    async def fake_debug(
        node_name: str,
        commands: str,
        *,
        image: str = "busybox:latest",
        timeout: int = 60,
    ) -> str:
        assert node_name == "gpu-01"
        return """===PCIE_DEVICES===
(lspci unavailable)
===NVIDIA===
(nvidia-smi not found)
===NVIDIA_CUDA===
(no cuda)
===HYGON_DCU===
(no hygon/rocm tools)
===HYGON_DRIVER===
(no amdgpu driver)
===ASCEND_NPU===
(npu-smi not found)
===ASCEND_DRIVER===
(no ascend driver)
===CAMBRICON_MLU===
(cnmon not found)
===COMM_LIBS===
NCCL:
  not found
HCCL:
  not found
RCCL:
  not found
MPI:
  not found
===ACCEL_MODULES===
(no accelerator modules)
"""

    ex._run_kubectl_debug_node = fake_debug  # type: ignore[method-assign]

    result = await ex.execute(
        tool_name="get_node_accelerator_info",
        tool_args={"node_name": "gpu-01"},
        session_id="s",
        user_id=1,
        agent_role="single_agent",
        actor_role="admin",
    )
    await ex.close()

    assert result["status"] == "success"
    payload = result["result"]
    assert payload["accelerator_detected"] is False
    assert "lspci unavailable" in payload["capability_notes"]
    assert "nvidia-smi not found" in payload["capability_notes"]
