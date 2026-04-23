# 角色区分机制（Admin / Portal）

> 记录当前 admin 与 portal（普通用户）的区分实现：从后端传参到 agent 的 prompt/工具/权限各层。
> 最后更新: 2026-04-21

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
  - enabled_tools: 根据 pageScope 过滤
  - confirm_tools: 写操作工具列表
  - surface: { page_scope, page_route, page_url }
```

### 1.2 Agent 侧（graph.py tools_node）

Agent 侧在 tools_node 中重新推断 actor_role（`graph.py:426-433`）:

```python
actor_role = str(actor.get("role") or "user").strip().lower() or "user"
page = context.get("page", {})
route = str(page.get("route") or "").strip().lower()
url = str(page.get("url") or "").strip().lower()

# 后备推断：即使 actor 声称是 user，如果在 admin 页面则升级为 admin
if actor_role == "user" and (
    route.startswith("/admin") or "/admin/" in route or
    url.startswith("/admin") or "/admin/" in url
):
    actor_role = "admin"
```

### 1.3 Prompt 侧（prompts.py）

```python
role = "admin" if route.startswith("/admin") or page_url.startswith("/admin") else "user"
```

`{role}` 被注入到 system prompt 的"当前用户信息"区域。

---

## 2. 各层区分对比

| 层级 | 区分方式 | admin | user |
|------|---------|-------|------|
| **工具可见性** | Go 后端 enabled_tools 过滤 | 全部 ~91 工具 | ~52 工具（USER_TOOL_NAMES） |
| **工具执行权限** | Go 后端 isAgentAdminOnlyTool 校验 | 通过 | 被拒（403） |
| **Agent 角色策略** | Python is_actor_allowed_for_tool | 全部 | 排除 ADMIN_ONLY 工具 |
| **System Prompt** | `{role}` 占位符 | 显示 "admin" | 显示 "user" |
| **Prompt 内容** | **完全相同的模板** | 24 条原则全部可见 | 24 条原则全部可见 |
| **Agent 类型** | 固定 single_agent | 不区分 | 不区分 |
| **LLM 模型** | 固定 "default" client | 不区分 | 不区分 |
| **Skills 知识** | load_all_skills() | 不区分 | 不区分 |

---

## 3. Admin Only 工具清单

Go 后端 `isAgentAdminOnlyTool()`（`tools_dispatch.go:86-127`）:

| 类别 | 工具 |
|------|------|
| 集群级查询 | get_cluster_health_report, list_cluster_jobs, list_cluster_nodes, get_admin_ops_report |
| 节点详情 | get_node_detail |
| 存储 | list_storage_pvcs, get_pvc_detail, get_pvc_events, inspect_job_storage, get_storage_capacity_overview |
| 网络 | get_node_network_summary, diagnose_distributed_job_network |
| K8s 直接操作 | k8s_list_nodes, k8s_list_pods, k8s_get_events, k8s_describe_resource, k8s_get_pod_logs |
| 可观测 | prometheus_query, harbor_check |
| 运维 | run_ops_script, get_runtime_summary, web_search, sandbox_grep |
| 节点管理（写） | cordon_node, uncordon_node, drain_node, delete_pod, restart_workload |
| 审计 | get_latest_audit_report, list_audit_items, save_audit_report, mark_audit_handled |
| 通知 | batch_stop_jobs, notify_job_owner |

---

## 4. User 工具清单

普通用户可用的工具（**不含** admin only 工具）:

| 类别 | 工具 |
|------|------|
| 作业信息 | get_job_detail, get_job_events, get_job_logs, diagnose_job, get_diagnostic_context |
| 搜索 | search_similar_failures |
| 指标 | query_job_metrics, analyze_queue_status, get_realtime_capacity |
| 资源 | list_available_images, list_cuda_base_images, list_available_gpu_models, recommend_training_images |
| 配额 | check_quota |
| 概览 | get_cluster_health_overview |
| 列表 | list_user_jobs |
| 模板 | get_job_templates |
| 统计 | get_failure_stats |
| 闲置检测 | detect_idle_jobs |
| 资源推荐 | get_resource_recommendation |
| 作业管理（写） | resubmit_job, stop_job, delete_job, create_jupyter_job, create_training_job |
| 审批历史 | get_approval_history |

---

## 5. 当前状态（已改进）

### 5.1 Prompt 已按角色分离

admin 和 user 使用不同的 prompt addon:
- **admin**: `_ADMIN_ADDON`（6 条管理员原则 + 集群诊断 + 可观测性指引）
- **user**: `_USER_ADDON`（10 条用户原则，强调歧义澄清、确认优先、个人作业视角）
- **共享**: `_BASE_PROMPT`（12 条核心原则 + 平台规约 + 资源推荐 + 工具选择指引）

选择逻辑: `build_system_prompt()` 优先读取 `context.capabilities.surface.page_scope`（Go 后端计算），不匹配时 fallback 到 page route/url 推断。

### 5.2 Page Context 差异化

- **admin**: `- 当前关注作业: {job_name}`（直接使用）
- **user**: `- 页面上下文（仅供参考，操作前需用户确认）: 用户正在查看作业 {job_name}`
- user 原则 U7 要求基于页面推断意图时必须先向用户确认

### 5.3 仍未区分的部分

| 方面 | 当前状态 | 改进方向 |
|------|---------|---------|
| Agent 类型 | 固定 single_agent | 可按角色选不同 orchestration 模式 |
| LLM 模型 | 固定 "default" client | 可为 admin/user 配置不同 client key |
| Skills 知识 | load_all_skills() | 可按角色过滤技能集 |

---

## 6. 代码位置

| 组件 | 文件 | 关键位置 |
|------|------|---------|
| Go pageScope 判定 | `backend/.../capabilities.go` | pageScope 推断逻辑 |
| Go admin 工具校验 | `backend/.../tools_dispatch.go` | 86-127 (isAgentAdminOnlyTool) |
| Agent actor_role 推断 | `crater-agent/.../graph.py` | 426-433 |
| Prompt role 注入 | `crater-agent/.../prompts.py` | 133 |
| 工具过滤 | `crater-agent/.../tool_selector.py` | select_tools_for_context |
| 工具权限定义 | `crater-agent/.../definitions.py` | ADMIN_ONLY_TOOL_NAMES, ROLE_ALLOWED_TOOL_NAMES |
