# Explorer Agent

> Evidence collector — gathers information through read-only tools.

---

## Role

The Explorer executes the Planner's investigation steps by iteratively selecting and calling read-only tools. It builds an evidence base that the Verifier can validate and the Executor can act on.

---

## Tool Loop

```
Planner's candidate_tools + enabled_tools
  ↓
LLM selects next batch of tools (run_json)
  ↓
Execute each tool via GoBackendToolExecutor
  ↓
Build compact_evidence (structured summaries)
  ↓
LLM decides: more tools needed?
  ├─ YES → select next batch (loop)
  └─ NO → summarize evidence
```

### Constraints

- **Read-only only**: Hard filter against `READ_ONLY_TOOL_NAMES` — write tools are rejected even if the LLM requests them
- **Deduplication**: Skips tool calls with identical `(tool_name, args)` signatures already attempted
- **Iteration limit**: `subagent_max_iterations` (default 25) caps total tool calls

---

## Output

```python
class ObservationArtifact:
    summary: str              # Natural language evidence summary
    facts: list[str]          # Extracted facts
    open_questions: list[str] # Unresolved questions
    evidence: list[dict]      # Compact evidence items
    stage_complete: bool      # Whether exploration is complete
```

---

## Code

| Component | File |
|-----------|------|
| Explorer agent | `crater_agent/agents/explorer.py` |
