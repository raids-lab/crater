# 第 1 章 绪论

## 1.1 课题背景与意义

### 1.1.1 课题来源

本课题来源于某高校智算平台 Crater 的实际运维实践。Crater 是一个面向科研与教学场景的异构 GPU 集群管理平台，承载了包括深度学习训练、大模型微调、交互式开发与科学计算在内的多类型 AI 作业。随着平台用户规模与作业复杂度的持续扩张，平台运维人员面临大量重复性工单处理、故障诊断、资源审计与配额审批任务；与此同时，普通用户在面对作业失败、资源紧张、镜像构建、数据集挂载等问题时，也常常缺乏自助排查手段，不得不依赖管理员介入。这种"用户—管理员—平台"之间的信息不对称与高运维负载，成为制约智算平台服务质量与规模化扩展的瓶颈。

本课题针对上述问题，提出**MOps（Multi-agent Orchestration for AI Platform Operations，面向智算平台的多智能体运维编排框架）**，试图以大语言模型驱动的多智能体协作方式，同时为普通用户与管理员提供便捷、智能、可信的运维辅助能力。课题依托 Crater 平台展开，最终产出包含框架设计、工具体系、评测基准与实验验证在内的一整套解决方案。

### 1.1.2 智算平台的兴起

智算平台（AI Computing Platform，又称算力平台或 AI 基础设施）的出现，是深度学习与大规模预训练模型蓬勃发展的直接产物。自 2017 年 Transformer 架构提出以来<sup>[1]</sup>，大语言模型（Large Language Model，LLM）的参数规模以接近每年十倍的速度增长：2018 年 BERT-Large 参数量为 3.4 亿<sup>[2]</sup>，2020 年 GPT-3 达到 1750 亿<sup>[3]</sup>，2022 年 PaLM 达 5400 亿<sup>[4]</sup>，而近两年开源的 DeepSeek-V3<sup>[5]</sup>、Llama-3.1-405B<sup>[6]</sup>、Qwen-2.5-Max 等模型也纷纷进入千亿级参数区间。Kaplan 等人在神经语言模型的缩放定律（Scaling Laws）研究中证明了模型性能与参数量、数据量、计算量之间的幂律关系<sup>[7]</sup>，进一步推动了"更大即更强"的研究范式。

大模型参数规模的爆炸式增长直接转化为算力需求的激增。据公开资料<sup>[8]</sup>，GPT-3 的预训练需要约 3640 PF-days 的算力，相当于 1024 张 NVIDIA V100 GPU 连续训练超过一个月；而 GPT-4 据估算使用了约 25000 张 A100 GPU 训练了数月；Meta 在训练 Llama-3.1-405B 时使用了一个由 16000 张 H100 GPU 组成的集群<sup>[6]</sup>。如此规模的算力消耗已经远超单机或小规模集群的承载能力，必须依赖大规模、高性能、异构化的分布式智算基础设施。

与此同时，AI 模型的训练与推理已经成为各行各业的基础能力需求。除了科技巨头自建 AI Foundry，大量高校、科研机构、企业研发部门也纷纷部署自有智算平台。在政策层面，中国自 2022 年起实施"东数西算"工程，将全国算力网络纳入国家级战略<sup>[9]</sup>；工信部 2023 年发布的《算力基础设施高质量发展行动计划》进一步明确到 2025 年全国算力规模达到 300 EFLOPS 的目标<sup>[10]</sup>。中国信通院发布的《中国智能算力发展评估报告（2024）》预测，2024 年中国智能算力规模将达 725 EFLOPS，年均增速超过 40%<sup>[11]</sup>。智算平台已从"可选基础设施"演变为 AI 产业链的核心枢纽。

> **图 1-1** 大语言模型参数规模与训练算力需求增长趋势（2018–2025）  
> 建议图像提示词：*An academic-style line chart with logarithmic Y-axis showing parameter count (in billions) and training compute (in PF-days) of major LLMs from BERT (2018) to DeepSeek-V3 (2025). Dual Y-axis, clean minimalist style suitable for a thesis.*

### 1.1.3 智算平台的独特性与运维挑战

智算平台并非传统云平台的简单延伸。相比以 Web 应用、微服务、中间件为主要负载的通用云平台，智算平台在作业形态、资源特性与运维模式上呈现出显著差异。

**在作业维度上**，智算平台承载的 AI 工作负载可以从多个维度进行划分。根据代码库 `backend/dao/model/job.go` 中的定义，Crater 平台支持八类作业框架：Jupyter 交互式笔记本、WebIDE 开发环境、PyTorch 分布式训练、TensorFlow 分布式训练、DeepSpeed 大模型训练、OpenMPI 高性能计算、KubeRay 分布式计算以及用户自定义作业。按照作业性质进一步归纳，可得到如表 1-1 所示的分类体系。不同类型的作业对调度、存储、网络、可观测的需求截然不同：训练作业对 GPU 集群带宽敏感、运行时长从小时到数周不等；推理服务对尾延迟敏感、资源弹性伸缩要求高；交互作业追求快速响应与长时空闲容忍；科学计算作业则往往需要 MPI 通信模型与节点级亲和性调度。

**表 1-1** 智算平台典型作业类型分类

| 维度 | 类别 | 典型代表 | 运维关注点 |
|---|---|---|---|
| 计算目的 | 训练 | PyTorch/TensorFlow/DeepSpeed | GPU 利用率、梯度同步、OOM |
| 计算目的 | 推理 | Triton、vLLM | 吞吐、尾延迟、Batch |
| 并行模式 | 单机 | 单卡调试、小模型微调 | 显存、启动延迟 |
| 并行模式 | 分布式 | 多机多卡、流水线并行 | NCCL 通信、RDMA、节点故障 |
| 交互模式 | 批处理 | 提交即运行直至完成 | 排队、抢占、失败重试 |
| 交互模式 | 交互式 | Jupyter、WebIDE | 会话存活、空闲回收、端口暴露 |

**在资源维度上**，智算平台面临前所未有的异构性。GPU 市场已不再是 NVIDIA 一家独大：华为昇腾 910/910B、寒武纪思元、壁仞 BR100、摩尔线程 MTT、燧原 T20 等国产加速卡相继进入主流智算平台；更复杂的是，同一集群内常常混布多代次、多型号的 GPU，甚至同时存在 GPU 与专用 AI 芯片（NPU/TPU/IPU）。Crater 在 `backend/dao/model/resource.go` 中将自定义资源抽象为 `CraterResourceType`，区分 GPU、RDMA 网络与 VGPU 三大类，并通过 `VendorDomain` 字段识别不同厂商。除计算资源外，高速互联网络（InfiniBand、RoCE v2）、高性能并行存储（NFS、Lustre、Ceph）以及 GPU 虚拟化（VGPU）能力同样是平台运维必须管理的一等对象。

**在运维维度上**，智算平台的故障模式与传统云平台存在本质区别。一个典型的分布式训练作业失败可能来源于：GPU 显存溢出（OOM）、NCCL 初始化超时、RDMA 网卡硬件错误、Pod 被驱逐（preemption）、共享存储 I/O 抖动、数据集路径配置错误、容器镜像 CUDA 版本不匹配、用户代码死锁等数十种原因，且这些原因常常耦合发生。相比微服务场景中以"调用链—指标—日志"三大支柱<sup>[12]</sup>为核心的 APM 方法论，智算作业的可观测对象涵盖了从 GPU SM 利用率、HBM 带宽到分布式训练 loss 曲线、NCCL 拓扑的大量领域特异信号。此外，作业生命周期通常远长于微服务请求（小时到数周），单次故障的代价（算力浪费）也远高于微服务单次失败。

上述三个维度上的独特性，决定了智算平台运维不能简单复用面向 Web 服务的通用 AIOps 方案，而亟需面向 AI 工作负载特点的专用方法论与工具体系。

### 1.1.4 智算平台运维的演进历程

从更宏观的视角看，智算平台运维是整个 IT 运维方法论演进过程在 AI 时代的新阶段。回顾近二十年运维技术的发展脉络，大致可划分为五个阶段，如图 1-2 所示。

> **图 1-2** IT 运维方法论演进时间线（2005–2026）  
> 建议图像提示词：*A horizontal timeline infographic showing five stages of IT operations evolution: Traditional Ops (pre-2010), DevOps & Cloud (2010-2015), Cloud-Native Observability (2015-2019), AIOps (2016-2022), LLM Ops (2022-2024), and Agentic AIOps (2024-present). Each stage with representative tools and key characteristics. Academic style, suitable for thesis.*

**第一阶段：传统运维（~2010 年前）**。依赖专职运维工程师执行人工巡检、Shell 脚本、Zabbix/Nagios 等工具进行被动告警与故障处理，典型缺陷是"救火式"响应、知识难以沉淀、扩展性差。

**第二阶段：DevOps 与云计算（2010–2015）**。以基础设施即代码（IaC）、持续集成/持续部署（CI/CD）、OpenStack 与 AWS/Azure 公有云为代表，运维开始走向标准化与自动化。但监控体系仍以单一指标阈值告警为主。

**第三阶段：云原生可观测性（2015–2019）**。Kubernetes 成为事实标准<sup>[13]</sup>，Prometheus<sup>[14]</sup>、Grafana、Jaeger<sup>[15]</sup>、ELK 栈、OpenTelemetry<sup>[16]</sup>等工具构建起可观测性三大支柱（Metrics/Logs/Traces）。这一阶段解决了"看得到"的问题，但海量数据的根因分析仍依赖专家经验。

**第四阶段：AIOps（2016–2022）**。Gartner 在 2016 年首次提出 AIOps 概念<sup>[17]</sup>，倡导以机器学习方法实现异常检测、日志聚类、根因定位与预测性维护。学术界与工业界涌现出大量工作：Lou 等人利用聚类做日志异常检测<sup>[18]</sup>、Microsoft 的 DejaVu 做微服务故障定位<sup>[19]</sup>、阿里、腾讯等厂商发布了各自的 AIOps 平台。Notaro 等人对该阶段的方法做了系统综述<sup>[20]</sup>。然而传统 AIOps 模型往往是任务专用、样本依赖严重，且难以处理文本语义、跨模态信号与复杂推理。

**第五阶段：LLM Ops（2022–2024）**。ChatGPT 的出现让大语言模型成为运维领域新的可能性：日志摘要、告警解释、故障说明书自动生成等任务展现出亮眼的效果。然而初期工作多为"LLM + prompt"的单轮调用，容易产生幻觉、无法主动调用工具、难以处理多步推理。Ahmed 等人评测发现，单 LLM 在根因分析任务上的准确率往往不足 40%<sup>[21]</sup>，离实用仍有距离。

**第六阶段：Agentic AIOps（2024 至今）**。ReAct<sup>[22]</sup>、Toolformer<sup>[23]</sup>、Plan-and-Solve<sup>[24]</sup>、Reflexion<sup>[25]</sup>等 LLM 智能体技术的成熟，使得 LLM 不再是被动的问答组件，而能够主动规划、调用工具、观察结果、自我反思。Microsoft 的 RCACopilot<sup>[26]</sup>率先将此类框架用于微服务根因分析，随后涌现出 D-Bot（数据库诊断）<sup>[27]</sup>、AIOpsLab 基准<sup>[28]</sup>、ITBench<sup>[29]</sup>等一批工作。多智能体编排框架如 AutoGen<sup>[30]</sup>、MetaGPT<sup>[31]</sup>、LangGraph<sup>[32]</sup>也陆续进入运维领域，预示着 Agentic AIOps 将成为下一代运维的核心范式。

然而值得注意的是，上述 Agentic AIOps 工作**绝大多数聚焦于微服务、数据库、通用 IT 系统**，针对智算平台与 GPU 集群的研究仍几近空白。这正是本课题着重填补的方向。

### 1.1.5 研究意义

综合上述背景分析，本课题的研究具有以下几方面的意义：

**学术层面**，本课题首次系统性地将多智能体 LLM 编排框架引入智算平台运维领域，填补了 Agentic AIOps 在 GPU 集群场景的研究空白；提出的任务感知编排、异构 LLM 分层、证据工程等关键技术可为后续相关研究提供参考与基准。

**工程层面**，本课题基于 Crater 平台的真实运维场景，实现了可落地的 MOps 框架原型，支持 70+ 种运维工具调用、用户与管理员双视角、读写操作分级审批等生产级特性；并配套提出了 Crater-Bench 基准，包含 60+ 个标注完善的运维场景，为社区提供了第一个专门面向智算平台的 Agentic AIOps 评测数据集。

**社会层面**，通过降低智算平台的运维门槛，本课题有望让更多高校、科研机构、中小企业以更低的人力成本运营自有智算基础设施，间接支持国产异构算力生态的发展，呼应"东数西算"国家战略对算力普惠化的要求。

## 1.2 国内外研究现状

本节从四个相邻方向综述国内外已有工作，分别是：智算平台与 GPU 集群管理、AIOps 与可观测性、LLM 智能体与工具使用、以及面向运维的 Agentic AIOps。第 2 章将对这些方向展开更详细的比较。

### 1.2.1 智算平台与 GPU 集群管理

GPU 集群管理的研究起步于 2018 年前后。Microsoft Research 的 Gandiva<sup>[33]</sup>首次提出针对深度学习作业特点（迭代性、可预测性）的反思式调度。Tiresias<sup>[34]</sup>进一步引入无先验知识的 GPU 调度策略。OSDI 2021 的 Pollux<sup>[35]</sup>基于 goodput 概念实现了吞吐—统计效率联合优化的 co-adaptive 调度。Alibaba 发表于 NSDI 2022 的 MLaaS in the Wild<sup>[36]</sup>公开了包含 650 万个 GPU 作业的真实 trace，揭示了生产环境中 GPU 集群作业具有显著的长尾分布、异构资源需求与频繁失败的特征。以上工作奠定了智算平台底层调度的理论基础，但几乎全部聚焦于"**如何更好地调度**"，而非"**如何更好地运维**"。

工业界方面，NVIDIA Base Command<sup>[37]</sup>、AWS SageMaker HyperPod、华为 ModelArts、阿里云 PAI 灵骏、百度百舸等智算云平台提供了相对完整的训练生命周期管理，但其运维能力主要体现在基础监控与工单系统层面，智能化辅助决策仍有限。开源生态中，Kubeflow<sup>[38]</sup>、Volcano<sup>[39]</sup>、Ray<sup>[40]</sup>等组件提供了作业编排与批处理支持，但缺乏统一的智能运维层。

### 1.2.2 AIOps 与可观测性

AIOps 领域已有较为丰富的综述工作。Notaro 等人<sup>[20]</sup>将 AIOps 研究分为故障预防、故障检测、根因分析与故障修复四类，指出当时主流方法集中在前两类，而根因分析与修复仍是难点。Dang 等人<sup>[41]</sup>从工业实践视角总结了微软大规模 AIOps 部署中遇到的七大挑战，包括数据质量、标签稀缺、模型泛化、可解释性等。Soldani 等人<sup>[42]</sup>则从分布式追踪角度综述了微服务根因分析方法。

微服务可观测性研究上，Jaeger<sup>[15]</sup>、Zipkin、OpenTelemetry<sup>[16]</sup>构建起事实标准；SkyWalking<sup>[43]</sup>在国内应用广泛。学术研究方面，TraceAnomaly<sup>[44]</sup>用变分自编码器检测调用链异常，MicroRCA<sup>[45]</sup>基于因果图实现微服务根因定位。然而这些工作全部建立在"调用链 + HTTP/gRPC 请求"假设之上，无法迁移到以长时 GPU 作业为核心的智算场景。

### 1.2.3 LLM 智能体与工具使用

自 Chain-of-Thought<sup>[46]</sup>证明了 LLM 的多步推理能力以来，面向工具使用的智能体技术迅速发展。ReAct<sup>[22]</sup>提出将推理（Reasoning）与行动（Acting）交替进行的范式，奠定了当前主流智能体架构的基础。Toolformer<sup>[23]</sup>研究了 LLM 自学习调用外部 API 的能力，OpenAI Function Calling<sup>[47]</sup>将工具调用能力作为模型标准接口。Model Context Protocol（MCP）<sup>[48]</sup>则于 2024 年由 Anthropic 提出，意图统一工具协议。

在编排层面，Plan-and-Solve<sup>[24]</sup>让 LLM 先生成规划再执行、Reflexion<sup>[25]</sup>引入自我反思机制、Tree-of-Thoughts<sup>[49]</sup>探索多路径推理。在多智能体层面，AutoGen<sup>[30]</sup>提供对话式多智能体抽象，MetaGPT<sup>[31]</sup>将软件开发团队的角色分工引入 LLM 协作，LangGraph<sup>[32]</sup>以有向图的方式描述复杂工作流，CrewAI<sup>[50]</sup>强调任务分配与团队协作。

尽管上述框架能力强大，但它们都是通用型工具，并未针对特定领域做深度优化。在运维这类工具繁多、权限敏感、失误代价高的场景中，通用多智能体框架往往表现出工具选择盲目、推理链冗长、成本失控等问题。

### 1.2.4 Agentic AIOps 新进展

将 LLM 智能体用于运维的研究在 2024 年后进入爆发期。Chen 等人提出的 RCACopilot<sup>[26]</sup>将根因分析建模为"证据收集 + 假设检验"流程，使用 LLM 驱动的智能体自动执行监控查询并归纳故障原因。Zhou 等人的 D-Bot<sup>[27]</sup>面向数据库诊断任务，设计了包含 DBA 经验知识的多智能体系统。Shetty 等人发布的 AIOpsLab<sup>[28]</sup>是首个面向 Agentic AIOps 的标准化基准测试平台，包含故障诊断、异常检测、修复建议等多类任务；IBM Research 的 ITBench<sup>[29]</sup>则提供了 SRE 场景下的端到端基准。Roy 等人的 FLASH<sup>[51]</sup>尝试用链式结构化思维提升根因分析稳定性。

然而，从已发表的数据看，现有 Agentic AIOps 系统在以下三点仍有明显不足：（1）**领域偏狭**——绝大多数面向微服务、数据库或通用 IT 场景，针对 GPU 集群/智算平台的工作寥寥无几；（2）**编排单一**——多数工作采用单一编排模式（全程 ReAct 或全程 Plan-Execute），未充分利用任务复杂度感知的自适应编排；（3）**成本失控**——为追求效果多选用顶级 LLM 执行所有步骤，缺乏异构模型协作的工程优化。这也正是本课题试图弥补的关键缺口。

### 1.2.5 研究空白总结

综合上述四个方向的综述，可以清晰地看出以下研究空白：

1. GPU 集群管理研究聚焦调度优化，缺少针对 AI 作业运维全生命周期的智能化方案；
2. 传统 AIOps 研究建立在微服务体系之上，方法论与工具链均不适用于长时 GPU 作业与异构算力场景；
3. LLM 智能体研究提供了强大的底层能力，但通用多智能体框架缺乏对智算运维任务类型差异的感知；
4. Agentic AIOps 研究已开始涉及 IT 运维，但智算平台方向几近空白，且编排模式与成本控制仍存在明显提升空间。

本课题正是在这一交叉空白区域定位的，其创新性贡献将在 1.3 节中展开。

## 1.3 研究目标与内容

### 1.3.1 研究目标

本课题的总体目标是**设计、实现并评测一个面向智算平台运维的多智能体编排框架 MOps**，使其能够同时为普通用户与管理员提供便捷、智能、可信的运维辅助，从而降低智算平台的运维门槛并提升整体服务质量。具体目标包含以下三点：

**目标 1：高效**——相比单智能体 ReAct 基线，在常见运维任务上实现任务成功率显著提升、工具调用更精准、端到端延迟可控。

**目标 2：经济**——通过异构 LLM 分层与证据工程，在不显著损失质量的前提下显著降低 LLM 调用成本，让框架具备大规模部署的可行性。

**目标 3：可信**——对写操作类任务引入验证智能体与分级审批机制，保证高风险动作的安全性；同时提供完整的调用链审计与可观测性，满足生产环境的可追溯要求。

### 1.3.2 研究内容

围绕上述目标，本课题的研究内容包括：

**研究内容 1：MOps 多智能体编排框架的总体设计**。构建涵盖意图路由、规划智能体、探索智能体、执行智能体、验证智能体、协调器、审批智能体在内的八类角色分工体系，以有向图方式组织智能体间的协作流。

**研究内容 2：任务感知的自适应编排机制**。对智算运维任务按查询、诊断、运维审计、工单提交、修复五类进行划分，根据任务复杂度自适应选择 ReAct、Plan-Execute、Plan-Execute-Verify 三档编排模式，避免简单任务过度规划与复杂任务规划不足并存。

**研究内容 3：异构 LLM 分层与成本优化**。针对协调器、规划器、探索器、执行器等不同角色的能力需求差异，采用"主模型 + 辅助模型"的异构分层策略，在关键推理环节使用强模型（如 Qwen3-235B-Thinking），在工具选择与证据整理环节使用轻量模型（如 Qwen3-VL-Flash），显著降低总体成本。

**研究内容 4：上下文工程与证据压缩**。针对 LLM 上下文窗口有限与工具返回结果庞杂的矛盾，设计分层 System Prompt、基于 token 预算的工具结果截断、LLM 驱动的证据摘要、会话历史压缩等多项上下文工程技术，保证长程任务下的稳定性与一致性。

**研究内容 5：权限感知的工具层与双路由架构**。设计包含 70+ 种工具的专用工具库，覆盖作业诊断、指标查询、集群管理、存储审计、网络诊断、GPU 分析、工单操作等多个运维领域；通过进程内本地执行与 HTTP 远程调用相结合的双路由架构，同时满足高性能与安全权限隔离需求。

**研究内容 6：Crater-Bench 基准与实验评测**。构建包含 60 个以上场景、覆盖诊断、运维审计、查询、工单提交四类任务的标准化基准数据集；设计离线快照与在线只读两种评测模式，支持单/多智能体编排的对比实验；并通过消融实验验证 MOps 各关键组件的贡献。

## 1.4 论文组织结构

本论文共分为六章，具体结构如图 1-3 所示。

> **图 1-3** 论文组织结构  
> 建议图像提示词：*A minimalist flowchart showing 6 chapters of a thesis from top to bottom: Introduction → Related Work → System Design (核心) → Experimental Evaluation → Case Study → Conclusion. Arrows indicate logical flow. Academic style.*

**第 1 章 绪论**，即本章。介绍课题背景、智算平台的兴起与运维演进、国内外研究现状、研究目标与论文组织结构。

**第 2 章 国内外相关研究工作**。对智算平台与 GPU 集群管理、AIOps 与可观测性、LLM 智能体与工具使用、多智能体编排框架、Agentic AIOps 新进展等五个方向展开详细综述，分析已有工作的贡献与局限，明确本课题的切入点。

**第 3 章 MOps 多智能体运维编排框架的设计**。本论文的核心技术章节。按照总体架构—任务感知编排—异构 LLM 分层—证据工程—权限感知工具层的顺序，系统阐述 MOps 框架的设计理念、关键组件与实现技术。

**第 4 章 MOps 框架在 Crater 平台上的实现与实验评测**。介绍基于 Crater 的系统实现细节、Crater-Bench 基准构建过程、实验设定与评测指标；并通过与单智能体 ReAct 基线、LLM-only 基线、MOps 同构 LLM 变体的对比实验，验证 MOps 在任务成功率、成本与安全性等维度上的优势。

**第 5 章 典型案例分析**。选取诊断类、运维审计类、工单审批类三个具有代表性的真实场景，通过完整的智能体调用链走读，展示 MOps 的工作机制与运维效果；并以一个反例分析框架的局限性。

**第 6 章 总结与展望**。总结本课题的主要贡献，讨论研究过程中遇到的困难与解决思路，并对 MOps 框架的后续扩展方向（如 Remedy 自动修复基准、多 LLM 家族验证、在线部署实践）进行展望。

## 参考文献（第 1 章引用列表，完整版见论文末参考文献）

[1] Vaswani A, Shazeer N, Parmar N, et al. Attention Is All You Need[C]. NeurIPS, 2017.  
[2] Devlin J, Chang M W, Lee K, et al. BERT: Pre-training of Deep Bidirectional Transformers for Language Understanding[C]. NAACL, 2019.  
[3] Brown T B, Mann B, Ryder N, et al. Language Models are Few-Shot Learners[C]. NeurIPS, 2020. arXiv:2005.14165.  
[4] Chowdhery A, Narang S, Devlin J, et al. PaLM: Scaling Language Modeling with Pathways[J]. arXiv:2204.02311, 2022.  
[5] DeepSeek-AI. DeepSeek-V3 Technical Report[J]. arXiv:2412.19437, 2024.  
[6] Dubey A, Jauhri A, Pandey A, et al. The Llama 3 Herd of Models[J]. arXiv:2407.21783, 2024.  
[7] Kaplan J, McCandlish S, Henighan T, et al. Scaling Laws for Neural Language Models[J]. arXiv:2001.08361, 2020.  
[8] Patterson D, Gonzalez J, Le Q, et al. Carbon Emissions and Large Neural Network Training[J]. arXiv:2104.10350, 2021.  
[9] 国家发改委. "东数西算"工程战略布局[R]. 2022.  
[10] 工业和信息化部. 算力基础设施高质量发展行动计划[R]. 2023.  
[11] 中国信息通信研究院. 中国智能算力发展评估报告（2024）[R]. 2024.  
[12] Sridharan C. Distributed Systems Observability[M]. O'Reilly, 2018.  
[13] Burns B, Grant B, Oppenheimer D, et al. Borg, Omega, and Kubernetes[J]. CACM, 2016, 59(5).  
[14] Rabenstein B, Volz J. Prometheus: A Next-Generation Monitoring System[C]. SREcon, 2015.  
[15] Shkuro Y. Mastering Distributed Tracing[M]. Packt, 2019.  
[16] OpenTelemetry Community. OpenTelemetry Specification[EB/OL]. 2021.  
[17] Gartner. Market Guide for AIOps Platforms[R]. 2016.  
[18] Lou J G, Fu Q, Yang S, et al. Mining Invariants from Console Logs for System Problem Detection[C]. USENIX ATC, 2010.  
[19] Chen Y, Yang X, Lin Q, et al. Outage Prediction and Diagnosis for Cloud Service Systems[C]. WWW, 2019.  
[20] Notaro P, Cardoso J, Gerndt M. A Survey of AIOps Methods for Failure Management[J]. ACM TOIT, 2021.  
[21] Ahmed T, Ghosh S, Bansal C, et al. Recommending Root-Cause and Mitigation Steps for Cloud Incidents using LLMs[C]. ICSE, 2023.  
[22] Yao S, Zhao J, Yu D, et al. ReAct: Synergizing Reasoning and Acting in Language Models[C]. ICLR, 2023. arXiv:2210.03629.  
[23] Schick T, Dwivedi-Yu J, Dessì R, et al. Toolformer: Language Models Can Teach Themselves to Use Tools[C]. NeurIPS, 2023.  
[24] Wang L, Xu W, Lan Y, et al. Plan-and-Solve Prompting[C]. ACL, 2023. arXiv:2305.04091.  
[25] Shinn N, Cassano F, Gopinath A, et al. Reflexion: Language Agents with Verbal Reinforcement Learning[C]. NeurIPS, 2023.  
[26] Chen Y, Xie H, Ma M, et al. Automatic Root Cause Analysis via Large Language Models for Cloud Incidents[C]. EuroSys, 2024.  
[27] Zhou X, Li G, Sun Z, et al. D-Bot: Database Diagnosis System using Large Language Models[J]. arXiv:2312.01454, 2024.  
[28] Shetty M, Chen Y, Somashekar G, et al. Building AI Agents for Autonomous Clouds: Challenges and Design Principles[J]. arXiv:2407.12165, 2024.  
[29] IBM Research. ITBench: Evaluating AI Agents for IT Automation[J]. arXiv, 2024.  
[30] Wu Q, Bansal G, Zhang J, et al. AutoGen: Enabling Next-Gen LLM Applications via Multi-Agent Conversation Framework[J]. arXiv:2308.08155, 2023.  
[31] Hong S, Zheng X, Chen J, et al. MetaGPT: Meta Programming for Multi-Agent Collaborative Framework[C]. ICLR, 2024. arXiv:2308.00352.  
[32] LangChain. LangGraph Documentation[EB/OL]. 2024.  
[33] Xiao W, Bhardwaj R, Ramjee R, et al. Gandiva: Introspective Cluster Scheduling for Deep Learning[C]. OSDI, 2018.  
[34] Gu J, Chowdhury M, Shin K G, et al. Tiresias: A GPU Cluster Manager for Distributed Deep Learning[C]. NSDI, 2019.  
[35] Qiao A, Choe S K, Subramanya S J, et al. Pollux: Co-adaptive Cluster Scheduling for Goodput-Optimized Deep Learning[C]. OSDI, 2021.  
[36] Weng Q, Xiao W, Yu Y, et al. MLaaS in the Wild: Workload Analysis and Scheduling in Large-Scale Heterogeneous GPU Clusters[C]. NSDI, 2022.  
[37] NVIDIA. NVIDIA Base Command Platform[EB/OL]. 2022.  
[38] Kubeflow Community. Kubeflow[EB/OL]. 2018.  
[39] Volcano. Volcano: Cloud Native Batch System[EB/OL]. 2019.  
[40] Moritz P, Nishihara R, Wang S, et al. Ray: A Distributed Framework for Emerging AI Applications[C]. OSDI, 2018.  
[41] Dang Y, Lin Q, Huang P. AIOps: Real-World Challenges and Research Innovations[C]. ICSE-SEIP, 2019.  
[42] Soldani J, Brogi A. Anomaly Detection and Failure Root Cause Analysis in (Micro)Service-Based Cloud Applications: A Survey[J]. ACM Computing Surveys, 2022.  
[43] Apache SkyWalking. SkyWalking: APM for Cloud-Native Era[EB/OL]. 2018.  
[44] Liu P, Xu H, Ouyang Q, et al. Unsupervised Detection of Microservice Trace Anomalies through Service-Level Deep Bayesian Networks[C]. ISSRE, 2020.  
[45] Wu L, Tordsson J, Elmroth E, et al. MicroRCA: Root Cause Localization of Performance Issues in Microservices[C]. NOMS, 2020.  
[46] Wei J, Wang X, Schuurmans D, et al. Chain-of-Thought Prompting Elicits Reasoning in Large Language Models[C]. NeurIPS, 2022.  
[47] OpenAI. Function Calling and Other API Updates[EB/OL]. 2023.  
[48] Anthropic. Introducing the Model Context Protocol[EB/OL]. 2024.  
[49] Yao S, Yu D, Zhao J, et al. Tree of Thoughts: Deliberate Problem Solving with Large Language Models[C]. NeurIPS, 2023.  
[50] CrewAI. CrewAI Documentation[EB/OL]. 2024.  
[51] Roy D, Zhang X, Bhave S, et al. FLASH: A Workflow Automation Agent for Diagnosing Recurring Incidents[J]. arXiv, 2024.
