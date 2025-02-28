---
id: vLLM
title: vLLM 部署
description: 本平台提供了包含 vLLM 的镜像，可以直接使用其创建 Jupyter 任务，快速部署属于您自己的DeepSeek🐋。

sidebar_label: vLLM 部署
sidebar_position: 1
---

# vLLM 部署

## 选择 vLLM 镜像创建 Jupyter 任务

本平台提供了**包含 vLLM 的镜像**，可以直接使用其创建 Jupyter 任务。

![images](./img/vllm-image.png)

在提交任务时需要挂载后续使用的模型文件，目前已提供 **DeepSeek-R1:7b** 的模型文件，可以直接挂载。

![images](./img/ds-model.png)

提交任务时参考的配置文件如下（挂载路径**替换为您的用户名**，**模型挂载路径可自行指定**，但后续**运行 vLLM 时需要同步修改命令中的模型路径**）：

```json
{
  "version": "20241217",
  "type": "jupyter",
  "data": {
    "taskname": "vllm-test",
    "cpu": 32,
    "gpu": {
      "count": 1,
      "model": "nvidia.com/v100"
    },
    "memory": 64,
    "image": "crater-harbor.act.buaa.edu.cn/user-liyilong/vllm:v0.1",
    "envs": [],
    "volumeMounts": [
      {
        "type": 1,
        "subPath": "user/liuxw24",
        "mountPath": "/home/liuxw24"
      },
      {
        "type": 2,
        "subPath": "44",
        "datasetID": 44,
        "mountPath": "/home/liuxw24/model/deepseek-r1-7b"
      }
    ],
    "observability": {
      "tbEnable": false
    },
    "nodeSelector": {
      "enable": false
    },
    "alertEnabled": true,
    "openssh": false
  }
}
```

## 准备工作

### 1. 安装 gcc 及 curl

进入 Jupyter 交互式页面，在终端中执行以下命令来安装 `gcc`（GNU C 编译器）：

```bash
sudo apt update
sudo apt install build-essential
sudo apt install curl
```

- `build-essential` 软件包组包含了 `gcc`、`g++` 以及其他编译 C 和 C++ 代码所需的工具。

- `curl` 在后续访问 vLLM 服务时使用。

### 2. 配置 `CC` 环境变量

在安装好 C 编译器之后，需要确保 `CC` 环境变量指向正确的 C 编译器。

```bash
echo 'export CC=/usr/bin/gcc' >> ~/.bashrc # 如果你使用的是 Bash
echo 'export CC=/usr/bin/gcc' >> ~/.zshrc  # 如果你使用的是 Zsh
```

保存文件后，执行以下命令使设置生效：

```bash
source ~/.bashrc  # Bash
source ~/.zshrc   # Zsh
```

## 运行 vLLM

进入到挂载模型文件的目录（在启动 Jupyter 作业时设置），执行以下命令：

```bash
cd ~/model
vllm serve ./deepseek-r1-7b --dtype=half --enable-chunked-prefill=False --max-model-len=8192 --model-name=deepseek-r1-7b
```

各个参数的解释如下：

- ./deepseek-r1-7b

  - 指定要使用的模型的路径。当前工作目录下的 deepseek-r1-7b 文件夹内包含模型的权重文件、配置文件等必要信息（在启动 Jupyter 作业时挂载，也可以指定为自己的模型路径）


- --dtype=half

  - 指定模型参数的数据类型为半精度浮点数（float16）

- --enable-chunked-prefill=False

  - 禁用分块预填充功能

- --max-model-len=8192

  - 指定模型能够处理的最大输入长度为 8192 个词元（tokens）

:::tip

其中后三个参数可以**根据所申请的 GPU 型号自行调整**，此处为了**保证可以在 V100 上正常运行**，**禁用了 vLLM 的某些特性**。

vLLM serve 时可指定的完整参数解释可以参考 [Engine Arguments](https://docs.vllm.ai/en/latest/serving/engine_args.html)。

:::

成功运行后显示如下：

![images](./img/serve.png)

## 向 vLLM 服务发送请求

当您启动 vllm 服务时，默认会开启一个 HTTP 服务，您可以使用 curl 命令向该服务发送请求，示例如下：

```bash
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
  "model": "./deepseek-r1-7b",
  "messages": [
    {"role": "user", "content": "人工智能中的深度学习和机器学习有什么区别？"}
  ]
}'
```

参考输出如下：

![images](./img/curl.png)

至此，您已经可以和使用 vLLM 部署的 DeepSeek 对话啦🥳！