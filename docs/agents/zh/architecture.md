# Mops 系统架构

本文档描述了 Mops 多智能体系统的完整架构，涵盖数据流、状态管理、工具执行、token 管理和安全机制。

---

## 1. 系统层次

系统由五个层次组成，从面向用户到基础设施依次为：

```
┌──────────────────────────────────────────────────────────────┐
│  (1) 应用层                                                   │
│  FastAPI 端点、SSE 流式传输、请求路由                            │
├──────────────────────────────────────────────────────────────┤
│  (2) 编排层                                                   │
│  单智能体 ReAct / 多智能体协调器流水线                            │
├──────────────────────────────────────────────────────────────┤
│  (3) 智能体层                                                  │
│  Planner、Explorer、Executor、Verifier、Approval 等             │
├──────────────────────────────────────────────────────────────┤
│  (4) 工具与知识层                                               │
│  工具定义、执行器、选择器、诊断技能                                │
├──────────────────────────────────────────────────────────────┤
│  (5) 基础设施层                                                 │
│  Go 后端 (K8s、DB、Prometheus)，本地 kubectl/PromQL              │
└──────────────────────────────────────────────────────────────┘
```

### 各层职责

| 层次 | 职责 | 关键文件 |
|------|------|----------|
| 应用层 | HTTP 路由、请求解析、SSE 发送 | `app.py` |
| 编排层 | 智能体生命周期、阶段转换、状态管理 | `orchestrators/single.py`, `orchestrators/multi.py` |
| 智能体层 | 领域特定推理、工具选择、输出格式化 | `agents/*.py` |
| 工具与知识层 | 工具模式定义、执行、角色过滤、诊断知识 | `tools/`, `skills/` |
| 基础设施层 | 实际的 K8s/Prometheus/DB 操作、认证、审计 | Go 后端, `tools/local_executor.py` |

---

## 2. 数据流

### 2.1 单智能体请求流程

```
Go 后端                      crater-agent                    Go 后端 (工具)
    │                            │                                │
    │  POST /chat (SSE)          │                                │
    ├───────────────────────────>│                                │
    │                            │  build_history_messages()      │
    │                            │  build_system_prompt()         │
    │                            │  create_agent_graph()          │
    │                            │                                │
    │  SSE: agent_status         │  ┌─── ReAct 循环 ───┐         │
    │<───────────────────────────│  │                   │         │
    │                            │  │  LLM 思考         │         │
    │  SSE: tool_call_started    │  │  ↓                │         │
    │<───────────────────────────│  │  选择工具          │         │
    │                            │  │  ↓                │         │
    │                            │  │  执行 ────────────│────────>│
    │                            │  │  ↓                │         │
    │  SSE: tool_call_result     │  │  观察结果          │<────────│
    │<───────────────────────────│  │  ↓                │         │
    │                            │  │  LLM 再次思考     │         │
    │                            │  │  ↓                │         │
    │                            │  │  无更多工具调用    │         │
    │  SSE: agent_response       │  └───────────────────┘         │
    │<───────────────────────────│                                │
    │  SSE: done                 │                                │
    │<───────────────────────────│                                │
```

### 2.2 多智能体请求流程

```
Go 后端                      crater-agent
    │                            │
    │  POST /chat                │
    ├───────────────────────────>│
    │                            │
    │                      IntentRouter (确定性路由)
    │                            │
    │                      Coordinator (LLM 路由)
    │                        ┌───┴───┐
    │                   guide/general  diagnostic
    │                        │         │
    │                   [智能体]    Planner (LLM)
    │                        │         │
    │                   响应结果    PlanArtifact
    │                                  │
    │                             Explorer (工具循环)
    │                                  │
    │                          ObservationArtifact
    │                                  │
    │                            Executor (工具循环)
    │                                  │
    │                          ExecutionArtifact
    │                                  │
    │                             Verifier (LLM)
    │                                  │
    │                         通过 / 有风险 / 信息不足
    │                              │
    │                        Coordinator 决策：
    │                        回退重试或最终输出
    │                              │
    │  SSE: final_answer           │
    │<─────────────────────────────│
```

### 2.3 任务智能体请求流程（如审批）

```
Go 后端                      crater-agent
    │                            │
    │  POST /evaluate/approval   │
    │  （同步调用，非 SSE）        │
    ├───────────────────────────>│
    │                            │
    │                      ApprovalAgent
    │                      create_agent_graph()
    │                      （受限工具白名单）
    │                            │
    │                      ReAct 循环（最多 8 次工具调用）
    │                      ↓
    │                      提取裁决 JSON
    │                      ↓（必要时回退）
    │                      BaseRoleAgent.run_json()
    │                            │
    │  JSON: ApprovalEvalResponse│
    │<───────────────────────────│
```

---

## 3. 状态管理

### 3.1 单智能体状态 (`CraterAgentState`)

```python
class CraterAgentState(MessagesState):
    context: dict           # 来自 Go 后端的用户、页面、权限信息
    tool_call_count: int    # 安全限制计数器
    attempted_tool_calls: dict  # 基于 (工具, 参数) 签名的去重
    pending_confirmation: dict | None  # 暂停图等待用户确认
    force_limit_reached: bool  # 工具限制触发标志
    trace: list[dict]       # 审计轨迹（仅追加）
```

状态仅在单轮对话中存在。单智能体模式下无跨轮次持久化。

### 3.2 多智能体状态 (`MASState`)

```
MASState
  ├── goal: GoalArtifact
  │     user_message, actor_role, page_context, routing
  ├── observation: ObservationArtifact
  │     evidence[], facts[], open_questions[], stage_complete
  ├── plan: PlanArtifact
  │     steps[], candidate_tools[], risk
  ├── execution: ExecutionArtifact
  │     actions[], results[], summary
  ├── actions: List[MultiAgentActionItem]
  │     pending / completed / awaiting_confirmation
  ├── flow_control
  │     loop_round, no_progress_count, stop_reason
  ├── usage_summary: MultiAgentUsageSummary
  │     llm_calls, tokens, tool_calls, evidence_items
  └── runtime_config: MASRuntimeConfig
        lead_max_rounds=8, subagent_max_iterations=25
```

MASState 支持工作流检查点，用于跨轮次持久化（确认流程）。

---

## 4. 工具执行

### 4.1 执行流水线

```
LLM 发出 tool_call
  ↓
is_tool_allowed_for_role(agent_role, tool_name)?
  ├─ 否 → 错误：违反工具策略
  └─ 是
      ↓
is_actor_allowed_for_tool(actor_role, tool_name)?
  ├─ 否 → 错误：需要管理员权限
  └─ 是
      ↓
LocalToolExecutor.supports(tool_name)?
  ├─ 是 → 本地执行 (kubectl, PromQL, web)
  └─ 否 → GoBackendToolExecutor (HTTP POST)
              ↓
          Go 后端验证、执行并返回结果
              ↓
          状态：success | error | confirmation_required
```

### 4.2 工具分类

| 类别 | 数量 | 执行方式 | 认证 | 示例 |
|------|------|----------|------|------|
| 诊断（只读） | ~10 | Go 后端 | 用户/管理员 | `get_job_detail`, `diagnose_job` |
| 指标（只读） | ~15 | Go 后端 | 用户/管理员 | `query_job_metrics`, `check_quota` |
| 管理员只读 | ~20 | Go 后端 | 仅管理员 | `list_cluster_nodes`, `get_node_detail` |
| K8s 直连 | ~15 | 本地 kubectl | 仅管理员 | `k8s_list_pods`, `k8s_describe_resource` |
| Prometheus 直连 | 1 | 本地 HTTP | 仅管理员 | `prometheus_query` |
| 写操作（需确认） | ~15 | Go 后端 | 用户+确认 | `stop_job`, `cordon_node` |
| 审批 | 1 | Go 后端 | 系统 | `get_approval_history` |

### 4.3 确认流程

写操作工具返回 `confirmation_required` 而非立即执行：

```
智能体调用 stop_job("my-training")
  ↓
Go 后端返回：
  { status: "confirmation_required",
    confirm_id: "abc123",
    description: "停止作业 my-training",
    risk_level: "medium" }
  ↓
编排器暂停，将确认请求返回前端
  ↓
前端向用户展示确认对话框
  ↓
用户批准
  ↓
携带 confirm_id 和批准状态发起恢复调用
  ↓
Go 后端执行操作
  ↓
结果返回智能体，循环继续
```

---

## 5. Token 管理

上下文窗口是有限资源。系统在四个层面对其进行管理：

### 5.1 输入历史

```
来自 Go 后端的对话历史（逆时间序）
  ↓
build_history_messages(max_tokens=4000)
  ↓
将工具结果截断为 160 字符（首尾拼接）
  ↓
token 预算耗尽时停止加载
  ↓
恢复为时间正序
```

### 5.2 工具结果预算

每个工具有独立的单次结果 token 预算：

| 工具 | 预算 | 原因 |
|------|------|------|
| `get_job_logs` | 4000 | 日志可能非常长 |
| `diagnose_job` | 4000 | 结构化诊断输出 |
| `get_job_detail` | 3000 | 作业元数据 |
| `prometheus_query` | 2000 | 时间序列数据 |
| 默认 | 3000 | - |

超出预算的结果先通过 LLM 语义提取，再以硬截断作为兜底方案。

### 5.3 主动压缩

在达到上下文限制之前：

```
estimated_tokens = count_message_tokens(all_messages)
available = max_context_tokens - tool_schema_budget(8000) - response_budget(4000)

if estimated_tokens > available:
  try: 基于 LLM 的压缩（保留近期消息，摘要较早消息）
  except: 硬截断（保留系统提示 + 最近 6 条消息）
```

### 5.4 被动恢复

当出现 `BadRequestError("context_length_exceeded")` 时：

```
捕获错误
  → compact_messages_with_llm()
  → 使用压缩后的消息重试 LLM 调用
  → 如果仍然失败：_compact_messages_for_retry()（硬截断）
  → 再重试一次
```

---

## 6. 安全机制

### 6.1 工具策略（智能体级别）

智能体角色定义了可调用的工具范围：
- Explorer：仅限只读工具（硬编码检查）
- Executor：可使用读写工具
- Approval：固定白名单，共 8 个工具
- General/Guide/Verifier/Planner：无工具（纯 LLM）

### 6.2 操作者策略（用户级别）

用户角色定义了可见的工具范围：
- 管理员：所有工具
- 普通用户：约 30 个安全工具（无集群管理、无节点操作）

### 6.3 确认屏障（操作级别）

写操作工具（停止、删除、创建、封锁、驱逐等）在执行前始终需要用户确认。图会暂停运行，必须获得明确批准后才能继续。

### 6.4 速率限制（系统级别）

针对自动化智能体（如审批钩子）：
- 令牌桶：每分钟最多 N 次调用
- 并发信号量：最多 M 个并发评估
- 熔断器：连续失败触发冷却期
- 超时：总评估时间设上限

### 6.5 异常恢复（进程级别）

Go 后端中所有智能体评估路径均使用 `defer recover()` 包装：

```go
func() {
    defer func() {
        if r := recover(); r != nil {
            klog.Errorf("agent panic recovered: %v", r)
        }
    }()
    // 智能体评估调用
}()
```

智能体故障绝不会导致主应用崩溃或阻塞用户操作。

---

## 7. 配置

### 7.1 LLM 客户端 (`config/llm-clients.json`)

可为每个智能体角色配置多个 LLM 提供商：

```json
{
  "default": { "model": "...", "base_url": "...", "temperature": 0.1 },
  "planner": { "model": "...", "temperature": 0.0 },
  "explorer": { "model": "...", "max_tokens": 2048 }
}
```

### 7.2 MAS 运行时 (`config/agent-runtime.json`)

多智能体编排的护栏参数：

```json
{
  "lead_max_rounds": 8,
  "subagent_max_iterations": 25,
  "no_progress_rounds": 2,
  "max_actions_per_round": 4
}
```

### 7.3 平台运行时 (`config/platform-runtime.yaml`)

平台特定的端点和访问配置：

```yaml
kubernetes:
  kubeconfig_path: /path/to/kubeconfig
prometheus:
  endpoint: http://prometheus:9090
harbor:
  endpoint: https://harbor.example.com
```

---

## 8. 评估框架

### 8.1 基于场景的基准测试

每个场景定义：
- 用户查询和页面上下文
- 可用工具和模拟响应
- 预期结果：期望的工具调用、根因关键词、建议

### 8.2 评估指标

| 指标 | 测量内容 |
|------|----------|
| 工具选择 F1 | 智能体是否调用了正确的工具？ |
| 根因命中率 | 智能体是否识别了正确的根本原因？ |
| 建议相关性 | 建议是否恰当且安全？ |
| 权限合规性 | 需确认的工具是否触发了确认流程？ |
| 效率比 | 实际工具调用数 vs 最优工具调用数 |

### 8.3 执行模式

| 模式 | 数据来源 | 使用场景 |
|------|----------|----------|
| 快照模式 | 模拟 JSON 响应 | CI 回归测试、模型对比 |
| 线上只读模式 | 真实后端，仅只读 | 冒烟测试、工具验证 |

---

## 代码参考

| 组件 | 文件 |
|------|------|
| ReAct 图 | `crater_agent/agent/graph.py` |
| 智能体状态 | `crater_agent/agent/state.py` |
| 系统提示词 | `crater_agent/agent/prompts.py` |
| 消息压缩 | `crater_agent/agent/compaction.py` |
| 基础智能体类 | `crater_agent/agents/base.py` |
| 工具定义 | `crater_agent/tools/definitions.py` |
| 工具执行器 | `crater_agent/tools/executor.py` |
| 工具选择器 | `crater_agent/tools/tool_selector.py` |
| 本地执行器 | `crater_agent/tools/local_executor.py` |
| 单智能体编排器 | `crater_agent/orchestrators/single.py` |
| 多智能体编排器 | `crater_agent/orchestrators/multi.py` |
| MAS 状态 | `crater_agent/orchestrators/state.py` |
| 技能加载器 | `crater_agent/skills/loader.py` |
| 会话记忆 | `crater_agent/memory/session.py` |
| 配置 | `crater_agent/config.py` |
| 应用入口 | `crater_agent/app.py` |
| 评估运行器 | `crater_agent/eval/runner.py` |
| 评估指标 | `crater_agent/eval/metrics.py` |
