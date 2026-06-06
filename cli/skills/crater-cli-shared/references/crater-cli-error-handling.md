# Crater CLI Error Handling

调用 `crater` 失败时，优先根据结构化错误字段和退出码判断问题类型，不要解析自然语言 `message`。

## 错误输出位置

- 成功输出写到 stdout。
- 失败输出写到 stderr。
- 默认模式下 stderr 是人类可读错误。
- `--json` 模式下 stderr 是结构化 JSON 错误对象。

## JSON 错误形状

```json
{
  "category": "usage_error | api_error | system_error | cancelled",
  "code": "ERR_...",
  "message": "Human readable message",
  "context": {}
}
```

判断时优先使用：

- `category`：错误大类。
- `code`：稳定错误码。
- `context`：结构化上下文，例如 `http_status`；多条本地用法错误时可能有 `issues`（`field` / `code` / `message` 数组，见 `auth login` 等非交互聚合校验）。
- `message`：只用于向用户解释，不作为稳定程序接口。

`context` 应始终是 JSON 可序列化对象。如果开发者错误地放入无法 JSON 化的值，CLI 会保留原始 `category` / `code` / `message`，并把 `context` 替换为诊断信息，提示错误 context JSON 编码失败、需要联系开发者修复。

## 退出码

| 退出码 | 含义 | 优先处理 |
|------:|------|----------|
| `1` | 非结构化执行错误 | 读取 stderr，按系统异常处理 |
| `2` | `usage_error` | 检查参数、缺失必填信息、非法值或未知命令 |
| `3` | `cancelled` | 用户取消操作；不要自动重试 |
| `4` | `api_error` | 检查网络、认证、权限或平台响应 |
| `5` | `system_error` | 检查本地配置、文件权限、Keyring 或 JSON 编码等本机问题 |

## HTTP 错误码

当 `category == "api_error"` 且 `context.http_status` 存在时，按 HTTP 状态优先判断：

| HTTP | `code` | 常见含义 |
|-----:|--------|----------|
| `401` | `ERR_UNAUTHORIZED_401` | 未登录、token 失效、凭据不可用 |
| `403` | `ERR_FORBIDDEN_403` | 当前账号无权限 |
| `404` | `ERR_NOT_FOUND_404` | 平台资源或接口不存在 |
| 其它 `4xx` | `ERR_CLIENT_4XX` | 请求参数或客户端侧问题 |
| `5xx` | `ERR_SERVER_INTERNAL_5XX` | 平台服务端错误 |
| 其它 | `ERR_API_OTHER` | 无法归类的平台错误 |

## 排查顺序

1. 先看退出码和 `category`，判断是用法错误、取消、API 错误还是本机系统错误。
2. 若是 `usage_error`，查看命令帮助并修正参数：`crater <command> --help`。
3. 若是 `api_error` 且 HTTP 为 401，优先检查登录状态或重新登录。
4. 若是 `api_error` 且 HTTP 为 403，优先检查当前账号权限。
5. 若是 `system_error`，优先检查本地配置文件、Keyring、HOME 和文件权限。
6. 若是 `cancelled`，说明操作被用户主动取消，不要擅自添加 `--yes` 重试。
