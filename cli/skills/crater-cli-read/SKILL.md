---
name: crater-cli-read
version: 1.0.0
description: "Crater CLI 读取域：指导 AI Agent 通过 crater node、crater job、crater image 查看节点列表、作业列表、作业详情和镜像列表。用户提到 CLI 查看节点、任务/作业、镜像列表时使用。"
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater node --help && crater job --help && crater image --help"
---

# Crater CLI 读取

**CRITICAL — 开始前 MUST 先读取 `crater-cli-shared`（可能路径：[`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)），其中包含全局选项、非交互调用、错误处理和敏感信息规则。**

通过 `crater node`、`crater job`、`crater image` 帮助用户查看 Crater 平台信息时，遵守本规则。

## 适用场景

- 用户需要查看集群节点列表或节点详情。
- 用户需要查看作业列表或作业详情。
- 用户需要查看可见镜像或创建作业时可用的镜像。
- 用户需要把节点、作业、镜像数据交给脚本或 AI Agent 继续处理。

## 安全原则

- 本领域当前命令均为只读命令，不修改平台资源。
- 仍需先确认存在 active credentials；不要要求用户提供 token 或 Keyring 内容。
- 脚本化或 Agent 场景优先使用 `--json --no-interactive`。

## 常用范例

```bash
crater node ls --json
crater node get gpu-node-01 --json
crater job ls --json
crater job ls --all --days 7 --status Running --json
crater job ls --interactive --json
crater job get my-job-name --json
crater image ls --json
crater image ls --available --type jupyter --json
```

## 排查顺序

1. 先用 `crater auth ls --json` 确认存在 active credentials。
2. 需要机器解析时加 `--json --no-interactive`，读取 stdout 中的 `data.nodes`、`data.jobs` 或 `data.images`。
3. 作业列表第一版读取 Volcano 作业；如果用户询问 colocate/aijobs 或 sparse/spjobs，需要说明当前 CLI 读取域尚未覆盖。
4. API 失败时根据 stderr JSON 的 `category`、`code`、`context.http_status` 判断是未登录、无权限、资源不存在还是服务端错误。
