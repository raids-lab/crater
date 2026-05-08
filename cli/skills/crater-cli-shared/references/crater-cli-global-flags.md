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

## `--json`

`--json` 用于脚本化调用和 AI 解析输出：

```bash
crater auth ls --json
```

规则：

- 成功输出写到 stdout，且是纯 JSON。
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

## 错误输出

失败输出写到 stderr：

- 默认模式：人类可读错误。
- `--json` 模式：结构化错误对象，包含 `category`、`code`、`message`、可选 `context`。
- 脚本和 AI 判断应消费 `category`、`code`、`context` 等结构化字段，不要解析自然语言 `message`。

## 敏感信息

- 不要让用户在聊天里发送密码、token、cookie、Keyring 内容或完整认证文件。
- 普通 shell 中不推荐使用明文 `--password`，因为可能进入 shell history。
- 需要登录时，优先让用户在本机终端交互式输入密码。
