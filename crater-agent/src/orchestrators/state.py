"""Runtime state models for orchestrators."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import TYPE_CHECKING, Any

from orchestrators.artifacts import GoalArtifact
from tools.definitions import CONFIRM_TOOL_NAMES
from tools.tool_selector import (
    _resolve_actor_role,
    sanitize_capabilities_for_context,
    sanitize_enabled_tool_names,
)

if TYPE_CHECKING:
    from orchestrators.artifacts import (
        ExecutionArtifact,
        ObservationArtifact,
        PlanArtifact,
        StateView,
    )


def _normalize_source_turn_event(event: dict[str, Any]) -> dict[str, Any] | None:
    event_type = str(event.get("event_type") or event.get("eventType") or "").strip()
    if not event_type:
        return None
    metadata = event.get("metadata") if isinstance(event.get("metadata"), dict) else {}
    content = str(event.get("content") or metadata.get("content") or metadata.get("summary") or "").strip()
    if not content:
        if result_summary := str(metadata.get("resultSummary") or "").strip():
            content = result_summary
        elif description := str(metadata.get("description") or "").strip():
            content = description
    role = "assistant"
    prefix = event_type
    if event_type.startswith("tool_call"):
        role = "tool"
        tool_name = str(metadata.get("toolName") or metadata.get("action") or event.get("title") or "").strip()
        status = str(event.get("status") or metadata.get("status") or "").strip()
        parts = [f"event={event_type}"]
        if tool_name:
            parts.append(f"tool={tool_name}")
        if status:
            parts.append(f"status={status}")
        if metadata.get("toolArgs") is not None:
            parts.append(f"args={metadata.get('toolArgs')}")
        if content:
            parts.append(f"result={content}")
        content = " ; ".join(parts)
    else:
        agent_role = str(event.get("agent_role") or metadata.get("agentRole") or "").strip()
        if agent_role:
            prefix = f"{event_type}/{agent_role}"
        content = f"[{prefix}] {content}" if content else f"[{prefix}]"
    if not content:
        return None
    return {
        "role": role,
        "content": content,
        "tool_call_id": str(metadata.get("toolCallId") or metadata.get("tool_call_id") or "").strip(),
    }


def _source_turn_history_from_continuation(continuation: dict[str, Any]) -> list[dict[str, Any]]:
    source_context = continuation.get("source_turn_context")
    if not isinstance(source_context, dict):
        resume = continuation.get("resume_after_confirmation")
        source_context = resume.get("source_turn_context") if isinstance(resume, dict) else None
    if not isinstance(source_context, dict):
        return []
    events = source_context.get("events")
    if not isinstance(events, list):
        return []
    normalized: list[dict[str, Any]] = []
    for item in events:
        if isinstance(item, dict):
            event = _normalize_source_turn_event(item)
            if event:
                normalized.append(event)
    return normalized


def _source_turn_context_from_continuation(continuation: dict[str, Any]) -> dict[str, Any]:
    source_context = continuation.get("source_turn_context")
    if isinstance(source_context, dict):
        return source_context
    resume = continuation.get("resume_after_confirmation")
    if isinstance(resume, dict) and isinstance(resume.get("source_turn_context"), dict):
        return resume["source_turn_context"]
    return {}


def _tool_signature(tool_name: str, tool_args: dict[str, Any]) -> str:
    import json

    try:
        args = json.dumps(tool_args or {}, ensure_ascii=False, sort_keys=True)
    except TypeError:
        args = str(tool_args or {})
    return f"{tool_name}:{args}"


@dataclass
class MultiAgentUsageSummary:
    """Runtime usage summary tracked across a multi-agent turn."""

    llm_calls: int = 0
    llm_input_tokens: int = 0
    llm_output_tokens: int = 0
    llm_total_tokens: int = 0
    llm_reported_token_calls: int = 0
    llm_missing_token_calls: int = 0
    llm_latency_ms: int = 0
    tool_latency_ms: int = 0
    tool_calls: int = 0
    read_tool_calls: int = 0
    write_tool_calls: int = 0
    evidence_items: int = 0
    llm_by_role: dict[str, dict[str, int]] = field(default_factory=dict)

    @property
    def total_tokens(self) -> int:
        return self.llm_total_tokens or (self.llm_input_tokens + self.llm_output_tokens)

    def to_dict(self) -> dict[str, Any]:
        return {
            "llm_calls": self.llm_calls,
            "llm_input_tokens": self.llm_input_tokens,
            "llm_output_tokens": self.llm_output_tokens,
            "llm_total_tokens": self.llm_total_tokens,
            "llm_reported_token_calls": self.llm_reported_token_calls,
            "llm_missing_token_calls": self.llm_missing_token_calls,
            "reported_token_coverage": (
                self.llm_reported_token_calls / self.llm_calls
                if self.llm_calls
                else 0.0
            ),
            "llm_latency_ms": self.llm_latency_ms,
            "tool_latency_ms": self.tool_latency_ms,
            "tool_calls": self.tool_calls,
            "read_tool_calls": self.read_tool_calls,
            "write_tool_calls": self.write_tool_calls,
            "evidence_items": self.evidence_items,
            "llm_by_role": {
                role: dict(values)
                for role, values in sorted(self.llm_by_role.items())
            },
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
            llm_total_tokens=int(payload.get("llm_total_tokens") or payload.get("total_tokens") or 0),
            llm_reported_token_calls=int(payload.get("llm_reported_token_calls") or 0),
            llm_missing_token_calls=int(payload.get("llm_missing_token_calls") or 0),
            llm_latency_ms=int(payload.get("llm_latency_ms") or 0),
            tool_latency_ms=int(payload.get("tool_latency_ms") or 0),
            tool_calls=int(payload.get("tool_calls") or 0),
            read_tool_calls=int(payload.get("read_tool_calls") or 0),
            write_tool_calls=int(payload.get("write_tool_calls") or 0),
            evidence_items=int(payload.get("evidence_items") or 0),
            llm_by_role={
                str(role): {
                    "llm_calls": int((values or {}).get("llm_calls") or 0),
                    "input_tokens": int((values or {}).get("input_tokens") or 0),
                    "output_tokens": int((values or {}).get("output_tokens") or 0),
                    "total_tokens": int((values or {}).get("total_tokens") or 0),
                    "reported_token_calls": int((values or {}).get("reported_token_calls") or 0),
                    "missing_token_calls": int((values or {}).get("missing_token_calls") or 0),
                    "latency_ms": int((values or {}).get("latency_ms") or 0),
                }
                for role, values in (
                    payload.get("llm_by_role")
                    if isinstance(payload.get("llm_by_role"), dict)
                    else {}
                ).items()
                if isinstance(values, dict)
            },
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

    def to_dict(self) -> dict[str, Any]:
        return {
            "agent_id": self.agent_id,
            "agent_role": self.agent_role,
            "tool_name": self.tool_name,
            "tool_args": dict(self.tool_args),
            "tool_call_id": self.tool_call_id,
            "result": self.result,
        }

    @classmethod
    def from_dict(cls, payload: dict[str, Any] | None) -> MultiAgentToolRecord | None:
        if not isinstance(payload, dict):
            return None
        tool_name = str(payload.get("tool_name") or "").strip()
        tool_call_id = str(payload.get("tool_call_id") or "").strip()
        if not tool_name or not tool_call_id:
            return None
        tool_args = payload.get("tool_args") if isinstance(payload.get("tool_args"), dict) else {}
        return cls(
            agent_id=str(payload.get("agent_id") or "").strip(),
            agent_role=str(payload.get("agent_role") or "").strip(),
            tool_name=tool_name,
            tool_args=tool_args,
            tool_call_id=tool_call_id,
            result=payload.get("result") if isinstance(payload.get("result"), dict) else None,
        )


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

    # Confirmations waiting for user approval. A single confirmation is stored
    # as a one-item list so MAS uses the same shape as the single-agent graph.
    pending_confirmations: list[dict[str, Any]] = field(default_factory=list)

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
        original_user_message = (
            str(workflow.get("original_user_message") or "").strip()
            or str(resume_context.get("original_user_message") or "").strip()
            or str(continuation.get("original_user_message") or "").strip()
        )
        page_context = dict(context.get("page") or {})
        actor_role = _resolve_actor_role(context)
        sanitized_capabilities = sanitize_capabilities_for_context(context, context.get("capabilities"))

        # Build initial goal (routing will be set by IntentRouter)
        goal = GoalArtifact(
            user_message=request.message,
            original_user_message=original_user_message or request.message,
            actor_role=actor_role,
            page_context=page_context,
        )

        state = cls(
            session_id=request.session_id,
            turn_id=request.turn_id,
            goal=goal,
            capabilities=sanitized_capabilities,
            continuation=continuation,
            history=list(context.get("history") or []),
            workflow=workflow,
            resume_context=resume_context,
            clarification_context=dict(continuation.get("clarification") or {}),
        )
        if resume_context:
            source_history = _source_turn_history_from_continuation(continuation)
            if source_history:
                state.history.extend(source_history)

        # Restore from checkpoint if resuming
        state._restore_from_workflow(workflow)
        if resume_context:
            state._restore_source_turn_tool_context(
                _source_turn_context_from_continuation(continuation)
            )
        if resume_context and not state.pending_confirmations:
            pending_confirmations = continuation.get("pending_confirmations")
            if isinstance(pending_confirmations, list):
                state.pending_confirmations = [
                    item for item in pending_confirmations if isinstance(item, dict)
                ]
            elif isinstance(continuation.get("pending_confirmation"), dict):
                state.pending_confirmations = [continuation["pending_confirmation"]]

        return state

    def _restore_from_workflow(self, workflow: dict[str, Any]) -> None:
        """Restore state from a persisted workflow checkpoint."""
        if not workflow or not isinstance(workflow, dict):
            return

        # Restore artifacts
        from orchestrators.artifacts import (
            ExecutionArtifact,
            ObservationArtifact,
            PlanArtifact,
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
        self.tool_records = [
            record for record in (
                MultiAgentToolRecord.from_dict(item)
                for item in (workflow.get("tool_records") or [])
            )
            if record is not None
        ]
        self.actions = [
            action for action in (
                MultiAgentActionItem.from_dict(item)
                for item in (workflow.get("actions") or [])
            )
            if action is not None
        ]
        if self.resume_context:
            writable_signatures = {
                _tool_signature(action.tool_name, action.tool_args)
                for action in self.actions
                if action.tool_name in CONFIRM_TOOL_NAMES
            }
            self.attempted_tool_signatures = [
                signature
                for signature in self.attempted_tool_signatures
                if signature in writable_signatures
            ]
        pending_confirmations = workflow.get("pending_confirmations")
        if isinstance(pending_confirmations, list):
            self.pending_confirmations = [
                item for item in pending_confirmations if isinstance(item, dict)
            ]
        elif isinstance(workflow.get("pending_confirmation"), dict):
            self.pending_confirmations = [workflow["pending_confirmation"]]

    def _restore_source_turn_tool_context(self, source_context: dict[str, Any]) -> None:
        if not isinstance(source_context, dict):
            return
        tool_calls = source_context.get("tool_calls")
        if not isinstance(tool_calls, list):
            return

        existing_tool_call_ids = {
            record.tool_call_id for record in self.tool_records if record.tool_call_id
        }
        existing_signatures = set(self.attempted_tool_signatures)
        existing_action_confirm_ids = {
            str(item.get("confirm_id") or "").strip()
            for item in self.action_history
            if isinstance(item, dict)
        }
        for index, item in enumerate(tool_calls, start=1):
            if not isinstance(item, dict):
                continue
            tool_name = str(item.get("tool_name") or item.get("toolName") or "").strip()
            if not tool_name:
                continue
            tool_args = item.get("tool_args") if isinstance(item.get("tool_args"), dict) else {}
            tool_call_id = str(
                item.get("tool_call_id")
                or item.get("toolCallId")
                or f"source-turn-tool-{index}"
            ).strip()
            signature = _tool_signature(tool_name, tool_args)
            # Source-turn read tools describe the state before a confirmed
            # action. Keep them as evidence, but do not let their signatures
            # block post-action rechecks in the resume turn. Write/confirm
            # tools remain protected so a resume cannot repeat the action.
            if tool_name in CONFIRM_TOOL_NAMES and signature not in existing_signatures:
                self.attempted_tool_signatures.append(signature)
                existing_signatures.add(signature)

            result_status = str(item.get("result_status") or item.get("resultStatus") or "").strip()
            result = item.get("result") if isinstance(item.get("result"), dict) else {}
            if (
                result_status in {"success", "completed", "error", "failed"}
                and result_status != "skipped"
                and tool_call_id not in existing_tool_call_ids
            ):
                self.tool_records.append(
                    MultiAgentToolRecord(
                        agent_id=str(item.get("agent_id") or item.get("agentId") or "").strip(),
                        agent_role=str(item.get("agent_role") or item.get("agentRole") or "").strip(),
                        tool_name=tool_name,
                        tool_args=tool_args,
                        tool_call_id=tool_call_id,
                        result=result,
                    )
                )
                existing_tool_call_ids.add(tool_call_id)

            confirm_id = str(item.get("id") or item.get("confirm_id") or item.get("confirmId") or "").strip()
            if (
                result_status in {"success", "completed", "error", "failed", "rejected"}
                and confirm_id
                and confirm_id not in existing_action_confirm_ids
            ):
                action = next((a for a in self.actions if a.confirm_id == confirm_id), None)
                if action is None:
                    action = next(
                        (
                            a for a in self.actions
                            if a.tool_name == tool_name and a.tool_args == tool_args
                        ),
                        None,
                    )
                self.action_history.append({
                    "action_id": action.action_id if action else "",
                    "tool_name": tool_name,
                    "tool_args": tool_args,
                    "title": action.title if action else "",
                    "reason": action.reason if action else "source_turn_tool_call",
                    "status": "completed" if result_status == "success" else result_status,
                    "confirmed": item.get("confirmed"),
                    "confirm_id": confirm_id,
                    "result": result,
                })
                existing_action_confirm_ids.add(confirm_id)

        self.usage_summary.evidence_items = len(self.tool_records)

    @property
    def enabled_tools(self) -> list[str]:
        return sanitize_enabled_tool_names(
            {"actor": {"role": self.goal.actor_role}, "capabilities": self.capabilities, "page": self.goal.page_context},
            self.capabilities.get("enabled_tools") or [],
        )

    def build_state_view(self, for_role: str) -> "StateView":
        from orchestrators.artifacts import StateView
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
        # Sliding window eviction: keep the most recent records to bound context growth
        _MAX_EVIDENCE_ITEMS = 30
        if len(self.tool_records) > _MAX_EVIDENCE_ITEMS:
            self.tool_records = self.tool_records[-_MAX_EVIDENCE_ITEMS:]
        self.usage_summary.evidence_items = len(self.tool_records)

    def remember_controller_decision(self, decision: dict[str, Any]) -> None:
        self.controller_trace.append(decision)
        if len(self.controller_trace) > 16:
            self.controller_trace = self.controller_trace[-16:]

    def record_llm_usage(self, usage: dict[str, Any] | None, *, role: str = "") -> None:
        if not isinstance(usage, dict):
            return
        self.usage_summary.llm_calls += int(usage.get("llm_calls") or 0)
        self.usage_summary.llm_input_tokens += int(usage.get("input_tokens") or 0)
        self.usage_summary.llm_output_tokens += int(usage.get("output_tokens") or 0)
        self.usage_summary.llm_total_tokens += int(
            usage.get("total_tokens")
            or (
                int(usage.get("input_tokens") or 0)
                + int(usage.get("output_tokens") or 0)
            )
        )
        self.usage_summary.llm_reported_token_calls += int(usage.get("reported_token_calls") or 0)
        self.usage_summary.llm_missing_token_calls += int(usage.get("missing_token_calls") or 0)
        self.usage_summary.llm_latency_ms += int(usage.get("latency_ms") or 0)
        role = str(role or "").strip()
        if role:
            role_usage = self.usage_summary.llm_by_role.setdefault(
                role,
                {
                    "llm_calls": 0,
                    "input_tokens": 0,
                    "output_tokens": 0,
                    "total_tokens": 0,
                    "reported_token_calls": 0,
                    "missing_token_calls": 0,
                    "latency_ms": 0,
                },
            )
            role_usage["llm_calls"] += int(usage.get("llm_calls") or 0)
            role_usage["input_tokens"] += int(usage.get("input_tokens") or 0)
            role_usage["output_tokens"] += int(usage.get("output_tokens") or 0)
            role_usage["total_tokens"] += int(
                usage.get("total_tokens")
                or (
                    int(usage.get("input_tokens") or 0)
                    + int(usage.get("output_tokens") or 0)
                )
            )
            role_usage["reported_token_calls"] += int(usage.get("reported_token_calls") or 0)
            role_usage["missing_token_calls"] += int(usage.get("missing_token_calls") or 0)
            role_usage["latency_ms"] += int(usage.get("latency_ms") or 0)

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
        awaiting_action_ids = [
            action.action_id
            for action in self.actions
            if action.status == "awaiting_confirmation"
        ]
        pending_confirmation_ids = [
            action.confirm_id
            for action in self.actions
            if action.status == "awaiting_confirmation" and action.confirm_id
        ]
        return {
            "version": 5,
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
            "tool_records": [record.to_dict() for record in self.tool_records],
            "pending_confirmations": list(self.pending_confirmations),
            "current_action_ids": awaiting_action_ids,
            "pending_confirmation_ids": pending_confirmation_ids,
        }

    def apply_resume_outcome(self) -> dict[str, Any] | None:
        if not self.resume_context:
            return None

        outcomes = self.resume_context.get("confirmation_results")
        if not isinstance(outcomes, list) or not outcomes:
            outcomes = [self.resume_context]
        outcomes = self._order_resume_outcomes(outcomes)

        applied: list[dict[str, Any]] = []
        for outcome in outcomes:
            if not isinstance(outcome, dict):
                continue
            applied_action = self._apply_single_resume_outcome(outcome)
            if applied_action:
                applied.append(applied_action)
        return applied[-1] if applied else None

    def _order_resume_outcomes(self, outcomes: list[Any]) -> list[dict[str, Any]]:
        clean = [outcome for outcome in outcomes if isinstance(outcome, dict)]
        if len(clean) <= 1:
            return clean
        confirm_rank = {
            action.confirm_id: index
            for index, action in enumerate(self.actions)
            if action.confirm_id
        }
        action_rank = {
            action.action_id: index
            for index, action in enumerate(self.actions)
            if action.action_id
        }

        def rank(outcome: dict[str, Any]) -> int:
            confirm_id = str(outcome.get("confirm_id") or "").strip()
            if confirm_id and confirm_id in confirm_rank:
                return confirm_rank[confirm_id]
            action_id = str(outcome.get("action_id") or "").strip()
            if action_id and action_id in action_rank:
                return action_rank[action_id]
            tool_name = str(outcome.get("tool_name") or "").strip()
            tool_args = outcome.get("tool_args") if isinstance(outcome.get("tool_args"), dict) else {}
            for index, action in enumerate(self.actions):
                if action.tool_name == tool_name and action.tool_args == tool_args:
                    return index
            return 1 << 20

        return sorted(clean, key=rank)

    def _apply_single_resume_outcome(self, outcome: dict[str, Any]) -> dict[str, Any] | None:
        confirm_id = str(outcome.get("confirm_id") or "").strip()
        result_status = str(outcome.get("result_status") or "").strip() or "completed"
        result = outcome.get("result")
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
            self.pending_confirmations = [
                item for item in self.pending_confirmations
                if str((item.get("confirmation") or {}).get("confirm_id") or "").strip() != action.confirm_id
            ]
            if not any(
                str(item.get("confirm_id") or "").strip() == action.confirm_id and action.confirm_id
                for item in self.action_history
            ):
                self.record_action_result(
                    action=action,
                    result_status=action.status,
                    result=result,
                    confirmed=outcome.get("confirmed"),
                )
            return {
                "action_id": action.action_id,
                "tool_name": action.tool_name,
                "title": action.title,
                "status": action.status,
                "confirm_id": action.confirm_id,
                "result": result,
            }
        return None
