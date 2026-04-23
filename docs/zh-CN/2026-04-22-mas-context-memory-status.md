# Crater Multi-Agent 上下文与记忆管理现状分析

> 日期: 2026-04-22
> 关联设计: `docs/reference/memory-context-engineering.md`

---

## 一、MAS 信息流全景

```
用户消息
  │
  ├── IntentRouter (1次LLM)
  │     输入: user_message + page_context + history
  │     输出: RoutingDecision (operation_mode, targets)
  │
  └── Coordinator Loop (最多 lead_max_rounds=8 轮)
        │
        ├── Fast-path 判断 (无LLM)
        │     └→ 直接决定 plan/observe/act/finalize
        │
        ├── Coordinator Decision (1次LLM)
        │     输入: StateView (所有 artifacts + tool 统计)
        │     输出: next_stage
        │
        ├─→ "plan" ──→ Planner (1次LLM)
        │     输入: goal + page_context + capabilities + evidence_summary
        │     输出: PlanArtifact (steps, candidate_tools, risk)
        │     特点: 不执行工具，纯推理
        │
        ├─→ "observe" ──→ Explorer (N次LLM tool loop)
        │     输入: goal + plan (hints) + compact_evidence + attempted_sigs
        │     可调: 仅 READ_ONLY 工具
        │     输出: ObservationArtifact (summary, evidence)
        │     特点: 可偏离 plan，自主选择工具
        │
        ├─→ "act" ──→ Executor (N次LLM tool loop)
        │     输入: goal + plan + observation + compact_evidence + pending_actions
        │     可调: 全部工具 (read + write)
        │     输出: ExecutionArtifact (actions, summary)
        │
        └─→ "finalize" ──→ Final Summarization (1次LLM)
              输入: StateView + terminal_answer
              输出: 用户可读的最终回答
```

---

## 二、各 Agent 的信息可见性

| Agent | 看到什么 | 看不到什么 |
|-------|---------|-----------|
| **Coordinator** | 所有 artifacts + tool 统计 | 原始 tool result 全文 |
| **Planner** | goal + page_context + capabilities + evidence_summary | 原始 tool result、具体 tool_args |
| **Explorer** | goal + plan.steps + plan.candidate_tools + compact_evidence + attempted_sigs | Executor 的 action 结果、plan 的 risk_assessment |
| **Executor** | goal + plan + observation + compact_evidence + pending_actions | 无（最完整的视图） |

**设计亮点 — StateView 投影**

`state.py:292` 的 `build_state_view(for_role)` 实现了按角色投影，每个 agent 只看到与自己角色相关的 artifact。这比"所有 agent 共享同一个 message list"好很多。

---

## 三、Tool Result 压缩现状

### 3.1 当前 turn 的 tool result (Explorer/Executor 的 tool loop 内)

```
tool 返回完整结果
  → _truncate_text(result, max_chars=1600)  # 通用截断
  → 作为 ToolMessage 加入该 agent 的消息列表

错误结果:
  → _truncate_text(error_message, max_chars=1200)
```

**问题**: 统一 1600 chars 硬截断，没有 per-tool 差异化，没有 LLM extract。

### 3.2 跨 Agent 共享的 evidence (compact_evidence)

```python
# multi.py:163-214
_compact_tool_result_for_prompt(tool_name, result):
  list_user_jobs    → 保留 12 个 job，每个 5 字段 (jobName/status/jobType/...)
  get_health_overview → 3 字段 (totalJobs/statusCount/lookbackDays)
  get_job_detail    → 4 字段 (jobName/status/jobType/resources)
  error             → message 截断到 320 chars
  其他              → 整体截断到 500 chars
```

**优点**: 对高频工具做了精细化字段提取，信噪比高。
**问题**: 
1. 只覆盖 3 个工具，其余 73 个 fallback 到 500 chars
2. 没有用 token 计数，纯字符限制
3. 大工具结果（如 get_job_logs）的 log 字段被 500 chars 截断掉了

### 3.3 证据滑动窗口

```
state.tool_records: 最多 30 条
state.attempted_tool_signatures: 无上限，用于去重
```

**问题**: 30 条记录不考虑 token 总量。如果每条 500 chars ≈ 125 tokens，30 条 ≈ 3750 tokens。但如果某些记录没被截断好，可能远超预算。

---

## 四、Token 估算现状

```python
# multi.py:116-124
def _estimate_tokens_from_messages(messages: list) -> int:
    total_chars = 0
    for message in messages:
        content = getattr(message, "content", "")
        if isinstance(content, list):
            total_chars += sum(len(str(item)) for item in content)
        else:
            total_chars += len(str(content or ""))
    return max(1, total_chars // 4) if total_chars else 0
```

**问题**: `total_chars // 4` 是最粗糙的估算，对中文误差极大（中文约 1.5-2 chars/token，不是 4）。

---

## 五、历史消息处理

```python
# state.py:335-355
max_history_items = 12
max_history_total_chars = 6000
每条消息截断到 1200 chars（只保留头部，不是 head+tail）
```

**问题**:
1. 只保留头部，tail 可能有关键信息（错误堆栈）
2. 没用 token 计数
3. 12 条 × 1200 chars = 14400 chars 最大值，但 6000 total 限制实际只允许约 5 条

---

## 六、上下文污染风险

### 已有的隔离机制
1. **StateView 角色投影** — 每个 agent 只看到相关 artifact
2. **工具权限** — Explorer 只能 read，Executor 可以 read+write
3. **去重签名** — 相同参数的工具调用不重复执行
4. **迭代上限** — subagent_max_iterations=25，lead_max_rounds=8

### 存在的污染风险
1. **Planner 偏差传播** — Planner 的错误判断会通过 plan.steps 影响 Explorer 的工具选择
2. **旧 evidence 堆积** — 30 条滑动窗口不考虑相关性，early-stage 的不相关证据可能占满窗口
3. **无 "遗忘" 机制** — 所有证据平等对待，没有按当前阶段目标筛选
4. **跨轮次 Coordinator 累积** — Coordinator 每轮看所有 artifacts，如果 artifacts 没被压缩，后期上下文越来越大

---

## 七、与 Single-Agent 的对比

| 维度 | Single-Agent (graph.py) | Multi-Agent (multi.py) |
|------|------------------------|----------------------|
| Token 计数 | tiktoken cl100k_base ✅ | total_chars // 4 ❌ |
| Tool result 截断 | per-tool token 预算 ✅ | 3 个工具精细化 + 其余 500 chars ❌ |
| LLM Extract | 超限时 LLM 提取 ✅ | 无 ❌ |
| 历史截断 | head+tail 1200 chars ✅ | head-only 1200 chars ❌ |
| LLM 压缩 | compaction.py ✅ | 无 ❌ |
| Micro compact | 未实现 ❌ | 未实现 ❌ |
| 上下文隔离 | 无（单 agent） | StateView 角色投影 ✅ |
| 工具去重 | 同参数去重 ✅ | 同参数去重 ✅ |

---

## 八、总结

MAS 的上下文管理有良好的 **架构基础**（StateView 角色投影、工具权限隔离、证据去重），但在 **工程精度** 上落后 single-agent：

1. Token 估算最粗糙
2. Tool result 压缩覆盖面不足（73 个工具 fallback 500 chars）
3. 无 LLM 介入的智能压缩
4. 历史截断策略原始（头部截断，无 head+tail）
5. 无上下文窗口主动管理（没有 proactive_compact 机制）
