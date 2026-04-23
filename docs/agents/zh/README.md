# Mops：面向智能运维的多智能体框架

> **M**ulti-agent **Op**eration**s** — 一个用于在智算平台上构建智能运维智能体的框架。

---

## 概述

Mops 是驱动 Crater 智算平台的多智能体运维框架。它提供了一套可组合的架构，用于构建、编排和评估基于 LLM 的智能体，这些智能体可在 GPU/加速器集群上执行故障诊断、资源优化、主动巡检和自动化运维。

该框架基于三个核心设计原则：

1. **平台无关的工具层** — 智能体通过可插拔的工具执行器接口与基础设施交互，而非直接调用 API
2. **可组合的智能体角色** — 每个智能体承担单一职责并拥有受限的工具集；添加新智能体无需修改现有智能体
3. **安全优先的运维** — 写操作需要显式确认，所有操作均可审计，故障始终回退至人工操作

---

## 框架架构

```
                         ┌─────────────────────────────────────┐
                         │           应用层                     │
                         │   FastAPI /chat（SSE 流式传输）       │
                         │   FastAPI /evaluate/*（同步）         │
                         └──────────────┬──────────────────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    │                   │                   │
            ┌───────▼───────┐   ┌──────▼──────┐   ┌───────▼───────┐
            │   单智能体     │   │  多智能体    │   │  任务专用      │
            │   编排器       │   │  编排器      │   │  智能体        │
            │ （ReAct 循环） │   │（协调器      │   │ （审批、       │
            │               │   │  流水线）    │   │   流水线）     │
            └───────┬───────┘   └──────┬──────┘   └───────┬───────┘
                    │                  │                   │
                    └──────────────────┼───────────────────┘
                                       │
                         ┌─────────────▼─────────────┐
                         │     智能体图层              │
                         │  LangGraph ReAct StateGraph│
                         │  (agent → tools → agent)   │
                         └─────────────┬─────────────┘
                                       │
                    ┌──────────────────┼──────────────────┐
                    │                  │                  │
            ┌───────▼───────┐  ┌──────▼──────┐  ┌───────▼───────┐
            │  工具选择器    │  │    Token     │  │   技能        │
            │ （基于角色）   │  │    管理      │  │   知识库      │
            └───────┬───────┘  └─────────────┘  └───────────────┘
                    │
         ┌──────────┼──────────┐
         │          │          │
   ┌─────▼────┐ ┌──▼─────┐ ┌──▼──────────┐
   │  本地     │ │  Go    │ │   Mock      │
   │ 执行器   │ │ 后端   │ │  执行器     │
   │(kubectl, │ │执行器  │ │（基准测试） │
   │PromQL,web)│ │(HTTP) │ │             │
   └──────────┘ └────────┘ └─────────────┘
```

---

## 智能体目录

| 智能体 | 模式 | 用途 | 工具 | 文档 |
|--------|------|------|------|------|
| **单智能体** | ReAct | 通用诊断与运维 | 全部（按角色过滤） | [single-agent.md](single-agent.md) |
| **协调器** | MAS | 请求路由与阶段编排 | 无（纯 LLM） | [coordinator.md](coordinator.md) |
| **规划器** | MAS | 将请求分解为调查计划 | 无（纯 LLM） | [planner.md](planner.md) |
| **探索器** | MAS | 通过只读工具收集证据 | 只读子集 | [explorer.md](explorer.md) |
| **执行器** | MAS | 执行需确认的写操作 | 读 + 写 | [executor.md](executor.md) |
| **验证器** | MAS | 验证结论并标记证据缺口 | 无（纯 LLM） | [verifier.md](verifier.md) |
| **工单智能体** | 任务 | 工单/审批单评估智能体的基类 | 可配置白名单 | [ticket-agent.md](ticket-agent.md) |
| **审批** | 任务 | 评估作业锁定审批单 | 固定白名单（7 个工具） | [approval-agent.md](approval-agent.md) |
| **巡检** | 任务 | 定时集群健康巡检与报告 | 只读（通过 PipelineToolClient） | [inspection-pipeline.md](inspection-pipeline.md) |
| **引导** | MAS | 回答平台"如何操作"类问题 | 无（纯 LLM） | [guide-general.md](guide-general.md) |
| **通用** | MAS | 处理问候语和简单的平台问答 | 无（纯 LLM） | [guide-general.md](guide-general.md) |

---

## 编排模式

### 单智能体（ReAct）

单个具有工具访问权限的 LLM 实例运行"思考-行动-观察"循环，直到生成最终答案或达到工具调用限制。

```
用户消息 → [LLM 思考] → [调用工具] → [观察结果] → [LLM 思考] → ... → 最终答案
```

适用场景：简单查询、直接诊断、单作业操作。

### 多智能体（协调器流水线）

协调器 LLM 路由请求，并通过分阶段流水线编排专用子智能体。

```
用户消息 → [协调器路由] → [规划器] → [探索器循环] → [执行器循环] → [验证器] → 最终答案
                                                                        ↑
                                                  协调器可能回环 ────────┘
```

适用场景：复杂的多步骤调查、跨对象关联分析、需要规划的运维操作。

### 任务专用（同步）

独立的智能体，复用 ReAct 图但以同步方式运行，用于特定的后端钩子（非面向用户的对话）。

```
后端事件 → [拥有受限工具的任务智能体] → 结构化结果 → 后端动作
```

适用场景：自动化决策（审批评估）、批量分析（GPU 审计）、定时报告。

---

## 可扩展性

### 添加新智能体

1. 创建 `agents/your_agent.py`，继承 `BaseRoleAgent` 的模式
2. 定义工具白名单（现有工具的子集）
3. 编写针对该智能体角色的系统提示词
4. 选择集成模式：
   - **MAS 子智能体**：在 `MultiAgentOrchestrator` 中添加为新阶段
   - **任务智能体**：在 `app.py` 中添加新的 FastAPI 端点
   - **ReAct 智能体**：复用 `create_agent_graph` 并配合 `capabilities.enabled_tools`

无需修改现有代码 — 图、工具执行器和 Token 管理均可复用。

### 添加新工具

1. 在 `tools/definitions.py` 中使用 `@tool` 装饰器定义工具函数
2. 添加到 `AUTO_TOOLS`（只读）或 `CONFIRM_TOOLS`（写操作）
3. 在 `handler/agent/tools_readonly.go` 或 `tools_dispatch.go` 中添加 Go 后端处理器
4. 如果可移植（无 Go 依赖），在 `tools/local_executor.py` 中实现

### 适配到不同平台

该框架将**智能体逻辑**与**平台细节**分离：

| 层级 | 需要修改的内容 | 保持不变的内容 |
|------|---------------|---------------|
| 工具 | 工具定义 + Go 处理器 | 智能体图、编排、提示词模式 |
| 认证 | `GoBackendToolExecutor` 认证头 | 智能体角色、工具选择逻辑 |
| 配置 | `platform-runtime.yaml` 端点 | LLM 客户端工厂、Token 管理 |
| 技能 | 诊断知识 YAML 文件 | 技能加载机制 |
| 提示词 | 领域专用提示词模板 | 提示词注入基础设施 |

`LocalToolExecutor` 支持脱离 Go 后端的独立部署 — 智能体可直接对接 Kubernetes、Prometheus 及其他 API。

---

## 关键设计决策

| 决策 | 理由 |
|------|------|
| 选择 LangGraph 而非工作流引擎（Temporal、Dagster） | 轻量级、LLM 原生的状态管理；无外部依赖 |
| 通过 HTTP 代理执行工具 | 在 Go 后端集中处理认证、审计和限流；智能体不持有集群凭据 |
| 确认机制作为图暂停 | 在异步用户审批过程中保持会话状态；无需轮询 |
| 按工具设置 Token 预算 | 防止过大的工具结果消耗上下文；工具级别的限制匹配预期输出大小 |
| LLM 压缩优先于硬截断 | 保留语义内容；硬截断是备用策略，而非主要策略 |
| 确定性 + LLM 路由 | 对常见场景使用快速模式匹配；仅对歧义请求使用 LLM |
| 选择时基于角色过滤工具 | 工具只定义一次；访问控制与工具逻辑正交 |

---

## 项目结构

```
crater-agent/
  crater_agent/
    agent/          # ReAct 图、状态、提示词、压缩
    agents/         # 专用智能体角色（规划器、探索器、执行器等）
    orchestrators/  # 单智能体与多智能体编排
    tools/          # 工具定义、执行器、选择器
    skills/         # 诊断知识 YAML 文件
    memory/         # 会话历史管理
    llm/            # LLM 客户端工厂与分词器
    pipeline/       # 批处理流水线（GPU 审计、运维报告）
    eval/           # 评估运行器与指标
    config.py       # 设置与配置
    app.py          # FastAPI 应用入口
  crater_bench/     # 基准测试场景与 Mock 响应
  config/           # 运行时配置文件
  dataset/          # 数据收集与转换
  tests/            # 测试套件
```

---

## 文档索引

### 系统设计
- [architecture.md](architecture.md) — 系统层次、数据流、状态管理、安全机制
- [memory-context.md](memory-context.md) — 对话历史、上下文注入、消息压缩
- [tools.md](tools.md) — 工具声明、执行后端、角色过滤、结果处理
- [skills.md](skills.md) — 诊断知识 YAML 注入

### 智能体角色
- [single-agent.md](single-agent.md) — ReAct 循环（基础构建模块）
- [coordinator.md](coordinator.md) — 请求路由与 MAS 流程控制
- [planner.md](planner.md) — 调查计划分解
- [explorer.md](explorer.md) — 证据收集（只读工具）
- [executor.md](executor.md) — 需确认的写操作
- [verifier.md](verifier.md) — 质量门控与证据验证
- [guide-general.md](guide-general.md) — 帮助与对话智能体
- [ticket-agent.md](ticket-agent.md) — 工单智能体基类（可扩展框架）
- [approval-agent.md](approval-agent.md) — 作业锁定审批（工单智能体实例）
- [inspection-pipeline.md](inspection-pipeline.md) — 定时集群巡检与报告

### 工程实践
- [evaluation.md](evaluation.md) — 基准测试工具、场景、指标、数据收集

### 规格说明
- [../../specs/agent-approval-hook.md](../../specs/agent-approval-hook.md) — 审批智能体设计规格

---

*Mops 是 Crater 智算平台的一部分。有关整体平台架构，请参阅主项目文档。*
