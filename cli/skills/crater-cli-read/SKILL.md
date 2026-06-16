---
name: crater-cli-read
version: 1.2.0
description: "Crater CLI 用户视图读取域：指导 AI Agent 通过 crater node、job、image、account、resource、dataset、model-download、pod 等用户可见命令查看平台只读信息。管理员视图请使用 crater-cli-admin-read。"
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater node --help && crater job --help && crater image --help && crater account --help && crater resource --help && crater dataset --help"
---

# Crater CLI 读取

**CRITICAL — 开始前 MUST 先读取 `crater-cli-shared`（可能路径：[`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)），其中包含全局选项、非交互调用、错误处理和敏感信息规则。**

通过 `crater node`、`crater job`、`crater image`、`crater account`、`crater resource`、`crater dataset`、`crater model-download`、`crater pod` 等用户视图命令帮助用户查看 Crater 平台信息时，遵守本规则。管理员接口统一走 `crater admin ...`，请使用单独的 `crater-cli-admin-read` Skill。

## 适用场景

- 用户需要查看集群节点列表或节点详情。
- 用户需要查看节点上的 Pod 或节点 GPU 详情。
- 用户需要查看作业列表、作业详情、作业 Pod、事件或 YAML。
- 用户需要查看可见镜像或创建作业时可用的镜像。
- 用户需要查看自己可见的账户、资源、数据集/模型、模板、模型下载、审批单、用户详情、计费或上下文摘要。
- 用户需要查看 Pod 容器、事件、日志、Ingress 或 NodePort。
- 用户需要把平台只读数据交给脚本或 AI Agent 继续处理。

## 安全原则

- 本领域命令均为只读命令，不修改平台资源。
- 仍需先确认存在 active credentials；不要要求用户提供 token 或 Keyring 内容。
- 脚本化或 Agent 场景优先使用 `--json --no-interactive`。
- 不主动调用 token、secret、credential、websocket、terminal 或 log streaming 端点；这些不是普通只读清单能力。
- 不要在普通用户场景调用 `crater admin ...`；如果用户明确要求管理员或平台级数据，切换到 `crater-cli-admin-read`。

## 常用范例

```bash
crater node ls --json
crater node get gpu-node-01 --json
crater node pods gpu-node-01 --json
crater node gpu gpu-node-01 --json
crater job ls --json
crater job ls --all --days 7 --status Running --json
crater job ls --interactive --json
crater job get my-job-name --json
crater job pods my-job-name --json
crater job events my-job-name --json
crater job yaml my-job-name
crater image ls --json
crater image ls --available --type jupyter --json
crater account ls --json
crater resource ls --with-vendor-domain --json
crater dataset ls --json
crater template ls --json
crater model-download ls --category model --json
crater context resources --json
crater billing jobs --all --days 7 --json
crater order ls --json
crater user get wangjh --json
crater pod containers crater-workspace my-pod --json
```

## 排查顺序

1. 先用 `crater auth ls --json` 确认存在 active credentials。
2. 需要机器解析时加 `--json --no-interactive`，读取 stdout 中的 `data.*` 对象，例如 `data.nodes`、`data.jobs`、`data.images`、`data.resources`。
3. Volcano 作业使用 `crater job`。AIJob/SPJob 读命令暂未暴露在本 Skill 中，避免错误使用不一致的后端 ID 契约。
4. API 失败时根据 stderr JSON 的 `category`、`code`、`context.http_status` 判断是未登录、无权限、资源不存在还是服务端错误。
