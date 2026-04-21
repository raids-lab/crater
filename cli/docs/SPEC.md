# Crater CLI 开发规范和约束

**职责划分**：本文档说明开发 CLI 时要**完成哪些工作**、这些工作**如何落实**，以及须遵守的工程约定与**跨命令公共契约**。各命令的选项、分命令行为与人类可读 / JSON 输出的**指令级细则**以 **[COMMANDS.md](./COMMANDS.md)** 为准，勿在本规范中逐条重复定义。现有代码如何实现、为何如此拆分、模块与调用链如何协作等**实现叙事**写在 **[ARCHITECTURE.md](./ARCHITECTURE.md)**。人类与 AI 增补内容时请分清三者，勿将实现叙事或指令级契约混入本规范。

本规范面向人类与 AI 开发者，须严格遵守。修改意见请通过 ISSUE 或 PR 更新本文档。

---

## 总体

### 名词优先

命令路径遵循 **名词优先** 原则，即域名词在前，动词在后，满足 `crater <domain> <verb>` 的形式。

域名词使用单数形式（如 `image`、`job`），与 API 资源名保持一致。

**例外**：常用命令的简写或模仿 Unix 传统命令名可不遵守此约束。

### 文档驱动

一方面，整个开发过程中必须严格遵守本规范文档；另一方面，使用[指令文档](./COMMANDS.md)驱动新指令的开发和已有指令的修改。

推荐首先在指令文档中详尽的约束指令的**描述、选项、处理逻辑和具体行为**，然后让 AI 根据本规范文档和指令文档进行开发。

需要保证指令的具体行为，和指令文档中所描述的严格一致。

---

## API

本节只规定**开发者**在扩展或修改「与 Crater 平台的 HTTP 交互」时必须怎么做。包分工见 [「模块分层」](./ARCHITECTURE.md#arch-layers)；`internal/api` 组织、错误边界、`cmd` 映射与 `CRATER_HTTP_SIM` 等实现说明见 [「网络通信」](./ARCHITECTURE.md#arch-network)。

### 与平台 HTTP 相关的实现约束

- 所有对外请求的 path 常量写在 `internal/api/paths.go`，禁止在方法体内手写未集中管理的 `"/api/..."` 片段。
- 通用客户端、`Response[T]`、与连接/传输层直接相关的入口（如 `NewClient`、`SetToken`）放在 `internal/api/client.go`；新增域接口时，该域的 DTO、小接口（如 `AuthClient`）、`NewXxxClient` 与同域的 `(*Client)` 方法放在**同一域文件**（如 `auth.go`），避免在 `cmd` 或无关包中散落 HTTP 细节。
- `internal/api` 的导出方法只返回业务数据与 `error`；使用本包错误类型表达「已收到 HTTP 响应但未成功」「网络层失败」等语义。**禁止**在 `internal/api` 内构造 `*clierror.Error`、向 stdout/stderr 输出、或调用 `i18n`。
- 将 `internal/api` 返回的错误映射为 `*clierror.Error` 并在 `RunE` 中 `return` 的工作留在 `cmd`（例如 `cmd/errors.go`）；HTTP 档位与 `Code` 的对应关系遵循本文档「命令结果」中的 `api_error` 约定。
- 仅在本地开发或自测需要**伪造传输层结果**时，使用环境变量 `CRATER_HTTP_SIM`；**允许取值与行为以架构文档「网络通信」节为准**。契约验证与前后端联调须使用真实服务或后端提供的 mock，不得把模拟响应当作平台契约。

---

## 命令结果：错误与成功

**相关章节**：[COMMANDS「全局通用规范」](./COMMANDS.md#commands-global)、[COMMANDS「错误处理规范」](./COMMANDS.md#commands-errors)、[ARCHITECTURE「终端输出」](./ARCHITECTURE.md#arch-terminal)。

### 责任路由

分工如下。

| 位置 | 失败时 | 成功时 |
|------|--------|--------|
| `cmd/*` 各命令 `RunE` | 编排业务；失败则 `return err`。常规路径不要自行往 stderr 写错误 JSON，由 `Execute` → `handleError` 写 stderr 并退出。 | 在 `return nil` 之前完成成功输出，且只在此处写成功内容：`--json` 时用 `internal/output` 的 `WriteSuccessJSON`、`SuccessEnvelope`；否则用 `fmt` 与 `i18n` 写 stdout。`root` 不会替你拼成功 JSON。 |
| `cmd/root.go` 的 `Execute` | `rootCmd.Execute()` 非 nil 时调用 `handleError(err)`，再经 `exitCodeFor` 使用 `pkg/errorcodes.ExitCodeForCategory` 映射退出码并 `os.Exit`。 | `RunE` 成功返回后不再写 stdout；业务成功输出已在各命令内写完。 |

### 失败：`return` 什么

需要稳定错误 JSON 与退出码时，优先 `return` 已组装好的 `*clierror.Error`：`Category`、`Code`（`pkg/errorcodes`）、`Message`（`i18n`）、可选 `Context`。脚本应消费 `Context` 等结构化字段，勿依赖解析自然语言 `Message`。`Message` **允许多行**；人类可读 stderr 由 `internal/output` 在 `Error:` 下对每行统一加基础缩进，行内额外空格由你写入 `Message` 即可，会与基础缩进叠加（见 [COMMANDS「错误处理规范」](./COMMANDS.md#commands-errors) 默认模式说明）。

若不是 `*clierror.Error`：JSON 模式下退化为 `system_error` 与 `ERR_COMMAND_EXECUTION`，`message` 取自 `err.Error()`；字段名与整体形状仍须满足 COMMANDS「错误处理规范」。

### `api_error` 与 HTTP：公共 `Code` 档位

凡 `Category == api_error` 且语义来自**已收到的** HTTP 响应时，`Code` 必须按下表与状态码档位一致，以便跨命令对脚本暴露同一套档位。`usage_error`、`system_error` 等不必带 `_401` 这类后缀。无 HTTP 结果时不得使用带 HTTP 后缀的 `Code`，不得伪造 `http_status`；超时、DNS、连接失败等用 `pkg/errorcodes/codes.go` 中的非 HTTP 后缀码。

| HTTP | `Code`（名与值一致） |
|------|----------------------|
| 401 | `ERR_UNAUTHORIZED_401` |
| 403 | `ERR_FORBIDDEN_403` |
| 404 | `ERR_NOT_FOUND_404` |
| 其它 4xx（非上列） | `ERR_CLIENT_4XX` |
| 500–599 | `ERR_SERVER_INTERNAL_5XX` |
| 其它或无法归类 | `ERR_API_OTHER` |

有 HTTP 时，`Context` 建议包含整数 `http_status`；能读到响应体时再带 `crater_code`、`msg` 等事实。映射以 `cmd/errors.go` 的 `apiCodeForHTTP` 为准；新增档位须同步修改 `pkg/errorcodes/codes.go`、上表，以及 COMMANDS 中引用本节的表述。

### 成功：`--json` 的公共形状

在 stdout 只出合法 JSON、禁止混入装饰性文字等前提下，须满足 COMMANDS 全局与各命令对成功体的规定。实现上：

- 使用 `output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(payload))`。顶层 `status` 使用常量 `output.JSONSuccessStatus`；业务载荷只进 `data` 对象。成功体不使用错误体的 `category` / `code`；`data` 不得含 `http_status`。
- `data` 允许哪些键、是否可为 `{}`、顶层是否带 `message`、敏感字段能否出现等，**只以 COMMANDS 为准**，本规范不列举。

### 成功：非 `--json`

不用顶层信封。`fmt` 与 `i18n` 写人类可读输出至 stdout；字段不必与 JSON 形态一一对应。

### 成功路径上的 JSON 编码失败

`WriteSuccessJSON` 内编码失败时 `return` `*clierror.Error`：`system_error` 与 `ERR_JSON_ENCODE_FAILED`，经 `handleError` 以错误 JSON 写 stderr，进程非零退出。

### 退出码与 `--json` 预解析

`Category` 到进程退出码的映射以 `pkg/errorcodes.ExitCodeForCategory` 为唯一实现源；数值须与 COMMANDS「错误处理规范」一致。`RunE` 返回非 `*clierror.Error` 的退化行为同上节。各命令内禁止 `os.Exit`。若要调整映射，改 `pkg/errorcodes` 并同步 **COMMANDS** 与本文档涉及处。

`Execute` 在 Cobra 解析前对 `os.Args` 做 `--json` 预扫描并与 `viper` 同步，使解析阶段失败时仍可按 JSON 写 stderr；**该行为的产品定义在 COMMANDS 全局 `--json`，实现须与之完全一致**。

### 日志

[TODO] 增加日志系统

## 人文关怀

### 多语言支持

单一二进制支持中文（`zh-CN`）和英文（`en`）。文案经 i18n；`--json` 仅要求键名为英文。错误体 `Message`、成功体可选 `message` 的约定见上文「命令结果：错误与成功」与下文「约束」。

**翻译文件分布**

翻译 key 分散在按命令域分组的 catalog 文件中，统一注册到 `internal/i18n/i18n.go`：

-   catalog_root.go   -> 根命令、全局选项
-   catalog_auth.go   -> auth 命令族
-   catalog_config.go -> config 命令族
-   catalog_errors.go -> 通用错误消息

文件划分逻辑：按命令域划分，新增命令类型时在 `internal/i18n/` 下新建 `catalog_<域>.go`，并在 `i18n.go` 的 `mergeCatalogs()` 中注册。

**Key 规范**

-   命令描述：`<域>_<命令>_short` / `_long`
-   选项描述（i18n key，沿用 `flag_` 前缀）：`flag_<选项名>`
-   提示语：`prompt_<字段>`
-   表格标题：`table_<列名>`
-   结果消息：`<动作>_<结果>`

**用法**

命令定义（Short/Long 在 `cmd/*.go` 中）：

```go
var whoamiCmd = &cobra.Command{
    Use:   "whoami",
    Short: i18n.T("auth_whoami_short"),
    Long:  i18n.T("auth_whoami_long"),
}
```

选项描述（对应 Cobra `Flags()` / `PersistentFlags()` 的 Usage，文案来自 i18n）：

```go
cmd.PersistentFlags().String("platform", "", i18n.T("flag_platform"))
```

消息输出（在 `RunE` 或辅助函数中）：

```go
fmt.Println(i18n.T("login_success", platform, username))
```

**约束**

- `--json`：JSON 键名英文；错误体中的 `message` 用 i18n（与当前语言一致）。成功体为 `status`（固定 `OK`）+ `data` + 可选 `message`，见上文「命令结果：错误与成功」；`message` 是否省略及 `data` 内键以 `COMMANDS.md` 为准。
