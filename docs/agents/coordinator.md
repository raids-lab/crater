# Coordinator Agent

> The orchestration brain of the Multi-Agent System — routes requests and manages stage transitions.

---

## Role

The Coordinator is the only agent that sees the full MAS state. It makes two types of decisions:

1. **Routing**: Where should this request go? (guide / general / diagnostic)
2. **Flow control**: Should the pipeline continue, loop back, or finalize?

The Coordinator does NOT call tools or produce user-facing content. It only makes structural decisions.

---

## Routing Pipeline

```
User message arrives
  ↓
IntentRouter (deterministic)
  - Regex job name extraction
  - "all" keyword detection
  - Continuation signal detection
  ↓
Coordinator LLM
  - Considers: message content, page context, conversation history
  - Outputs: TurnContextDecision
```

### TurnContextDecision

```python
class TurnContextDecision:
    route: str          # "guide" | "general" | "diagnostic"
    action_intent: str  # "resubmit" | "stop" | "delete" | None
    selected_job_name: str | None  # Explicit job binding
    requested_scope: str  # "single" | "all" | "unspecified"
    rationale: str      # Explanation for debugging
```

### Route semantics

| Route | When | Sub-agents invoked |
|-------|------|-------------------|
| `guide` | "How do I...", "What can you do" | Guide Agent only |
| `general` | Greetings, simple Q&A | General Agent only |
| `diagnostic` | Job failures, resource issues, operations | Planner → Explorer → Executor → optional Verifier → Final answer |

---

## Flow Control

When the diagnostic pipeline believes it is ready to finalize, the Coordinator may choose to run the Verifier first for higher-risk or lower-confidence cases:

| Verdict | Coordinator action |
|---------|-------------------|
| `pass` | Finalize — emit answer to user |
| `risk` | Finalize, but surface caveats / incomplete items |
| `missing_evidence` | Loop back to Planner for replanning (up to `lead_max_rounds`) |

Additional routing safety:

- `guide/general` are only used for non-operational help/Q&A turns
- If LLM routing returns an inconsistent combination such as `help + write`, runtime coercion forces it back to `agent`
- `replan` should not repeat forever without new evidence; runtime now blocks consecutive no-progress replans and pushes the loop back to `observe` / `act`

### Termination conditions

- `loop_round >= lead_max_rounds` (default 8) → forced finalization
- `no_progress_count >= no_progress_rounds` (default 2) → abort with partial answer
- `pending_confirmation` → pause for user approval, resume on next call

---

## Code

| Component | File |
|-----------|------|
| Coordinator agent | `crater_agent/agents/coordinator.py` |
| Intent router | `crater_agent/orchestrators/intent_router.py` |
| Multi-agent orchestrator | `crater_agent/orchestrators/multi.py` |
| MAS state | `crater_agent/orchestrators/state.py` |
