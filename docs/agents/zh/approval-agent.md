# 审批智能体

> 用于自动评估作业锁定审批工单的工单智能体。
> 继承自 [TicketAgent](ticket-agent.md) 基类。

---

## 用途

当用户请求延长运行中作业的锁定（防止自动清理）时，审批智能体评估该请求应自动批准、给予紧急保护，还是转交人工管理员处理。

这是 `TicketAgent` 框架的第一个实例。它仅定义领域特定部分（工具、提示词、裁决解析）；整个评估流水线（ReAct 循环、回退、错误处理、轨迹收集）均继承自基类。

---

## 设计

### 集成点

```
用户提交锁定请求 (POST /v1/approvalorder)
  ↓
Go 后端: checkAutoApprovalEligibility()
  ├─ 通过 (< 12h, 冷却期 OK) → system_auto 批准 → 完成
  └─ 未通过 → 智能体评估钩子
              ↓
         Go 后端调用 crater-agent 的 POST /evaluate/approval
              ↓
         ApprovalAgent 使用受限工具运行 ReAct 循环
              ↓
         返回结构化裁决
              ↓
         Go 后端执行裁决 (approve / emergency lock / escalate)
```

### 三种裁决

| 裁决 | 条件 | 操作 |
|------|------|------|
| `approve` | 普通作业，请求 < 48h，资源充足 | 按请求时长锁定，工单状态 = 已批准 |
| `approve_emergency` | 作业即将被清理 (`reminded=true`)，请求 >= 48h | 立即锁定 6 小时，工单保持待处理状态交由管理员 |
| `escalate` | 请求 >= 48h，或资源存在问题 | 不锁定，工单保持待处理状态并附带智能体报告 |

### 紧急检测

智能体通过 `get_job_detail` 返回的 `reminded` 字段检测紧急状态：

- `reminded = true` -> 作业已收到清理提醒，将在 24 小时内被删除
- 此情况下，即使请求时长较长（>= 48h），智能体也会立即锁定 6 小时以防止作业被终止，同时将剩余时长转交管理员处理

### 时长调整

智能体可以批准比请求更短的时长：

- `approved_hours = null` -> 使用用户原始请求的时长
- `approved_hours = N` -> 锁定 N 小时（只允许缩短，不允许延长）

Go 后端强制执行：`approved_hours` 必须为正数且小于原始请求时长。

---

## 工具白名单

智能体可以使用 7 个只读工具（系统共有 88 个工具）：

| 工具 | 用途 |
|------|------|
| `get_job_detail` | 作业类型、资源、运行时长、`reminded` 状态 |
| `get_job_events` | 调度和重启事件 |
| `query_job_metrics` | GPU 利用率趋势（交互式作业跳过） |
| `check_quota` | 用户配额使用情况 |
| `get_realtime_capacity` | 集群资源可用性 |
| `list_cluster_jobs` | 用户的其他运行中作业（总资源占用） |
| `get_approval_history` | 近期审批频率 |

工具 schema 开销：约 500 tokens（完整工具集约 8000 tokens）。

---

## 评估逻辑

### 决策流程

```
get_job_detail → 检查作业类型、资源、reminded 状态
  ↓
reminded=true ?
  ├─ 是（紧急通道）
  │   请求 < 48h → 批准
  │   请求 >= 48h → approve_emergency (6h) + 转交剩余部分
  │
  └─ 否（正常通道）
      请求 < 48h → 检查资源
      │   配额充足，无排队压力 → 批准
      │   资源存在问题 → 转交
      请求 >= 48h → 附带分析转交
```

### 转交触发条件（覆盖任何通道）

即使是短期请求（< 48h），在以下情况下智能体也会转交：

- 用户已占用大量高端 GPU 资源
- 相同资源类型有大量排队积压
- 批处理作业 GPU 利用率持续接近零
- 用户在过去 7 天内已请求 3 次以上审批
- 用户超出其配额分配

### 交互式作业处理

Jupyter 和 WebIDE 会话享受宽松处理：

- GPU 利用率不作为主要判断依据（用户可能正在配置环境、调试或运行 CPU 密集型代码）
- 决策基于：会话时长、审批频率、排队压力

---

## 回退策略

```
ReAct 循环 (最多 15 次工具调用，全局设置)
  ↓
智能体自然输出裁决 JSON？
  ├─ 是 → 使用该结果
  └─ 否（图达到工具限制 → summarize node）
      ↓
  从 summarize 输出中提取裁决？
  ├─ 是 → 使用该结果
  └─ 否 → 回退：BaseRoleAgent.run_json() 不使用工具
      ↓
  JSON 修复成功？
  ├─ 是 → 使用该结果
  └─ 否 → 关键词检测 ("通过"/"转交")
      ↓
  找到关键词？
  ├─ 是 → 推断裁决，confidence=0.5
  └─ 否 → 默认：escalate，confidence=0.3
```

智能体**永远不会向调用方抛出异常**。所有失败都会产生安全的 `escalate` 裁决。

---

## 限流（Go 后端侧）

三层保护机制防止智能体被过载：

| 层级 | 机制 | 默认值 |
|------|------|--------|
| 速率限制 | 每分钟令牌桶 | 10/分钟 |
| 并发限制 | 基于 channel 的信号量 | 3 并发 |
| 熔断器 | 连续失败计数 | 5 次后熔断，冷却 60 秒 |

当任一层级拒绝请求时，工单回退至人工审核。不会丢失任何请求。

---

## 审计追踪

每次评估都会被记录：

1. **AgentReport 字段**：存储在 ApprovalOrder 上（JSON 格式，包含裁决、置信度、原因、工具轨迹）
2. **ReviewSource 字段**：`system_auto` / `agent_auto` / `admin_manual`
3. **OperationLog 条目**：operator=agent, type=approval_evaluation

前端在审批详情页的专用标签页中展示智能体报告。

---

## 代码

| 组件 | 文件 |
|------|------|
| 智能体类 | `crater_agent/agents/approval.py` |
| FastAPI 端点 | `crater_agent/app.py` (`POST /evaluate/approval`) |
| Go 评估服务 | `backend/internal/service/agent_approval.go` |
| Go 钩子 | `backend/internal/handler/approvalorder.go` (CreateApprovalOrder) |
| 工具处理器 | `backend/internal/handler/agent/tools_readonly.go` (toolGetApprovalHistory) |
| 数据库模型 | `backend/dao/model/approvalorder.go` (ReviewSource, AgentReport) |
| 配置 | `backend/pkg/config/config.go` (Agent.ApprovalHook) |
| 数据库迁移 | `backend/hack/sql/20260418_approval_agent.sql` |
| 前端类型 | `frontend/src/services/api/approvalorder.ts` |
| 前端徽章 | `frontend/src/components/badge/approvalorder-badge.tsx` (ReviewSourceBadge) |
| 前端详情标签页 | `frontend/src/routes/admin/more/orders/$id.tsx` (Agent 标签页) |
| 前端列表列 | `frontend/src/components/approval-order/approval-order-data-table.tsx` |
| 规格文档 | `docs/specs/agent-approval-hook.md` |
