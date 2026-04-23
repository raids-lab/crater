# Approval Agent

> Ticket agent for automated evaluation of job lock approval orders.
> Inherits from [TicketAgent](ticket-agent.md) base class.

---

## Purpose

When users request to extend the lock on a running job (preventing auto-cleanup), the Approval Agent evaluates whether the request should be automatically approved, given emergency protection, or escalated to a human administrator.

This is the first instance of the `TicketAgent` framework. It defines only the domain-specific parts (tools, prompt, verdict parsing); the entire evaluation pipeline (ReAct loop, fallback, error handling, trace collection) is inherited from the base class.

---

## Design

### Integration Point

```
User submits lock request (POST /v1/approvalorder)
  ↓
Go Backend: checkAutoApprovalEligibility()
  ├─ Pass (< 12h, cooldown OK) → system_auto approve → done
  └─ Fail → Agent evaluation hook
              ↓
         Go Backend calls POST /evaluate/approval on crater-agent
              ↓
         ApprovalAgent runs ReAct loop with restricted tools
              ↓
         Returns structured verdict
              ↓
         Go Backend applies verdict (approve / emergency lock / escalate)
```

### Three Verdicts

| Verdict | Condition | Action |
|---------|-----------|--------|
| `approve` | Normal job, request < 48h, resources OK | Lock for requested hours, order = Approved |
| `approve_emergency` | Job about to be cleaned (`reminded=true`), request >= 48h | Lock for 6h immediately, order stays Pending for admin |
| `escalate` | Request >= 48h, or resource concerns | No lock, order stays Pending with agent report |

### Emergency Detection

The agent detects emergency status through the `reminded` field returned by `get_job_detail`:

- `reminded = true` → job has received a cleanup reminder and will be deleted within 24h
- In this case, even if the request is long (>= 48h), the agent locks 6h immediately to prevent the job from being killed, while escalating the remaining duration to admin

### Duration Adjustment

The agent can approve with fewer hours than requested:

- `approved_hours = null` → use the user's original request
- `approved_hours = N` → lock for N hours instead (only allows shortening, never extending)

Go backend enforces: `approved_hours` must be positive and less than the original request.

---

## Tool Whitelist

The agent has access to 7 read-only tools (vs. 88 total in the system):

| Tool | Why |
|------|-----|
| `get_job_detail` | Job type, resources, runtime, `reminded` status |
| `get_job_events` | Scheduling and restart events |
| `query_job_metrics` | GPU utilization trends (skipped for interactive jobs) |
| `check_quota` | User quota usage |
| `get_realtime_capacity` | Cluster resource availability |
| `list_cluster_jobs` | User's other running jobs (total resource footprint) |
| `get_approval_history` | Recent approval frequency |

Tool schema overhead: ~500 tokens (vs ~8000 for full tool set).

---

## Evaluation Logic

### Decision Flow

```
get_job_detail → check job type, resources, reminded status
  ↓
Is reminded=true?
  ├─ YES (emergency track)
  │   request < 48h → approve
  │   request >= 48h → approve_emergency (6h) + escalate remainder
  │
  └─ NO (normal track)
      request < 48h → check resources
      │   quota OK, no queue pressure → approve
      │   resource concerns → escalate
      request >= 48h → escalate with analysis
```

### Escalation Triggers (override any track)

Even for short requests (< 48h), the agent escalates when:

- User already occupies large amounts of high-end GPU resources
- Same resource type has significant queue backlog
- Batch job GPU utilization is near zero sustained
- User has requested 3+ approvals in the past 7 days
- User exceeds their quota allocation

### Interactive Job Handling

Jupyter and WebIDE sessions get lenient treatment:

- GPU utilization is NOT used as a primary signal (users may be configuring environments, debugging, or running CPU-bound code)
- Decision is based on: session duration, approval frequency, queue pressure

---

## Fallback Strategy

```
ReAct loop (max 15 tool calls, global setting)
  ↓
Agent outputs verdict JSON naturally?
  ├─ YES → use it
  └─ NO (graph hits tool limit → summarize node)
      ↓
  Extract verdict from summarize output?
  ├─ YES → use it
  └─ NO → Fallback: BaseRoleAgent.run_json() with no tools
      ↓
  JSON repair succeeds?
  ├─ YES → use it
  └─ NO → Keyword detection ("通过"/"转交")
      ↓
  Keywords found?
  ├─ YES → infer verdict with confidence=0.5
  └─ NO → default: escalate with confidence=0.3
```

The agent **never raises exceptions** to the caller. All failures result in a safe `escalate` verdict.

---

## Rate Limiting (Go Backend Side)

Three-layer protection prevents the agent from being overwhelmed:

| Layer | Mechanism | Default |
|-------|-----------|---------|
| Rate limit | Token bucket per minute | 10/min |
| Concurrency | Channel-based semaphore | 3 concurrent |
| Circuit breaker | Consecutive failure count | Open after 5, cooldown 60s |

When any layer rejects a request, the order falls back to manual review. No request is lost.

---

## Audit Trail

Every evaluation is recorded:

1. **AgentReport field** on the ApprovalOrder (JSON with verdict, confidence, reason, tool trace)
2. **ReviewSource field**: `system_auto` / `agent_auto` / `admin_manual`
3. **OperationLog entry**: operator=agent, type=approval_evaluation

The frontend displays the agent report in a dedicated tab on the approval detail page.

---

## Code

| Component | File |
|-----------|------|
| Agent class | `crater_agent/agents/approval.py` |
| FastAPI endpoint | `crater_agent/app.py` (`POST /evaluate/approval`) |
| Go evaluator service | `backend/internal/service/agent_approval.go` |
| Go hook | `backend/internal/handler/approvalorder.go` (CreateApprovalOrder) |
| Tool handler | `backend/internal/handler/agent/tools_readonly.go` (toolGetApprovalHistory) |
| DB model | `backend/dao/model/approvalorder.go` (ReviewSource, AgentReport) |
| Config | `backend/pkg/config/config.go` (Agent.ApprovalHook) |
| Migration | `backend/hack/sql/20260418_approval_agent.sql` |
| Frontend types | `frontend/src/services/api/approvalorder.ts` |
| Frontend badge | `frontend/src/components/badge/approvalorder-badge.tsx` (ReviewSourceBadge) |
| Frontend detail tab | `frontend/src/routes/admin/more/orders/$id.tsx` (Agent tab) |
| Frontend list column | `frontend/src/components/approval-order/approval-order-data-table.tsx` |
| Spec | `docs/specs/agent-approval-hook.md` |
