# Planner Agent

> Decomposes user requests into structured investigation plans.

---

## Role

The Planner analyzes the user's request and produces a `PlanArtifact` containing:

- **Goal**: One-sentence objective
- **Steps**: Ordered investigation/action steps
- **Candidate tools**: Read-only and write tools to try
- **Risk**: Low / medium / high assessment

The Planner does NOT execute tools. It only produces a plan for the Explorer and Executor to follow.

---

## Output

```python
class PlanOutput:
    goal: str                    # "诊断作业 OOM 失败原因"
    steps: list[str]             # ["查看作业详情", "分析 GPU 指标", ...]
    candidate_tools: list[str]   # ["get_job_detail", "query_job_metrics"]
    risk: str                    # "low" | "medium" | "high"
    raw_summary: str             # Free-text summary for coordinator
```

---

## Key Behaviors

- Respects page context boundaries (user sees own jobs, admin sees cluster)
- First-pass plans prioritize read-only tools
- Receives continuation state for replanning when Verifier returns `missing_evidence`
- Keeps plans concise (typically 3-5 steps)

---

## Code

| Component | File |
|-----------|------|
| Planner agent | `crater_agent/agents/planner.py` |
