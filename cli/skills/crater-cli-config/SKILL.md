---
name: crater-cli-config
version: 1.0.0
description: "Crater CLI 配置域：指导 AI Agent 帮用户查看和修改 CLI 本地配置，当前重点支持显示语言切换。用户提到 crater config、language、语言、中文、英文、切换语言、显示语言、配置项、state.json 时使用。"
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater config --help"
---

# Crater CLI 配置

**CRITICAL — 开始前 MUST 先读取 `crater-cli-shared`（可能路径：[`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)），其中包含全局选项、非交互调用、错误处理和敏感信息规则。**

通过 `crater config` 命令帮助用户处理 CLI 本地配置时，遵守本规则。

## 适用场景

- 用户需要切换 Crater CLI 的显示语言。
- 用户询问支持哪些语言，或希望改成中文/英文。
- 用户在脚本或 AI Agent 场景中需要非交互式设置语言。
- 用户遇到语言配置、`state.json` 写入或配置项相关问题。

## 安全原则

- `crater config language` 会修改用户本地 CLI 状态；执行前确认用户确实要切换语言。
- 配置命令只应通过 `crater config ...` 操作，不要要求用户手动编辑 `state.json`。
- 如果用户要求精确语法，先运行 `crater config --help` 或 `crater config language --help`。

## 配置模型

- Crater CLI 的本地状态保存在 `state.json` 中。
- `language` 字段控制 CLI 的显示语言。
- 当前支持语言代码：`en`、`zh-CN`。

## 工作流参考

- 切换显示语言、非交互设置语言、排查语言参数错误：读取 `crater-cli-config-language`（可能路径：[`references/crater-cli-config-language.md`](references/crater-cli-config-language.md)）。

## 常用范例

```bash
crater config language zh-CN
crater config language en
crater config language zh-CN --json
```

## 排查顺序

1. 如果非交互或 `--json` 模式下失败，检查是否提供了 `[LANG]` 位置参数。
2. 如果提示语言非法，确认目标语言是否为 `en` 或 `zh-CN`。
3. 如果配置写入失败，优先判断本地配置目录或 `state.json` 是否不可写。
