# 确认机制（Confirmation Mechanism）

> 记录写操作确认卡片的完整前后端+Agent链路，包括当前行为和已知缺陷。
> 最后更新: 2026-04-21

---

## 1. 设计意图

所有写操作（停止/删除/重提/创建作业、节点 cordon/drain、运维脚本等）**不直接执行**，而是先返回确认卡片，让用户审查参数后手动确认。

---

## 2. 确认工具清单

以下工具在 Go 后端 `isAgentConfirmTool()` 中注册（`tools_dispatch.go:74-84`）:

| 工具 | 风险等级 | 交互类型 |
|------|---------|---------|
| `resubmit_job` | high | form（可编辑参数） |
| `create_jupyter_job` | high | form |
| `create_training_job` | high | form |
| `stop_job` | high | approval（确认/拒绝） |
| `delete_job` | high | approval |
| `batch_stop_jobs` | high | approval |
| `notify_job_owner` | high | approval |
| `cordon_node` | high | approval |
| `uncordon_node` | medium | approval |
| `drain_node` | critical | approval |
| `delete_pod` | critical | approval |
| `restart_workload` | high | approval |
| `run_ops_script` | critical | approval |
| `mark_audit_handled` | high | approval |

---

## 3. 完整链路

### 3.1 Agent 侧（graph.py tools_node）

```
LLM 产出 tool_calls: [resubmit_job({job_name: "xxx", ...})]
  ↓
tools_node 遍历 tool_calls，逐个执行:
  ↓
GoBackendToolExecutor.execute(tool_name="resubmit_job", ...)
  ↓ HTTP POST /api/agent/tools/execute
Go 后端识别为 confirm 工具:
  → 构建 confirmation（描述、风险、表单字段）
  → 创建 AgentToolCall 记录（状态="await_confirm"）
  → 返回 { status: "confirmation_required", confirmation: {...} }
  ↓
tools_node 收到 confirmation_required:
  → 存入 state.pending_confirmation（覆盖式赋值）
  → 作为 ToolMessage 放入 messages
  ↓
should_continue / after_tools 检测到 pending_confirmation
  → 路由到 END（暂停 ReAct 循环）
```

### 3.2 Orchestrator 侧（single.py）

Orchestrator 通过 SSE 事件流将确认信号传递给前端:

```
astream_events 产出 on_tool_end 或 on_chain_end(name="tools"):
  ↓
检测 result.status == "confirmation_required":
  → 产出 SSE 事件: { event: "tool_call_confirmation_required", data: {...} }
  → 设置 emitted_confirmation = True
  ↓
循环结束后:
  → 如果 emitted_confirmation，不产出 final_answer
  → 产出 { event: "done" }
```

### 3.3 Go 后端 SSE 代理（python_proxy.go）

Go 后端作为 SSE 代理：
- 收到 `tool_call_confirmation_required` 事件 → 设置 turn 状态为 `"awaiting_confirmation"`
- 透传事件到前端

### 3.4 前端渲染（AIChatDrawer.tsx / ConfirmActionCard.tsx）

```
收到 SSE tool_call_confirmation_required:
  ↓
创建 conversation item (kind: 'confirmation_required')
  → confirmId, confirmAction, confirmDescription, confirmForm, confirmInteraction
  ↓
渲染 ConfirmActionCard 组件:
  → approval 类型: 确认/拒绝按钮
  → form 类型: 弹出对话框，用户可编辑参数字段（名称、CPU、内存、GPU 等）
```

### 3.5 用户确认流程

```
用户点击"确认"（或编辑表单后确认）:
  ↓
POST /api/v1/agent/chat/confirm
  body: { confirmId, confirmed: true, payload: {编辑后的字段值} }
  ↓
Go 后端 ConfirmToolExecution():
  → 校验 toolCall.ResultStatus == "await_confirm"
  → mergeToolArgsWithPayload()（合并用户编辑）
  → executeWriteTool()（真正执行写操作）
  → 更新 toolCall 记录（状态=success/error）
  → 返回执行结果
  ↓
前端收到确认结果:
  → 更新卡片状态
  → 调用 startAgentResume(confirmId)
  ↓
POST /api/v1/agent/chat/resume
  ↓
Go 后端 ResumeAfterConfirmation():
  → single_agent 模式: streamConfirmationOutcome()（生成执行结果的自然语言说明）
  → multi_agent 模式: 创建新 turn，发送"继续完成上一轮计划"给 agent
```

### 3.6 用户拒绝流程

```
用户点击"拒绝":
  ↓
POST /api/v1/agent/chat/confirm
  body: { confirmId, confirmed: false }
  ↓
Go 后端: 更新 toolCall 状态为 "rejected"，返回
```

---

## 4. Break-on-First-Confirmation（已修复）

**修复前问题**: LLM 单次 response 产出多个 tool_calls 时，tools_node 的 for 循环会逐个执行所有工具，`pending_confirmation` 每次被覆盖，只保留最后一个确认。前面的确认丢失。

**修复后行为**（graph.py tools_node）:

```python
for idx, tc in enumerate(all_tool_calls):
    ...
    if result.get("status") == "confirmation_required":
        pending_confirmation = result
        tool_messages.append(ToolMessage(...))
        # 为剩余未执行的 tool_calls 填充占位 ToolMessage
        for remaining_tc in all_tool_calls[idx + 1:]:
            tool_messages.append(ToolMessage(
                content="前序写操作需要用户确认，本次调用已暂停，待确认后继续。",
                tool_call_id=remaining_tc["id"],
            ))
        break   # ← 遇到第一个确认即停止
```

**行为**:
- 遇到第一个 confirmation_required → 立即 break，不执行后续 tool_calls
- 剩余 tool_calls 获得占位 ToolMessage（LangGraph 要求每个 tool_call 有对应 ToolMessage）
- `pending_confirmation` 保证只有一个，不会被覆盖
- 用户确认后 resume，agent 可在下一轮继续调用剩余工具

---

## 5. Orchestrator 层的取消机制

当 `should_continue` 判定工具调用达上限 → 路由到 summarize（而非 tools）时，LLM 最后一次产出的 tool_calls 永远不会被 tools_node 执行。Orchestrator 层在 `astream_events` 结束后检查 `pending_tool_calls` 列表，将未执行的标记为 `cancelled`（single.py:248-266）:

```python
for tc in pending_tool_calls:
    yield {
        "event": "tool_call_completed",
        "data": {
            ...
            "resultSummary": "已超过单轮工具调用上限，本次调用已取消",
            "status": "cancelled",
        },
    }
```

---

## 6. 确认卡片与聊天流的关系

**当前行为**：
- SSE 事件 `tool_call_confirmation_required` 由前端作为独立的 `ConfirmActionCard` 组件渲染，**不是**纯文本聊天消息
- 但在 agent 内部，confirmation 结果作为 `ToolMessage` 放入了 messages 列表（graph.py:501-506），LLM 在 summarize 或后续 agent_node 中会看到这些内容并可能"解读"它
- 如果 pending_confirmation 导致 END → 不走 summarize → LLM 不会再回复 → 不会出现"解读"问题
- **只在 pending_confirmation + 达到工具上限的边界情况下**，LLM 可能在 summarize_node 中解读 confirmation JSON

---

## 7. 代码位置

| 组件 | 文件 | 关键行 |
|------|------|-------|
| 确认工具注册 | `backend/.../tools_dispatch.go` | 74-84 |
| 确认表单构建 | `backend/.../confirmation.go` | 31-197 |
| 确认结果描述 | `backend/.../confirmation.go` | 395-492 |
| 确认 API | `backend/.../handlers.go` | 215-333 (confirm), 125-213 (resume) |
| Agent graph 处理 | `crater-agent/.../graph.py` | 498-520 (pending_confirmation) |
| Orchestrator SSE | `crater-agent/.../single.py` | 128-146, 175-203 |
| 前端确认卡片 | `frontend/.../ConfirmActionCard.tsx` | 全文 |
| 前端 SSE 处理 | `frontend/.../AIChatDrawer.tsx` | 1408-1450 |
| 前端确认提交 | `frontend/.../agent.ts` | 565-574 |
