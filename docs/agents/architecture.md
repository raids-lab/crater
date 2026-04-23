# Mops System Architecture

This document describes the complete architecture of the Mops multi-agent system, covering data flow, state management, tool execution, token management, and safety mechanisms.

---

## 1. System Layers

The system consists of five layers, from user-facing to infrastructure:

```
┌──────────────────────────────────────────────────────────────┐
│  (1) Application Layer                                       │
│  FastAPI endpoints, SSE streaming, request routing           │
├──────────────────────────────────────────────────────────────┤
│  (2) Orchestration Layer                                     │
│  Single-agent ReAct / Multi-agent coordinator pipeline       │
├──────────────────────────────────────────────────────────────┤
│  (3) Agent Layer                                             │
│  Planner, Explorer, Executor, Verifier, Approval, ...        │
├──────────────────────────────────────────────────────────────┤
│  (4) Tool & Knowledge Layer                                  │
│  Tool definitions, executors, selectors, diagnostic skills   │
├──────────────────────────────────────────────────────────────┤
│  (5) Infrastructure Layer                                    │
│  Go backend (K8s, DB, Prometheus), local kubectl/PromQL      │
└──────────────────────────────────────────────────────────────┘
```

### Layer responsibilities

| Layer | Responsibility | Key files |
|-------|---------------|-----------|
| Application | HTTP routing, request parsing, SSE emission | `app.py` |
| Orchestration | Agent lifecycle, stage transitions, state management | `orchestrators/single.py`, `orchestrators/multi.py` |
| Agent | Domain-specific reasoning, tool selection, output formatting | `agents/*.py` |
| Tool & Knowledge | Tool schema, execution, role filtering, diagnostic knowledge | `tools/`, `skills/` |
| Infrastructure | Actual K8s/Prometheus/DB operations, auth, audit | Go backend, `tools/local_executor.py` |

---

## 2. Data Flow

### 2.1 Single-Agent Request Flow

```
Go Backend                   crater-agent                    Go Backend (tools)
    │                            │                                │
    │  POST /chat (SSE)          │                                │
    ├───────────────────────────>│                                │
    │                            │  build_history_messages()      │
    │                            │  build_system_prompt()         │
    │                            │  create_agent_graph()          │
    │                            │                                │
    │  SSE: agent_status         │  ┌─── ReAct Loop ───┐         │
    │<───────────────────────────│  │                   │         │
    │                            │  │  LLM thinks       │         │
    │  SSE: tool_call_started    │  │  ↓                │         │
    │<───────────────────────────│  │  Select tool      │         │
    │                            │  │  ↓                │         │
    │                            │  │  Execute ─────────│────────>│
    │                            │  │  ↓                │         │
    │  SSE: tool_call_result     │  │  Observe result   │<────────│
    │<───────────────────────────│  │  ↓                │         │
    │                            │  │  LLM thinks again │         │
    │                            │  │  ↓                │         │
    │                            │  │  No more tools    │         │
    │  SSE: agent_response       │  └───────────────────┘         │
    │<───────────────────────────│                                │
    │  SSE: done                 │                                │
    │<───────────────────────────│                                │
```

### 2.2 Multi-Agent Request Flow

```
Go Backend                   crater-agent
    │                            │
    │  POST /chat                │
    ├───────────────────────────>│
    │                            │
    │                      IntentRouter (deterministic)
    │                            │
    │                      Coordinator (LLM routing)
    │                        ┌───┴───┐
    │                   guide/general  diagnostic
    │                        │         │
    │                   [Agent]    Planner (LLM)
    │                        │         │
    │                   response   PlanArtifact
    │                                  │
    │                             Explorer (tool loop)
    │                                  │
    │                          ObservationArtifact
    │                                  │
    │                            Executor (tool loop)
    │                                  │
    │                          ExecutionArtifact
    │                                  │
    │                             Verifier (LLM)
    │                                  │
    │                         pass / risk / missing
    │                              │
    │                        Coordinator decides:
    │                        loop back or finalize
    │                              │
    │  SSE: final_answer           │
    │<─────────────────────────────│
```

### 2.3 Task-Agent Request Flow (e.g., Approval)

```
Go Backend                   crater-agent
    │                            │
    │  POST /evaluate/approval   │
    │  (synchronous, not SSE)    │
    ├───────────────────────────>│
    │                            │
    │                      ApprovalAgent
    │                      create_agent_graph()
    │                      (restricted tool whitelist)
    │                            │
    │                      ReAct loop (max 8 tools)
    │                      ↓
    │                      Extract verdict JSON
    │                      ↓ (fallback if needed)
    │                      BaseRoleAgent.run_json()
    │                            │
    │  JSON: ApprovalEvalResponse│
    │<───────────────────────────│
```

---

## 3. State Management

### 3.1 Single-Agent State (`CraterAgentState`)

```python
class CraterAgentState(MessagesState):
    context: dict           # User, page, capabilities from Go backend
    tool_call_count: int    # Safety limit counter
    attempted_tool_calls: dict  # Deduplication by (tool, args) signature
    pending_confirmation: dict | None  # Pause graph for user approval
    force_limit_reached: bool  # Tool limit hit flag
    trace: list[dict]       # Audit trail (append-only)
```

State lives for one turn. No cross-turn persistence in single-agent mode.

### 3.2 Multi-Agent State (`MASState`)

```
MASState
  ├── goal: GoalArtifact
  │     user_message, actor_role, page_context, routing
  ├── observation: ObservationArtifact
  │     evidence[], facts[], open_questions[], stage_complete
  ├── plan: PlanArtifact
  │     steps[], candidate_tools[], risk
  ├── execution: ExecutionArtifact
  │     actions[], results[], summary
  ├── actions: List[MultiAgentActionItem]
  │     pending / completed / awaiting_confirmation
  ├── flow_control
  │     loop_round, no_progress_count, stop_reason
  ├── usage_summary: MultiAgentUsageSummary
  │     llm_calls, tokens, tool_calls, evidence_items
  └── runtime_config: MASRuntimeConfig
        lead_max_rounds=8, subagent_max_iterations=25
```

MASState supports workflow checkpoint for cross-turn persistence (confirmation flow).

---

## 4. Tool Execution

### 4.1 Execution Pipeline

```
LLM emits tool_call
  ↓
is_tool_allowed_for_role(agent_role, tool_name)?
  ├─ NO → error: tool policy violation
  └─ YES
      ↓
is_actor_allowed_for_tool(actor_role, tool_name)?
  ├─ NO → error: requires admin privileges
  └─ YES
      ↓
LocalToolExecutor.supports(tool_name)?
  ├─ YES → execute locally (kubectl, PromQL, web)
  └─ NO → GoBackendToolExecutor (HTTP POST)
              ↓
          Go Backend validates, executes, returns result
              ↓
          status: success | error | confirmation_required
```

### 4.2 Tool Categories

| Category | Count | Execution | Auth | Example |
|----------|-------|-----------|------|---------|
| Diagnosis (read) | ~10 | Go backend | User/Admin | `get_job_detail`, `diagnose_job` |
| Metrics (read) | ~15 | Go backend | User/Admin | `query_job_metrics`, `check_quota` |
| Admin read | ~20 | Go backend | Admin only | `list_cluster_nodes`, `get_node_detail` |
| K8s direct | ~15 | Local kubectl | Admin only | `k8s_list_pods`, `k8s_describe_resource` |
| Prometheus direct | 1 | Local HTTP | Admin only | `prometheus_query` |
| Write (confirm) | ~15 | Go backend | User+Confirm | `stop_job`, `cordon_node` |
| Approval | 1 | Go backend | System | `get_approval_history` |

### 4.3 Confirmation Flow

Write tools return `confirmation_required` instead of executing immediately:

```
Agent calls stop_job("my-training")
  ↓
Go backend returns:
  { status: "confirmation_required",
    confirm_id: "abc123",
    description: "停止作业 my-training",
    risk_level: "medium" }
  ↓
Orchestrator pauses, returns confirmation to frontend
  ↓
Frontend shows dialog to user
  ↓
User approves
  ↓
Resume call with confirm_id + approved status
  ↓
Go backend executes the operation
  ↓
Result flows back to agent, loop continues
```

---

## 5. Token Management

Context window is a finite resource. The system manages it at four levels:

### 5.1 Input History

```
Conversation history from Go backend (reverse chronological)
  ↓
build_history_messages(max_tokens=4000)
  ↓
Truncate tool results to 160 chars (head+tail)
  ↓
Stop loading when token budget exhausted
  ↓
Reverse to chronological order
```

### 5.2 Tool Result Budgets

Each tool has a per-result token budget:

| Tool | Budget | Rationale |
|------|--------|-----------|
| `get_job_logs` | 4000 | Logs can be very long |
| `diagnose_job` | 4000 | Structured diagnosis output |
| `get_job_detail` | 3000 | Job metadata |
| `prometheus_query` | 2000 | Time series data |
| Default | 3000 | - |

Over-budget results go through LLM semantic extraction first, then hard truncation as fallback.

### 5.3 Proactive Compaction

Before hitting context limits:

```
estimated_tokens = count_message_tokens(all_messages)
available = max_context_tokens - tool_schema_budget(8000) - response_budget(4000)

if estimated_tokens > available:
  try: LLM-based compaction (preserve recent, summarize older)
  except: hard truncation (keep system + last 6 messages)
```

### 5.4 Reactive Recovery

On `BadRequestError("context_length_exceeded")`:

```
catch error
  → compact_messages_with_llm()
  → retry LLM call with compacted messages
  → if still fails: _compact_messages_for_retry() (hard truncation)
  → retry once more
```

---

## 6. Safety Mechanisms

### 6.1 Tool Policy (Agent Level)

Agent roles define which tools they can call:
- Explorer: read-only tools only (hardcoded check)
- Executor: read + write tools
- Approval: fixed whitelist of 8 tools
- General/Guide/Verifier/Planner: no tools (LLM-only)

### 6.2 Actor Policy (User Level)

User roles define which tools are visible:
- Admin: all tools
- User: ~30 safe tools (no cluster management, no node operations)

### 6.3 Confirmation Barrier (Operation Level)

Write tools (stop, delete, create, cordon, drain, ...) always require user confirmation before execution. The graph pauses and cannot proceed without explicit approval.

### 6.4 Rate Limiting (System Level)

For automated agents (e.g., approval hook):
- Token bucket: max N calls per minute
- Concurrency semaphore: max M concurrent evaluations
- Circuit breaker: consecutive failures trigger cooldown
- Timeout: total evaluation time capped

### 6.5 Panic Recovery (Process Level)

All agent evaluation paths in the Go backend are wrapped with `defer recover()`:

```go
func() {
    defer func() {
        if r := recover(); r != nil {
            klog.Errorf("agent panic recovered: %v", r)
        }
    }()
    // agent evaluation call
}()
```

Agent failures never crash the main application or block user operations.

---

## 7. Configuration

### 7.1 LLM Clients (`config/llm-clients.json`)

Multiple LLM providers can be configured per agent role:

```json
{
  "default": { "model": "...", "base_url": "...", "temperature": 0.1 },
  "planner": { "model": "...", "temperature": 0.0 },
  "explorer": { "model": "...", "max_tokens": 2048 }
}
```

### 7.2 MAS Runtime (`config/agent-runtime.json`)

Guardrails for multi-agent orchestration:

```json
{
  "lead_max_rounds": 8,
  "subagent_max_iterations": 25,
  "no_progress_rounds": 2,
  "max_actions_per_round": 4
}
```

### 7.3 Platform Runtime (`config/platform-runtime.yaml`)

Platform-specific endpoints and access:

```yaml
kubernetes:
  kubeconfig_path: /path/to/kubeconfig
prometheus:
  endpoint: http://prometheus:9090
harbor:
  endpoint: https://harbor.example.com
```

---

## 8. Evaluation Framework

### 8.1 Scenario-Based Benchmarking

Each scenario defines:
- User query and page context
- Available tools and mock responses
- Ground truth: expected tools, root cause keywords, suggestions

### 8.2 Metrics

| Metric | What it measures |
|--------|-----------------|
| Tool Selection F1 | Did the agent call the right tools? |
| Root Cause Hit | Did the agent identify the correct root cause? |
| Suggestion Relevance | Were suggestions appropriate and safe? |
| Permission Compliance | Did confirm tools trigger confirmation? |
| Efficiency Ratio | Actual vs optimal tool calls |

### 8.3 Execution Modes

| Mode | Data source | Use case |
|------|------------|----------|
| Snapshot | Mock JSON responses | CI regression, model comparison |
| Live-readonly | Real backend, read-only | Smoke tests, tool validation |

---

## Code References

| Component | File |
|-----------|------|
| ReAct graph | `crater_agent/agent/graph.py` |
| Agent state | `crater_agent/agent/state.py` |
| System prompts | `crater_agent/agent/prompts.py` |
| Message compaction | `crater_agent/agent/compaction.py` |
| Base agent class | `crater_agent/agents/base.py` |
| Tool definitions | `crater_agent/tools/definitions.py` |
| Tool executors | `crater_agent/tools/executor.py` |
| Tool selector | `crater_agent/tools/tool_selector.py` |
| Local executor | `crater_agent/tools/local_executor.py` |
| Single orchestrator | `crater_agent/orchestrators/single.py` |
| Multi orchestrator | `crater_agent/orchestrators/multi.py` |
| MAS state | `crater_agent/orchestrators/state.py` |
| Skill loader | `crater_agent/skills/loader.py` |
| Session memory | `crater_agent/memory/session.py` |
| Configuration | `crater_agent/config.py` |
| Application | `crater_agent/app.py` |
| Eval runner | `crater_agent/eval/runner.py` |
| Eval metrics | `crater_agent/eval/metrics.py` |
