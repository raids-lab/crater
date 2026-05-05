# Inspection Pipeline (Smart Patrol)

> Automated cluster health inspection — a scheduled pipeline agent that collects evidence, reasons about anomalies, and generates structured operational reports.

---

## Overview

The Inspection Pipeline is a **Task-Specific agent** in the Mops framework that runs as a scheduled background task (not user-facing chat). It periodically collects cluster metrics, job status, and resource utilization, then uses LLM analysis to generate structured inspection reports visible on the admin dashboard.

Unlike chat agents that respond to user messages, the Inspection Pipeline is triggered by cron jobs and operates autonomously with a fixed report schema that the frontend renders as dashboard cards.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ (1) Trigger Layer — CronJobManager (Go backend)                 │
│     robfig/cron scheduler → patrol function registry            │
│     API: POST /v1/operations/patrol/{jobName} (manual trigger)  │
└──────────────────────────────┬──────────────────────────────────┘
                               │ HTTP POST /pipeline/admin-ops-report
                               │ Headers: X-Agent-Internal-Token
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ (2) Pipeline Layer — crater-agent (FastAPI)                      │
│     router.py → ops_report.py (orchestration)                   │
│     ┌─────────────────────────────────────────────────────────┐ │
│     │  Step 1: Collect compute domain                         │ │
│     │  Step 2: Collect storage domain                         │ │
│     │  Step 3: Collect network domain                         │ │
│     │  Step 4: Fetch previous report (trend comparison)       │ │
│     │  Step 5: LLM analysis (or deterministic fallback)       │ │
│     │  Step 6: Build pipeline payload + audit items           │ │
│     │  Step 7: Persist to database                            │ │
│     └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────┬──────────────────────────────────┘
                               │ Tool calls via PipelineToolClient
                               │ POST /api/agent/tools/execute
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ (3) Tool Layer — Go backend tool handlers                       │
│     get_admin_ops_report (aggregates 7 sub-tool results)       │
│     list_storage_pvcs, get_storage_capacity_overview            │
│     get_node_network_summary, diagnose_distributed_job_network  │
│     get_latest_audit_report, save_audit_report                  │
└──────────────────────────────┬──────────────────────────────────┘
                               │ JSON persisted via save_audit_report
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ (4) Storage Layer — PostgreSQL                                   │
│     ops_audit_reports (report metadata + report_json JSONB)     │
│     ops_audit_items (per-job action items with categories)      │
└──────────────────────────────┬──────────────────────────────────┘
                               │ REST API: GET /admin/agent/ops-reports/*
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ (5) Presentation Layer — Frontend (React)                        │
│     OpsReportTab.tsx — 60s polling for latest report            │
│     ├─ ReportSummaryCard (executive summary + stat cards)      │
│     ├─ FailureAnalysisCard (categories table + affected users)  │
│     ├─ SuccessAnalysisCard (resource efficiency metrics)        │
│     ├─ ResourceUtilizationCard (GPU/CPU/Memory bars + alerts)   │
│     ├─ RecommendationsCard (severity-colored action items)      │
│     └─ ReportHistoryTable (paginated past reports)              │
└─────────────────────────────────────────────────────────────────┘
```

---

## Data Collection

### Compute Domain

The `get_admin_ops_report` backend tool aggregates data from seven sub-tools in a single call:

| Sub-tool | Data |
|----------|------|
| Job query (completed) | Success job samples (configurable limit, default 5) |
| Job query (failed) | Failed job samples (configurable limit, default 5) |
| `get_cluster_health_overview` | Cluster-wide health metrics |
| `get_failure_statistics` | Failure category breakdown |
| `detect_idle_jobs` | Low GPU utilization detection |
| `list_cluster_nodes` | Node status snapshots |
| `get_realtime_capacity` | Current resource availability |

### Storage Domain

| Tool | Data |
|------|------|
| `list_storage_pvcs` | PVC inventory, unbound/anomalous PVCs |
| `get_storage_capacity_overview` | Storage utilization, capacity pressure |

### Network Domain

| Tool | Data |
|------|------|
| `get_node_network_summary` | Network health per node, degraded interfaces |
| `diagnose_distributed_job_network` | NCCL/RDMA diagnostics for distributed jobs |

---

## Analysis Pipeline

### Deterministic Baseline

`build_deterministic_ops_report()` produces a structured report from raw data using pure Python logic (no LLM). This serves as:

1. **Source of truth for numbers** — all counters, percentages, and aggregates are computed deterministically
2. **Fallback** — if LLM analysis fails or times out, the deterministic report is used directly
3. **Merge target** — LLM-generated text fields are merged into the deterministic structure

### LLM Enhancement

`analyze_ops_report_with_llm()` sends the baseline data to the `ops_report` LLM client (DashScope Qwen) with a structured prompt. The LLM enhances three categories of fields:

| Field | Source | What LLM adds |
|-------|--------|---------------|
| `executive_summary` | LLM | Natural language overview |
| `failure_analysis.patterns` | LLM | Cross-job failure pattern analysis |
| `recommendations` | LLM | Prioritized action items with severity |

### Merge Strategy

Numerical fields always come from the deterministic report. The LLM can only override text-based analysis fields. If the LLM output fails JSON parsing, the deterministic report is returned unchanged.

```
Deterministic report (numbers + template text)
  ↓
LLM report (executive_summary + patterns + recommendations)
  ↓
_merge_llm_report() — only overwrite if LLM field is non-empty
  ↓
Final report_json → saved to DB
```

---

## Report Schema

```json
{
  "executive_summary": "string (2-3 sentences)",
  "job_overview": {
    "total": "int",
    "success": "int",
    "failed": "int",
    "pending": "int",
    "success_rate": "float (percentage)",
    "delta": {
      "total": "int (vs previous report)",
      "failed": "int",
      "pending": "int"
    }
  },
  "failure_analysis": {
    "categories": [
      {
        "reason": "string (e.g., ContainerError, OOMKilled)",
        "count": "int",
        "top_job": { "name": "string", "owner": "string" }
      }
    ],
    "top_affected_users": ["string"],
    "patterns": "string (failure pattern analysis)"
  },
  "success_analysis": {
    "avg_duration_by_type": { "training": "float (seconds)" },
    "resource_efficiency": {
      "avg_cpu_ratio": "float (0-1)",
      "avg_gpu_ratio": "float (0-1)",
      "avg_memory_ratio": "float (0-1)"
    }
  },
  "resource_utilization": {
    "cluster_gpu_avg": "float (percentage)",
    "cluster_cpu_avg": "float (percentage)",
    "cluster_memory_avg": "float (percentage)",
    "over_provisioned_count": "int",
    "idle_gpu_jobs": "int",
    "node_hotspots": ["string (node names)"]
  },
  "recommendations": [
    { "severity": "high|medium|low", "text": "string" }
  ]
}
```

The frontend (`OpsReportTab.tsx`) renders this JSON with fixed components — each top-level key maps to a specific card widget.

---

## Trigger Configuration

### Cron Job Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `days` | 1 | Lookback window for job queries |
| `lookback_hours` | 1 | Recent running jobs window |
| `gpu_threshold` | 5 | Idle GPU utilization threshold (%) |
| `idle_hours` | 1 | Time window for idle detection |
| `running_limit` | 20 | Max running job samples |
| `node_limit` | 10 | Max node snapshots |
| `use_llm` | true | Python `/pipeline/admin-ops-report` parameter; Go patrol cron currently uses the Python default |

### Scheduling

Cron schedules are stored in the `cron_job_configs` database table and managed by `CronJobManager`. Jobs can be:
- **Scheduled**: Runs on cron expression (e.g., `0 */1 * * *` for hourly)
- **Manual**: Triggered via `POST /v1/operations/patrol/trigger-admin-ops-report`
- **Suspended**: Paused without deletion

---

## Mops Integration

The Inspection Pipeline fits into the Mops framework as a **Task-Specific Agent** (third orchestration mode):

```
Backend event (cron tick)
  → [Pipeline Agent with tool access via PipelineToolClient]
  → Structured result (report_json)
  → Backend persists + frontend renders
```

### Shared Infrastructure

| Component | Shared with chat agents? | Notes |
|-----------|------------------------|-------|
| Tool definitions | Yes | Same `tools/definitions.py` |
| Tool execution | Partial | Uses `PipelineToolClient` (not `GoBackendToolExecutor`) |
| LLM clients | Yes | Same `ModelClientFactory`, `ops_report` client config |
| Token management | No | Single-shot LLM call, no ReAct loop |
| Audit trail | Separate | `ops_audit_reports` table (not `agent_sessions`) |

### PipelineToolClient vs GoBackendToolExecutor

| Aspect | PipelineToolClient | GoBackendToolExecutor |
|--------|-------------------|----------------------|
| Identity | Fixed system identity (`agent-pipeline`) | Per-user identity from session |
| Auth | `X-Agent-Internal-Token` | `X-Agent-Internal-Token` + user context |
| Session | Static pipeline session ID | Per-conversation session |
| Role | Always `admin` | Derived from user role |
| Used by | Inspection pipeline, GPU audit | Chat agents (single, multi, task) |

---

## Notifications & Email

### Current State

The inspection pipeline now saves reports and can send policy-gated email for `admin_ops_report`:
- `GpuAuditRequest.auto_notify` is reserved and not implemented.
- Chat tool `notify_job_owner` sends job-owner email through Go `pkg/alert`, but remains a confirmed Agent tool.
- `AdminOpsReportService.TriggerAdminOpsReport()` can load the saved report and send deterministic patrol notifications when `cron_job_configs.config.notification.enabled=true`.
- Backend SMTP already exists in `pkg/alert` for job lifecycle, low-utilization, and long-running-job alerts, with per-job alert dedupe through the `Alert` table.

### Implemented Design

Automated patrol email does not rely on a user-facing chat tool as the scheduler. SMTP credentials, recipient resolution, severity gates, and cooldown stay in Go; Python provides the structured report plus optional natural-language copy.

```
CronJobManager
  → AdminOpsReportService.TriggerAdminOpsReport()
  → Python pipeline returns report_json + ops_audit_items
  → Go persists/loads the report
  → admin_ops_report_notifications.go applies deterministic policy
      ├─ Admin email for critical cluster/storage/network/capacity issues
      └─ User email for failed jobs when failure-count or failure-rate thresholds are reached
  → pkg/alert SMTP handler + notification audit/dedupe
```

Rules:
- Keep `smtp.*` in Go backend config, not `platform-runtime.yaml`.
- System patrol notifications do not need HITL, but they need severity gates, recipient rules, dedupe, cooldown, and max-emails-per-run limits.
- LLM may write the summary text, but recipients, severity thresholds, and cooldown are deterministic.
- Store notification policy with the cron job config because thresholds depend on the patrol schedule.

Example `cron_job_configs.config`:

```json
{
  "days": 1,
  "lookback_hours": 1,
  "notification": {
    "enabled": true,
    "notify_admins": true,
    "notify_job_owners": true,
    "failure_job_threshold": 10,
    "failure_rate_threshold_percent": 15,
    "unhealthy_node_threshold": 1,
    "network_alert_threshold": 3,
    "high_risk_network_job_threshold": 1,
    "max_job_owner_emails": 10,
    "cooldown_hours": 12
  }
}
```

Implementation:
- `backend/internal/service/admin_ops_report_notifications.go` reads `ops_audit_reports` and `ops_audit_items` after the Python pipeline finishes.
- Admin recipients are active platform admins with configured email, with fallback to `smtp.notify`.
- Job-owner recipients are resolved by `pkg/alert.NotifyJobOwner()` from `job_name`; owner emails missing are skipped.
- Cooldown reuses the `Alert` table. Admin notifications use a stable `ops-report:admin_ops_report:admins` key; job-owner notifications use the job name.

---

## Safety & Reliability

### Timeout Chain

```
Go backend HTTP timeout: 3 minutes
  └─ PipelineToolClient: each backend tool call defaults to 120 seconds
      └─ LLM analysis timeout: 45 seconds
          └─ Fallback: deterministic report (no LLM)
```

### Error Recovery

| Failure | Recovery |
|---------|----------|
| Backend tool call fails | Pipeline returns error status, no report saved |
| LLM times out (> 45s) | Falls back to deterministic report |
| LLM returns invalid JSON | Falls back to deterministic report |
| LLM partially succeeds | Merges only valid fields, rest from deterministic |
| Database save fails | Pipeline returns success but logs warning; report not persisted |
| Email send fails | Report remains saved; notification results record per-recipient errors for retry/manual handling |

### Read-Only Guarantee

The pipeline only calls read-only tools for data collection. The current writes are `save_audit_report` for persisting the final report and optional SMTP/audit records for notifications; no jobs are stopped and no resources are modified.

---

## Code

| Component | File |
|-----------|------|
| Pipeline entry point | `crater_agent/pipeline/ops_report.py` |
| LLM analysis | `crater_agent/pipeline/ops_report_llm.py` |
| Pipeline API router | `crater_agent/pipeline/router.py` |
| Pipeline tool client | `crater_agent/pipeline/tool_client.py` |
| Report utilities | `crater_agent/report_utils.py` |
| Backend patrol trigger | `backend/pkg/patrol/patrol.go` |
| Backend service | `backend/internal/service/admin_ops_report_service.go` |
| Backend data tool | `backend/internal/handler/agent/tools_readonly.go` (`toolGetAdminOpsReport`) |
| Backend save tool | `backend/internal/handler/agent/tools_readonly.go` (`toolSaveAuditReport`) |
| Backend API handlers | `backend/internal/handler/agent/ops_report_api.go` |
| Cron manager | `backend/pkg/cronjob/manger.go` |
| DB model | `backend/dao/model/cron_job.go` |
| DB migrations | `backend/hack/sql/20260404_ops_audit.sql`, `20260405_ops_report_enhance.sql` |
| Frontend API | `frontend/src/services/api/ops-report.ts` |
| Frontend UI | `frontend/src/components/aiops/OpsReportTab.tsx` |
| LLM config | `crater-agent/config/llm-clients.json` (`ops_report` client) |
