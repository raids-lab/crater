# Crater CLI 架构说明

本文档从**行为与模块边界**说明当前 CLI 代码如何组织。**必须遵守的写法与契约**以 [SPEC.md](./SPEC.md) 与 [COMMANDS.md](./COMMANDS.md) 为准；若与本文档叙述冲突，请在 ISSUE 中报告。

## 模块分层

| 区域 | 职责（实现视角） |
|------|------------------|
| `cmd/` | Cobra 命令树、`RunE` 编排；读 flag；调用 `internal/api`、`internal/state`、`internal/credential` 等；成功时调用 `internal/output` 写 stdout；失败时 `return`（多为 `*clierror.Error`）。`cmd/root.go` 的 `Execute` 在调用 Cobra 前预扫描 `--json`、初始化语言与帮助、`handleError` + `exitCodeFor` + `os.Exit`。 |
| `internal/api/` | 与 Crater 平台的 HTTP：拼 URL、发请求、按 `Response[T]` 解包；定义 `RequestError`、`NetworkError` 等供上层映射。 |
| `internal/clierror/` | 结构化 CLI 错误类型 `Error`（`Category` / `Code` / `Message` / `Context`），供 `cmd` 返回、`internal/output` 渲染。 |
| `internal/output/` | 成功 JSON 信封与编码；错误写到 stderr 的渲染。不负责退出码与进程退出。 |
| `internal/state/`、`internal/credential/`、`internal/i18n/` | 本地状态、凭据存储、文案与语言。 |
| `internal/snaptest/` | 快照测试工具库：构建 `crater` 二进制、运行子进程、收集 `stdout/stderr/exit`、读写与比对 `txtar` golden。仅供测试包使用。 |
| `pkg/errorcodes/` | 稳定字符串错误码、`Category` 常量、与退出码映射 `ExitCodeForCategory`。 |
| `skills/` | 面向平台用户 AI Agent 分发的 Skills，按 `crater-cli-<domain>` 组织；用于说明如何安全调用 CLI，不参与二进制运行时。 |
| `skill-template/` | 编写 Skills 时复用的模板材料，用于保持领域 Skill 与 references 的结构一致。 |

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
- 二进制入口：`main` 识别到 argv 为 `__complete` 时早退，不进入 `rootCmd.Execute()`；由 `cmd/complete_fast.go` 解析 shell 参数并调用引擎。
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
- 按域文件（如 `auth.go`）：该域请求/响应 DTO、对外小接口（如 `AuthClient`）、`NewXxxClient` 及 `(*Client)` 上的 HTTP 方法；测试可注入假实现而不必连网。

`cmd` 将 `RequestError` / `NetworkError` 等映射为 `*clierror.Error`（见 `cmd/errors.go` 的 `cliErrFromAPI` 与 `apiCodeForHTTP`）；`internal/api` 不打印、不决定 `--json`。

### 传输层模拟（`CRATER_TEST_SANDBOX_HTTP`）

**目的**：不经由真实网络时快速走通 CLI 的错误分支。**不**替代 OpenAPI 契约或联调。

**实现要点**：`NewClient` 创建 req 客户端后调用 `applyHTTPSim`：读取 `CRATER_TEST_SANDBOX_HTTP`，按取值在 Transport 上 `WrapRoundTripFunc`，对**所有**经该客户端发出的请求返回同一种伪造结果（与 path 无关）。

| 取值 | 行为 |
|------|------|
| （未设置） | 正常发起真实 HTTP。 |
| `error404` / `404` | 返回 HTTP 404 + 固定 JSON body。 |
| `timeout` / `hang` | RoundTrip 直接返回超时类错误（不睡眠）。 |

## 测试沙箱（Sandbox）

CLI 的快照测试与可复现测试通过环境变量实现“网络与存储”两类外部副作用隔离：

- **网络隔离**：`CRATER_TEST_SANDBOX_HTTP` 由 `internal/api/client.go` 的 `applyHTTPSim` 实现，对经 `NewClient` 创建的客户端统一模拟传输层失败（如超时、404）。
- **存储隔离**：`CRATER_TEST_SANDBOX=1` 由 `internal/session` 实现。开启后，`session` 返回稳定的 fake session（多账号上下文 + fake token），并使写入操作 no-op，从而避免触达开发者真实 `state.json` 与 OS keyring。

该机制的目的有二：

- 避免测试过程中修改或影响开发者外部环境（配置、凭据、真实网络）。
- 避免测试结果依赖外部环境（登录态、网络波动、残留文件），保证稳定可复现。

**边界**：沙箱不是完整的进程隔离。语言/区域相关变量（如 `CRATER_LANG`、`LANG/LC_ALL`）仍会影响输出，快照 harness 需要显式固定它们。

## 终端输出

人类可读文案与 `--json` 下的结构化输出，由命令在成功路径调用 `internal/output` 完成；错误由 `Execute` 统一收口后同样经 `internal/output` 写到 stderr。退出进程与退出码仍由 `cmd/root.go` 决定。

### 成功与错误渲染

- 成功：`RunE` 在 `--json` 下调用 `output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(...))`。
- 失败：`Execute` 在 `rootCmd.Execute()` 返回错误后调用 `handleError`，内部为 `output.WriteError(os.Stderr, …)`；退出码由 `exitCodeFor` 结合 `pkg/errorcodes` 与 `*clierror.Error` 的 `Category` 得到。人类可读路径在 `stderr.go`：`Error:` 后按行加两格基础缩进，多行 `Message` 与行首额外空格均支持（空格与基础缩进叠加）；`--json` 时 stderr 为 `internal/output.MarshalJSONPretty` 格式化的 JSON（`message` 字段内换行仍转义为 `\n`）。

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

### 多语言支持

CLI 的多语言由 `internal/i18n` 提供，命令层只负责“选择语言”与“在输出时调用翻译”：

- **文案与字典**：翻译条目分散在 `internal/i18n/catalog_*.go`，在 `internal/i18n/i18n.go` 中合并注册；`i18n.T(key, ...)` 按当前语言取文案。实现上存在回退路径（先回退英文，再回退 key 字面量）用于开发期快速暴露问题，但在产品与 CI 语义上**不允许出现缺失翻译 key**：一旦缺失应视为 bug，必须补齐对应条目。
- **语言决策**：启动时由 `cmd/root.go` 的 `Execute()` 在调用 Cobra 之前完成语言初始化（优先读取本地配置 `state.json` 中的 `language`，否则按环境变量/系统语言探测；实现见 `initLanguageAndHelp()`）。
- **帮助文案覆盖机制**：由于命令树与 flag 通常在包初始化阶段构建，`Short` / `Long` / `Usage` 可能已被“硬编码占位文案”填充。为保证 `--help` 输出与当前语言一致，`Execute()` 会在语言初始化后递归遍历命令树并统一覆盖帮助文案（实现为 `updateHelpTexts(rootCmd)` → `updateAllCommands`）。
  - key 推导规则：对任意命令，取 `CommandPath()`（如 `crater auth login`），去掉根命令后用 `_` 连接得到 `auth_login`，再拼出 `auth_login_short` / `auth_login_long`；flag 的 `Usage` 使用 `flag_<name>`。
  - 边界：该机制只负责帮助/用法相关文本的统一覆盖；业务成功/失败消息仍应在执行路径中直接使用 `i18n.T(...)` 输出（stdout/stderr 与 `--json` 规则见 SPEC/COMMANDS）。

## 本地数据与配置

`internal/state` 管理 `state.json` 等；`internal/credential` 对接系统 keyring 存 token；`internal/i18n` 提供多语言文案。命令层读写这些模块，与网络包解耦。
