---
name: crater-devel-docs
version: 0.1.0
description: "Crater 文档开发：在 website/、docs/ 及仓库各级 Markdown 中维护平台用户文档、开发者文档、i18n、术语与 Chart 版本占位。用户修改文档、文档站、多语言文档或文档规范时使用；开始前须应用 crater-devel-shared。"
---

# Crater 开发 · Docs

**开始前先应用 [`crater-devel-shared`](../crater-devel-shared/SKILL.md)。** 文档站基于 Next.js + Fumadocs。

**完整规范见 `website/CONTRIBUTING.md`**。本 Skill 只保留 Agent 易漏的高优先提醒；文档分类、本地运行、术语、Chart 版本占位、写作规范和验证细节均以 CONTRIBUTING 为准。

## 高优先提醒

- 先判断读者和文档归属：`website/` 面向平台用户 / 管理员；`docs/` 与仓库各级 `.md` 默认面向开发者 / 贡献者。不要把文档写错位置。
- `website/` 的 Helm 部署 / 安装 / 升级命令严禁硬编码 Chart 版本：命令附近用 `<CraterChartVersionNotice />`，代码块内用 `<chart-version>`，Chart 配置页用 `<ChartBadge />`。
- CLI 文档职责要分清：`cli/CONTRIBUTING.md` 是薄入口和本地工作流；`cli/docs/COMMANDS.md` 承载指令级用户可见契约；`cli/docs/SPEC.md` 承载跨命令公共约束、开发规则和 Skills 分发规则；`cli/docs/ARCHITECTURE.md` 承载实现结构与调用链；`cli/docs/REVIEW.md` 承载审查流程与检查重点。维护 CLI 文档时不要把这些职责混写。
- 维护 CLI 使用 Skill 时，常见报错要和对应场景一起沉淀；写入前先把“报错字段、触发场景、含义、用户修正动作、管理员排查事实”的理解展示给开发者检查。
- “账户 (Account)” 在 Crater 中特指调度队列，不要与通用账号概念混用。
- 补充文档前先完整阅读目标文档，把新增内容融入合适章节；必要时重组相邻段落、列表或标题，避免补丁式追加让文档失去清晰脉络。文档改动必须要求开发者人工阅读检查。
- 提交 `website/` 前按 README 处理图片并走 README 的构建指引。
