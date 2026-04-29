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


def _looks_like_simple_help_or_chat(user_message: str) -> bool:
    normalized = str(user_message or "").strip().lower()
    if not normalized:
        return True
    compact = re.sub(r"[\s，。！？!?,.]+", "", normalized)
    if compact in {"hi", "hello", "hey", "你好", "您好", "在吗", "谢谢", "thanks", "thx"}:
        return True
    help_tokens = (
        "怎么用",
        "怎么创建",
        "怎么提交",
        "怎么停止",
        "怎么删除",
        "怎么重提",
        "如何使用",
        "如何创建",
        "如何提交",
        "如何停止",
        "如何删除",
        "如何重提",
        "在哪",
        "哪里",
        "支持什么",
        "能做什么",
        "有什么功能",
        "区别是什么",
        "介绍一下",
        "说明一下",
        "帮助",
        "文档",
        "what can you do",
        "how to",
    )
    agent_required_tokens = (
        "为什么",
        "失败",
        "报错",
        "异常",
        "卡住",
        "卡在哪",
        "rollout",
        "正常吗",
        "需要我处理",
        "能不能提交",
        "帮我看看",
        "帮我看下",
        "有没有",
        "推荐",
        "配置",
        "a100",
        "v100",
        "deepspeed",
        "llama",
        "配额",
        "镜像",
        "提交配置",
        "完整配置",
        "完整提交配置",
        "分布式训练",
        "torchrun",
        "资源",
    )
    if any(token in normalized for token in agent_required_tokens):
        return False
    if any(token in normalized for token in help_tokens):
        data_tokens = (
            "当前",
            "现在",
            "这次",
            "这个作业",
            "我的作业",
            "节点状态",
            "集群状态",
            "日志",
            "报错",
            "失败",
            "为什么",
        )
        execution_phrases = (
            "帮我停止",
            "帮我删除",
            "帮我重提",
            "帮我重启",
            "帮我创建",
            "帮我提交",
            "请停止",
            "请删除",
            "请重提",
            "请重启",
            "请创建",
            "请提交",
        )
        return not any(token in normalized for token in data_tokens + execution_phrases)
    return False


def _looks_like_scale_write_intent(user_message: str) -> bool:
    normalized = str(user_message or "").strip().lower()
    if not normalized:
        return False
    scale_tokens = ("缩容", "扩容", "scale", "replica", "replicas", "副本")
    write_tokens = ("帮我", "请", "把", "调整", "调到", "改成", "改为", "设为", "缩到", "扩到")
    return any(token in normalized for token in scale_tokens) and any(
        token in normalized for token in write_tokens
    )


def _looks_like_admin_write_intent(user_message: str) -> bool:
    normalized = str(user_message or "").strip().lower()
    if not normalized:
        return False

    write_verbs = (
        "重启",
        "restart",
        "reboot",
        "删除",
        "delete",
        "停止",
        "stop",
        "终止",
        "kill",
        "扩容",
        "缩容",
        "scale",
        "cordon",
        "uncordon",
        "drain",
        "隔离",
        "驱逐",
        "下线",
        "恢复调度",
    )
    write_context = (
        "帮我",
        "请",
        "直接",
        "把",
        "一下",
        "执行",
        "处理",
        "操作",
    )
    return any(token in normalized for token in write_verbs) and any(
        token in normalized for token in write_context
    )


def _has_task_bound_page_context(page_context: dict[str, Any]) -> bool:
    if not isinstance(page_context, dict):
        return False
    for key in ("job_name", "jobName", "node_name", "nodeName"):
        if str(page_context.get(key) or "").strip():
            return True
    page_markers = (
        str(page_context.get("path") or "").strip().lower(),
        str(page_context.get("pathname") or "").strip().lower(),
        str(page_context.get("route") or "").strip().lower(),
        str(page_context.get("entryPoint") or page_context.get("entrypoint") or "").strip().lower(),
    )
    task_paths = (
        "/jobs",
        "/admin",
        "/nodes",
        "/monitor",
        "/inspection",
        "job_detail",
        "job_create",
        "admin",
        "ops",
    )
    return any(marker and any(token in marker for token in task_paths) for marker in page_markers)


def is_strict_toolless_fast_path_candidate(
    *,
    user_message: str,
    page_context: dict[str, Any],
    routing: RoutingDecision,
) -> bool:
    normalized = str(user_message or "").strip().lower()
    compact = re.sub(r"[\s，。！？!?,.]+", "", normalized)
    if not normalized:
        return False
    if routing.requested_action or routing.operation_mode in {"read", "write"}:
        return False
    if routing.targets.job_name or routing.targets.node_name or routing.targets.scope != "unspecified":
        return False
    if _has_task_bound_page_context(page_context):
        return False

    social_exact = {
        "hi",
        "hello",
        "hey",
        "你好",
        "您好",
        "在吗",
        "谢谢",
        "thanks",
        "thx",
    }
    if routing.entry_mode == "simple":
        return compact in social_exact

    if routing.entry_mode != "help":
        return False

    generic_help_tokens = (
        "你能做什么",
        "你会什么",
        "有什么功能",
        "支持什么能力",
        "支持哪些能力",
        "怎么用这个助手",
        "如何使用这个助手",
        "这个助手怎么用",
        "帮助文档",
        "使用说明",
        "功能介绍",
        "介绍一下你",
        "介绍下你",
        "文档入口",
    )
    task_tokens = (
        "作业",
        "job",
        "节点",
        "node",
        "rollout",
        "prometheus",
        "gpu",
        "a100",
        "v100",
        "镜像",
        "配额",
        "日志",
        "指标",
        "监控",
        "k8s",
        "提交",
        "创建",
        "重提",
        "删除",
        "停止",
        "重启",
        "扩容",
        "缩容",
        "训练",
        "jupyter",
        "webide",
        "资源",
        "配置",
        "模型",
        "llama",
        "deepspeed",
        "pvc",
        "rdma",
        "容量",
        "异常",
        "失败",
        "正常吗",
        "帮我",
    )
    return any(token in normalized for token in generic_help_tokens) and not any(
        token in normalized for token in task_tokens
    )


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
    actor_role: str = "",
) -> RoutingDecision:
    """Extract deterministic routing hints without LLM. No keyword tables."""
    del actor_role
    targets = RoutingTargets()
    entry_mode = "agent"
    operation_mode = "unknown"
    requested_action: str | None = None
    confidence = 0.0
    complexity = "normal"

    if _looks_like_simple_help_or_chat(user_message):
        entry_mode = "help"
        complexity = "simple"
        confidence = max(confidence, 0.75)

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
        complexity = "simple"
        confidence = max(confidence, 0.8)

    # 5. K8s workload scaling is a write operation even if the target is
    # supplied by page context rather than a jobName-style identifier.
    if _looks_like_scale_write_intent(user_message):
        entry_mode = "agent"
        operation_mode = "write"
        requested_action = "scale"
        complexity = "complex"
        confidence = max(confidence, 0.78)
    elif _looks_like_admin_write_intent(user_message):
        entry_mode = "agent"
        operation_mode = "write"
        complexity = "complex"
        confidence = max(confidence, 0.78)
        if any(token in str(user_message or "").strip().lower() for token in ("隔离", "isolate", "taint")):
            requested_action = "node_isolation"

    return RoutingDecision(
        entry_mode=entry_mode,
        operation_mode=operation_mode,
        targets=targets,
        requested_action=requested_action,
        confidence=confidence,
        complexity=complexity,
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
        )

        # Only stateful follow-ups with explicit carry-over/selection bypass LLM.
        # All regular user utterances should still be classified by the LLM with
        # history + page context, so the orchestrator remains model-driven.
        if self._should_use_hints_directly(
            hints=hints,
            continuation=continuation,
            resume_context=resume_context,
            clarification_context=dict(clarification_context or {}),
        ):
            logger.info(
                "IntentRouter: deterministic routing (confidence=%.2f, mode=%s, action=%s)",
                hints.confidence,
                hints.operation_mode,
                hints.requested_action,
            )
            return hints

        # LLM classification for ambiguous requests
        try:
            classified = await self._classify_with_llm(
                user_message=user_message,
                page_context=page_context,
                actor_role=actor_role,
                history_context=history_context,
                deterministic_hints=hints,
            )
            if classified.entry_mode in {"simple", "help"} and not is_strict_toolless_fast_path_candidate(
                user_message=user_message,
                page_context=page_context,
                routing=classified,
            ):
                classified.entry_mode = "agent"
                if classified.complexity == "simple":
                    classified.complexity = "normal"
                if classified.confidence < 0.6:
                    classified.confidence = 0.6
            return classified
        except Exception:
            logger.exception("IntentRouter LLM classification failed, using hints")
            # Fallback: ambiguous → agent mode
            hints.entry_mode = "agent"
            hints.confidence = 0.3
            return hints

    @staticmethod
    def _should_use_hints_directly(
        *,
        hints: RoutingDecision,
        continuation: dict[str, Any],
        resume_context: dict[str, Any],
        clarification_context: dict[str, Any],
    ) -> bool:
        if resume_context and hints.requested_action:
            return True

        if clarification_context and (
            hints.targets.job_name
            or hints.targets.scope == "all"
        ):
            return True

        workflow = continuation.get("workflow") or {}
        if (
            isinstance(workflow, dict)
            and hints.requested_action
            and (hints.targets.job_name or hints.targets.node_name)
        ):
            return True

        return False

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
        if deterministic_hints.confidence > 0:
            hints_context += (
                f"\n上下文提示 confidence={deterministic_hints.confidence:.2f}, "
                f"complexity={deterministic_hints.complexity}"
            )
        if deterministic_hints.targets.job_name:
            hints_context += f"\n已知目标作业: {deterministic_hints.targets.job_name}"
        if deterministic_hints.targets.node_name:
            hints_context += f"\n已知目标节点: {deterministic_hints.targets.node_name}"
        if deterministic_hints.requested_action:
            hints_context += f"\n已知动作候选: {deterministic_hints.requested_action}"
        if deterministic_hints.targets.scope != "unspecified":
            hints_context += f"\n已知作用范围: {deterministic_hints.targets.scope}"
        if deterministic_hints.entry_mode != "agent":
            hints_context += f"\n上下文提示 entry_mode 倾向: {deterministic_hints.entry_mode}"
        if deterministic_hints.operation_mode != "unknown":
            hints_context += f"\n上下文提示 operation_mode 倾向: {deterministic_hints.operation_mode}"

        result = await self.coordinator_agent.run_json(
            system_prompt=(
                "你是意图路由器。分析用户请求，判断：\n"
                "1. entry_mode: 'simple'（问候/闲聊/无需工具的极简单答复）、'help'（纯帮助/文档/概念解释）或 'agent'（需要工具/数据/操作）\n"
                "2. operation_mode: 'read'（查询/诊断/查看）, 'write'（创建/停止/删除/重提交）, 'unknown'\n"
                "3. requested_action: 具体操作名（resubmit/stop/delete/create/scale），无则 null\n"
                "4. complexity: 'simple'（不需要 MAS 循环）、'normal'、'complex'（多轮/多工具/故障诊断/写操作）\n"
                "5. confidence: 0.0-1.0\n\n"
                "关键原则：\n"
                "- '集群资源如何'、'当前作业情况'、'节点状态' → agent + read（需要工具查数据）\n"
                "- '怎么创建作业'、'在哪看日志' → help（纯文档指引）\n"
                "- '重提/停止/删除/扩容/缩容 xxx' → agent + write + 对应 action\n"
                "- 'rollout 卡在哪'、'有没有带 DeepSpeed 的镜像'、'LLaMA-7B 应该用什么配置'、'这个配置能不能提交'、'现在正常吗' 这类请求都不是 help，而是 agent；因为它们依赖平台实时工具结果\n"
                "- '我需要 8 张 A100 做分布式训练，要怎么配置？最好给我一个完整的提交配置' 也不是 help，而是 agent；这类问题需要结合实时配额、模板、镜像或容量信息，不能只做页面导航\n"
                "- 纯问候、感谢、无平台数据依赖的极简单问题 → simple + unknown + simple，直接交 General，不进入 MAS 循环\n"
                "- 帮助/文档/概念解释 → help + unknown + simple，交 Guide/General，不进入 MAS 循环\n"
                "- 需要实时平台数据、工具结果、诊断、写操作或多步推理 → agent\n"
                "- 只要用户问题是在问某个对象“现在怎样/是否存在/该怎么配/能不能做”，且平台工具能直接核实，就必须判为 agent，不要输出页面导航式帮助\n"
                "- 如果页面上下文已在新建作业/提交配置页，而用户在问推荐配置、镜像是否存在、某种 GPU 怎么配、能不能提交、给我完整配置，默认判为 agent，不要判成 help\n"
                "- 如果当前输入是在追问、纠正、补充或质疑上一轮回答，要结合近期对话理解，但不要继承未经工具证实的旧结论\n"
                "- 普通问答、概念解释、怎么做、去哪做，优先根据历史和页面上下文理解，不要被孤立关键词误导\n"
                "- deterministic hints 只是上下文提示，不是必须采纳的最终结论；若与当前 message 和 history 不一致，以整体语义为准\n"
                "- 模糊且涉及业务数据/平台状态/实际对象的请求，默认 agent + unknown\n\n"
                "输出 JSON:\n"
                '{"entry_mode": "simple|help|agent", "operation_mode": "read|write|unknown", '
                '"requested_action": "resubmit|stop|delete|create|scale|node_isolation|null", '
                '"complexity": "simple|normal|complex", '
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
        if entry_mode not in {"simple", "help", "agent"}:
            entry_mode = "agent"

        operation_mode = str(result.get("operation_mode") or "unknown").strip().lower()
        if operation_mode not in {"read", "write", "unknown"}:
            operation_mode = "unknown"

        requested_action = str(result.get("requested_action") or "").strip().lower() or None
        if requested_action in {"null", "none", ""}:
            requested_action = None
        if requested_action and requested_action not in {
            "resubmit",
            "stop",
            "delete",
            "create",
            "scale",
            "node_isolation",
        }:
            requested_action = None

        complexity = str(result.get("complexity") or fallback.complexity or "normal").strip().lower()
        if complexity not in {"simple", "normal", "complex"}:
            complexity = "normal"

        confidence = 0.5
        try:
            confidence = float(result.get("confidence") or 0.5)
        except (TypeError, ValueError):
            pass

        # Merge with deterministic hints (deterministic targets take precedence)
        targets = fallback.targets
        preserve_hint_bias = fallback.confidence >= 0.75
        if preserve_hint_bias and fallback.entry_mode == "simple" and not requested_action:
            entry_mode = fallback.entry_mode
            operation_mode = fallback.operation_mode
            complexity = fallback.complexity
        if requested_action:
            operation_mode = "write"
            entry_mode = "agent"
            complexity = "complex"
        elif entry_mode in {"simple", "help"} and operation_mode in {"read", "write"}:
            entry_mode = "agent"
            if complexity == "simple":
                complexity = "normal"
        if operation_mode == "write" and not requested_action:
            requested_action = fallback.requested_action

        return RoutingDecision(
            entry_mode=entry_mode,
            operation_mode=operation_mode,
            targets=targets,
            requested_action=requested_action or fallback.requested_action,
            confidence=confidence,
            rationale=str(result.get("rationale") or "").strip(),
            complexity=complexity,
        )
