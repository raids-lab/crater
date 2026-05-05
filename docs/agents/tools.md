# Tool System

> Pluggable tool layer that connects agents to infrastructure through multiple execution backends.

---

## 1. Overview

Tools are the interface between agent reasoning and platform operations. The system separates **tool definition** (what the agent can call) from **tool execution** (how the call is fulfilled), enabling the same agent logic to run against different backends.

```
Agent (LLM)
  | tool_call(name, args)
Tool Selector (role-based filtering)
  |
Tool Executor (routes to backend)
  |-- GoBackendToolExecutor  -> Go backend HTTP API
  |-- LocalToolExecutor      -> kubectl / PromQL / Harbor (portable)
  |-- MockToolExecutor       -> pre-recorded responses (benchmark)
  +-- CompositeToolExecutor  -> platform-runtime routing; confirm writes enter Go confirmation first
```

---

## 2. Tool Declaration

Tools are declared in `tools/definitions.py` using LangChain's `@tool` decorator:

```python
@tool
def get_job_detail(job_name: str) -> dict:
    """获取指定作业的详细信息，包括状态、资源配置、时间线。

    Args:
        job_name: 作业的系统唯一名（如 sg-xxx / jpt-xxx）
    """
    pass  # Body is never executed — execution is delegated to an executor
```

The decorator generates an OpenAI-compatible function schema from the docstring and type annotations. The function body is `pass` because actual execution happens in the executor layer.

### Tool Registries

| Registry | Count | Description |
|----------|-------|-------------|
| `AUTO_TOOLS` | 69 | Read-only, auto-executed without confirmation |
| `AUTO_ACTION_TOOLS` | 0 | Reserved for system-only side effects without HITL |
| `CONFIRM_TOOLS` | 26 | Write/external action tools requiring user confirmation |
| `ALL_TOOLS` | 95 | `AUTO_TOOLS + AUTO_ACTION_TOOLS + CONFIRM_TOOLS` |
| `INTERNAL_TOOLS` | 1 | Pipeline-internal tools, not exposed to LLM |
| `DEPRECATED_TOOL_NAMES` | 1 | Declared but not bound to LLM |

---

## 3. Tool Categories

### Diagnosis (read-only)
Job-level fault analysis and information retrieval.

| Tool | Description |
|------|-------------|
| `get_job_detail` | Job metadata, resources, timeline, node info, exit code |
| `get_job_logs` | Container stdout/stderr (tail N lines, optional keyword filter) |
| `diagnose_job` | Rules-based fault classification and root cause analysis |
| `get_diagnostic_context` | Full context bundle (meta + events + termination + metrics + scheduling) |
| `search_similar_failures` | Historical failure pattern matching (exit code, image, fault category) |

### Metrics & Queue Analysis (read-only)
Resource utilization, capacity, and queue diagnostics.

| Tool | Description |
|------|-------------|
| `query_job_metrics` | GPU/CPU/Memory aggregates (avg, max, stddev) |
| `analyze_queue_status` | Pending job queue analysis (scheduling events, quota, capacity) |
| `get_realtime_capacity` | Cluster capacity snapshot (node-level resource usage and status) |
| `check_quota` | User quota limits and resource usage (capability/used/no_limit) |
| `detect_idle_jobs` | Low GPU utilization detection (threshold-based) |
| `list_user_jobs` | User's jobs listing with status/type filters |
| `list_cluster_jobs` | Cluster-wide jobs listing (admin) |

### Image & Resource Query (read-only)
Image discovery, GPU inventory, and resource recommendations.

| Tool | Description |
|------|-------------|
| `list_available_images` | Available images list (filterable by type/keyword) |
| `list_cuda_base_images` | CUDA base images |
| `list_available_gpu_models` | GPU models with total/used/remaining summary |
| `recommend_training_images` | Training image recommendation based on task description |
| `list_image_builds` | User's image build task list (status, source, image link, final-image association) |
| `get_image_build_detail` | Single image build detail (script, pod info, final image association) |
| `get_image_access_detail` | Owner-only image access detail (granted users/accounts) |
| `get_job_templates` | Available job templates |
| `list_cluster_nodes` | Cluster node summary (status, workloads, vendor, count) |
| `get_resource_recommendation` | Resource configuration recommendation based on task description |

### Image Authoring Model
Image authoring is intentionally modeled as two related but different lifecycles:

| Layer | Why it exists |
|------|-------------|
| `image build` | Asynchronous build task with status, script, pod, cancellation, and deletion semantics |
| `image` | Final reusable artifact with visibility, sharing, import, and job-reuse semantics |

This is why the agent keeps `list_image_builds` and `get_image_build_detail` as separate read tools even though the GUI can hide some of this complexity.

The write surface is intentionally compact:
- Use `create_image_build` instead of separate mode-specific tools.
- Use `manage_image_build(action=cancel|delete)` instead of exploding into `cancel_image_build` and `delete_image_build`.
- Use `manage_image_access(action=grant|revoke)` instead of separate `share_image` / `revoke_image_share` tools.
- Keep access inspection read-only via `get_image_access_detail`, so planner/explorer roles can inspect without being allowed to mutate.

### Operations Reports (read-only)
Health overviews, failure statistics, and ops analysis.

| Tool | Description |
|------|-------------|
| `get_health_overview` | User job health overview (total, failed, running, failure rate) |
| `get_failure_statistics` | Failure category distribution statistics |
| `get_cluster_health_report` | Cluster health report (jobs, nodes, GPU, failures) (admin) |
| `get_admin_ops_report` | Admin intelligent ops analysis report (admin) |
| `get_node_detail` | Single cluster node details (admin) |

### Audit (admin, read-only)
GPU audit and compliance.

| Tool | Description |
|------|-------------|
| `get_latest_audit_report` | Latest audit report summary |
| `list_audit_items` | Audit item listing with filters |

### Storage Diagnostics (admin, read-only)
PVC and volume diagnostics.

| Tool | Description |
|------|-------------|
| `list_storage_pvcs` | PVC summary (capacity, status, namespace, bindings) |
| `get_pvc_detail` | Single PVC details (capacity, access modes, storage class, mount refs) |
| `get_pvc_events` | PVC-related events (scheduling, mounting, binding, expansion failures) |
| `inspect_job_storage` | Job storage mount and volume claim inspection |
| `get_storage_capacity_overview` | Storage capacity overview (used/available/abnormal PVC summary) |

### Network Diagnostics (admin, read-only)
Node network, RDMA/InfiniBand, and kernel-level diagnostics.

| Tool | Description |
|------|-------------|
| `get_node_network_summary` | Node network health summary |
| `diagnose_distributed_job_network` | NCCL/RDMA distributed job network diagnostics |
| `get_rdma_interface_status` | HPC RDMA/InfiniBand interface status |
| `get_node_kernel_diagnostics` | Node kernel-level diagnostics (dmesg, D-state, GPU Xid) |

### GPU & Distributed Training Diagnostics (admin, read-only)
GPU hardware inspection, distributed training environment checks, and training anomaly detection.

| Tool | Description |
|------|-------------|
| `get_node_gpu_info` | Node GPU hardware info via nvidia-smi (driver version, CUDA version, model, memory, ECC, temperature) |
| `get_nccl_env_config` | Extract NCCL/distributed training env vars from all Pods of a Volcano job |
| `check_node_nic_status` | Node NIC link status, speed, error counters, packet drops (covers switch port issues from node side) |
| `detect_training_anomaly_patterns` | Scan training job logs for known failure patterns (NaN loss, CUDA OOM, NCCL errors, gradient overflow, etc.) |
| `get_distributed_job_overview` | Holistic distributed training job health: rank mapping, readiness, NCCL env, cross-node distribution |

### Enrichment & Analysis (admin, read-only)
Cross-job aggregation and distributed training diagnostics.

| Tool | Description |
|------|-------------|
| `aggregate_image_pull_errors` | Cluster-wide image pull error aggregation |
| `detect_zombie_jobs` | Detect potentially zombie Running jobs |
| `get_ddp_rank_mapping` | DDP/Volcano rank-to-pod mapping for distributed training |

### K8s Core (read-only)
Direct Kubernetes queries via kubectl subprocess.

| Tool | Description |
|------|-------------|
| `k8s_list_nodes` | List Kubernetes nodes (admin only) |
| `k8s_list_pods` | List Kubernetes pods; scoped to the current user's owned workloads for non-admins |
| `k8s_get_events` | Kubernetes events (image pull, scheduling, PVC mount failures) |
| `k8s_describe_resource` | Detailed resource description (node, pod, PVC, deployment, etc.) |
| `k8s_get_pod_logs` | Pod log retrieval (container, tail, since, previous) |

### K8s Extended Read-only
Additional Kubernetes resource queries and metrics.

| Tool | Description |
|------|-------------|
| `k8s_get_service` | Kubernetes Service resources; scoped to the current user's owned workloads for non-admins |
| `k8s_get_endpoints` | Kubernetes Endpoints resources; scoped to the current user's owned workloads for non-admins |
| `k8s_get_ingress` | Kubernetes Ingress resources; scoped to the current user's owned workloads for non-admins |
| `get_volcano_queue_state` | Volcano scheduling queue state |
| `k8s_get_configmap` | Kubernetes ConfigMap resources |
| `k8s_get_networkpolicy` | Kubernetes NetworkPolicy resources |
| `k8s_top_nodes` | Node real-time CPU/Memory usage |
| `k8s_top_pods` | Pod real-time CPU/Memory usage |
| `k8s_rollout_status` | Deployment/StatefulSet/DaemonSet rollout status |

### Infrastructure (admin, read-only)
Platform service health and direct metric queries.

| Tool | Description |
|------|-------------|
| `prometheus_query` | Direct PromQL queries (instant/range, series/points trimming) |
| `harbor_check` | Harbor/OCI registry health and image verification |

### Approval (read-only)
Approval workflow support.

| Tool | Description |
|------|-------------|
| `get_approval_history` | User's recent approval orders |

### Misc (read-only)
Agent-side utilities.

| Tool | Description |
|------|-------------|
| `get_agent_runtime_summary` | Agent runtime config summary (platform-agnostic, local) |

### Write Operations (confirmation required)
Platform mutation tools return `confirmation_required` status, pausing the agent loop for user approval.

**User write tools:**

| Tool | Description |
|------|-------------|
| `stop_job` | Stop a running job |
| `delete_job` | Delete a job record |
| `resubmit_job` | Resubmit with optional resource overrides |
| `create_jupyter_job` | Create a Jupyter interactive job |
| `create_webide_job` | Create a WebIDE interactive job |
| `create_training_job` | Create a custom/training job |
| `create_custom_job` | Semantic alias for creating a custom job |
| `create_pytorch_job` | Create a PyTorch distributed job |
| `create_tensorflow_job` | Create a TensorFlow distributed job |
| `create_image_build` | Submit a new image build draft (`pip_apt`, `dockerfile`, `envd`, `envd_raw`) |
| `manage_image_build` | Cancel or delete an image build task |
| `register_external_image` | Register an existing Harbor / OCI image into Crater |
| `manage_image_access` | Grant or revoke image access for users/accounts |

**Admin write tools (K8s node/pod management):**

| Tool | Description |
|------|-------------|
| `cordon_node` | Mark node as unschedulable |
| `uncordon_node` | Restore node scheduling |
| `drain_node` | Drain node and disable scheduling |
| `delete_pod` | Delete pod to trigger rebuild or clear stuck instance |
| `restart_workload` | Rolling restart of Deployment/StatefulSet/DaemonSet |

**K8s Extended Write (admin, confirm):**

| Tool | Description |
|------|-------------|
| `k8s_scale_workload` | Scale Deployment/StatefulSet replicas |
| `k8s_label_node` | Add/update node labels |
| `k8s_taint_node` | Add node taints |
| `run_kubectl` | Execute confirmed high-risk kubectl write commands |
| `execute_admin_command` | Execute non-kubectl admin commands (helm/velero/istioctl/psql) |

**Admin ops write tools:**

| Tool | Description |
|------|-------------|
| `batch_stop_jobs` | Bulk stop jobs |
| `mark_audit_handled` | Mark audit items as handled |
| `notify_job_owner` | Confirmed admin email notification to job owners via Go backend `pkg/alert`; returns per-job send/skipped/error results |

---

## 4. Tool Executors

### GoBackendToolExecutor

Sends tool calls to the Go backend via HTTP POST:

```
POST /api/agent/tools/execute
Headers: X-Agent-Internal-Token: <shared_secret>
Body: { tool_name, tool_args, session_id, turn_id, agent_id, agent_role }
```

Response status:
- `success` -- tool executed, result returned
- `error` -- execution failed (with `error_type` and `retryable` flag)
- `confirmation_required` -- write tool needs user approval

Error types: `tool_policy`, `auth`, `not_found`, `rate_limit`, `server`, `network`, `timeout`

### LocalToolExecutor

Executes tools directly in the agent process (no Go backend dependency):

- **kubectl**: subprocess execution against kubeconfig
- **Prometheus**: HTTP queries to Prometheus API
- **Harbor**: OCI registry API calls

Key features:
- Ownership checks for non-admin users (verifies pod ownership via Go backend callback)
- Prometheus response trimming (max 20 series, 120 points per series)

### CompositeToolExecutor

Routes each tool call to the appropriate executor:

```python
default_target = "local" if local.supports(tool_name) and runtime.default_route_for_tool(tool_name) == "local" else "backend"
target = route_for_tool(runtime, tool_name, default=default_target)

if tool_name in CONFIRM_TOOL_NAMES:
    execution_backend = "python_local" if target == "local" else "backend"
    return go_backend_executor.execute(..., execution_backend=execution_backend)

if target == "local":
    return local_executor.execute(..., execution_backend="python_local")

return go_backend_executor.execute(...)
```

Notes:
- Read-only tools may run locally or through Go depending on `platform-runtime.yaml` and backend debug-config routing.
- Confirm tools never execute directly from the LLM. Go first creates the pending confirmation record.
- Chat-triggered external actions such as `notify_job_owner` stay in the confirmation path even though they only send email.
- After approval, Go either executes a backend write handler or dispatches to Python `/internal/tools/execute-local-write` when `execution_backend=python_local`.
- Go caches Python `/internal/tools/catalog` for 30 seconds and merges it into `capabilities.tool_catalog`.

### MockToolExecutor

Returns pre-recorded responses for benchmarking:
- Loaded from `crater_bench/mock_responses/`
- Supports arg-based snapshot lookup
- Records all calls in `call_log` for evaluation
- Confirm tools always return `confirmation_required`

### ReadOnlyToolExecutor

Wraps another executor to enforce read-only mode:
- Blocks all write tools with `confirmation_required`
- Used in `live-readonly` evaluation mode

---

## 5. Tool Selection

### Role-Based Filtering

```
All bound tools (currently 95)
  |
Go buildAgentCapabilities() creates enabled_tools / confirm_tools / tool_catalog
  |
Actor role = admin? -> return all
Actor role = user? -> return every tool that is not admin-only
```

**User-visible tools** include:
- All diagnosis and query tools (user-scoped)
- Job management and authoring tools (`jupyter`, `webide`, `custom`, `pytorch`, `tensorflow`)
- Image authoring tools (`list_image_builds`, `get_image_build_detail`, `create_image_build`, etc.)
- Resource query tools (images, GPU models, capacity, quota)
- Scoped K8s tools (`k8s_list_pods`, `k8s_get_service`, `k8s_get_endpoints`, `k8s_get_ingress`, events, describe, pod logs)

**Admin-only tools** include:
- Cluster-level queries, node management, storage/network diagnostics
- Direct K8s write operations, confirmed admin commands, audit reports
- Prometheus, Harbor
- K8s extended read/write, enrichment and analysis tools

### Agent-Role Policy

Beyond user/admin filtering, each agent role has a tool policy:

| Agent Role | Allowed Tools |
|------------|--------------|
| `explorer` | Read-only only |
| `executor` / `single_agent` | All (read + write) |
| `planner` / `coordinator` / `verifier` / `general` | Read-only |
| `guide` | None |

---

## 6. Tool Result Processing

Each tool result goes through a token-budget-aware pipeline before being added to the conversation:

```
Raw tool result (may be very large)
  |
Within per-tool token budget?
  |-- YES -> use as-is
  +-- NO
      |
  LLM semantic extraction (10s timeout)
    "Extract only user-relevant info: errors, status, key metrics"
  |-- SUCCESS -> use extracted text
  +-- FAIL
      |
  Hard truncation (head + tail)
```

### Per-Tool Token Budgets

| Tool | Budget | Reason |
|------|--------|--------|
| `get_job_logs` | 4000 | Logs can be very long |
| `diagnose_job` | 4000 | Structured diagnosis output |
| `get_diagnostic_context` | 4000 | Full context bundle |
| `get_job_detail` | 3000 | Job metadata |
| `prometheus_query` | 2000 | Time series data |
| Default | 3000 | -- |

---

## 7. Adding a New Tool

1. **Define** the tool in `tools/definitions.py`:
   ```python
   @tool
   def my_new_tool(param: str, limit: int = 10) -> dict:
       """Tool description for LLM.
       
       Args:
           param: Parameter description
           limit: Limit description
       """
       pass
   ```

2. **Register** in `AUTO_TOOLS` (read-only), `AUTO_ACTION_TOOLS` (reserved system-only side effects without HITL), or `CONFIRM_TOOLS` (confirmed writes/external actions)

3. **Implement backend handler** in Go:
   - Add constant in `handler/agent/agent.go`
   - Add to `isAgentReadOnlyTool()`, `isAgentAutoActionTool()`, or `isAgentConfirmTool()` in `tools_dispatch.go`
   - Add case in `executeReadTool()`, `executeAutoActionTool()`, or `executeWriteTool()` in `tools_dispatch.go`
   - Write handler function in the matching backend file (`tools_readonly.go`, `tools_write.go`, or a focused tool module)

4. **Or implement locally** in `tools/local_executor.py`:
   - Add to `LocalToolExecutor._handlers`
   - Add to `runtime/platform.py` defaults when it should be locally routable
   - Enable it through `platform-runtime.yaml` `toolRouting.localCoreTools` / `localWriteTools`
   - Local write tools must still be in `CONFIRM_TOOLS` and execute only after Go confirmation dispatch

### Current Gaps

- Python declares 69 read-only tools, but Go `executeReadTool()` implements only the backend-handled subset; the rest depend on `LocalToolExecutor` and runtime routing.
- `k8s_scale_workload`, `k8s_label_node`, `k8s_taint_node`, `run_kubectl`, and `execute_admin_command` are local write tools. They execute only when Python exposes them in the local catalog and Go records `execution_backend=python_local`.
- Automated patrol email should still use a backend system notification service with deterministic policy, not rely on a user-facing chat tool as the scheduler.

---

## 8. Legacy / Deferred Tools

The following tools are either still supported as explicit agent tools, or kept deferred until their behavior and sandboxing are re-validated.

| Tool | Replacement | Status |
|------|-------------|--------|
| `web_search` | None | Active, bound to LLM and no longer admin-only |
| `fetch_url` | None | Active, bound to LLM |
| `execute_code` | Sandboxed interpreter | Deferred, not bound |

### Internal Tools

| Tool | Description |
|------|-------------|
| `save_audit_report` | Save audit report to database (pipeline internal, not exposed to LLM) |

---

## Code

| Component | File |
|-----------|------|
| Tool definitions | `crater_agent/tools/definitions.py` |
| Go backend executor | `crater_agent/tools/executor.py` (`GoBackendToolExecutor`) |
| Local executor | `crater_agent/tools/local_executor.py` |
| Composite executor | `crater_agent/tools/executor.py` (`CompositeToolExecutor`) |
| Tool selector | `crater_agent/tools/tool_selector.py` |
| Go tool dispatch | `backend/internal/handler/agent/tools_dispatch.go` |
| Go tool handlers | `backend/internal/handler/agent/tools_readonly.go` |
| Go tool constants | `backend/internal/handler/agent/agent.go` |
