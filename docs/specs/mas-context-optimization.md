# Crater Multi-Agent 上下文管理优化 Spec

> 日期: 2026-04-22 (v4)
> 前置: `docs/zh-CN/2026-04-22-mas-context-memory-status.md`
> 参考: `docs/reference/memory-context-engineering.md`
> 状态: 待确认

---

## 核心发现：当前 budget 严重低估

| 模型 | 实际窗口 | 当前 budget | 利用率 |
|------|---------|------------|--------|
| qwen3-vl-235b-thinking (default) | 131,072 | 30,000 | 23% |
| qwen3-vl-flash (MAS sub-agents) | ~1,000,000 | 30,000 | 3% |
| qwen3.6-plus | 262,144 | 30,000 | 11% |

在 3% 窗口利用率下做激进截断是没有意义的。**很多截断本来就不需要发生。**

来源: [OpenRouter Qwen3 VL Thinking](https://openrouter.ai/qwen/qwen3-vl-235b-a22b-thinking), [Qwen3.5-Flash](https://openrouter.ai/qwen/qwen3.5-flash-02-23)

---

## 设计原则

1. **先提升 budget，再谈压缩** — 30K → 合理值后，大部分场景不需要压缩
2. **Python 进程内缓存，不全塞 LLM** — tool result 存在内存里，按需注入 LLM 调用
3. **LLM Extract 是主路径（对超限结果），硬截断是 fallback**
4. **DashScope 返回的 `usage.prompt_tokens` 可以做真实追踪** — 不需要完全靠自己估算
5. **MAS 当前执行不过度压缩，大头压缩在历史 session**

---

## 一、模型窗口自动推断

### 问题
`max_context_tokens=30000` 是个硬编码猜测值。不应该在 llm-clients.json 中配（那是路由配置），也不应该猜。

### 方案：从模型名自动推断 + 可覆盖

新建 `crater_agent/llm/model_info.py`:

```python
"""Model metadata: context windows, pricing tiers, capabilities."""

# 已知模型的 context window（tokens）
# 来源: DashScope 官方文档 / OpenRouter / HuggingFace model card
_MODEL_CONTEXT_WINDOWS: dict[str, int] = {
    # Qwen3 VL 系列
    "qwen3-vl-235b-a22b-thinking": 131_072,
    "qwen3-vl-235b-a22b": 131_072,
    "qwen3-vl-flash": 1_000_000,
    # Qwen3 文本系列
    "qwen3-235b-a22b": 262_144,
    "qwen3-max": 262_144,
    "qwen-plus": 131_072,
    # Qwen3.5 / 3.6
    "qwen3.5": 262_144,
    "qwen3.5-flash": 1_000_000,
    "qwen3.6-plus": 262_144,
}

# 合理的工作预算上限（考虑 context rot，不应超过窗口的 50%）
_MAX_WORKING_BUDGET = 65_536

def get_model_context_window(model_name: str) -> int:
    """从模型名推断 context window 大小。"""
    name_lower = model_name.lower()
    for key, window in _MODEL_CONTEXT_WINDOWS.items():
        if key in name_lower:
            return window
    return 131_072  # 保守默认值

def get_working_budget(model_name: str) -> int:
    """计算合理的工作预算（考虑 context rot）。
    
    规则: min(模型窗口 * 50%, 65536)
    - 不超过窗口的 50% — 避免 context rot（12-Factor Agent 的 "dumb zone" 在 40-60%）
    - 不超过 65K — 即使 1M 窗口也没必要用那么多，信噪比会下降
    """
    window = get_model_context_window(model_name)
    return min(window // 2, _MAX_WORKING_BUDGET)
```

### 使用方式

```python
# 在 single-agent graph.py 中:
from crater_agent.llm.model_info import get_working_budget

# 替换 settings.max_context_tokens:
budget = get_working_budget(llm.model_name)  # 如 qwen3-vl-flash → 65536

# 在 MAS multi.py 中:
# 每个 sub-agent 用自己模型的 budget
explorer_budget = get_working_budget(explorer.llm.model_name)
```

### .env 可覆盖

```python
# config.py:
max_context_tokens: int = Field(
    default=0,  # 0 表示自动推断
    description="Override context budget. 0=auto-detect from model.",
)
```

如果 `max_context_tokens > 0`，用配置值；如果 `=0`，从模型名自动推断。

---

## 二、Python 进程内多级缓存

### 问题
当前所有 tool result 都塞进 LLM 的 messages 列表。但 LLM 不需要看到所有历史 tool result — 只需要看到**当前推理相关的**。

### 方案：三级存储架构（对齐 Google ADK）

```
L1: Working Context (每次 LLM 调用的 messages)
    - 只包含当前推理需要的信息
    - 生命周期: 单次 LLM 调用
    - 大小: 受 working budget 控制

L2: Request-Scoped Memory (Python 进程内 dict)
    - 所有 tool result（完整 + compact 两个版本）
    - 当前 turn 的所有 agent 的推理结论
    - 生命周期: 单个用户请求（FastAPI request scope）
    - 大小: 无限制（内存中，不受 LLM 窗口限制）

L3: Session Storage (Go 后端 PostgreSQL)
    - 跨 turn 的完整消息历史
    - 工具调用记录
    - 生命周期: 会话级
    - 大小: 无限制（数据库存储）
```

### L2 的具体实现

在 `state.py` 的 `MASState` 中，`tool_records` 已经是 L2 — 它存在 Python 内存中，不在 LLM messages 里。问题是当前把 compact_evidence 全部注入 prompt，应该改为**按需取用**。

```python
class ToolResultCache:
    """Request-scoped tool result cache.
    
    存储完整 + compact 两个版本。任何 agent 可以查询。
    不进入 LLM messages，只在构建 prompt 时按需注入。
    """
    
    def __init__(self):
        self._records: dict[str, ToolCacheEntry] = {}  # signature → entry
    
    def store(self, signature: str, tool_name: str, tool_args: dict,
              full_result: dict, compact_result: str):
        self._records[signature] = ToolCacheEntry(
            tool_name=tool_name,
            tool_args=tool_args,
            full_result=full_result,
            compact_result=compact_result,
        )
    
    def get_compact(self, signature: str) -> str | None:
        """获取紧凑版本（用于注入 LLM prompt）"""
        entry = self._records.get(signature)
        return entry.compact_result if entry else None
    
    def get_full(self, signature: str) -> dict | None:
        """获取完整版本（用于需要全量数据的场景）"""
        entry = self._records.get(signature)
        return entry.full_result if entry else None
    
    def get_all_compact(self, max_tokens: int = 8000) -> list[dict]:
        """获取所有记录的紧凑版本（用于跨 agent 证据共享）"""
        # 最新优先，token 预算控制
        ...
    
    def has(self, signature: str) -> bool:
        return signature in self._records
```

### L2 和 LLM 调用的关系

```
Explorer 的 tool loop:
  每轮 LLM 调用:
    messages = [SystemMsg, UserPrompt, tool_call_1, tool_result_1, ...]
    # tool_result 用 compact 版本（从 L2 取）
    # 但 L2 中始终保存完整版本
  
  tool 调用时:
    result = await call_tool(...)          # Go 后端返回完整结果
    compact = extract_if_needed(result)    # 超限时 LLM extract
    cache.store(sig, full=result, compact=compact)  # 存入 L2

Executor 需要 Explorer 的证据:
    evidence = cache.get_all_compact(max_tokens=8000)  # 从 L2 取
    # 不需要重新截断，因为 compact 版本已经是高质量的
```

---

## 三、Tool Result Extract（大结果处理）

### 触发条件

```python
budget = _TOOL_TOKEN_BUDGETS.get(tool_name, settings.tool_result_default_budget)
if count_tokens(full_text) > budget:
    → 触发 LLM Extract
```

### Tool 专用 Prompt

```python
_TOOL_EXTRACT_PROMPTS = {
    "get_job_logs": (
        "从以下容器日志中提取关键信息：\n"
        "1. 所有 ERROR/FATAL/OOM/Kill/Exception 及完整堆栈\n"
        "2. 进程退出信号、退出码、最后运行状态\n"
        "3. 关键状态变化时间点（启动/重启/终止）\n"
        "4. GPU/内存/存储相关的资源异常或警告\n"
        "删除：重复正常日志行、心跳探针、健康检查输出。"
    ),
    "get_diagnostic_context": (
        "保留完整诊断结论和根因分析。\n"
        "保留资源使用异常数据（CPU/GPU/内存具体数值）。\n"
        "保留 Event 类型、原因和消息。\n"
        "删除冗余的正常状态检查数据。"
    ),
    "prometheus_query": (
        "保留超阈值或异常的数据点及精确时间戳。\n"
        "保留趋势变化前后的关键数据点。\n"
        "删除正常范围内的重复数据点。\n"
        "保留指标名、标签、单位。"
    ),
}
```

### Extract Once, Share Everywhere

MAS 中 extract 只做一次，结果存入 L2 cache，所有下游 agent 复用：

```
get_job_logs 返回 50KB
  → LLM Extract (tool 专用 prompt) → ~3K tokens 高质量摘要
  → cache.store(sig, full=50KB, compact=3K)
  
  Explorer 看: compact (3K tokens)     ← 高质量
  Executor 看: 同一份 compact          ← 零额外损失
  Finalize 看: 同一份 compact          ← 结论基于完整信息
```

### Fallback 链

```
超限 → LLM Extract (tool 专用 prompt, 10s 超时)
  → 成功 → 存 compact
  → 失败 → head+tail 硬截断 → 存 compact
```

---

## 四、跨 Agent 同参数缓存命中

```python
if cache.has(signature):
    # 从 L2 cache 直接取 compact 版本
    tool_observation = cache.get_compact(signature)
    result = cache.get_full(signature)
    # → success 状态，不是 error
```

---

## 五、Compaction 阈值管理

### 配置方式

```python
# config.py 新增:
compaction_threshold_pct: int = Field(
    default=85,
    description="Trigger compaction when context usage exceeds this % of working budget",
)
tool_result_default_budget: int = Field(
    default=3000,
    description="Default per-tool token budget",
)
evidence_total_budget: int = Field(
    default=8000,
    description="Total token budget for cross-agent evidence",
)
```

```bash
# .env:
CRATER_AGENT_COMPACTION_THRESHOLD_PCT=85
CRATER_AGENT_TOOL_RESULT_DEFAULT_BUDGET=3000
CRATER_AGENT_EVIDENCE_TOTAL_BUDGET=8000
# max_context_tokens=0 表示自动推断
CRATER_AGENT_MAX_CONTEXT_TOKENS=0
```

### 阈值判断流程

```python
# 每次 LLM 调用前:
model_name = llm.model_name or "unknown"

if settings.max_context_tokens > 0:
    working_budget = settings.max_context_tokens  # 手动覆盖
else:
    working_budget = get_working_budget(model_name)  # 自动推断

threshold = working_budget * settings.compaction_threshold_pct / 100
schema_reserve = 8000
response_reserve = 4000
available = threshold - schema_reserve - response_reserve

estimated = count_message_tokens(messages)
if estimated > available:
    # 触发压缩（single-agent: LLM compact → fallback 硬截断）
    # （MAS tool loop: 渐进式 — 先压 AI 推理）
```

### DashScope 真实 token 追踪

每次 LLM 调用后，DashScope 返回 `usage.prompt_tokens`。可以用来：
1. **校准估算**: 对比 tiktoken 估算 vs DashScope 实际值，调整系数
2. **成本追踪**: 记录每个 agent 的实际 token 消耗
3. **动态调整**: 如果实际消耗远低于估算，可以放宽预算

```python
# 在 LLM 调用后:
actual_prompt_tokens = response.response_metadata.get("token_usage", {}).get("prompt_tokens")
if actual_prompt_tokens:
    logger.info("Token estimate vs actual: estimated=%d, actual=%d, ratio=%.2f",
                estimated, actual_prompt_tokens, estimated / actual_prompt_tokens)
```

---

## 六、MAS 各场景完整方案

| 场景 | 策略 | 注意 |
|------|------|------|
| **当前 tool result 不超限** | 原样使用，零处理 | 大部分走这里（提升 budget 后） |
| **当前 tool result 超限** | LLM Extract (tool 专用 prompt) → 存入 L2 cache | Extract 一次，所有 agent 复用 |
| **跨 Agent evidence** | 从 L2 cache 取 compact 版本 | 不再二次截断 |
| **同参数复用** | L2 cache 命中 → success | 不是 error |
| **Tool loop 接近窗口** | 渐进式压缩 AI 推理（≥85% 时） | Tool result 不压缩（已经是 compact） |
| **历史 session** | head+tail + token 预算 (已实现) | 大头压缩区 |

---

## 七、执行优先级

| 优先级 | 优化项 | 工作量 | 价值 |
|--------|--------|--------|------|
| P0 | model_info.py — 模型窗口自动推断 | 小 | **极高** — 把 budget 从 30K 提到 65K，大部分截断不再需要 |
| P0 | config.py — 新增可配置项 | 小 | 高 — 后续优化基础 |
| P0 | Token 计数对齐 (multi.py) | 小 | 高 |
| P0 | Tool Extract Once + L2 Cache | 大 | **极高** — 解决 MAS 信息传递根本问题 |
| P0 | Tool 专用 Extract Prompt | 中 | 高 — 提取质量 |
| P0 | 缓存命中返回 success | 小 | 高 |
| P1 | 去掉 explorer.py 二次截断 | 小 | 中 |
| P1 | evidence 总量 token 预算 | 中 | 中 |
| P2 | 渐进式 tool loop 压缩 | 中 | 低 — 提升 budget 后极少触发 |
| P2 | DashScope usage 追踪 | 中 | 中 — 成本可观测 |