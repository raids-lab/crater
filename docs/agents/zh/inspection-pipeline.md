# 巡检流水线（智能巡检）

> 自动化集群健康巡检 -- 一个定时流水线智能体，收集证据、推理异常并生成结构化运维报告。

---

## 概述

巡检流水线是 Mops 框架中的**任务型智能体**，作为定时后台任务运行（非面向用户的对话）。它定期收集集群指标、作业状态和资源利用率，然后使用 LLM 分析生成结构化巡检报告，展示在管理员仪表板上。

与响应用户消息的对话智能体不同，巡检流水线由 cron 作业触发，以固定的报告 schema 自主运行，前端将其渲染为仪表板卡片。

---

## 架构

```
┌─────────────────────────────────────────────────────────────────┐
│ (1) 触发层 — CronJobManager (Go 后端)                           │
│     robfig/cron 调度器 → patrol 函数注册表                       │
│     API: POST /v1/operations/patrol/{jobName} (手动触发)         │
└──────────────────────────────┬──────────────────────────────────┘
                               │ HTTP POST /pipeline/admin-ops-report
                               │ Headers: X-Agent-Internal-Token
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ (2) 流水线层 — crater-agent (FastAPI)                            │
│     router.py → ops_report.py (编排)                             │
│     ┌─────────────────────────────────────────────────────────┐ │
│     │  步骤 1: 收集计算域数据                                  │ │
│     │  步骤 2: 收集存储域数据                                  │ │
│     │  步骤 3: 收集网络域数据                                  │ │
│     │  步骤 4: 获取上一期报告（趋势对比）                       │ │
│     │  步骤 5: LLM 分析（或确定性回退）                        │ │
│     │  步骤 6: 构建流水线载荷 + 审计项                         │ │
│     │  步骤 7: 持久化到数据库                                  │ │
│     └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────┬──────────────────────────────────┘
                               │ 通过 PipelineToolClient 调用工具
                               │ POST /api/agent/tools/execute
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ (3) 工具层 — Go 后端工具处理器                                    │
│     get_admin_ops_report (聚合 7 个子工具结果)                    │
│     list_storage_pvcs, get_storage_capacity_overview              │
│     get_node_network_summary, diagnose_distributed_job_network    │
│     get_latest_audit_report, save_audit_report                    │
└──────────────────────────────┬──────────────────────────────────┘
                               │ 通过 save_audit_report 持久化 JSON
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ (4) 存储层 — PostgreSQL                                          │
│     ops_audit_reports (报告元数据 + report_json JSONB)            │
│     ops_audit_items (按类别划分的逐作业操作项)                     │
└──────────────────────────────┬──────────────────────────────────┘
                               │ REST API: GET /admin/agent/ops-reports/*
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ (5) 展示层 — 前端 (React)                                        │
│     OpsReportTab.tsx — 60 秒轮询获取最新报告                      │
│     ├─ ReportSummaryCard (执行摘要 + 统计卡片)                   │
│     ├─ FailureAnalysisCard (分类表格 + 受影响用户)                │
│     ├─ SuccessAnalysisCard (资源效率指标)                         │
│     ├─ ResourceUtilizationCard (GPU/CPU/内存进度条 + 告警)        │
│     ├─ RecommendationsCard (按严重程度着色的操作项)                │
│     └─ ReportHistoryTable (分页历史报告)                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## 数据收集

### 计算域

`get_admin_ops_report` 后端工具通过单次调用聚合七个子工具的数据：

| 子工具 | 数据 |
|--------|------|
| 作业查询（已完成） | 成功作业样本（可配置上限，默认 5） |
| 作业查询（已失败） | 失败作业样本（可配置上限，默认 5） |
| `get_cluster_health_overview` | 集群整体健康指标 |
| `get_failure_statistics` | 失败类别分布 |
| `detect_idle_jobs` | 低 GPU 利用率检测 |
| `list_cluster_nodes` | 节点状态快照 |
| `get_realtime_capacity` | 当前资源可用性 |

### 存储域

| 工具 | 数据 |
|------|------|
| `list_storage_pvcs` | PVC 清单、未绑定/异常 PVC |
| `get_storage_capacity_overview` | 存储利用率、容量压力 |

### 网络域

| 工具 | 数据 |
|------|------|
| `get_node_network_summary` | 每节点网络健康状况、降级接口 |
| `diagnose_distributed_job_network` | 分布式作业的 NCCL/RDMA 诊断 |

---

## 分析流水线

### 确定性基线

`build_deterministic_ops_report()` 使用纯 Python 逻辑（无 LLM）从原始数据生成结构化报告。其作用包括：

1. **数值真实来源** -- 所有计数、百分比和聚合值均通过确定性计算得出
2. **回退方案** -- 如果 LLM 分析失败或超时，直接使用确定性报告
3. **合并目标** -- LLM 生成的文本字段合并到确定性结构中

### LLM 增强

`analyze_ops_report_with_llm()` 将基线数据发送给 `ops_report` LLM 客户端（DashScope Qwen），附带结构化提示词。LLM 增强三类字段：

| 字段 | 来源 | LLM 添加的内容 |
|------|------|----------------|
| `executive_summary` | LLM | 自然语言概述 |
| `failure_analysis.patterns` | LLM | 跨作业失败模式分析 |
| `recommendations` | LLM | 按严重程度排序的操作建议 |

### 合并策略

数值字段始终来自确定性报告。LLM 只能覆盖基于文本的分析字段。如果 LLM 输出 JSON 解析失败，确定性报告将原样返回。

```
确定性报告 (数值 + 模板文本)
  ↓
LLM 报告 (executive_summary + patterns + recommendations)
  ↓
_merge_llm_report() — 仅在 LLM 字段非空时覆盖
  ↓
最终 report_json → 保存到数据库
```

---

## 报告 Schema

```json
{
  "executive_summary": "string (2-3 句话)",
  "job_overview": {
    "total": "int",
    "success": "int",
    "failed": "int",
    "pending": "int",
    "success_rate": "float (百分比)",
    "delta": {
      "total": "int (与上一期报告对比)",
      "failed": "int",
      "pending": "int"
    }
  },
  "failure_analysis": {
    "categories": [
      {
        "reason": "string (例如 ContainerError, OOMKilled)",
        "count": "int",
        "top_job": { "name": "string", "owner": "string" }
      }
    ],
    "top_affected_users": ["string"],
    "patterns": "string (失败模式分析)"
  },
  "success_analysis": {
    "avg_duration_by_type": { "training": "float (秒)" },
    "resource_efficiency": {
      "avg_cpu_ratio": "float (0-1)",
      "avg_gpu_ratio": "float (0-1)",
      "avg_memory_ratio": "float (0-1)"
    }
  },
  "resource_utilization": {
    "cluster_gpu_avg": "float (百分比)",
    "cluster_cpu_avg": "float (百分比)",
    "cluster_memory_avg": "float (百分比)",
    "over_provisioned_count": "int",
    "idle_gpu_jobs": "int",
    "node_hotspots": ["string (节点名)"]
  },
  "recommendations": [
    { "severity": "high|medium|low", "text": "string" }
  ]
}
```

前端 (`OpsReportTab.tsx`) 渲染此 JSON 时使用固定组件 -- 每个顶层键对应一个特定的卡片组件。

---

## 触发配置

### Cron 作业参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `days` | 1 | 作业查询回溯窗口 |
| `lookback_hours` | 1 | 近期运行中作业窗口 |
| `gpu_threshold` | 5 | 空闲 GPU 利用率阈值 (%) |
| `idle_hours` | 1 | 空闲检测时间窗口 |
| `running_limit` | 20 | 运行中作业最大采样数 |
| `node_limit` | 10 | 节点快照最大数量 |
| `use_llm` | true | Python `/pipeline/admin-ops-report` 参数；Go 巡检 cron 当前总是使用默认值 true |

### 调度

Cron 调度表达式存储在 `cron_job_configs` 数据库表中，由 `CronJobManager` 管理。作业支持以下状态：
- **定时执行**：按 cron 表达式运行（例如 `0 */1 * * *` 表示每小时一次）
- **手动触发**：通过 `POST /v1/operations/patrol/trigger-admin-ops-report` 触发
- **暂停**：暂停但不删除

---

## Mops 集成

巡检流水线作为**任务型智能体**（第三种编排模式）接入 Mops 框架：

```
后端事件 (cron 触发)
  → [流水线智能体通过 PipelineToolClient 访问工具]
  → 结构化结果 (report_json)
  → 后端持久化 + 前端渲染
```

### 共享基础设施

| 组件 | 是否与对话智能体共享？ | 备注 |
|------|----------------------|------|
| 工具定义 | 是 | 使用相同的 `tools/definitions.py` |
| 工具执行 | 部分共享 | 使用 `PipelineToolClient`（非 `GoBackendToolExecutor`） |
| LLM 客户端 | 是 | 使用相同的 `ModelClientFactory`、`ops_report` 客户端配置 |
| Token 管理 | 否 | 单次 LLM 调用，无 ReAct 循环 |
| 审计追踪 | 独立 | `ops_audit_reports` 表（非 `agent_sessions`） |

### PipelineToolClient 与 GoBackendToolExecutor 对比

| 维度 | PipelineToolClient | GoBackendToolExecutor |
|------|-------------------|----------------------|
| 身份 | 固定系统身份 (`agent-pipeline`) | 来自会话的用户身份 |
| 认证 | `X-Agent-Internal-Token` | `X-Agent-Internal-Token` + 用户上下文 |
| 会话 | 静态流水线会话 ID | 按对话分配的会话 |
| 角色 | 始终为 `admin` | 从用户角色派生 |
| 使用者 | 巡检流水线、GPU 审计 | 对话智能体（单轮、多轮、任务） |

---

## 通知与邮件设计

### 当前状态

巡检流水线当前会生成并保存报告，并且 `admin_ops_report` 已支持按策略发送邮件：
- `GpuAuditRequest.auto_notify` 是保留字段，尚未实现自动通知。
- 对话工具 `notify_job_owner` 已通过 Go `pkg/alert` 向作业所有者发邮件，但仍是需要确认的 Agent 工具。
- `AdminOpsReportService.TriggerAdminOpsReport()` 会在 `cron_job_configs.config.notification.enabled=true` 时读取已保存的报告并触发确定性巡检通知。
- backend 已有 SMTP 能力，集中在 `pkg/alert`：作业开始、失败、完成、低利用率提醒、长时间运行提醒和删除通知会复用 `Alert` 表做每个 `job_name + alert_type` 的去重。

### 已实现设计

自动巡检邮件不依赖用户聊天 tool 作为调度入口。SMTP 凭据、收件人解析、严重度阈值、去重和冷却都在 Go 后端；Python Agent 只负责给出结构化报告和可选的自然语言摘要。

```
CronJobManager
  ↓
AdminOpsReportService.TriggerAdminOpsReport()
  ↓
Python pipeline 生成 report_json + ops_audit_items
  ↓
Go 保存/读取报告
↓
admin_ops_report_notifications.go 按策略筛选通知项
  ├─ 管理员邮件：严重集群/存储/网络/容量问题
  └─ 用户邮件：失败作业数或失败率达到阈值时通知对应作业 owner
  ↓
pkg/alert SMTP handler + notification audit/dedupe
```

关键原则：
- **SMTP 配置只放 Go 后端**。`platform-runtime.yaml` 明确不承载 `smtp.*`；Python 不应直接持有邮件账号密码。
- **系统巡检不走 HITL**。触发源是 cron/system，发送动作由确定性阈值、去重和频控保护；普通聊天中的写操作仍保持确认机制。
- **LLM 只写表达，不决定权限边界**。邮件正文中的摘要、影响分析和建议可以来自 `report_json.executive_summary` / `recommendations`，但收件人、严重级别、冷却窗口由后端规则决定。
- **复用现有邮件风格**。沿用 `generateHTMLEmail()` 的标题、正文、CTA 按钮结构；新增模板时提取公共 header/footer，避免每个告警复制一段 HTML。

### 收件人与策略

通知策略作为巡检任务配置的一部分，而不是新增一份独立 agent 配置。原因是阈值和调度强相关：每天巡检、每小时巡检、GPU 专项巡检需要不同的严重度门槛和冷却时间。

示例 `cron_job_configs.config`：

```json
{
  "days": 1,
  "lookback_hours": 1,
  "gpu_threshold": 5,
  "idle_hours": 1,
  "running_limit": 20,
  "node_limit": 10,
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

收件人解析：
- 管理员：优先平台 admin 用户邮箱；无可用 admin 邮箱时 fallback 到 `smtp.notify`。
- 用户：从 `ops_audit_items.job_name` 反查 Job Owner；owner 邮箱缺失时跳过。
- 去重：复用 `Alert` 表。管理员通知使用稳定 key `ops-report:admin_ops_report:admins`，作业所有者通知使用 `job_name + alert_type`，配合 `cooldown_hours` 防止同一问题每轮重复轰炸。

---

## 安全性与可靠性

### 超时链

```
Go 后端 HTTP 超时: 3 分钟
  └─ PipelineToolClient: 单个后端工具调用默认 120 秒
      └─ LLM 分析超时: 45 秒
          └─ 回退: 确定性报告（无 LLM）
```

### 错误恢复

| 故障 | 恢复方式 |
|------|---------|
| 后端工具调用失败 | 流水线返回错误状态，不保存报告 |
| LLM 超时 (> 45s) | 回退到确定性报告 |
| LLM 返回无效 JSON | 回退到确定性报告 |
| LLM 部分成功 | 仅合并有效字段，其余来自确定性报告 |
| 数据库保存失败 | 流水线返回成功但记录警告；报告未持久化 |
| 邮件发送失败（设计目标） | 不影响报告保存；记录 notification audit 和错误原因，等待下轮或人工重试 |

### 只读保证

流水线在数据收集阶段仅调用只读工具。当前唯一的写操作是 `save_audit_report`，用于持久化最终报告。新增邮件通知后也应只发送通知和写审计记录，不停止任何作业，不修改任何资源。

---

## 代码

| 组件 | 文件 |
|------|------|
| 流水线入口 | `crater_agent/pipeline/ops_report.py` |
| LLM 分析 | `crater_agent/pipeline/ops_report_llm.py` |
| 流水线 API 路由 | `crater_agent/pipeline/router.py` |
| 流水线工具客户端 | `crater_agent/pipeline/tool_client.py` |
| 报告工具库 | `crater_agent/report_utils.py` |
| 后端巡检触发 | `backend/pkg/patrol/patrol.go` |
| 后端服务 | `backend/internal/service/admin_ops_report_service.go` |
| 后端数据工具 | `backend/internal/handler/agent/tools_readonly.go` (`toolGetAdminOpsReport`) |
| 后端保存工具 | `backend/internal/handler/agent/tools_readonly.go` (`toolSaveAuditReport`) |
| 后端 API 处理器 | `backend/internal/handler/agent/ops_report_api.go` |
| Cron 管理器 | `backend/pkg/cronjob/manger.go` |
| 数据库模型 | `backend/dao/model/cron_job.go` |
| 数据库迁移 | `backend/hack/sql/20260404_ops_audit.sql`, `20260405_ops_report_enhance.sql` |
| 前端 API | `frontend/src/services/api/ops-report.ts` |
| 前端 UI | `frontend/src/components/aiops/OpsReportTab.tsx` |
| LLM 配置 | `crater-agent/config/llm-clients.json` (`ops_report` 客户端) |
