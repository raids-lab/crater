---
name: crater-devel-release
version: 1.0.0
description: "Crater 发布（开发者侧）：Chart 开发、What's New 与 Release 文案、正式 tag 发布、CI 产物验证和 npm 暂存包人工审批交接。用户修改 charts/、准备更新公告或发布正式版本时使用；开始前须应用 crater-devel-shared。具体集群部署与 rollout 不属于本 Skill。"
---

# Crater 开发 · 发布

**开始前先应用 [`crater-devel-shared`](../crater-devel-shared/SKILL.md)。** 本 Skill 负责开发者侧的发布准备与执行流程，也覆盖 `charts/` 和 `grafana-dashboards/` 开发。只更新公告文案时完成相关准备步骤即可，不自动开始发布。

## 先读权威规范

按任务读取以下章节；版本规则、文案要求与发布约束以这些文档为准，不在 Skill 中另立标准。

| 范围 | 规范入口 |
|------|----------|
| 正式发布、tag、Release 文案、发布状态 | 根 `CONTRIBUTING.md` → Publish Workflows、Release Preparation And Verification、Release Notes、Application Build Versions |
| What's New 内容、版本与验证 | `frontend/CONTRIBUTING.md` → What's New；实现时同时加载 `crater-devel-code` |
| CLI 暂存、管理员审批与安装验证 | `cli/CONTRIBUTING.md` → Release Maintenance、Staged npm publication |
| Chart 版本、配置、README 与验证 | `charts/CONTRIBUTING.md`、`charts/crater/README.md` |
| 文档整理与 PR 审查 | `crater-devel-docs`、`crater-devel-review` |

**边界**：把 Crater 部署 / rollout 到具体（生产 / 内部）集群、`kubectl` 重启、核对 GHCR 镜像 digest、`act-gpu-cluster` 等**集群运维**面向集群管理员，**不属于本 Skill 也不属于 `crater-devel-*` 这套开发者 Skill**；那类任务用面向管理员的 `crater-rollout` Skill。

## Chart 开发

按 Chart 贡献规范判断版本级别，联动 values、模板、README 和英文注释，并执行对应验证；文档引用版本遵循 `website/CONTRIBUTING.md` 的占位约定。对照真实配置时遵守根文档的敏感信息边界。Chart 发布交由 CI，通常无需本地手动打包或推送。

## 正式发布流程

1. **确定范围**：确认目标版本、上一个正式版本、预期发布组件，以及用户授权的是文案准备还是实际发布。读取当前 workflow，不凭历史发布经验猜测触发器或产物。
2. **整理文案**：阅读上一版公告与本次包含的合并改动，按前端 What's New 规范起草简短引言与主题要点，交维护者核对事实和措辞，再更新组件版本与各语言文案。GitHub Release 草稿按根 Release Notes 从同一份已审阅内容转写，不机械保留反馈邀请。
3. **准备合并**：按 shared 的实现、验证、人工检查和 PR 流程完成发布准备；用 review Skill 自检。记录需要维护者亲自查看的弹窗、语言和确认行为。只完成本次授权范围，不能把“准备发布”理解为自动合并 PR。
4. **锁定发布点**：准备内容合入 `main` 后，核对根发布检查项，向用户展示准确的 tag、目标 SHA、包含范围和预期产物。确认已有授权覆盖该版本的 tag 推送及可选 Release 发布；未覆盖时才请求补充授权。维护者授权后只推送明确的 tag，不顺带推送本地分支。
5. **跟踪实际状态**：按目标 tag / SHA 查找各发布 workflow，分别记录成功、失败、仍运行和待审批的组件。失败时先读取 job 日志并核实已有产物；按贡献规范决定重试路径，不把重跑当作自动采用新源码的方式。
6. **交接 npm 审批**：CLI 暂存成功后，给管理员提供 `cli/CONTRIBUTING.md` 中的 Staged Packages 网页链接、目标版本、七个预期包及 workflow 链接，指引其按该文档检查并审批。没有审批证据时明确标为“待管理员审批”，不宣称公开发布完成，也不代替管理员处理 2FA。
7. **验证并收尾**：按根文档和 CLI 文档核验公开产物、隔离安装与版本信息；如获授权创建 Release，先让维护者核对最终文案，确保它与实际可用状态一致。输出 tag / SHA、workflow / Release 链接、产物和测试结果、待办项；发布与集群部署分开交接。
