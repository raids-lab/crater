# 记忆与上下文系统

> Agent 如何访问对话历史、用户身份和页面级上下文。

---

## 1. 上下文流转

上下文从 Go 后端经过 FastAPI 应用流转到 Agent。Go 后端负责会话归属、历史裁剪和能力集生成；Python Agent 只消费本轮请求中携带的 `context`，不维护跨请求全局记忆：

```
Go Backend (JWT, session_id, 页面状态)
  ↓ GetOrCreateSession + owner/account 校验
AgentSession / AgentMessage / AgentToolCall
  ↓ buildAgentHistory() + buildAgentCapabilities()
  ↓ POST /chat
ChatRequest { session_id, message, context, user_id, page_context }
  ↓ build_request_context()
context dict { actor, page, history, continuation, capabilities, orchestration }
  ↓ single_agent 或 multi_agent 编排
系统提示词 / MAS StateView / tool loop messages
  ↓
LLM 只看到本 session、本 turn 显式传入的上下文
```

### 上下文结构

```python
{
    "actor": {
        "username": "user123",
        "user_id": 42,
        "account_name": "team-a",
        "account_id": 10,
        "role": "user",          # "user" | "admin"
        "locale": "zh-CN"
    },
    "page": {
        "url": "/portal/jobs/sg-xxx",
        "route": "/jobs",
        "job_name": "sg-xxx",    # 用户在作业详情页时填充
        "job_status": "Failed",
        "node_name": null,       # 在节点详情页时填充
        "pvc_name": null         # 在存储页面时填充
    },
    "capabilities": {
        "enabled_tools": [...],  # 可选：限制工具集
        "confirm_tools": [...],  # 需要用户确认的工具
        "tool_catalog": [...],   # Go 合并静态工具与 Python 本地工具目录
        "surface": { "page_scope": "user" }
    },
    "history": [...],            # Go 从本 session 读取并裁剪
    "continuation": {...},       # 澄清、待确认操作或 MAS workflow checkpoint
    "orchestration": {
        "mode": "single_agent"   # "single_agent" | "multi_agent"
    }
}
```

### 上下文控制项

| 上下文字段 | 影响范围 |
|------------|----------|
| `actor.role` | 工具可见性（admin 可见全部，user 可见子集） |
| `actor.username` | 注入系统提示词用于个性化 |
| `page.job_name` | 将 Agent 注意力绑定到特定作业 |
| `page.url` | 管理员路由检测（`/admin/*` → admin 角色） |
| `capabilities.enabled_tools` | 限制绑定到 LLM 的工具集 |
| `capabilities.confirm_tools` | 列入提示词，使 Agent 知道哪些工具需要确认 |
| `capabilities.tool_catalog[].mode` | 区分 `read_only`、预留的 `auto_action` 和 `confirm`；聊天触发的邮件动作如 `notify_job_owner` 会出现在 `confirm_tools` |
| `continuation` | 仅来自同一 session 最新 turn，用于澄清、确认恢复和 MAS workflow 续跑 |

---

## 2. 对话历史（记忆）

每轮对话时从 Go 后端加载对话历史。Agent 侧没有持久化记忆，Go 后端数据库是唯一数据源。

### 会话隔离结论

- `Chat` 入口按 `session_id` 调用 `GetOrCreateSession()`，随后校验 `session.UserID == token.UserID` 且 `session.AccountID == token.AccountID`；不匹配直接返回 403。
- 会话列表、置顶、删除、确认恢复等入口也通过 `GetOwnedSession()` 或同等 owner/account 校验。
- 历史消息和工具调用均按 `session_id` 查询；确认恢复先通过 `confirm_id` 找到 tool call，再校验该 tool call 所属 session 的 owner/account。
- Python 侧的 `SingleAgentOrchestrator`、`MASState`、`ModelClientFactory` 都是按请求/turn 接收上下文；全局对象只保留无用户数据的执行器、工具目录缓存和 `.env` 缓存。

因此，在后端不复用他人 `session_id` 且 owner/account 校验生效的前提下，不同用户、不同会话的历史、workflow checkpoint 和确认状态不会互相串线。

### 加载策略

Go 侧先把 `AgentMessage` 与 `AgentToolCall` 合并成最多 24 条历史条目：

| Go 侧限制 | 当前值 |
|-----------|--------|
| 历史条目数 | 24 |
| 普通消息最大长度 | 1600 字符 |
| Assistant 最大长度 | 1200 字符 |
| Tool call 历史最大长度 | 480 字符 |

然后 Python 侧再按 token 预算转换为 LangChain 消息：

```python
build_history_messages(
    history: list[dict],        # 来自 Go 后端
    max_tokens: int = 4000,     # token 预算
    tool_result_max_chars: int = 1200,  # 截断工具结果
    tool_error_max_chars: int = 1600,   # 为错误保留更多空间
) -> list[BaseMessage]
```

**算法**：
1. **逆序**遍历历史（最新的优先）
2. 将 `user` 转成 `HumanMessage`，`assistant` 转成 `AIMessage`
3. 将历史 `tool` 结果包装成 `AIMessage("【历史工具结果 ...】...")`，避免生成孤立 `ToolMessage`
4. 使用首尾截断策略截断工具结果
5. 累计 token 计数；预算耗尽时停止
6. 反转回时间顺序

### 截断

工具结果使用首尾截断，同时保留开头（上下文）和结尾（错误信息、摘要）：

```
[前 600 字符] ... (内容过长，已截断) ... [后 600 字符]
```

错误消息获得更多空间（1600 字符），因为错误详情对诊断至关重要。

### 单智能体与多智能体差异

| 模式 | 历史使用方式 | 工作记忆 |
|------|--------------|----------|
| `single_agent` | 将 `build_history_messages()` 结果直接拼到当前用户消息前 | LangGraph `CraterAgentState`，只在本 turn 内存在；过长时可触发 LLM 压缩 |
| `multi_agent` | IntentRouter 额外使用文本历史摘要；各 role tool loop 使用结构化 history messages | `MASState` 持有 observation/plan/execution/action/evidence；确认时通过 Go 持久化 workflow checkpoint 续跑 |

MAS 的工作记忆会限制 `tool_records` 最近 30 条，并把证据压缩为 `StateView` 传给不同角色。Planner、Explorer、Executor、Coordinator 看到的是投影后的状态，不是完整无限历史。

### 为何不使用 Agent 侧记忆

- Go 后端已存储完整会话历史（`AgentSession`、`AgentMessage`、`AgentToolCall`）
- Agent 侧记忆会与后端数据库产生一致性问题
- token 预算机制确保历史在任意对话长度下都能放入上下文窗口
- 多轮连续性由 Go 后端的会话管理处理
- MAS 需要跨确认恢复时，只持久化当前 workflow checkpoint，而不是把其他会话的状态放入全局内存

---

## 3. 系统提示词构建

系统提示词是上下文注入的主要载体。每次 ReAct 循环调用时构建一次：

```python
build_system_prompt(
    context: dict,
    skills_context: str = "",      # 来自 YAML 的诊断知识
    is_first_time: bool = False,   # 新用户的欢迎附加内容
    user_message: str = "",        # 当前未在模板中使用
) -> str
```

### 提示词结构

```
1. 角色定义（Crater 智能运维助手）
2. 22 条工作原则（证据优先、最少工具、确认机制等）
3. 平台规格说明（资源限制、挂载路径、配额）
4. 资源推荐流程
5. 管理员专用指导（集群诊断，仅 admin）
6. 可观测性与指标（PromQL 示例）
7. 工具选择指南
8. --- 动态注入 ---
9. 当前用户：{username}，角色：{role}，账户：{account_name}
10. 当前页面：job={job_name} (status={status}) / node={node_name}
11. 可用工具：{tool_list}
12. 需确认工具：{confirm_tool_list}
13. 技能上下文：诊断知识（来自 YAML 文件）
14. [可选] 首次使用欢迎附加内容
```

### Token 预算

| 部分 | 约 token 数 |
|------|-------------|
| 基础模板 + 工作原则 | ~1200 |
| 平台规格 + 管理员指导 | ~500 |
| 页面上下文注入 | ~50-200 |
| 能力详情 | ~200-400 |
| 技能上下文 | ~800-1500 |
| **合计** | **~2500-3500** |

---

## 4. 消息压缩

当对话过长时，消息会被压缩以保持在上下文窗口内。

### 主动压缩（在达到限制之前）

```
estimated_tokens = count_message_tokens(all_messages)
available = max_context_tokens(30000) - tool_schema_budget(8000) - response_reserve(4000)

if estimated_tokens > available:
    → 基于 LLM 的压缩（总结旧消息，保留近期消息）
    → 若 LLM 压缩失败：硬截断兜底
```

### LLM 压缩（`compact_messages_with_llm`）

1. 拆分消息：system（始终保留）+ body
2. 划分 body：可压缩部分（较旧）+ 保留部分（最近 N 条消息）
3. 调用 LLM 执行压缩提示词（15 秒超时）
4. 用单条摘要 `AIMessage` 替换可压缩消息

### 硬截断兜底（`_compact_messages_for_retry`）

- 保留：system 消息 + 最后一条 human 消息 + 最近 6 条消息
- 每条消息截断：system（1600 字符）、human（600）、tool（800）、AI（600）

### 被动恢复

遇到 `BadRequestError("context_length_exceeded")` 时：
1. 尝试 LLM 压缩
2. 重试 LLM 调用
3. 若仍失败：硬截断 + 再重试一次

---

## 代码

| 组件 | 文件 |
|------|------|
| 历史加载 | `crater_agent/memory/session.py` |
| 上下文构建 | `crater_agent/app.py` (`build_request_context`) |
| 系统提示词 | `crater_agent/agent/prompts.py` |
| 消息压缩 | `crater_agent/agent/compaction.py` |
| Token 计数器 | `crater_agent/llm/tokenizer.py` |
| LLM 客户端工厂 | `crater_agent/llm/client.py` |
| Agent 状态 | `crater_agent/agent/state.py` |
| 配置 | `crater_agent/config.py` |
