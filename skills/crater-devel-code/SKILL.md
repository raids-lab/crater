---
name: crater-devel-code
version: 1.0.0
description: "Crater 代码开发：在 backend/、frontend/、cli/ 下开发 Go 后端、React 前端、CLI、API、作业模板、组件、表单、hooks、i18n 与测试。用户修改 backend/、frontend/、cli/ 代码或前后端/CLI 联动时使用；调用 crater 命令则用 cli/skills/crater-cli-*；开始前须应用 crater-devel-shared。"
---

# Crater 开发 · Code

**开始前先应用 [`crater-devel-shared`](../crater-devel-shared/SKILL.md)。**

**完整开发规范见 `backend/CONTRIBUTING.md`、`frontend/CONTRIBUTING.md` 与 `cli/CONTRIBUTING.md`**。本 Skill 只保留 Agent 易漏的代码开发高优先提醒；API、错误、数据库、组件、hooks、i18n、本地调试与 `make` target 细节均以对应 CONTRIBUTING 为准。

若任务是教用户**调用** `crater` 命令，而不是修改 `cli/` 代码，应改用 `cli/skills/crater-cli-*`，不要用本 Skill。

## CLI / 后端 API 兼容版本

处理 CLI 所用 API 前，先按根 `CONTRIBUTING.md` 的版本决策表理解以下参数；它们只描述 CLI 与后端的 API 契约，不适用于管理员统一部署的前端：

| 参数 | 所有者与位置 | 语义和作用 |
|------|--------------|------------|
| 后端 `APIVersion` | `backend/internal/version/api.go` | 当前后端构建实现的最新 CLI API 契约版本。 |
| CLI `APIVersion` | `cli/internal/version/version.go` | 当前 CLI 构建所依据的 CLI API 契约版本；它与后端使用同一个只增不减的逻辑计数器，契约变化时在源码中同步更新。 |
| `MinSupportedCLIAPIVersion` | 后端 `api.go` | 后端仍能支持的最低 CLI 契约版本；用于检查 `CLI APIVersion >= 此值`。 |
| `MinSupportedBackendAPIVersion` | CLI `version.go` | CLI 能支持的最低后端契约版本；用于检查 `后端 APIVersion >= 此值`。 |

双方最低版本相互独立，不要求相等，也不与当前 `APIVersion` 机械联动；两个不等式都满足才表示显式握手兼容。CLI `ProductVersion`、后端 `AppVersion`、提交 SHA、构建类型和构建时间只标识构建产物，不参与 API 兼容判断。

更新时遵循：

- CLI 消费的路径、方法、字段、类型、语义、鉴权或响应结构等 API 契约发生变化时，将后端和 CLI 的 `APIVersion` 同步递增到同一个新值；与 CLI 无关的后端实现或前端接口不提升。
- 分别判断双方是否还存在兼容旧版本对方的回退路径。只有确实无法继续支持时，才把本方的最低支持版本提升到首次提供所需契约的对方 `APIVersion`；CLI 只是开始强依赖已有后端能力时，当前 `APIVersion` 不变，只可能提升 CLI 的最低后端版本。
- PR 必须分别记录当前 `APIVersion` 和本方最低支持版本是否变化及原因，即使结论是不变。版本 Header 只是非可信诊断信息，不得用于鉴权、请求拦截或响应分流；普通业务命令不自动握手，只有 `crater compatibility` 显式检查。

## Backend 提醒

- 管理员接口走 `Admin` 路由和 `Admin*` 命名；用户接口走 `Protected` 路由和 `User*` 命名。外部 API 变更要同步 `swag` 注释。
- 修改 CLI 会调用的 API 时，按上节语义检查并更新 `backend/internal/version/api.go`；后端独立决定 `MinSupportedCLIAPIVersion`，不要代替 CLI 提升其最低后端版本。
- 新接口和新错误路径使用 `bizerr` + `resputil.HandleError`：HTTP 状态表达错误类别，业务码表达稳定机器原因，`msg` 用清晰安全的英文说明用户可理解的问题；底层 cause 用 `Wrap` 留给后端日志，不把内部细节暴露给客户端。
- 修改作业配置字段、请求 / 响应结构或 template 序列化时，按 `backend/CONTRIBUTING.md` 检查克隆作业 / 导入导出配置兼容性；需要阻断旧配置时提升对应前端 `MetadataForm*` version，旧配置仍需可用时补兼容与验证。
- DAO 与 `internal/storage/` 严禁拼接 SQL 字符串，必须参数化查询；不得硬编码密钥、Token、密码、内网 IP。
- 改数据库结构按 `backend/CONTRIBUTING.md` 和 `backend/cmd/gorm-gen/README.md` 走迁移与生成流程，不要只改 model 或只改业务代码。

## Frontend 提醒

- 更新 What's New 时读取 `frontend/CONTRIBUTING.md` 的同名章节，并加载 `crater-devel-release` 串联公告、版本与发布准备；不要把弹窗版本与构建版本或 API 版本混淆。
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
- 维护兼容性诊断时，区分接口明确报告的版本 `0` 与缺失/未知信息：前者作为早于首个正式契约的旧版本参与比较并输出完整不兼容诊断，后者保持未知；两者必须有独立测试。
- CLI 调用新接口时必须保留后端错误事实：终端人类输出说明失败原因，`--json` 错误信封保留 `http_status`、`crater_code`、`msg` 等可序列化字段，方便用户修改命令输入，也方便管理员按业务码和后端日志排查。
- CLI 所用 API 变化时，按上节语义检查并更新 `cli/internal/version/version.go`；CLI 独立决定 `MinSupportedBackendAPIVersion`，不要代替后端提升其最低 CLI 版本。
- 维护 `cli/skills/` 时，把稳定或高频报错沉淀为“错误信息 / 结构化字段 -> 对应情况 -> 用户修正动作 / 管理员排查事实”；写入前先把该理解展示给开发者检查，确认没有误解后再更新 Skill 并提升版本。
- 构建与测试走 `cli/Makefile`；运行前检查 `go version`。如果本地 Go 版本不匹配，提醒开发者可能通过 gvm 管理 Go 版本，并按 `cli/CONTRIBUTING.md` / `go.mod` 切换后再测试。提交前优先 `make pre-commit-check`，包含单元测试、快照校验和 npm 打包脚本测试。涉及用户可见 CLI 行为时，要求开发者手动执行关键命令并检查输入输出。

## 验证

构建、lint、迁移、测试走对应模块 `make`；涉及 Go 前先检查 `go version`。如果版本不符合对应 `go.mod`，提醒开发者可用 gvm 管理并切换 Go 版本，通常在对应 `go.mod` 所在目录按 CONTRIBUTING 指引执行。本地调试通常前后端一起启动，后端通过配置连接测试集群依赖；`make run-storage` 只在 storage-server 相关任务中按需使用。前端 / UI 变化通常提醒开发者在 PR 描述或评论中提供实际界面截图；其他改动在实际执行截图能帮助审查时推荐添加，例如 CLI 输出。
