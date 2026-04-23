# Skills System

> Diagnostic knowledge injection via YAML files — giving agents domain expertise without tool calls.

---

## Purpose

Skills are structured knowledge files that inject diagnostic patterns, trigger signals, and common solutions directly into the agent's system prompt. Unlike RAG (retrieval-augmented generation), skills are loaded deterministically — all skills are always available, not retrieved based on similarity search.

This approach trades token cost for reliability: the agent always has access to diagnostic knowledge for known failure patterns, without the latency or recall uncertainty of a retrieval step.

---

## Skill Format

Each skill is a YAML file in `crater_agent/skills/`:

```yaml
name: "OOM Diagnosis"
description: "Diagnose Out-of-Memory failures in job containers"

trigger_signals:
  exit_codes: [137, 143]
  event_reasons: ["OOMKilled", "Evicted"]
  log_keywords: ["out of memory", "OOM", "Cannot allocate memory"]
  job_status: ["Failed"]

diagnosis_knowledge: |
  当作业因 OOM 被杀时，通常表明：
  1. 请求内存不足 (memory request < actual usage)
  2. 容器进程内存泄漏
  3. 数据加载一次性加载过大

common_solutions:
  - condition: "GPU 显存不足"
    suggestion: "增加 GPU 显存分配或使用梯度检查点"
  - condition: "CPU 内存不足"
    suggestion: "减少 batch size 或启用数据流式处理"
```

### Fields

| Field | Purpose |
|-------|---------|
| `name` | Skill identifier |
| `description` | When this skill applies |
| `trigger_signals.exit_codes` | Container exit codes that activate this knowledge |
| `trigger_signals.event_reasons` | K8s event reasons |
| `trigger_signals.log_keywords` | Log patterns to watch for |
| `trigger_signals.job_status` | Job status values |
| `diagnosis_knowledge` | Free-text diagnostic reasoning for the LLM |
| `common_solutions` | Condition → suggestion mappings |

---

## Current Skills

| File | Coverage |
|------|----------|
| `oom_diagnosis.yaml` | OOMKilled, memory exhaustion |
| `image_pull_error.yaml` | ImagePullBackOff, registry failures |
| `queue_pending.yaml` | Scheduling delays, resource contention |
| `scheduling_failed.yaml` | Taint/affinity mismatches, insufficient resources |

---

## Injection Mechanism

```python
# loader.py
load_all_skills(skills_dir) → str
  1. Scan *.yaml files in skills_dir
  2. Parse each with yaml.safe_load()
  3. Format each with format_skill_for_prompt(skill)
  4. Concatenate under "## 诊断参考知识" header

# prompts.py
build_system_prompt(context, skills_context=skills_text)
  → Appends skills_context to system prompt template
```

The formatted output looks like:

```markdown
## 诊断参考知识

### OOM Diagnosis
OOM 诊断参考...

触发信号: exit_codes=[137, 143], event_reasons=["OOMKilled"]

诊断知识:
当作业因 OOM 被杀时...

常见方案:
- GPU 显存不足 → 增加 GPU 显存分配或使用梯度检查点
- CPU 内存不足 → 减少 batch size

### Image Pull Error
...
```

---

## Adding a New Skill

1. Create `crater_agent/skills/your_skill.yaml`
2. Fill in the YAML structure (name, trigger_signals, diagnosis_knowledge, common_solutions)
3. No code changes needed — `load_all_skills()` automatically discovers new YAML files

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| YAML over database | Version-controlled, reviewable, no runtime dependency |
| Full injection over RAG | Deterministic, no recall failures, acceptable token cost (~1500 tokens total) |
| Trigger signals as metadata | Enables future selective injection based on job state (not yet implemented) |

---

## Code

| Component | File |
|-----------|------|
| Skill loader | `crater_agent/skills/loader.py` |
| Skill YAML files | `crater_agent/skills/*.yaml` |
| Prompt injection | `crater_agent/agent/prompts.py` |
