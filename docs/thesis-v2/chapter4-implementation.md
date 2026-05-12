# 第 4 章 面向智算平台的智能运维系统实现

本章把第 3 章的设计落地为可在 Crater 智算平台上运行的工程系统，是把"理念"转换成"参数"的过程。4.1 节给出三服务总体架构与部署形态；4.2 节给出业务网关层的数据持久化模型与对外能力，包括 SSE 流式输出与审计中继的关键机制；4.3 节给出 Agent Runtime 内部的记忆装载、多智能体编排与工具执行实现；4.4 节给出离线评测数据集构造与六维加权评分；4.5 节做本章小结。

## 4.1 系统整体架构

MOps 系统由三个独立部署的服务组成，统一运行在 Crater 同一 Kubernetes 集群中，如图 4-1 所示。**前端服务** 基于 React 18 + TypeScript 实现，承担 Portal 用户工作台与 Admin Console 管理控制台两套界面，副本数 2-5（HPA 弹性）；**业务网关服务** 基于 Go 1.22 + Gin + GORM 实现，副本数 3-10，是系统的安全网关；**Agent Runtime 服务** 基于 Python 3.11 + FastAPI + LangChain/LangGraph 实现，副本数 2-8，承担 LLM 推理与多智能体编排。

三个服务在部署形态上共享同一个 Kubernetes 集群，但权限边界与职责边界严格分离。前端通过 Ingress 暴露 HTTPS 端点，使用 EventSource API 与网关建立 SSE 长连接；网关与 Runtime 通过集群内 ClusterIP Service 通信，二者均不直接暴露至外网；Runtime 与外部 LLM 服务之间走 Egress Gateway 统一鉴权与限流。这种"前向收敛、横向隔离"的部署形态使得任何对外攻击面都被压缩在前端与 Ingress 两点上。

部署形态严格遵循两条非功能性约束。第一，**Runtime 不持有任何 Kubernetes 凭据**——Kubernetes ServiceAccount 仅绑定到网关的 Pod 上，Runtime 对平台资源的任何读写都必须通过网关转发，凭据隔离从根本上杜绝了"Runtime 一旦被攻破即获得集群级权限"的攻击路径。第二，**Runtime 是无状态服务**——所有跨轮状态（包括确认期间的中断状态）都以 workflow checkpoint 序列化进网关层的 PostgreSQL，因此任何 Pod 都可以处理任何续接请求，HPA 可以根据并发轮次数自由扩缩容。

在性能预算上，单轮端到端时延 90 百分位被设定为 60 秒以内、单轮 token 消耗的目标区间为 5000-15000、单 Pod 并发上限为 32 路 SSE 流。这些参数并非凭空设定，而是由 5.4 节的离线评测反推得到，使工程实现与评测口径保持一致。

## 4.2 前后端实现

### 4.2.1 数据持久化 ER 模型

业务网关在 PostgreSQL 中新增七张 `agent_*` 表（图 4-2），与平台原有的用户、作业、账户、告警等表共存。七张表分为四组职责：

**会话组**包含会话、轮次、消息三张表。`agent_sessions` 是用户与 Agent 之间一次完整交互的容器，主键为 uuid，附带 user_id、account_id、orchestration（编排模式）与状态字段。`agent_turns` 是一次具体问询的载体，关键字段是 page_context（页面上下文快照）、continuation（续接载荷）、workflow（v5 版本的 checkpoint JSONB）与 usage_summary——前者使每一轮请求都能恢复上下文，后者使任何一次失败都可以"原地复活"。`agent_messages` 记录每一轮内部产生的角色消息序列。

**审计组**仅包含一张 `agent_audit` 表，但它是整个系统的"事实记忆"。它记录每一次工具调用的发起者、目标工具、入参（JSONB）、结果或错误、延时（毫秒）、风险等级、对应确认 ID。表上设计了三类索引：`(session_id, turn_id)` 复合索引支撑按会话回放、`(actor_id, created_at DESC)` 索引支撑用户视角的审计、`(tool_name, created_at) WHERE error IS NOT NULL` 部分索引支撑"上周哪些工具失败率最高"这类运维分析查询；此外，`tool_args` JSONB 字段上建立了 GIN 索引以支撑按参数检索。审计表与其他业务表之间不设数据库级外键约束，外键仅在应用层管理——这使得即使会话被软删除，审计记录依然可独立保留，满足合规要求。

**确认与巡检组**包含 `agent_confirmations` 与 `agent_inspection_runs`。前者承载每一次暂停-确认-续接的中间状态，最关键的字段是 checkpoint（JSONB 序列化的 MASState），其上有部分索引 `(status, expires_at) WHERE status='pending'` 用于扫描待确认任务；后者承载 cron 巡检的元信息，其上有 `(scope, started_at DESC)` 索引支撑巡检列表页。

**质量回流组**仅含 `agent_quality_eval` 一张表，承载用户 👍/👎 反馈与离线评测的双裁判分数，使评测数据形成"在线 → 离线 → 评分回流"的闭环。

### 4.2.2 后端能力与 SSE 流式实现

业务网关向前端暴露三组路由：**对话路由** 承载 `POST /v1/chat`（SSE）与 `GET/POST /v1/confirmations/:id`；**审计路由** 承载会话列表与单轮详细轨迹查询；**巡检路由** 承载巡检任务列表与报告详情。除此之外还有一组内部路由 `POST /v1/agent/tools/execute`，仅供 Runtime 回调网关执行具体工具，前端不可访问。

**SSE 事件流式实现** 是整个系统体验顺滑度的核心。用户发送消息时，前端通过 EventSource 订阅 `/v1/chat`；网关解析 JWT、计算工具白名单、构造请求上下文，然后立即建立 HTTP 连接到 Runtime 的 `/chat` 端点，并把上游事件原样转发给前端，同时拷贝一份进入 ring buffer 异步审计。事件类型至少包括 `thinking`（思考过程）、`tool_call`（工具调用）、`tool_result`（工具结果）、`message`（中间消息）、`pause`（暂停等待确认）、`done`（完成）、`error`（错误）。流式转发与审计中继的核心逻辑由算法 4.1 给出：

```latex
\begin{algorithm}
\caption{Backend SSE 事件流式路由与审计中继}
\KwData{HTTP 请求 $\mathrm{req}$，响应写入器 $w$}
\KwResult{流式响应 + 异步审计落库}
$\mathrm{ctx} \gets \textsc{Authenticate}(\mathrm{req.headers})$ \tcp*{JWT 解析 actor.role}
$\mathrm{ctx.page} \gets \mathrm{req.page\_context}$\;
$\mathrm{ctx.cap} \gets \textsc{SanitizeTools}(\mathrm{ctx}, \mathrm{ToolCatalog})$\;
$\mathrm{tid} \gets \textsc{NewTurnId}()$;\ $\textsc{PersistTurnStart}(\mathrm{ctx}, \mathrm{tid})$\;
$\mathrm{rb} \gets \textsc{NewRingBuffer}(512\,\mathrm{KB})$ \tcp*{异步审计缓冲}
$\mathrm{up} \gets \textsc{HttpPostStream}(\mathrm{Runtime}/\texttt{chat}, \mathrm{ctx})$\;
\ForEach{$e \in \mathrm{up}$}{
  $w.\textsc{Write}(\textsc{SerializeSSE}(e))$ \tcp*{原样转发}
  $\mathrm{rb}.\textsc{Append}(e)$\;
  \If{$\mathrm{rb}.\textsc{Full}()$ \textbf{or} $e.\mathrm{type} \in \{\texttt{tool\_result}, \texttt{pause}, \texttt{done}\}$}{
    $\textsc{Persist}(\texttt{agent\_audit}, \mathrm{rb}.\textsc{Flush}())$\;
  }
  \If{$e.\mathrm{type} = \texttt{pause}$}{$\textsc{Persist}(\texttt{agent\_confirmations}, e.\mathrm{payload})$}
}
$\textsc{UpdateTurnStatus}(\mathrm{tid}, \texttt{completed})$\;
\end{algorithm}
```

该算法在工程上的几个关键点值得说明。第一，**ring buffer 异步批写**：单 Pod 平均承载 10-30 路 SSE 流时，若每个事件都同步写库，PostgreSQL 写入压力会成为瓶颈；通过 512 KB ring buffer + 关键事件强制刷盘的策略，可以在保留审计完整性的同时把数据库 IOPS 降低近一个数量级。第二，**关键事件同步刷盘**：`tool_result`、`pause`、`done` 三类事件被强制刷盘，因为它们承载了"实际发生过的事实"——其它中间事件即使丢失也可由结果反推。第三，**`pause` 事件的双写**：除了写入审计表，确认载荷还需要写入 `agent_confirmations` 表并通过部分索引 `(status, expires_at)` 暴露给后续的待确认扫描任务。

**确认续接的前后端协同**遵循一条简单约定：续接是一个**新的 chat 请求**而不是某个长连接的"恢复"。前端在用户点击批准/拒绝后，组装一个携带 `continuation.workflow`（来自 pause 事件的 checkpoint）与 `continuation.confirmation_results`（用户的决定）的新请求发往 `/v1/chat`；网关拿到该请求后透传给 Runtime，Runtime 从 checkpoint 反序列化 MASState、把 confirmation_results 合并入 actions 列表、然后从 EXECUTE 阶段继续推进。这种"无状态续接"使得续接请求可以被任意 Runtime 副本承接、可以跨域名跨 Pod 完成、甚至可以在原 Pod 已被回收后由新副本拾起——这是 Runtime 水平扩展的实际体现。

**前端页面要点**：Portal 侧的对话抽屉位于平台原生页面右侧，可被收起；抽屉内自顶向下依次是页面上下文徽章（"作业 jobX | 节点 nodeA"）、执行时间线（thinking/tool_call/tool_result 三类事件折叠展示）、消息气泡区、输入框。用户在不同页面切换时徽章自动更新，新建对话亦保留页面上下文作为隐式参数。Admin Console 侧新增两个页面：在线审计页按时间倒序展示会话列表，可下钻查看完整推理轨迹并支持按工具、按错误码、按用户检索；智能巡检页按时间展示每小时巡检报告，对每条异常作业提供下钻分析。AIOps 看板复用巡检页的批处理能力，仅在前端做不同的聚合视图。

## 4.3 智能体服务实现

### 4.3.1 记忆与上下文实现

第 3 章用 MASState 与 StateView 描述了记忆的"理念"，本节给出"工程实现"。Runtime 的记忆能力由三个步骤组成：上下文装载、视图投影、token 预算压缩。

**上下文装载**把后端传来的对话历史按时间逆序转换为 LangChain 的标准消息序列（HumanMessage / AIMessage / ToolMessage）。装载过程按总 token 预算贪心装箱：在默认 4000 token 预算下，保留最近 3 轮完整对话，更早的对话以"用户问 X，Agent 答 Y"摘要替代；工具结果按 1200/1200 字符的头尾截断保留，错误结果留更多文本（1600 字符）以利诊断。这种"近详远略 + 头尾保留"的策略遵循信息论意义上的最小损失装载——开头交代意图、结尾交代结论，最容易被截断的是中间冗余。

**视图投影**在每一个 Agent 被调用前发生，由 BuildView 函数承担。Planner 的视图仅含 Goal 与 capabilities，约 20% 预算；Explorer 视图增加 Plan 与已有 Observation，约 45% 预算；Executor 视图再增加 actions 草稿，约 25% 预算；Verifier 视图涵盖 Plan、Observation 与 Execution，约 10% 预算；Coordinator 看到全状态但以轻量摘要形态出现，控制在 1000 token 以内。预算分配的 $\alpha$ 系数（式 3.2）在配置文件中可调，但落地时遵循"Explorer 最厚、Verifier 最薄"的经验法则——前者承担证据搜集，需要更多空间；后者承担结构性判断，对噪声敏感反而要保持简洁。

**token 预算压缩**是工程实现的真正难点。系统全局维护一个 token 计数器，对历史消息使用 tiktoken 实时估算，对工具结果使用三档处理：原样保留（≤预算）、结构化头尾截断（结构化数据）、LLM 摘要（非结构化文本）。摘要由轻量 Flash 模型完成，仅在结构化截断不可行时触发，避免每条结果都做一次 LLM 调用。

### 4.3.2 多智能体编排与任务状态实现

Runtime 启动时实例化三种编排器：单 Agent ReAct、Plan-Execute 与完整 MAS。编排器之间通过工具集合与 MASState 字段子集保持兼容，使得它们可以在第 5 章的对比实验中公平比较。

**编排器选择**由 `IntentRouter` 与 `get_orchestration_mode` 协同完成。IntentRouter 是一个独立的轻量 LLM Agent，使用 Qwen 3 Flash 这类小模型完成两级路由：第一级是确定性启发式（页面命中、greeting 检测、resume 标志等），第二级才进入 LLM 分类，输出结构化的 JSON `{entry_mode, op_mode, action, confidence}`，对应"对话/帮助/续接"的入口模式与"读/写"的操作模式。两级路由的合并由式 (3.1) 给出——高置信启发式直通、中置信加权合并、低置信 LLM 优先；这种分级策略的目标是把意图分类的 token 成本压在不到 200 token，同时保持 90% 以上的分类准确率（第 5 章评测）。

**MAS 主循环**是 Runtime 内最复杂的子系统。它围绕 MASState 与 Coordinator 状态机展开：每一轮循环首先由 Coordinator 决定下一阶段，然后调度对应角色，把结果写回 MASState，最后判断终止条件。算法 4.2 给出主循环骨架：

```latex
\begin{algorithm}
\caption{MAS 主循环（Coordinator 状态机驱动）}
\KwData{目标 $g$，能力 $\mathcal{T}$，运行配置 $\theta$}
$M \gets \textsc{Init}(g, \mathcal{T})$;\ $r \gets 0$\;
\While{$r < \theta.\mathrm{lead\_max\_rounds}$ \textbf{and} $M.\mathrm{stop} = \varnothing$}{
  $\phi \gets \textsc{Coordinator.NextPhase}(M)$\;
  \uIf{$\phi = \texttt{plan}$}{$M.\mathrm{plan} \gets \textsc{Planner}(M.\mathrm{goal}, \mathcal{T})$}
  \uElseIf{$\phi = \texttt{explore}$}{$M.\mathrm{obs} \gets \textsc{Explorer}(M.\mathrm{plan}, M.\mathrm{obs})$}
  \uElseIf{$\phi = \texttt{execute}$}{
    $M.\mathrm{exec} \gets \textsc{Executor}(M.\mathrm{plan}, M.\mathrm{obs})$\;
    \If{\textbf{exists} 待确认动作}{
      $\textsc{Emit}(\texttt{pause}, \textsc{Checkpoint}(M))$;\ \textbf{return} $M$\;
    }
  }
  \uElseIf{$\phi = \texttt{verify}$}{
    $v \gets \textsc{Verifier}(M)$;\ $M \gets \textsc{Dispatch}(M, v)$\;
    \If{$v.\mathrm{verdict} = \texttt{pass}$}{\textbf{break}}
  }
  \If{\textsc{NoProgress}($M, \theta$)}{$M.\mathrm{stop} \gets \texttt{no\_progress}$}
  $r \gets r + 1$\;
}
\Return $\textsc{Coordinator.Summarize}(M)$\;
\end{algorithm}
```

每一个角色 Agent 都继承自一个公共基类 `BaseRoleAgent`，其调用流程被固化为三步：以 `BuildView` 构造该角色的局部状态视图、渲染模板化的角色提示词并调用配置好的 LLM（带超时与重试）、解析 LLM 输出并写回 MASState。各角色的核心差异体现在提示词模板、LLM 模型选择与允许写入的 MASState 字段三个维度。Planner 与 Explorer 使用低延时的 Flash 模型；Executor 与 Verifier 使用 Thinking 思考型模型——这种"重决策用大、轻执行用小"的异构 LLM 路由是 MOps 成本可控（D6）的工程基础。

**Explorer 的迭代式证据收集**值得单独描述。它的内层循环遵循"每轮最多两个工具、新颖度阈值守护"原则：每次从候选工具队列中选择两个可调用工具（受角色权限过滤）、执行后计算证据新颖度（与已有 Observation 的差异性），若连续两轮新颖度低于阈值则提前终止；同时检查 `attempted_tool_signatures` 集合避免对相同工具用相同参数重复调用。Verifier 在收尾阶段对 Observation 做"挑战式验证"：分别从证据-结论一致性、遗漏排查路径、建议风险、合规四个维度打分，加权得到总分；任一安全维度低于阈值则直接返回 `risk`，证据/遗漏维度低则返回 `insufficient_evidence` 并附带 replan 提示。这种"显式化失败模态"是 Verifier 区别于一般"质量评分器"的根本差异。

**任务状态序列化**采用 dataclass + JSONB 的混合序列化策略：MASState 的所有字段被深度序列化为 JSON 树，时间戳归一化为 ISO 8601，受信任的 Python 对象（如 Action、ToolRecord）实现 `to_dict`/`from_dict` 双向方法。序列化版本号目前是 5，每一次字段不兼容变更都递增版本号；反序列化时若版本号低于当前则触发迁移函数链 `migrate_v3_to_v4 → migrate_v4_to_v5`，确保线上数据库中遗留的旧版本 checkpoint 可在新版 Runtime 上继续续接。

### 4.3.3 工具与安全模块实现

**工具声明解析**采用装饰器模式：每个工具以 `@tool` 形式声明，装饰器除常规 schema 外额外接受 permission、risk、confirm 三个元字段。装饰器在装载时把工具元信息注册到全局注册表，并自动生成 LangChain 标准的 BaseTool 子类。Runtime 在每次请求构造可用工具列表时，从注册表中按 actor role 与 page 过滤——这一过滤即算法 3.3 的 SanitizeTools 在工程上的具体落地，结果作为白名单交给 LLM。

**工具执行** 通过抽象的 ToolExecutor 接口完成，实际承担生产流量的是 GoBackendToolExecutor。它的核心动作是把工具调用打包成 HTTPS 请求发往业务网关 `/v1/agent/tools/execute` 端点，并按响应状态做三态处理：

```latex
\begin{algorithm}
\caption{GoBackendToolExecutor 三态返回}
\KwData{工具 $\tau$，入参 $A$，上下文 $C$}
$t_0 \gets \textsc{Now}()$\;
$\mathrm{resp} \gets \textsc{HttpPost}(\mathrm{Backend}/\texttt{tools/execute}, \{\tau, A, C\}, C.\mathrm{jwt})$\;
\Switch{$\mathrm{resp.status}$}{
  \Case{\texttt{success}}{\Return $\textsc{ToolResult}(\texttt{ok}, \mathrm{resp.r})$}
  \Case{\texttt{confirmation\_required}}{\Return $\textsc{ToolResult}(\texttt{paused}, \textsc{NewAction}(\mathrm{cid}=\mathrm{resp.cid}))$}
  \Case{\texttt{permission\_denied}}{\Return $\textsc{ToolResult}(\texttt{error}, \texttt{permission\_denied})$}
  \Other{\Return $\textsc{ToolResult}(\texttt{error}, \mathrm{resp.err})$}
}
\end{algorithm}
```

第一态 `success` 是常规读类工具的成功返回，结果被写入 MASState 的 tool_records 与 Observation.evidence；第二态 `paused` 是写类工具命中确认流的情形——网关返回 `confirmation_required` 状态码并附带 confirm_id，Executor 据此把对应 action 标记为 `awaiting_confirmation` 并触发 Coordinator 进入 PAUSE 阶段；第三态 `error` 包括权限拒绝与底层异常，Verifier 据此决定是否进入 retry 分支。审计落库由网关在工具执行时同步完成，Runtime 不重复写入——这一约定是"审计真实性"的保证：如果 Runtime 自行写审计，恶意 Runtime 可能伪造记录，因此审计的写入主体必须是处于安全边界之内的网关而不是 LLM 推理服务。

**跨平台适配**通过 ToolExecutor 接口的多实现完成。除 GoBackend 之外，Runtime 还实例化两类适配器：LocalExecutor 仅在开发期使用，把工具调用映射到本地函数与内置 Mock；MockExecutor 在评测期使用，按场景快照回放工具返回。三种适配器对 MASState 透明——同一个角色 Agent 在生产/开发/评测三种模式下的代码路径完全一致，仅注入的 Executor 不同。这是工具声明-执行分离的工程红利：迁移到新智算平台时，只需要实现一个新的 Executor 子类，三个编排器、九个角色、几十种工具声明都不必修改。

**权限校验在双侧执行**：Runtime 侧通过 SanitizeTools 在 LLM 注入前裁剪工具白名单；网关侧在每一次工具执行请求到来时再次校验角色与目标对象的所有权（例如普通用户不能停止他人的作业）。两侧权限校验是有意冗余设计——任何一侧的失败都不会让无权操作泄漏到平台。这种"信任内层、防御外层"的双重防线借鉴了网络安全中的纵深防御思想。

**巡检流水线**复用 MAS 主循环，仅替换 Coordinator 的起始提示词与目标对象来源。cron 定时器每小时触发一次，按 scope（失败作业 / 空闲 GPU / 长时空闲 Jupyter）查询目标集合，对每个目标构造一个 admin 上下文然后调用 MAS 主循环，最终把所有目标的分析结果聚合为巡检报告并通知管理员。巡检与交互对话共用一套编排代码，差别仅在入口与上下文构造——这是"一套核心、多种场景"的工程典范。

## 4.4 质量评估模块

### 4.4.1 评测数据集构造

评测数据集是 MOps 系统迭代的"指南针"——它把"什么是好的多智能体运维助手"从模糊感受变成可度量的指标。本文的评测数据集由 85 个真实场景构成，每个场景以 JSON 格式描述，包含场景 ID、类别（诊断/运维/查询/提交）、难度等级（1-5）、轮次数、ground_truth（期望工具序列、根因关键词、答复关键要素、期望确认工具）、score_profile（六维评分权重）与难度权重等字段。

**数据采集流程**分两步进行。第一步通过 SQL 与 shell 脚本分别从 PostgreSQL 业务库与 Kubernetes 集群拉取真实快照——后端 API 真实返回、作业元数据、事件流、告警时序、容器日志等都被原样保留，这些快照构成评测的"地面真相"。第二步通过一个转换脚本把快照转化为 MockExecutor 可回放的格式：每个工具调用被映射为一个固定的快照查询键，相同入参在回放时返回完全相同的输出，使得评测结果可严格复现。

**标注流程**遵循"三轮收敛"原则。第一轮，作者根据场景类别与典型问询样本手工编写期望的工具序列（区分 must 与 nice-to-have）、答复关键要素与风险动作清单；第二轮，运行 MOps 系统 3-5 次试跑，记录其完整推理轨迹；第三轮，由两名研究人员对每条轨迹做盲审打分，分歧场景由第三人裁决。三轮收敛后得到的最终数据集既反映"模型应当做对的事"，也反映"模型容易做错的事"，两者结合后才能用作驱动迭代的有效信号。

### 4.4.2 评分 Profile 与评测执行器

评分系统采用六维加权评分，其中四维由规则计算、两维由 LLM-as-Judge 完成。**Tool F1** 比较实际工具序列与期望序列的集合 F1 分数；**RootCauseHit** 检查根因关键词是否在最终答复中出现；**ChatJudge** 由外部评测 LLM 对答复的连贯性、领域准确性、可操作性打分；**PermissionCompliance** 检查是否调用了无权工具，违规一票否决；**Efficiency** 衡量工具调用数与期望数的偏离；**MultiTurnPass** 用于包含确认环节的多轮场景，检查所有轮次是否合规通过。每个场景的最终分数按式 (4.1) 计算：

$$
\mathrm{OS}_i^{\mathrm{norm}} = \frac{w_{d(i)} \cdot \sum_{m \in \mathcal{M}} w_m\, s_{i, m}}{w_{d(i)} \cdot \sum_{m \in \mathcal{M}} w_m},
\quad w_{d(i)} = \begin{cases} 1.0, & \mathrm{easy} \\ 1.2, & \mathrm{medium} \\ 1.5, & \mathrm{hard} \end{cases}
\tag{4.1}
$$

其中 $w_m$ 为六维评分权重、$w_{d(i)}$ 为难度权重。难度归一化的目的是避免数据集中难易场景分布的偏置——若 hard 场景占少数却能拉高总分，则评分将过度奖励"避难"的策略；通过 1.0/1.2/1.5 的递增权重，难场景在加权平均中获得更高占比，使指标更贴近"难任务做好"的设计目标。

```latex
\begin{algorithm}
\caption{六维加权评分}
\KwData{场景 $S$，推理轨迹 $\mathrm{tr}$}
\KwResult{综合得分 $\mathrm{OS} \in [0, 100]$}
$f_1 \gets \textsc{ToolF1}(S.\mathrm{gt.tools}, \mathrm{tr.tools})$ \tcp*{AST 匹配}
$f_2 \gets \textsc{RootCauseHit}(S.\mathrm{gt.kw}, \mathrm{tr.answer})$\;
$f_3 \gets \textsc{ChatJudge}(S.\mathrm{gt.keys}, \mathrm{tr.answer})$\;
$f_4 \gets \textsc{PermissionCompliance}(\mathrm{tr})$\;
\If{$f_4 = 0$}{\Return $0$ \tcp*{一票否决}}
$f_5 \gets \textsc{Efficiency}(|S.\mathrm{gt.tools}|, |\mathrm{tr.unique\_calls}|)$\;
$f_6 \gets \textsc{MultiTurnPass}(S, \mathrm{tr})$\;
$\mathbf{w} \gets S.\mathrm{score\_profile.weights}$;\ $\mathbf{f} \gets (f_1, \ldots, f_6)$\;
\Return $\textsc{Clamp}(100 \cdot \mathbf{w} \cdot \mathbf{f}^\top \cdot w_{d(S)}, 0, 100)$\;
\end{algorithm}
```

**评测执行器** 同时提供顺序与并行两种执行模式。顺序模式逐场景串行执行，便于调试；并行模式以 asyncio 并发执行多个场景，吞吐率提升约 6 倍，适合大规模回归。每个场景执行完毕后被写入一组 CSV 详细日志，包括场景级摘要与轮次级详情，便于事后下钻。整个评测流水线在第 5 章被反复使用，是支撑实验结论的方法学基础。

**质量反馈闭环** 通过 `POST /eval/quality/feedback` 端点回收 Portal 中用户的 👍/👎 反馈，触发 ChatJudge（评对话质量）与 ChainJudge（评任务完成度）并行打分，结果写入 `agent_quality_eval` 表形成质量证据库。质量证据库与离线评测数据集相互印证：前者反映"用户认为做得好不好"，后者反映"评测口径认为做得好不好"，二者长期偏差若超出阈值则触发数据集更新——这是评测自身演化的元闭环。

## 4.5 本章小结

本章把第 3 章的设计落地为可在 Crater 智算平台上运行的工程系统。4.1 节确立前端、后端、Runtime 三服务架构与"凭据隔离、状态无栈"两条部署约束；4.2 节给出业务网关层的七张持久化表与三组路由能力，并通过 SSE 流式中继算法实现"事件转发 + 异步审计 + 暂停-续接"三件事一体化；4.3 节给出 Agent Runtime 内部的记忆装载、多智能体编排状态机、工具执行三态返回与跨平台适配机制；4.4 节给出 85 个真实场景的评测数据集构造与六维加权评分的具体实现。整个系统在"安全可控、可水平扩展、跨平台可迁移、可观测"四个维度上达到工程落地要求，并通过质量反馈与离线评测的双闭环驱动持续迭代。下一章将基于本章实现进行大规模离线评测与线上案例分析，验证 MOps 设计的有效性。
