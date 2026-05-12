# 第 4 章 MOps 框架在 Crater 平台上的实现

本章详细介绍 MOps 框架在 Crater 智算平台上的工程实现，涵盖技术栈选型、代码组织结构、编排层实现、Agent 层实现、工具层实现、上下文管理、前端交互与评测基础设施。

## 4.1 技术栈与代码组织

### 4.1.1 整体技术栈

Crater + MOps 的完整技术栈如表 4-1 所示。

> **表 4-1** Crater + MOps 系统技术栈

| 层次 | 技术选型 | 版本 | 说明 |
|---|---|---|---|
| 平台底层 | Kubernetes + Volcano | 1.28 + 1.9 | 异构作业调度与管理 |
| 容器运行时 | containerd + NVIDIA Container Runtime | - | GPU 容器化 |
| 监控 | Prometheus + Grafana + DCGM Exporter | - | 指标采集与可视化 |
| 存储 | NFS + Ceph RBD | - | 共享数据集与用户工作目录 |
| 后端服务 | Go + Gin + GORM + PostgreSQL | 1.21+ | 作业、用户、资源管理 API |
| 智能体核心 | Python + LangChain + LangGraph | 3.11 + 0.3 + 0.2 | Agent 编排与 LLM 交互 |
| LLM 主模型 | Qwen3-235B-Thinking | - | 深度推理（Executor/Verifier） |
| LLM 辅助模型 | Qwen3-Flash | - | 规划/探索/路由（Planner/Explorer/IntentRouter） |
| 评测 LLM | Kimi-K2.6 + tongyi-xiaomi-analysis-flash | - | 任务评测 + 对话评测 |
| 评测框架 | FastAPI + pytest-asyncio | - | Mock 后端 + 异步评测运行器 |
| 前端框架 | React 18 + TanStack Router + Vite | - | 用户界面 |

### 4.1.2 代码组织结构

MOps 智能体代码位于 `crater-agent/crater_agent/` 目录下，核心模块组织如下：

```
crater_agent/
├── agent/                  # 单智能体 ReAct 基础设施
│   ├── graph.py            # LangGraph 状态图定义
│   ├── state.py            # CraterAgentState 状态模型
│   ├── prompts.py          # 分层 System Prompt 模板
│   └── compaction.py       # 会话历史压缩与证据摘取
├── agents/                 # 9 种角色 Agent 实现
│   ├── base.py             # BaseRoleAgent 抽象基类
│   ├── coordinator.py      # 协调器
│   ├── planner.py          # 规划器
│   ├── explorer.py         # 探索器
│   ├── executor.py         # 执行器
│   ├── verifier.py         # 验证器
│   ├── approval.py         # 审批器
│   └── general.py / guide.py  # 通用/向导 Agent
├── orchestrators/          # 编排器
│   ├── single.py           # SingleAgentOrchestrator (ReAct)
│   ├── multi.py            # MultiAgentOrchestrator (Plan-Execute)
│   ├── state.py            # MAS 多智能体状态定义
│   └── intent_router.py    # IntentRouter 意图路由
├── tools/                  # 工具层（79 个工具）
│   ├── base.py             # CompositeToolExecutor 基类
│   ├── job_diagnosis.py    # 作业诊断工具集
│   ├── metrics_queue.py    # 指标与队列工具集
│   ├── cluster_mgmt.py     # 集群管理工具集
│   ├── storage.py          # 存储与数据工具集
│   ├── gpu_network.py      # GPU 与网络工具集
│   ├── job_ops.py          # 作业操作工具集
│   ├── audit_approval.py   # 审计与审批工具集
│   └── k8s_native.py       # K8s 原生工具集
├── models/                 # LLM 客户端工厂
│   └── client_factory.py   # ModelClientFactory (Thinking/Flash 路由)
├── config/                 # 配置管理
│   └── settings.py         # 全局配置与 Prompt 配置
├── crater_bench/           # Crater-Bench 评测基准
│   ├── scenarios/          # 30+ 标准化评测场景
│   ├── mock_backend/       # Mock 后端服务
│   ├── evaluator/          # 离线评测器
│   └── runner.py           # 评测运行器
└── utils/                  # 通用工具
    ├── trace_recorder.py   # 审计链记录
    └── token_counter.py    # Token 计数与预算管理
```

## 4.2 编排层实现

### 4.2.1 LangGraph 状态图

MOps 的编排层基于 LangGraph 的 `StateGraph` 实现。核心状态图包含以下节点：

```
[START] → IntentRouter → ModeRouter
                            ├── ReAct Mode → AgentLoop → [END]
                            ├── Plan-Execute Mode → Planner → Explorer → Executor → [END]
                            └── Plan-Execute-Verify Mode → Planner → Explorer
                                                              → Executor → ConfirmNode
                                                              → Verifier → [END]
```

**IntentRouter 节点**接收用户输入，调用 Flash LLM 进行任务分类，输出类别标签（query/diagnosis/ops/submission/remedy）与置信度。

**ModeRouter 节点**（条件边）根据分类结果路由到对应编排模式。

**AgentLoop 节点**（ReAct 模式）：LangGraph 的 `agent_loop` 内置实现——LLM 思考 → 工具调用 → 观察结果 → 继续思考，循环直到 LLM 输出 Final Answer。

**Planner 节点**（Plan-Execute 模式）：Flash LLM 根据任务类型和 System Prompt 生成结构化的 JSON 调查计划（步骤列表，每步包含工具名、参数和目的描述）。

**Explorer 节点**：遍历 Planner 生成的步骤列表，依次调用工具并收集结果。每次工具调用后判断是否触发证据摘取条件（累计返回长度超过阈值）。

**Executor 节点**：Thinking LLM 基于 Explorer 收集的结构化证据进行深度推理，生成最终输出。输出格式根据任务类型有所不同：诊断类输出包含根因判断、关键证据表、建议下一步；查询类输出包含数据汇总表、分析结论。

**ConfirmNode**：仅在上文需要用户确认写操作时激活，通过 LangGraph 的 `interrupt` 机制暂停图执行，向前端推送确认请求。

**Verifier 节点**：在写操作执行后运行，重新查询操作影响的对象状态，与操作前快照对比，输出验证结果（通过/异常/需人工介入）。

### 4.2.2 IntentRouter 算法

IntentRouter 的核心逻辑实现如下（伪代码）：

```
def classify(user_input, page_context):
    # Phase 1: 关键词快速匹配
    kws_diagnosis = ["失败", "OOM", "报错", "超时", "为什么", "怎么回事", "原因"]
    kws_query = ["多少", "查一下", "有没有", "状态", "列表", "剩余"]
    kws_ops = ["检查", "审计", "健康", "是否正常", "巡检"]
    kws_submission = ["提交", "创建", "申请", "帮我跑", "能不能跑"]
    kws_remedy = ["摘除", "重启", "删除", "扩容", "修改配额", "驱逐"]

    scores = count_matches(user_input, [kws_diagnosis, kws_query, 
               kws_ops, kws_submission, kws_remedy])
    
    if max(scores) >= 2 and max(scores) > second_max(scores):
        return argmax(scores), confidence=high
    
    # Phase 2: LLM 语义分类 (Flash 模型)
    prompt = CLASSIFICATION_PROMPT.format(user_input, page_context)
    result = flash_llm.invoke(prompt)
    return result.category, result.confidence
```

其中 Phase 1 纯字符串匹配，零延迟、零成本；Phase 2 仅当关键词匹配不明确时触发，由 Flash 模型以极低成本完成语义消歧。

### 4.2.3 模型客户端工厂

`ModelClientFactory` 是 LLM 路由的核心实现：

```
class ModelClientFactory:
    THINKING_MODELS = ["qwen3-235b-thinking", "deepseek-v4-pro"]
    FLASH_MODELS = ["qwen3-flash", "qwen3-turbo"]
    
    @classmethod
    def get_client(cls, role: AgentRole) -> ChatModel:
        if role.requires_deep_reasoning:
            return cls._create_thinking_client()
        else:
            return cls._create_flash_client()
```

`requires_deep_reasoning` 属性在各 Agent 角色定义中声明。当前实现中，Executor 和 Verifier 使用 Thinking 模型，其余角色使用 Flash 模型。工厂还支持配置 fallback 链：当主模型不可用时自动切换备用模型。

## 4.3 Agent 层实现

### 4.3.1 BaseRoleAgent 抽象

所有角色智能体继承自 `BaseRoleAgent`，其核心接口如下：

```python
class BaseRoleAgent:
    role_name: str           # 角色名 (如 "planner", "executor")
    system_prompt: str       # 三层合并后的 System Prompt
    llm: ChatModel           # LLM 客户端 (由 ModelClientFactory 分配)
    tools: List[BaseTool]    # 可用工具白名单
    
    def build_messages(self, state: CraterAgentState) -> List[Message]:
        """构建发送给 LLM 的消息序列"""
        ...
    
    def invoke(self, state: CraterAgentState) -> AgentResult:
        """调用 LLM 并返回结果"""
        ...
```

System Prompt 由三层合并而成：L1（角色定义与行为准则）从 `prompts.py` 加载；L2（智算平台领域知识）从 `config/domain_knowledge.yaml` 加载；L3（任务上下文）在运行时根据 IntentRouter 的分类结果动态注入。

### 4.3.2 关键角色 Prompt 设计

**Planner 的 System Prompt 核心片段：**

```
你是一个专门为智算平台运维任务生成调查计划的规划器。
给定用户的运维问题和任务类型，你需要生成一个结构化的调查步骤列表。

输出 JSON 格式：
{
  "plan": [
    {"step": 1, "tool": "get_job_detail", "args": {"job_name": "..."}, "purpose": "获取作业基本信息"},
    {"step": 2, "tool": "get_job_events", "args": {"job_name": "..."}, "purpose": "查看作业事件时间线"},
    ...
  ]
}

规则：
- 每个步骤必须对应一个已注册的工具名称，不允许虚构工具。
- 步骤按逻辑顺序排列（先基本信息，后深入调查）。
- 诊断类任务至少包含：基础信息查询 → 事件/日志查询 → 诊断/分析。
- 查询类任务步骤数不超过 3。
```

**Executor 的 System Prompt 核心片段：**

```
你是一个智算平台运维诊断专家。基于 Explorer 收集的证据，执行深度推理并生成最终报告。

输出必须包含以下结构化部分：
## 诊断总结
### 结论（根因判断）
### 关键证据（表格，含"证据项 | 详情"）
### 建议下一步（编号列表，每项一行）

领域知识提醒：
- GPU OOM 的常见原因：batch size 过大、模型参数量超出显存、数据加载并行度不当
- NCCL 超时可能是级联效应：一个 worker OOM → NCCL 超时 → 其他 worker 被拖垮
- 节点驱逐（Evicted）通常由于资源压力：DiskPressure、MemoryPressure、PIDPressure
- IB/RDMA 错误通常与物理链路相关：completion error、retransmit rate、link flap
```

### 4.3.3 角色协作流程

在典型的多轮诊断对话场景中，角色协作如下：

1. **IntentRouter** 将用户输入分类为 "diagnosis"，置信度 0.95
2. **Coordinator** 根据类别激活 Plan-Execute 模式
3. **Planner**（Flash）生成 4 步调查计划（约 0.5s，消耗 ~200 token）
4. **Explorer**（Flash）依次调用 3 个只读工具收集数据（每次工具调用约 0.3s）
5. **Explorer** 调用证据摘取：将原始工具输出（~3000 token）压缩为结构化摘要（~800 token）
6. **Executor**（Thinking）基于压缩后的证据进行深度推理生成诊断报告（约 3s，消耗 ~1500 token）

整个流程总计约 5-8 次 LLM 调用（其中仅最后 1 次使用 Thinking 模型），累计 token 消耗在 15000-20000 范围。

## 4.4 工具层实现

### 4.4.1 Composite 模式工具执行器

工具层采用 Composite 模式实现双路由透明执行。核心类设计如下：

```python
class CompositeToolExecutor:
    """统一工具执行接口，内部根据工具元数据选择执行路由"""
    
    def execute(self, tool_name: str, args: dict,
                auth_context: AuthContext) -> ToolResult:
        tool_meta = self.registry.get(tool_name)
        
        # 权限检查
        if not self._check_permission(auth_context, tool_meta):
            raise PermissionDeniedError(tool_name, auth_context.role)
        
        # 路由执行
        if tool_meta.execution_mode == "local":
            return self._execute_local(tool_name, args)
        else:
            return self._execute_remote(tool_name, args, auth_context)
    
    def _execute_local(self, tool_name: str, args: dict) -> ToolResult:
        """进程内执行：Mock 数据或本地函数调用"""
        ...
    
    def _execute_remote(self, tool_name: str, args: dict,
                        auth: AuthContext) -> ToolResult:
        """HTTP 调用 Crater 后端 API"""
        ...
```

### 4.4.2 核心工具列表

> **表 4-2** MOps 核心工具一览

| 类别 | 工具名称 | 功能描述 | 路由 |
|---|---|---|---|
| **作业诊断** | `get_job_detail` | 获取作业基本信息（状态、资源、镜像） | 本地 |
| | `get_job_events` | 获取作业 Kubernetes 事件时间线 | 本地 |
| | `get_job_logs` | 获取作业容器日志（支持 tail_lines） | 本地 |
| | `diagnose_job` | 综合诊断（聚合 detail+events+logs） | 远程 |
| | `diagnose_distributed_job_network` | 分布式作业网络诊断 | 远程 |
| | `search_similar_failures` | 搜索历史相似故障案例 | 本地 |
| **指标队列** | `get_cluster_health_report` | 集群健康度报告 | 远程 |
| | `analyze_queue_status` | 分析队列排队状态 | 本地 |
| | `k8s_top_nodes` | 节点资源占用排名 | 远程 |
| | `k8s_top_pods` | Pod 资源占用排名 | 远程 |
| | `check_quota` | 检查用户资源配额 | 远程 |
| | `get_realtime_capacity` | 实时可用资源查询 | 远程 |
| **集群管理** | `get_node_detail` | 节点详情（状态、标签、资源） | 远程 |
| | `cordon_node` | 封锁节点（禁止新调度） | 远程* |
| | `uncordon_node` | 解封节点 | 远程* |
| **GPU/网络** | `get_node_network_summary` | 节点网络健康摘要 | 远程 |
| | `get_gpu_metrics` | GPU 指标查询 | 远程 |
| **作业操作** | `submit_job` | 提交新作业 | 远程* |
| | `stop_job` | 停止运行中的作业 | 远程* |
| | `get_job_templates` | 获取可用作业模板 | 本地 |
| | `list_available_images` | 列出可用容器镜像 | 本地 |
| | `get_resource_recommendation` | 资源推荐 | 远程 |

> *标注的工具为写操作，需要用户确认或管理员审批。

## 4.5 上下文管理实现

### 4.5.1 证据摘取算法

```
def extract_evidence(tool_outputs: List[ToolResult]) -> str:
    prompt = EVIDENCE_EXTRACTION_PROMPT.format(
        raw_outputs=json.dumps(tool_outputs, indent=2)
    )
    # 使用 Flash 模型以低成本执行证据摘取
    return flash_llm.invoke(prompt, max_tokens=1000)
```

证据摘取 Prompt 引导 Flash LLM 从原始工具输出中提取：
- 关键数值字段（如显存使用量、作业状态码、错误关键词）
- 因果线索（如事件序列中的时间关联）
- 异常标记（如状态值异常、阈值超限）
- 无关内容的剔除（如 verbose 日志中的重复行、标准输出模板）

### 4.5.2 Token 预算与会话压缩

MOps 维护全局 token 计数器和三级预算：
- **L1 预算限制**（6000 token）：警告阈值，触发证据摘取（已在上文实现）
- **L2 预算限制**（12000 token）：触发历史压缩——保留最近 3 轮完整交互，将更早轮次替换为摘要
- **L3 预算限制**（20000 token）：硬限制，丢弃最早轮次

Token 计数通过 `tiktoken` 库实时估算（计算消息列表的总 token 数），不依赖 LLM API 返回的实际计数（存在延迟）。

## 4.6 前端与交互实现

MOps 前端基于 React 18 + TanStack Router 构建，核心交互组件包括：

- **ChatPanel**：类 ChatGPT 的对话界面，支持流式（SSE）输出。流式输出包含三类事件：`thought`（智能体推理过程，以折叠式展示）、`tool_call`（工具调用，含工具名和参数）、`tool_result`（工具结果，折叠展示）、`final_answer`（最终输出）。
- **AdminConsole**：管理员控制台，提供操作审批流管理、审计日志查询、Agent 运行监控。
- **ConfirmDialog**：确认对话框，展示操作描述、影响范围、风险评估、Pre-check 结果，用户可选择 "确认执行" 或 "取消"。
- **JobTemplateWizard**：作业提交向导，引导用户选择模板、镜像、资源配置，智能体在后台辅助检查配额、推荐资源。

SSE 流的实现基于 FastAPI 的 `StreamingResponse`，Agent 在每次 LLM 调用或工具执行后向前端推送事件。

## 4.7 Crater-Bench 评测基础设施

为支持离线可复现评测，Crater-Bench 提供了一套完整的 Mock 后端与评测运行器。

**场景定义**：每个评测场景是一个 YAML 文件，包含场景元数据（ID、类别、难度）、用户输入（支持单轮和多轮对话）、Mock 工具输出（每个工具调用的预期返回值）、评测标准（期望的工具调用序列、根因关键词、建议主题）。

**Mock 后端**：基于 FastAPI 实现，提供与 Crater 后端相同的 API 端点，但返回的是预定义的场景数据而非真实平台数据。Mock 后端在每次评测启动时从场景 YAML 加载数据。

**离线评测器**：在评测运行结束后，对 Agent 的完整响应进行多维度评分。评分由两部分组成：规则评分（工具选择 F1、根因命中、权限合规、确认完成）和 LLM-as-Judge 评分（由外部评测模型 Kimi-K2.6 对诊断准确性、建议质量、推理连贯性进行打分，由 tongyi-xiaomi-analysis-flash 对多轮对话的意图理解、完整性和满意度打分）。

**评测指标**：

> **表 4-3** Crater-Bench 评测指标体系

| 指标 | 计算方式 | 权重 |
|---|---|---|
| 工具选择 F1 | 预测工具集与期望工具集的 F1 | 19.35% |
| 根因命中 | 是否命中期望的根因关键词 | 19.35% |
| 建议质量 | 建议是否与期望主题一致 | 12.90% |
| 权限合规 | 是否未调用无权限的工具 | 12.90% |
| 效率得分 | 工具调用数量与效率 | 10.32% |
| Token 效率 | Token 消耗与预算的比例 | 4.52% |
| 延迟效率 | 耗时与预算的比例 | 1.29% |
| 任务链质量 | 推理链的连贯性（LLM-as-Judge） | 19.35% |
