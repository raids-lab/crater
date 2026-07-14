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
- 新接口和新错误路径使用 `bizerr` + `resputil.HandleError`：HTTP 状态表达错误类别，业务码表达稳定机器原因，`msg` 用清晰安全的英文说明用户可理解的问题；底层 cause 用 `Wrap` 留给后端日志，不把内部细节暴露给客户端。
- 修改作业配置字段、请求 / 响应结构或 template 序列化时，按 `backend/CONTRIBUTING.md` 检查克隆作业 / 导入导出配置兼容性；需要阻断旧配置时提升对应前端 `MetadataForm*` version，旧配置仍需可用时补兼容与验证。
- DAO 与 `internal/storage/` 严禁拼接 SQL 字符串，必须参数化查询；不得硬编码密钥、Token、密码、内网 IP。
- 改数据库结构按 `backend/CONTRIBUTING.md` 和 `backend/cmd/gorm-gen/README.md` 走迁移与生成流程，不要只改 model 或只改业务代码。

## Frontend 提醒

- 先找可复用组件、表单控件 / metadata form 和 hooks（尤其 `ui-custom/`、`components/form/`、`components/`、`hooks/`）；确实不适配再新建。
- 修改高复用组件、表单控件、metadata form、hooks 或 `ui-custom/` 前，先评估引用范围与兼容性，不要为单个页面随意改公共行为；提醒开发者人工抽查代表性受影响页面。
- 修改作业模板持久化 / 表单配置结构（`MetadataForm*`、`src/components/form/types.ts`、作业表单默认值、clone / template source）时，必须检查是否需要提升对应模板 `version`，并在模板迁移注册表中提供“上一版本 -> 当前版本”的迁移函数；后续旧版本通过链式迁移逐步升级，不支持的更早版本要明确报错，不要静默当作当前结构解析。
- 新增或修改任何使用作业模板配置的加载入口（作业模板、克隆作业、从模板 source 恢复等）时，必须复用统一的模板迁移 / 解析逻辑；不要为单个入口手写局部兼容，也不要把任意 JSON 导入误接入作业模板迁移。
- 身份判断用 `useIsAdmin()`；管理员视图调管理员接口，普通用户调用户接口，前后端身份边界要一致。
- API 错误默认走共享错误处理，保留后端 `msg`、HTTP 状态和业务码等排查事实；只有页面确实需要改变交互时才按 `src/services/error_code.ts` 的具体业务码特殊处理，并在消费错误后调用 `markApiErrorHandled`。
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

- 改变用户可见 CLI 行为时，先更新 `COMMANDS.md`；跨命令规则、参数校验、管理员命名空间、后端契约、Skills 分发和快照要求以 `SPEC.md` 为准；实现边界以 `ARCHITECTURE.md` 为准；收尾自检以 `REVIEW.md` 为准。
- CLI 调用新接口时必须保留后端错误事实：终端人类输出说明失败原因，`--json` 错误信封保留 `http_status`、`crater_code`、`msg` 等可序列化字段，方便用户修改命令输入，也方便管理员按业务码和后端日志排查。
- 维护 `cli/skills/` 时，把稳定或高频报错沉淀为“错误信息 / 结构化字段 -> 对应情况 -> 用户修正动作 / 管理员排查事实”；写入前先把该理解展示给开发者检查，确认没有误解后再更新 Skill 并提升版本。
- 构建与测试走 `cli/Makefile`；运行前检查 `go version`。如果本地 Go 版本不匹配，提醒开发者可能通过 gvm 管理 Go 版本，并按 `cli/CONTRIBUTING.md` / `go.mod` 切换后再测试。提交前优先 `make pre-commit-check`（当前等价于 `make test`）。涉及用户可见 CLI 行为时，要求开发者手动执行关键命令并检查输入输出。

## 验证

构建、lint、迁移、测试走对应模块 `make`；涉及 Go 前先检查 `go version`。如果版本不符合对应 `go.mod`，提醒开发者可用 gvm 管理并切换 Go 版本，通常在对应 `go.mod` 所在目录按 CONTRIBUTING 指引执行。本地调试通常前后端一起启动，后端通过配置连接测试集群依赖；`make run-storage` 只在 storage-server 相关任务中按需使用。涉及 UI 变化时提醒开发者在 PR 中附截图。
