# Single Agent (ReAct)

> The foundational agent mode — a single LLM with tools in a think-act-observe loop.

---

## Overview

The Single Agent uses LangGraph's `StateGraph` to implement a ReAct (Reasoning + Acting) loop. The LLM autonomously decides which tools to call and when to stop — there is no fixed workflow, intent classification, or stage transitions.

This is the core building block that all other agents reuse: Multi-Agent sub-agents, Task Agents (Approval), and Pipeline agents all run on the same graph with different configurations.

---

## Graph Structure

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

### Nodes

| Node | Role | Tools bound? |
|------|------|-------------|
| `agent_node` | LLM reasoning — decides next action or final answer | Yes (all enabled tools) |
| `tools_node` | Execute tool calls from LLM output | N/A (execution only) |
| `summarize_node` | Force synthesis when tool limit reached | No (LLM without tools) |

### Routing (`should_continue`)

```python
if tool_call_count >= max_tool_calls_per_turn:
    if LLM wanted more tools → "summarize"  # force conclusion
    else → "respond"  # natural end
elif pending_confirmation:
    → "respond"  # pause for user approval
elif LLM has tool_calls:
    → "tools"  # execute them
else:
    → "respond"  # final answer
```

---

## Key Features

### System Prompt Injection

On the first LLM call, the system prompt is dynamically built from:
- Base template (platform rules, working principles)
- User context (username, role, account)
- Page context (current job/node/pvc if any)
- Capabilities (enabled tools, confirm tools)
- Skills knowledge (diagnostic patterns from YAML)

### Tool Result Processing

Each tool result goes through a budget-aware pipeline:

```
Raw result (may be very large)
  ↓
Within per-tool token budget? → use as-is
  ↓ NO
LLM semantic extraction (10s timeout)
  ↓ FAIL
Hard truncation (head + tail)
```

### Deduplication

Same-turn tool calls with identical `(tool_name, args)` are skipped to prevent infinite loops.

### Confirmation Pause

When a write tool returns `confirmation_required`, the graph sets `pending_confirmation` in state and routes to END. The orchestrator yields this to the frontend, which shows a dialog. On resume, the graph continues from where it paused.

---

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `max_tool_calls_per_turn` | 15 | Safety cap on tool calls per ReAct loop |
| `tool_execution_timeout` | 30s | Per-tool HTTP timeout |
| `max_context_tokens` | 30000 | Trigger proactive compaction |
| `history_max_tokens` | 4000 | Budget for conversation history |

---

## Code

| Component | File |
|-----------|------|
| Graph builder | `crater_agent/agent/graph.py` |
| State definition | `crater_agent/agent/state.py` |
| System prompts | `crater_agent/agent/prompts.py` |
| Message compaction | `crater_agent/agent/compaction.py` |
| Orchestrator wrapper | `crater_agent/orchestrators/single.py` |
