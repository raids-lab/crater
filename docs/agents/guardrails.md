# Harness 限制与安全护栏

> 记录 single_agent 模式下的工具调用上限、token 预算、去重、截断等 harness 级限制。
> 最后更新: 2026-04-21

---

## 1. 工具调用上限

### 配置

```python
# config.py 代码默认值
max_tool_calls_per_turn: int = Field(default=15, description="Max tool calls in a single ReAct loop")
```

环境变量: `CRATER_AGENT_MAX_TOOL_CALLS_PER_TURN`

### 计数机制

- **计数器**: `state.tool_call_count`（CraterAgentState 字段）
- **初始化**: 每次 `SingleAgentOrchestrator.stream()` 调用时硬编码为 `0`（`single.py:44`）
- **递增**: tools_node 中每执行一个 tool_call 就 `+1`（`graph.py:445`），**包括**被去重跳过的调用
- **作用域**: 单个用户 turn 内的单次 graph invoke，**不跨 turn 累加**

### 触发行为

当 `tool_call_count >= max_tool_calls_per_turn` 时（`graph.py:590-595`）:

```python
if tool_call_count >= settings.max_tool_calls_per_turn:
    if isinstance(last_message, AIMessage) and last_message.tool_calls:
        return "summarize"   # LLM 还想调工具 → 强制总结
    return "respond"         # LLM 已经给了最终回复
```

### Summarize Node 行为

`summarize_node`（`graph.py:524-580`）:
- 注入系统提示: "[系统提示] 你已达到本轮工具调用上限，无法继续调用工具。请基于已收集到的所有工具返回结果，直接给出完整的综合分析回答。"
- 调用 LLM **不绑定任何工具**（纯文本生成）
- 如果遇到上下文超限 → 先尝试 LLM compaction → 退回 hard truncation

### Orphan Tool Calls 取消

当 should_continue 路由到 summarize 而非 tools 时，LLM 最后一次产出的 tool_calls 不会被执行。Orchestrator 在 stream 结束后将这些标记为 `cancelled`（`single.py:248-266`）:

```
前端显示: "已超过单轮工具调用上限，本次调用已取消"
状态: cancelled（非 error）
```

---

## 2. 工具去重

### 机制

`state.attempted_tool_calls`: `dict[str, int]`（tool_signature → 调用次数）

```python
tool_signature = json.dumps({"tool_name": name, "tool_args": args}, sort_keys=True)
if attempted_tool_calls.get(tool_signature, 0) >= 1:
    # 跳过，返回提示消息
    ToolMessage(content="工具 {name} 在本轮中已用相同参数调用过一次...")
```

### 作用域

- 同一次 graph invoke 内有效（`single.py:45` 每次初始化为 `{}`）
- 同名同参数的工具只执行一次
- **注意**: 去重的调用仍会递增 `tool_call_count`（`graph.py:445` 在去重检查之前递增）

---

## 3. Token 预算体系

### 3.1 会话历史预算

```python
# config.py
history_max_tokens: int = Field(default=4000)
```

`build_history_messages()`（`memory/session.py`）:
- 从最新消息开始倒序加载
- 工具结果截断: 正常 160 字符，错误 1600 字符（head+tail）
- 累计 token 达到预算时停止加载

### 3.2 单工具结果 Token 预算

```python
_DEFAULT_TOOL_TOKEN_BUDGET = 3000

_TOOL_TOKEN_BUDGETS = {
    "get_job_logs": 4000,
    "diagnose_job": 4000,
    "get_diagnostic_context": 4000,
    "get_job_detail": 3000,
    "prometheus_query": 2000,
    "query_job_metrics": 2000,
}
```

处理流程（`_build_tool_observation`，`graph.py:153-207`）:
1. 序列化结果为字符串
2. 计算 token 数 → 在预算内 → 原样返回
3. 超预算 + 有 LLM → 调用 `_extract_with_llm()`（10s 超时，语义压缩）
4. LLM 提取失败 → hard truncation（保留头尾，`_truncate_text`）

### 3.3 主动上下文压缩

```python
# config.py
max_context_tokens: int = Field(default=30000)
```

`_proactive_compact()`（`graph.py:272-306`）在每次 LLM 调用前检查:
- 预留: tool_schema_budget=8000 + response_budget=4000
- available = 30000 - 8000 - 4000 = 18000 tokens
- 如果 messages 总 token > available → 压缩
- 策略: LLM compaction 优先 → hard truncation 退回

### 3.4 反应式上下文恢复

当 LLM 调用返回 `BadRequestError("context_length_exceeded")`（`graph.py:367-383`）:
1. 尝试 `compact_messages_with_llm()`
2. 失败则 `_compact_messages_for_retry()`（保留 system + 最后 6 条消息）
3. 重试 LLM 调用

---

## 4. 工具执行超时

```python
# config.py
tool_execution_timeout: int = Field(default=30)
```

- 应用在 GoBackendToolExecutor 的 HTTP 请求超时
- LocalToolExecutor 各有独立超时（如 LLM extract 10s，code 执行 5-120s）

---

## 5. 限制汇总表

| 限制项 | 默认值 | 配置方式 | 作用域 |
|--------|-------|---------|-------|
| 工具调用上限 | 10 次/turn（.env 配置，代码默认 15） | `max_tool_calls_per_turn` | 单次 graph invoke |
| 工具去重 | 同名同参数跳过 | 硬编码 | 单次 graph invoke |
| 会话历史预算 | 4000 tokens | `history_max_tokens` | 每次 stream() |
| 单工具结果预算 | 2000-4000 tokens | `_TOOL_TOKEN_BUDGETS` | 每个工具调用 |
| 主动压缩阈值 | 18000 tokens (30000-12000) | `max_context_tokens` | 每次 LLM 调用前 |
| 工具执行超时 | 30s | `tool_execution_timeout` | 每次工具执行 |
| LLM 提取超时 | 10s | 硬编码 | 每次超预算工具结果 |
| Hard truncation | 2400 字符 | `_truncate_text` max_chars | 工具结果兜底 |

---

## 6. 代码位置

| 组件 | 文件 | 关键行 |
|------|------|-------|
| 配置定义 | `crater-agent/.../config.py` | 61-73 |
| 工具计数与上限 | `crater-agent/.../graph.py` | 438-445, 587-595 |
| Summarize node | `crater-agent/.../graph.py` | 524-580 |
| 去重逻辑 | `crater-agent/.../graph.py` | 446-472 |
| Token 预算 | `crater-agent/.../graph.py` | 43-52 |
| 工具结果处理 | `crater-agent/.../graph.py` | 153-207 |
| 主动压缩 | `crater-agent/.../graph.py` | 272-306 |
| 历史加载 | `crater-agent/.../memory/session.py` | 28-84 |
| State 初始化 | `crater-agent/.../single.py` | 37-48 |
| Orphan 取消 | `crater-agent/.../single.py` | 248-266 |
