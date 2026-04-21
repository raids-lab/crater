# Crater CLI 架构说明

本文档从**行为与模块边界**说明当前 CLI 代码如何组织。**必须遵守的写法与契约**以 [SPEC.md](./SPEC.md) 与 [COMMANDS.md](./COMMANDS.md) 为准；若与本文档叙述冲突，请在 ISSUE 中报告。

<a id="arch-layers"></a>

## 模块分层

| 区域 | 职责（实现视角） |
|------|------------------|
| `cmd/` | Cobra 命令树、`RunE` 编排；读 flag；调用 `internal/api`、`internal/config`、`internal/credential` 等；成功时调用 `internal/output` 写 stdout；失败时 `return`（多为 `*clierror.Error`）。`cmd/root.go` 的 `Execute` 在调用 Cobra 前预扫描 `--json`、初始化语言与帮助、`handleError` + `exitCodeFor` + `os.Exit`。 |
| `internal/api/` | 与 Crater 平台的 HTTP：拼 URL、发请求、按 `Response[T]` 解包；定义 `RequestError`、`NetworkError` 等供上层映射。 |
| `internal/clierror/` | 结构化 CLI 错误类型 `Error`（`Category` / `Code` / `Message` / `Context`），供 `cmd` 返回、`internal/output` 渲染。 |
| `internal/output/` | 成功 JSON 信封与编码；错误写到 stderr 的渲染。不负责退出码与进程退出。 |
| `internal/config/`、`internal/credential/`、`internal/i18n/` | 本地状态、凭据存储、文案与语言。 |
| `pkg/errorcodes/` | 稳定字符串错误码、`Category` 常量、与退出码映射 `ExitCodeForCategory`。 |

<a id="arch-network"></a>

## 网络通信

CLI 与 Crater 平台之间的请求、响应解析与传输层异常，集中在 `internal/api`；命令层不直接拼 HTTP 细节，只调用该包并处理返回的 `error`。

<a id="arch-internal-api"></a>

### `internal/api` 包内组织

- `paths.go`：仅 path 常量（含版本或模块前缀），避免在方法中散落魔法字符串。
- `client.go`：`Client`、`NewClient`、`SetToken`、`Response[T]`；读取 `CRATER_HTTP_SIM` 并在 `NewClient` 内对 req 客户端注册 Transport 拦截（见下小节）。
- 按域文件（如 `auth.go`）：该域请求/响应 DTO、对外小接口（如 `AuthClient`）、`NewXxxClient` 及 `(*Client)` 上的 HTTP 方法；测试可注入假实现而不必连网。

`cmd` 将 `RequestError` / `NetworkError` 等映射为 `*clierror.Error`（见 `cmd/errors.go` 的 `apiCodeForHTTP` 等）；`internal/api` 不打印、不决定 `--json`。

<a id="arch-http-sim"></a>

### 传输层模拟（`CRATER_HTTP_SIM`）

**目的**：不经由真实网络时快速走通 CLI 的错误分支。**不**替代 OpenAPI 契约或联调。

**实现要点**：环境变量名在代码中为 `internal/api.EnvHTTPSim`（值为 `CRATER_HTTP_SIM`）。`NewClient` 创建 req 客户端后调用 `applyHTTPSim`：按取值在 Transport 上 `WrapRoundTripFunc`，对**所有**经该客户端发出的请求返回同一种伪造结果（与 path 无关）。

| 取值 | 行为 |
|------|------|
| （未设置） | 正常发起真实 HTTP。 |
| `error404` / `404` | 返回 HTTP 404 + 固定 JSON body。 |
| `timeout` / `hang` | RoundTrip 直接返回超时类错误（不睡眠）。 |

<a id="arch-terminal"></a>

## 终端输出

人类可读文案与 `--json` 下的结构化输出，由命令在成功路径调用 `internal/output` 完成；错误由 `Execute` 统一收口后同样经 `internal/output` 写到 stderr。退出进程与退出码仍由 `cmd/root.go` 决定。

### 成功与错误渲染

- 成功：`RunE` 在 `--json` 下调用 `output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(...))`。
- 失败：`Execute` 在 `rootCmd.Execute()` 返回错误后调用 `handleError`，内部为 `output.WriteError(os.Stderr, …)`；退出码由 `exitCodeFor` 结合 `pkg/errorcodes` 与 `*clierror.Error` 的 `Category` 得到。人类可读路径在 `stderr.go`：`Error:` 后按行加两格基础缩进，多行 `Message` 与行首额外空格均支持（空格与基础缩进叠加）；`--json` 时 `message` 仍在单行 JSON 内（换行转义）。

### `--json` 与解析失败

`Execute` 在调用 Cobra 前预扫描 `os.Args` 是否包含 `--json`（及 `--json=true` / `--json` 后接布尔值），并同步 `viper`，使未知 flag 等**解析阶段**失败时，错误仍可按 JSON 输出（与 `COMMANDS.md` 全局说明一致）。

## 本地数据与配置

`internal/config` 管理 `state.json` 等；`internal/credential` 对接系统 keyring 存 token；`internal/i18n` 提供多语言文案。命令层读写这些模块，与网络包解耦。
