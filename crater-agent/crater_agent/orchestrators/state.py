"""Runtime state models for orchestrators."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from crater_agent.orchestrators.artifacts import GoalArtifact


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
            "total_tokens": self.total_tokens,
        }

    @classmethod
    def from_dict(cls, payload: dict[str, Any] | None) -> MultiAgentUsageSummary:
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
    def from_dict(cls, payload: dict[str, Any] | None) -> MultiAgentActionItem | None:
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
class MASRuntimeConfig:
    """Minimal runtime guardrails. No scenario-specific presets."""
    lead_max_rounds: int = 8
    subagent_max_iterations: int = 25
    no_progress_rounds: int = 2
    tool_timeout_seconds: int = 60
    max_actions_per_round: int = 4

    def to_dict(self) -> dict[str, Any]:
        return {
            "lead_max_rounds": self.lead_max_rounds,
            "subagent_max_iterations": self.subagent_max_iterations,
            "no_progress_rounds": self.no_progress_rounds,
            "tool_timeout_seconds": self.tool_timeout_seconds,
            "max_actions_per_round": self.max_actions_per_round,
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any] | None) -> MASRuntimeConfig:
        if not isinstance(data, dict):
            return cls()
        return cls(
            lead_max_rounds=int(data.get("lead_max_rounds") or 8),
            subagent_max_iterations=int(data.get("subagent_max_iterations") or 25),
            no_progress_rounds=int(data.get("no_progress_rounds") or 2),
            tool_timeout_seconds=int(data.get("tool_timeout_seconds") or 60),
            max_actions_per_round=int(data.get("max_actions_per_round") or 4),
        )


@dataclass
class MASState:
    """Per-turn state for the Coordinator multi-agent controller."""

    session_id: str
    turn_id: str

    # Goal (immutable input)
    goal: GoalArtifact

    # Stage outputs
    observation: "ObservationArtifact | None" = None
    plan: "PlanArtifact | None" = None
    execution: "ExecutionArtifact | None" = None

    # Request context (carried from request)
    capabilities: dict[str, Any] = field(default_factory=dict)
    continuation: dict[str, Any] = field(default_factory=dict)
    history: list[dict[str, Any]] = field(default_factory=list)
    workflow: dict[str, Any] = field(default_factory=dict)
    resume_context: dict[str, Any] = field(default_factory=dict)

    # Runtime
    runtime_config: MASRuntimeConfig = field(default_factory=MASRuntimeConfig)
    usage_summary: MultiAgentUsageSummary = field(default_factory=MultiAgentUsageSummary)

    # Flow control
    loop_round: int = 0
    no_progress_count: int = 0
    stop_reason: str = ""  # "completed" | "max_rounds" | "no_progress" | "awaiting_confirmation"
    final_answer: str = ""

    # Actions & evidence tracking
    actions: list[MultiAgentActionItem] = field(default_factory=list)
    action_history: list[dict[str, Any]] = field(default_factory=list)
    tool_records: list[MultiAgentToolRecord] = field(default_factory=list)
    attempted_tool_signatures: list[str] = field(default_factory=list)
    controller_trace: list[dict[str, Any]] = field(default_factory=list)

    # Confirmation
    pending_confirmation: dict[str, Any] | None = None

    # Clarification (from previous turn's final_answer continuation)
    clarification_context: dict[str, Any] = field(default_factory=dict)

    @classmethod
    def from_request(cls, request: Any) -> MASState:
        """Build MASState from incoming ChatRequest. GoalArtifact is populated later by IntentRouter."""
        context = dict(request.context or {})
        continuation = dict(context.get("continuation") or {})
        resume_context = dict(continuation.get("resume_after_confirmation") or {})
        workflow = dict(
            resume_context.get("workflow")
            or continuation.get("workflow")
            or {}
        )
        actor = dict(context.get("actor") or {})
        page_context = dict(context.get("page") or {})
        actor_role = str(actor.get("role") or "user").strip().lower() or "user"

        # Build initial goal (routing will be set by IntentRouter)
        goal = GoalArtifact(
            user_message=request.message,
            original_user_message=str(workflow.get("original_user_message") or "").strip() or request.message,
            actor_role=actor_role,
            page_context=page_context,
        )

        state = cls(
            session_id=request.session_id,
            turn_id=request.turn_id,
            goal=goal,
            capabilities=dict(context.get("capabilities") or {}),
            continuation=continuation,
            history=list(context.get("history") or []),
            workflow=workflow,
            resume_context=resume_context,
            clarification_context=dict(continuation.get("clarification") or {}),
        )

        # Restore from checkpoint if resuming
        state._restore_from_workflow(workflow)

        return state

    def _restore_from_workflow(self, workflow: dict[str, Any]) -> None:
        """Restore state from a persisted workflow checkpoint."""
        if not workflow or not isinstance(workflow, dict):
            return

        # Restore artifacts
        from crater_agent.orchestrators.artifacts import (
            ObservationArtifact,
            PlanArtifact,
            ExecutionArtifact,
            RoutingDecision,
        )
        self.observation = ObservationArtifact.from_dict(workflow.get("observation"))
        self.plan = PlanArtifact.from_dict(workflow.get("plan"))
        self.execution = ExecutionArtifact.from_dict(workflow.get("execution"))

        routing_data = workflow.get("routing")
        if routing_data:
            self.goal.routing = RoutingDecision.from_dict(routing_data)

        # Restore flow control
        self.loop_round = int(workflow.get("loop_round") or 0)
        self.no_progress_count = int(workflow.get("no_progress_count") or 0)
        self.stop_reason = str(workflow.get("stop_reason") or "").strip()

        # Restore runtime config
        self.runtime_config = MASRuntimeConfig.from_dict(workflow.get("runtime_config"))
        self.usage_summary = MultiAgentUsageSummary.from_dict(workflow.get("usage_summary"))

        # Restore tracking
        self.action_history = list(workflow.get("action_history") or [])
        self.controller_trace = list(workflow.get("controller_trace") or [])
        self.attempted_tool_signatures = [
            str(item).strip()
            for item in (workflow.get("attempted_tool_signatures") or [])
            if str(item).strip()
        ]
        self.actions = [
            action for action in (
                MultiAgentActionItem.from_dict(item)
                for item in (workflow.get("actions") or [])
            )
            if action is not None
        ]

    @property
    def enabled_tools(self) -> list[str]:
        return list(self.capabilities.get("enabled_tools") or [])

    def build_state_view(self, for_role: str) -> "StateView":
        from crater_agent.orchestrators.artifacts import StateView
        base = StateView(
            goal=self.goal,
            loop_round=self.loop_round,
            max_rounds=self.runtime_config.lead_max_rounds,
        )
        if for_role == "planner":
            base.observation = self.observation
        elif for_role == "explorer":
            base.plan = self.plan
        elif for_role == "executor":
            base.observation = self.observation
            base.plan = self.plan
        elif for_role == "coordinator":
            base.observation = self.observation
            base.plan = self.plan
            base.execution = self.execution
            # Include evidence from observation artifact so Coordinator
            # has access to actual tool results for final summarization
            if self.observation and self.observation.evidence:
                base.compact_evidence = list(self.observation.evidence)
        return base

    def recent_history_excerpt(self, *, max_items: int = 6, max_chars: int = 800) -> str:
        if not self.history:
            return ""
        lines: list[str] = []
        for item in self.history[-max_items:]:
            role = str(item.get("role") or "unknown").strip()
            content = str(item.get("content") or "").strip()
            if not content:
                continue
            if len(content) > max_chars:
                content = content[:max_chars] + "..."
            lines.append(f"[{role}] {content}")
        return "\n".join(lines)

    def recent_history_context(
        self,
        *,
        max_items: int = 12,
        max_chars_per_item: int = 1200,
        max_total_chars: int = 6000,
    ) -> str:
        if not self.history:
            return ""
        selected: list[str] = []
        total_chars = 0
        for item in reversed(self.history[-max_items:]):
            role = str(item.get("role") or "unknown").strip()
            content = str(item.get("content") or "").strip()
            if not content:
                continue
            if len(content) > max_chars_per_item:
                content = content[:max_chars_per_item] + "..."
            line = f"[{role}] {content}"
            if selected and total_chars + len(line) > max_total_chars:
                break
            if not selected and len(line) > max_total_chars:
                line = line[:max_total_chars] + "..."
            selected.append(line)
            total_chars += len(line)
        return "\n".join(reversed(selected))

    def remember_tool(self, *, agent_id: str, agent_role: str, tool_name: str, tool_args: dict[str, Any], tool_call_id: str, result: dict[str, Any]) -> None:
        self.tool_records.append(
            MultiAgentToolRecord(agent_id=agent_id, agent_role=agent_role, tool_name=tool_name, tool_args=tool_args, tool_call_id=tool_call_id, result=result)
        )
        self.usage_summary.evidence_items = len(self.tool_records)

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

    def record_action_result(self, *, action: MultiAgentActionItem, result_status: str, result: dict[str, Any] | None, confirmed: bool | None = None) -> None:
        self.action_history.append({
            "action_id": action.action_id,
            "tool_name": action.tool_name,
            "tool_args": action.tool_args,
            "title": action.title,
            "reason": action.reason,
            "status": result_status,
            "confirmed": confirmed,
            "confirm_id": action.confirm_id,
            "result": result,
        })

    def action_frontier(self) -> list[MultiAgentActionItem]:
        completed = {item.action_id for item in self.actions if item.status == "completed"}
        blocked = {item.action_id for item in self.actions if item.status in {"error", "rejected", "skipped"}}
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
            "version": 3,
            "original_user_message": self.goal.original_user_message,
            "routing": self.goal.routing.to_dict(),
            "observation": self.observation.to_dict() if self.observation else None,
            "plan": self.plan.to_dict() if self.plan else None,
            "execution": self.execution.to_dict() if self.execution else None,
            "loop_round": self.loop_round,
            "no_progress_count": self.no_progress_count,
            "runtime_config": self.runtime_config.to_dict(),
            "usage_summary": self.usage_summary.to_dict(),
            "stop_reason": self.stop_reason,
            "actions": [a.to_dict() for a in self.actions],
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
                self.record_action_result(action=action, result_status=action.status, result=result, confirmed=self.resume_context.get("confirmed"))
            return {
                "action_id": action.action_id,
                "tool_name": action.tool_name,
                "status": action.status,
                "confirm_id": action.confirm_id,
                "result": result,
            }
        return None
