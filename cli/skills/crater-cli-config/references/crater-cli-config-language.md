# Crater CLI Config Language

用户需要切换 CLI 显示语言、设置中文/英文，或处理语言配置错误时，按本流程操作。

**前置要求：先读取 `crater-cli-shared`（可能路径：[`../../crater-cli-shared/SKILL.md`](../../crater-cli-shared/SKILL.md)）。**

**注意：`crater config language` 会修改用户本地 CLI 配置。执行前确认用户确实要切换显示语言。**

## 典型范例

切换为中文：

```bash
crater config language zh-CN
```

切换为英文：

```bash
crater config language en
```

脚本化设置并读取 JSON 输出：

```bash
crater config language zh-CN --json
```

交互式选择语言：

```bash
crater config language
```

查看当前命令帮助：

```bash
crater config language --help
```

## 命令说明

`crater config language [LANG]` 用于切换 CLI 的显示语言。

位置参数：

- `[LANG]`：目标语言代码，可选值为 `en` 或 `zh-CN`。

行为：

- 提供 `[LANG]` 时，直接验证并写入本地配置。
- 未提供 `[LANG]` 且处于交互模式时，展示语言列表供用户选择。
- `--no-interactive` 或 `--json` 模式下必须提供 `[LANG]`，否则失败。
- 成功后立即使用新语言输出成功提示。

## 判断规则

- 用户说“切换中文”“改成中文”：使用 `zh-CN`。
- 用户说“切换英文”“use English”：使用 `en`。
- 用户只说“帮我改语言”但未指定目标语言：交互模式可以运行 `crater config language`；非交互场景应先询问目标语言。
- 用户要求脚本化、CI 或 JSON 输出：必须提供 `[LANG]`，不要依赖交互选择。

## 输出与字段

`--json` 成功数据：

- `data.language`：目标语言代码。

示例：

```bash
crater config language en --json
```

## 安全注意

- 不要要求用户手动编辑 `state.json`。
- 不要在没有明确目标语言时，替用户猜测并执行非交互切换。

## 排查重点

- 缺少 `[LANG]`：在 `--no-interactive` 或 `--json` 模式下补上 `en` 或 `zh-CN`。
- 非法语言：仅支持 `en` 与 `zh-CN`。
- 写入失败：检查本地配置路径或权限问题。
