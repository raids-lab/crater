---
title: 快速部署 DeepSeek R1 分布式推理
description: 本平台提供了快速部署 DeepSeek R1 分布式推理的作业模板，可以直接使用其创建分布式任务，快速部署属于您自己的 DeepSeek ，也可以启动 Web UI 界面和大模型交互。
---

# 快速部署 DeepSeek R1 分布式推理

**作业模板** 栏目提供了 **DeepSeek R1 分布式推理** 的任务模板，您可以直接选取该模板快速部署 DeepSeek R1 分布式推理，也可以启动 Web UI 界面和大模型交互。

## 选取模板创建作业

点击侧边栏的作业模板，之后选取  **DeepSeek R1 分布式推理** 模板。

![](./img/dis-deepseek-32b/dis-temp.png)

选取后跳转到新建自定义作业界面，可以看到相关模板参数已经填写完成：

![](./img/dis-deepseek-32b/dis-submit.png)

## 启动命令及常见问题

### 启动命令

模板中的启动命令为：

```bash
ray start --head --port=6667 --disable-usage-stats;
NCCL_DEBUG=TRACE python3 -m vllm.entrypoints.openai.api_server \
--model=/models/DeepSeek-R1-Distill-Qwen-32B \
--max-model-len 32768 \
--tensor-parallel-size 4 \
--pipeline-parallel-size 2 \
--gpu-memory-utilization 0.90 \
--max-num-seqs 128 \
--trust-remote-code \
--disable-custom-all-reduce \
--port 8000 \
--dtype=half;
```

### 常见问题

> 问题1 ：ValueError: Bfloat16 is only supported on GPUs with compute capability of at least 8.0. Your Tesla V100-SXM2-32GB GPU has compute capability 7.0. You can use float16 instead by explicitly setting the dtype flag in CLI, for example: --dtype=half.

回答1 ：添加 vLLM 启动参数--dtype=half，该问题主要原因是很多加速算子在v100上跑不起来的无奈之举，现在很多高性能算子都是有一定的硬件限制

> 问题2 ：如何预估需要申请多少算力？

回答2 ：显存用量（最少） >= 模型参数量 * 部署位宽（bit）/ 8 ，例如实例中使用32b的模型采用16bit进行部署，则最少需要64GB显存，crater中一张v100的显存为32GB，实际上3-4张v100就有可能跑起来，不用这块为了测试多机便用了8张卡

> 问题3：可以用sglang吗？

回答3 ：sglang框架不支持使用7.0的显卡进行部署，vllm有些功能也不太支持在v100上用（具体有哪些可以自行去翻一翻vllm的文档和issue区）

至此，您已经可以和使用作业模板快速部署的 DeepSeek R1 32b 模型对话啦🥳！

## 启动 Web UI 界面和大模型交互

**Open WebUI 客户端模板** 用于和模型部署类型的模板配合，提供友好的大模型试用体验。

点击侧边栏的作业模板，之后选取  **Open WebUI 客户端** 模板。

![](./img/dis-deepseek-32b/openweb-temp.png)

选取后跳转到新建自定义作业界面，可以看到相关模板参数已经填写完成：

![](./img/dis-deepseek-32b/openweb-submit.png)

使用 **DeepSeek R1 分布式推理** 的任务模板在 Crater 平台启动大模型推理服务后，您需要修改环境变量第一条，OpenAI 服务的地址：

对于多机部署模型的情况，对应于作业的「基本信息」处，**Ray Head 节点的「内网 IP」**

![](./img/dis-deepseek-32b/dis-ip.png)

Open WebUI 成功启动后，进入详情页，点击「外部访问」，我们已经设置好了转发，点击即可访问。

![](./img/dis-deepseek-32b/openweb-fw.png)

开始享受您的大模型之旅吧🥳！