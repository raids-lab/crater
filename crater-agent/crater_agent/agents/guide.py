"""Guide agent for product/help oriented questions."""

from __future__ import annotations

from crater_agent.agents.base import BaseRoleAgent, RoleExecutionResult


class GuideAgent(BaseRoleAgent):
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
            max_tools=8,
            include_descriptions=True,
            include_role_policies=False,
        )
        audience = "管理员" if actor_role == "admin" else "普通用户"
        summary = await self.run_text(
            system_prompt=(
                "你是 Crater 的 Guide Agent。"
                "你负责回答“怎么用、支持什么、区别是什么、在哪操作、如何排查”的帮助型问题。"
                "回答要适配当前用户角色，只介绍该角色合理可见的功能；管理员可以提及集群和全局视角，"
                "普通用户优先提及作业、镜像、配额和个人可见能力。"
                "如果某个能力仅在管理员侧可用，要明确标注。"
            ),
            user_prompt=(
                f"回答对象:\n{audience}\n\n"
                f"用户请求:\n{user_message}\n\n"
                f"页面上下文:\n{page_context or {}}\n\n"
                f"能力摘要:\n{capability_summary}\n\n"
                "请输出一段中文帮助说明，按“能做什么 / 去哪做 / 注意什么”组织。"
            ),
        )
        return RoleExecutionResult(summary=summary or "已生成使用说明。")
