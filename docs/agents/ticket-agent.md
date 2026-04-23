# Ticket Agent Framework

> Base class for automated ticket/order evaluation — enabling new ticket types with minimal code.

---

## Problem

Platform operations involve various types of tickets that need evaluation: job lock approvals, quota change requests, dataset access reviews, node maintenance windows. Each ticket type needs its own domain logic, but the evaluation infrastructure (run ReAct loop → extract verdict → fallback → error handling → audit) is identical.

Without abstraction, each ticket agent would duplicate ~150 lines of ReAct loop management, fallback logic, and error handling code.

---

## Design

`TicketAgent` is an abstract base class that provides the complete evaluation pipeline. Subclasses only define **five domain-specific methods**:

```
TicketAgent (base)
  │
  ├── ReAct loop execution        ← shared
  ├── Verdict extraction           ← shared
  ├── Fallback (BaseRoleAgent)     ← shared
  ├── Error handling (never raise) ← shared
  ├── Trace collection             ← shared
  │
  └── Subclass defines:
        ├── allowed_tools()        → which tools to bind
        ├── system_prompt()        → domain evaluation rules
        ├── build_user_message()   → request → LLM input
        ├── extract_verdict()      → LLM output → structured result
        └── default_verdict()      → safe fallback when everything fails
```

### Class Hierarchy

```
TicketAgent[TRequest, TVerdict]  (abstract, generic)
  │
  ├── ApprovalAgent              Job lock approval evaluation
  │     TRequest = ApprovalEvalRequest
  │     TVerdict = ApprovalEvalResponse
  │
  ├── QuotaAgent (future)        Resource quota change evaluation
  │     TRequest = QuotaEvalRequest
  │     TVerdict = QuotaEvalResponse
  │
  ├── DatasetAccessAgent (future) Dataset access review
  │
  └── MaintenanceAgent (future)   Node maintenance scheduling
```

---

## Creating a New Ticket Agent

### Step 1: Define request and response models

```python
class QuotaEvalRequest(BaseModel):
    account_id: int
    account_name: str
    requested_gpu: int
    requested_cpu: int
    reason: str

class QuotaEvalResponse(BaseModel):
    verdict: str = "escalate"  # "approve" | "adjust" | "escalate"
    confidence: float = 0.5
    reason: str = ""
    adjusted_gpu: int | None = None
    adjusted_cpu: int | None = None
    trace: list[dict[str, Any]] = []
```

### Step 2: Implement the agent

```python
class QuotaAgent(TicketAgent[QuotaEvalRequest, QuotaEvalResponse]):

    def __init__(self, **kwargs):
        super().__init__(agent_id="quota", llm_purpose="quota", **kwargs)

    def allowed_tools(self) -> list[str]:
        return ["check_quota", "get_realtime_capacity", "list_cluster_jobs"]

    def system_prompt(self) -> str:
        return "你是配额审批助手。评估用户的资源配额变更请求..."

    def build_user_message(self, request: QuotaEvalRequest) -> str:
        return f"请评估配额变更：{request.account_name} 申请 GPU={request.requested_gpu}..."

    def extract_verdict(self, text: str) -> QuotaEvalResponse | None:
        # Parse JSON from LLM output
        ...

    def default_verdict(self, *, reason: str = "") -> QuotaEvalResponse:
        return QuotaEvalResponse(verdict="escalate", reason=reason or "转交管理员")
```

### Step 3: Register the endpoint

```python
# app.py
@app.post("/evaluate/quota")
async def evaluate_quota(request: QuotaEvalRequest):
    agent = QuotaAgent()
    return await agent.evaluate(request)
```

### Step 4: Add Go backend hook (same pattern as approval)

That's it — no changes to the ReAct graph, tool executor, or base infrastructure.

---

## What the Base Class Handles

| Concern | How |
|---------|-----|
| ReAct loop | `create_agent_graph` with `capabilities.enabled_tools` |
| Tool execution | `GoBackendToolExecutor` (reused) |
| Token management | Graph's built-in compaction and tool result budgets |
| Tool call limit | Graph's `summarize_node` when limit reached |
| Verdict extraction | Subclass's `extract_verdict()` on last AI message |
| Fallback | `BaseRoleAgent.run_json()` with no tools bound |
| JSON repair | `run_json`'s built-in repair loop |
| Error recovery | `try/except` wrapping entire `evaluate()` → `default_verdict()` |
| Trace collection | Automatic from message history |
| Timeout | Caller wraps with `asyncio.wait_for()` |

---

## Optional Hooks

Beyond the five required methods, subclasses can override:

| Hook | Default | Override when |
|------|---------|--------------|
| `build_context(request)` | `{capabilities, actor: system}` | Need request-specific context (user_id, session_id) |
| `fallback_prompt()` | Generic ticket evaluation prompt | Need domain-specific fallback instructions |
| `set_trace(verdict, trace)` | Sets `verdict.trace = trace` | Verdict model has different trace field |
| `_parse_fallback_result(result)` | `extract_verdict(json.dumps(result))` | Fallback JSON has different structure |

---

## Code

| Component | File |
|-----------|------|
| Base class | `crater_agent/agents/ticket_base.py` |
| Approval agent | `crater_agent/agents/approval.py` |
