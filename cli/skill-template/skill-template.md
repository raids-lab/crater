---
name: crater-cli-{{domain}}
version: {{meta_version}}
description: "{{meta_description}}"
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater {{command}} --help"
---

# Crater CLI {{title}}

**CRITICAL — 开始前 MUST 先读取 `crater-cli-shared`（可能路径：[`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)），其中包含全局选项、非交互调用、错误处理和敏感信息规则。**

通过 `crater {{command}}` 命令帮助用户处理 {{title}} 相关任务时，遵守本规则。

## 适用场景

{{use_cases}}

## 安全原则

- **禁止要求用户把密码、token、cookie 或 Keyring 内容发到聊天里。**
- 会修改用户本地状态或平台资源的命令，执行前必须确认用户意图。
- 需要跳过确认时，只有在用户明确同意或要求非交互执行时才添加 `--yes` / `-y`。

{{concepts}}

## 工作流参考

{{reference_entries}}

## 常用范例

```bash
{{examples}}
```

## 排查顺序

{{troubleshooting_steps}}
