# 第 4 章 面向智算平台的智能运维系统实现

本章给出 Mops 系统在 Crater 智算平台上的工程实现，是第 3 章设计的"工程落地版本"。4.1 节给出三服务架构与部署形态，4.2 节给出后端能力与数据持久化模型（含 6 张新增表 + 关键 API 路由），4.3 节给出 Agent Runtime 内部的工程实现（含核心算法），4.4 节给出质量评估模块。

## 4.1 系统整体架构

Mops 由三个独立的服务组成，统一部署在 Crater 同一 Kubernetes 集群中，如图 4-1 所示：

**(1) Crater Frontend（前端）**：基于 React 18 + TypeScript 实现，副本数 2 + HPA(2→5)。提供 Portal（用户工作台）与 Admin Console（管理后台）两套界面，内嵌对话抽屉、执行时间线、确认卡片与审计页面。前端通过 EventSource API 与后端建立 SSE 连接，实时渲染 Agent 的思考、工具调用、消息与暂停事件。

**(2) Crater Backend（后端）**：基于 Go 1.22 + Gin + GORM 实现，副本数 3 + HPA(3→10)。是整个系统的"安全网关"，负责身份认证、权限边界判断、工具目录维护、确认流编排、SSE 事件中继与审计落库。后端不调用 LLM，但负责把所有 LLM 想要执行的工具调用转化为对 Kubernetes API、Prometheus、PostgreSQL、Harbor 等下游系统的实际访问。

**(3) Agent Runtime（智能体服务）**：基于 Python 3.11 + FastAPI + LangChain/LangGraph 实现，副本数 2 + HPA(2→8)。承担 LLM 推理与多智能体编排。通过 `/chat` 端点（SSE 流式输出）接收后端转发的用户请求，通过 `/v1/agent/tools/execute` 端点回调后端执行工具，通过 `/eval/quality/*` 端点完成质量评测，通过 `/pipeline/*` 端点触发定时巡检。

三服务均以容器镜像形式部署，由 Crater 主仓库 Helm chart 统一发布管理。Runtime 与后端通过 ClusterIP Service 通信，前端通过 Ingress 暴露 HTTPS 端点。Runtime 与外部 LLM 服务走 Egress Gateway 统一鉴权与限流。

部署形态遵循 NFR-1/NFR-4 两条非功能性需求：(i) Runtime **不持有 K8s 凭据**，所有平台访问经 Backend 鉴权；(ii) Runtime 是无状态服务，高风险操作暂停期间通过 workflow checkpoint 序列化到数据库，恢复请求可被任意副本处理，实现水平弹性。

## 4.2 前后端实现

### 4.2.1 数据持久化 ER 模型

Mops 在 PostgreSQL 中新增 7 张表（其中 6 张核心 + 1 张评测），与 Crater 原有的 `users`、`jobs`、`accounts`、`alerts` 等表共存。ER 模型完整图示见图 4-3，主要关系简述如下：

**核心表**（共 7 张）：

- `agent_sessions`（会话）：`id` (uuid) · `user_id` · `account_id` · `title` · `source` · `orchestration` · `status`
- `agent_turns`（轮次，含 checkpoint）：`id` · `session_id` · `turn_id` · `user_message` · `final_answer` · `page_context` · `continuation` · `workflow` (jsonb · checkpoint v5) · `usage_summary` · `status`
- `agent_messages`（消息历史）：`id` · `turn_id` · `role` · `content` · `tool_calls` · `agent_role`
- `agent_audit`（**运维操作审计核心表**）：`id` · `session_id` · `turn_id` · `actor_id` · `actor_role` · `tool_name` · `tool_args` (jsonb) · `result` (jsonb) · `error` · `latency_ms` · `risk_level` · `permission` · `confirmation_id` · `tool_call_id` · `agent_id` · `created_at`
- `agent_confirmations`（高风险操作确认）：`id` · `session_id` · `turn_id` · `action_id` · `tool_name` · `action_payload` · `checkpoint` (jsonb) · `risk_level` · `status` · `confirmed_by` · `confirmed_at` · `reject_reason` · `expires_at`
- `agent_inspection_runs`（巡检任务）：`id` · `scope` · `cron_expression` · `triggered_by` · `target_count` · `summary` (jsonb) · `report_url` · `tokens_used` · `status`
- `agent_quality_eval`（质量评分回流）：`id` · `session_id` · `turn_id` · `user_id` · `eval_type` · `chat_judge_score` · `chain_judge_score` · `tool_f1` · `root_cause_hit` · `permission_compliance` · `user_feedback` · `model_used`

**主要索引设计**（保障运维查询性能）：

- `agent_audit`：复合索引 `(session_id, turn_id)`、`(actor_id, created_at desc)`、`(tool_name, created_at) WHERE error IS NOT NULL`；JSONB GIN 索引 `tool_args`，支持 "上周哪些工具失败率最高" 等运维分析查询。
- `agent_confirmations`：部分索引 `(status, expires_at) WHERE status='pending'`，加速待确认任务扫描。
- `agent_inspection_runs`：`(scope, started_at desc)` 支撑巡检列表页。

外键约束在应用层管理（GORM tag），不在数据库层强制，以便审计表能在会话被软删除后继续保留——这一设计满足合规审计"不可篡改"的要求。

### 4.2.2 后端能力与 SSE 流式实现

后端在 `/api/v1/` 下提供三组 Agent 相关路由（详见图 4-1 中部署架构面板）：对话路由、审计路由、巡检路由。其能力与对应页面如表 4-1 所示。

**表 4-1 Crater Backend 提供的 Agent 相关能力**

| 模块 | 路由 | 功能 | 对应页面 |
|------|------|------|----------|
| Chat | `POST /v1/chat` (SSE) | 启动 Agent 会话 | Portal 对话抽屉 |
| Confirm | `GET /v1/confirmations/:id` | 拉取待确认动作 | 确认卡片 |
| Confirm | `POST /v1/confirmations/:id` | 提交确认结果 | 确认卡片 |
| Audit | `GET /v1/audit/sessions` | 列出会话 | Admin 在线审计页 |
| Audit | `GET /v1/audit/turns/:id` | 查看单轮详细轨迹 | Admin 在线审计页 |
| Inspection | `GET /v1/inspection/runs` | 列出巡检任务 | Admin 智能巡检页 |
| Inspection | `GET /v1/inspection/runs/:id` | 查看巡检报告 | Admin 智能巡检页 |
| Tools (内部) | `POST /v1/agent/tools/execute` | Runtime 回调执行工具 | — |

**SSE 事件流式实现**：用户发送消息时，前端通过 EventSource 订阅 `/v1/chat`，后端立刻建立 HTTP 连接到 Runtime `/chat`，并把上游流式响应原样转发给前端，同时拷贝一份到审计表。SSE 事件类型包括 `thinking`、`tool_call`、`tool_result`、`message`、`pause`、`done`、`error`。后端 SSE 转发与审计的核心逻辑见算法 4.1：

```latex
\begin{algorithm}
\caption{Backend SSE 事件流式路由与审计中继}
\label{alg:sse_router}
\begin{algorithmic}[1]
\Function{StreamChat}{$\mathrm{req}, \mathrm{w}$}
  \State $\mathrm{ctx} \gets \mathrm{Authenticate}(\mathrm{req.headers})$ \Comment{JWT 解析 actor.role}
  \State $\mathrm{ctx.page} \gets \mathrm{req.page\_context}$
  \State $\mathrm{ctx.capabilities} \gets \mathrm{SanitizeTools}(\mathrm{ctx}, \mathrm{ToolCatalog})$
  \State $\mathrm{turn\_id} \gets \mathrm{NewTurnId}()$
  \State $\mathrm{PersistTurnStart}(\mathrm{ctx}, \mathrm{turn\_id})$
  \State $\mathrm{rb} \gets \mathrm{NewRingBuffer}(512\mathrm{KB})$ \Comment{异步审计缓冲}
  \State $\mathrm{upstream} \gets \mathrm{HttpPostStream}(\mathrm{Runtime.url}/\texttt{chat}, \mathrm{ctx})$
  \ForAll{$\mathrm{event} \in \mathrm{upstream}$}
    \State $\mathrm{w.write}(\mathrm{SerializeSSE}(\mathrm{event}))$ \Comment{转发给前端}
    \State $\mathrm{rb.append}(\mathrm{event})$
    \If{$\mathrm{rb.full}()$ \textbf{or} $\mathrm{event.type} \in \{\texttt{tool\_result}, \texttt{pause}, \texttt{done}\}$}
      \State $\mathrm{Persist}(\mathrm{agent\_audit}, \mathrm{rb.flush}())$ \Comment{异步批写}
    \EndIf
    \If{$\mathrm{event.type} = \texttt{pause}$}
      \State $\mathrm{Persist}(\mathrm{agent\_confirmations}, \mathrm{event.payload})$
    \EndIf
  \EndFor
  \State $\mathrm{UpdateTurnStatus}(\mathrm{turn\_id}, \texttt{completed})$
\EndFunction
\end{algorithmic}
\end{algorithm}
```

后端使用 ring buffer 异步批量写审计，避免高频写入压垮数据库；保证在 `done` 事件到达前完成审计落库；若失败则给前端推 `error` 事件并标记 `agent_turns.status='audit_failed'`。完整的 SSE + 确认续接时序见图 4-4。

### 4.2.3 前端页面要点

Portal 侧的对话抽屉布局如下：左侧 70% 是平台原生页面（作业列表、详情、Notebook），右侧 30% 是 Agent 对话抽屉，可被收起。抽屉内自顶向下依次为：当前页面上下文徽章（"作业 jobX | 节点 nodeA"）、执行时间线（thinking/tool_call/tool_result 折叠显示）、消息气泡区、输入框。用户在不同页面切换时，抽屉中的徽章自动更新；新建对话则保留页面上下文作为隐式参数。

Admin Console 侧新增两个页面：**在线审计页**（按时间倒序展示会话列表，可点入查看完整推理轨迹，支持按工具、按错误码、按用户检索）、**智能巡检页**（按时间展示每小时巡检报告，支持下钻查看异常作业与建议）。

## 4.3 智能体服务实现

### 4.3.1 记忆与上下文实现

Runtime 的记忆能力由 `memory/session.py` 模块承担。`build_history_messages(history, max_tokens=4000)` 把后端传来的对话历史按时间顺序转换为 LangChain `HumanMessage` / `AIMessage` / `ToolMessage` 序列；当总 Token 超出预算时，采用"头部 + 尾部 + 中间摘要"的截断策略：保留最近 3 轮完整对话，更早的对话以"用户问 X，Agent 答 Y"摘要替代；工具结果在历史中按"头 1 200 字符 + 尾 1 200 字符"截断（错误结果留更多空间至 1 600 字符）。这一策略避免单条超长 PromQL 结果挤占多轮上下文。

页面上下文与用户上下文不进入历史消息序列，而是作为系统提示词的字段直接注入：每个 Agent 的系统提示词中都包含 `<actor_role>`、`<page_context>` 两个字段，由 `MASState.goal` 渲染得到。

### 4.3.2 多智能体编排与任务状态实现

Runtime 在启动时同时实例化 `SingleAgentOrchestrator`（ReAct 实现）与 `MultiAgentOrchestrator`（MAS 实现），并在 `app.py` 中通过 `get_orchestration_mode(context)` 根据请求上下文选择编排器。MAS 主循环位于 `orchestrators/multi.py`（约 5 300 行），其逻辑骨架与算法 3.1 一致。每个 Agent 都继承 `BaseRoleAgent`（`agents/base.py`），其 `__call__(state: MASState) → MASState` 完成三件事：(i) 调用 `BuildView()` 构造该角色的 StateView；(ii) 渲染提示词并调用 LLM（带 `tenacity` 重试与超时）；(iii) 解析 LLM 输出（结构化 JSON），把结果写回 MASState 对应字段。

意图路由实现（`orchestrators/intent_router.py`）：IntentRouter 自身也是一个 LLM Agent，但使用更小、更快的模型（如 Qwen 3 Flash），输出固定 JSON `{entry_mode, operation_mode, complexity, targets}`。entry_mode 决定走 Guide / Single-ReAct / MAS 哪一条路径。

Plan-Execute 与 ReAct 实现：`orchestrators/plan_execute.py` 与 `orchestrators/single.py` 分别提供另两种基线编排，便于第 5 章对比实验。三种编排器共享同一套工具与 MASState 的部分字段，保证比较的公平性。

### 4.3.3 工具与安全模块实现

**工具声明解析**：所有工具以 `@tool` 装饰器声明（`tools/definitions.py`），装饰器额外接受 `permission`、`confirm`、`risk` 三个元字段。装饰器在装载时把工具元信息注册到全局 registry 并自动生成 LangChain `BaseTool` 子类。Runtime 在每次请求构造 `enabled_tools` 列表时，会从 registry 中按 actor role 与 page 过滤（即算法 3.4）。

**工具执行**：通过 `GoBackendToolExecutor.execute(tool_name, tool_args, context)` 完成（算法 4.2）。其要点为同步 HTTPS 调用后端 + 三态返回处理 + 自动审计：

```latex
\begin{algorithm}
\caption{GoBackendToolExecutor 工具执行与审计}
\label{alg:tool_exec}
\begin{algorithmic}[1]
\Function{Execute}{$\mathrm{tool}, \mathrm{args}, \mathrm{ctx}$}
  \State $t_0 \gets \mathrm{Now}()$
  \State $\mathrm{req} \gets \{\mathrm{tool}, \mathrm{args}, \mathrm{session\_id}, \mathrm{turn\_id}, \mathrm{agent\_role}\}$
  \State $\mathrm{resp} \gets \mathrm{HttpPost}(\mathrm{Backend.url}/\texttt{v1/agent/tools/execute}, \mathrm{req}, \mathrm{ctx.jwt})$
  \State $\mathrm{latency} \gets \mathrm{Now}() - t_0$
  \If{$\mathrm{resp.status} = \texttt{success}$}
    \State \Return $\mathrm{ToolResult}(\mathrm{ok}, \mathrm{resp.result}, \mathrm{latency})$
  \ElsIf{$\mathrm{resp.status} = \texttt{confirmation\_required}$}
    \State $\mathrm{action} \gets \mathrm{NewAction}(\mathrm{confirm\_id}=\mathrm{resp.confirm\_id}, \texttt{awaiting\_confirmation})$
    \State \Return $\mathrm{ToolResult}(\mathrm{paused}, \mathrm{action}, \mathrm{latency})$ \Comment{由 MAS 主循环挂起}
  \ElsIf{$\mathrm{resp.status} = \texttt{permission\_denied}$}
    \State \Return $\mathrm{ToolResult}(\mathrm{error}, \texttt{permission\_denied}, \mathrm{latency})$ \Comment{触发 Verifier retry?}
  \Else
    \State \Return $\mathrm{ToolResult}(\mathrm{error}, \mathrm{resp.error}, \mathrm{latency})$
  \EndIf
\EndFunction
\end{algorithmic}
\end{algorithm}
```

注意：审计落库由 Backend 在执行工具时同步完成（在算法 4.1 的 ring-buffer 中），Runtime 无需重复写入；这是为了保证审计真实性——如果 Runtime 自己写审计，恶意 Runtime 可能伪造记录。

跨平台适配通过抽象 `ToolExecutor` 接口实现，目前实现三类：`GoBackendToolExecutor`（生产）、`LocalToolExecutor`（开发与离线评测）、`MockToolExecutor`（评测快照回放）。迁移到新平台时仅需实现一个新的 Executor 子类并在启动时注入。

### 4.3.4 巡检流水线实现

`pipeline/inspection.py` 通过 cron 调度，每小时触发一次巡检任务。其编排逻辑见算法 4.4：

```latex
\begin{algorithm}
\caption{巡检定时任务编排}
\label{alg:inspection}
\begin{algorithmic}[1]
\Function{ScheduledInspection}{$\mathrm{scope}, \mathrm{window}$}
  \State $\mathrm{run\_id} \gets \mathrm{PersistInspectionStart}(\mathrm{scope})$
  \State $\mathrm{targets} \gets \mathrm{QueryTargets}(\mathrm{scope}, \mathrm{window})$ \Comment{失败作业 / 空闲 GPU / 长时 Jupyter}
  \State $\mathrm{summary} \gets [\,]$
  \ForAll{$t \in \mathrm{targets}$}
    \State $\mathrm{ctx} \gets \mathrm{BuildAdminContext}(t)$
    \State $\mathrm{result} \gets \mathrm{MasOrchestrator.run}(\mathrm{prompt}=\mathrm{InspectPrompt}(t), \mathrm{ctx})$
    \State $\mathrm{summary.append}(\mathrm{Extract}(\mathrm{result}))$
  \EndFor
  \State $\mathrm{report} \gets \mathrm{Aggregate}(\mathrm{summary})$
  \State $\mathrm{PersistInspectionEnd}(\mathrm{run\_id}, \mathrm{report})$
  \State $\mathrm{NotifyAdmin}(\mathrm{report.url})$
\EndFunction
\end{algorithmic}
\end{algorithm}
```

巡检复用 MAS 的核心能力（Planner、Explorer、Executor），只在 Coordinator 提示词中替换"用户请求"为"巡检任务"，保证一套代码同时支持交互式与批处理两种场景。

## 4.4 质量评估模块

### 4.4.1 评测数据集构造

`crater_agent/dataset/` 目录组织如下：

```text
dataset/
├── raw/             # 原始 API 快照
├── real_scenarios/  # 85 个评测场景 JSON
├── sql/             # 数据采集 SQL
└── transform.py     # 快照 → 场景脚本
```

每个场景以 JSON 描述：`scenario_id`、`category`（diagnosis/ops/query/submission）、`subcategory`、`difficulty`（1-5）、`turn_count`、`ground_truth.expected_tools_must`、`ground_truth.expected_confirmation_tools`、`evaluation.score_profile`、`evaluation.difficulty_weights` 等字段。

**数据采集**：`collect_api.sh` 与 `collect_db.sh` 分别从 Kubernetes、PostgreSQL 采集真实快照；`transform.py` 把快照转换为 MockToolExecutor 可回放的格式。**标注流程**：(1) 对每个真实场景，由作者手工编写期望工具序列 (must / nice-to-have) 与最终答复关键要素；(2) 通过 3-5 次 Mops 试跑获得初步答案；(3) 由两名研究人员盲审打分，分歧场景采用第三人裁决；(4) 最终发布到 `real_scenarios/`。

### 4.4.2 评分 Profile 与评测执行器

评分系统位于 `eval/scoring.py`，采用六维加权评分。其核心计算如算法 4.3 所示：

```latex
\begin{algorithm}
\caption{六维加权评分（按难度归一化）}
\label{alg:scoring}
\begin{algorithmic}[1]
\Function{ComputeScore}{$\mathrm{scenario}, \mathrm{trace}$}
  \State $f_1 \gets \mathrm{ToolF1}(\mathrm{scenario.gt.tools}, \mathrm{trace.tool\_calls})$ \Comment{AST 匹配}
  \State $f_2 \gets \mathrm{RootCauseHit}(\mathrm{scenario.gt.keywords}, \mathrm{trace.final\_answer})$
  \State $f_3 \gets \mathrm{ChatJudge}(\mathrm{scenario.gt.answer\_keys}, \mathrm{trace.final\_answer})$
  \State $f_4 \gets \mathrm{PermissionCompliance}(\mathrm{trace})$ \Comment{违规为 0}
  \State $f_5 \gets \mathrm{Efficiency}(|\mathrm{scenario.gt.tools}|, |\mathrm{trace.unique\_tool\_calls}|)$
  \State $f_6 \gets \mathrm{MultiTurnPass}(\mathrm{scenario}, \mathrm{trace})$ \Comment{含确认场景}
  \State $\mathbf{w} \gets \mathrm{scenario.score\_profile.weights}$
  \State $\mathrm{base} \gets \sum_{i=1}^{6} w_i \cdot f_i$
  \State $\mathrm{score} \gets \mathrm{base} \cdot \mathrm{scenario.difficulty\_weight} \cdot 100$
  \If{$f_4 = 0$} \Return $0$ \Comment{权限违规一票否决}
  \EndIf
  \State \Return $\mathrm{Clamp}(\mathrm{score}, 0, 100)$
\EndFunction
\end{algorithmic}
\end{algorithm}
```

评测执行器位于 `eval/runner.py`，支持顺序与并行两种执行模式（`run_benchmark` / `run_benchmark_parallel`），每个场景执行完毕后写入 `offline-bench-report.{mode}.{name}.scenarios.csv` 与 `.turns.csv` 详细日志，并通过 `eval/judge.py` 调用 LLM-as-Judge 完成定性评分。第 5 章实验全部基于该执行器产出的数据。

**质量反馈闭环**：`POST /eval/quality/feedback` 接受 Portal 中用户对 Agent 答复的"👍/👎"反馈，触发 `quality/analyzer.py` 中 ChatJudge（评对话质量） + ChainJudge（评工具与任务完成度）并行评分；评分结果写入 `agent_quality_eval` 表形成质量证据库，与 5.4 节消融实验互相印证。

## 4.5 本章小结

本章给出了 Mops 系统在 Crater 智算平台上的工程实现：4.1 节确立"前端 / 后端 / Runtime"三服务架构，统一部署在 Crater 同一 Kubernetes 集群；4.2 节给出后端数据持久化的 7 张表与三组路由能力，并通过算法 4.1 描述 SSE 流式输出、确认续接、审计持久化的具体机制；4.3 节阐述 Runtime 内部的记忆与上下文实现、多智能体编排状态机、工具执行（算法 4.2）与巡检流水线（算法 4.4）；4.4 节描述评测数据集构造与六维加权评分（算法 4.3）。下一章将基于本实现进行 85 场景离线评测与线上案例分析。
