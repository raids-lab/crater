"""Approval Agent for job lock order evaluation.

Inherits from TicketAgent base class. Only defines domain-specific parts:
  - Tool whitelist (7 read-only tools)
  - System prompt (approval evaluation rules)
  - Verdict extraction (approve / approve_emergency / escalate)
  - Request/Response models

All ReAct loop execution, fallback, error handling, and trace collection
are handled by TicketAgent.
"""

from __future__ import annotations

import json
import logging
import re
from typing import Any

from pydantic import BaseModel

from crater_agent.agents.ticket_base import TicketAgent
from crater_agent.tools.executor import GoBackendToolExecutor

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Tool whitelist — read-only tools only
# ---------------------------------------------------------------------------
APPROVAL_ALLOWED_TOOLS = [
    "get_job_detail",
    "get_job_events",
    "query_job_metrics",
    "check_quota",
    "get_realtime_capacity",
    "list_cluster_jobs",
    "get_approval_history",
]


# ---------------------------------------------------------------------------
# Request / Response models
# ---------------------------------------------------------------------------
class ApprovalEvalRequest(BaseModel):
    """Request from Go backend to evaluate an approval order."""

    order_id: int
    job_name: str
    extension_hours: int
    user_reason: str = ""
    user_id: int
    username: str = ""
    job_type: str = ""  # hint from Go, agent will verify via tool


class ApprovalEvalResponse(BaseModel):
    """Structured evaluation result returned to Go backend."""

    verdict: str = "escalate"  # "approve" | "approve_emergency" | "escalate"
    confidence: float = 0.5
    reason: str = ""
    approved_hours: int | None = None  # agent-adjusted hours (None = use original)
    user_message: str = ""  # shown to user
    admin_summary: str = ""  # shown to admin (escalate or emergency)
    trace: list[dict[str, Any]] = []  # tool call trace for audit


# ---------------------------------------------------------------------------
# Prompt
# ---------------------------------------------------------------------------
_APPROVAL_SYSTEM_PROMPT = """\
你是 Crater 智算平台的工单审批助手。你的任务是评估一个作业锁定延期申请。

## 工作流程
1. 调用 get_job_detail 获取作业详情（类型、资源规格、运行时长）
   - 特别关注 reminded 字段：true 表示作业已收到清理提醒，24h 内将被删除（应急作业）
   - 关注 lockedTimestamp：当前锁定到期时间
2. 根据作业类型选择性查询：
   - 批处理/训练作业：调用 query_job_metrics 查 GPU 利用率
   - 交互式作业（Jupyter/WebIDE）：不以 GPU 利用率为主要依据
3. 调用 check_quota 查用户配额与资源使用情况
4. 如有需要，调用 get_approval_history 查用户近期审批频率
5. 综合判断后输出结论

## 两条判断轨道

### 轨道一：普通作业（未处于清理倒计时）
- 申请 < 48 小时（2天）：在配额内且资源不紧张时，直接 approve
- 申请 ≥ 48 小时：生成分析报告，escalate 给管理员

### 轨道二：应急作业（即将被清理）
判断标准：作业 reminded=true（已收到清理提醒，意味着最多还有 24h 就会被自动删除）。\
这类作业用户来申请锁定本身就说明情况紧急。

- 申请 < 48 小时：同普通轨道，直接 approve
- 申请 ≥ 48 小时：使用 approve_emergency 先锁定 6h 保命，\
  剩余时间转管理员审批

## 转交条件（无论哪个轨道）
以下情况即使 < 48h 也应 escalate：
- 用户已大量占用高端资源（如已有 4+ 张 A100 在跑）且本次还要加
- 同类资源队列严重紧张
- 批处理 GPU 利用率持续近零
- 频繁申请（7天3+次）
- 不在自身配额范围内

## 交互式作业
Jupyter/WebIDE 不以 GPU 利用率为主要信号，默认宽松。

## 信号缺失
查不到的数据不做假设，倾向转交。

## 效率要求
4-6 次工具调用足够。不要重复调用同一工具。

## 输出格式（三种 verdict）
直接输出 JSON（不要 markdown 代码块）：

approve（直接通过）：
{"verdict":"approve","confidence":0.8,"reason":"...",\
"approved_hours":null,"user_message":"","admin_summary":""}

approve_emergency（应急保命 6h + 剩余转管理员）：
{"verdict":"approve_emergency","confidence":0.7,"reason":"作业即将被清理...",\
"approved_hours":6,"user_message":"已为您应急锁定6小时，剩余时间需管理员审批",\
"admin_summary":"该作业即将被清理，Agent 已应急锁定 6h，用户原始申请 Xh，剩余 X-6h 待审批"}

escalate（转交管理员）：
{"verdict":"escalate","confidence":0.6,"reason":"...",\
"approved_hours":null,"user_message":"...","admin_summary":"..."}
"""

# ---------------------------------------------------------------------------
# Verdict extraction
# ---------------------------------------------------------------------------
_JSON_PATTERN = re.compile(r'\{[^{}]*"verdict"\s*:\s*"[^"]*"[^{}]*\}', re.DOTALL)
_VALID_VERDICTS = {"approve", "approve_emergency", "escalate"}


def _extract_approval_verdict(text: str) -> ApprovalEvalResponse | None:
    """Try to extract verdict JSON from agent's final response."""
    if not text:
        return None

    match = _JSON_PATTERN.search(text)
    if match:
        try:
            data = json.loads(match.group())
            if data.get("verdict") in _VALID_VERDICTS:
                return ApprovalEvalResponse(**{
                    k: v for k, v in data.items()
                    if k in ApprovalEvalResponse.model_fields
                })
        except (json.JSONDecodeError, TypeError, ValueError):
            pass

    # Keyword fallback
    lower = text.lower()
    if any(kw in lower for kw in ("通过", "approve", "批准", "同意")):
        return ApprovalEvalResponse(verdict="approve", confidence=0.5, reason=text[:500])
    if any(kw in lower for kw in ("转交", "escalate", "人工", "管理员")):
        return ApprovalEvalResponse(verdict="escalate", confidence=0.5, reason=text[:500])

    return None


# ---------------------------------------------------------------------------
# ApprovalAgent
# ---------------------------------------------------------------------------
class ApprovalAgent(TicketAgent[ApprovalEvalRequest, ApprovalEvalResponse]):
    """Evaluates job lock approval orders.

    Inherits the full ReAct evaluation pipeline from TicketAgent.
    Only defines approval-specific: tools, prompt, verdict parsing.
    """

    def __init__(
        self,
        tool_executor: GoBackendToolExecutor | None = None,
        llm=None,
    ):
        super().__init__(
            agent_id="approval",
            tool_executor=tool_executor,
            llm=llm,
            llm_purpose="approval",
        )

    def allowed_tools(self) -> list[str]:
        return APPROVAL_ALLOWED_TOOLS

    def system_prompt(self) -> str:
        return _APPROVAL_SYSTEM_PROMPT

    def build_user_message(self, request: ApprovalEvalRequest) -> str:
        parts = [
            "请评估以下作业锁定延期申请：",
            f"- 作业名：{request.job_name}",
            f"- 申请延长：{request.extension_hours} 小时",
        ]
        if request.user_reason:
            parts.append(f"- 用户理由：{request.user_reason}")
        if request.username:
            parts.append(f"- 申请用户：{request.username}")
        if request.job_type:
            parts.append(f"- 作业类型提示：{request.job_type}（请用 get_job_detail 确认）")
        return "\n".join(parts)

    def extract_verdict(self, text: str) -> ApprovalEvalResponse | None:
        return _extract_approval_verdict(text)

    def default_verdict(self, *, reason: str = "") -> ApprovalEvalResponse:
        return ApprovalEvalResponse(
            verdict="escalate",
            confidence=0.1,
            reason=reason or "Agent 评估失败，转交管理员处理",
        )

    def build_context(self, request: ApprovalEvalRequest) -> dict[str, Any]:
        ctx = super().build_context(request)
        ctx["actor"] = {
            "role": "system",
            "user_id": request.user_id,
            "username": request.username or f"user-{request.user_id}",
        }
        ctx["session_id"] = f"approval-{request.order_id}"
        return ctx

    def fallback_prompt(self) -> str:
        return (
            "你是审批助手。基于以下工具调查结果，直接输出评估 JSON。\n"
            '格式：{"verdict":"approve 或 escalate","confidence":0.0-1.0,'
            '"reason":"判断依据","user_message":"","admin_summary":""}'
        )
