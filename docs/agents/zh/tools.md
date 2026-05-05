# 工具系统

> 可插拔的工具层，通过多种执行后端将 Agent 连接到基础设施。

---

## 1. 概述

工具是 Agent 推理与平台操作之间的接口。系统将**工具定义**（Agent 可以调用什么）与**工具执行**（调用如何被执行）分离，使得同一套 Agent 逻辑可以对接不同的后端。

```
Agent (LLM)
  ↓ tool_call(name, args)
Tool Selector (基于角色过滤)
  ↓
Tool Executor (路由到后端)
  ├─ GoBackendToolExecutor  → Go 后端 HTTP API
  ├─ LocalToolExecutor      → kubectl / PromQL / Harbor（可移植）
  ├─ MockToolExecutor       → 预录制响应（基准测试）
  └─ CompositeToolExecutor  → 按 platform runtime 路由；需确认写工具先进入 Go 确认流
```

---

## 2. 工具声明

工具在 `tools/definitions.py` 中使用 LangChain 的 `@tool` 装饰器声明：

```python
@tool
def get_job_detail(job_name: str) -> dict:
    """获取指定作业的详细信息，包括状态、资源配置、时间线。

    Args:
        job_name: 作业的系统唯一名（如 sg-xxx / jpt-xxx）
    """
    pass  # 函数体不会被执行 —— 执行委托给 executor
```

装饰器根据文档字符串和类型注解生成 OpenAI 兼容的函数 schema。函数体为 `pass`，因为实际执行发生在 executor 层。

### 工具注册表

| 注册表 | 数量 | 说明 |
|--------|------|------|
| `AUTO_TOOLS` | 69 | 只读，无需确认自动执行 |
| `AUTO_ACTION_TOOLS` | 0 | 预留给 system-only、无需 HITL 的后台副作用动作 |
| `CONFIRM_TOOLS` | 26 | 写操作/外部动作，需要用户确认 |
| `ALL_TOOLS` | 95 | `AUTO_TOOLS + AUTO_ACTION_TOOLS + CONFIRM_TOOLS` |
| `DEPRECATED_TOOL_NAMES` | 1 | 已废弃，不绑定到 LLM |
| `INTERNAL_TOOLS` | 1 | Pipeline 内部使用，不暴露给 LLM |

---

## 3. 工具分类

### 诊断类（只读）

作业级故障分析和信息检索。

| 工具 | 说明 |
|------|------|
| `get_job_detail` | 作业元数据、资源配置、时间线、节点信息、退出码 |
| `get_job_logs` | 容器 stdout/stderr（可按关键词过滤） |
| `diagnose_job` | 基于规则的故障分类和根因分析 |
| `get_diagnostic_context` | 完整上下文（元数据 + 事件 + 终止状态 + 指标 + 调度） |
| `search_similar_failures` | 历史模式匹配（退出码、镜像、故障类别） |

### 指标与查询类（只读）

资源利用率和平台状态。

| 工具 | 说明 |
|------|------|
| `query_job_metrics` | GPU/CPU/内存聚合指标（avg, max, stddev） |
| `get_realtime_capacity` | 集群容量快照 |
| `check_quota` | 用户配额上限与资源使用情况（capability/used/no_limit） |
| `detect_idle_jobs` | 低 GPU 利用率闲置作业检测 |
| `list_user_jobs` | 当前用户作业列表（可按状态/类型过滤） |
| `list_cluster_jobs` | 管理员视角集群作业列表（可按状态/类型过滤） |
| `analyze_queue_status` | Pending 作业排队原因分析（调度事件 + 配额 + 容量） |

### 存储诊断类（admin，只读）

| 工具 | 说明 |
|------|------|
| `list_storage_pvcs` | PVC 摘要（容量、状态、命名空间、绑定关系） |
| `get_pvc_detail` | 单个 PVC 详情（容量、访问模式、存储类、挂载引用） |
| `get_pvc_events` | PVC 相关事件（调度、挂载、绑定、扩容失败） |
| `inspect_job_storage` | 作业存储挂载与卷声明检查 |
| `get_storage_capacity_overview` | 存储容量总览（已用/可用/异常 PVC 摘要） |

### 网络诊断类（admin，只读）

| 工具 | 说明 |
|------|------|
| `get_node_network_summary` | 节点网络健康摘要 |
| `diagnose_distributed_job_network` | 分布式作业 NCCL/RDMA 网络诊断 |
| `get_rdma_interface_status` | HPC RDMA/InfiniBand 接口状态 |
| `get_node_kernel_diagnostics` | 节点内核级诊断（dmesg、D 状态进程、GPU Xid） |

### GPU 与分布式训练诊断类（admin，只读）

| 工具 | 说明 |
|------|------|
| `get_node_gpu_info` | 节点 GPU 硬件信息（驱动版本、CUDA 版本、型号、显存、ECC、温度） |
| `get_nccl_env_config` | 提取分布式训练作业所有 Pod 的 NCCL 通信环境变量配置 |
| `check_node_nic_status` | 节点网卡链路状态、协商速率、错误计数、丢包统计（覆盖交换机端口异常） |
| `detect_training_anomaly_patterns` | 扫描训练作业日志检测已知故障模式（NaN loss、CUDA OOM、NCCL 错误、梯度异常等） |
| `get_distributed_job_overview` | 分布式训练作业综合健康视图（rank 映射、就绪状态、NCCL 环境、跨节点分布） |

### K8s 直接操作类（读写，admin）

通过 kubectl 子进程执行的 Kubernetes 直接操作。

| 工具 | 说明 |
|------|------|
| `k8s_list_nodes` | 列出节点摘要（仅 admin） |
| `k8s_list_pods` | 列出 Pod 摘要；对普通用户自动收敛到本人拥有的作业范围 |
| `k8s_get_events` | 集群事件（也对普通用户开放，带所有权检查） |
| `k8s_describe_resource` | 资源详细描述（也对普通用户开放，带所有权检查） |
| `k8s_get_pod_logs` | Pod 日志获取（也对普通用户开放，带所有权检查） |
| `cordon_node` | 将节点标记为不可调度（需确认） |
| `uncordon_node` | 恢复节点调度（需确认） |
| `drain_node` | 排空节点并禁止新调度（需确认） |
| `delete_pod` | 删除 Pod 以触发重建或清理卡死实例（需确认） |
| `restart_workload` | 对 Deployment/StatefulSet/DaemonSet 执行滚动重启（需确认） |

### K8s 扩展只读类

| 工具 | 说明 |
|------|------|
| `k8s_get_service` | Kubernetes Service 资源；对普通用户自动收敛到本人拥有的作业范围 |
| `k8s_get_endpoints` | Kubernetes Endpoints 资源；对普通用户自动收敛到本人拥有的作业范围 |
| `k8s_get_ingress` | Kubernetes Ingress 资源；对普通用户自动收敛到本人拥有的作业范围 |
| `get_volcano_queue_state` | Volcano 调度队列状态 |
| `k8s_get_configmap` | Kubernetes ConfigMap 资源 |
| `k8s_get_networkpolicy` | Kubernetes NetworkPolicy 资源 |
| `k8s_top_nodes` | 节点实时 CPU/Memory 使用率 |
| `k8s_top_pods` | Pod 实时 CPU/Memory 使用率 |
| `k8s_rollout_status` | Deployment/StatefulSet/DaemonSet 滚动发布状态 |

### K8s 扩展写操作类（admin，需确认）

| 工具 | 说明 |
|------|------|
| `k8s_scale_workload` | 调整 Deployment/StatefulSet 副本数 |
| `k8s_label_node` | 添加/更新节点标签 |
| `k8s_taint_node` | 添加节点 taint |
| `run_kubectl` | 执行受安全策略约束的高风险 `kubectl` 写命令 |
| `execute_admin_command` | 执行非 kubectl 管理命令（`helm` / `velero` / `istioctl` / `psql`） |

### 基础设施类（只读，admin）

平台服务健康状态和诊断。

| 工具 | 说明 |
|------|------|
| `prometheus_query` | 直接 PromQL 查询（instant/range） |
| `harbor_check` | Harbor/OCI 镜像仓库健康状态和镜像验证 |

### 聚合增强类（admin，只读）

| 工具 | 说明 |
|------|------|
| `aggregate_image_pull_errors` | 集群级镜像拉取失败聚合 |
| `detect_zombie_jobs` | 检测疑似僵尸 Running 作业 |
| `get_ddp_rank_mapping` | 分布式训练 DDP/Volcano rank→Pod 映射 |

### 运维报告类

| 工具 | 说明 |
|------|------|
| `get_health_overview` | 用户作业健康概览（总数、失败数、运行中、失败率） |
| `get_failure_statistics` | 故障类别分布统计 |
| `get_cluster_health_report` | 集群健康报告（作业、节点、GPU、故障） |
| `get_admin_ops_report` | 管理员智能运维分析报告 |
| `get_node_detail` | 单个集群节点详情 |
| `get_resource_recommendation` | 基于任务描述的资源配置推荐 |

### 镜像与资源查询类

| 工具 | 说明 |
|------|------|
| `list_available_images` | 可用镜像列表（可按类型/关键词过滤） |
| `list_cuda_base_images` | CUDA 基础镜像 |
| `list_available_gpu_models` | GPU 型号及总量/已用/剩余摘要 |
| `recommend_training_images` | 基于描述的训练镜像推荐 |
| `list_image_builds` | 当前用户镜像构建任务列表（状态、来源、镜像链接、最终镜像关联） |
| `get_image_build_detail` | 单个镜像构建任务详情（脚本、Pod 信息、最终镜像关联） |
| `get_image_access_detail` | 镜像授权详情，仅镜像拥有者可查看 |
| `get_job_templates` | 可用作业模板 |
| `list_cluster_nodes` | 集群节点摘要（状态、工作负载、供应商、数量） |
| `analyze_queue_status` | Pending 作业排队原因分析 |

### 镜像创建模型

镜像创建在 Agent 侧有意拆成两层生命周期：

| 层级 | 为什么单独建模 |
|------|------|
| `image build` | 异步构建任务，关心状态、脚本、Pod、取消、删除 |
| `image` | 最终可复用制品，关心可见性、分享、导入、作业复用 |

因此即使 GUI 上“创建镜像”入口看起来很简单，Agent 侧仍需要保留 `list_image_builds` 和 `get_image_build_detail` 这类只读工具，供单智能体和多智能体在后续跟踪、诊断、取消时复用。

写工具面故意保持紧凑，而不是继续膨胀：
- 用 `create_image_build` 统一承接 `pip_apt`、`dockerfile`、`envd`、`envd_raw` 四种模式。
- 用 `manage_image_build(action=cancel|delete)` 代替单独的 `cancel_image_build`、`delete_image_build`。
- 用 `manage_image_access(action=grant|revoke)` 代替单独的 `share_image`、`revoke_image_share`。
- 用只读的 `get_image_access_detail` 承接授权查看，这样 planner / explorer 角色也能安全使用。

### 审计类（admin）

| 工具 | 说明 |
|------|------|
| `get_latest_audit_report` | 最近审计报告摘要 |
| `list_audit_items` | 审计条目列表（可过滤） |
| `mark_audit_handled` | 标记审计条目为已处理（需确认） |

### 审批类（只读）

审批工作流支持。

| 工具 | 说明 |
|------|------|
| `get_approval_history` | 用户近期审批工单 |

### 写操作（需要确认）

会修改平台资源的写工具返回 `confirmation_required` 状态，暂停 Agent 循环等待用户批准。

| 工具 | 说明 |
|------|------|
| `stop_job` | 停止运行中的作业 |
| `delete_job` | 删除作业记录 |
| `resubmit_job` | 重新提交（可选覆盖参数） |
| `create_jupyter_job` | 创建 Jupyter 交互式作业 |
| `create_webide_job` | 创建 WebIDE 交互式作业 |
| `create_training_job` | 创建自定义训练作业 |
| `create_custom_job` | 创建自定义作业（`create_training_job` 的语义化别名） |
| `create_pytorch_job` | 创建 PyTorch 分布式作业 |
| `create_tensorflow_job` | 创建 TensorFlow 分布式作业 |
| `create_image_build` | 提交新的镜像构建草案（支持 `pip_apt`、`dockerfile`、`envd`、`envd_raw`） |
| `manage_image_build` | 取消或删除镜像构建任务 |
| `register_external_image` | 将 Harbor / OCI 中已有镜像登记到 Crater |
| `manage_image_access` | 向用户或账户授予/撤销镜像权限 |
| `batch_stop_jobs` | 批量停止多个作业 |
| `mark_audit_handled` | 标记审计条目为已处理 |
| `notify_job_owner` | 需管理员确认的邮件通知工具，通过 Go 后端 `pkg/alert` 向作业所有者发送邮件，并返回逐作业发送/跳过/失败结果 |
| `cordon_node` / `uncordon_node` / `drain_node` | 节点管理 |
| `delete_pod` / `restart_workload` | Pod 管理 |
| `k8s_scale_workload` | 调整副本数 |
| `k8s_label_node` / `k8s_taint_node` | 节点标签/taint 管理 |
| `run_kubectl` | 执行高风险 `kubectl` 写命令 |
| `execute_admin_command` | 执行非 kubectl 管理命令（`helm` / `velero` / `istioctl` / `psql`） |

### 其他

| 工具 | 说明 |
|------|------|
| `get_agent_runtime_summary` | Agent 运行时配置摘要（平台无关，本地执行） |

---

## 4. 工具执行器

### GoBackendToolExecutor

通过 HTTP POST 将工具调用发送到 Go 后端：

```
POST /api/agent/tools/execute
Headers: X-Agent-Internal-Token: <shared_secret>
Body: { tool_name, tool_args, session_id, turn_id, agent_id, agent_role }
```

响应状态：
- `success` — 工具执行成功，返回结果
- `error` — 执行失败（包含 `error_type` 和 `retryable` 标志）
- `confirmation_required` — 写工具需要用户批准

错误类型：`tool_policy`、`auth`、`not_found`、`rate_limit`、`server`、`network`、`timeout`

### LocalToolExecutor

在 Agent 进程中直接执行工具（不依赖 Go 后端）：

- **kubectl**：通过子进程对 kubeconfig 执行
- **Prometheus**：向 Prometheus API 发送 HTTP 查询
- **Harbor**：OCI 镜像仓库 API 调用

关键特性：
- 非 admin 用户的所有权检查（通过 Go 后端回调验证 Pod 归属）
- Prometheus 响应裁剪（最多 20 个序列，每序列最多 120 个数据点）

### CompositeToolExecutor

将每个工具调用路由到合适的执行器：

```python
default_target = "local" if local.supports(tool_name) and runtime.default_route_for_tool(tool_name) == "local" else "backend"
target = route_for_tool(runtime, tool_name, default=default_target)

if tool_name in CONFIRM_TOOL_NAMES:
    # 无论最终由 Go 还是 Python 本地执行，先让 Go 创建 pending confirmation
    execution_backend = "python_local" if target == "local" else "backend"
    return go_backend_executor.execute(..., execution_backend=execution_backend)

if target == "local":
    return local_executor.execute(..., execution_backend="python_local")

return go_backend_executor.execute(...)
```

要点：
- 只读工具可直接本地执行或走 Go 后端，取决于 `platform-runtime.yaml` / backend debug config 的 `toolRouting`。
- 需确认工具永远不会由 LLM 直接调用本地执行器；Go 先落库 `AgentToolCall` 并返回 `confirmation_required`。
- `notify_job_owner` 这类由聊天 Agent 触发的外部动作也走确认流；系统巡检自动邮件走后台通知策略，不进入聊天工具目录。
- 用户确认后，如果 `execution_backend=python_local`，Go 调用 Python `/internal/tools/execute-local-write`；否则调用 Go 的 `executeWriteTool()`。
- Go 会缓存 Python `/internal/tools/catalog` 30 秒，用于把本地工具动态并入 `capabilities.tool_catalog`。

### MockToolExecutor

为基准测试返回预录制响应：
- 从 `crater_bench/mock_responses/` 加载
- 支持基于参数的快照查找
- 在 `call_log` 中记录所有调用供评估使用
- 需确认的工具始终返回 `confirmation_required`

### ReadOnlyToolExecutor

包装另一个执行器以强制只读模式：
- 阻止所有写工具，返回 `confirmation_required`
- 用于 `live-readonly` 评估模式

---

## 5. 工具选择

### 基于角色的过滤

```
全部已绑定工具（当前 `ALL_TOOLS` 95 个）
  ↓
Go buildAgentCapabilities() 生成 enabled_tools / confirm_tools / tool_catalog
  ↓
Actor 角色 = admin? → 返回全部
Actor 角色 = user? → 返回所有非 admin-only 工具
```

**用户可见工具**包含：
- 所有诊断和查询工具（用户范围）
- 作业管理和创建工具（`jupyter`、`webide`、`custom`、`pytorch`、`tensorflow`）
- 镜像创建工具（`list_image_builds`、`get_image_build_detail`、`create_image_build` 等）
- 资源查询工具（镜像、GPU 型号、容量、配额）
- 有范围限制的 K8s 工具（`k8s_list_pods`、`k8s_get_service`、`k8s_get_endpoints`、`k8s_get_ingress`、事件、describe、Pod 日志）

**仅 admin 可用的工具**包括：
- 集群级查询（`list_cluster_jobs`、`list_cluster_nodes`）
- 节点管理（`cordon_node`、`drain_node`、`k8s_label_node`、`k8s_taint_node`）
- 存储诊断（`list_storage_pvcs`、`get_pvc_detail`、`get_pvc_events`、`inspect_job_storage`、`get_storage_capacity_overview`）
- 网络诊断（`get_node_network_summary`、`diagnose_distributed_job_network`、`get_rdma_interface_status`、`get_node_kernel_diagnostics`）
- K8s 扩展（`get_volcano_queue_state`、`k8s_get_configmap`、`k8s_get_networkpolicy`、`k8s_top_nodes`、`k8s_top_pods`、`k8s_rollout_status`）
- K8s 写操作（`k8s_scale_workload`、`execute_admin_command`、`delete_pod`、`restart_workload`）
- 聚合增强（`aggregate_image_pull_errors`、`detect_zombie_jobs`、`get_ddp_rank_mapping`）
- 运维报告（`get_cluster_health_report`、`get_admin_ops_report`、`get_node_detail`、`get_failure_statistics`）
- 审计（`get_latest_audit_report`、`list_audit_items`、`mark_audit_handled`）
- 基础设施（`prometheus_query`、`harbor_check`、`k8s_list_nodes`、`get_volcano_queue_state`）
- 运维操作（`batch_stop_jobs`、`notify_job_owner`、`run_kubectl`、`execute_admin_command`）

### Agent 角色策略

在用户/admin 过滤之外，每种 Agent 角色有自己的工具策略：

| Agent 角色 | 允许的工具 |
|------------|-----------|
| `explorer` | 仅只读（AUTO_TOOLS） |
| `executor` / `single_agent` | 全部（读 + 写） |
| `planner` / `coordinator` / `verifier` | 只读（AUTO_TOOLS） |
| `general` | 只读（AUTO_TOOLS） |
| `guide` | 无 |

---

## 6. 工具结果处理

每个工具结果在加入对话前都经过 token 预算感知的处理流水线：

```
原始工具结果（可能非常大）
  ↓
在单工具 token 预算内?
  ├─ 是 → 直接使用
  └─ 否
      ↓
  LLM 语义提取（10 秒超时）
    "仅提取用户相关信息：错误、状态、关键指标"
  ├─ 成功 → 使用提取后的文本
  └─ 失败
      ↓
  硬截断（首尾保留）
```

### 单工具 Token 预算

| 工具 | 预算 | 原因 |
|------|------|------|
| `get_job_logs` | 4000 | 日志可能非常长 |
| `diagnose_job` | 4000 | 结构化诊断输出 |
| `get_diagnostic_context` | 4000 | 完整上下文包 |
| `get_job_detail` | 3000 | 作业元数据 |
| `prometheus_query` | 2000 | 时序数据 |
| 默认 | 3000 | — |

---

## 7. 添加新工具

1. 在 `tools/definitions.py` 中**定义**工具：
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

2. **注册**到 `AUTO_TOOLS`（只读）、`AUTO_ACTION_TOOLS`（预留 system-only 无 HITL 副作用动作）或 `CONFIRM_TOOLS`（需确认的写操作/外部动作）

3. **在 Go 中实现后端处理器**：
   - 在 `handler/agent/agent.go` 中添加常量
   - 在 `tools_dispatch.go` 中添加到 `isAgentReadOnlyTool()`、`isAgentAutoActionTool()` 或 `isAgentConfirmTool()`
   - 在 `tools_dispatch.go` 中的 `executeReadTool()`、`executeAutoActionTool()` 或 `executeWriteTool()` 添加 case
   - 在匹配的后端文件中编写处理函数（`tools_readonly.go`、`tools_write.go` 或专门的工具模块）

4. **或在本地实现**，位于 `tools/local_executor.py`：
   - 添加到 `LocalToolExecutor._handlers`
   - 必要时加入 `runtime/platform.py` 的 `DEFAULT_LOCAL_CORE_TOOLS` 或 `DEFAULT_LOCAL_WRITE_TOOLS`
   - 在 `platform-runtime.yaml` 的 `toolRouting.localCoreTools` / `localWriteTools` 中启用
   - 本地写工具仍必须加入 `CONFIRM_TOOLS`，并通过 Go 确认后由 `/internal/tools/execute-local-write` 分发

### 现有缺口

- Python definitions 中声明了 69 个只读工具，但 Go `executeReadTool()` 只实现了后端可直接处理的一部分；其余依赖 `LocalToolExecutor` 和 `platform-runtime.yaml` 路由。
- `k8s_scale_workload`、`k8s_label_node`、`k8s_taint_node`、`run_kubectl`、`execute_admin_command` 属于本地写工具，只有当 Python 本地目录暴露且 Go 记录 `execution_backend=python_local` 后，确认恢复才会转发给 Python 执行。
- 自动巡检邮件仍建议走后台通知服务和确定性策略，不依赖用户聊天工具作为调度入口。

---

## 8. 遗留 / 暂缓工具

以下工具要么继续作为显式 Agent 工具保留，要么暂缓开放，等待行为和沙箱能力进一步校验。

| 工具 | 替代方案 | 状态 |
|------|---------|------|
| `web_search` | 暂无 | 已启用并绑定到 LLM，不再 admin-only |
| `fetch_url` | 暂无 | 已启用，绑定到 LLM |
| `execute_code` | 受限沙箱解释器 | 暂缓开放，不绑定 |

### 内部工具

| 工具 | 说明 |
|------|------|
| `save_audit_report` | 保存审计报告到数据库（Pipeline 内部调用，不暴露给 LLM） |

---

## 代码

| 组件 | 文件 |
|------|------|
| 工具定义 | `crater_agent/tools/definitions.py` |
| Go 后端执行器 | `crater_agent/tools/executor.py` (`GoBackendToolExecutor`) |
| 本地执行器 | `crater_agent/tools/local_executor.py` |
| 组合执行器 | `crater_agent/tools/executor.py` (`CompositeToolExecutor`) |
| 工具选择器 | `crater_agent/tools/tool_selector.py` |
| Go 工具分发 | `backend/internal/handler/agent/tools_dispatch.go` |
| Go 工具处理器 | `backend/internal/handler/agent/tools_readonly.go` |
| Go 工具常量 | `backend/internal/handler/agent/agent.go` |
