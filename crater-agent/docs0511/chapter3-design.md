# 第 3 章 面向智算平台的多智能体协作框架设计

本章给出 Mops 多智能体协作框架的系统设计，是全文最核心的内容。3.1 节抽象智算平台运维任务并展开需求分析，3.2 节给出框架总体分层架构，3.3 节定义多智能体角色与含 replan/reassign 的协作机制，3.4 节阐述记忆与上下文管理，3.5 节阐述跨平台工具体系，3.6 节阐述安全控制与审计机制，3.7 节进行本章小结。

## 3.1 智算平台运维任务建模与需求分析

### 3.1.1 运维对象与用户角色

如 2.1.2 节所述，智算平台的运维对象可由"资源-作业-用户"三层与"提交-调度-执行-终止-审计"五阶段交织而成。形式化地，本文将运维上下文 $C$ 建模为三元组：

$$C = (\mathrm{actor}, \mathrm{page}, \mathrm{capabilities})$$

其中：
- $\mathrm{actor} = (\mathrm{user\_id}, \mathrm{role}, \mathrm{account\_id})$ 描述发起者身份与组织归属。Crater 平台目前定义 `user` 和 `admin` 两种角色，分别对应面向最终研究人员的 Portal 与面向平台管理员的 Admin Console。
- $\mathrm{page} = (\mathrm{route}, \mathrm{job\_name}, \mathrm{node\_name}, \mathrm{pvc\_name})$ 描述当前会话所处的页面位置与关联对象。例如 `route="/jobs/my_jupyter_001"` 表示用户正在查看名为 `my_jupyter_001` 的 Jupyter 作业详情页。
- $\mathrm{capabilities} = \{T_i\}$ 表示当前会话可见的工具集合，由后端基于 actor 与 page 联合裁剪而成。

### 3.1.2 任务类型、风险等级与数据依赖

本文将面向智算平台的运维任务划分为四类，每一类的特征与典型示例如表 3-1 所示。

**表 3-1 智算平台运维任务分类与特征**

| 类别 | 典型问题示例 | 数据依赖 | 工具风险等级 | 多轮交互需求 |
|------|--------------|----------|--------------|--------------|
| 诊断 Diagnosis | "为什么 jobX 运行 30 秒就 OOM 了？" | 作业元数据 + Pod 事件 + 容器日志 + GPU 指标 | 只读 | 可能多轮（追问） |
| 运维 Ops | "把队列里超过 24 小时的 Jupyter 关掉" | 全平台作业列表 + 用户配置 | **写（需确认）** | 通常多轮（确认） |
| 查询 Query | "我账户目前空余多少 GPU 卡时？" | 配额 + 用量统计 | 只读 | 单轮居多 |
| 作业提交 Submission | "帮我提一个 PyTorch 单机 2 卡的训练任务" | 镜像 + 资源 + 用户偏好 | **写（需确认）** | 多轮（参数补全） |

每类任务还附加一个**风险等级**字段：`low`（仅查询）、`medium`（修改用户自有资源）、`high`（影响他人或不可逆）。这一字段直接决定 3.6 节的确认流程是否触发。

### 3.1.3 需求分析

通过对 Crater 平台 16 个月真实运行数据的归纳（详见 1.1.1 节量化数据），并结合用户问询案例分析，本文识别出 Mops 系统需要满足的功能性需求（FR）与非功能性需求（NFR）：

**功能性需求**：
- **FR-1 自然语言入口**：以对话方式承接用户与管理员两类角色的运维问询，自动绑定页面上下文。
- **FR-2 多源证据整合**：覆盖 Kubernetes 事件、Volcano 队列、Prometheus 指标、容器日志、Harbor 镜像、PostgreSQL 业务库等至少 6 类数据源。
- **FR-3 多步任务编排**：支持单一问题触发多达 8 轮、25 次工具调用的长链协作。
- **FR-4 高风险操作确认**：写入类工具（重启、关停、调整配额等）必须经过用户显式确认。
- **FR-5 续接已暂停任务**：确认完成后能从原任务流恢复，不丢失中间证据。
- **FR-6 定时巡检**：每小时扫描失败作业、低利用率作业、长时间空闲资源并生成分析报告。
- **FR-7 全链路审计**：所有运维操作（含被拒绝的请求）均记录并可回放。
- **FR-8 评测反馈闭环**：通过用户反馈（👍/👎）与离线评测数据集驱动系统迭代。

**非功能性需求**：
- **NFR-1 安全可控**：零信任设计，LLM 输出不可绕过权限边界。
- **NFR-2 跨平台可迁移**：框架不绑死 Crater，迁移到不同智算平台仅需替换工具适配层。
- **NFR-3 成本可控**：单轮 Token < 5 000，wall-clock 时延 < 60 秒（90 百分位）。
- **NFR-4 水平可扩展**：Runtime 必须是无状态服务，可在 K8s 上 HPA 弹性扩缩容。
- **NFR-5 工程可演进**：模块边界清晰，新增 Agent / 工具不影响既有代码。
- **NFR-6 可观测**：Runtime 输出 OpenTelemetry traces，便于线上排障与成本分析。

### 3.1.4 任务建模的数据来源

任务建模并非凭空抽象，而是基于 Crater 平台 16 个月真实运行数据归纳得到：8 957 条失败作业（exit code 1: 54%, exit code 137: 27%, exit code 255: 6%, exit code 127: 4%）、7 248 条 FailedMount 与 3 677 条 FailedScheduling 事件、6.8 万条告警（GPU 低利用率占 56%）、9.4 万次作业提交（交互式 Jupyter 60.08%、自定义脚本 39.67%、PyTorch DDP 0.23%）。通过对真实问询样本的归类，本文最终凝练出 85 个典型场景作为评测基础（详见第 5 章）。

## 3.2 Mops 框架总体架构设计

### 3.2.1 设计目标

Mops 的设计目标可以归纳为以下六个非功能性约束（与 3.1.3 节 NFR 对应）：**(D1) 任务覆盖完备**、**(D2) 多轮交互稳健**、**(D3) 安全可控**、**(D4) 跨平台可迁移**、**(D5) 工程可演进**、**(D6) 成本可控**。这六个目标既约束本章设计，也是第 5 章评测指标体系的来源。

### 3.2.2 分层架构

Mops 的总体分层架构如图 3-1 所示，由四层组成：

**① 用户与接入层**：嵌入 Crater Portal 与 Admin Console 的对话抽屉、执行时间线、确认卡片与审计页。用户输入自然语言问题，并通过 `page_context` 字段向后端上报 `route` 与目标对象。

**② 业务网关层（Crater Backend · Go）**：负责 JWT 鉴权、权限边界判断、工具目录维护、确认流编排、SSE 事件中继与审计落库。后端是 Agent Runtime 与平台 API 之间的"安全网关"，所有平台访问最终经由后端统一调度。

**③ Agent Runtime（Python · FastAPI · LangChain/LangGraph）**：本课题的核心。内部按职责进一步分为意图路由、编排器选择、任务型 Agent、多智能体核心、工具适配/记忆/评测五大子系统。

**④ 智算平台与基础设施层**：Kubernetes + Volcano、Prometheus + DCGM、PostgreSQL、Harbor、NFS/Ceph、LDAP/SMTP 以及外部 LLM 服务。

四层之间通过明确的接口边界解耦。Runtime **不持有 K8s 凭据**，所有平台访问均经由 Backend 转发——这是 NFR-1 安全可控的工程实现基础。

## 3.3 多智能体角色设计与协作机制

### 3.3.1 智能体角色定义

Mops 多智能体角色分工与协作关系如图 3-2 所示，包含五个核心角色与四个任务型角色。

**核心角色（参与 MAS 主循环）：**

- **Coordinator（协调器）**：作为 MAS 循环的中央控制器，负责调度其余四个角色、判断终止条件、产出最终答复。其状态机详见图 3-4。
- **Planner（规划器）**：根据 Goal 与可用 capabilities 生成 PlanArtifact。Planner 不直接调用工具，避免角色越界。
- **Explorer（探索器）**：根据 Plan 与已有 Observation，调用只读工具迭代采集证据。
- **Executor（执行器）**：根据 Plan 与 Observation 生成写操作，并通过 Tool Executor 触发执行；高风险操作进入 `awaiting_confirmation` 状态。
- **Verifier（验证器）**：对 Execution 结果与 Observation 复核，输出 `pass | missing_evidence | risk | retry` 四类裁决——后三类触发回退，是 Mops 多智能体相对单 Agent 的关键优势来源。

**任务型角色（独立于 MAS 主循环）**：Guide（帮助）、Approval（审批，仅 7 个白名单工具）、Quota（配额分析）、Inspection（cron 巡检）。

### 3.3.2 多智能体协作机制设计

**协作模式选型**：Mops 采用 **工件共享 + 阶段状态机** 的混合模式（图 2-4 的 (d) 范式）。Coordinator 与 Planner/Explorer/Executor/Verifier 通过 MASState 共享结构化字段（而非自然语言消息），避免每轮重复"解释上下文"。

**阶段状态机**：Coordinator 维护一个显式状态机：`INIT → ROUTE → PLAN → EXPLORE → EXECUTE → (PAUSE → RESUME) → VERIFY → SUMMARIZE → END`，每一个状态由 Coordinator 的提示词描述并由 `next_action` 字段驱动。状态机完整转移图（含 replan / reassign / retry / HITL）见图 3-4。其核心循环以算法 3.1 给出：

```latex
\begin{algorithm}
\caption{MAS 主循环（Coordinator 状态机驱动）}
\label{alg:mas_loop}
\begin{algorithmic}[1]
\State \textbf{Input:} GoalArtifact $g$, Capabilities $\mathcal{T}$, RuntimeConfig $\theta$
\State \textbf{State:} MASState $s \gets \mathrm{Init}(g, \mathcal{T})$
\State $r \gets 0$ \Comment{loop\_round}
\While{$r < \theta.\mathrm{lead\_max\_rounds}$ \textbf{and} $s.\mathrm{stop\_reason} = \emptyset$}
  \State $\mathrm{stage} \gets \mathrm{Coordinator.next\_action}(s)$
  \If{$\mathrm{stage} = \texttt{PLAN}$}
    \State $s.\mathrm{plan} \gets \mathrm{Planner}(s.\mathrm{goal}, \mathcal{T})$
  \ElsIf{$\mathrm{stage} = \texttt{EXPLORE}$}
    \State $s.\mathrm{observation} \gets \mathrm{Explorer}(s.\mathrm{plan}, s.\mathrm{observation})$
  \ElsIf{$\mathrm{stage} = \texttt{EXECUTE}$}
    \State $s.\mathrm{execution} \gets \mathrm{Executor}(s.\mathrm{plan}, s.\mathrm{observation})$
    \If{$\exists a \in s.\mathrm{execution.actions}: a.\mathrm{status} = \texttt{awaiting\_confirmation}$}
      \State $\mathrm{Emit}(\texttt{pause}, \mathrm{BuildCheckpoint}(s))$
      \State \textbf{return} $s$ \Comment{挂起等待外部 RESUME}
    \EndIf
  \ElsIf{$\mathrm{stage} = \texttt{VERIFY}$}
    \State $v \gets \mathrm{Verifier}(s)$
    \State $s \gets \mathrm{Dispatch}(s, v)$ \Comment{见算法 3.2}
    \If{$v.\mathrm{verdict} = \texttt{pass}$} \textbf{break} \EndIf
  \EndIf
  \If{$\mathrm{NoProgress}(s, \theta)$}
    \State $s.\mathrm{stop\_reason} \gets \texttt{no\_progress}$
  \EndIf
  \State $r \gets r + 1$
\EndWhile
\State \Return $\mathrm{Coordinator.summarize}(s)$
\end{algorithmic}
\end{algorithm}
```

**回退反馈机制（Mops 区别于 PS 的关键）**：Verifier 不只是"通过/不通过"的二值评分器，而是显式分派回退入口。当 verdict 为 `missing_evidence`/`risk`/`retry` 时，Coordinator 把控制权重新交回 Explorer/Planner/Executor。这一机制以算法 3.2 表达：

```latex
\begin{algorithm}
\caption{Verifier 三类裁决与回退分派}
\label{alg:verifier_dispatch}
\begin{algorithmic}[1]
\Function{Dispatch}{$s$, $v$}
  \If{$v.\mathrm{verdict} = \texttt{missing\_evidence}$ \textbf{and} $|s.\mathrm{observation.open\_questions}| > 0$}
    \State $s.\mathrm{plan}.\mathrm{stage\_complete} \gets \mathrm{True}$
    \State $s.\mathrm{observation}.\mathrm{stage\_complete} \gets \mathrm{False}$ \Comment{① 回退至 EXPLORE}
  \ElsIf{$v.\mathrm{verdict} = \texttt{risk}$}
    \State $s.\mathrm{execution}.\mathrm{actions} \gets [\,]$
    \State $s.\mathrm{plan} \gets \emptyset$ \Comment{② replan：清空计划}
  \ElsIf{$v.\mathrm{verdict} = \texttt{retry}$ \textbf{and} $\mathrm{IsRecoverable}(v.\mathrm{error})$}
    \For{$a \in s.\mathrm{execution}.\mathrm{actions}: a.\mathrm{status} = \texttt{error}$}
      \State $a.\mathrm{status} \gets \texttt{pending}$ \Comment{③ retry：保留指纹避免无限重试}
    \EndFor
  \ElsIf{\textbf{Coordinator 检测同一角色连续两轮 stage\_complete = False}}
    \State $\mathrm{ReassignRole}(s, \text{切换模型/降级到 Plan-Execute})$ \Comment{④ reassign}
  \EndIf
  \State \Return $s$
\EndFunction
\end{algorithmic}
\end{algorithm}
```

四种回退入口（① 补充探索、② replan、③ retry、④ 重新指派）共同构成 Mops 的"自纠错能力"。第 5.4 节消融实验显示：取消 Verifier 会让简单任务略微变快，但复杂诊断任务的根因命中率从 100% 跌至 91.5%——验证器是 Mops 多 Agent 优势的核心来源之一。

**终止与无进展守护**：长链 Agent 系统的两大风险是死循环与重复执行。Mops 通过三道守护：(i) 硬上界 `lead_max_rounds = 8`、`subagent_max_iterations = 25`；(ii) 无进展计数器 `no_progress_count`，若连续两轮 Observation 与 Action 未变则停止；(iii) 工具签名去重 `attempted_tool_signatures`，相同工具+参数已尝试过则禁止重复调用。

**意图路由与编排模式选择**：在进入 MAS 主循环前，IntentRouter 基于用户输入与 `page_context` 进行粗分类：纯文档帮助路由到 Guide；简单查询路由到 simple ReAct；多轮诊断/运维任务路由到 MAS。这种分层路由避免对所有问题都启动重量级 MAS 流水线，能在保留多 Agent 优势的同时显著降低简单问题的开销。

**审批与巡检流**：Approval Agent 在管理员触发"作业锁单评估"时被独立调度，与主对话流并行；Inspection Agent 通过 cron 调度，把批量分析结果写入审计库供管理员事后审阅。这两个角色不与主 MAS 共享 MASState，避免互相干扰。

## 3.4 记忆与上下文管理

### 3.4.1 多源异构数据与上下文建模

Mops 系统需要整合的数据来源至少包括六类（图 3-5）：**用户画像数据**、**会话上下文数据**、**页面上下文数据**、**任务状态数据**、**工具返回数据**、**平台元数据**。本文采用一个结构化状态对象 `MASState` 对这些异构数据进行统一建模，其字段如图 3-3 所示。

### 3.4.2 MASState：多智能体共享状态与 StateView

`MASState` 是一个 dataclass，承载一次多智能体协作的全部状态，分为五组：(a) 四阶段工件（Goal/Plan/Observation/Execution）；(b) 运行时控制（loop_round、no_progress_count、stop_reason、attempted_tool_signatures）；(c) 证据与审计（tool_records、action_history）；(d) 确认与恢复（pending_confirmations、resume_context、workflow checkpoint）；(e) 使用统计（usage_summary、llm_by_role）。详细字段见图 3-3。

每个角色看到的并不是完整的 MASState，而是经过 `StateView` 过滤后的子集（算法 3.3）：

```latex
\begin{algorithm}
\caption{StateView 按角色视图过滤}
\label{alg:state_view}
\begin{algorithmic}[1]
\Function{BuildView}{$s$, $\mathrm{role}$}
  \State $\mathcal{V} \gets \{\,\}$
  \If{$\mathrm{role} = \texttt{Planner}$}
    \State $\mathcal{V}.\mathrm{goal}, \mathcal{V}.\mathrm{capabilities} \gets s.\mathrm{goal}, s.\mathrm{capabilities}$
  \ElsIf{$\mathrm{role} = \texttt{Explorer}$}
    \State $\mathcal{V}.\mathrm{goal}, \mathcal{V}.\mathrm{plan}, \mathcal{V}.\mathrm{observation} \gets s.\mathrm{goal}, s.\mathrm{plan}, s.\mathrm{observation}$
  \ElsIf{$\mathrm{role} = \texttt{Executor}$}
    \State $\mathcal{V} \gets \mathrm{Project}(s, \{\mathrm{goal}, \mathrm{plan}, \mathrm{observation}, \mathrm{execution.actions}\})$
  \ElsIf{$\mathrm{role} = \texttt{Verifier}$}
    \State $\mathcal{V} \gets \mathrm{Project}(s, \{\mathrm{goal}, \mathrm{plan}, \mathrm{observation}, \mathrm{execution}\})$
  \ElsIf{$\mathrm{role} = \texttt{Coordinator}$}
    \State $\mathcal{V} \gets s$ \Comment{协调器看到全工件}
  \EndIf
  \State $\mathcal{V}.\mathrm{tool\_records} \gets \mathrm{TruncateByTokenBudget}(s.\mathrm{tool\_records}, \theta.\mathrm{budget}_{\mathrm{role}})$
  \State \Return $\mathcal{V}$
\EndFunction
\end{algorithmic}
\end{algorithm}
```

这种"按需视图"既减少 Token 占用（NFR-3 成本可控），也降低跨角色互相干扰的风险。

### 3.4.3 页面、会话与用户上下文

Mops 通过三层机制获取与维护上下文（图 3-5）：

**用户上下文（① User Context）**：来自 Crater 后端 JWT 鉴权信息，包含 `user_id`、`account_id`、`role`，作为**可信源**；任何来自前端或 LLM 输出的角色信息均不可覆盖此字段，防止"提示词注入"导致权限提升。整轮不变。

**页面上下文（② Page Context）**：前端在用户切换页面时把 `route`、`job_name`、`node_name`、`pvc_name` 通过 `page_context` 字段上报，作为隐式参数注入到 IntentRouter 与 Planner 中。例如用户在"作业详情页"问"为什么失败"，page_context 自动注入对应的 `job_name`，无需用户重复说明。每请求更新。

**会话上下文（③ Session Context）**：以 PostgreSQL `agent_sessions` / `agent_turns` 表为持久层。Runtime 在每次请求时只从后端取最近 N 轮历史（默认按 4 000 Token 预算）；工具结果在历史中以"头部 + 尾部 + 省略号"方式截断，避免单条超长结果挤占多轮上下文。跨轮持久。

### 3.4.4 多轮状态与确认恢复机制

当任务跨多轮交互（用户确认或高风险审批）时，Mops 采用 **workflow checkpoint** 实现状态续接。具体机制为：

1. **暂停**：Executor 把高风险动作标记为 `awaiting_confirmation`，Coordinator 序列化 MASState 为 checkpoint（含 `actions`、`tool_records`、`attempted_tool_signatures`、`pending_confirmations`），通过 SSE `pause` 事件返回前端。
2. **持久化**：后端将 checkpoint 写入 `agent_confirmations.checkpoint` JSONB 字段，将 `confirm_id` 关联到 SSE 流。
3. **用户确认**：用户在确认卡片上点击"批准/拒绝"，后端将结果写回 `agent_confirmations.status`。
4. **恢复**：前端基于 `confirm_id` 触发新一轮 `/chat` 请求，后端在请求里附带 `resume_context`，Runtime 通过 `MASState.from_request()` 重建状态对象，并通过 `apply_resume_outcome()` 匹配 `confirm_id`，更新 action 状态。

通过 checkpoint，Runtime 在确认期间不需要持有任何内存状态，等同于无状态服务（满足 NFR-4 水平可扩展）。

## 3.5 跨智算平台适配的工具体系

### 3.5.1 工具声明与执行分离

为保证框架的可迁移性（D4），Mops 采用**工具声明层与工具执行层分离**的设计，每个工具被形式化为五元组（图 3-6）：

$$T = (\mathrm{name},\ \mathrm{schema},\ \mathrm{executor},\ \mathrm{permission},\ \mathrm{confirm})$$

**工具声明层**（`tools/definitions.py`）：通过 `@tool` 装饰器声明，平台无关，描述 *what & how to use* 与 risk/permission 元字段。

**工具执行层**（`tools/executor.py` + 适配器）：将声明转化为具体平台调用。Mops 实现三种适配器：`GoBackendToolExecutor`（生产）、`LocalToolExecutor`（开发）、`MockToolExecutor`（评测）。迁移到新平台仅需实现新适配器，声明层无需修改。

### 3.5.2 工具分类、权限矩阵与零信任过滤

Mops 当前实现包含 **88 个工具**，按角色和风险分为四大类，详见图 3-6 的权限矩阵。工具与用户权限的映射在 `tools/tool_selector.py` 中实现。其核心算法 3.4 遵循零信任三原则：

```latex
\begin{algorithm}
\caption{工具权限过滤（零信任三原则）}
\label{alg:tool_sanitize}
\begin{algorithmic}[1]
\Function{SanitizeTools}{$\mathrm{context}$, $\mathrm{enabled\_tool\_names}$}
  \State \Comment{原则 1：角色字段仅来自 Backend JWT}
  \State $\mathrm{role}^* \gets \mathrm{context.actor.role}$ \Comment{忽略 LLM 输出与前端字段}
  \State \Comment{原则 2：页面 scope 可收窄但不可放大}
  \If{$\mathrm{context.page.is\_user\_page}$ \textbf{and} $\mathrm{role}^* = \texttt{admin}$}
    \State $\mathrm{role}^* \gets \texttt{user}$ \Comment{admin 浏览用户页面时降级}
  \EndIf
  \State \Comment{原则 3：暴露给 LLM 前完成裁剪}
  \State $\mathrm{visible} \gets \{\,\}$
  \ForAll{$t \in \mathrm{enabled\_tool\_names}$}
    \If{$\mathrm{IsAllowed}(\mathrm{role}^*, t.\mathrm{permission})$}
      \State $\mathrm{visible} \gets \mathrm{visible} \cup \{t\}$
    \EndIf
  \EndFor
  \State \Return $\mathrm{visible}$
\EndFunction
\end{algorithmic}
\end{algorithm}
```

这一过滤发生在工具被注入 LLM 提示词**之前**，LLM 无从"请求"被过滤掉的工具，杜绝了提示词注入导致权限提升的攻击面。

## 3.6 安全控制与审计机制

智算平台的运维操作可能产生不可逆后果（删除资源、修改配额、重启训练作业），因此安全控制是 Mops 的一等公民。本节描述权限过滤、操作确认、确认续接与审计追踪四道闸门，整体如图 3-7 所示。

**闸门 ① 权限过滤**：算法 3.4 所述，工具暴露给 LLM 前已完成裁剪；权限不足时直接拒绝执行并返回结构化错误，避免静默失败。

**闸门 ② 操作确认**：对中风险（影响自有资源）与高风险（影响他人或不可逆）操作，系统在执行前要求用户明确确认。确认卡片清晰展示：(i) 操作类型与目标对象；(ii) 潜在影响（"将会重启 jobX，导致正在运行的 3 000 步训练丢失"）；(iii) 工具风险等级；(iv) 类似历史案例（若有）。批量操作支持逐项勾选。

**闸门 ③ 确认续接**：依赖 3.4.4 节的 workflow checkpoint 机制。续接算法详见算法 3.5：

```latex
\begin{algorithm}
\caption{Pause-Confirm-Resume 续接}
\label{alg:resume}
\begin{algorithmic}[1]
\Function{Pause}{$s$, $\mathrm{actions}_{\mathrm{pending}}$}
  \State $\mathrm{ckpt} \gets \mathrm{Serialize}(s, \mathrm{version}=5)$
  \State $\mathrm{payload} \gets \{\mathrm{actions}: \mathrm{actions}_{\mathrm{pending}}, \mathrm{checkpoint}: \mathrm{ckpt}, \mathrm{hint}, \mathrm{history}\}$
  \State $\mathrm{Emit}(\texttt{SSE.pause}, \mathrm{payload})$
  \State $\mathrm{Backend.Persist}(\mathrm{agent\_confirmations}, \mathrm{payload})$
\EndFunction
\Statex
\Function{Resume}{$\mathrm{request}$}
  \State $\mathrm{ckpt} \gets \mathrm{request.continuation.workflow}$
  \State $\mathrm{results} \gets \mathrm{request.continuation.confirmation\_results}$
  \State $s \gets \mathrm{Deserialize}(\mathrm{ckpt})$
  \ForAll{$r \in \mathrm{results}$}
    \State $a \gets \mathrm{FindAction}(s, r.\mathrm{confirm\_id})$
    \State $a.\mathrm{status} \gets r.\mathrm{decision}$ \Comment{confirmed / rejected}
    \State $s.\mathrm{attempted\_tool\_signatures}.\mathrm{add}(\mathrm{Fingerprint}(a))$
  \EndFor
  \State \Return \Call{MasLoop}{$s$} \Comment{从 EXECUTE 状态继续}
\EndFunction
\end{algorithmic}
\end{algorithm}
```

**闸门 ④ 审计追踪**：所有运维操作均记录到 PostgreSQL `agent_audit` 表，字段包括 `session_id`、`turn_id`、`actor`、`tool_name`、`tool_args`（JSONB）、`result` 或 `error`、`latency_ms`、`risk_level`、`confirmation_id`、`created_at`。除了基础事件，Mops 还会把每轮的 `tool_records[]` 与 `action_history[]` 整体序列化保留，便于事后回放完整推理轨迹。审计日志支持按用户、按工具、按时间窗口检索。

这四道闸门借鉴 STRATUS[15] 的 Transactional No-Regression (TNR) 思想，并扩展到智算训练作业的不可逆操作（已释放的 GPU 卡时、已删除的 checkpoint 文件等）；对应的安全保证形式化命题为："任意时刻系统外部可见的风险等级不会高于初始基线"，由 ① + ② 联合保障，由 ④ 提供事后核查依据。

## 3.7 本章小结

本章给出了 Mops 多智能体协作框架的系统设计。围绕 3.1 节定义的四类任务与 8 项功能性 + 6 项非功能性需求，3.2 节给出四层架构；3.3 节定义五个核心角色与四个任务型角色，并通过算法 3.1 / 3.2 形式化了 MAS 主循环与含 replan/reassign/retry 的回退机制；3.4 节通过 MASState + StateView（算法 3.3）实现按角色视图的上下文裁剪，并通过 workflow checkpoint 支撑多轮续接；3.5 节通过工具声明-执行分离与零信任过滤（算法 3.4）保证跨平台可迁移与安全可控；3.6 节通过四闸门 + Pause-Confirm-Resume 续接（算法 3.5）保障高风险操作可控可追溯。下一章将基于本设计给出系统在 Crater 平台上的工程实现。
