# 单智能体（ReAct）

> 基础智能体模式 -- 单个 LLM 配合工具，在"思考-行动-观察"循环中运行。

---

## 概述

单智能体使用 LangGraph 的 `StateGraph` 实现 ReAct（推理 + 行动）循环。LLM 自主决定调用哪些工具以及何时停止 -- 没有固定的工作流、意图分类或阶段转换。

这是所有其他智能体复用的核心构建模块：多智能体子智能体、任务智能体（审批）和流水线智能体都基于相同的图结构运行，只是配置不同。

---

## 图结构

```
Entry → [agent_node] → should_continue?
              ↑              │
              │         ┌────┼────┐
              │      "tools" │ "respond"  "summarize"
              │         │    │         │
              │    [tools_node]  END   [summarize_node]
              │         │                    │
              └─────────┘                   END
```

### 节点

| 节点 | 职责 | 是否绑定工具？ |
|------|------|---------------|
| `agent_node` | LLM 推理 -- 决定下一步操作或最终回答 | 是（所有已启用的工具） |
| `tools_node` | 执行 LLM 输出中的工具调用 | 不适用（仅执行） |
| `summarize_node` | 当工具调用次数达到上限时，强制生成总结 | 否（不带工具的 LLM） |

### 路由 (`should_continue`)

```python
if tool_call_count >= max_tool_calls_per_turn:
    if LLM wanted more tools → "summarize"  # 强制结束
    else → "respond"  # 自然结束
elif pending_confirmation:
    → "respond"  # 暂停等待用户审批
elif LLM has tool_calls:
    → "tools"  # 执行工具调用
else:
    → "respond"  # 最终回答
```

---

## 关键特性

### 系统提示词注入

在首次 LLM 调用时，系统提示词由以下内容动态构建：
- 基础模板（平台规则、工作原则）
- 用户上下文（用户名、角色、账户）
- 页面上下文（当前作业/节点/PVC，如果有的话）
- 能力清单（已启用的工具、需确认的工具）
- 技能知识（来自 YAML 的诊断模式）

### 工具结果处理

每个工具的返回结果都经过预算感知的处理流水线：

```
原始结果（可能非常大）
  ↓
在单工具 token 预算内？ → 直接使用
  ↓ 否
LLM 语义提取（10 秒超时）
  ↓ 失败
硬截断（头部 + 尾部）
```

### 去重

同一轮中具有相同 `(tool_name, args)` 的工具调用会被跳过，以防止无限循环。

### 确认暂停

当写操作工具返回 `confirmation_required` 时，图会在状态中设置 `pending_confirmation` 并路由到 END。编排器将此结果推送给前端，前端展示确认对话框。恢复后，图从暂停处继续执行。

---

## 配置

| 设置项 | 默认值 | 说明 |
|--------|--------|------|
| `max_tool_calls_per_turn` | 15 | 每轮 ReAct 循环的工具调用安全上限 |
| `tool_execution_timeout` | 30s | 单个工具的 HTTP 超时时间 |
| `max_context_tokens` | 30000 | 触发主动上下文压缩的阈值 |
| `history_max_tokens` | 4000 | 对话历史的 token 预算 |

---

## 代码

| 组件 | 文件 |
|------|------|
| 图构建器 | `crater_agent/agent/graph.py` |
| 状态定义 | `crater_agent/agent/state.py` |
| 系统提示词 | `crater_agent/agent/prompts.py` |
| 消息压缩 | `crater_agent/agent/compaction.py` |
| 编排器封装 | `crater_agent/orchestrators/single.py` |
