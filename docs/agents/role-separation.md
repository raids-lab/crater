# 角色区分机制（Admin / Portal）

> 记录当前 admin 与 portal（普通用户）的区分实现：从后端传参到 agent 的 prompt/工具/权限各层。
> 最后更新: 2026-04-28

---

## 1. 角色判定链路

### 1.1 Go 后端（入口）

Go 后端在构建 agent 请求时传递用户角色和页面信息:

```
用户请求 → JWT Token 中的 RolePlatform（admin / user）
                 ↓
页面路由（前端传入 pageRoute / pageURL）
                 ↓
capabilities.go 判定 pageScope:
  if token.RolePlatform == admin AND route.startsWith("/admin"):
      pageScope = "admin"
  else:
      pageScope = "user"
                 ↓
构建 capabilities 对象:
  - enabled_tools: 根据 JWT 角色 + Python 本地工具目录生成
  - confirm_tools: 写操作工具列表
  - tool_catalog: 静态工具 + Python 本地工具目录
  - surface: { page_scope, page_route, page_url }
```

### 1.2 Agent 侧（tool selection / tools_node）

Agent 侧会先使用 Go 后端传入的 `capabilities.enabled_tools` 缩小可绑定工具集，再按页面/角色做二次过滤：

- `create_agent_graph().get_enabled_tools()`：如果有 `enabled_tools`，只从 `ALL_TOOLS` 中绑定这些工具；否则才回退到全部工具。
- `select_tools_for_context()`：admin 页面返回当前候选集；普通页面过滤掉 `ADMIN_ONLY_TOOL_NAMES`。
- `tools_node` 执行工具时会推断 `actor_role` 并传给 Go/Python 工具执行器，用于二次权限判断与审计。

`tools_node` 的执行角色推断逻辑如下：

```python
page = context.get("page", {})
route = str(page.get("route") or "").strip().lower()
url = str(page.get("url") or "").strip().lower()

if route.startswith("/admin") or "/admin/" in route or url.startswith("/admin") or "/admin/" in url:
    actor_role = "admin"
elif route or url:
    actor_role = "user"
else:
    actor_role = str(actor.get("role") or "user").strip().lower() or "user"
```

实际安全边界仍在 Go 后端：非管理员 JWT 不会获得 admin `enabled_tools`，并且执行时还会被 `isAgentAdminOnlyTool()` 拦截。

### 1.3 Prompt 侧（prompts.py）

```python
page_scope = context["capabilities"]["surface"]["page_scope"]
addon = _ADMIN_ADDON if page_scope == "admin" else _USER_ADDON
role = "admin" if page_scope == "admin" else "user"
```

`{role}` 被注入到 system prompt 的"当前用户信息"区域，且 admin/user 会使用不同的 prompt addon。

---

## 2. 各层区分对比

| 层级 | 区分方式 | admin | user |
|------|---------|-------|------|
| **工具可见性** | Go 后端 `enabled_tools` + Python 本地工具目录 | 用户工具 + 管理员增量 + admin local tools | 用户工具 + 非 admin local tools |
| **工具执行权限** | Go 后端 isAgentAdminOnlyTool 校验 | 通过 | 被拒（403） |
| **Agent 角色策略** | Python is_actor_allowed_for_tool | 全部 | 排除 ADMIN_ONLY 工具 |
| **System Prompt** | `{role}` 占位符 | 显示 "admin" | 显示 "user" |
| **Prompt 内容** | `_BASE_PROMPT` + 角色 addon | admin addon | user addon |
| **Agent 类型** | 固定 single_agent | 不区分 | 不区分 |
| **LLM 模型** | 固定 "default" client | 不区分 | 不区分 |
| **Skills 知识** | load_all_skills() | 不区分 | 不区分 |

---

## 3. 工具可见性来源

Go 后端 `buildAgentCapabilitiesWithCatalog()` 是当前工具可见性的第一入口：

- `agentUserTools`：普通用户基础工具，当前静态列表 45 个。
- `agentAdminTools`：管理员增量工具，当前静态列表 27 个；只有 `token.RolePlatform == admin` 时合并。
- `localCatalogEntries`：从 Python `/internal/tools/catalog` 动态加载；`admin_only=true` 的本地工具只给管理员。
- `confirm_tools`：从静态 `agentConfirmToolSet` 和本地目录 `mode=confirm` 派生，用于提示词和确认流。
- `tool_catalog`：把静态工具描述和本地目录合并后传给 Agent，供多智能体规划和前端能力展示。

这意味着文档里不应再使用旧的 `USER_TOOL_NAMES` 固定集合来描述普通用户能力；实际能力由 Go 静态列表、Python 本地目录和页面 surface 共同决定。

---

## 4. Admin Only 工具清单

Go 后端 `isAgentAdminOnlyTool()` 与 Python `ADMIN_ONLY_TOOL_NAMES` 共同约束高危或全局工具。代表性类别如下：

| 类别 | 工具 |
|------|------|
| 集群级查询 | get_cluster_health_report, list_cluster_jobs, list_cluster_nodes, get_admin_ops_report |
| 节点详情 | get_node_detail |
| 存储 | list_storage_pvcs, get_pvc_detail, get_pvc_events, inspect_job_storage, get_storage_capacity_overview |
| 网络 / 分布式训练 | get_node_network_summary, diagnose_distributed_job_network, get_node_kernel_diagnostics, get_rdma_interface_status, get_nccl_env_config |
| K8s 全局读 | k8s_list_nodes, get_volcano_queue_state, k8s_get_configmap, k8s_get_networkpolicy, k8s_top_nodes, k8s_top_pods, k8s_rollout_status |
| K8s 用户范围读 | k8s_list_pods, k8s_get_events, k8s_describe_resource, k8s_get_pod_logs, k8s_get_service, k8s_get_endpoints, k8s_get_ingress 对普通用户开放，但后端/本地执行侧必须做所有权收敛 |
| 可观测 | prometheus_query, harbor_check |
| 外部检索 / 运行时 | sandbox_grep, get_agent_runtime_summary |
| 节点和 K8s 写操作 | cordon_node, uncordon_node, drain_node, delete_pod, restart_workload, k8s_scale_workload, k8s_label_node, k8s_taint_node, run_kubectl, execute_admin_command |
| 审计 | get_latest_audit_report, list_audit_items, save_audit_report, mark_audit_handled |
| 通知 | batch_stop_jobs, notify_job_owner |

---

## 5. User 工具清单

普通用户可用的工具是不含 admin-only 的用户域能力，并且 K8s 读工具要收敛到本人拥有的工作负载：

| 类别 | 工具 |
|------|------|
| 作业信息 | get_job_detail, get_job_events, get_job_logs, diagnose_job, get_diagnostic_context |
| 搜索 | search_similar_failures, web_search, fetch_url |
| 指标 | query_job_metrics, analyze_queue_status, get_realtime_capacity |
| 资源 / 镜像 | list_available_images, list_cuda_base_images, list_available_gpu_models, recommend_training_images, list_image_builds, get_image_build_detail, get_image_access_detail |
| 配额 | check_quota |
| 概览 | get_cluster_health_overview |
| 列表 | list_user_jobs |
| 模板 | get_job_templates |
| 统计 | get_failure_stats |
| 闲置检测 | detect_idle_jobs |
| 资源推荐 | get_resource_recommendation |
| K8s 范围读 | k8s_list_pods, k8s_get_events, k8s_describe_resource, k8s_get_pod_logs, k8s_get_service, k8s_get_endpoints, k8s_get_ingress |
| 作业管理（写） | resubmit_job, stop_job, delete_job, create_jupyter_job, create_webide_job, create_training_job, create_custom_job, create_pytorch_job, create_tensorflow_job |
| 镜像管理（写） | create_image_build, manage_image_build, register_external_image, manage_image_access |
| 审批历史 | get_approval_history |

---

## 6. 当前状态（已改进）

### 6.1 Prompt 已按角色分离

admin 和 user 使用不同的 prompt addon:
- **admin**: `_ADMIN_ADDON`（6 条管理员原则 + 集群诊断 + 可观测性指引）
- **user**: `_USER_ADDON`（11 条用户原则，强调歧义澄清、确认优先、个人作业视角）
- **共享**: `_BASE_PROMPT`（13 条核心原则 + 平台规约 + 资源推荐 + 工具选择指引）

选择逻辑: `build_system_prompt()` 优先读取 `context.capabilities.surface.page_scope`（Go 后端计算），不匹配时 fallback 到 page route/url 推断。

### 6.2 Page Context 差异化

- **admin**: `- 当前关注作业: {job_name}`（直接使用）
- **user**: `- 页面上下文（仅供参考，操作前需用户确认）: 用户正在查看作业 {job_name}`
- user 原则 U7 要求基于页面推断意图时必须先向用户确认

### 6.3 仍未区分的部分

| 方面 | 当前状态 | 改进方向 |
|------|---------|---------|
| Agent 类型 | 固定 single_agent | 可按角色选不同 orchestration 模式 |
| LLM 模型 | 固定 "default" client | 可为 admin/user 配置不同 client key |
| Skills 知识 | load_all_skills() | 可按角色过滤技能集 |

---

## 7. 代码位置

| 组件 | 文件 | 关键位置 |
|------|------|---------|
| Go pageScope 判定 | `backend/.../capabilities.go` | pageScope 推断逻辑 |
| Go admin 工具校验 | `backend/.../tools_dispatch.go` | 86-127 (isAgentAdminOnlyTool) |
| Agent actor_role 推断 | `crater-agent/.../graph.py` | 426-433 |
| Prompt role 注入 | `crater-agent/.../prompts.py` | build_system_prompt |
| 工具过滤 | `crater-agent/.../tool_selector.py` | select_tools_for_context |
| 工具权限定义 | `crater-agent/.../definitions.py` | ADMIN_ONLY_TOOL_NAMES, ROLE_ALLOWED_TOOL_NAMES |
