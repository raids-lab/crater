# 工单智能体框架

> 自动化工单/订单评估的基类 -- 只需极少代码即可支持新的工单类型。

---

## 问题

平台运维涉及多种需要评估的工单类型：作业锁定审批、配额变更请求、数据集访问审核、节点维护窗口等。每种工单类型需要各自的领域逻辑，但评估基础设施（运行 ReAct 循环 -> 提取裁决 -> 回退 -> 错误处理 -> 审计）是完全相同的。

如果不进行抽象，每个工单智能体都要重复约 150 行 ReAct 循环管理、回退逻辑和错误处理代码。

---

## 设计

`TicketAgent` 是一个抽象基类，提供完整的评估流水线。子类只需定义**五个领域特定方法**：

```
TicketAgent (基类)
  │
  ├── ReAct 循环执行              ← 共享
  ├── 裁决提取                    ← 共享
  ├── 回退 (BaseRoleAgent)        ← 共享
  ├── 错误处理 (永不抛出异常)      ← 共享
  ├── 轨迹收集                    ← 共享
  │
  └── 子类定义：
        ├── allowed_tools()        → 绑定哪些工具
        ├── system_prompt()        → 领域评估规则
        ├── build_user_message()   → 请求 → LLM 输入
        ├── extract_verdict()      → LLM 输出 → 结构化结果
        └── default_verdict()      → 全部失败时的安全回退
```

### 类层次结构

```
TicketAgent[TRequest, TVerdict]  (抽象, 泛型)
  │
  ├── ApprovalAgent              作业锁定审批评估
  │     TRequest = ApprovalEvalRequest
  │     TVerdict = ApprovalEvalResponse
  │
  ├── QuotaAgent (未来)           资源配额变更评估
  │     TRequest = QuotaEvalRequest
  │     TVerdict = QuotaEvalResponse
  │
  ├── DatasetAccessAgent (未来)   数据集访问审核
  │
  └── MaintenanceAgent (未来)     节点维护调度
```

---

## 创建新的工单智能体

### 第一步：定义请求和响应模型

```python
class QuotaEvalRequest(BaseModel):
    account_id: int
    account_name: str
    requested_gpu: int
    requested_cpu: int
    reason: str

class QuotaEvalResponse(BaseModel):
    verdict: str = "escalate"  # "approve" | "adjust" | "escalate"
    confidence: float = 0.5
    reason: str = ""
    adjusted_gpu: int | None = None
    adjusted_cpu: int | None = None
    trace: list[dict[str, Any]] = []
```

### 第二步：实现智能体

```python
class QuotaAgent(TicketAgent[QuotaEvalRequest, QuotaEvalResponse]):

    def __init__(self, **kwargs):
        super().__init__(agent_id="quota", llm_purpose="quota", **kwargs)

    def allowed_tools(self) -> list[str]:
        return ["check_quota", "get_realtime_capacity", "list_cluster_jobs"]

    def system_prompt(self) -> str:
        return "你是配额审批助手。评估用户的资源配额变更请求..."

    def build_user_message(self, request: QuotaEvalRequest) -> str:
        return f"请评估配额变更：{request.account_name} 申请 GPU={request.requested_gpu}..."

    def extract_verdict(self, text: str) -> QuotaEvalResponse | None:
        # 从 LLM 输出中解析 JSON
        ...

    def default_verdict(self, *, reason: str = "") -> QuotaEvalResponse:
        return QuotaEvalResponse(verdict="escalate", reason=reason or "转交管理员")
```

### 第三步：注册端点

```python
# app.py
@app.post("/evaluate/quota")
async def evaluate_quota(request: QuotaEvalRequest):
    agent = QuotaAgent()
    return await agent.evaluate(request)
```

### 第四步：添加 Go 后端钩子（与审批相同的模式）

就这样 -- 无需修改 ReAct 图、工具执行器或基础设施。

---

## 基类处理的内容

| 关注点 | 实现方式 |
|--------|---------|
| ReAct 循环 | `create_agent_graph` 配合 `capabilities.enabled_tools` |
| 工具执行 | `GoBackendToolExecutor`（复用） |
| Token 管理 | 图内置的压缩和工具结果预算 |
| 工具调用限制 | 达到限制时图的 `summarize_node` |
| 裁决提取 | 子类的 `extract_verdict()` 作用于最后的 AI 消息 |
| 回退 | `BaseRoleAgent.run_json()` 不绑定工具 |
| JSON 修复 | `run_json` 内置的修复循环 |
| 错误恢复 | `try/except` 包裹整个 `evaluate()` -> `default_verdict()` |
| 轨迹收集 | 从消息历史中自动收集 |
| 超时 | 调用方通过 `asyncio.wait_for()` 包裹 |

---

## 可选钩子

除了五个必需方法外，子类还可以重写：

| 钩子 | 默认值 | 何时重写 |
|------|--------|---------|
| `build_context(request)` | `{capabilities, actor: system}` | 需要请求特定的上下文（user_id、session_id） |
| `fallback_prompt()` | 通用工单评估提示词 | 需要领域特定的回退指令 |
| `set_trace(verdict, trace)` | 设置 `verdict.trace = trace` | 裁决模型有不同的轨迹字段 |
| `_parse_fallback_result(result)` | `extract_verdict(json.dumps(result))` | 回退 JSON 结构不同 |

---

## 代码

| 组件 | 文件 |
|------|------|
| 基类 | `crater_agent/agents/ticket_base.py` |
| 审批智能体 | `crater_agent/agents/approval.py` |
