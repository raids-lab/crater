# Crater 开发 Skills

面向 Crater monorepo **贡献者**及其 AI Agent 的开发操作指引，源码位于仓库根目录 `skills/`。每个 Skill 只做**指引与任务路由**，不复述权威文档——开发规范以各级 `CONTRIBUTING` 为准，CLI 契约以 `cli/docs/*` 为准，审查清单以 `.github/instructions/*` 为准。

## 边界

| 范围 | 受众 | 归属 |
|------|------|------|
| `skills/crater-devel-*` | 贡献者 | 如何**开发** Crater monorepo（含 Chart 开发与发布产物准备） |
| `cli/skills/crater-cli-*` | 平台用户 | 如何**调用** `crater` 命令 |
| 部署 / rollout 到集群、运维内部环境 | 集群管理员 | 不属本套；用面向管理员的 `crater-rollout` |

本套面向**开发过程**；与具体部署环境强相关的集群运维（如 `act-gpu-cluster`、`kubectl` rollout、镜像 digest 校验）不在范围内。

## 各 Skill 职责

`crater-devel-shared` 是入口：处理任何开发任务前先应用它，再按其路由表加载对应领域 Skill；它也定义“先方案、确认后实现”的 Agent 开发流程与临时 task note 记录方式。

| Skill | 职责 |
|-------|------|
| [`crater-devel-shared`](./crater-devel-shared/) | 入口：仓库结构、文档地图、任务路由与核心工程规则 |
| [`crater-devel-code`](./crater-devel-code/) | `backend/` + `frontend/` + `cli/`：代码开发、API 联动、作业模板、组件、表单、hooks、i18n、CLI 契约 |
| [`crater-devel-docs`](./crater-devel-docs/) | `website/`、`docs/`、仓库 Markdown：文档分类、术语、Chart 版本占位、写作规范 |
| [`crater-devel-release`](./crater-devel-release/) | `charts/` Helm 开发、Chart 版本与发布产物（开发者侧） |
| [`crater-devel-review`](./crater-devel-review/) | 全仓库 PR 审查分级与 PR 描述 |
