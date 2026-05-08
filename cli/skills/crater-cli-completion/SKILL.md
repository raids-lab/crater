---
name: crater-cli-completion
version: 1.0.1
description: "Crater CLI 补全域：指导 AI Agent 帮用户生成、安装、更新或卸载 bash/zsh Tab 补全脚本，并排查 shell 补全不可用问题。用户提到 crater completion、crater comp、Tab 补全、bash、zsh、.bashrc、.zshrc、补全脚本、completion install/uninstall 时使用。"
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater completion --help"
---

# Crater CLI 补全

**CRITICAL — 开始前 MUST 先读取 `crater-cli-shared`（可能路径：[`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)），其中包含全局选项、非交互调用、错误处理和敏感信息规则。**

通过 `crater completion` 或别名 `crater comp` 帮助用户配置 shell Tab 补全时，遵守本规则。

## 适用场景

- 用户需要安装、更新或卸载 Crater CLI 的 bash/zsh Tab 补全。
- 用户需要输出补全脚本，用于手动查看或集成。
- 用户在按 Tab 时没有补全、补全候选异常，或 shell rc 文件中补全块需要修复。
- 用户询问 Windows、PowerShell、bash、zsh 下如何启用补全。

## 安全原则

- `completion install` 和 `completion uninstall` 会修改用户的 shell rc 文件；执行前必须确认用户意图。
- 非交互执行安装或卸载时，只有用户明确同意跳过确认时才添加 `--yes` / `-y`。
- 只移除 Crater 写入的 marker 块，不建议手动删除用户 rc 文件中的其他内容。

## 补全定位

- 补全功能主要服务人类终端体验，不是 AI 的通用命令发现接口。
- AI 探索命令时优先使用 `crater <command> --help`、领域 skill 和结构化输出。
- `crater __complete ...` 是 shell 钩子使用的快路径，不作为稳定用户接口；只有在排查补全机制本身时才考虑。
- 补全查询不得发起网络请求，适合本地候选（如语言、认证方式）但不应替代领域命令。

## 工作流参考

- 输出补全脚本、安装/更新补全、卸载补全、排查补全不可用：读取 `crater-cli-completion-shell`（可能路径：[`references/crater-cli-completion-shell.md`](references/crater-cli-completion-shell.md)）。

## 常用范例

```bash
crater completion zsh
crater completion bash
crater completion install zsh
crater completion install bash
crater completion uninstall zsh
```

## 排查顺序

1. 确认用户使用的 shell 是 `bash` 或 `zsh`；PowerShell 当前不在支持范围内。
2. 如果只是查看脚本，使用 `crater completion <shell>`，它不修改文件。
3. 如果要安装或更新补全，使用 `crater completion install <shell>`，并确认会修改对应 rc 文件。
4. 如果非交互模式失败，检查是否缺少 `--yes`。
5. 如果安装后仍不可用，先检查对应 rc 文件中是否存在 Crater marker 块，再提示用户重启终端；仅 `source` 当前 rc 文件有时不足以刷新补全状态。
