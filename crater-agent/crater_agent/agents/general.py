"""General-purpose agent for platform Q&A and lightweight routing."""

from __future__ import annotations

from crater_agent.agents.base import BaseRoleAgent, RoleExecutionResult


class GeneralPurposeAgent(BaseRoleAgent):
    async def respond(
        self,
        *,
        user_message: str,
        page_context: dict,
        capabilities: dict | None = None,
        actor_role: str = "user",
    ) -> RoleExecutionResult:
        capability_summary = self.summarize_capabilities(
            capabilities,
            max_tools=6,
            include_descriptions=True,
            include_role_policies=False,
        )
        summary = await self.run_text(
            system_prompt=(
                "你是 Crater 的 General Purpose Agent。"
                "你负责处理平台内的大多数常规问答、轻量诊断解释和上下文衔接。"
                "如果上下文不足以确认事实，要明确说明“基于当前上下文无法确认”，不要编造。"
                "默认不要建议高风险写操作，除非用户明确要求。"
            ),
            user_prompt=(
                f"当前用户角色:\n{actor_role}\n\n"
                f"用户请求:\n{user_message}\n\n"
                f"页面上下文:\n{page_context or {}}\n\n"
                f"能力摘要:\n{capability_summary}\n\n"
                "请直接输出面向用户的中文回复，结论优先，尽量简洁。"
            ),
        )
        return RoleExecutionResult(summary=summary or "已完成常规平台答复。")
