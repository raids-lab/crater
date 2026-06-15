---
name: crater-cli-download
version: 1.0.0
description: "Crater CLI 下载域：指导 AI Agent 通过 crater download 创建、等待、查看、暂停、恢复、重试、删除模型和数据集下载任务，并安全处理 Hugging Face / ModelScope token。用户提到 crater download、模型下载、数据集下载、ModelScope、Hugging Face、hf、ms、下载日志、暂停/恢复/重试下载时使用。"
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater download --help"
---

# Crater CLI 下载

**CRITICAL — 开始前 MUST 先读取 `crater-cli-shared`（可能路径：[`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)），其中包含全局选项、非交互调用、错误处理和敏感信息规则。**

通过 `crater download` 帮助用户创建和管理模型/数据集下载任务时，遵守本规则。

## 适用场景

- 用户需要从 ModelScope 或 Hugging Face 下载模型到 Crater 平台。
- 用户需要从 ModelScope 或 Hugging Face 下载数据集到 Crater 平台。
- 用户需要查看下载任务列表、详情、状态或日志。
- 用户需要暂停、恢复、重试或删除下载任务。
- 用户需要处理 gated/private 仓库 token。

## 安全原则

- **禁止要求用户把 Hugging Face、ModelScope token 发到聊天里。**
- 优先推荐 `--token-env` 或 `--token-stdin`，避免 token 进入 shell history。
- 只有用户明确要求在安全脚本环境中直接传参时，才考虑 `--token`；普通交互不要推荐。
- `download create`、`model`、`dataset` 会创建平台侧下载任务；执行前必须确认用户意图。
- `download pause`、`resume`、`retry`、`rm` 会修改平台侧任务状态或用户关联；执行前必须确认用户意图。
- 删除任务需要确认时，只有用户明确同意跳过确认或要求非交互执行时才添加 `--yes` / `-y`。

## 来源与类别

- 来源支持 `modelscope` / `ms` 和 `huggingface` / `hf`。
- 类别支持 `model` 和 `dataset`。
- `ms` 与 `hf` 是 CLI 简写，CLI 会在请求平台前规范化为全拼。

## 工作流参考

- 创建模型或数据集下载、处理 token、等待完成：读取 `crater-cli-download-create`（可能路径：[`references/crater-cli-download-create.md`](references/crater-cli-download-create.md)）。
- 查看任务、日志、暂停、恢复、重试、删除：读取 `crater-cli-download-manage`（可能路径：[`references/crater-cli-download-manage.md`](references/crater-cli-download-manage.md)）。

## 常用范例

```bash
crater download model qwen/Qwen2.5-Coder-7B-Instruct --source hf
crater download dataset AI-ModelScope/alpaca-gpt4-data-zh --source ms
crater download create --name qwen/Qwen2.5-Coder-7B-Instruct --category model --source hf --json
crater download model meta-llama/Llama-2-7b-hf --source hf --token-env HF_TOKEN
crater download model qwen/Qwen2.5-Coder-7B-Instruct --source hf --wait
crater download ls --json
crater download get 123 --json
crater download logs 123
crater download logs 123 --follow
crater download rm 123 --yes --no-interactive
```

## 排查顺序

1. 先用 `crater auth ls --json` 确认存在 active credentials；下载命令需要登录。
2. 参数错误时检查 `--name` 是否为 `owner/name`，`--category` 是否为 `model|dataset`，`--source` 是否为 `modelscope|ms|huggingface|hf`。
3. 私有或 gated 仓库失败时，建议用户在本机设置环境变量并使用 `--token-env`。
4. 任务创建后优先用 `crater download get <id> --json` 查看状态；需要查看所有任务时用 `crater download ls --json`。
5. 下载卡住时先看 `crater download logs <id>`；需要持续观察时使用 `--follow`，不要和 `--json` 组合。
6. 如果需要创建后等待终态，用 `--wait`；脚本场景可组合 `--wait --json`，最终从 `data.download.status` 判断结果。
