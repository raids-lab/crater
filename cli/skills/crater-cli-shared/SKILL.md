---
name: crater-cli-shared
version: 1.0.0
description: "Crater CLI 共享基础：安全调用 crater 命令的通用规则，包括可执行文件选择、全局选项、--json、--no-interactive、--help、错误输出、退出码、敏感信息处理，以及执行会修改用户环境的命令前的确认规则。处理任何 Crater CLI 操作前使用。"
metadata:
  requires:
    bins: ["crater"]
---

# Crater CLI 共享规则

通过 `crater` 命令帮助用户操作 Crater 平台时，先遵守本共享规则。

**CRITICAL — 处理任何 Crater CLI 操作前，先读取本文件。**

## 基础规则

- **不要要求用户把密码、token、cookie 或 Keyring 内容发到聊天里。**
- **执行会修改用户本地状态或平台资源的命令前，必须确认用户意图。**
- 默认调用已安装的 `crater`；如果用户说明正在本地开发、测试或验证刚编译出的 CLI，则优先调用工作区内的二进制，例如从仓库根目录使用 `./cli/crater`，从 `cli/` 目录使用 `./crater`。
- 当本地 CLI 版本可能变化、选项不确定或用户要求精确用法时，先用所选可执行文件运行 `<command> --help` 查看当前帮助。
- 脚本化读取结果时优先使用 `--json`，从 stdout 解析成功数据。
- 失败信息从 stderr 读取；脚本和 AI 判断应优先使用结构化字段，不要依赖自然语言错误文案。

## 全局选项

常用全局行为见 [`references/crater-cli-global-flags.md`](references/crater-cli-global-flags.md)。

## 错误排查

错误分类、退出码与 HTTP 错误码约定见 [`references/crater-cli-error-handling.md`](references/crater-cli-error-handling.md)。

## 安全边界

- 不要静默添加 `--yes` / `-y` 来跳过确认；只有用户明确同意跳过确认或明确要求非交互执行时才使用。
- 不要把用户提供的自由文本拼进 shell 字符串执行；需要执行命令时应保留参数边界，避免 shell 注入。
- 普通交互登录时，让用户在本机终端输入密码；不要让用户把密码发到聊天里。
