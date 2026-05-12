# 摘要

智算平台（AI Computing Platform）是承载深度学习训练、大模型微调、科学计算等 AI 工作负载的关键基础设施。随着平台规模与作业复杂度的持续扩张，用户在面对 GPU 显存溢出（OOM）、NCCL 通信失败、分布式训练异常等问题时缺乏自助排查手段，管理员则承担着高重复性的工单处理、故障诊断与资源审计负担。现有 Agentic AIOps 研究多集中于微服务与传统云资源管理场景，对智算平台的特有挑战缺乏系统性解决方案。

针对上述问题，本文提出并实现了**MOps（Multi-agent Orchestration for AI Platform Operations）**——面向智算平台的多智能体运维编排框架。MOps 包含五项关键技术创新：（1）**任务感知的自适应编排**，通过意图路由器（IntentRouter）将运维任务分类为查询、诊断、运维审计、工单提交、修复五大类，自适应选择 ReAct、Plan-Execute、Plan-Execute-Verify 三档编排模式；（2）**异构 Agent 栈与 LLM 分层**，设计 9 种角色智能体，采用主模型（Qwen3-235B-Thinking）与辅助模型（Qwen3-Flash）的异构路由以优化成本-质量权衡；（3）**上下文工程与证据压缩**，通过分层 System Prompt、Token 预算控制、LLM 驱动的证据摘取与会话历史压缩解决长轨迹任务的上下文爆炸问题；（4）**权限感知的双路由工具层**，79 个工具通过进程内本地执行与 HTTP 远程执行的双路由架构同时满足高性能与安全权限隔离需求；（5）**验证智能体与审批链**，对写操作设计风险分级、确认流、后验证与完整审计链的可信流程。

基于某高校真实智算平台 Crater 的 25812 个历史作业数据，本文构建了首个面向智算平台的离线评测基准 **Crater-Bench**，覆盖诊断、查询、运维审计、工单提交四大类 30 个标准化场景（已标注），并支持扩展至 60 个场景的完整评测体系。通过 MOPS（多智能体编排）、PS（Plan-Solve 编排）和 React（单智能体）三种方法的对照实验证明：在多轮诊断对话场景中，MOPS 相比 React 在 OOM 诊断等困难任务上得分提升 8.3 分（从 88.0 到 96.3），在终止类诊断中提升 16.1 分（从 70.4 到 86.4）；在简单查询和运维审计场景中三者表现接近，验证了自适应编排的合理性。在 4 场景 DeepSeek-V4-Pro 扩展实验中，MOPS 取得平均分 93.78、工具选择 F1 满分（1.0）的优异表现，证实了框架的跨模型泛化能力。

本课题的研究不仅填补了 Agentic AIOps 在智算平台方向的空白，还提供了完整可落地的开源框架与评测基准。MOps 已作为 Crater 智算平台的一部分在实际生产环境中运行。

**关键词**：智算平台；多智能体编排；AIOps；大语言模型；智能体框架；GPU 集群运维；工具使用

---

# Abstract

AI Computing Platforms (AICPs) have become the foundational infrastructure for deep learning training, large-model fine-tuning, and scientific computing workloads. As platform scale and workload complexity continue to grow, regular users struggle to self-diagnose issues such as GPU out-of-memory errors, NCCL communication failures, and distributed training anomalies, while administrators bear heavy burdens of repetitive ticket handling, fault diagnosis, and resource auditing. Existing Agentic AIOps research has focused primarily on microservices and traditional cloud resource management, leaving the unique challenges of AI computing platforms largely unaddressed.

This thesis proposes and implements **MOps (Multi-agent Orchestration for AI Platform Operations)**, a domain-specialized multi-agent framework for AICP operations. MOps contributes five key technical innovations: (1) **task-aware adaptive orchestration** via an IntentRouter that classifies operational tasks into five categories and dynamically selects among ReAct, Plan-Execute, and Plan-Execute-Verify modes; (2) **heterogeneous agent stack and LLM tiering** with nine role agents using primary (Qwen3-235B-Thinking) and auxiliary (Qwen3-Flash) models for cost-quality optimization; (3) **context engineering and evidence compaction** through layered system prompts, token budgets, LLM-driven evidence extraction, and conversation history compression; (4) **permission-aware dual-routing tool layer** with 79 tools routed between in-process local execution and HTTP remote execution; and (5) **verifier agents and approval chains** with risk stratification, confirmation flows, post-action verification, and full audit trails.

We construct **Crater-Bench**, the first offline evaluation benchmark for AICP operations, built from 25,812 real historical job records. The benchmark contains 30 annotated scenarios across four categories (diagnosis, query, ops audit, submission) with an extensible 60-scenario framework. Controlled experiments comparing MOPS (multi-agent), PS (Plan-Solve), and React (single-agent) methods demonstrate that MOPS outperforms React by 8.3 points (88.0→96.3) on difficult OOM diagnosis tasks and by 16.1 points (70.4→86.4) on termination diagnosis, while matching on simple queries—validating adaptive orchestration. In 4-scenario DeepSeek-V4-Pro extension experiments, MOPS achieves 93.78 average score with perfect tool selection F1 (1.0), confirming cross-model generalization capability.

This work fills the research gap of Agentic AIOps in AI computing platforms and provides a deployable open-source framework and benchmark. MOps has been integrated into the Crater AICP as part of its production service.

**Keywords**: AI Computing Platform; Multi-Agent Orchestration; AIOps; Large Language Models; Agentic Framework; GPU Cluster Operations; Tool Use
