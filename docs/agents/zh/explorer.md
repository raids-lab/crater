# 探索者智能体

> 证据收集器 — 通过只读工具收集信息。

---

## 角色

探索者通过迭代选择和调用只读工具来执行规划者的调查步骤。它构建一个证据库，供验证者进行验证，并供执行者据此采取行动。

---

## 工具循环

```
规划者的 candidate_tools + enabled_tools
  ↓
LLM 选择下一批工具 (run_json)
  ↓
通过 GoBackendToolExecutor 执行每个工具
  ↓
构建 compact_evidence（结构化摘要）
  ↓
LLM 判断：是否需要更多工具？
  ├─ 是 → 选择下一批（循环）
  └─ 否 → 汇总证据
```

### 约束条件

- **仅限只读**：通过 `READ_ONLY_TOOL_NAMES` 进行硬过滤 — 即使 LLM 请求写操作工具也会被拒绝
- **去重**：跳过已尝试过的具有相同 `(tool_name, args)` 签名的工具调用
- **迭代次数限制**：`subagent_max_iterations`（默认 25）限定工具调用总次数上限

---

## 输出

```python
class ObservationArtifact:
    summary: str              # 自然语言证据摘要
    facts: list[str]          # 提取的事实
    open_questions: list[str] # 未解决的问题
    evidence: list[dict]      # 精简证据条目
    stage_complete: bool      # 探索是否完成
```

---

## 代码

| 组件 | 文件 |
|------|------|
| 探索者智能体 | `crater_agent/agents/explorer.py` |
