---
name: crater-cli-image-management
version: 1.0.0
description: "Crater CLI 用户镜像与环境管理：指导 AI Agent 使用 crater image 构建、上传、删除、分享、更新自己可管理的镜像、查看 CUDA base image 和获取 Harbor 凭据。管理员镜像操作请使用 crater-cli-admin-image-management。"
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater image --help"
---

# Crater CLI 镜像与环境管理

**CRITICAL — 开始前 MUST 先读取 `crater-cli-shared`（可能路径：[`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)），其中包含全局选项、非交互调用、错误处理和敏感信息规则。**

通过 `crater image` 管理当前用户可见/可操作的 Crater 镜像、镜像构建和 Harbor 项目时，遵守本规则。

## 适用场景

- 用户需要用 pip/apt、Dockerfile 或 envd 构建镜像。
- 用户需要上传/登记已有镜像链接。
- 用户需要删除镜像或取消/删除镜像构建任务。
- 用户需要修改镜像描述、类型、标签或架构。
- 用户需要分享或取消分享镜像。
- 用户需要查看 CUDA base image。
- 用户需要查看 Harbor 地址、配额或生成 Harbor 项目凭据。

## 安全边界

- 本领域大部分命令会修改平台状态。
- 需要脚本化时优先使用 `--json --no-interactive`。
- `crater image harbor credential` 会创建并输出 Harbor 凭据，必须显式加 `--yes`，不要在不安全日志中暴露输出。
- 不要在用户侧命令使用 `--admin`。管理员镜像操作统一使用 `crater admin image ...`，并切换到 `crater-cli-admin-image-management`。

## 常用范例

```bash
crater image build pip-apt \
  --name cuda-demo \
  --tag v1 \
  --image nvidia/cuda:12.4.1-devel-ubuntu22.04 \
  --packages "git vim" \
  --requirements "torch==2.4.0" \
  --tags CUDA,Pytorch \
  --json --no-interactive

crater image build dockerfile --name custom --tag v1 --file ./Dockerfile --json --no-interactive
crater image build envd --name envd-demo --tag v1 --file ./build.envd --json --no-interactive
crater image upload --image registry/project/repo:tag --type custom --json --no-interactive
crater image delete-many --ids 1,2 --json --no-interactive
crater image description 1 --description "Updated description" --json --no-interactive
crater image type 1 --type jupyter --json --no-interactive
crater image tags 1 --tags CUDA,Jupyter --json --no-interactive
crater image arch 1 --archs linux/amd64 --json --no-interactive
crater image share add 1 --share-type user --ids 10,11 --json --no-interactive
crater image share remove 1 --share-type user --target-id 10 --json --no-interactive
crater image cuda ls --json --no-interactive
crater image harbor credential --yes --json --no-interactive
```

## 工作流

1. 先用 `crater auth ls --json` 确认 active credentials。
2. 构建镜像前确认 `--name`、`--tag` 和构建内容非空。
3. Dockerfile/envd 大内容优先通过 `--file` 传入，避免 shell 转义问题。
4. 批量删除或取消构建时使用逗号分隔 ID。
5. 处理 Harbor 凭据时只在用户明确要求时执行，并提醒其输出敏感。
6. 如果用户要求管理 CUDA base image、平台级镜像列表、删除他人镜像或修改公共可见性，切换到管理员 skill。

## 输出处理

- 成功 JSON 读取 `stdout.data.message` 或具体数据键。
- API 失败读取 stderr JSON 的 `category`、`code`、`context.http_status`。
- 401 表示需要重新登录；403 通常表示权限不足。
