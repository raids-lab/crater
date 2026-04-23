# 面向智算平台的运维智能体设计与实现

---

**摘要**

随着深度学习模型参数规模的持续增长，智算平台已成为高校与科研机构开展 AI 研究的核心基础设施。智算平台普遍采用云原生架构，以 Kubernetes 为资源编排基础，集成 GPU 调度、分布式存储、容器镜像管理与性能监控等功能。然而，此类平台的运维复杂度高、故障模式多样、跨层因果传播特征显著，传统的人工运维模式已难以满足平台规模化运行的需求。本文以高校智算平台 Crater 为研究对象，设计并实现了一个基于大语言模型（LLM）的运维智能体系统。该系统采用前端-后端-Agent 三层架构，通过 ReAct（Reasoning + Acting）推理模式驱动 75 个平台运维工具的灵活组合，实现作业故障诊断、集群健康巡检、资源利用率分析和安全缓解操作四大类运维场景的智能化处理。系统引入分层可组合的工具架构、基于角色的访问控制、写操作确认机制和完整的审计追踪，在保障运维操作安全性的前提下显著降低了用户的排查门槛。在 Crater 平台（59 节点、220 张 GPU、464 名用户）的实际运行环境中，系统能够有效处理 OOM 故障诊断、分布式训练网络排查、GPU 空闲检测等典型运维场景。

**关键词**：智算平台；运维智能体；大语言模型；Kubernetes；ReAct；工具调用

---

**Abstract**

With the continuous growth of deep learning model parameters, intelligent computing platforms have become core infrastructure for AI research in universities and research institutions. These platforms typically adopt cloud-native architectures based on Kubernetes orchestration, integrating GPU scheduling, distributed storage, container image management, and performance monitoring. However, the operational complexity of such platforms is high, with diverse failure modes and significant cross-layer causal propagation characteristics. Traditional manual operations can no longer meet the demands of large-scale platform operation. This paper takes the university intelligent computing platform Crater as the research subject, and designs and implements an operations intelligence agent system based on Large Language Models (LLMs). The system adopts a three-tier architecture (Frontend-Backend-Agent), drives flexible composition of 75 platform operation tools through the ReAct (Reasoning + Acting) reasoning pattern, and achieves intelligent handling of four major categories of operational scenarios: job failure diagnosis, cluster health inspection, resource utilization analysis, and safe mitigation operations. The system introduces a layered composable tool architecture, role-based access control, write operation confirmation mechanisms, and complete audit trails, significantly lowering the troubleshooting threshold while ensuring operational safety. In the actual operating environment of the Crater platform (59 nodes, 220 GPUs, 464 users), the system effectively handles typical operational scenarios such as OOM failure diagnosis, distributed training network troubleshooting, and GPU idle detection.

**Keywords**: Intelligent Computing Platform; Operations Intelligence Agent; Large Language Model; Kubernetes; ReAct; Tool Calling

---

## 第一章 绪论

### 1.1 研究背景

#### 1.1.1 智算平台的兴起与发展

随着人工智能技术的快速发展，以 GPU/AI 芯片为核心算力的智能计算平台（简称智算平台）已成为支撑 AI 研究与产业应用的关键基础设施。根据国家信息中心发布的《智能计算中心创新发展指南》，智算平台是以人工智能训练与推理任务为核心的计算基础设施，其建设和运营对国家人工智能产业发展具有重要战略意义[1]。

从全球范围来看，智算平台正处于快速扩张期。截至 2024 年底，中国算力总规模达到 280 EFLOPS，其中智能算力约 90 EFLOPS，占比超过 30%。IDC 预测，2025 年中国智能算力将达到 1,037.3 EFLOPS，同比增长超过 40%[2]。在"东数西算"工程的推动下，全国八大国家枢纽节点已集聚 60% 以上的新增算力，智算规模达到 62 万 Pflops[3]。

在高校领域，智算平台同样得到了广泛部署。从各大高校的传统 HPC 集群向 AI 计算转型，到专门建设面向深度学习的 GPU 集群，高校智算平台已成为支撑科学研究、人才培养和学科建设的重要基础设施。这些平台通常采用 Kubernetes 容器编排技术作为资源管理基础，集成 Volcano 等批处理调度框架、Harbor 镜像仓库、Prometheus 监控系统等云原生组件，形成了完整的 AI 开发与训练环境。

#### 1.1.2 智算平台运维面临的挑战

智算平台的运维复杂度远高于传统 IT 基础设施，主要体现在以下几个方面：

**（1）多层架构的因果耦合**。智算平台的运行环境可按五层模型描述：硬件/驱动层（GPU、InfiniBand、网卡驱动）、集群编排层（Kubernetes 控制面、调度器）、平台服务层（监控、镜像仓库、存储）、作业运行时层（容器网络、分布式通信）和用户负载层（训练脚本、Jupyter 会话）。各层之间存在密切的因果耦合——底层的 RDMA 驱动故障可能导致上层分布式训练静默卡住，中间层的 Prometheus 存储耗尽可能导致监控数据缺失进而使自动作业释放机制失效。这种跨层因果传播是智算平台运维区别于通用微服务运维的关键特征。

**（2）故障模式的多样性**。以高校智算平台为例，作业失败的原因涵盖应用程序错误（54%）、内存溢出 OOM（27%）、未知错误（6%）、运行环境缺失（4%）等多种类型，每种故障的诊断路径和所需证据各不相同。管理员需要同时关注节点硬件健康（GPU Xid 错误、PCIe 链路状态）、调度器状态（Volcano Queue 配额、Gang Scheduling）、网络连通性（NCCL 超时、RDMA 死锁）等多个维度。

**（3）运维人才短缺**。智算平台运维人员需要同时具备 Kubernetes 集群管理、GPU 硬件诊断、分布式训练框架调优、网络排查等多领域专业知识。在高校场景下，平台的用户群体以师生为主，通常不具备 Kubernetes 运维经验，对容器退出码、Pod 事件、资源配额等概念缺乏理解，导致大量本可自助解决的问题需要管理员人工介入。

**（4）GPU 资源利用率低下**。根据行业统计，智算中心平均利用率不足 45%[4]。在高校场景中，GPU 低利用率告警往往占据告警总量的 56% 以上，大量已分配但实际空闲的 GPU 资源无法被其他用户使用，造成了严重的资源浪费。

#### 1.1.3 智能运维（AIOps）的发展现状

AIOps（Artificial Intelligence for IT Operations）的概念由 Gartner 于 2017 年首次提出，指利用人工智能、机器学习和大数据分析来自动化和增强 IT 运维管理。其核心能力包括异常检测、根因分析和自动修复三个方面[5]。

近年来，大语言模型（LLM）的突破性进展为 AIOps 带来了新的技术路径。2023 年以来，基于 LLM 的运维智能体成为学术界和工业界的研究热点。微软研究院在 FSE 2024 发表的工作表明，基于 ReAct 模式的 LLM Agent 在配备检索工具后，在生产环境事件根因分析的事实准确性上大幅优于传统方法[6]。IBM 与 UIUC 合作开发的 STRATUS 多智能体系统在 NeurIPS 2025 发表，提出了"事务性无退化"安全规范，在 AIOpsLab 基准上故障修复成功率比现有最优方法高出 1.5 倍[7]。清华大学裴丹教授团队在 KDD 2024、FSE 2024、WWW 2024 等顶级会议上持续发表微服务系统根因定位、异常检测等方面的研究成果[8]。

在开源社区，面向 Kubernetes 运维的 AI 工具也在快速发展。K8sGPT 项目将 SRE 经验编码为分析器，结合 LLM 后端实现集群诊断[9]。Kagent 作为首个开源 Kubernetes Agentic AI 框架，已在 2025 年 KubeCon Europe 上宣布贡献给 CNCF[10]。这些项目验证了 LLM 驱动的运维智能体在 Kubernetes 场景中的可行性。

然而，现有 AIOps 研究和工具在智算平台场景中仍存在明显的覆盖缺口。AIOpsLab[11]、ITBench[12] 等评测基准面向通用微服务故障注入，未覆盖智算平台特有的 GPU 硬件诊断、分布式训练运行时故障、批作业调度器分析等场景。K8sGPT 等工具虽然面向 Kubernetes，但缺乏对智算平台业务层面（如作业生命周期管理、训练资源优化）的深度集成。

### 1.2 研究目标与范围界定

#### 1.2.1 研究目标

本课题面向智算平台运维场景，设计并实现一个基于大语言模型的运维智能体系统，使其能够：

1. **理解用户自然语言描述的运维问题**，结合平台上下文自动选择排查路径；
2. **调用平台工具链采集多源证据**（Kubernetes 资源状态、Prometheus 监控指标、容器日志、集群事件），进行跨层因果推理；
3. **生成结构化诊断报告和可执行的操作建议**，降低用户排查门槛；
4. **在安全约束下执行变更操作**（节点隔离、作业停止、Pod 重启等），通过确认机制保障操作可控性。

#### 1.2.2 范围界定

本课题聚焦**运维诊断与辅助操作**，明确以下边界：

| 在范围内 | 不在范围内 |
|---------|---------|
| 作业失败根因分析 | 自动代码修复 |
| 资源排队原因解释 | 模型训练调优 |
| 集群健康巡检与报告 | 数据集清洗 |
| 节点故障隔离操作 | 平台功能开发 |
| 分布式训练网络诊断 | 安全入侵检测 |
| GPU 空闲资源检测 | 硬件物理维修 |

### 1.3 论文组织结构

本文共分为七章。第一章阐述研究背景和目标；第二章介绍相关技术与研究现状；第三章进行需求分析；第四章描述系统总体设计；第五章详述关键模块的实现；第六章展示系统测试与运行效果；第七章总结全文并展望未来工作。

---

## 第二章 相关技术与研究现状

### 2.1 大语言模型与 Function Calling

大语言模型（Large Language Model, LLM）是基于 Transformer 架构的深度学习模型，通过在大规模文本语料上进行预训练，具备了强大的自然语言理解与生成能力。以 GPT-4、Claude、Qwen、DeepSeek 等为代表的 LLM，在代码理解、逻辑推理、知识问答等任务上展现出接近人类专家的水平[13]。

Function Calling（工具调用）是 LLM 从纯文本生成向外部工具交互演进的关键技术。其工作机制如下：

```
┌────────────────────────────────────────────────────┐
│                Function Calling 工作流程              │
├────────────────────────────────────────────────────┤
│                                                     │
│  ① 用户自然语言输入 + 可用工具定义 (JSON Schema)      │
│       │                                             │
│       ▼                                             │
│  ② LLM 分析意图，选择工具，生成结构化参数              │
│       │                                             │
│       ▼                                             │
│  ③ 外部运行时执行工具调用，获取结果                    │
│       │                                             │
│       ▼                                             │
│  ④ 结果回传 LLM，继续推理或生成最终回答               │
│                                                     │
└────────────────────────────────────────────────────┘
```

**图 2-1 Function Calling 工作流程**

LLM 本身并不执行函数，而是生成结构化的调用意图（工具名称和参数），由外部运行时负责实际执行。这种分离设计使得 LLM 可以与任意外部系统集成，同时保持安全边界的清晰。2024 年 11 月，Anthropic 提出的 Model Context Protocol（MCP）进一步推动了工具接口的标准化[14]。

### 2.2 ReAct 推理模式

ReAct（Reasoning + Acting）是 Yao 等人在 ICLR 2023 提出的 LLM Agent 推理范式[15]，累计引用超过 5,250 次，是 LLM Agent 领域被引最多的论文之一。ReAct 的核心思想是将推理（Reasoning）与行动（Acting）交替进行：

```
┌──────────────────────────────────────────────┐
│            ReAct 推理-行动-观察循环              │
├──────────────────────────────────────────────┤
│                                               │
│  Thought: 分析目标和历史，制定下一步计划         │
│     │                                         │
│     ▼                                         │
│  Action: 选择并调用工具                        │
│     │       (K8s 查询 / PromQL / 日志读取)     │
│     ▼                                         │
│  Observation: 接收工具返回结果                  │
│     │                                         │
│     ▼                                         │
│  (循环重复，直到问题解决或达到迭代上限)           │
│     │                                         │
│     ▼                                         │
│  Final Answer: 生成诊断报告或操作建议           │
│                                               │
└──────────────────────────────────────────────┘
```

**图 2-2 ReAct 推理循环**

相比于纯链式思维（Chain-of-Thought）推理，ReAct 具有两个关键优势：（1）通过外部工具调用 grounding 推理过程，有效减少 LLM 幻觉；（2）可以动态收集未知信息并自我纠错。在 HotpotQA 和 Fever 基准测试中，ReAct 克服了单纯 CoT 的幻觉和错误传播问题；在 ALFWorld 和 WebShop 基准中，分别比强化学习方法高出 34% 和 10% 的成功率[15]。

在运维场景中，ReAct 模式特别适用于故障诊断——智能体可以先查询 Pod 状态，根据观察到的退出码决定下一步查日志还是查指标，再根据日志内容决定是否需要查节点硬件状态，形成动态的证据收集链路。

### 2.3 Kubernetes 容器编排与调度

Kubernetes 是当前最主流的容器编排平台，提供自动化部署、扩缩容、负载均衡和故障恢复等能力。智算平台通常基于 Kubernetes 构建，并集成以下关键组件：

| 组件 | 功能 | 在智算平台中的作用 |
|------|------|---------------|
| Kubernetes | 容器编排 | 管理计算资源、Pod 调度 |
| Volcano | 批处理调度 | GPU 资源调度、Gang Scheduling |
| Prometheus | 监控采集 | GPU 利用率、节点负载、作业指标 |
| Harbor | 镜像仓库 | 训练镜像存储与分发 |
| NFS/CephFS | 分布式存储 | 数据集和模型存储 |
| NVIDIA GPU Operator | GPU 管理 | GPU 驱动、设备插件 |

**表 2-1 智算平台典型技术栈**

Kubernetes 运维面临的核心挑战包括：配置错误导致约 80% 的事件[16]、运维复杂性使 75% 的组织报告集群运行问题[17]、82% 以上工作负载存在资源过度配置[18]。这些问题在智算平台中尤为突出，因为 GPU 资源的高成本使得资源浪费的代价远高于通用计算场景。

### 2.4 AIOps 领域研究现状

ACM Computing Surveys 在 2025 年发表了 LLM 时代 AIOps 的系统性综述，分析了 2020-2024 年发表的 183 篇相关研究论文[19]。该综述指出，传统 ML/DL 方法在 AIOps 中面临复杂特征工程、非结构化数据理解能力有限、跨平台泛化性差等核心挑战，而 LLM 的涌现能力为这些问题提供了新的解决思路。

在根因分析领域，微软 FSE 2024 的工作表明 ReAct Agent 配备检索工具后在事实准确性上显著优于传统方法[6]。WWW 2025 的 Flow-of-Action 工作创新性地集成 SOP 知识库和历史事件知识，解决了传统深度学习方法新场景适应性差和可解释性不足的问题[20]。

在安全性方面，IBM 的 STRATUS 系统提出"事务性无退化（Transactional No-Regression, TNR）"安全规范，确保只有可逆的、不会破坏现有功能的变更才能执行[7]。这一理念对智算平台的运维智能体设计具有重要参考价值。

---

## 第三章 需求分析

### 3.1 Crater 智算平台概述

本课题依托的实验平台 Crater 是一个面向高校的智能计算平台，提供多用户、多类型训练任务的统一承载能力。平台的核心规模参数如表 3-1 所示。

| 参数 | 数值 |
|------|------|
| 计算节点 | 59 个 |
| GPU 总量 | 220 张（A100, V100, T4） |
| 注册用户 | 464 名 |
| 累计作业 | 9.4 万个 |
| 运行时间 | 16 个月（2024.12–2026.04） |

**表 3-1 Crater 平台规模参数**

Crater 平台的功能覆盖 AI 研究的完整工作流：

```
┌───────────────────────────────────────────────────────┐
│                  Crater 平台功能架构                     │
├───────────────┬───────────────┬───────────────────────┤
│   用户服务层   │   资源管理层   │      运维管理层         │
├───────────────┼───────────────┼───────────────────────┤
│ Jupyter 交互  │ GPU 配额管理   │ 集群监控告警           │
│ 训练作业提交  │ Volcano 调度   │ 节点健康检查           │
│ 分布式训练    │ 存储卷管理     │ 作业故障排查           │
│ 镜像构建     │ 网络策略管理    │ 资源利用率分析         │
│ 数据集管理   │ 多租户隔离     │ 容量规划              │
└───────────────┴───────────────┴───────────────────────┘
```

**图 3-1 Crater 平台功能架构**

平台运行环境可按五层模型描述：

```
┌───────────────────────────────────────────────────────────────┐
│ L5 用户负载层   PyTorch DDP · Jupyter Lab · 用户训练脚本        │
├───────────────────────────────────────────────────────────────┤
│ L4 作业运行时层  NCCL/RDMA 通信 · PVC 存储挂载 · Service 网络   │
├───────────────────────────────────────────────────────────────┤
│ L3 平台服务层   Prometheus · Harbor · NFS · MetalLB · Ingress  │
├───────────────────────────────────────────────────────────────┤
│ L2 集群编排层   kube-scheduler · Volcano 调度器 · CoreDNS       │
├───────────────────────────────────────────────────────────────┤
│ L1 硬件/驱动层  NVIDIA GPU · InfiniBand · GPU Operator · mlx5   │
└───────────────────────────────────────────────────────────────┘
         ▲                    ▲                    ▲
         │     跨层因果传播     │                    │
         └────────────────────┘                    │
              故障沿层级向上传播                      │
```

**图 3-2 智算平台五层运行模型**

### 3.2 运维痛点量化分析

基于 Crater 平台 16 个月的全量生产数据，对运维痛点进行了系统性的量化分析。

#### 3.2.1 作业失败模式分布

在 8,957 条失败记录中，按退出码分类的失败模式分布如表 3-2 所示。

| 失败类型 | 占比 | 退出码 | 诊断难度 | Agent 可辅助程度 |
|---------|------|--------|---------|---------------|
| 应用程序错误 | 54% | exit 1 | 高（需理解日志语义） | 中等 |
| 内存溢出 OOM | 27% | exit 137 | 低（结构化证据） | 高 |
| 未知错误 | 6% | exit 255 | 高（需跨层排查） | 中等 |
| 命令不存在 | 4% | exit 127 | 低 | 高 |
| 其他 | 9% | 多种 | 中 | 中等 |

**表 3-2 作业失败模式分布**

核心发现：约 40% 的失败场景具有结构化诊断证据（退出码、Kubernetes 事件），适合由智能体基于多源证据辅助用户理解和处理。剩余 60%（以应用程序错误为主）的诊断深度受限于容器日志的可读性，但智能体仍可提供日志要点提取和方向性建议。

#### 3.2.2 告警统计与资源浪费

平台累计 6.8 万条告警中，56% 与 GPU 低利用率相关。这表明大量 GPU 资源已分配给用户但未被有效利用——用户申请了 GPU 资源后，可能因为调试代码、等待数据或忘记释放等原因导致 GPU 长时间空闲。

#### 3.2.3 运维工单分析

通过对管理员日常运维工作的观察和记录，归纳出以下高频运维场景：

| 场景 | 频次 | 典型耗时 | 所需技能 |
|------|------|---------|---------|
| 作业失败原因查询 | 高（日均 5-10 次） | 5-30 分钟 | K8s 基础 + 日志分析 |
| "为什么排不上队" | 高（日均 3-5 次） | 10-20 分钟 | 调度器 + 配额知识 |
| 节点故障隔离 | 中（周均 1-2 次） | 30-60 分钟 | K8s 运维 + 硬件诊断 |
| GPU 利用率巡检 | 中（周均 1 次） | 60+ 分钟 | Prometheus + 脚本 |
| 分布式训练卡住 | 低（月均 2-3 次） | 60+ 分钟 | NCCL/RDMA + 网络诊断 |

**表 3-3 高频运维场景统计**

### 3.3 功能需求

基于上述痛点分析，提炼出以下功能需求：

**FR1 自然语言交互**：用户和管理员可以通过自然语言描述运维问题，系统自动理解意图并规划排查路径。

**FR2 多源证据采集**：系统能够自动采集 Kubernetes 资源状态、容器日志、Prometheus 监控指标、集群事件等多源数据。

**FR3 作业故障诊断**：支持单机训练 OOM、分布式训练 NCCL 超时、Jupyter 不可达、镜像拉取失败等典型作业故障的根因分析。

**FR4 集群健康巡检**：支持 GPU 空闲检测、僵尸作业识别、节点健康评估、存储容量分析等巡检功能。

**FR5 安全变更操作**：支持节点隔离（cordon/drain）、作业停止、Pod 删除等运维操作，所有写操作需经用户明确确认。

**FR6 会话管理与审计**：支持多轮对话会话管理，记录完整的工具调用审计日志。

### 3.4 非功能需求

**NFR1 安全性**：写操作必须经过用户确认；管理员工具对普通用户不可见；工具执行在后端进行权限校验。

**NFR2 可追溯性**：每次工具调用的参数、结果、执行时间均需持久化存储，支持事后审计。

**NFR3 响应及时性**：简单查询应在 10 秒内给出回答；复杂诊断不超过 60 秒。

**NFR4 可扩展性**：工具体系支持增量扩展，新增工具无需修改核心推理引擎。

---

## 第四章 系统设计

### 4.1 总体架构设计

系统采用**前端-后端-Agent 三层架构**，各层职责清晰分离：

```
┌──────────────────────────────────────────────────────────────┐
│                    前端层 (React + TypeScript)                 │
│                                                               │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ 对话窗口  │ │ 工具时间线│ │ 确认卡片  │ │ 会话管理 · 巡检  │ │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘ │
└──────────────────────┬───────────────────────────────────────┘
                       │ SSE (Server-Sent Events) 流式通信
                       │ Bearer Token 身份认证
┌──────────────────────▼───────────────────────────────────────┐
│                    Go 后端层 (Gin + GORM)                      │
│                                                               │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ 身份鉴权  │ │ 上下文构建│ │ 工具执行  │ │ 确认流 · 审计    │ │
│  │ JWT 校验  │ │ 角色/页面 │ │ K8s/DB   │ │ 会话持久化       │ │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘ │
└──────────────────────┬───────────────────────────────────────┘
                       │ HTTP (内部 Token 认证)
                       │ AgentTurnRequest / AgentToolResponse
┌──────────────────────▼───────────────────────────────────────┐
│                 Python Agent 层 (FastAPI + LangGraph)          │
│                                                               │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ LLM 推理  │ │ 工具编排  │ │ 状态管理  │ │ 本地工具执行     │ │
│  │ ReAct 循环│ │ 角色分派  │ │ 上下文截断│ │ kubectl/PromQL   │ │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

**图 4-1 系统总体架构**

三层架构的通信流程如下：

```
用户消息 ──── ① HTTP POST ────▶ Go 后端
                                  │
                           ② 身份校验 + 上下文构建
                                  │
                           ③ HTTP POST ────▶ Python Agent
                                              │
                                    ④ ReAct 推理循环
                                    ┌─────────┤
                                    │  Thought │
                                    │     │    │
                                    │  Action ─┼──▶ 工具执行请求
                                    │     │    │       │
                                    │     │    │   ⑤ HTTP POST ──▶ Go 后端
                                    │     │    │       │            (执行工具)
                                    │     │    │   ⑥ 返回结果 ◀────┘
                                    │     │    │
                                    │  Observation
                                    │     │    │
                                    └─────┘    │
                                              │
                                    ⑦ 最终回答
                                              │
用户接收 ◀── ⑧ SSE 事件流 ◀──────── Go 后端 ◀─┘
```

**图 4-2 三层通信流程**

**设计决策说明**：采用三层分离而非在后端内嵌 LLM 推理，主要基于以下考虑：（1）Python 生态在 LLM/AI 领域拥有更成熟的工具链（LangChain、LangGraph）；（2）Agent 层与后端层可独立部署和扩缩容；（3）Go 后端专注于身份认证、权限校验和安全执行，形成清晰的信任边界。

### 4.2 工具体系设计

#### 4.2.1 分层可组合工具架构

本系统的工具体系采用**分层可组合**的设计理念。不同于为每个故障场景开发专用工具的方式，系统将工具按抽象层级组织为三层，智能体的诊断能力来自基础工具的灵活组合：

```
┌──────────────────────────────────────────────────────────────┐
│  L3 写操作层（14 工具）                                        │
│  需要用户确认，带风险评估                                       │
│  cordon_node · drain_node · delete_pod · restart_workload     │
│  stop_job · resubmit_job · batch_stop_jobs · ...              │
├──────────────────────────────────────────────────────────────┤
│  L2 领域组合层（36 工具）                                      │
│  对 L1 工具的预封装，减少 Agent Token 消耗                     │
│  get_job_detail · diagnose_job · detect_idle_jobs              │
│  get_cluster_health_report · get_volcano_queue_state · ...     │
├──────────────────────────────────────────────────────────────┤
│  L1 基础积木层（25 工具）                                      │
│  通用可复用，跨实体跨场景                                       │
│  K8s: list_pods · get_events · get_pod_logs · describe         │
│  监控: prometheus_query（支持任意 PromQL）                      │
│  网络: get_service · get_endpoints · get_ingress               │
│  硬件: get_node_kernel_diagnostics · get_rdma_interface        │
└──────────────────────────────────────────────────────────────┘
```

**图 4-3 三层工具架构**

**关键设计决策**：L1 层工具是通用的 Kubernetes 资源查询，不绑定特定实体。同一个 `k8s_list_pods` 工具既可以查训练 Pod（namespace=crater-jobs），也可以查构建 Pod（namespace=crater-images），还可以查监控组件（namespace=monitoring）。Agent 的智能在于知道何时查哪个 namespace、用什么 label selector。

#### 4.2.2 工具覆盖矩阵

表 4-1 展示了工具对平台各基建组件的覆盖情况：

| 基建组件 | 覆盖工具 | 诊断能力 |
|---------|---------|---------|
| Kubernetes 控制面 | k8s_list_nodes, k8s_list_pods, k8s_get_events, k8s_describe_resource, k8s_get_pod_logs | 节点/Pod/事件/日志完整查询 |
| 网络层 | k8s_get_service, k8s_get_endpoints, k8s_get_ingress, k8s_get_networkpolicy | Service→Endpoints→Ingress 全链路 |
| Volcano 调度器 | get_volcano_queue_state, analyze_queue_status | 队列容量/公平性/Pending 原因 |
| Prometheus 监控 | prometheus_query（instant/range 双模式） | 任意 PromQL：GPU/CPU/内存/网络 |
| Harbor 镜像仓库 | harbor_check, aggregate_image_pull_errors | 仓库健康 + 镜像拉取错误聚合 |
| 存储系统 | list_storage_pvcs, get_pvc_detail, get_pvc_events | PVC 容量/绑定/事件排查 |
| GPU/RDMA 硬件 | get_node_kernel_diagnostics, get_rdma_interface_status | dmesg/D状态进程/IB端口 |
| 分布式训练 | get_ddp_rank_mapping, diagnose_distributed_job_network | rank→Pod→Node 映射 + NCCL 分析 |

**表 4-1 工具覆盖矩阵**

#### 4.2.3 工具复用性设计

分层可组合架构的核心价值在于工具的复用性。表 4-2 展示了同一组 L1 工具如何覆盖不同平台实体的故障排查：

| 平台实体 | 排查路径 | 复用的 L1 工具 |
|---------|---------|-------------|
| 训练作业失败 | 查 Pod 日志 → 查事件 → 查指标 | k8s_get_pod_logs + k8s_get_events + prometheus_query |
| 镜像构建失败 | 查构建 Pod 日志 → 查网络策略 | **同上**（namespace 不同） |
| 数据集挂载失败 | 查 PVC 事件 → 查 NFS Pod → 查日志 | k8s_get_events + k8s_list_pods + k8s_get_pod_logs |
| Jupyter 不可达 | 查 Service → Endpoints → Ingress | k8s_get_service + k8s_get_endpoints + k8s_get_ingress |
| 分布式训练卡住 | 查 rank 映射 → 查节点 dmesg → 查 IB | get_ddp_rank_mapping + get_node_kernel_diagnostics |

**表 4-2 工具复用性分析**

#### 4.2.4 工具执行路由

系统支持两条工具执行路径，由 CompositeToolExecutor 根据配置自动路由：

```
┌────────────────┐      ┌────────────────────────────┐
│  Agent 工具调用  │─────▶│ CompositeToolExecutor 路由  │
└────────────────┘      └──────────┬─────────────────┘
                                   │
                    ┌──────────────┼──────────────┐
                    ▼                             ▼
        ┌──────────────────┐          ┌──────────────────┐
        │ 本地执行 (Python) │          │ 后端执行 (Go)     │
        │                  │          │                  │
        │ · kubectl 查询   │          │ · 作业诊断        │
        │ · PromQL 查询    │          │ · 写操作(需确认)   │
        │ · Harbor 检查    │          │ · 数据库查询       │
        │ · 沙箱读写       │          │ · 权限校验        │
        └──────────────────┘          └──────────────────┘
```

**图 4-4 工具执行路由**

### 4.3 智能体编排设计

#### 4.3.1 单智能体编排（ReAct 基线）

基于 LangGraph StateGraph 实现 ReAct 推理-行动-观察循环。状态定义如下：

```
状态 S = {
    messages:             对话消息列表
    context:              {actor, page, capabilities, orchestration}
    tool_call_count:      工具调用计数
    pending_confirmation: 待确认操作
    trace:                执行追踪
}
```

状态转移流程：

```
            ┌──────────────────┐
            │    用户输入消息    │
            └────────┬─────────┘
                     ▼
            ┌──────────────────┐
      ┌────▶│    LLM 推理节点   │◀────────────┐
      │     └────────┬─────────┘              │
      │              │                         │
      │    ┌─────────▼──────────┐             │
      │    │ 是否需要工具调用？    │             │
      │    └─────────┬──────────┘             │
      │         是   │      否                │
      │              ▼       ▼                │
      │    ┌─────────────┐  ┌──────────┐     │
      │    │  工具执行节点  │  │ 最终回答  │     │
      │    └──────┬──────┘  └──────────┘     │
      │           │                           │
      │           │  是否需要确认？              │
      │           ├──── 是 ──▶ 暂停等待确认     │
      │           │                           │
      │           └──── 否 ──── Observation ──┘
      │
      └── tool_call_count < MAX_CALLS
```

**图 4-5 单智能体状态转移图**

**终止条件**：到达最终回答，或工具调用次数达到上限（默认 10 次）。

**上下文管理策略**：对话历史按 token 数截断（默认 4000 tokens）；工具返回结果截断为 1400 字符以防止上下文膨胀；当上下文超过 LLM 限制时自动触发消息压缩。

#### 4.3.2 多智能体编排（Plan-and-Execute + 五角色）

针对单智能体在复杂诊断场景中工具选择准确率下降的问题，设计五角色串行编排架构：

```
┌──────────────────────────────────────────────────────┐
│                                                       │
│  用户消息 ──▶ 协调者 (Coordinator)                     │
│                  │                                    │
│           ┌──────┼──────────────┐                     │
│           ▼      ▼              ▼                     │
│       [查询]  [诊断]        [写操作]                   │
│           │      │              │                     │
│           │      ▼              ▼                     │
│           │  规划者 (Planner)  执行者 (Executor)       │
│           │      │              │                     │
│           │      ▼              │                     │
│           │  探索者 (Explorer)  │                     │
│           │      │              │                     │
│           │      ▼              │                     │
│           │  验证者 (Verifier)  │                     │
│           │      │              │                     │
│           ▼      ▼              ▼                     │
│           ◀──── 最终回答 ────▶                        │
│                                                       │
└──────────────────────────────────────────────────────┘
```

**图 4-6 五角色多智能体编排**

各角色的职责与工具权限如表 4-3 所示：

| 角色 | 职责 | 运维场景作用 | 工具权限 |
|------|------|-----------|---------|
| 协调者 | 意图路由、流程控制 | 区分查询/诊断/写操作 | 无工具 |
| 规划者 | 分解为调查计划 | "OOM→查退出码→查日志→查指标→建议" | 无工具 |
| 探索者 | 执行只读工具，整理证据 | 依次调用 get_job_detail, get_job_logs 等 | 只读工具 |
| 执行者 | 生成写操作提案 | "建议 cordon 故障节点"→确认卡片 | 写工具(需确认) |
| 验证者 | 质疑结论充分性 | "退出码 137=OOM，但内存指标只有 70%？" | 无工具 |

**表 4-3 多智能体角色职责**

**运行时保护**：按场景配置差异化参数，防止 Agent 过度消耗资源：

| 场景类型 | 最大迭代 | 只读工具预算 | 证据上限 | 验证次数 |
|---------|---------|-----------|---------|---------|
| 查询 | 4 | 2 | 4 | 1 |
| 诊断 | 6 | 6 | 8 | 1 |
| 运维分析 | 5 | 2 | 6 | 1 |
| 写操作 | 4 | 2 | 4 | 1 |

**表 4-4 场景差异化参数配置**

### 4.4 安全控制设计

#### 4.4.1 基于角色的工具访问控制

工具集按角色进行严格划分：

```
┌──────────────────────────────────────────────────┐
│              工具访问控制矩阵                       │
├────────────────────┬─────────────┬───────────────┤
│     工具类别        │  普通用户    │   管理员       │
├────────────────────┼─────────────┼───────────────┤
│ 作业诊断(自有作业)  │    ✓        │     ✓         │
│ 资源查询(自有)      │    ✓        │     ✓         │
│ 集群概览(只读)      │    ✓        │     ✓         │
│ K8s 资源查询        │    ✗        │     ✓         │
│ 节点诊断            │    ✗        │     ✓         │
│ 存储管理            │    ✗        │     ✓         │
│ 网络诊断            │    ✗        │     ✓         │
│ 审计报告            │    ✗        │     ✓         │
│ 写操作(自有作业)    │  ✓(需确认)   │   ✓(需确认)   │
│ 写操作(节点/批量)   │    ✗        │   ✓(需确认)   │
└────────────────────┴─────────────┴───────────────┘
```

**图 4-7 工具访问控制矩阵**

#### 4.4.2 写操作确认流程

所有写操作遵循"提案→确认→执行→审计"四步流程：

```
Agent 推理       Go 后端          前端            用户
  │                │               │               │
  │  tool_call     │               │               │
  │──(cordon_node)─▶               │               │
  │                │               │               │
  │  confirmation_ │               │               │
  │  required      │ SSE: confirm  │               │
  │◀───────────────│──────────────▶│  确认卡片      │
  │                │               │──────────────▶│
  │   [暂停等待]    │               │               │
  │                │               │  用户确认/拒绝  │
  │                │  POST confirm │◀──────────────│
  │                │◀──────────────│               │
  │                │               │               │
  │                │  执行 K8s 操作 │               │
  │                │  记录审计日志   │               │
  │                │               │               │
  │  resume        │               │               │
  │◀───────────────│               │               │
  │                │               │               │
  │  继续推理...    │               │               │
```

**图 4-8 写操作确认时序图**

### 4.5 数据模型设计

系统采用 PostgreSQL 存储会话、消息、工具调用和事件数据。核心实体关系如下：

```
┌──────────────┐     ┌──────────────┐     ┌──────────────────┐
│ AgentSession │     │ AgentMessage │     │  AgentToolCall   │
├──────────────┤     ├──────────────┤     ├──────────────────┤
│ session_id   │──┐  │ session_id   │     │ session_id       │
│ user_id      │  │  │ role         │     │ turn_id          │
│ account_id   │  │  │ content      │     │ tool_name        │
│ title        │  └─▶│ tool_calls   │     │ tool_args (JSON) │
│ message_count│     │ created_at   │     │ tool_result(JSON)│
│ pinned_at    │     └──────────────┘     │ user_confirmed   │
│ created_at   │                          │ execution_backend│
└──────────────┘                          │ latency_ms       │
       │                                  └──────────────────┘
       │                                           ▲
       │          ┌──────────────┐                 │
       └─────────▶│  AgentTurn   │─────────────────┘
                  ├──────────────┤
                  │ turn_id      │     ┌──────────────────┐
                  │ session_id   │     │ AgentRunEvent    │
                  │ orchestration│     ├──────────────────┤
                  │ status       │     │ turn_id          │
                  │ started_at   │────▶│ agent_id         │
                  │ ended_at     │     │ agent_role       │
                  └──────────────┘     │ event_type       │
                                       │ content          │
                                       │ sequence         │
                                       └──────────────────┘
```

**图 4-9 数据模型 ER 图**

各实体的设计目的：

- **AgentSession**：对话容器，关联用户身份和页面上下文，支持会话置顶和软删除。
- **AgentMessage**：用户和助手消息，支持 user、assistant、tool 三种角色。
- **AgentTurn**：一次完整的推理执行单元，记录编排模式和完成状态。
- **AgentToolCall**：工具调用审计日志，记录参数、结果、确认状态、执行后端和延迟。
- **AgentRunEvent**：语义级事件序列，支持多智能体流程的细粒度追踪。

### 4.6 上下文感知模型

每次对话注入的上下文 C = (actor, page, capabilities) 使智能体能够感知当前操作环境：

| 维度 | 内容 | 影响 |
|------|------|------|
| actor | 用户 ID、角色（普通/管理员）、所属账户 | 决定可用工具集 |
| page | 当前页面路由、关联作业名/节点名 | 自动聚焦当前实体 |
| capabilities | 当前角色可用的工具列表 | RBAC 过滤 |

**表 4-5 上下文感知维度**

例如，当用户在作业详情页发起对话时，page 上下文自动包含当前作业名，Agent 无需追问即可直接针对该作业进行诊断。

---

## 第五章 系统实现

### 5.1 技术选型

| 层级 | 技术栈 | 选择理由 |
|------|--------|---------|
| 前端 | React 19 + TypeScript + Tailwind CSS | 组件化 UI、类型安全、样式效率 |
| 状态管理 | Jotai + TanStack Query v5 | 原子化状态 + 服务端状态缓存 |
| Go 后端 | Gin + GORM + client-go | 高性能 HTTP + ORM + K8s 原生客户端 |
| Agent | FastAPI + LangGraph + Pydantic | 异步 SSE + 状态图编排 + 类型校验 |
| LLM | Qwen / DeepSeek（OpenAI 兼容接口） | 国产模型、性价比高、私有化部署 |
| 数据库 | PostgreSQL (CloudNativePG) | 成熟稳定、JSONB 支持 |
| 部署 | Helm v3 + Kubernetes | 与平台自身一致的部署方式 |

**表 5-1 技术选型**

### 5.2 Agent 推理引擎实现

#### 5.2.1 LLM 客户端管理

系统通过配置文件支持多 LLM 后端，不同角色可使用不同模型：

```json
{
  "default": {
    "base_url": "https://api.deepseek.com/v1",
    "model": "deepseek-chat",
    "api_key_env": "DEEPSEEK_API_KEY"
  },
  "planner": {
    "base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "model": "qwen3.5-122b",
    "api_key_env": "DASHSCOPE_API_KEY"
  }
}
```

ModelClientFactory 支持两种创建模式：简单模式 `create("default")` 和角色感知模式 `create(purpose="planner", orchestration_mode="multi_agent")`，后者允许多智能体编排中不同角色使用不同能力层级的模型。

#### 5.2.2 ReAct 循环实现

基于 LangGraph StateGraph 构建 ReAct 状态机。核心实现逻辑为：

1. **状态初始化**：将用户消息、上下文（actor、page、history、capabilities）和工具调用计数封装为 `CraterAgentState`。

2. **LLM 调用**：将启用的工具绑定到 LLM，发送对话消息获取响应。系统兼容 Qwen 的 thinking 模式——当 content 为空但 reasoning_content 非空时，自动提取推理内容。

3. **工具执行**：通过 `CompositeToolExecutor` 统一调度。本地核心工具（kubectl、PromQL 等）在 Python 侧直接执行；业务工具和写操作通过 HTTP 回调 Go 后端执行。

4. **结果截断**：工具返回结果截断为 1400 字符，防止长日志或大量 Pod 列表导致的上下文膨胀。

5. **流式事件输出**：每个阶段（推理中、工具调用开始、工具调用完成、需要确认、最终回答）均通过 SSE 事件实时推送。

#### 5.2.3 多智能体编排实现

多智能体编排基于串行协作模型实现，而非并发 fan-out。MultiAgentTurnState 维护全局状态：

```
MultiAgentTurnState = {
    session_id, turn_id, user_message,
    actor, page_context, capabilities, history,
    route,               // 协调者的路由决策
    plan,                // 规划者的调查计划
    exploration,         // 探索者的证据收集
    execution,           // 执行者的操作结果
    verification,        // 验证者的质疑反馈
    final_answer,        // 最终回答
    evidence,            // 累积证据池
    tool_records,        // 工具调用记录
    pending_confirmation // 待确认操作
}
```

各角色依次执行，前一角色的输出作为后一角色的输入。事件流通过 `agent_run_events` 表持久化，每个事件携带 `agent_id`、`agent_role`、`event_type` 和递增的 `sequence` 编号，支持完整的执行流程回放。

### 5.3 Go 后端工具执行实现

#### 5.3.1 工具路由分发

Go 后端的 `ExecuteTool()` 方法作为核心分发入口，处理流程如下：

1. 验证 `X-Agent-Internal-Token` 请求头（Python-to-Go 内部认证）；
2. 通过 `session_id` 从数据库恢复用户身份（不信任 Python 传递的 user_id）；
3. 检查工具对当前角色的启用状态；
4. 根据 `tool_name` 路由到对应的处理函数。

这种基于会话恢复身份的设计确保了即使 Python Agent 层被攻破，也无法伪造用户身份执行越权操作。

#### 5.3.2 典型工具实现示例

以 **作业诊断工具 `diagnose_job`** 为例，其实现涉及多源数据采集与规则推理：

```
输入: job_name
  │
  ├──▶ 查询数据库获取作业基本信息
  │     (状态、退出码、创建时间、资源请求)
  │
  ├──▶ 查询 Kubernetes API 获取 Pod 状态
  │     (phase, containerStatuses, exitCode)
  │
  ├──▶ 查询 Kubernetes Events
  │     (FailedScheduling, FailedMount, OOMKilled)
  │
  ├──▶ 规则引擎分类
  │     exitCode=137 → OOM
  │     exitCode=1   → AppError
  │     exitCode=127 → CommandNotFound
  │     Pending 超时  → SchedulingFailure
  │
  └──▶ 返回结构化诊断结果
        {category, evidence[], suggestion}
```

以 **节点隔离工具 `cordon_node`** 为例，其写操作流程为：

```
输入: node_name
  │
  ├──▶ 权限校验 (仅管理员)
  │
  ├──▶ 查询节点当前状态
  │     (确认节点存在且未被 cordon)
  │
  ├──▶ 生成确认请求
  │     {confirm_id, action, description, risk_level: "high"}
  │
  │     [等待用户确认]
  │
  ├──▶ 确认通过后执行 kubectl cordon
  │
  ├──▶ 记录审计日志
  │
  └──▶ 返回执行结果
```

### 5.4 前端交互实现

#### 5.4.1 对话界面设计

前端对话界面支持多种内容类型的渲染：

| 内容类型 | 组件 | 展示形式 |
|---------|------|---------|
| 用户消息 | ChatBubble | 右对齐文本气泡 |
| 助手回答 | MarkdownRenderer | Markdown + 语法高亮 |
| 思考过程 | ThinkingIndicator | 动画加载指示器 |
| 工具调用 | ToolCallCard | 工具名 + 参数 + 结果折叠 |
| 确认请求 | ConfirmActionCard | 操作描述 + 风险等级 + 确认/拒绝按钮 |
| 多智能体流程 | AgentTimeline | 垂直时间线 + 角色标签 |

**表 5-2 前端内容类型与组件映射**

#### 5.4.2 SSE 事件处理

前端通过 EventSource 接收后端推送的 SSE 事件流。关键事件类型及其处理逻辑：

- `agent_run_started`：初始化新的对话轮次
- `thinking`：更新思考指示器，展示推理过程
- `tool_call_started`：创建工具调用卡片，显示 "执行中" 状态
- `tool_call_completed`：更新工具调用卡片，展示执行结果
- `tool_call_confirmation_required`：渲染确认卡片，等待用户操作
- `final_answer`：渲染最终 Markdown 回答
- `done`：标记轮次结束

#### 5.4.3 会话管理

前端支持完整的会话管理功能：

- **会话列表**：按最近使用排序，支持置顶和删除
- **会话持久化**：当前会话 ID 存储在 localStorage，刷新页面自动恢复
- **浮动入口**：FloatingAssistantButton 组件提供全局的对话入口，可在任意页面唤起

### 5.5 部署实现

系统通过 Helm v3 Chart 部署到 Kubernetes 集群，与 Crater 平台共享同一集群：

```
┌─────────────────────────────────────────────────┐
│               Kubernetes 集群                    │
│                                                  │
│  ┌───────────────┐  ┌───────────────┐           │
│  │ crater-backend│  │ crater-agent  │           │
│  │ (Go, 1 副本)  │  │ (Python, 1副本)│          │
│  └───────┬───────┘  └───────┬───────┘           │
│          │                  │                    │
│          ▼                  ▼                    │
│  ┌───────────────┐  ┌───────────────┐           │
│  │  PostgreSQL   │  │ LLM API       │           │
│  │ (CloudNativePG)│ │ (外部服务)     │           │
│  └───────────────┘  └───────────────┘           │
│                                                  │
│  ┌───────────────┐  ┌───────────────┐           │
│  │crater-frontend│  │  Ingress      │           │
│  │ (Nginx, 1副本) │  │ (nginx-ingress)│          │
│  └───────────────┘  └───────────────┘           │
└─────────────────────────────────────────────────┘
```

**图 5-1 部署架构**

---

## 第六章 系统测试与运行效果

### 6.1 测试环境

测试在 Crater 平台的实际生产环境中进行，环境参数如表 6-1 所示。

| 参数 | 配置 |
|------|------|
| 集群规模 | 59 节点、220 GPU |
| Kubernetes 版本 | v1.28 |
| 调度器 | Volcano v1.9 |
| 监控 | Prometheus + Grafana |
| LLM 后端 | DeepSeek-Chat / Qwen3.5-122B |
| Agent 运行环境 | Python 3.11 + FastAPI + LangGraph |

**表 6-1 测试环境配置**

### 6.2 测试场景设计

围绕四个研究问题设计 10 个运维场景：

| RQ | 研究问题 | 场景数 | 验证目标 |
|----|---------|--------|---------|
| RQ1 | Agent 能否诊断用户视角的作业故障？ | 3 | 基础诊断能力 |
| RQ2 | Agent 能否进行跨层因果推理？ | 3 | 智算特有差异化价值 |
| RQ3 | Agent 能否支撑主动运维？ | 2 | 活动多样性 |
| RQ4 | Agent 能否安全执行缓解动作？ | 2 | 安全操作闭环 |

**表 6-2 测试场景分类**

### 6.3 典型场景测试

#### 6.3.1 场景一：单机训练 OOMKilled 诊断

**用户输入**："我的作业 sg-train-bert 挂了，帮我看看什么原因。"

**Agent 工具调用链**：

```
Step 1: get_job_detail(job_name="sg-train-bert")
        → 状态: Failed, 退出码: 137

Step 2: get_job_events(job_name="sg-train-bert")
        → 事件: OOMKilled, container memory limit exceeded

Step 3: get_job_logs(job_name="sg-train-bert", tail=100)
        → 日志: RuntimeError: CUDA out of memory...
        → 尝试分配 4.2 GiB, 但仅剩 1.1 GiB 可用

Step 4: diagnose_job(job_name="sg-train-bert")
        → 诊断类别: OOM (GPU显存不足)
        → 证据: exit_code=137 + CUDA OOM 日志
```

**Agent 最终回答**：

> 您的作业 sg-train-bert 因 **GPU 显存不足（OOM）** 而失败。
>
> **诊断依据**：
> - 退出码 137 表示容器被 OOMKilled
> - 日志显示 CUDA 尝试分配 4.2 GiB 显存，但仅剩 1.1 GiB
>
> **建议措施**：
> 1. 减小 batch_size（当前可能过大）
> 2. 启用混合精度训练（`torch.cuda.amp`）减少显存占用
> 3. 使用 `gradient_checkpointing` 以时间换空间
> 4. 如需更多显存，可申请配备更大显存的 GPU（如 A100 80GB）

#### 6.3.2 场景二：分布式训练 NCCL 超时

**管理员输入**："分布式作业 sg-ddp-resnet 卡住超过 30 分钟了。"

**Agent 工具调用链**：

```
Step 1: get_ddp_rank_mapping(job_name="sg-ddp-resnet")
        → rank 0: pod-0 @ node-gpu-03
        → rank 1: pod-1 @ node-gpu-07

Step 2: diagnose_distributed_job_network(job_name="sg-ddp-resnet")
        → NCCL timeout detected between rank 0 and rank 1

Step 3: get_rdma_interface_status(node_name="node-gpu-07")
        → mlx5_0: port_state=DOWN, link_layer=InfiniBand

Step 4: get_node_kernel_diagnostics(node_name="node-gpu-07")
        → dmesg: mlx5_core: firmware error, PCI link degraded
```

**Agent 最终回答**：

> 作业 sg-ddp-resnet 卡住的根因是 **node-gpu-07 节点的 InfiniBand 网卡故障**。
>
> **因果链**：
> - node-gpu-07 的 mlx5_0 网卡端口状态为 DOWN
> - dmesg 显示固件错误和 PCI 链路降级
> - 导致 rank 1 (pod-1) 无法与 rank 0 (pod-0) 完成 NCCL 通信
> - 表现为分布式训练作业静默卡住
>
> **建议**：
> 1. 对 node-gpu-07 执行 `cordon` 隔离，防止新作业调度到该节点
> 2. 停止当前作业，重新提交时排除该节点
> 3. 联系硬件运维团队检查 InfiniBand 网卡

#### 6.3.3 场景三：GPU 空闲资源巡检

**管理员输入**："帮我巡检一下，看看有没有浪费 GPU 的作业。"

**Agent 工具调用链**：

```
Step 1: detect_idle_jobs()
        → 发现 12 个低利用率作业
        → 其中 5 个 GPU 利用率 < 5%（连续 2 小时）

Step 2: query_job_metrics(job_name="sg-jupyter-user42")
        → GPU 利用率: 0%（最近 6 小时）
        → GPU 显存占用: 200MB / 16GB

Step 3: get_admin_ops_report()
        → 集群 GPU 总量: 220 张
        → 已分配: 180 张
        → 低利用率(<10%): 38 张（占已分配 21%）
```

**Agent 最终回答**：

> **GPU 资源巡检报告**
>
> 发现 12 个低利用率作业，共占用 **38 张 GPU**（占已分配的 21%）：
>
> | 作业 | 用户 | GPU | 利用率 | 空闲时长 |
> |------|------|-----|-------|---------|
> | sg-jupyter-user42 | user42 | 2×V100 | 0% | 6小时 |
> | sg-debug-user18 | user18 | 1×A100 | 3% | 4小时 |
> | ... | ... | ... | ... | ... |
>
> **建议**：
> 1. 向空闲作业用户发送释放通知
> 2. 对超过 24 小时零利用率的 Jupyter 作业考虑自动回收

#### 6.3.4 场景四：节点故障隔离（写操作确认）

**管理员输入**："node-gpu-03 的 GPU 不断报 Xid 错误，需要隔离。"

**Agent 工具调用链**：

```
Step 1: get_node_kernel_diagnostics(node_name="node-gpu-03")
        → dmesg: NVRM: Xid 79 (GPU has fallen off the bus)
        → 确认 GPU 硬件故障

Step 2: k8s_list_pods(node_name="node-gpu-03")
        → 当前运行 3 个作业 Pod

Step 3: cordon_node(node_name="node-gpu-03")
        → [确认卡片] 操作: 隔离节点 node-gpu-03
        → 风险等级: HIGH
        → 影响: 阻止新 Pod 调度到该节点
        → 当前运行 Pod 不受影响

        [用户点击确认]

        → 执行成功: node-gpu-03 已标记为 SchedulingDisabled

Step 4: notify_job_owner(node="node-gpu-03")
        → 向 3 个受影响用户发送通知
```

**关键特征**：Step 3 中的 `cordon_node` 操作触发了确认流程，前端展示确认卡片供管理员审核。只有管理员明确确认后，Go 后端才会执行实际的 Kubernetes 操作。

### 6.4 场景分类覆盖

表 6-3 汇总了 10 个测试场景的分类属性：

| # | 故障域 | 作业类型 | 影响范围 | 风险等级 | 运维活动 |
|---|-------|--------|--------|---------|---------|
| 1 | L5 | 单机训练 | 单作业 | 只读 | 响应式诊断 |
| 2 | L5+L1 | DDP 分布式 | 单作业 | 只读 | 响应式诊断 |
| 3 | L4+L3 | 交互式 | 单作业 | 只读 | 响应式诊断 |
| 4 | L3→L4→L5 | 全类型 | 平台级 | 只读 | 跨层推理 |
| 5 | L1→L5 | DDP | 节点+作业 | 只读 | 跨层推理 |
| 6 | L3→L5 | 全类型 | 平台级 | 只读 | 跨层推理 |
| 7 | L5 | 混合 | 多用户 | 需确认 | 主动巡检 |
| 8 | L4 | 混合 | 集群全域 | 只读 | 资源分析 |
| 9 | L1+L2 | — | 节点级 | 需确认 | 故障隔离 |
| 10 | L5 | 混合 | 多用户 | 批量确认 | 僵尸清理 |

**表 6-3 测试场景分类覆盖**

### 6.5 评估方法与指标

采用 "真实数据快照 + 工具模拟" 的离线评测方式，从生产数据中导出故障快照封装为 Mock 工具返回值，确保可复现。评估指标包括：

| 指标 | 定义 | 参考来源 |
|------|------|---------|
| 任务成功率 (SR) | 达成用户目标的比例 | τ-bench |
| 工具选择 F1 | Precision × Recall | BFCL |
| LLM-as-Judge | 准确性+可行性+清晰度 (1-5 分) | MT-Bench |
| Token 消耗 | 单次任务总 token 数 | AIOpsLab |
| 端到端时延 | 问题到回答的 wall-clock 时间 | 自定义 |

**表 6-4 评估指标体系**

### 6.6 对比基线

为验证系统各组件的必要性，设置四组对比基线：

```
┌─────────────────────────────────────────────────────┐
│                  对比实验设计                          │
├──────────────────┬──────────────────────────────────┤
│   基线 1: 规则引擎  │ 平台内置退出码映射 + 分类器       │
│                    │ → 验证 Agent 优于静态规则         │
├──────────────────┼──────────────────────────────────┤
│   基线 2: 纯 LLM   │ 仅 LLM 推理，无工具调用          │
│                    │ → 验证工具的必要性               │
├──────────────────┼──────────────────────────────────┤
│   基线 3: 单智能体  │ 完整工具 + ReAct 循环            │
│                    │ → 对照基线                      │
├──────────────────┼──────────────────────────────────┤
│   基线 4: 多智能体  │ 5 角色 Plan-and-Execute          │
│                    │ → 主实验                        │
└──────────────────┴──────────────────────────────────┘
```

**图 6-1 对比实验设计**

---

## 第七章 总结与展望

### 7.1 工作总结

本文面向智算平台运维场景，设计并实现了一个基于大语言模型的运维智能体系统。主要贡献包括：

**（1）面向智算平台的分层可组合工具架构**。设计了包含 75 个工具的三层工具体系（L1 基础积木层 25 个、L2 领域组合层 36 个、L3 写操作层 14 个），覆盖 Kubernetes 控制面、Volcano 调度器、Prometheus 监控、Harbor 镜像仓库、分布式存储、GPU/RDMA 硬件、分布式训练等智算平台全栈基建。工具的分层设计使得同一组基础工具可以灵活组合覆盖不同平台实体的故障排查，体现了 LLM Agent 的核心价值——由推理能力驱动工具的灵活组合，而非工具数量的堆砌。

**（2）安全可控的三层系统架构**。采用 React 前端、Go 后端、Python Agent 三层分离架构，其中 Go 后端作为安全信任边界，负责身份认证、权限校验和工具执行；Python Agent 层专注于 LLM 推理和工具编排。所有写操作遵循"提案→确认→执行→审计"四步流程，通过角色访问控制（40+ 管理员专用工具）和操作确认机制保障运维安全。

**（3）面向运维场景的智能体编排设计**。实现了单智能体 ReAct 基线和多智能体五角色（协调者、规划者、探索者、执行者、验证者）两种编排模式。多智能体模式通过角色分工实现关注点分离，规划者负责制定调查计划，探索者负责证据收集，验证者负责质疑结论充分性，有效应对了工具集规模大、诊断路径复杂的挑战。

**（4）完整的工程实现与平台集成**。系统已在 Crater 平台（59 节点、220 GPU、464 用户）的实际环境中部署运行，实现了前后端联调、SSE 流式通信、确认卡片交互、会话管理等完整功能链路。覆盖作业故障诊断、分布式训练网络排查、GPU 空闲检测、节点故障隔离等典型运维场景。

### 7.2 不足与展望

本系统仍存在以下可改进之处：

**（1）评测数据集规模有限**。当前仅设计了 10 个核心运维场景用于验证，未来可扩展至更大规模的场景集，并引入自动化的回归测试机制。

**（2）多智能体编排效率**。当前多智能体采用串行协作模型，复杂诊断场景的端到端时延较高。未来可探索部分角色的并行执行（如探索者和验证者并行）以提升效率。

**（3）知识库增强**。当前智能体的运维知识完全来自 LLM 的预训练语料和实时工具调用结果。未来可引入 RAG（Retrieval-Augmented Generation）机制，将平台历史故障案例、运维手册等知识库接入智能体的推理过程，提升诊断准确性和针对性。

**（4）自动修复能力**。当前系统侧重于诊断和辅助操作，写操作需要人工确认。未来可参考 IBM STRATUS 的 TNR 安全规范[7]，在可证明安全的前提下逐步引入自动修复能力，从"人工确认"向"自主执行"演进。

**（5）多模态输入支持**。当前仅支持文本交互。未来可引入截图分析（用户截取错误页面）、日志文件上传等多模态输入方式，进一步降低用户的问题描述门槛。

---

## 参考文献

[1] 国家信息中心. 智能计算中心创新发展指南[R]. 2023.

[2] IDC. 中国智能算力市场规模报告[R]. 2024.

[3] 国家发展改革委. "东数西算"工程进展报告[R]. 2024.

[4] 信通院. 2024年智算平台运维运营技术研究报告[R]. 2024.

[5] Gartner. Market Guide for AIOps Platforms[R]. 2019.

[6] Ahmed T, Ghosh S, Bansal C, et al. Exploring LLM-based Agents for Root Cause Analysis[C]. ACM SIGSOFT International Symposium on the Foundations of Software Engineering (FSE), Industry Track, 2024.

[7] Chen Y, et al. STRATUS: A Multi-agent System for Autonomous Reliability Engineering of Modern Clouds[C]. NeurIPS, 2025.

[8] 裴丹, 等. 微服务系统根因定位[C]. KDD, 2024; 微服务失败测试用例诊断[C]. FSE, 2024; 无监督 KPI 异常检测[C]. WWW, 2024.

[9] K8sGPT Project. k8sgpt.ai[EB/OL]. 2024.

[10] Solo.io. Bringing Agentic AI to Kubernetes: Contributing Kagent to CNCF[EB/OL]. 2025.

[11] Chen Y, Shetty M, Somashekar G, et al. AIOpsLab: A Holistic Framework to Evaluate AI Agents for Enabling Autonomous Clouds[C]. MLSys, 2025.

[12] ITBench: Evaluating AI agents across diverse real-world IT automation tasks[C]. ICML, 2025.

[13] OpenAI. GPT-4 Technical Report[R]. 2023.

[14] Anthropic. Model Context Protocol Specification[S]. 2024.

[15] Yao S, Zhao J, Yu D, et al. ReAct: Synergizing Reasoning and Acting in Language Models[C]. ICLR, 2023.

[16] Spacelift. 15 Common Kubernetes Pitfalls[EB/OL]. 2024.

[17] Site24x7. Kubernetes 2024: Challenges and Solutions[EB/OL]. 2024.

[18] Robustcloud. The Future of Kubernetes[EB/OL]. 2025.

[19] A Survey of AIOps in the Era of Large Language Models[J]. ACM Computing Surveys, 2025.

[20] Flow-of-Action: SOP Enhanced LLM-Based Multi-Agent System for Root Cause Analysis[C]. WWW Companion, 2025.

[21] Schick T, et al. Toolformer: Language Models Can Teach Themselves to Use Tools[C]. NeurIPS, 2023.

[22] Wei J, et al. Chain-of-Thought Prompting Elicits Reasoning in Large Language Models[C]. NeurIPS, 2022.

[23] Martin Fowler. Function Calling using LLMs[EB/OL]. 2025.

[24] Fortune Business Insights. AIOps Market Report[R]. 2025.

[25] τ-bench: A Benchmark for Tool-Agent-User Interaction[C]. 2024.

[26] BFCL: Berkeley Function Calling Leaderboard[EB/OL]. 2024.

[27] MT-Bench: Judging LLM-as-a-Judge[C]. NeurIPS, 2023.

---

## 致谢

感谢导师在选题方向、研究方法和论文写作方面的悉心指导。感谢 Crater 平台运维团队提供的生产数据和运维经验，使本课题能够基于真实场景展开研究。感谢开源社区在 LangGraph、Kubernetes client-go、Prometheus 等项目上的贡献，为本系统的开发提供了坚实的技术基础。
