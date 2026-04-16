"""Tests for system prompt builder."""
from __future__ import annotations

from crater_agent.agent.prompts import build_system_prompt


def _make_context(enabled_tools: list[str], confirm_tools: list[str]) -> dict:
    return {
        "actor": {
            "username": "testuser",
            "user_id": 1,
            "account_name": "test",
            "account_id": 1,
        },
        "page": {"route": "/user/jobs", "url": ""},
        "capabilities": {
            "enabled_tools": enabled_tools,
            "confirm_tools": confirm_tools,
        },
    }


def test_prompt_does_not_list_tool_names_as_text():
    """Tool names should not appear as plain text in the prompt (delivered via bind_tools API)."""
    tools = ["get_job_detail", "diagnose_job", "k8s_get_events"]
    ctx = _make_context(enabled_tools=tools, confirm_tools=["stop_job"])
    prompt = build_system_prompt(ctx)

    for tool in tools:
        assert tool not in prompt, f"Tool name '{tool}' should not appear in system prompt text"


def test_prompt_still_mentions_confirm_tools():
    """Confirm tools must still appear in prompt as a behavioral hint."""
    ctx = _make_context(enabled_tools=["stop_job", "get_job_detail"], confirm_tools=["stop_job"])
    prompt = build_system_prompt(ctx)

    assert "stop_job" in prompt, "confirm tool 'stop_job' should appear in prompt as behavioral hint"
