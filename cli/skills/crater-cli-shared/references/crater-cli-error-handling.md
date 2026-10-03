# Crater CLI Error Handling

调用 `crater` 失败时，优先根据结构化错误字段和退出码判断问题类型。不要凭感觉解析自然语言 `message`；同一 `code` 对应多种本地情形时，只对照下文已记录的 `message`。

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
- `context`：结构化上下文，例如 `http_status`、`crater_code`、`msg`；多条本地用法错误时可能有 `issues`（`field` / `code` / `message` 数组，分页与领域筛选校验也会采用该聚合形态）。
- `message`：主要用于向用户解释。一般不作为程序接口；本地 `ERR_NOT_FOUND` 多种情形的区分见下文已记录的文案。

当领域 Skill 或 reference 有“常见错误与场景”时，优先按其中记录的结构化字段和触发场景判断问题。不要只凭自然语言相似度下结论；若错误字段与记录不一致，应按当前 stderr / `--json` 事实重新分析。

`context` 应始终是 JSON 可序列化对象。如果开发者错误地放入无法 JSON 化的值，CLI 会保留原始 `category` / `code` / `message`，并把 `context` 替换为诊断信息，提示错误 context JSON 编码失败、需要联系开发者修复。

## 退出码

| 退出码 | 含义 | 优先处理 |
|------:|------|----------|
| `1` | 非结构化执行错误 | 读取 stderr，按系统异常处理 |
| `2` | `usage_error` | 检查参数、缺失必填信息、非法值或未知命令 |
| `3` | `cancelled` | 用户取消操作；不要自动重试 |
| `4` | `api_error` | 检查网络、认证、权限或平台响应 |
| `5` | `system_error` | 检查本地配置、文件权限或 JSON 编码等本机问题 |

## 本地认证错误（请求发出前）

部分错误在发出平台请求之前就会返回。它们的 `category` 是 `usage_error`，退出码 `2`，**没有** `context.http_status`。这表示命令还没打到平台，不要按平台 API 失败或普通缺参来处理。

当 `code == "ERR_NOT_FOUND"` 且没有 HTTP 状态、也没有 `context.issues` 时，用下面已记录的 `message` 区分（没有单独错误码，这是少数需要对照文案的情况）：

| `message`（en / zh-CN） | 情况 | 处理 |
|-------------------------|------|------|
| `no token saved for these credentials; please log in again` / `当前账号未保存 token，请重新登录` | 本地已有激活身份，但该条目没有 token。常见于从旧 Keyring 升级后尚未重新登录。`auth ls --json` 能看到身份，但输出故意不含 `token`，**不能**用来判断磁盘上有没有 token。 | 不要读取或粘贴 `state.json`。对同一 `(platform_url, username, method)` 执行 `crater auth login`（普通交互让用户在本机输入密码）。 |
| `no active credentials found` / `未找到当前激活的账号` | 没有 `active_context`。 | 先 `crater auth ls --json`，再按需 `login` 或 `switch`。 |
| `no saved credentials match the criteria` / `未找到匹配的已保存账号` | 筛选条件没有匹配到已保存身份。 | 放宽 `--platform` / `--username` / `--mode`，或先登录。 |

`ERR_NOT_FOUND` 和 `ERR_NOT_FOUND_404` 名字相近，读 stderr 时不要当成同一类问题：前者是本机身份/token 状态（无 `http_status`），后者是平台返回 404、找不到资源或接口。看到本地 `ERR_NOT_FOUND` 时，不要去查作业名、镜像 ID 或接口路径。

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

如果 `context.crater_code` 和 `context.msg` 存在，应把它们作为向用户解释失败原因和向管理员提供排查事实的主要依据。`crater_code` 是后端业务码整数，适合和后端日志、接口实现或管理员排查材料对应；`msg` 是后端返回的可展示消息，适合解释当前请求为什么失败。

## 排查顺序

1. 先看退出码和 `category`，判断是用法错误、取消、API 错误还是本机系统错误。
2. 若是 `usage_error`，有 `context.issues` 时一次修正其中全部字段；再查看命令帮助确认参数：`crater <command> --help`。
3. 若是 `usage_error` 且 `code` 为 `ERR_NOT_FOUND`、没有 `context.issues` 也没有 `http_status`，按上一节「本地认证错误」处理；文案指向未保存 token 时重新登录同一三元组。不要把它判断成平台 404，也不要在文案已指向缺 token 时按「筛不到已保存身份」处理。
4. 若是 `api_error` 且 HTTP 为 401，优先检查登录状态或重新登录。
5. 若是 `api_error` 且 HTTP 为 403，优先检查当前账号权限。
6. 若是 `system_error`，优先检查本地配置文件、`state.json`、HOME 和文件权限。不要读取或粘贴 `state.json` 内容。
7. 若是 `cancelled`，说明操作被用户主动取消，不要擅自添加 `--yes` 重试。
8. 不要先把普通命令失败归因于 API 版本。先排查原命令提示的参数、配置、认证、网络、权限、平台状态等问题，也要考虑 CLI 自身（尤其开发版本）的缺陷。

## API 兼容性后置诊断

只有常见原因无法解释问题，或现象明显指向 CLI 与后端的 API 契约差异时，才显式运行：

```bash
crater compatibility
crater compatibility --platform <URL>
```

该命令无需登录，且普通业务命令不会自动执行它；未提供 `--platform` 时使用当前激活身份保存的平台地址，没有当前激活身份时必须显式提供 `--platform <URL>`。结果按以下方式简读：

- `compatible` 表示双方满足各自声明的最低 API 版本，不代表具体业务命令一定成功。
- `ERR_API_VERSION_MISMATCH` 会指出未满足的最低版本条件；握手接口返回 404 则表示后端较旧、尚未提供该诊断接口。版本号用于兼容性判断，产品版本、提交 SHA 和构建时间等信息只用于定位部署。
- 若版本不一致但用户没有遇到实际问题，说明存在潜在风险即可，让用户继续使用，不要立即要求升级或降级 CLI。
- 若实际问题在排除更常见原因后仍很像 API 不兼容，再把命令给出的升级或降级方向作为最后的排查手段；仍无法解决时，保留结构化输出并联系平台管理员。
