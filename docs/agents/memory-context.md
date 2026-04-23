# Memory & Context System

> How agents access conversation history, user identity, and page-level context.

---

## 1. Context Flow

Context flows from the Go backend through the FastAPI application to the agent's system prompt:

```
Go Backend (user session, page state)
  ↓ POST /chat
ChatRequest { session_id, message, context, user_id, page_context }
  ↓ build_request_context()
context dict { actor, page, capabilities, orchestration }
  ↓ build_system_prompt(context, skills_context)
System prompt (injected with user/page/capability details)
  ↓
LLM receives full context on first call
```

### Context Structure

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
        "job_name": "sg-xxx",    # populated when user is on job detail page
        "job_status": "Failed",
        "node_name": null,       # populated when on node detail page
        "pvc_name": null         # populated when on storage page
    },
    "capabilities": {
        "enabled_tools": [...],  # optional: restrict tool set
        "confirm_tools": [...]   # tools requiring user confirmation
    },
    "orchestration": {
        "mode": "single_agent"   # "single_agent" | "multi_agent"
    }
}
```

### What Context Controls

| Context field | What it affects |
|---------------|-----------------|
| `actor.role` | Tool visibility (admin sees all, user sees subset) |
| `actor.username` | Injected into system prompt for personalization |
| `page.job_name` | Binds agent attention to specific job |
| `page.url` | Admin route detection (`/admin/*` → admin role) |
| `capabilities.enabled_tools` | Restricts which tools are bound to LLM |
| `capabilities.confirm_tools` | Listed in prompt so agent knows which tools need confirmation |

---

## 2. Conversation History (Memory)

Conversation history is loaded from the Go backend on each turn. There is no agent-side persistent memory — the Go backend is the source of truth.

### Loading Strategy

```python
build_history_messages(
    history: list[dict],        # from Go backend
    max_tokens: int = 4000,     # token budget
    tool_result_max_chars: int = 1200,  # truncate tool results
    tool_error_max_chars: int = 1600,   # more room for errors
) -> list[BaseMessage]
```

**Algorithm**:
1. Iterate history in **reverse** (most recent first)
2. Convert each dict to LangChain message (`HumanMessage`, `AIMessage`, `ToolMessage`)
3. Truncate tool results with head+tail strategy
4. Accumulate token count; stop when budget exhausted
5. Reverse back to chronological order

### Truncation

Tool results are truncated using head+tail to preserve both the beginning (context) and end (errors, summaries):

```
[first 600 chars] ... (内容过长，已截断) ... [last 600 chars]
```

Error messages get more room (1600 chars) because error details are critical for diagnosis.

### Why No Agent-Side Memory

- The Go backend already stores full session history (`AgentSession`, `AgentMessage`, `AgentToolCall`)
- Agent-side memory would create consistency issues with the backend DB
- Token budget enforcement ensures history fits in context window regardless of conversation length
- Multi-turn continuity is handled by the Go backend's session management

---

## 3. System Prompt Building

The system prompt is the primary vehicle for context injection. It is built once per ReAct loop invocation:

```python
build_system_prompt(
    context: dict,
    skills_context: str = "",      # diagnostic knowledge from YAML
    is_first_time: bool = False,   # welcome addon for new users
    user_message: str = "",        # not currently used in template
) -> str
```

### Prompt Structure

```
1. Role definition (Crater 智能运维助手)
2. 22 working principles (evidence-first, minimal tools, confirmation, ...)
3. Platform specifications (resource limits, mount paths, quotas)
4. Resource recommendation flow
5. Admin-specific guidance (cluster diagnostics, if admin)
6. Observability & metrics (PromQL examples)
7. Tool selection guide
8. --- Dynamic injection ---
9. Current user: {username}, role: {role}, account: {account_name}
10. Current page: job={job_name} (status={status}) / node={node_name}
11. Available tools: {tool_list}
12. Confirm tools: {confirm_tool_list}
13. Skills context: diagnostic knowledge (from YAML files)
14. [Optional] First-time welcome addon
```

### Token Budget

| Section | Approx tokens |
|---------|---------------|
| Base template + principles | ~1200 |
| Platform specs + admin guidance | ~500 |
| Page context injection | ~50-200 |
| Capabilities detail | ~200-400 |
| Skills context | ~800-1500 |
| **Total** | **~2500-3500** |

---

## 4. Message Compaction

When the conversation grows too long, messages are compacted to stay within the context window.

### Proactive Compaction (before hitting limit)

```
estimated_tokens = count_message_tokens(all_messages)
available = max_context_tokens(30000) - tool_schema_budget(8000) - response_reserve(4000)

if estimated_tokens > available:
    → LLM-based compaction (summarize older messages, preserve recent)
    → If LLM compaction fails: hard truncation fallback
```

### LLM Compaction (`compact_messages_with_llm`)

1. Split messages: system (always preserved) + body
2. Partition body: compactable (older) + preserved (recent N messages)
3. Call LLM with compaction prompt (15s timeout)
4. Replace compactable messages with single summary `AIMessage`

### Hard Truncation Fallback (`_compact_messages_for_retry`)

- Keep: system message + last human message + most recent 6 messages
- Truncate each message: system (1600 chars), human (600), tool (800), AI (600)

### Reactive Recovery

On `BadRequestError("context_length_exceeded")`:
1. Try LLM compaction
2. Retry LLM call
3. If still fails: hard truncation + retry once more

---

## Code

| Component | File |
|-----------|------|
| History loading | `crater_agent/memory/session.py` |
| Context building | `crater_agent/app.py` (`build_request_context`) |
| System prompt | `crater_agent/agent/prompts.py` |
| Message compaction | `crater_agent/agent/compaction.py` |
| Token counter | `crater_agent/llm/tokenizer.py` |
| LLM client factory | `crater_agent/llm/client.py` |
| Agent state | `crater_agent/agent/state.py` |
| Config | `crater_agent/config.py` |
