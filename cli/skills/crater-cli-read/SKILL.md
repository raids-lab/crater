---
name: crater-cli-read
version: 1.3.2
description: "Crater CLI 用户视图读取域：指导 AI Agent 通过 crater node、job、image、account、resource、dataset、model-download、pod 等用户可见命令查看平台只读信息。管理员视图请使用 crater-cli-admin-read。"
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater node --help && crater job --help && crater image --help && crater download --help && crater billing --help && crater order --help && crater pod --help"
---

# Crater CLI 读取

**CRITICAL — 开始前 MUST 先读取 `crater-cli-shared`（可能路径：[`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)），其中包含全局选项、非交互调用、错误处理和敏感信息规则。**

通过 `crater node`、`crater job`、`crater image`、`crater account`、`crater resource`、`crater dataset`、`crater model-download`、`crater pod` 等用户视图命令帮助用户查看 Crater 平台信息时，遵守本规则。管理员视图请使用单独的 `crater-cli-admin-read` Skill。

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
- 仍需先确认存在 active credentials；不要要求用户提供 token 或 `state.json` 内容。
- 脚本化或 Agent 场景优先使用 `--json --no-interactive`。
- 不主动调用 token、secret、credential、websocket、terminal 或 log streaming 端点；这些不是普通只读清单能力。
- 不要在普通用户场景调用管理员命令；如果用户明确要求管理员或平台级数据，切换到 `crater-cli-admin-read`。

## 常用范例

```bash
crater node ls --json
crater node get gpu-node-01 --json
crater node pods gpu-node-01 --namespace team-workloads --page-size 15 --json
crater node pods gpu-node-01 --namespace team-workloads --type batch.volcano.sh/v1alpha1/Job --json
crater node pods gpu-node-01 --all-namespaces --all-pages --json
crater node gpu gpu-node-01 --json
crater job ls --search experiment --page-size 15 --json
crater job ls --all --days 7 --search experiment --status Running,Pending --type pytorch --schedule normal --all-pages --json
crater job ls --interactive --json
crater job get my-job-name --json
crater job pods my-job-name --status Running --page-size 15 --json
crater job events my-job-name --json
crater job yaml my-job-name
crater image ls --page-size 15 --json
crater image ls --available --type jupyter --json
crater image build ls --page-size 15 --json
crater account ls --json
crater resource ls --with-vendor-domain --json
crater dataset ls --json
crater template ls --json
crater model-download ls --category model --page-size 15 --json
crater download ls --status Downloading --page-size 15 --json
crater context resources --json
crater billing jobs --all --days 7 --search experiment --page-size 15 --json
crater order ls --status Pending --page-size 15 --json
crater user get wangjh --json
crater pod containers my-pod --namespace team-workloads --json
crater pod logs my-pod main --namespace team-workloads --tail 100 --json
```

## 排查顺序

1. 先用 `crater auth ls --json` 确认存在 active credentials。
2. 需要机器解析时加 `--json --no-interactive`，读取 stdout 中的 `data.*` 对象，例如 `data.nodes`、`data.jobs`、`data.images`、`data.resources`。
3. 公共分页列表默认每页 `15` 条，并返回 `data.pagination`。优先按页读取；只有明确需要完整结果时才使用 `--all-pages`。Job/download 使用服务端分页；Pod、工单、镜像、计费等数组列表会先完成命令约定的本地筛选再分页。
4. `node pods` 必须显式提供 `--namespace` 或 `--all-namespaces`；直接 `pod containers|events|logs|ingresses|nodeports` 必须提供 `--namespace`，也兼容旧的显式 namespace 位置参数。平台没有向普通用户暴露全局作业命名空间配置，因此不要猜测固定默认值；`job get|pods|events|yaml` 按作业 API 定位并保留后端真实 namespace。
5. Volcano 作业使用 `crater job`。AIJob/SPJob 读命令暂未暴露在本 Skill 中，避免错误使用不一致的后端 ID 契约。
6. API 失败时根据 stderr JSON 的 `category`、`code`、`context.http_status` 判断是未登录、无权限、资源不存在还是服务端错误；`usage_error` 有 `context.issues` 时一次修正全部字段。
