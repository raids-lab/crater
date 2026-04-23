# Verifier Agent

> Quality gate — validates conclusions and flags evidence gaps.

---

## Role

The Verifier reviews the full pipeline output (plan, evidence, execution results) and produces a verdict. It acts as a quality gate before the Coordinator finalizes the answer.

---

## Verdicts

| Verdict | Meaning | Coordinator action |
|---------|---------|-------------------|
| `pass` | Evidence is sufficient, conclusions are sound | Finalize answer |
| `risk` | Answer is usable but has caveats | Finalize with warning |
| `missing_evidence` | Gaps or inconsistencies in evidence | Loop back to Planner |

---

## Key Behaviors

- **Prioritizes actual evidence over summaries**: If raw tool outputs are available, they take precedence over the Explorer's summary
- **Identifies specific gaps**: Returns actionable notes about what's missing
- **Strict JSON validation**: Uses `BaseRoleAgent.run_json` with fallback to `missing_evidence` on parse failure (safe default)
- **No tool access**: The Verifier is LLM-only — it reviews what others have gathered, not what it can find

---

## Code

| Component | File |
|-----------|------|
| Verifier agent | `crater_agent/agents/verifier.py` |
