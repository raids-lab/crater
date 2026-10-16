---
title: 通过 CLI 连接作业与执行命令
description: 使用 Crater CLI 进入作业 shell、连接 SSH 或运行自动化命令。
---

登录 Crater CLI 后，先用 `crater job ls` 确认平台作业名，再连接自己的运行中作业。以下命令中的 `JOB` 为平台作业名。命令适用于纯 CPU、国产加速卡和其他平台支持的节点；`job exec` 需要支持该功能的后端（API v4），无需本机 kubectl、kubeconfig 或容器 SSH 服务。

## 进入用户 shell

在本机终端执行：

```bash
crater job exec JOB
```

CLI 自动转发输入、分配终端并同步终端尺寸，使用任务用户自己的 HOME 和交互式登录 shell。退出后恢复本机终端。多 Pod 或多容器作业需通过 `--pod POD`、`--container CONTAINER` 选择目标，可先用 `crater job pods JOB` 查看。

## 执行命令和脚本

```bash
crater job exec JOB -- python -V
crater job exec JOB --pod POD -c CONTAINER -- uname -m
cat script.sh | crater job exec JOB -i -- sh
crater job exec JOB -it -- python
crater job exec JOB --json --no-interactive -- python -V
```

命令及参数放在 `--` 后，参数边界原样保留。需要管道、变量展开或重定向时，明确使用 `-- sh -c '...'`。`-i` 转发标准输入；`-t` 分配终端，要求同时使用 `-i` 且本机输入输出均为终端，与 `--json`、`--no-interactive` 互斥。

普通模式流式输出 stdout 和 stderr；TTY 模式合并两者。JSON 成功结果位于 `data.exec`，包含目标、stdout、stderr 和 `exit_code`，输出总量上限为 16 MiB。大文件或二进制输出可使用普通模式重定向。远端命令非零退出时 CLI 返回退出码 4，远端状态保存在错误的 `context.exit_code` 中。

## 用户环境

SSH 连接和 exec 共用用户环境处理，无需手动执行 `/crater-start.sh`。默认优先使用容器的 `NB_USER`，未设置时使用容器当前用户；HOME 和 shell 取自容器用户信息。shell 按自身规则读取配置，例如 Bash 是否读取 `.bashrc` 由登录配置决定。

平台启动脚本会保存初始化后导出的环境，新会话再加载用户配置。已有终端内临时 export 的变量或 Conda 激活状态不会自动传入新会话。普通命令保留容器工作目录，不可访问时回到 HOME；交互 shell 从 HOME 开始。

非 TTY 命令默认先探测用户 shell 导出的环境（最多 10 秒、1 MiB），忽略启动提示后执行原始参数。如果用户配置阻塞或只需容器环境，可执行：

```bash
crater job exec JOB --env-mode container -- env
```

此模式仍使用任务用户和 HOME。别名和 shell 函数需要显式调用对应 shell。自定义镜像若绕过平台启动脚本，需在镜像或作业启动阶段准备好任务用户；镜像还需具备 `/bin/sh`、`env -0`、dd/wc 等基础工具，跨用户执行需要 runuser、sudo 或 su。缺失用户或无法切换身份会报错。旧作业可使用容器环境和用户配置；需要启动脚本保存的新环境时，应通过更新后的部署重新创建作业。

## WebIDE 和 Jupyter 作业

`job exec` 在同一作业容器中新建进程，WebIDE、Jupyter 服务和已有 Notebook 继续运行。它使用任务用户环境，不读取 WebIDE 当前选择的 Python 解释器，也不附着到 Notebook kernel。若需要指定的 Conda 环境或解释器，可明确选择：

```bash
crater job exec JOB -- conda run -n myenv python script.py
crater job exec JOB -- /opt/conda/envs/myenv/bin/python script.py
```

## 使用本机 SSH

先按 [SSH 连接指南](./ssh/ssh-func.md) 配置容器 SSH 服务和公钥，再执行：

```bash
crater job ssh JOB
crater job ssh JOB --connect
crater job ssh JOB --connect --identity-file ~/.ssh/id_ed25519
crater job ssh JOB --connect --user USER
```

不加 `--connect` 时显示连接地址；加上后使用本机 OpenSSH 连接，保留本机 SSH 配置和主机密钥校验。默认使用后端确认的任务用户。`--user` 和 `--identity-file` 仅与 `--connect` 一起使用；指定其他用户时，该用户需能访问共用会话入口。交互连接要求本机输入输出均为终端，不能使用 `--json` 或 `--no-interactive`。没有 sshd 的镜像可使用 `job exec`。
