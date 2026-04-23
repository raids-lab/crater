# 已知问题与根因分析

> 记录当前 single_agent 模式下已观察到的效果问题、根因分析和改进方向。
> 最后更新: 2026-04-21

---

## 问题 1: Agent 自作主张绑定作业 ✅ 已修复

### 修复内容

1. user 场景的 page_context_detail 注入文案改为"页面上下文（仅供参考，操作前需用户确认）"
2. 新增用户原则 U7: 基于页面推断意图时必须先向用户确认
3. 用户原则 U6: 单作业绑定要谨慎，只有用户明确给出才绑定

---

## 问题 2: 批量操作撞工具调用上限

### 现象

用户说"诊断并批量重提所有失败作业"，agent 先 `diagnose_job` + `list_user_jobs`（累计调用若干次），然后尝试对多个作业分别调用 `resubmit_job`，后几个被取消（"已超过单轮工具调用上限"）。

### 根因

1. **缺少批量工具**: 目前没有 `batch_resubmit_job` 之类的原子工具，每个重提都是独立调用
2. **前序调用消耗配额**: 诊断 + 列表已消耗若干次，留给重提的配额不够
3. **LLM 一次性生成多个 tool_calls**: LLM 在单次 response 中产出 N 个 resubmit_job 调用，但 should_continue 在 agent_node 输出后（tool_calls 执行前）就判定达上限 → 全部取消

### 触发细节

`should_continue` 检查发生在 agent_node 之后:
- 如果 tools_node 刚执行完第 14 次调用（`tool_call_count=14`），agent_node 产出 3 个新 tool_calls
- should_continue: `14 < 15` → 路由到 tools
- tools_node 执行第 1 个: `tool_call_count=15` → 第 2、3 个正常执行（因为 for 循环在 tools_node 内，不会中途检查上限）
- 但如果之前已经 `tool_call_count=15`，should_continue 直接路由到 summarize → 所有 tool_calls 被取消

**关键: tools_node 内部的 for 循环不检查上限**，一旦进入 tools_node，当前批次的所有 tool_calls 都会执行完。上限只在 should_continue（节点间路由）时检查。

### 改进方向

- 实现 `batch_resubmit_job` 工具（一次调用处理多个作业）
- 或在 prompt 中引导 LLM: "批量操作时分批提交，每批不超过 3 个"
- 或在 tools_node 内增加上限检查，遇到超限时提前 break

---

## 问题 3: 写操作缺少确认（用户感知） ✅ 已修复

### 修复内容

1. tools_node 遇到第一个 confirmation_required 立即 break，不执行后续 tool_calls
2. 剩余 tool_calls 作为 paused_for_confirmation 状态返回
3. 不再出现"有的确认了有的没确认"的混乱感
4. 新增共享原则 12: "确认卡片由系统渲染"，LLM 不再在回复中描述确认流程

---

## 问题 4: 确认卡片内容污染聊天 ✅ 已修复

### 根因（纠正）

这**不是** graph 流程问题。真正的发生路径：

1. ReAct 循环中 LLM 先调读工具（diagnose_job、list_user_jobs）获取到作业列表
2. LLM 在**下一轮 agent_node** 中产出**纯文本回复**（没有 tool_calls），内容包含"确认卡片已生成：{json}"
3. should_continue 判定没有 tool_calls → END
4. orchestrator 将这段文字作为 `final_answer` 发送给前端

LLM 自己编造了确认卡片的内容（带着工具返回的真实 job name/id），并在文字中假装"已生成"，实际上根本没调写操作工具。这是 LLM 行为问题 — 它把"描述计划"和"执行操作"混为一谈。

### 修复内容

加强共享原则 12: "禁止在回复中编造确认卡片 — 写操作必须通过调用对应写工具触发，绝对不要在文字回复中伪造确认卡片内容或说'确认卡片已生成'。如果准备执行写操作，直接调用工具。"

---

## 问题 5: Admin/Portal Prompt 无差异化 ✅ 已修复

### 修复内容

1. Prompt 拆分为 `_BASE_PROMPT` + `_ADMIN_ADDON` / `_USER_ADDON` 三段
2. `build_system_prompt()` 根据 `capabilities.surface.page_scope` 选择 addon
3. admin 原则 6 条（A1-A6），user 原则 10 条（U1-U10），互不可见
4. user 场景 prompt 精简约 40%（去掉了管理员诊断指引、PromQL 参考等）

---

## 问题 6: 模型选择对工具调用遵循度的影响

### 现象

LLM 频繁违反 prompt 中的工作原则（如不澄清歧义、过度绑定作业、不遵守最少调用原则）。

### 根因

当前 `default` client 使用 `qwen3-vl-235b-a22b-thinking`（llm-clients.json）:
- 这是 VL 多模态 + thinking（思维链）模型
- 思维链模型在工具调用场景下，可能在 thinking 阶段消耗大量 token 但最终决策不遵循指令
- VL 模型的工具调用能力通常不如纯文本指令模型

### 改进方向

- 对话 agent 场景考虑使用非 thinking 模型（如 `qwen-plus` 或 `qwen-max`）
- 或保留 thinking 但减少 prompt 复杂度，让模型更容易遵循
- 可在 llm-clients.json 中为 single_agent 配置专用 client key

---

## 代码位置索引

| 问题 | 相关文件 |
|------|---------|
| page_context 注入 | `crater-agent/.../prompts.py:114-123` |
| prompt 原则 | `crater-agent/.../prompts.py:13-37` |
| 工具上限 | `crater-agent/.../graph.py:587-595`, `config.py:61` |
| tools_node 循环 | `crater-agent/.../graph.py:442-520` |
| 确认机制 | `backend/.../tools_dispatch.go:236-282` |
| Orchestrator 取消 | `crater-agent/.../single.py:248-266` |
| LLM 模型配置 | `crater-agent/config/llm-clients.json` |
