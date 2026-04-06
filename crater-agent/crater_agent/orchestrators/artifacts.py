"""Typed artifacts for multi-agent stage communication."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Literal


@dataclass
class RoutingTargets:
    job_name: str | None = None
    node_name: str | None = None
    scope: str = "unspecified"  # "single" | "all" | "unspecified"


@dataclass
class RoutingDecision:
    entry_mode: str = "agent"  # "help" | "agent"
    operation_mode: str = "unknown"  # "read" | "write" | "unknown"
    targets: RoutingTargets = field(default_factory=RoutingTargets)
    requested_action: str | None = None  # "resubmit" | "stop" | "delete" | "create" etc.
    confidence: float = 0.5
    rationale: str = ""

    def to_dict(self) -> dict[str, Any]:
        return {
            "entry_mode": self.entry_mode,
            "operation_mode": self.operation_mode,
            "targets": {
                "job_name": self.targets.job_name,
                "node_name": self.targets.node_name,
                "scope": self.targets.scope,
            },
            "requested_action": self.requested_action,
            "confidence": self.confidence,
            "rationale": self.rationale,
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any] | None) -> RoutingDecision:
        if not isinstance(data, dict):
            return cls()
        targets_data = data.get("targets") or {}
        if not isinstance(targets_data, dict):
            targets_data = {}
        return cls(
            entry_mode=str(data.get("entry_mode") or "agent").strip(),
            operation_mode=str(data.get("operation_mode") or "unknown").strip(),
            targets=RoutingTargets(
                job_name=str(targets_data.get("job_name") or "").strip() or None,
                node_name=str(targets_data.get("node_name") or "").strip() or None,
                scope=str(targets_data.get("scope") or "unspecified").strip(),
            ),
            requested_action=str(data.get("requested_action") or "").strip() or None,
            confidence=float(data.get("confidence") or 0.5),
            rationale=str(data.get("rationale") or "").strip(),
        )


@dataclass
class GoalArtifact:
    user_message: str = ""
    original_user_message: str = ""
    actor_role: str = "user"
    page_context: dict[str, Any] = field(default_factory=dict)
    routing: RoutingDecision = field(default_factory=RoutingDecision)

    def to_dict(self) -> dict[str, Any]:
        return {
            "user_message": self.user_message,
            "original_user_message": self.original_user_message,
            "actor_role": self.actor_role,
            "page_context": self.page_context,
            "routing": self.routing.to_dict(),
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any] | None) -> GoalArtifact:
        if not isinstance(data, dict):
            return cls()
        return cls(
            user_message=str(data.get("user_message") or "").strip(),
            original_user_message=str(data.get("original_user_message") or "").strip(),
            actor_role=str(data.get("actor_role") or "user").strip(),
            page_context=dict(data.get("page_context") or {}),
            routing=RoutingDecision.from_dict(data.get("routing")),
        )


@dataclass
class ObservationArtifact:
    summary: str = ""
    facts: list[str] = field(default_factory=list)
    open_questions: list[str] = field(default_factory=list)
    evidence: list[dict[str, Any]] = field(default_factory=list)
    stage_complete: bool = False

    def to_dict(self) -> dict[str, Any]:
        return {
            "summary": self.summary,
            "facts": list(self.facts),
            "open_questions": list(self.open_questions),
            "evidence": list(self.evidence),
            "stage_complete": self.stage_complete,
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any] | None) -> ObservationArtifact | None:
        if not isinstance(data, dict):
            return None
        return cls(
            summary=str(data.get("summary") or "").strip(),
            facts=[str(f) for f in (data.get("facts") or []) if f],
            open_questions=[str(q) for q in (data.get("open_questions") or []) if q],
            evidence=list(data.get("evidence") or []),
            stage_complete=bool(data.get("stage_complete")),
        )


@dataclass
class PlanArtifact:
    summary: str = ""
    steps: list[str] = field(default_factory=list)
    candidate_tools: list[str] = field(default_factory=list)
    risk: str = "low"

    def to_dict(self) -> dict[str, Any]:
        return {
            "summary": self.summary,
            "steps": list(self.steps),
            "candidate_tools": list(self.candidate_tools),
            "risk": self.risk,
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any] | None) -> PlanArtifact | None:
        if not isinstance(data, dict):
            return None
        return cls(
            summary=str(data.get("summary") or "").strip(),
            steps=[str(s) for s in (data.get("steps") or []) if s],
            candidate_tools=[str(t) for t in (data.get("candidate_tools") or []) if t],
            risk=str(data.get("risk") or "low").strip(),
        )


@dataclass
class ExecutionArtifact:
    summary: str = ""
    actions: list[dict[str, Any]] = field(default_factory=list)
    awaiting_confirmation: bool = False

    def to_dict(self) -> dict[str, Any]:
        return {
            "summary": self.summary,
            "actions": list(self.actions),
            "awaiting_confirmation": self.awaiting_confirmation,
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any] | None) -> ExecutionArtifact | None:
        if not isinstance(data, dict):
            return None
        return cls(
            summary=str(data.get("summary") or "").strip(),
            actions=list(data.get("actions") or []),
            awaiting_confirmation=bool(data.get("awaiting_confirmation")),
        )


@dataclass
class StateView:
    """Projected state view for subagents. Each role sees only what it needs."""
    goal: GoalArtifact = field(default_factory=GoalArtifact)
    observation: ObservationArtifact | None = None
    plan: PlanArtifact | None = None
    execution: ExecutionArtifact | None = None
    compact_evidence: list[dict[str, Any]] = field(default_factory=list)
    loop_round: int = 0
    max_rounds: int = 6

    def for_prompt(self) -> str:
        """Serialize to a compact text block for inclusion in subagent prompts."""
        parts: list[str] = []
        parts.append(f"用户请求: {self.goal.user_message}")
        if self.goal.actor_role != "user":
            parts.append(f"用户角色: {self.goal.actor_role}")
        if self.goal.page_context:
            parts.append(f"页面上下文: {self.goal.page_context}")
        if self.goal.routing.requested_action:
            parts.append(f"请求动作: {self.goal.routing.requested_action}")
        if self.goal.routing.targets.job_name:
            parts.append(f"目标作业: {self.goal.routing.targets.job_name}")
        if self.goal.routing.targets.node_name:
            parts.append(f"目标节点: {self.goal.routing.targets.node_name}")
        if self.observation:
            parts.append(f"\n已有观测:\n{self.observation.summary}")
            if self.observation.facts:
                parts.append("已确认事实:\n" + "\n".join(f"- {f}" for f in self.observation.facts))
            if self.observation.open_questions:
                parts.append("待解决问题:\n" + "\n".join(f"- {q}" for q in self.observation.open_questions))
        if self.plan:
            parts.append(f"\n当前计划:\n{self.plan.summary}")
            if self.plan.steps:
                parts.append("计划步骤:\n" + "\n".join(f"{i+1}. {s}" for i, s in enumerate(self.plan.steps)))
            if self.plan.candidate_tools:
                parts.append(f"推荐工具: {', '.join(self.plan.candidate_tools)}")
        if self.execution:
            parts.append(f"\n执行结果:\n{self.execution.summary}")
        if self.compact_evidence:
            import json
            parts.append(f"\n工具调用原始结果:\n{json.dumps(self.compact_evidence, ensure_ascii=False, indent=None)}")
        parts.append(f"\n当前轮次: {self.loop_round}/{self.max_rounds}")
        return "\n".join(parts)
