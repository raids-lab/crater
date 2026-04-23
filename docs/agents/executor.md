# Executor Agent

> Performs write operations based on evidence and user intent.

---

## Role

The Executor decides and performs write operations (stop job, resubmit, cordon node, etc.) based on the Planner's intent, the Explorer's evidence, and the user's explicit request. All write operations require user confirmation.

---

## Action Planning

```
Evidence + action_intent + user_message
  ↓
LLM decides actions (run_json):
  [
    { "tool": "stop_job", "args": {"job_name": "..."}, "title": "停止作业",
      "reason": "GPU 利用率为 0，建议释放资源", "depends_on": [] },
    { "tool": "resubmit_job", "args": {"job_name": "..."}, "title": "重新提交",
      "reason": "...", "depends_on": ["action_0"] }
  ]
  ↓
Execute in dependency order
  ↓
Write tool returns confirmation_required → pause for user approval
```

### Action Dependencies

Actions can declare `depends_on: [action_id]`:
- Actions only execute when all dependencies are satisfied
- If a dependency fails, dependent actions are skipped
- Enables multi-step operations (e.g., stop old → resubmit with new config)

---

## Key Behaviors

- **Never acts without explicit user intent**: The Executor only performs write operations that the user's original request implies
- **Confirmation is mandatory**: All write tools return `confirmation_required`; the user must approve in the frontend
- **Read + write access**: Can also call read-only tools if additional information is needed before acting

---

## Code

| Component | File |
|-----------|------|
| Executor agent | `crater_agent/agents/executor.py` |
