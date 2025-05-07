---
title: 快速部署 DeepSeek R1 单机推理
description: 本平台提供了快速部署 DeepSeek R1 单机推理的作业模板，可以直接使用其创建单机任务，快速部署属于您自己的 DeepSeek ，也可以启动 Web UI 界面和大模型交互。
---

# 快速部署 DeepSeek R1 单机推理

**作业模板** 栏目提供了 **DeepSeek R1 单机推理** 的任务模板，您可以直接选取该模板快速部署 DeepSeek R1 单机推理，也可以启动 Web UI 界面和大模型交互。

## 选取模板创建作业

点击侧边栏的作业模板，之后选取  **DeepSeek R1 单机推理** 模板。

![](./img/sin-deepseek-7b/sin-temp.png)

选取后跳转到新建自定义作业界面，可以看到相关模板参数已经填写完成：

![](./img/sin-deepseek-7b/sin-submit.png)

## 启动命令解释

模板中的启动命令为：

```bash
vllm serve ./deepseek-r1-7b --dtype=half --enable-chunked-prefill=False --max-model-len=8192
```

其中各个参数的解释如下：

- ./deepseek-r1-7b

  - 指定要使用的模型的路径。当前工作目录下的 deepseek-r1-7b 文件夹内包含模型的权重文件、配置文件等必要信息（在启动 Jupyter 作业时挂载，也可以指定为自己的模型路径）


- --dtype=half

  - 指定模型参数的数据类型为半精度浮点数（float16）

- --enable-chunked-prefill=False

  - 禁用分块预填充功能

- --max-model-len=8192

  - 指定模型能够处理的最大输入长度为 8192 个词元（tokens）


> 其中后三个参数可以**根据所申请的 GPU 型号自行调整**，此处为了**保证可以在 V100 上正常运行**，**禁用了 vLLM 的某些特性**。
vLLM serve 时可指定的完整参数解释可以参考 [Engine Arguments](https://docs.vllm.ai/en/latest/serving/engine_args.html)。

## 作业成功运行

提交作业后等待作业运行，进入作业详情页，在基本信息栏可以查看实时输出，可以看到模型已经成功运行。

![](./img/sin-deepseek-7b/sin-detail.png)

可以点击网页终端，使用 curl 命令向该服务发送请求，示例如下：

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

至此，您已经可以和使用作业模板快速部署的 DeepSeek R1 7b 模型对话啦🥳！

## 启动 Web UI 界面和大模型交互

**Open WebUI 客户端模板** 用于和模型部署类型的模板配合，提供友好的大模型试用体验。

点击侧边栏的作业模板，之后选取  **Open WebUI 客户端** 模板。

![](./img/sin-deepseek-7b/openweb-temp.png)

选取后跳转到新建自定义作业界面，可以看到相关模板参数已经填写完成：

![](./img/sin-deepseek-7b/openweb-submit.png)

使用 **DeepSeek R1 单机推理** 的任务模板在 Crater 平台启动大模型推理服务后，您需要修改环境变量第一条，OpenAI 服务的地址：

对于单机部署模型的情况，对应于作业的 **「基本信息」处的「内网 IP」**

![](./img/sin-deepseek-7b/sin-ip.png)

Open WebUI 成功启动后，进入详情页，点击「外部访问」，我们已经设置好了转发，点击即可访问。

![](./img/sin-deepseek-7b/openweb-fw.png)

开始享受您的大模型之旅吧🥳！