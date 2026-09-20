# Crater CLI Global Flags

调用 Crater CLI 时，始终遵守以下通用命令规则，尤其是在为用户执行脚本化、非交互或有副作用的操作时。

## 可执行文件选择

默认使用用户环境中的已安装命令：

```bash
crater --help
```

如果用户说明正在本地开发、测试、调试、验证刚编译出的可执行文件，或当前任务发生在 Crater 仓库源码工作区内，优先使用工作区里的二进制，避免误调用全局旧版本：

```bash
./cli/crater --help   # 从仓库根目录运行
./crater --help       # 从 cli/ 目录运行
```

后续命令也应保持同一个可执行文件前缀，例如把 `crater auth ls --json` 改成 `./cli/crater auth ls --json`。

## `--help` / `-h`

当命令选项不确定、本地 CLI 版本可能变化，或用户要求精确语法时，先查看帮助：

```bash
crater --help
crater auth --help
crater auth login --help
```

不要把 `--help` 当成唯一信息来源；常见工作流仍应优先使用当前任务相关说明中的范例和判断规则。

## 本地版本

根命令提供仅限根级的短版本输出，不会被子命令继承：

```bash
crater --version
crater -v
```

单行输出产品版本和 7 位短 commit。不能与 `--json` 同时使用，也不能带位置参数。

完整本地构建信息用 `crater version`。脚本和 Agent 应读取 `crater version --json` 的 `data.version`：

```bash
crater version
crater version --json
```

`data.version` 含产品版本、完整 commit、构建类型/时间、Go 运行时、`os` / `arch`，以及 CLI `api_version` 与 `min_supported_backend_api_version`。未注入且无法确定的值是 `unknown`；本地开发构建默认产品版本 `dev`、构建类型 `development`。

这两条都不要求登录、不使用 token，也不访问 Crater 平台。它们只标识当前二进制；判断 CLI 与后端 API 是否兼容仍用 `crater compatibility`。

## `--json`

`--json` 用于脚本化调用和 AI 解析输出：

```bash
crater auth ls --json
```

规则：

- 成功输出会被写到 stdout，且是纯 JSON。
- `--json` 会强制 `--no-interactive`。
- `--json` 可以出现在参数序列任意位置。
- 成功体使用顶层信封，业务数据在 `data` 内。

## `--no-interactive`

`--no-interactive` 禁用所有 prompt：

```bash
crater auth logout --yes --no-interactive
```

规则：

- 缺少必要信息时直接失败。
- 需要确认的命令通常必须同时提供 `--yes` / `-y`。
- 不要在非交互模式下期待密码输入、确认框或列表选择。
- 不要为了让命令通过而自动追加 `--yes`；必须先确认用户意图。

## 列表分页

采用公共分页契约的列表会在该命令帮助中提供：

```bash
crater job ls --page 1 --page-size 15 --json
crater job ls --all-pages --json
```

规则：

- `--page` 默认 `1`；`--page-size` 默认 `15`。公共最大值通常为 `200`，`download ls` 及兼容的 `model-download ls`/管理员列表最大为 `100`；以当前命令 `--help` 为准。
- 普通分页 JSON 在资源数组旁返回 `data.pagination.page`、`page_size`、`total`。使用 `total` 与当前页边界判断是否需要请求下一页。
- `--all-pages` 从第一页返回全部筛选结果，并省略 `data.pagination`。服务端分页命令在未显式指定 `--page-size` 时使用端点最大批量，显式合法值仍优先。不要仅为“保险”默认使用它；数据量较大时优先逐页处理。
- 某些命令由服务端分页，某些命令在取得 typed 数组后本地筛选和分页，但对调用方使用相同的 JSON 契约。
- 非法分页参数会和状态、类型等其它本地可发现的问题聚合为一次 `usage_error`；两条及以上问题可从 `context.issues` 逐项修正。

## 错误输出

失败输出会被写到 stderr：

- 默认模式：人类可读错误。
- `--json` 模式：结构化错误对象，包含 `category`、`code`、`message`、可选 `context`。
- 脚本和 AI 判断应消费 `category`、`code`、`context` 等结构化字段，不要解析自然语言 `message`。

## 敏感信息

- 不要让用户在聊天里发送密码、token、cookie 或完整认证文件（含 `state.json`）。
- 普通 shell 中不推荐使用明文 `--password`，因为可能进入 shell history。
- 需要登录时，优先让用户在本机终端交互式输入密码。
