# 规划者智能体

> 将用户请求分解为结构化的调查计划。

---

## 职责

规划者分析用户的请求，生成包含以下内容的 `PlanArtifact`：

- **目标**：一句话描述的目标
- **步骤**：有序的调查/操作步骤
- **候选工具**：可尝试的只读工具和写操作工具
- **风险**：低 / 中 / 高 等级评估

规划者不执行工具。它只为探索者和执行者生成待执行的计划。

---

## 输出

```python
class PlanOutput:
    goal: str                    # "诊断作业 OOM 失败原因"
    steps: list[str]             # ["查看作业详情", "分析 GPU 指标", ...]
    candidate_tools: list[str]   # ["get_job_detail", "query_job_metrics"]
    risk: str                    # "low" | "medium" | "high"
    raw_summary: str             # 供协调者参考的自由文本摘要
```

---

## 关键行为

- 遵守页面上下文边界（普通用户只能看到自己的作业，管理员可看到集群全局）
- 首次规划优先使用只读工具
- 当验证者返回 `missing_evidence` 时，接收续接状态进行重新规划
- 保持计划简洁（通常 3-5 个步骤）

---

## 代码

| 组件 | 文件 |
|------|------|
| 规划者智能体 | `crater_agent/agents/planner.py` |
