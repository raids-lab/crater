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


import respx
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
