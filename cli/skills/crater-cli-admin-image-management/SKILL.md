---
name: crater-cli-admin-image-management
version: 1.0.0
description: "Crater CLI 管理员镜像管理：指导 AI Agent 使用 crater admin image 查看、删除、修改所有用户镜像、镜像构建记录与 CUDA base image。仅在用户明确要求管理员/平台级镜像操作时使用。"
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater admin image --help"
---

# Crater CLI 管理员镜像管理

**CRITICAL — 开始前 MUST 先读取 `crater-cli-shared`（可能路径：[`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)）。**

本 Skill 仅用于管理员镜像操作。普通用户镜像构建、上传、分享和 Harbor 凭据流程请使用 `crater-cli-image-management`。

## 命令

```bash
crater admin image build-ls --json
crater admin image build-remove --ids 1,2 --json --no-interactive
crater admin image ls --json
crater admin image delete-many --ids 1,2 --json --no-interactive
crater admin image description 1 --description "Updated" --json --no-interactive
crater admin image type 1 --type jupyter --json --no-interactive
crater admin image tags 1 --tags CUDA,Jupyter --json --no-interactive
crater admin image arch 1 --archs linux/amd64 --json --no-interactive
crater admin image public 1 --json --no-interactive
crater admin image cuda add --image-label cuda124 --label "CUDA 12.4" --value registry/nvidia/cuda:12.4 --json --no-interactive
crater admin image cuda delete 1 --json --no-interactive
```

## 规则

- 管理员命令统一使用 `crater admin image ...` 前缀；不要使用 `--admin`。
- 修改/删除操作会影响平台级镜像资源，执行前确认用户明确要求管理员操作。
- 优先使用 `--json --no-interactive`，读取 `stdout.data.message` 或对应数据键。
- 403 表示 active credentials 不是平台管理员或权限不足。
