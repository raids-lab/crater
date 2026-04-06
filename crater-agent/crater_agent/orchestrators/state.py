"""Runtime state models for orchestrators."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


@dataclass
class MultiAgentUsageSummary:
    """Runtime usage summary tracked across a multi-agent turn."""

    llm_calls: int = 0
    llm_input_tokens: int = 0
    llm_output_tokens: int = 0
    tool_calls: int = 0
    read_tool_calls: int = 0
    write_tool_calls: int = 0
    evidence_items: int = 0
    verification_calls: int = 0

    @property
    def total_tokens(self) -> int:
        return self.llm_input_tokens + self.llm_output_tokens

    def to_dict(self) -> dict[str, Any]:
        return {
            "llm_calls": self.llm_calls,
            "llm_input_tokens": self.llm_input_tokens,
            "llm_output_tokens": self.llm_output_tokens,
            "tool_calls": self.tool_calls,
            "read_tool_calls": self.read_tool_calls,
            "write_tool_calls": self.write_tool_calls,
            "evidence_items": self.evidence_items,
            "verification_calls": self.verification_calls,
            "total_tokens": self.total_tokens,
        }

    @classmethod
    def from_dict(cls, payload: dict[str, Any] | None) -> "MultiAgentUsageSummary":
        if not isinstance(payload, dict):
            return cls()
        return cls(
            llm_calls=int(payload.get("llm_calls") or 0),
            llm_input_tokens=int(payload.get("llm_input_tokens") or 0),
            llm_output_tokens=int(payload.get("llm_output_tokens") or 0),
            tool_calls=int(payload.get("tool_calls") or 0),
            read_tool_calls=int(payload.get("read_tool_calls") or 0),
            write_tool_calls=int(payload.get("write_tool_calls") or 0),
            evidence_items=int(payload.get("evidence_items") or 0),
            verification_calls=int(payload.get("verification_calls") or 0),
        )


@dataclass
class MultiAgentRuntimeGuard:
    """Scenario-specific runtime guard for the multi-agent controller loop."""

    scenario: str = "query"
    max_loop_iterations: int = 4
    max_replans: int = 1
    max_frontier_actions: int = 2
    max_read_only_explore_rounds: int = 1
    max_read_tool_calls: int = 3
    max_evidence_items: int = 4
    max_verifications: int = 1
    max_no_progress_rounds: int = 1
    max_budget_tokens: int = 6000
    structured_retry_limit: int = 1

    def to_dict(self) -> dict[str, Any]:
        return {
            "scenario": self.scenario,
            "max_loop_iterations": self.max_loop_iterations,
            "max_replans": self.max_replans,
            "max_frontier_actions": self.max_frontier_actions,
            "max_read_only_explore_rounds": self.max_read_only_explore_rounds,
            "max_read_tool_calls": self.max_read_tool_calls,
            "max_evidence_items": self.max_evidence_items,
            "max_verifications": self.max_verifications,
            "max_no_progress_rounds": self.max_no_progress_rounds,
            "max_budget_tokens": self.max_budget_tokens,
            "structured_retry_limit": self.structured_retry_limit,
        }

    @classmethod
    def from_dict(cls, payload: dict[str, Any] | None) -> "MultiAgentRuntimeGuard | None":
        if not isinstance(payload, dict):
            return None
        scenario = str(payload.get("scenario") or "").strip().lower()
        if not scenario:
            return None
        return cls(
            scenario=scenario,
            max_loop_iterations=int(payload.get("max_loop_iterations") or 4),
            max_replans=int(payload.get("max_replans") or 1),
            max_frontier_actions=int(payload.get("max_frontier_actions") or 2),
            max_read_only_explore_rounds=int(payload.get("max_read_only_explore_rounds") or 1),
            max_read_tool_calls=int(payload.get("max_read_tool_calls") or 3),
            max_evidence_items=int(payload.get("max_evidence_items") or 4),
            max_verifications=int(payload.get("max_verifications") or 1),
            max_no_progress_rounds=int(payload.get("max_no_progress_rounds") or 1),
            max_budget_tokens=int(payload.get("max_budget_tokens") or 6000),
            structured_retry_limit=int(payload.get("structured_retry_limit") or 1),
        )


@dataclass
class MultiAgentRoleOutput:
    """Normalized role output stored in the turn state."""

    agent_id: str
    agent_role: str
    summary: str
    status: str = "completed"
    metadata: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        return {
            "agent_id": self.agent_id,
            "agent_role": self.agent_role,
            "summary": self.summary,
            "status": self.status,
            "metadata": self.metadata,
        }

    @classmethod
    def from_dict(cls, payload: dict[str, Any] | None) -> "MultiAgentRoleOutput | None":
        if not isinstance(payload, dict):
            return None
        agent_id = str(payload.get("agent_id") or "").strip()
        agent_role = str(payload.get("agent_role") or "").strip()
        summary = str(payload.get("summary") or "").strip()
        if not agent_id or not agent_role:
            return None
        return cls(
            agent_id=agent_id,
            agent_role=agent_role,
            summary=summary,
            status=str(payload.get("status") or "completed").strip() or "completed",
            metadata=dict(payload.get("metadata") or {}),
        )


@dataclass
class MultiAgentActionItem:
    """A planned write action tracked across loop iterations and resumes."""

    action_id: str
    tool_name: str
    tool_args: dict[str, Any]
    title: str = ""
    reason: str = ""
    depends_on: list[str] = field(default_factory=list)
    status: str = "pending"
    confirm_id: str = ""
    result: dict[str, Any] | None = None

    def to_dict(self) -> dict[str, Any]:
        return {
            "action_id": self.action_id,
            "tool_name": self.tool_name,
            "tool_args": self.tool_args,
            "title": self.title,
            "reason": self.reason,
            "depends_on": list(self.depends_on),
            "status": self.status,
            "confirm_id": self.confirm_id,
            "result": self.result,
        }

    @classmethod
    def from_dict(cls, payload: dict[str, Any] | None) -> "MultiAgentActionItem | None":
        if not isinstance(payload, dict):
            return None
        action_id = str(payload.get("action_id") or "").strip()
        tool_name = str(payload.get("tool_name") or "").strip()
        if not action_id or not tool_name:
            return None
        tool_args = payload.get("tool_args") or {}
        if not isinstance(tool_args, dict):
            tool_args = {}
        depends_on = payload.get("depends_on") or []
        return cls(
            action_id=action_id,
            tool_name=tool_name,
            tool_args=tool_args,
            title=str(payload.get("title") or "").strip(),
            reason=str(payload.get("reason") or "").strip(),
            depends_on=[str(item).strip() for item in depends_on if str(item).strip()],
            status=str(payload.get("status") or "pending").strip() or "pending",
            confirm_id=str(payload.get("confirm_id") or "").strip(),
            result=payload.get("result") if isinstance(payload.get("result"), dict) else None,
        )


@dataclass
class MultiAgentToolRecord:
    """A tool execution captured as part of the turn working state."""

    agent_id: str
    agent_role: str
    tool_name: str
    tool_args: dict[str, Any]
    tool_call_id: str
    result: dict[str, Any] | None = None


@dataclass
class MultiAgentTurnState:
    """Ephemeral per-turn state for the loop-based multi-agent controller."""

    session_id: str
    turn_id: str
    user_message: str
    actor: dict[str, Any] = field(default_factory=dict)
    page_context: dict[str, Any] = field(default_factory=dict)
    capabilities: dict[str, Any] = field(default_factory=dict)
    continuation: dict[str, Any] = field(default_factory=dict)
    history: list[dict[str, Any]] = field(default_factory=list)
    workflow: dict[str, Any] = field(default_factory=dict)
    resume_context: dict[str, Any] = field(default_factory=dict)
    route: str = "diagnostic"
    root_agent_id: str = "coordinator-1"
    plan: MultiAgentRoleOutput | None = None
    exploration: MultiAgentRoleOutput | None = None
    execution: MultiAgentRoleOutput | None = None
    verification: MultiAgentRoleOutput | None = None
    final_answer: str = ""
    evidence: list[dict[str, Any]] = field(default_factory=list)
    tool_records: list[MultiAgentToolRecord] = field(default_factory=list)
    actions: list[MultiAgentActionItem] = field(default_factory=list)
    action_history: list[dict[str, Any]] = field(default_factory=list)
    controller_trace: list[dict[str, Any]] = field(default_factory=list)
    attempted_tool_signatures: list[str] = field(default_factory=list)
    loop_iteration: int = 0
    replan_count: int = 0
    runtime_scenario: str = ""
    runtime_guard: MultiAgentRuntimeGuard | None = None
    usage_summary: MultiAgentUsageSummary = field(default_factory=MultiAgentUsageSummary)
    stop_reason: str = ""
    pending_confirmation: dict[str, Any] | None = None

    @classmethod
    def from_request(cls, request: Any) -> "MultiAgentTurnState":
        context = dict(request.context or {})
        continuation = dict(context.get("continuation") or {})
        resume_context = dict(continuation.get("resume_after_confirmation") or {})
        workflow = dict(
            resume_context.get("workflow")
            or continuation.get("workflow")
            or {}
        )
        state = cls(
            session_id=request.session_id,
            turn_id=request.turn_id,
            user_message=request.message,
            actor=dict(context.get("actor") or {}),
            page_context=dict(context.get("page") or {}),
            capabilities=dict(context.get("capabilities") or {}),
            continuation=continuation,
            history=list(context.get("history") or []),
            workflow=workflow,
            resume_context=resume_context,
        )
        state.route = str(workflow.get("route") or state.route).strip() or "diagnostic"
        state.plan = MultiAgentRoleOutput.from_dict(workflow.get("plan"))
        state.exploration = MultiAgentRoleOutput.from_dict(workflow.get("exploration"))
        state.execution = MultiAgentRoleOutput.from_dict(workflow.get("execution"))
        state.verification = MultiAgentRoleOutput.from_dict(workflow.get("verification"))
        state.evidence = list(workflow.get("evidence") or [])
        state.action_history = list(workflow.get("action_history") or [])
        state.controller_trace = list(workflow.get("controller_trace") or [])
        state.attempted_tool_signatures = [
            str(item).strip()
            for item in (workflow.get("attempted_tool_signatures") or [])
            if str(item).strip()
        ]
        state.loop_iteration = int(workflow.get("loop_iteration") or 0)
        state.replan_count = int(workflow.get("replan_count") or 0)
        state.runtime_scenario = str(workflow.get("runtime_scenario") or "").strip().lower()
        state.runtime_guard = MultiAgentRuntimeGuard.from_dict(workflow.get("runtime_guard"))
        state.usage_summary = MultiAgentUsageSummary.from_dict(workflow.get("usage_summary"))
        state.stop_reason = str(workflow.get("stop_reason") or "").strip().lower()
        state.actions = [
            action
            for action in (
                MultiAgentActionItem.from_dict(item)
                for item in (workflow.get("actions") or [])
            )
            if action is not None
        ]
        return state

    @property
    def enabled_tools(self) -> list[str]:
        return list(self.capabilities.get("enabled_tools") or [])

    @property
    def actor_role(self) -> str:
        return str((self.actor.get("role") or "user")).strip().lower() or "user"

    @property
    def original_user_message(self) -> str:
        workflow_message = str(self.workflow.get("original_user_message") or "").strip()
        return workflow_message or self.user_message

    def recent_history_excerpt(self, *, max_items: int = 6, max_chars: int = 800) -> str:
        if not self.history:
            return ""
        lines: list[str] = []
        for item in self.history[-max_items:]:
            role = str(item.get("role") or "unknown").strip() or "unknown"
            content = str(item.get("content") or "").strip()
            if not content:
                continue
            if len(content) > max_chars:
                content = content[:max_chars] + "..."
            lines.append(f"[{role}] {content}")
        return "\n".join(lines)

    def remember_tool(
        self,
        *,
        agent_id: str,
        agent_role: str,
        tool_name: str,
        tool_args: dict[str, Any],
        tool_call_id: str,
        result: dict[str, Any],
    ) -> None:
        self.tool_records.append(
            MultiAgentToolRecord(
                agent_id=agent_id,
                agent_role=agent_role,
                tool_name=tool_name,
                tool_args=tool_args,
                tool_call_id=tool_call_id,
                result=result,
            )
        )
        self.evidence.append(
            {
                "agent_id": agent_id,
                "agent_role": agent_role,
                "tool_name": tool_name,
                "tool_args": tool_args,
                "result": result,
            }
        )
        self.usage_summary.evidence_items = len(self.evidence)

    def remember_controller_decision(self, decision: dict[str, Any]) -> None:
        self.controller_trace.append(decision)
        if len(self.controller_trace) > 16:
            self.controller_trace = self.controller_trace[-16:]

    def record_llm_usage(self, usage: dict[str, Any] | None) -> None:
        if not isinstance(usage, dict):
            return
        self.usage_summary.llm_calls += int(usage.get("llm_calls") or 0)
        self.usage_summary.llm_input_tokens += int(usage.get("input_tokens") or 0)
        self.usage_summary.llm_output_tokens += int(usage.get("output_tokens") or 0)

    def record_action_result(
        self,
        *,
        action: MultiAgentActionItem,
        result_status: str,
        result: dict[str, Any] | None,
        confirmed: bool | None = None,
    ) -> None:
        self.action_history.append(
            {
                "action_id": action.action_id,
                "tool_name": action.tool_name,
                "tool_args": action.tool_args,
                "title": action.title,
                "reason": action.reason,
                "status": result_status,
                "confirmed": confirmed,
                "confirm_id": action.confirm_id,
                "result": result,
            }
        )

    def action_frontier(self) -> list[MultiAgentActionItem]:
        completed = {item.action_id for item in self.actions if item.status == "completed"}
        blocked = {
            item.action_id
            for item in self.actions
            if item.status in {"error", "rejected", "skipped"}
        }
        frontier: list[MultiAgentActionItem] = []
        for action in self.actions:
            if action.status != "pending":
                continue
            if any(dep in blocked for dep in action.depends_on):
                action.status = "skipped"
                continue
            if all(dep in completed for dep in action.depends_on):
                frontier.append(action)
        return frontier

    def build_workflow_checkpoint(self) -> dict[str, Any]:
        return {
            "version": 2,
            "original_user_message": self.original_user_message,
            "route": self.route,
            "loop_iteration": self.loop_iteration,
            "replan_count": self.replan_count,
            "runtime_scenario": self.runtime_scenario,
            "runtime_guard": self.runtime_guard.to_dict() if self.runtime_guard else None,
            "usage_summary": self.usage_summary.to_dict(),
            "stop_reason": self.stop_reason,
            "plan": self.plan.to_dict() if self.plan else None,
            "exploration": self.exploration.to_dict() if self.exploration else None,
            "execution": self.execution.to_dict() if self.execution else None,
            "verification": self.verification.to_dict() if self.verification else None,
            "evidence": list(self.evidence),
            "actions": [action.to_dict() for action in self.actions],
            "action_history": list(self.action_history),
            "controller_trace": list(self.controller_trace),
            "attempted_tool_signatures": list(self.attempted_tool_signatures),
        }

    def apply_resume_outcome(self) -> dict[str, Any] | None:
        if not self.resume_context:
            return None

        confirm_id = str(self.resume_context.get("confirm_id") or "").strip()
        result_status = str(self.resume_context.get("result_status") or "").strip() or "completed"
        result = self.resume_context.get("result")
        if not isinstance(result, dict):
            result = {"result": result} if result is not None else {}

        for action in self.actions:
            if confirm_id:
                if action.confirm_id != confirm_id:
                    continue
            elif action.status != "awaiting_confirmation":
                continue

            if action.confirm_id == "" and confirm_id:
                action.confirm_id = confirm_id
            action.result = result
            if result_status in {"success", "completed"}:
                action.status = "completed"
            elif result_status == "rejected":
                action.status = "rejected"
            else:
                action.status = "error"

            if not any(
                str(item.get("confirm_id") or "").strip() == action.confirm_id and action.confirm_id
                for item in self.action_history
            ):
                self.record_action_result(
                    action=action,
                    result_status=action.status,
                    result=result,
                    confirmed=self.resume_context.get("confirmed"),
                )

            return {
                "action_id": action.action_id,
                "tool_name": action.tool_name,
                "status": action.status,
                "confirm_id": action.confirm_id,
                "result": result,
            }
        return None
