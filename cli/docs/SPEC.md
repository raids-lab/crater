# Crater CLI 开发规范和约束

提供给人类和 AI 开发者的开发规范和约束，必须严格遵守。

如有修改意见和建议，请提请 ISSUE 和 PR 修改本文档。

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

### `internal/api`（HTTP 客户端）

本包负责与 Crater **后端的 HTTP 通信**：拼 URL、发请求、解 JSON。**`cmd` 只编排流程，不要在这里写业务状态机。**

**文件分工**

- **`paths.go`**：只放 path 常量（含版本/模块前缀），禁止在方法里散落 `"/api/..."` 字符串。
- **`client.go`**：`Client`、`NewClient`、`SetToken`、通用 `Response[T]`；可选的 **`CRATER_HTTP_SIM`** 调试开关也挂在这里（见下）。
- **按域拆分**（如 **`auth.go`**）：该域 DTO、`AuthClient`、`NewAuthClient`（当前返回 `*Client`）、以及 `(*Client)` 上的 HTTP 方法；新接口优先加在对应域文件。

**调试开关 `CRATER_HTTP_SIM`（仅开发/自测）**

在 **`NewClient` 创建的客户端**上，通过 req 的 **Transport 拦截**统一伪造失败：**不连远端**，所有经该客户端发出的请求都会得到同一种结果（与 path 无关）。用于本地快速验证 CLI 的错误分支；**契约与前后端联调**仍应使用**真实或后端提供的 mock 服务**。

| 取值 | 行为 |
|------|------|
| （未设置） | 正常发起真实 HTTP。 |
| `error404` / `404` | 返回 HTTP 404 + 固定 JSON body。 |
| `timeout` / `hang` | RoundTrip 直接返回超时类错误（不睡眠）。 |

之后这个应该会和错误处理逻辑进行联动和合并。

## 错误处理

### 错误处理逻辑



### 退出码

[TODO] 需要和错误处理一起重构，还有 ctrl c 的问题

### 日志

[TODO] 增加日志系统

## 人文关怀

### 多语言支持

单一二进制支持中文（`zh-CN`）和英文（`en`），切换语言只影响人类可读文本，不影响 `--json` 输出（字段 Key 始终英文）。

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

`--json` 输出的字段 key 必须为英文，不走 i18n。
