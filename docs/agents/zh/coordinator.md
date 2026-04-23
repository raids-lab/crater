# 协调者智能体

> 多智能体系统的编排大脑 -- 路由请求并管理阶段转换。

---

## 职责

协调者是唯一能看到完整多智能体系统状态的智能体。它做出两类决策：

1. **路由**：这个请求应该发往哪里？（引导 / 通用 / 诊断）
2. **流程控制**：流水线应该继续、回退还是终结？

协调者不调用工具，也不生成面向用户的内容。它只做结构性决策。

---

## 路由流水线

```
用户消息到达
  ↓
IntentRouter（确定性规则）
  - 正则表达式提取作业名称
  - "all" 关键词检测
  - 续接信号检测
  ↓
协调者 LLM
  - 综合考虑：消息内容、页面上下文、对话历史
  - 输出：TurnContextDecision
```

### TurnContextDecision

```python
class TurnContextDecision:
    route: str          # "guide" | "general" | "diagnostic"
    action_intent: str  # "resubmit" | "stop" | "delete" | None
    selected_job_name: str | None  # 显式作业绑定
    requested_scope: str  # "single" | "all" | "unspecified"
    rationale: str      # 用于调试的决策理由
```

### 路由语义

| 路由 | 触发条件 | 调用的子智能体 |
|------|----------|---------------|
| `guide` | "我该怎么...""你能做什么" | 仅引导智能体 |
| `general` | 问候语、简单问答 | 仅通用智能体 |
| `diagnostic` | 作业失败、资源问题、运维操作 | 规划者 → 探索者 → 执行者 → 验证者 |

---

## 流程控制

每次诊断流水线迭代结束后，协调者评估验证者的裁定结果：

| 裁定结果 | 协调者动作 |
|----------|-----------|
| `pass` | 终结 -- 向用户输出回答 |
| `risk` | 输出回答并附加风险警告标注 |
| `missing_evidence` | 回退到规划者重新规划（最多 `lead_max_rounds` 轮） |

### 终止条件

- `loop_round >= lead_max_rounds`（默认 8）→ 强制终结
- `no_progress_count >= no_progress_rounds`（默认 2）→ 中止并给出部分回答
- `pending_confirmation` → 暂停等待用户审批，下次调用时恢复

---

## 代码

| 组件 | 文件 |
|------|------|
| 协调者智能体 | `crater_agent/agents/coordinator.py` |
| 意图路由器 | `crater_agent/orchestrators/intent_router.py` |
| 多智能体编排器 | `crater_agent/orchestrators/multi.py` |
| 多智能体系统状态 | `crater_agent/orchestrators/state.py` |
