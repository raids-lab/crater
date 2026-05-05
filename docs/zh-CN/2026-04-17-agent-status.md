# Crater Agent 现状全景评审（2026-04-17）

> 基于 2026-04-17 仓库代码实际分析，覆盖工具体系、编排架构、上下文与记忆机制、K8s/Prometheus 能力边界、冗余与优化方向。
>
> 本文不沿用 04-07 文档口径，所有结论均从当日代码一手确认。

---

## 0. 一句话结论

当前 crater-agent 拥有 **76 个工具定义**（优化后：59 自动 + 17 确认）、**双编排模式**（单智能体 ReAct / 多智能体 Coordinator 流水线），已具备故障诊断、集群巡检、受控运维操作的完整链路。本次优化完成以下改进：

**已完成的优化（04-17）：**
- 移除 5 个冗余/无效工具（sandbox 三件套 + get_cluster_health_overview + get_job_events）
- 新增 6 个 K8s 工具（3 只读：top_nodes/top_pods/rollout_status；3 写操作：scale/label/taint，均在 Python 侧闭环）
- 动态工具注入：根据页面路由选择 26-42 个相关工具（原先全量 76 个绑定 LLM）
- 主动上下文预算管理（不再等 API 报错才压缩）
- History 分级截断（错误信息保留 500 字符，普通结果 200 字符）
- Multi-agent evidence 滑动窗口（最多 30 条）
- System Prompt 补全 Prometheus/工具选择指引

---

## 1. 工具体系全景

### 1.1 工具总量与分类

| 类别 | 数量 | 执行方式 | 权限 | 确认 |
|------|------|----------|------|------|
| 作业诊断（get_job_detail, diagnose_job 等） | 7 | Go Backend | 用户级 | 自动 |
| 指标与队列（query_job_metrics, analyze_queue_status 等） | 3 | Go Backend | 用户级 | 自动 |
| 镜像/资源推荐 | 5 | Go Backend | 用户级 | 自动 |
| 健康聚合（get_health_overview, list_user_jobs 等） | 3 | Go Backend | 用户级 | 自动 |
| 集群管理员只读（cluster_health, ops_report 等） | 6 | Go Backend | 管理员 | 自动 |
| 审计（audit_report, audit_items 等） | 4 | Go Backend | 管理员 | 自动 |
| 存储/网络（PVC, network_summary 等） | 7 | Go Backend | 管理员 | 自动 |
| K8s 只读（list_nodes, list_pods, describe 等） | 11 | 本地 kubectl | 管理员* | 自动 |
| 调度器/基础设施（volcano, zombie, ddp, kernel, rdma） | 6 | 本地 kubectl | 管理员 | 自动 |
| 可观测性（prometheus_query, harbor_check） | 2 | 本地 HTTP | 管理员 | 自动 |
| 外部搜索（web_search, fetch_url） | 2 | 本地 CAMEL | 管理员 | 自动 |
| 沙箱文件（sandbox_grep/list/read） | 3 | 本地文件 I/O | 管理员 | 自动 |
| 代码执行（execute_code） | 1 | 本地 CAMEL | 管理员/executor | 自动 |
| 运行时元信息（get_agent_runtime_summary） | 1 | 本地 | 全部 | 自动 |
| **用户级写操作**（stop/delete/resubmit/create_*） | 5 | Go Backend | 用户级 | **需确认** |
| **管理员写操作**（cordon/drain/delete_pod/restart 等） | 9 | Go Backend | 管理员 | **需确认** |
| **合计** | **~75 去重后有效** | | | |

> *注：K8s 只读工具对非管理员用户有 ownership scoping——通过 `/api/agent/k8s-ownership` 接口校验 Pod 归属。

### 1.2 双层执行架构

```
用户请求 → Python Agent → CompositeToolExecutor
                              ├─ LocalToolExecutor（24 个工具，本地执行）
                              │   ├─ kubectl 子进程（K8s 工具）
                              │   ├─ HTTP 直连（Prometheus / Harbor）
                              │   ├─ CAMEL Toolkit（web_search / execute_code）
                              │   └─ 文件 I/O（sandbox_grep / list / read）
                              └─ GoBackendToolExecutor（其余工具）
                                  POST /api/agent/tools/execute
                                  Header: X-Agent-Internal-Token
```

路由规则：`CompositeToolExecutor` 根据 `route_for_tool()` 决定走本地还是后端，可通过环境变量 `CRATER_AGENT_TOOL_ROUTE_<tool_name>=local|backend` 覆盖。

### 1.3 确认流机制

所有写操作工具（CONFIRM_TOOLS 列表）执行时：

1. Go Backend 返回 `{"status": "confirmation_required", "confirm_id": "...", "risk_level": "high"}`
2. Agent 暂停，前端展示确认卡片
3. 用户批准后，续跑（单智能体恢复 graph，多智能体恢复 checkpoint）
4. 全程有审计记录（AgentToolCall 表）

**无预批准白名单**——每次执行都需要用户逐一确认。

---

## 2. 编排架构

### 2.1 单智能体模式（single_agent）

**文件**：`crater_agent/agent/graph.py`

经典三节点 LangGraph ReAct 循环：

```
agent_node ──(有 tool_calls)──→ tools_node ──→ agent_node（循环）
    │                                              │
    │(无 tool_calls)                    (达上限)    │
    ↓                                              ↓
   END                                    summarize_node → END
```

- **工具调用上限**：每轮最多 15 次（`max_tool_calls_per_turn`）
- **上下文溢出处理**：捕获 `BadRequestError`，被动压缩消息（保留 system + 最后 6 条 + 截断）
- **Qwen 思维模式**：支持提取 `reasoning_content`

**局限**：
- 无主动上下文预算管理——等 API 报错才压缩，浪费一次调用
- 工具达上限时硬切到 summarize，无优雅降级
- 同一轮内相同工具+参数不去重（仅多智能体模式有签名去重）

### 2.2 多智能体模式（multi_agent）

**文件**：`crater_agent/orchestrators/multi.py`

Coordinator 驱动的阶段制流水线：

```
                    ┌───────────────────────────────────┐
                    │         Coordinator LLM            │
                    │  决定: plan / observe / act /       │
                    │        replan / finalize            │
                    └──────┬────────────────────┬────────┘
                           │                    │
              ┌────────────▼──────┐   ┌────────▼──────────┐
              │   Planner Agent   │   │  Explorer Agent    │
              │ 生成调查步骤+候选工具│   │ 只读工具循环采证     │
              └───────────────────┘   └────────────────────┘
                                              │
                                    ┌─────────▼──────────┐
                                    │  Executor Agent     │
                                    │ 执行写操作+确认流     │
                                    └─────────────────────┘
```

**阶段转移逻辑**（确定性快速路径 + LLM 兜底）：

| 条件 | 走向 | 决策方式 |
|------|------|----------|
| 首轮且无 plan | → plan | 确定性 |
| 确认恢复后无待执行动作 | → finalize | 确定性 |
| 已知写操作且无前置工作 | → act | 确定性 |
| 其他情况 | → LLM 判断 | Coordinator LLM |

**护栏配置**（`MASRuntimeConfig`）：

| 参数 | 默认值 | 含义 |
|------|--------|------|
| `lead_max_rounds` | 8 | Coordinator 最大循环轮数 |
| `subagent_max_iterations` | 25 | 每个子智能体最大工具调用次数 |
| `no_progress_rounds` | 2 | 连续无新证据则停止 |
| `tool_timeout_seconds` | 60 | 单次工具执行超时 |
| `max_actions_per_round` | 4 | 每轮最大并发动作数 |

**局限**：
- Coordinator 每轮都调一次 LLM，8 轮 × 多次决策 = 大量 LLM 调用开销
- `state.tool_records` 无上限增长，无 eviction
- 工具签名去重基于 JSON 字符串，参数顺序不同会误判为不同调用
- Checkpoint 序列化全部 artifact，无自适应裁剪

### 2.3 角色权限矩阵

| 角色 | 只读工具 | 写操作工具 | execute_code |
|------|---------|-----------|-------------|
| planner | 可用（除 executor-only） | 不可用 | 不可用 |
| coordinator | 可用（除 executor-only） | 不可用 | 不可用 |
| explorer | 可用（除 executor-only） | 不可用 | 不可用 |
| executor | 全部 | 全部（需确认） | **可用** |
| verifier | 可用（除 executor-only） | 不可用 | 不可用 |
| guide | 无工具 | 不可用 | 不可用 |
| general | 可用（除 executor-only） | 不可用 | 不可用 |
| single_agent | 全部 | 全部（需确认） | **可用** |

---

## 3. 上下文与记忆机制

### 3.1 System Prompt 构成

**文件**：`crater_agent/agent/prompts.py`

| 组成部分 | 内容 | 是否动态 |
|----------|------|----------|
| 22 条核心原则 | 采集优先、不臆测、写操作确认等 | **静态** |
| 平台规格 | 家目录、数据挂载、资源上下限 | **静态** |
| 资源推荐流程 | 提交辅助决策树 | **静态** |
| 管理员诊断规则 | 集群诊断 SOP | **静态** |
| 页面上下文 | 当前聚焦的 job_name/node_name/pvc_name | **动态（每轮）** |
| 能力声明 | confirm_tools 列表 | **动态** |
| Skills 上下文 | 从 skills 目录加载 | **首次加载后固定** |
| 新用户引导 | emoji 功能指南 | **条件性** |

**问题**：System prompt 体积约 2000-3000 tokens，24 条原则完全静态，不会根据 LLM 已完成的工作自适应裁剪。

### 3.2 History 管理

**文件**：`crater_agent/memory/session.py`

- **来源**：Go Backend 在 `request.context.history` 中传入历史消息
- **预算**：固定 4000 tokens（`history_max_tokens`）
- **策略**：从最新消息向前加载，直到 token 预算耗尽
- **工具结果截断**：160 字符（极度激进）
- **Token 估算**：中文 1 token/2 字符，英文 1 token/4 字符（粗略）

**问题**：
- 4000 tokens 固定不考虑当前 system prompt 大小或上下文窗口余量
- 工具结果截断到 160 字符，关键错误信息可能丢失
- 无跨会话持久记忆，Checkpoint 传回前端而非服务端存储

### 3.3 当前轮内工具结果截断

**文件**：`crater_agent/agent/graph.py` `_build_tool_observation()`

| 内容类型 | 截断长度 |
|----------|----------|
| dict 结果 | 1400 字符 |
| 错误信息 | 1200 字符 |
| 日志内容 | 1000 字符 |
| 普通文本 | 1400 字符 |

### 3.4 被动压缩机制

当 LLM API 返回 context limit error 时（而非主动管理）：

1. 保留 system message
2. 保留最后 6 条消息
3. 如果最后一条 HumanMessage 不在尾部 6 条中，额外保留
4. 截断：System 1600 字符 / Human 600 字符 / Tool 800 字符 / AI 600 字符

**核心问题**：这是一次浪费的 API 调用触发的被动行为，不是主动的 context window 预算管理。

---

## 4. K8s 与 Prometheus 能力深度分析

### 4.1 K8s 工具：严格参数化，LLM 不能生成任意命令

**所有 K8s 工具都是结构化参数 → 固定 kubectl 命令模板**，LLM 不能注入任意 kubectl 命令：

| 工具 | LLM 可控参数 | 实际生成的 kubectl | 灵活度 |
|------|-------------|-------------------|--------|
| `k8s_list_nodes` | label_selector, field_selector, limit | `kubectl get nodes -l {label} --field-selector {field} -o json` | 低 |
| `k8s_list_pods` | namespace, label_selector, field_selector, node_name, limit | `kubectl get pods -n {ns} -l {label} --field-selector {field} -o json` | 低 |
| `k8s_get_events` | namespace, field_selector, limit | `kubectl get events --field-selector {field} -o json` | 低 |
| `k8s_describe_resource` | kind (Pod/VCJob only for non-admin), name, namespace | `kubectl describe {kind} {name} -n {ns}` | **极低** |
| `k8s_get_pod_logs` | pod_name, container, tail (1-5000), since_seconds, previous | `kubectl logs {pod} --tail {n} -c {container}` | 低 |
| `k8s_get_service` | name, namespace, label_selector | `kubectl get svc -o json` | 低 |
| `k8s_get_configmap` | name, namespace | `kubectl get configmap -o json`（data 截断 2KB/field） | 极低 |
| `k8s_get_networkpolicy` | name, namespace | `kubectl get networkpolicy -o json` | 极低 |

**结论**：K8s 工具层面 **安全但不灵活**。LLM 不能做到 "根据诊断结果自主决定执行 `kubectl top nodes`、`kubectl rollout status` 或 `kubectl scale`" 这类自由操作。

### 4.2 Prometheus：唯一允许 LLM 自由生成查询的工具

**`prometheus_query` 是整个系统中唯一真正由 LLM 自由构造查询语句的工具。**

```python
query = str(tool_args.get("query") or "").strip()  # LLM 自由写 PromQL
# 直接发送到 Prometheus /api/v1/query 或 /api/v1/query_range
```

| 能力 | 状态 |
|------|------|
| LLM 自由编写 PromQL | **支持** |
| 即时查询 / 范围查询 | 支持（query_type 参数） |
| PromQL 语法校验 | **无** |
| 查询复杂度限制 | **无**（仅结果集裁剪：max_series=20, max_points=120） |
| 速率限制 | **无** |

**风险**：LLM 可能生成高基数查询导致 Prometheus 压力，目前仅靠结果集后裁剪缓解。

### 4.3 管理员写操作：6 个写死动作，无 LLM 自主判断

**文件**：`backend/internal/handler/agent/tools_cluster_write.go`

| 工具 | 实际操作 | LLM 可控性 |
|------|----------|-----------|
| `cordon_node` | `UpdateNodeUnschedule(true)` | 仅 node_name + reason |
| `uncordon_node` | `UpdateNodeUnschedule(false)` | 仅 node_name + reason |
| `drain_node` | `DrainNode()` | 仅 node_name + reason |
| `delete_pod` | `CoreV1().Pods().Delete()` | pod_name, namespace, force, grace_period |
| `restart_workload` | 更新 Pod template annotation 触发滚动重启 | kind (Deployment/StatefulSet/DaemonSet), name, namespace |
| `run_kubectl` | 执行高风险 kubectl 写命令 | command + reason（统一走确认与策略校验） |

**核心问题**：

1. **LLM 不能只靠单一 Go 写工具闭环所有场景**——仍需要结构化工具 + 高风险兜底命令并存
2. **高风险自由命令必须显式确认**：不能把 `kubectl patch/apply/scale` 等混进普通查询链路
3. **复杂多步运维仍需编排能力**：例如发现 OOM 后自动调整 resource limits 并重提作业
4. **K8s 写路径需要统一审计**：不管最终在 Go 还是 Python local 执行，都要先进入确认与落库链路

---

## 5. 工具冗余分析

### 5.1 明确冗余

| 冗余组 | 工具 | 问题 | 建议 |
|--------|------|------|------|
| 作业诊断 | `get_job_detail` vs `get_diagnostic_context` | diagnostic_context 是 detail 的超集（含事件+指标+调度） | 考虑合并，detail 作为 context 的轻量模式 |
| 集群健康 | `get_cluster_health_overview` vs `get_cluster_health_report` vs `get_admin_ops_report` | 三者高度重叠，report 包含 overview 的全部信息 | 合并为一个带 detail_level 参数的工具 |
| 节点信息 | `get_node_detail`（Backend）vs `k8s_list_nodes`（本地） | 两者都返回节点状态，来源不同 | 保留但明确分工：Backend 聚合 vs K8s 实时 |
| 事件查询 | `get_job_events`（Backend）vs `k8s_get_events`（本地） | job_events 是 k8s_get_events 的场景化封装 | 可接受，但 LLM 可能困惑该用哪个 |

### 5.2 潜在可合并

| 组 | 工具 | 建议 |
|----|------|------|
| 镜像推荐 | `list_available_images` + `recommend_training_images` | 合并为 `find_images(task_description?, framework?)` |
| 资源推荐 | `list_available_gpu_models` + `get_resource_recommendation` | 合并为 `recommend_resources(task_description?)` |
| 存储查询 | `list_storage_pvcs` + `get_storage_capacity_overview` | 合并为 `query_storage(scope="overview"|"detail")` |

**工具数量过多的实际影响**：110 个工具定义全部作为 function schema 传给 LLM，占用大量上下文窗口。即使角色过滤后 single_agent 仍可见全部工具，对小模型（如 Qwen-72B）决策质量有显著负面影响。

---

## 6. 关键优化方向

### 6.1 K8s 操作灵活性（优先级：高）

**现状问题**：管理员操作被限制在 6 个硬编码动作，LLM 无法自主判断执行更丰富的运维操作。

**建议方案**：

| 方案 | 描述 | 风险 | 推荐度 |
|------|------|------|--------|
| A. 增加更多参数化工具 | 为 `kubectl scale`, `kubectl rollout`, `kubectl patch` 等逐一添加参数化工具 | 低风险，但工作量大 | 中 |
| B. 引入受控 kubectl 代理 | 允许 LLM 生成 kubectl 命令，但经白名单动词 + 资源类型过滤后执行 | 中风险，需审计 | **高** |
| C. 引入运维 Playbook | 预定义常见故障的多步修复流程，LLM 选择 Playbook 而非单一命令 | 低风险，可扩展 | **高** |

方案 B 示例设计：

```python
# 允许 LLM 生成的 kubectl 命令模板
ALLOWED_KUBECTL_PATTERNS = {
    "scale": ["deployment", "statefulset"],      # kubectl scale deployment X --replicas=N
    "rollout": ["deployment", "statefulset"],      # kubectl rollout restart/status
    "top": ["nodes", "pods"],                      # kubectl top nodes/pods
    "label": ["nodes"],                            # kubectl label nodes X key=value
    "taint": ["nodes"],                            # kubectl taint nodes X key=value:effect
}
# 所有生成的命令仍需用户确认
```

### 6.2 上下文管理（优先级：高）

| 问题 | 建议 |
|------|------|
| 被动压缩（等 API 报错） | 改为主动 token 预算：计算 system_prompt + history + 当前轮消息总量，预判是否超限 |
| History 固定 4000 tokens | 改为动态预算：`context_window - system_prompt - reserve_for_response` |
| 工具结果截断 160 字符 | 分级截断：错误信息保留完整（至少 500 字符），正常结果可激进截断 |
| multi-agent evidence 无上限 | 添加 evidence 滑动窗口或 LLM 摘要压缩 |
| 110 个工具 schema 占满上下文 | **动态工具注入**：根据当前意图/阶段只绑定相关子集的工具 |

### 6.3 动态工具注入（优先级：高）

当前 single_agent 模式下 LLM 可见全部工具，这对小模型有很大上下文压力。

建议引入 **分层工具注入**：

```
阶段 1（意图识别后）：只注入 5-10 个与意图相关的核心工具
阶段 2（诊断中）：根据已采集证据，追加 3-5 个深入工具
阶段 3（操作阶段）：注入写操作工具
```

### 6.4 故障自愈闭环（优先级：中）

当前链路是 **诊断 → 展示 → 人工决策**，缺少 **诊断 → 建议修复方案 → 用户确认 → 自动执行** 的完整闭环。

需要补充：
1. 根因到修复方案的映射（Playbook 或 LLM 推理）
2. 多步操作的事务编排（如：cordon → drain → 修复 → uncordon）
3. 操作回滚机制（执行失败时自动恢复）

### 6.5 主动告警触发（优先级：低，但对论文叙事重要）

当前所有诊断都是 **用户主动询问** 触发。缺少：
- Prometheus AlertManager webhook → 自动拉起 agent 排查
- 定时巡检 → 生成运维报告
- 异常检测 → 主动通知

这对 "智能运维" 叙事非常关键——真正的 AIOps 不应该等人来问。

---

## 7. 与 04-07 文档对比：两周内的变化

| 维度 | 04-07 状态 | 04-17 状态 | 变化 |
|------|-----------|-----------|------|
| 工具总数 | ~80（估计） | 110（确认） | 新增存储/网络/RDMA/审计/代码执行等 |
| 本地执行工具 | K8s + Prometheus | 24 个（含 CAMEL web_search, execute_code） | 扩展了本地执行层 |
| web_search | 自建 HTML scraping | 替换为 CAMEL SearchToolkit | 重构完成 |
| execute_code | 无 | CAMEL sandbox（executor-only） | 新增 |
| 多智能体 | "可运行但更接近场景化串行" | 完整 5 阶段流水线 + checkpoint/resume | 成熟度提升 |
| 工具签名去重 | 无 | 多智能体模式支持（基于 JSON 字符串） | 新增 |
| 管理员操作灵活性 | 6 个硬编码 | 仍为 6 个硬编码 | **未改善** |
| 上下文管理 | 被动压缩 | 仍为被动压缩 | **未改善** |
| 动态工具注入 | 无 | 无 | **未改善** |

---

## 8. 对论文实验设计的影响

基于以上分析，建议实验聚焦以下可验证的能力：

### 可直接验证的能力（工具链路已通）

1. **故障诊断准确性**：diagnose_job + search_similar_failures + get_diagnostic_context 链路完整
2. **证据采集完整性**：multi-agent explorer 的工具循环 + evidence 收集
3. **受控操作安全性**：确认机制 + 审计日志 + 权限隔离
4. **Prometheus 自由查询**：LLM 生成 PromQL 的准确性和有效性

### 需要补充才能验证的能力

1. **故障自愈闭环**：需扩展管理员操作工具集或引入 Playbook
2. **主动监控触发**：需要 AlertManager → Agent 的集成
3. **复杂运维编排**：需要多步操作事务支持

### 不建议作为核心实验的方向

1. 提交辅助（更偏交互增强，不是 AIOps 核心）
2. 通用对话（guide/general 角色能力，与 AIOps 无直接关系）

---

## 9. 总结：当前 Agent 在智能运维坐标中的位置

```
        被动响应 ◄──────────────────────────────► 主动感知
            │                                        │
  ┌─────────┼────────────────────────────────────────┤
  │         │          ★ 当前位置                     │
  │  只读诊断│     诊断+受控操作      诊断+自愈+主动告警 │
  │         │     (6个写死动作)       (灵活操作编排)     │
  └─────────┼────────────────────────────────────────┤
            │                                        │
    单步查询 ◄──────────────────────────────────────► 多步编排
```

**当前定位**：已超越单步查询，具备多步诊断流水线，但在操作层面仍停留在 "6 个固定动作 + 人工确认"，距离 "LLM 自主判断 + 自动修复" 还有明确的工程差距。

这个差距不大——核心架构（确认机制、审计、权限隔离）已经就绪，缺的是：
1. 更多参数化写操作工具或受控 kubectl 代理
2. 主动上下文预算管理
3. 动态工具注入减少上下文压力
