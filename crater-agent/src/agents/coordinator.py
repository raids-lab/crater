"""Coordinator agent for multi-agent orchestration."""

from __future__ import annotations

import logging
from dataclasses import dataclass
from typing import Any

from langchain_core.messages import AIMessage, HumanMessage

from agents.base import BaseRoleAgent, RoleExecutionResult

logger = logging.getLogger(__name__)


def _looks_like_system_job_name(value: str) -> bool:
    normalized = value.strip().lower()
    if not normalized:
        return False
    if normalized.count("-") < 2:
        return False
    return any(ch.isdigit() for ch in normalized)


_EMPTY_SUMMARY_MARKERS = {
    "",
    "(empty)",
    "暂无执行动作",
    "无需执行写操作，继续进入验证。",
    "已完成最终总结。",
}


def _is_effectively_empty_summary(value: str) -> bool:
    normalized = str(value or "").strip()
    if not normalized:
        return True
    return normalized in _EMPTY_SUMMARY_MARKERS


def _looks_complete_user_summary(value: str) -> bool:
    normalized = str(value or "").strip()
    if not normalized:
        return False
    if normalized.startswith(("## 探索总结", "**探索总结**", "探索总结")):
        return False
    has_conclusion = any(token in normalized for token in ("结论", "运行正常", "无需额外处理", "建议下一步"))
    has_guidance = any(token in normalized for token in ("建议下一步", "建议", "后续观察", "继续观察"))
    return has_conclusion and has_guidance


def _prepare_explorer_summary_for_user(value: str) -> str:
    lines = [line for line in str(value or "").splitlines()]
    while lines and lines[0].strip() in {"## 探索总结", "**探索总结**", "探索总结"}:
        lines.pop(0)
    return "\n".join(lines).strip()


def _filter_summary_history_messages(history_messages: list | None) -> list:
    if not history_messages:
        return []
    filtered: list = []
    assistant_summaries_kept = 0
    for message in history_messages:
        content = str(getattr(message, "content", "") or "").strip()
        if not content:
            continue
        if isinstance(message, HumanMessage):
            filtered.append(message)
            continue
        if isinstance(message, AIMessage) and content.startswith("【历史工具结果"):
            filtered.append(message)
            continue
        if isinstance(message, AIMessage) and assistant_summaries_kept < 2:
            filtered.append(
                AIMessage(
                    content=(
                        "【历史助手结论：仅作已确认事实和追问上下文，若无新增反证不得推翻】"
                        f"{content[:1200]}"
                    )
                )
            )
            assistant_summaries_kept += 1
    return filtered


@dataclass
class TurnContextDecision:
    route: str = "diagnostic"
    action_intent: str | None = None
    selected_job_name: str | None = None
    requested_scope: str = "unspecified"
    rationale: str = ""


@dataclass
class LoopDecision:
    step: str = "finalize"
    rationale: str = ""


class CoordinatorAgent(BaseRoleAgent):
    async def decide_turn_context(
        self,
        *,
        user_message: str,
        page_context: dict[str, Any],
        continuation: dict[str, Any] | None = None,
        recent_history_excerpt: str = "",
        capabilities: dict[str, Any] | None = None,
    ) -> TurnContextDecision:
        capability_summary = self.summarize_capabilities(
            capabilities,
            max_tools=6,
            include_descriptions=False,
            include_role_policies=False,
        )

        result = await self.run_json(
            system_prompt=(
                "你是 Crater 的 Coordinator Agent。你负责解释“当前这一轮”用户输入，"
                "然后决定 multi-agent 的路由与上下文绑定。\n\n"
                "你会拿到四类信息：\n"
                "1. 当前用户输入：这是本轮真正要处理的内容，优先级最高。\n"
                "2. 页面上下文：帮助判断用户当前关注的页面或作业。\n"
                "3. continuation：由后端提供的结构化续接上下文，表示上一轮是否还在等待用户补充信息或确认。\n"
                "4. recent_history_excerpt：最近对话摘录，只能辅助理解省略表达，不能盖过当前输入。\n\n"
                "工作原则：\n"
                "- 必须以当前用户输入为主；不要因为历史里提到某个动作，就忽略当前输入的新意图。\n"
                "- 如果 continuation.clarification 表示上一轮在等待用户从候选作业中选择，"
                "而当前输入只是“第一个/这个/全部/某个 jobName”，你应结合 continuation 解析。\n"
                "- 如果当前输入已经改变主题，例如从“重提失败作业”切换到“为什么失败/怎么看日志/怎么创建作业”，"
                "就不要机械沿用旧动作。\n"
                "- guide: 帮助、文档、概念解释、使用说明。\n"
                "- general: 普通平台问答，不需要具体作业诊断或写操作。\n"
                "- diagnostic: 具体作业排障、作业列表查询、资源分析、以及任何写操作意图。\n"
                "- action_intent 仅在你确认当前输入仍是 stop/delete/resubmit 其中之一时填写，否则填 null。\n"
                "- selected_job_name 只在你能明确绑定到单个系统 jobName 时填写；无法确定就填 null。\n"
                "- requested_scope=all 只在当前输入明确表达“全部/所有/all/every one of them”等整体范围时填写；"
                "不要因为 failed jobs / 失败作业 这类泛指复数表达就自动推断为 all。\n"
                '- requested_scope 只能是 "single"、"all"、"unspecified"。\n\n'
                "输出 JSON：\n"
                "{\n"
                '  "route": "guide|general|diagnostic",\n'
                '  "action_intent": "resubmit|stop|delete|null",\n'
                '  "selected_job_name": "sg-xxx|jpt-xxx|null",\n'
                '  "requested_scope": "single|all|unspecified",\n'
                '  "rationale": "简短理由"\n'
                "}\n"
            ),
            user_prompt=(
                f"当前用户输入:\n{user_message}\n\n"
                f"页面上下文:\n{page_context}\n\n"
                f"continuation:\n{continuation or {}}\n\n"
                f"recent_history_excerpt:\n{recent_history_excerpt or '(empty)'}\n\n"
                f"能力摘要:\n{capability_summary}\n\n"
                "请输出结构化 JSON。"
            ),
        )

        return self._parse_turn_context(result)

    @staticmethod
    def _parse_turn_context(result: dict[str, Any] | list[Any]) -> TurnContextDecision:
        if not isinstance(result, dict) or ("raw" in result and len(result) == 1):
            logger.warning("Coordinator turn-context decision was invalid: %s", result)
            return TurnContextDecision()

        route = str(result.get("route") or "").strip().lower()
        if route not in {"guide", "general", "diagnostic"}:
            route = "diagnostic"

        action_intent = str(result.get("action_intent") or "").strip().lower() or None
        if action_intent in {"null", "none"}:
            action_intent = None
        if action_intent not in {None, "resubmit", "stop", "delete"}:
            action_intent = None

        selected_job_name = (
            str(result.get("selected_job_name") or result.get("job_name") or "").strip().lower()
            or None
        )
        if selected_job_name in {"null", "none"}:
            selected_job_name = None
        if selected_job_name and not _looks_like_system_job_name(selected_job_name):
            selected_job_name = None

        requested_scope = str(result.get("requested_scope") or "").strip().lower()
        if requested_scope not in {"single", "all", "unspecified"}:
            requested_scope = "unspecified"

        rationale = str(result.get("rationale") or "").strip()

        return TurnContextDecision(
            route=route,
            action_intent=action_intent,
            selected_job_name=selected_job_name,
            requested_scope=requested_scope,
            rationale=rationale,
        )

    async def decide_next_step(
        self,
        *,
        user_message: str,
        page_context: dict[str, Any],
        plan_summary: str,
        evidence_summary: str,
        action_history_summary: str,
        pending_actions: list[dict[str, Any]],
        continuation: dict[str, Any] | None = None,
        loop_iteration: int = 1,
        replan_count: int = 0,
        verification_summary: str = "",
    ) -> LoopDecision:
        result = await self.run_json(
            system_prompt=(
                "你是 Crater 的 Coordinator Agent。你现在不负责高层路由，"
                "而是负责当前 turn 内的 controller loop 决策。\n\n"
                "你必须在下面步骤中选一个：\n"
                '- "explore": 继续收集只读证据\n'
                '- "execute": 执行当前已准备好的写操作，或让 Executor 产出下一批动作\n'
                '- "verify": 对现有结论做挑战式验证\n'
                '- "replan": 当前计划失效，需要 Planner 基于新信息重规划\n'
                '- "finalize": 信息已足够，可以直接输出最终答复\n\n'
                "决策原则：\n"
                "- 如果已经存在 pending_actions，优先 execute，不要重复 explore。\n"
                "- 如果证据明显不够、参数不明确或目标对象未定位，优先 explore。\n"
                "- explore 只允许用于明确列出的未决事实，并且必须能指出一个直接相关的只读工具；不要把语义相同的缺口拆成多轮 observe/explore。\n"
                "- 如果已有证据能回答、对象健康无需动作、权限不允许、能力边界明确、或没有相关只读工具，优先 finalize。\n"
                "- 只有在证据或世界状态变化导致原计划不再成立时才 replan。\n"
                "- verify 是低频复核，只在写后验证、证据冲突、复杂根因链或确认流安全风险明显时使用；普通只读查询、权限拒绝、noop 健康概览和证据已足够的诊断不要强行 verify。\n"
                "- 对 scale/restart/stop/uncordon/resubmit/create/label/taint 这类明确写意图，如果可用工具里存在真实确认工具且没有权限/风险阻断，必须 execute 推进确认，不能用泛化建议替代。\n"
                "- 不要因为历史动作而忽略当前输入，但 continuation 可以表示这是确认后的继续执行。\n"
                "- continuation、workflow、source_turn_context、confirmation_results、action_history 都是一等上下文；续接时不要丢失已完成/已等待工具和计划进度，也不要重复同一写工具。\n"
                "- 如果用户在问“为什么失败 / 卡在哪 / 现在正常吗 / 有没有 / 用什么配置 / 能不能提交 / 给我完整配置”，且相关只读工具可用，但 evidence_summary 仍为空或只覆盖了部分关键事实，不能 finalize，优先 explore。\n"
                "- 对新建/配置咨询类问题，只要还没覆盖推荐或模板、配额、镜像这些关键事实桶中的大部分，就不能 finalize 成为泛化建议或页面导航。\n"
                "- 对具名对象或明确目标（如 rollout、Prometheus、某作业、某 GPU 型号）的核实请求，只要还没做直接取证，就不能 finalize。\n"
                "- 若当前证据只来自单一推荐类工具，而用户问题还涉及提交可行性、镜像存在性或容量/配额判断，优先继续 explore，而不是直接总结。\n"
                "输出 JSON:\n"
                '{\n'
                '  "step": "explore|execute|verify|replan|finalize",\n'
                '  "rationale": "简短理由"\n'
                '}\n'
            ),
            user_prompt=(
                f"当前用户输入:\n{user_message}\n\n"
                f"页面上下文:\n{page_context}\n\n"
                f"当前计划摘要:\n{plan_summary or '(empty)'}\n\n"
                f"探索摘要:\n{evidence_summary or '(empty)'}\n\n"
                f"执行历史摘要:\n{action_history_summary or '(empty)'}\n\n"
                f"待执行动作:\n{pending_actions or []}\n\n"
                f"验证摘要:\n{verification_summary or '(empty)'}\n\n"
                f"continuation:\n{continuation or {}}\n\n"
                f"loop_iteration={loop_iteration}, replan_count={replan_count}\n\n"
                "请输出结构化 JSON。"
            ),
        )
        return self._parse_loop_decision(result)

    @staticmethod
    def _parse_loop_decision(result: dict[str, Any] | list[Any]) -> LoopDecision:
        if not isinstance(result, dict) or ("raw" in result and len(result) == 1):
            logger.warning("Coordinator loop decision was invalid: %s", result)
            return LoopDecision(step="finalize")

        step = str(result.get("step") or "").strip().lower()
        if step not in {"explore", "execute", "verify", "replan", "finalize"}:
            step = "finalize"
        rationale = str(result.get("rationale") or "").strip()
        return LoopDecision(step=step, rationale=rationale)

    async def summarize(
        self,
        *,
        user_message: str,
        plan_summary: str,
        evidence_summary: str,
        compact_evidence: list[dict[str, Any]] | None = None,
        executor_summary: str,
        verifier_summary: str,
        history_messages: list | None = None,
    ) -> RoleExecutionResult:
        if (
            _looks_complete_user_summary(evidence_summary)
            and _is_effectively_empty_summary(executor_summary)
            and _is_effectively_empty_summary(verifier_summary)
            and not history_messages
        ):
            return RoleExecutionResult(summary=_prepare_explorer_summary_for_user(evidence_summary))

        summary = await self.run_text(
            system_prompt=(
                "你是 Crater 的 Coordinator Agent。你负责整合 Planner、Explorer、Executor、Verifier "
                "的输出，向用户给出最终答复。要求中文、结论在前、证据在后、建议最后。\n"
                "请优先基于实际证据作答，不要只复述其他 agent 的摘要。\n"
                "- 如果当前轮证据已经给出直接根因或直接目标对象，必须以当前轮证据为准；历史里早期的待确认、低置信度或泛化建议只能作为背景，不能和当前结论并列拼接。\n"
                "- 如果本轮是追问上一轮诊断结论，且历史助手结论或历史工具结果已经明确给出根因、错误码、日志关键词或建议动作，在没有新增反证时必须继承这些已确认事实；不要重新说“缺少现场数据”或把已排除的原因重新并列。\n"
                "- 最终答复第一句话必须直接回答用户核心问题；默认按“结论 / 关键证据 / 建议下一步 / 仍缺口或观察点”组织。\n"
                "- 诊断类答复必须覆盖根因、排除依据、用户或管理员下一步；指标对比类必须列出双方数值、差异方向、结论和可优化项；确认/续接类必须说明确认动作、目标对象、执行结果和已有验证证据。\n"
                "- 最终答复必须以成功工具结果为事实源；如果证据无法支持某个结论，必须写成“未命中/无法验证/仍缺少”，不要用空结果或泛化推测支撑正向结论。\n"
                "- Evicted/DiskPressure/ephemeral-storage 类答复不能只说重提或换节点；应结合证据判断是否存在临时目录、debug 输出、日志、checkpoint 或临时文件写爆节点，并给出平台可执行的存储/日志/资源限制建议。\n"
                "- 当证据已经明确是 Prometheus/PVC/TSDB 存储满导致 no data 或 CrashLoopBackOff 时，建议顺序必须是：先恢复或扩容存储，再校验 TSDB/数据写入是否恢复，最后再决定是否谨慎重启；不要把“检查集群健康”放在第一步，也不要建议删除用户作业作为主要处理方式。\n"
                "- 节点 NotReady、RDMA/GPU 驱动死锁、reboot 后仍异常类答复应区分平台内缓解动作和带外恢复边界；不要只给 SSH 排查命令。\n"
                "- 集群健康概览要保留健康等级、关键数字、异常对象、影响范围和紧急程度；建议段直接说明哪些需要马上关注、哪些只是计划维护。\n"
                "- 性能/指标对比类答复的建议段要复述最重要的比较结论，例如双方利用率、吞吐量差异、哪一侧更高效，以及低效一侧可能的 I/O、dataloader、batch size 或同步等待瓶颈，避免只写泛化优化建议。\n"
                "- 指标对比类答复如果证据中包含 throughput、samples/sec、吞吐量、显存占用或效率字段，必须在关键证据或结论里保留这些数值和倍数；不要只写 GPU 利用率或“更高效”。\n"
                "如果 Explorer 摘要里已经出现了准确的结论、根因关键词、状态值或建议动作，优先保留这些事实和动作，不要在润色时丢失关键含义。\n"
                "- 最终答复要保留证据中的对象名、状态值、数值、单位、错误码、关键动作短语；不要把 degraded/healthy、"
                "Pending/Running、OOM、FileNotFoundError、No FailedMount、确认/已执行/已拒绝 等关键信号改写丢失。\n"
                "- 开放式健康、统计、对比和诊断类问题必须覆盖：结论、关键证据、影响范围、建议动作、后续观察点；"
                "如果证据本身已经足以回答，不要再扩展成无关操作清单。\n"
                "- 如果本轮用户是在追问上一轮结论，例如“那为什么/然后呢/还需要做什么/这个是不是说明...”，"
                "必须结合历史消息和本轮证据，先直接回答本轮追问；只保留必要的上一轮背景，不要重新完整复述上一轮诊断、证据和建议。\n"
                "- 多轮对话里，最终答复应围绕当前用户输入的新问题、新确认结果或新增证据收束；"
                "除非用户要求回顾全过程，否则不要把历史回答拼接成一份长报告。\n"
                "- 如果 Executor 里已有确认型写操作的真实结果，最终答复必须优先引用该结果；不要因为后续核查不足或运行时收束，就把已完成动作说成没有结果。\n"
                "- 确认/续接类答复如果涉及执行后验证，必须区分目标状态证据和症状/指标证据；Kubernetes 扩缩容、重启、节点隔离/解除隔离等动作不能只用 Prometheus 指标证明动作已完全落地，应保留 Pod、Endpoints、节点或工作负载状态等直接证据；若缺少这类直接证据，要把它写成仍缺口，而不是说已经完全验证。\n"
                "- 如果执行结果显示新建作业是 Pending，只能说“已提交/已创建且当前 Pending”，不能写成 Running、运行正常或指标正常；若后续证据解释了 Pending 原因，要直接回答排队/资源原因。\n"
                "- 如果证据表明对象运行正常或当前无需动作，结论句必须明确说明对象状态、关键指标是否正常、当前是否需要额外处理。若只是状态确认类问题，建议里给出继续观察或无需额外操作的判断，并补一句具体观察点。\n"
                "- 如果是 Kubernetes rollout / workload 发布卡住，且证据已经指向镜像拉取或发布超时，结论或证据段里要保留原始错误信号，并明确写出 rollout 卡住。建议段要覆盖修正镜像、重新发布或回滚、确认新 Pod 拉取成功，并说明镜像修复后 rollout 才会继续推进。\n"
                "- 如果用户问的是推荐配置、完整提交配置、镜像是否存在、能不能提交，回答必须体现已经核实到的平台事实；不要退化成“能做什么 / 去哪做 / 注意什么”的帮助说明。\n"
                "- 提交/配置建议类答复必须明确覆盖：推荐配置或模板、配额是否满足、可用镜像或镜像缺口、是否能提交、还需要用户确认哪些参数。"
                "其中 `get_resource_recommendation` 只能证明配置建议，不能替代 `list_available_images` 对平台可用镜像的核实；如果镜像未核实，必须直接写成“镜像仍需核实”，不要暗示已有可用镜像。\n"
                "- 不要把用户自述、页面信息或常识默认值写成已验证事实；未核实的信息必须明确标注为建议或待确认。\n"
                "- 若关键事实尚不充分，不要假装已经闭环；应如实指出缺口并给出下一步最小核验。"
            ),
            user_prompt=(
                f"用户请求:\n{user_message}\n\n"
                f"Planner:\n{plan_summary}\n\n"
                f"Explorer:\n{evidence_summary}\n\n"
                f"紧凑证据:\n{compact_evidence or []}\n\n"
                f"Executor:\n{executor_summary}\n\n"
                f"Verifier:\n{verifier_summary}\n\n"
                "请输出最终面向用户的自然语言回复。"
            ),
            history_messages=_filter_summary_history_messages(history_messages),
        )
        return RoleExecutionResult(summary=summary or "已完成最终总结。")
