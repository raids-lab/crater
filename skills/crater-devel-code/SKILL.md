---
name: crater-devel-code
version: 0.1.0
description: "Crater 代码开发：在 backend/、frontend/、cli/ 下开发 Go 后端、React 前端、CLI、API、作业模板、组件、表单、hooks、i18n 与测试。用户修改 backend/、frontend/、cli/ 代码或前后端/CLI 联动时使用；调用 crater 命令则用 cli/skills/crater-cli-*；开始前须应用 crater-devel-shared。"
---

# Crater 开发 · Code

**开始前先应用 [`crater-devel-shared`](../crater-devel-shared/SKILL.md)。**

**完整开发规范见 `backend/CONTRIBUTING.md`、`frontend/CONTRIBUTING.md` 与 `cli/CONTRIBUTING.md`**。本 Skill 只保留 Agent 易漏的代码开发高优先提醒；API、错误、数据库、组件、hooks、i18n、本地调试与 `make` target 细节均以对应 CONTRIBUTING 为准。

若任务是教用户**调用** `crater` 命令，而不是修改 `cli/` 代码，应改用 `cli/skills/crater-cli-*`，不要用本 Skill。

## Backend 提醒

- 管理员接口走 `Admin` 路由和 `Admin*` 命名；用户接口走 `Protected` 路由和 `User*` 命名。外部 API 变更要同步 `swag` 注释。
- 修改作业配置字段、请求 / 响应结构或 template 序列化时，按 `backend/CONTRIBUTING.md` 检查克隆作业 / 导入导出配置兼容性；需要阻断旧配置时提升对应前端 `MetadataForm*` version，旧配置仍需可用时补兼容与验证。
- DAO 与 `internal/storage/` 严禁拼接 SQL 字符串，必须参数化查询；不得硬编码密钥、Token、密码、内网 IP。
- 改数据库结构按 `backend/CONTRIBUTING.md` 和 `backend/cmd/gorm-gen/README.md` 走迁移与生成流程，不要只改 model 或只改业务代码。

## Frontend 提醒

- 先找可复用组件、表单控件 / metadata form 和 hooks（尤其 `ui-custom/`、`components/form/`、`components/`、`hooks/`）；确实不适配再新建。
- 修改高复用组件、表单控件、metadata form、hooks 或 `ui-custom/` 前，先评估引用范围与兼容性，不要为单个页面随意改公共行为；提醒开发者人工抽查代表性受影响页面。
- 身份判断用 `useIsAdmin()`；管理员视图调管理员接口，普通用户调用户接口，前后端身份边界要一致。
- 非幂等操作必须有确认弹窗；耗时请求加 loading / disabled 防重复提交。
- i18n 不硬编码文本；翻译 key 用英文语义 key 并放到合适 domain；新增 / 修改文案时同步所有语言 `translation.json`。
- 不好理解的输入 / 配置项加帮助图标和 hover tooltip；不要假设平台用户或管理员懂云计算、Kubernetes、调度、存储、网络等术语。

## CLI 提醒

`cli/` 采用文档驱动开发。开发入口与工作流见 `cli/CONTRIBUTING.md`；具体行为契约参考它索引的 `cli/docs/*` 文档。

| 文档 | 权威范围 | 何时读 |
|------|----------|--------|
| `cli/docs/COMMANDS.md` | 指令级契约：命令、flag、位置参数、输出、错误、退出码、交互 | 新增 / 修改任何用户可见行为前先读并更新它 |
| `cli/docs/SPEC.md` | 跨命令公共契约：`--json`、`--no-interactive`、错误信封、退出码、i18n、Tab 补全、快照测试、测试沙箱 | 改公共模块或公共行为前 |
| `cli/docs/ARCHITECTURE.md` | 实现结构、模块边界、调用链、网络通信、补全机制 | 理解实现或调整模块边界时 |
| `cli/docs/REVIEW.md` | 审查流程与检查重点 | 阶段收尾自检或审查时 |

- 改变用户可见 CLI 行为时，先更新 `COMMANDS.md`，再改代码与测试，三者不得漂移。
- `cli/skills/` 面向平台用户，按 `crater-cli-<domain>` 组织；新增 / 修改时参考 `cli/skill-template/`、`crater-cli-shared` 和 `SPEC.md` 的版本规则。
- 构建与测试走 `cli/Makefile`；运行前检查 `go version`。如果本地 Go 版本不匹配，提醒开发者可能通过 gvm 管理 Go 版本，并按 `cli/CONTRIBUTING.md` / `go.mod` 切换后再测试。提交前优先 `make pre-commit-check`（当前等价于 `make test`）。golden 快照必须由规定命令生成，不得手工编辑。

## 验证

构建、lint、迁移、测试走对应模块 `make`；涉及 Go 前先检查 `go version`。如果版本不符合对应 `go.mod`，提醒开发者可用 gvm 管理并切换 Go 版本，通常在对应 `go.mod` 所在目录按 CONTRIBUTING 指引执行。本地调试通常前后端一起启动，后端通过配置连接测试集群依赖；`make run-storage` 只在 storage-server 相关任务中按需使用。涉及 UI 变化时提醒开发者在 PR 中附截图。
