"""Intent router for MAS v2. Replaces keyword-based coordinator routing."""

from __future__ import annotations

import logging
import re
from typing import Any

from crater_agent.agents.base import BaseRoleAgent
from crater_agent.orchestrators.artifacts import RoutingDecision, RoutingTargets

logger = logging.getLogger(__name__)


def _looks_like_system_job_name(value: str) -> bool:
    """Check if a string looks like a system-generated job name (e.g. sg-xxx-123)."""
    normalized = value.strip().lower()
    if not normalized:
        return False
    if normalized.count("-") < 2:
        return False
    return any(ch.isdigit() for ch in normalized)


def _extract_explicit_job_name(user_message: str) -> str | None:
    for candidate in re.findall(r"[A-Za-z0-9]+(?:-[A-Za-z0-9]+){2,}", str(user_message or "")):
        normalized = candidate.strip().lower()
        if _looks_like_system_job_name(normalized):
            return normalized
    return None


def _message_requests_all(user_message: str) -> bool:
    normalized = str(user_message or "").strip().lower()
    if not normalized:
        return False
    if normalized in {"all", "all of them", "全部", "所有", "全都", "都处理"}:
        return True
    return any(token in normalized for token in ("全部", "所有", "all of them", "全都", "都处理"))


def _looks_like_continuation_reply(user_message: str) -> bool:
    normalized = str(user_message or "").strip().lower()
    if not normalized:
        return False
    if _extract_explicit_job_name(normalized) or _message_requests_all(normalized):
        return True
    if normalized in {"确认", "继续", "这个", "就这个", "first", "second", "1", "2", "3"}:
        return True
    return any(token in normalized for token in ("第一个", "第1个", "第二个", "第2个", "改成", "名字叫"))


def _resolve_job_selection_reply(
    *,
    user_message: str,
    clarification_context: dict[str, Any],
) -> tuple[str | None, str | None]:
    if not isinstance(clarification_context, dict):
        return None, None
    if _message_requests_all(user_message):
        return None, "all"

    explicit_job_name = _extract_explicit_job_name(user_message)
    if explicit_job_name:
        return explicit_job_name, "single"

    candidates = clarification_context.get("candidate_jobs") or []
    if not isinstance(candidates, list) or not candidates:
        return None, None

    normalized = str(user_message or "").strip().lower()
    if normalized.isdigit():
        index = int(normalized)
        if 1 <= index <= len(candidates):
            job_name = str((candidates[index - 1] or {}).get("job_name") or "").strip().lower()
            if job_name:
                return job_name, "single"

    ordinal_aliases = {
        "第一个": 0,
        "第1个": 0,
        "first": 0,
        "第二个": 1,
        "第2个": 1,
        "second": 1,
    }
    index = ordinal_aliases.get(normalized)
    if index is not None and index < len(candidates):
        job_name = str((candidates[index] or {}).get("job_name") or "").strip().lower()
        if job_name:
            return job_name, "single"

    for candidate in candidates:
        if not isinstance(candidate, dict):
            continue
        job_name = str(candidate.get("job_name") or "").strip().lower()
        display_name = str(candidate.get("display_name") or "").strip().lower()
        if normalized and normalized in {job_name, display_name}:
            return job_name, "single"
    return None, None


def _extract_deterministic_hints(
    *,
    user_message: str,
    page_context: dict[str, Any],
    continuation: dict[str, Any],
    resume_context: dict[str, Any],
    clarification_context: dict[str, Any],
    actor_role: str,
) -> RoutingDecision:
    """Extract deterministic routing hints without LLM. No keyword tables."""
    targets = RoutingTargets()
    entry_mode = "agent"
    operation_mode = "unknown"
    requested_action: str | None = None
    confidence = 0.0

    # 1. Resume after confirmation → continue previous operation
    if resume_context:
        action_intent = str(resume_context.get("action_intent") or "").strip().lower()
        job_name = str(resume_context.get("job_name") or "").strip().lower()
        if action_intent:
            requested_action = action_intent
            operation_mode = "write"
            confidence = 0.9
        if job_name and _looks_like_system_job_name(job_name):
            targets.job_name = job_name

    # 1b. Clarification follow-up (e.g. job_selection from previous turn)
    if clarification_context and _looks_like_continuation_reply(user_message):
        clar_kind = str(clarification_context.get("kind") or "").strip().lower()
        if clar_kind == "job_selection":
            clar_action = str(clarification_context.get("action_intent") or "").strip().lower()
            if clar_action:
                requested_action = clar_action
                operation_mode = "write"
                confidence = max(confidence, 0.85)
            selected_job_name, requested_scope = _resolve_job_selection_reply(
                user_message=user_message,
                clarification_context=clarification_context,
            )
            if requested_scope == "all":
                targets.scope = "all"
                confidence = max(confidence, 0.95)
            if selected_job_name:
                targets.job_name = selected_job_name
                targets.scope = "single"
                confidence = max(confidence, 0.95)

    # 2. Page context → bind targets
    if not targets.job_name:
        for key in ("job_name", "jobName"):
            val = str(page_context.get(key) or "").strip().lower()
            if val and _looks_like_system_job_name(val):
                targets.job_name = val
                break

    if not targets.node_name:
        for key in ("node_name", "nodeName"):
            val = str(page_context.get(key) or "").strip()
            if val:
                targets.node_name = val
                break

    # 3. Continuation context (workflow carry-over)
    workflow = continuation.get("workflow") or {}
    if isinstance(workflow, dict) and _looks_like_continuation_reply(user_message):
        wf_routing = workflow.get("routing") or {}
        if isinstance(wf_routing, dict):
            wf_action = str(wf_routing.get("requested_action") or "").strip().lower()
            if wf_action and not requested_action:
                requested_action = wf_action
                operation_mode = "write"
                confidence = max(confidence, 0.7)
            wf_targets = wf_routing.get("targets") or {}
            if isinstance(wf_targets, dict):
                if not targets.job_name:
                    wf_job = str(wf_targets.get("job_name") or "").strip().lower()
                    if wf_job and _looks_like_system_job_name(wf_job):
                        targets.job_name = wf_job
                if not targets.node_name:
                    wf_node = str(wf_targets.get("node_name") or "").strip()
                    if wf_node:
                        targets.node_name = wf_node

    # 4. Explicit entryPoint hint
    entrypoint = str(page_context.get("entryPoint") or page_context.get("entrypoint") or "").strip().lower()
    if entrypoint == "guide":
        entry_mode = "help"
        confidence = max(confidence, 0.8)

    return RoutingDecision(
        entry_mode=entry_mode,
        operation_mode=operation_mode,
        targets=targets,
        requested_action=requested_action,
        confidence=confidence,
    )


class IntentRouter:
    """Hybrid intent router: deterministic hints + optional LLM classification."""

    def __init__(self, *, coordinator_agent: BaseRoleAgent):
        self.coordinator_agent = coordinator_agent

    async def route(
        self,
        *,
        user_message: str,
        page_context: dict[str, Any],
        continuation: dict[str, Any],
        resume_context: dict[str, Any],
        actor_role: str,
        history_context: str = "",
        clarification_context: dict[str, Any] | None = None,
    ) -> RoutingDecision:
        """Route a request. Uses deterministic hints first, LLM classification as fallback."""
        hints = _extract_deterministic_hints(
            user_message=user_message,
            page_context=page_context,
            continuation=continuation,
            resume_context=resume_context,
            clarification_context=dict(clarification_context or {}),
            actor_role=actor_role,
        )

        # If deterministic hints are confident enough, skip LLM
        if hints.confidence >= 0.7:
            logger.info(
                "IntentRouter: deterministic routing (confidence=%.2f, mode=%s, action=%s)",
                hints.confidence,
                hints.operation_mode,
                hints.requested_action,
            )
            return hints

        # LLM classification for ambiguous requests
        try:
            return await self._classify_with_llm(
                user_message=user_message,
                page_context=page_context,
                actor_role=actor_role,
                history_context=history_context,
                deterministic_hints=hints,
            )
        except Exception:
            logger.exception("IntentRouter LLM classification failed, using hints")
            # Fallback: ambiguous → agent mode
            hints.entry_mode = "agent"
            hints.confidence = 0.3
            return hints

    async def _classify_with_llm(
        self,
        *,
        user_message: str,
        page_context: dict[str, Any],
        actor_role: str,
        history_context: str,
        deterministic_hints: RoutingDecision,
    ) -> RoutingDecision:
        """Use LLM to classify intent when deterministic hints are insufficient."""
        hints_context = ""
        if deterministic_hints.targets.job_name:
            hints_context += f"\n已知目标作业: {deterministic_hints.targets.job_name}"
        if deterministic_hints.targets.node_name:
            hints_context += f"\n已知目标节点: {deterministic_hints.targets.node_name}"

        result = await self.coordinator_agent.run_json(
            system_prompt=(
                "你是意图路由器。分析用户请求，判断：\n"
                "1. entry_mode: 'help'（纯帮助/文档/概念解释）或 'agent'（需要工具/数据/操作）\n"
                "2. operation_mode: 'read'（查询/诊断/查看）, 'write'（创建/停止/删除/重提交）, 'unknown'\n"
                "3. requested_action: 具体操作名（resubmit/stop/delete/create），无则 null\n"
                "4. confidence: 0.0-1.0\n\n"
                "关键原则：\n"
                "- '集群资源如何'、'当前作业情况'、'节点状态' → agent + read（需要工具查数据）\n"
                "- '怎么创建作业'、'在哪看日志' → help（纯文档指引）\n"
                "- '重提/停止/删除 xxx' → agent + write + 对应 action\n"
                "- 如果当前输入是在追问或质疑上一轮回答，要结合近期对话理解，但不要继承未经工具证实的旧结论\n"
                "- 模糊请求默认 agent + unknown\n\n"
                "输出 JSON:\n"
                '{"entry_mode": "help|agent", "operation_mode": "read|write|unknown", '
                '"requested_action": "resubmit|stop|delete|create|null", '
                '"confidence": 0.8, "rationale": "简短理由"}\n'
            ),
            user_prompt=(
                f"用户请求: {user_message}\n"
                f"用户角色: {actor_role}\n"
                f"页面上下文: {page_context}\n"
                f"{hints_context}\n"
                f"近期对话上下文: {history_context or '(empty)'}\n\n"
                "请输出 JSON。"
            ),
        )

        return self._parse_llm_result(result, deterministic_hints)

    @staticmethod
    def _parse_llm_result(
        result: dict[str, Any] | list[Any],
        fallback: RoutingDecision,
    ) -> RoutingDecision:
        """Parse LLM classification result."""
        if not isinstance(result, dict) or ("raw" in result and len(result) == 1):
            logger.warning("IntentRouter LLM result invalid, using fallback")
            fallback.entry_mode = "agent"
            fallback.confidence = 0.3
            return fallback

        entry_mode = str(result.get("entry_mode") or "agent").strip().lower()
        if entry_mode not in {"help", "agent"}:
            entry_mode = "agent"

        operation_mode = str(result.get("operation_mode") or "unknown").strip().lower()
        if operation_mode not in {"read", "write", "unknown"}:
            operation_mode = "unknown"

        requested_action = str(result.get("requested_action") or "").strip().lower() or None
        if requested_action in {"null", "none", ""}:
            requested_action = None
        if requested_action and requested_action not in {"resubmit", "stop", "delete", "create"}:
            requested_action = None

        confidence = 0.5
        try:
            confidence = float(result.get("confidence") or 0.5)
        except (TypeError, ValueError):
            pass

        # Merge with deterministic hints (deterministic targets take precedence)
        targets = fallback.targets
        if operation_mode == "write" and not requested_action:
            requested_action = fallback.requested_action

        return RoutingDecision(
            entry_mode=entry_mode,
            operation_mode=operation_mode,
            targets=targets,
            requested_action=requested_action or fallback.requested_action,
            confidence=confidence,
            rationale=str(result.get("rationale") or "").strip(),
        )
