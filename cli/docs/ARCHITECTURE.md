# Crater CLI 架构说明

本文档从**行为与模块边界**说明当前 CLI 代码如何组织。**必须遵守的写法与契约**以 [SPEC.md](./SPEC.md) 与 [COMMANDS.md](./COMMANDS.md) 为准；阶段性开发完成后的审查流程见 [REVIEW.md](./REVIEW.md)。若与本文档叙述冲突，请在 ISSUE 中报告。

## 模块分层

| 区域 | 职责（实现视角） |
|------|------------------|
| `cmd/` | Cobra 命令树、`RunE` 编排；读 flag；调用 `internal/api`、`internal/session` 等；成功时调用 `internal/output` 写 stdout；失败时 `return`（多为 `*clierror.Error`）。`cmd/root.go` 的 `Execute` 在调用 Cobra 前预扫描 `--json`、初始化语言与帮助、`handleError` + `exitCodeFor` + `os.Exit`。 |
| `internal/api/` | 与 Crater 平台的 HTTP：拼 URL、发请求、按 `Response[T]` 解包；定义 `RequestError`、`NetworkError` 等供上层映射。 |
| `internal/clierror/` | 结构化 CLI 错误类型 `Error`（`Category` / `Code` / `Message` / `Context`），供 `cmd` 返回、`internal/output` 渲染。 |
| `internal/output/` | 成功 JSON 信封与编码；错误写到 stderr 的渲染。不负责退出码与进程退出。 |
| `internal/state/`、`internal/session/`、`internal/i18n/` | 本地状态（含明文 token）、session 门面、文案与语言。 |
| `internal/version/` | CLI 产品版本、源码提交、构建类型、构建时间、Go/平台信息，以及独立的 CLI / 后端 API 兼容版本。 |
| `internal/snaptest/` | 快照测试工具库：构建 `crater` 二进制、运行子进程、收集 `stdout/stderr/exit`、读写与比对 `txtar` golden。仅供测试包使用。 |
| `pkg/errorcodes/` | 稳定字符串错误码、`Category` 常量、与退出码映射 `ExitCodeForCategory`。 |
| `skills/` | 面向平台用户 AI Agent 分发的 Skills，按 `crater-cli-<domain>` 组织；用于说明如何安全调用 CLI，不参与二进制运行时。 |
| `skill-template/` | 编写 Skills 时复用的模板材料，用于保持领域 Skill 与 references 的结构一致。 |
| `hack/`、`npm/` | 可发布二进制的构建/聚合脚本，以及 npm 入口包、平台选择器和包生成/发布工具。 |

## 构建与分发

`cli/Makefile` 的 `release-build` 接收 `GOOS`、`GOARCH`、`OUTPUT` 和四个版本字段，以 `CGO_ENABLED=0` 构建原生二进制。`hack/build-release-artifact.sh` 为 Linux、macOS、Windows 的 `amd64` / `arm64` 六种目标补充 LICENSE、NOTICE 与 README，供 npm 打包使用。

npm 分发采用“一个入口包 + 六个平台包”：

- `@raids-lab/crater-cli` 提供 Node 启动器和 `crater` bin 映射。
- `@raids-lab/crater-cli-<platform>-<arch>` 只包含一个原生二进制，并通过 package.json 的 `os` / `cpu` 限制安装平台。
- 入口包把六个平台包固定为同版本 `optionalDependencies`。`npm/lib/platform.cjs` 将 Node 的 `win32` / `x64` 等命名映射到 Go 的 `windows` / `amd64` 构建产物，运行时只启动当前平台的二进制。
- `npm/scripts/build-packages.mjs` 从 workflow 汇总的原生二进制生成七个可发布目录。workflow 将它们打包为 tarball，先通过独立 job 暂存六个平台包，全部成功后再暂存入口包；公开发布由维护者在 npm 审批完成。

`.github/workflows/cli-pr.yml` 与 `cli-release.yml` 都先运行 `make pre-commit-check`（CLI 单元测试、快照和 npm 打包脚本测试），再由六个独立 job 交叉编译，汇总构建产物并打包全部七个 npm tarball，最后在 Linux 上安装入口包进行冒烟测试。PR 使用 `0.0.0` 作为仅供打包的 npm 版本，不向 npm 提交；正式发布 workflow 只接受精确的 `vX.Y.Z` tag，在确认远端 tag 仍指向原始 commit 后暂存 npm 包，等待维护者审批，不创建 GitHub Release。正式 tag 同时直接触发现有的前端、后端、Storage 与 Helm workflow；GitHub Release 若由维护者填写，只作为更新说明，不触发任何 workflow。

## AI Agent Skills

Skills 是随 CLI 仓库分发给用户 AI Agent 的操作指南，帮助 Agent 在终端中选择正确命令、遵守安全边界并处理常见错误。它们与 Go 代码解耦，不被 `main` 或 `cmd` 加载。

当前目录结构遵循“共享基础 + 领域 Skill + 场景 reference”：

- `skills/crater-cli-shared/`：全局调用规则，包括 `--json`、`--no-interactive`、`--help`、错误输出、敏感信息与确认规则。
- `skills/crater-cli-<domain>/`：具体任务域的 Agent 指引，命名与 CLI 能力域保持一致。
- `skills/crater-cli-<domain>/references/`：同一领域下按用户任务场景拆分的参考文档。
- `skill-template/skill-template.md` 与 `skill-template/reference-template.md`：维护新领域 Skill 与 reference 时的起始模板。

## Tab 补全

Tab 补全由“shell 侧钩子脚本 + 二进制内的 `__complete` 快路径”共同实现：rc 中的钩子在用户按 Tab 时启动一次 `crater __complete ...` 子进程，二进制根据命令树与注册表计算候选并以纯候选行写回 stdout。

- 用户侧安装：`crater completion install bash|zsh` 会在 `~/.bashrc` / `~/.zshrc` 写入一个带 marker 的内联块，注册 shell 的补全钩子（bash `complete`，zsh `compdef`）。脚本用 `command <crater_path> __complete ...` 调用二进制，避免 alias 干扰。
- 用户侧按 Tab：shell 调起钩子，启动一次 `crater __complete bash|zsh ...` 子进程，把当前行词元与光标信息传入；子进程 stdout 只输出候选行，shell 读取后展示/插入。
- 二进制入口：`main` 识别到 argv 为 `__complete` 时早退，不进入根命令执行；由 `cmd/complete_fast.go` 解析 shell 参数并调用引擎。
- 引擎路由：`internal/completion` 先尝试 flag 值补全（仅当该 flag 注册了 `RegisterFlagValue` 才会返回），否则在当前词以 `-` 开头时补 flag 名；再否则依次补子命令与位置参数（`RegisterPositional`）。
  - flag 值补全为兼容不同 shell 的断词规则，会识别三种输入形态并统一路由到同一注册表：
    - `--flag=valuePrefix`
    - `--flag valuePrefix` / `-f valuePrefix`
    - `--flag = valuePrefix` / `-f = valuePrefix`（bash `COMP_WORDBREAKS` 可能将 `=` 断成独立词元）
- 描述与语言：补全快路径只做最小语言初始化，不做 help 文案的全树覆盖；子命令描述与 flag 说明在生成候选时按 `CommandPath()` 推导 i18n key（如 `auth_short`、`flag_mode`）并即时翻译，避免依赖 `cmd/root.go` 的覆盖逻辑。
- 适配与编码：`internal/completion/shell/{bash,zsh}.go` 负责把 shell 侧参数解析成 `completion.Context`，并把 `[]Candidate` 编码为对应 shell 的候选行格式；`registry.go` 提供位置参数与 flag 值的注册表，供各命令在 `init()` 中注册。
- 对外命令与范围：`cmd/completion.go` 提供 `completion bash|zsh` 与 `install` / `uninstall`。当前仅 bash / zsh，不包含 pwsh；指令细则以 COMMANDS / SPEC 为准。

## 网络通信

CLI 与 Crater 平台之间的请求、响应解析与传输层异常，集中在 `internal/api`；命令层不直接拼 HTTP 细节，只调用该包并处理返回的 `error`。

### `internal/api` 包内组织

- `paths.go`：仅 path 常量（含版本或模块前缀），避免在方法中散落魔法字符串。
- `client.go`：`Client`、`NewClient`、`SetToken`、`Response[T]`；读取 `CRATER_TEST_SANDBOX_HTTP` 并在 `NewClient` 内对 req 客户端注册 Transport 拦截（见下小节）。
- `compatibility.go`：调用公开的 `/api/cli/compatibility` 并解包版本范围与可选后端构建信息；`internal/version` 保存 CLI 当前 API 版本、最低后端 API 版本与产品版本，统一客户端据此写入 `User-Agent` 和 `X-Crater-API-Version`。兼容性判断只读取 API 版本字段；后端产品版本、短提交 SHA、构建类型和构建时间仅用于诊断展示。
- 按域文件（如 `auth.go`）：该域请求/响应 DTO、对外小接口（如 `AuthClient`）、`NewXxxClient` 及 `(*Client)` 上的 HTTP 方法；测试可注入假实现而不必连网。

`cmd` 将 `RequestError` / `NetworkError` 等映射为 `*clierror.Error`（见 `cmd/errors.go` 的 `cliErrFromAPI` 与 `apiCodeForHTTP`）；`internal/api` 不打印、不决定 `--json`。

只有 `cmd/compatibility.go` 实现的显式 `crater compatibility` 命令调用版本握手接口，并使用独立短超时。该命令在命令层把接口 404 特殊映射为 `ERR_API_VERSION_MISMATCH`，同时保留底层 HTTP context；`internal/api` 仍按普通 `RequestError` 表达 404，因此其他命令不受影响。普通业务命令和登录命令不自动握手；统一客户端只负责在所有请求上携带诊断 Header。诊断结果不会改变接口响应，也不会成为其他请求的门禁。

### 传输层模拟（`CRATER_TEST_SANDBOX_HTTP`）

**目的**：在不连接外部真实网络的前提下，快速走通 CLI 的错误分支，或让成功快照访问由当前测试进程管理的 loopback fixture。**不**替代 OpenAPI 契约或联调。

**实现要点**：`NewClient` 创建 req 客户端后调用 `applyHTTPSim`，读取 `CRATER_TEST_SANDBOX_HTTP` 并按取值在 Transport 上 `WrapRoundTripFunc`。`error404` 和 `timeout` 对**所有**经该客户端发出的请求返回同一种伪造结果（与 path 无关）；`passthrough` 保留真实 RoundTripper，但在连接前拒绝非 loopback host；兼容性模式只为兼容性接口返回对应结果。

| 取值 | 行为 |
|------|------|
| （未设置） | 正常发起真实 HTTP。 |
| `error404` / `404` | 返回 HTTP 404 + 固定 JSON body。 |
| `timeout` / `hang` | RoundTrip 直接返回超时类错误（不睡眠）。 |
| `passthrough` | 仅允许请求 `localhost` 或 loopback IP，将请求交给下一层 RoundTripper；非 loopback host 在传输前失败。 |
| `compatibility-compatible` | 兼容性接口返回双方兼容；用于诊断命令成功快照。 |
| `compatibility-cli-too-old` | 兼容性接口返回“CLI 版本过旧”；用于诊断命令不兼容快照。 |
| `compatibility-backend-zero` | 兼容性接口明确返回 `0/0`；用于验证前契约后端按“后端版本过旧”输出完整诊断，而不是被当成字段缺失。 |
| `compatibility-unavailable` | 兼容性接口返回旧后端常见的纯文本 404；用于验证 JSON 解码失败不会遮盖真实 HTTP 错误。 |

成功快照的调用链如下：

1. 快照测试进程使用 `httptest.Server` 启动仅服务固定响应的 loopback fixture，并负责其完整生命周期。
2. harness 设置 `CRATER_TEST_SANDBOX_HTTP=passthrough`，并通过 `CRATER_TEST_SANDBOX_PLATFORM_URL` 把 Server URL 传给 CLI 子进程的沙箱 session。
3. `internal/session.fakePlatformURL` 只接受 `http` / `https` 且 host 为 `localhost` 或 loopback IP 的 URL；`internal/api.wrapLoopbackPassthrough` 在实际请求时再次校验目标 host。
4. CLI 子进程请求 fixture，测试收集输出后关闭 Server。

该模式禁止访问外部网络、真实 Crater 平台以及测试启动前已存在的有状态本地服务。fixture 的响应必须由测试代码固定且不依赖机器外部状态；运行快照的环境必须支持测试进程 loopback bind 和 CLI 子进程 loopback connect。

## 列表分页

列表分页由 `internal/api` 的传输模型与 `cmd` 的公共展示 helper 两层协作：

- `internal/api/pagination.go` 定义 typed `ListOptions`、`Page[T]`、默认页大小 `15` 与 `FetchAllPages`。支持分页的域客户端负责把这些选项转换成端点真实 query（例如 Job 使用 `page_size`，download 使用 `pageSize`），命令层不手写分页 URL。
- `cmd/list_options.go` 统一挂载 `--page` / `--page-size` / `--all-pages`、收集分页用法问题、本地切页、构造 `data.pagination` 与打印表格页摘要。服务端分页命令使用 `--all-pages` 且用户未显式指定 `--page-size` 时，公共解析器采用对应端点最大批量；显式值保持不变。领域命令先把分页问题和自己的筛选问题合并，再一次返回 `errUsageFromIssues`。
- Job 与 download 列表使用服务端分页。Job 还支持 owner / 时间范围等本地筛选：存在这些筛选时，命令先以服务端允许的批次顺序拉取全部候选页，再筛选并按用户请求的页大小重新切页；无本地筛选时保留后端返回的页码和总数。
- Node Pod、Job Pod、审批工单、管理员用户、镜像/构建记录和计费作业等数组端点使用本地分页：typed 响应先经过命令约定的过滤与稳定排序，再调用公共 helper。`--all-pages` 绕过本地截断；JSON 完整结果省略 `pagination`。

这样，服务端和本地列表向用户暴露相同的 JSON / 表格契约，但 `total` 始终描述最终筛选结果，而不是某个中间候选页。

## 测试沙箱（Sandbox）

CLI 的快照测试与可复现测试通过环境变量实现“网络与存储”两类外部副作用隔离：

- **网络隔离**：`CRATER_TEST_SANDBOX_HTTP` 由 `internal/api/client.go` 的 `applyHTTPSim` 实现；通常统一模拟传输层失败（如超时、404），成功快照可按上节契约仅放行测试管理的 loopback fixture。
- **存储隔离**：`CRATER_TEST_SANDBOX=1` 由 `internal/session` 实现。开启后，`session` 返回稳定的 fake session（多账号上下文 + fake token），并使写入操作 no-op，从而避免触达开发者真实 `state.json`。

该机制的目的有二：

- 避免测试过程中修改或影响开发者外部环境（配置、凭据、外部网络与真实平台）。
- 避免测试结果依赖外部环境（登录态、网络波动、残留文件），保证稳定可复现。

**边界**：沙箱不是完整的进程隔离。语言/区域相关变量（如 `CRATER_LANG`、`LANG/LC_ALL`）仍会影响输出，快照 harness 需要显式固定它们。

## 终端输出

人类可读文案与 `--json` 下的结构化输出，由命令在成功路径调用 `internal/output` 完成；错误由 `Execute` 统一收口后同样经 `internal/output` 写到 stderr。退出进程与退出码仍由 `cmd/root.go` 决定。

### 成功与错误渲染

- 成功：`RunE` 在 `--json` 下调用 `output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(...))`。
- 失败：`Execute` 在 `rootCmd.Execute()` 返回错误后调用 `handleError`，内部为 `output.WriteError(os.Stderr, …)`；退出码由 `exitCodeFor` 结合 `pkg/errorcodes` 与 `*clierror.Error` 的 `Category` 得到。人类可读路径在 `stderr.go`：`Error:` 后按行加两格基础缩进，多行 `Message` 与行首额外空格均支持（空格与基础缩进叠加）；`--json` 时 stderr 为 `internal/output.MarshalJSONPretty` 格式化的 JSON（`message` 字段内换行仍转义为 `\n`）。需要信号取消的长连接由命令局部创建 context；`job logs --follow` 使用 `signal.NotifyContext(cmd.Context(), os.Interrupt)` 并将其传给日志流请求，不改变其它命令的 Ctrl+C 行为。

### `--json` 与解析失败

`Execute` 在调用 Cobra 前按 pflag bool flag 语义预扫描 `os.Args` 是否包含 `--json` 或 `--json=<bool>`，并同步 `viper`，使未知 flag 等**解析阶段**失败时，错误仍可按 JSON 输出（与 `COMMANDS.md` 全局说明一致）。

## 快照测试（Snapshot / Golden）

快照测试用于锁定 CLI 的关键输出与错误处理行为，避免回归；其职责边界与约定由 [SPEC.md](./SPEC.md) 的「快照测试」节定义。

## 代码单元测试（Unit Test）

单元测试用于覆盖**不依赖子进程**的核心纯逻辑（例如补全引擎、注册表、环境变量开关解析等），以便在重构时提供更快、更精确的失败定位；新增命令/新增包若引入此类逻辑，应视需要补充包内 `_test.go`。

### 代码与数据布局

- **测试代码**：`cli/test/snapshots/<domain>/..._test.go`。每个域可独立维护一组快照用例；测试通过 `internal/snaptest` 执行二进制并比对 golden。
- **golden 数据**：`cli/testdata/snapshots/<domain>/<stem>.<lang>.txtar`，其中 `lang` 为 `en` / `zh-CN`。

### 执行模型

- 测试进程先构建 `crater` 可执行文件（由 `internal/snaptest` 内部一次性完成），再以子进程方式运行各用例。
- 运行环境由快照 harness 统一设置：隔离 HOME、固定 `CRATER_LANG` 与 `LANG/LC_ALL`、关闭交互（用例通常显式传 `--no-interactive`），并默认开启存储沙箱 `CRATER_TEST_SANDBOX=1`，以确保不同机器输出一致。
- `snapshot-check`、`snapshot-update` 与 `pre-commit-check` 都会通过 `cli/test/snapshots/**` 自动执行快照用例；包含成功 fixture 的用例会在测试进程内启动并关闭 loopback Server，因此这些目标的运行环境必须允许 loopback bind/connect。

### 多语言支持

CLI 的多语言由 `internal/i18n` 提供，命令层只负责“选择语言”与“在输出时调用翻译”：

- **文案与字典**：翻译条目分散在 `internal/i18n/catalog_*.go`，在 `internal/i18n/i18n.go` 中合并注册；`i18n.T(key, ...)` 按当前语言取文案。实现上存在回退路径（先回退英文，再回退 key 字面量）用于开发期快速暴露问题，但在产品与 CI 语义上**不允许出现缺失翻译 key**：一旦缺失应视为 bug，必须补齐对应条目。
- **语言决策**：启动时由 `cmd/root.go` 的 `Execute()` 在调用 Cobra 之前完成语言初始化（优先读取本地配置 `state.json` 中的 `language`，否则按环境变量/系统语言探测；实现见 `initLanguageAndHelp()`）。
- **帮助文案覆盖机制**：由于命令树与 flag 通常在包初始化阶段构建，`Short` / `Long` / `Usage` 可能已被“硬编码占位文案”填充。为保证 `--help` 输出与当前语言一致，`Execute()` 会在语言初始化后递归遍历命令树并统一覆盖帮助文案（实现为 `updateHelpTexts(rootCmd)` → `updateAllCommands`）。
  - key 推导规则：对任意命令，取 `CommandPath()`（如 `crater auth login`），去掉根命令后用 `_` 连接得到 `auth_login`，再拼出 `auth_login_short` / `auth_login_long`；flag 的 `Usage` 使用 `flag_<name>`。
  - 边界：该机制只负责帮助/用法相关文本的统一覆盖；业务成功/失败消息仍应在执行路径中直接使用 `i18n.T(...)` 输出（stdout/stderr 与 `--json` 规则见 SPEC/COMMANDS）。

## 本地数据与配置

`internal/state` 管理 `state.json`（身份摘要与 access token 明文同文件）；`internal/session` 是命令层读写本地状态与 token 的入口，输出前必须去掉 `token`；`internal/i18n` 提供多语言文案。命令层不应直接拼配置路径或把磁盘上的 `AuthInfo` 原样打到 stdout。
