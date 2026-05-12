# 第 4 章 面向智算平台的智能运维系统实现

本章把第 3 章给出的平台无关多智能体协作框架（MOps）落地为运行在 Crater 智算平台上的工程系统，是把"理念"转换为"参数"的过程。4.1 节给出三服务整体架构与部署形态；4.2 节给出前后端实现，包括数据持久化模型与后端 SSE 流式能力；4.3 节给出 Agent Runtime 服务的工程实现，覆盖记忆、多智能体编排与任务状态、工具与安全模块；4.4 节给出质量评估模块；4.5 节小结。

## 4.1 系统整体架构

MOps 系统由三个独立部署的服务组成，统一运行在 Crater 同一 Kubernetes 集群中。**前端服务**基于 React 18 + TypeScript 实现，承担 Portal 用户工作台与 Admin Console 管理控制台两套界面；**业务网关服务**基于 Go 1.22 + Gin + GORM 实现，是系统的安全网关；**Agent Runtime 服务**基于 Python 3.11 + FastAPI + LangChain/LangGraph 实现，承担 LLM 推理与多智能体编排。前端通过 Ingress 暴露 HTTPS 端点并使用 EventSource API 与网关建立 SSE 长连接；网关与 Runtime 通过集群内 ClusterIP Service 通信，二者均不直接暴露至外网；Runtime 访问外部 LLM 服务时经由 Egress Gateway 统一鉴权与限流。这种"前向收敛、横向隔离"的部署形态把对外攻击面压缩在 Ingress 与前端两点上。

本课题的智能运维系统并非替代 Crater 现有的核心控制面，而是作为"智能化副屏（Copilot）"与"后台大脑（Brain）"插件式地挂载于 Crater 平台之上：Agent 获取平台状态或对其变更，完全依赖 Crater 暴露的 OpenAPI 与业务库。例如 Agent 规划出的"查询节点资源"操作，最终落地为对网关 `/api/v1/nodes/metrics` 的代理调用。这种"旁路集成"既保证了 Crater 核心调度链路的稳定性，又赋予了平台强大的自然语言交互与自动化排障能力。

部署形态严格遵循两条非功能性约束。其一，**Runtime 不持有任何 Kubernetes 凭据**——ServiceAccount 仅绑定到网关 Pod 上，Runtime 对平台资源的任何读写都必须经由网关转发；凭据隔离从根本上杜绝了"Runtime 一旦被攻陷即获得集群级权限"的攻击路径。其二，**Runtime 是无状态服务**——所有跨轮状态（包括确认期间的中断状态）都以 workflow checkpoint 序列化进网关层的 PostgreSQL，因此任何 Pod 都可以处理任何续接请求，HPA 可以根据并发轮次数自由扩缩容。在开发与联调环境中，主仓库通过软链接机制（如根目录的 `make config-link` 调用的 `hack/config.sh`）打通三端的配置文件，缺失关键配置文件时网关会在加载阶段直接熔断，从而保证三端协同开发时环境配置的强一致性。

## 4.2 前后端实现

### 4.2.1 数据持久化 ER 模型

业务网关在 PostgreSQL 中新增七张以 `agent_*` 为前缀的表，与平台原有的用户、作业、账户、告警等表共存。七张表分为四组职责。**会话组**包含 `agent_sessions`（一次完整交互的容器，主键 uuid，附带 user_id、account_id、orchestration 编排模式与状态）、`agent_turns`（一次具体问询的载体，关键字段是 page_context、continuation、workflow 即 v5 版本的 checkpoint JSONB 与 usage_summary）以及 `agent_messages`（每一轮内部产生的角色消息序列）；前者使每一轮都能恢复上下文，中者使任何一次失败都可以"原地复活"。

**审计组**仅包含一张 `agent_audit` 表，却是整个系统的"事实记忆"。它记录每一次工具调用的发起者、目标工具、入参（JSONB）、结果或错误、延时、风险等级、对应确认 ID。为了支撑常见运维查询，表上设计了三类索引：`(session_id, turn_id)` 复合索引支持按会话回放；`(actor_id, created_at DESC)` 索引支持用户视角的审计；针对错误工具的部分索引 `(tool_name, created_at) WHERE error IS NOT NULL` 支持"上周哪些工具失败率最高"这类故障复盘；另外在 `tool_args` JSONB 字段上建立 GIN 索引以支持按参数检索。审计表与其他业务表之间不设数据库级外键，外键仅在应用层管理，使得即使会话被软删除，审计记录依然可以独立保留，满足合规审计"不可篡改"的要求。

**确认与巡检组**包含 `agent_confirmations` 与 `agent_inspection_runs` 两张表。前者承载每一次"暂停—确认—续接"的中间状态，最关键的字段是 `checkpoint`（JSONB 序列化的 MASState），并附部分索引 `(status, expires_at) WHERE status='pending'` 用于扫描待确认任务；后者承载 cron 巡检的元信息，索引 `(scope, started_at DESC)` 支撑巡检列表页。**质量回流组**仅含 `agent_quality_eval` 一张表，承载用户 👍/👎 反馈与离线评测的双裁判分数，使评测数据形成"在线 → 离线 → 评分回流"的闭环。

### 4.2.2 后端能力与 SSE 流式实现

业务网关向前端暴露三组路由：**对话路由** `POST /v1/chat`（SSE）与 `GET/POST /v1/confirmations/:id`，**审计路由**承载会话列表与单轮详细轨迹查询，**巡检路由**承载巡检任务列表与报告详情。此外还有一组内部路由 `POST /v1/agent/tools/execute`，仅供 Runtime 回调网关执行具体工具，前端不可访问。所有发往 Python Agent 的请求在进入网关时首先经过 JWT 中间件解析出用户身份，权限拦截器查询 `Users` 与 `Role_Permissions` 表完成 RBAC 校验，校验失败的请求直接在网关层面被拒绝（HTTP 403），防止恶意指令消耗昂贵的 LLM 推理算力。

**SSE 事件流式实现**是整个系统体验顺滑度的核心。用户发送消息时，前端通过 EventSource 订阅 `/v1/chat`；网关解析 JWT、计算工具白名单（即第 3 章的 SanitizeTools）、构造请求上下文，然后立即建立 HTTP 连接到 Runtime 的 `/chat` 端点，并把上游事件原样转发给前端，同时拷贝一份进入 ring buffer 异步审计。SSE 事件类型包括 `thinking`（思考过程）、`tool_call`（工具调用）、`tool_result`（工具结果）、`message`（中间消息）、`pause`（暂停等待确认）、`done`（完成）、`error`（错误）。

工程实现上有三个关键点。**第一**，单 Pod 平均承载 10–30 路 SSE 流时，若每个事件都同步写库，PostgreSQL 写入压力会成为瓶颈；通过 512 KB ring buffer + 关键事件强制刷盘的策略，可以在保留审计完整性的同时把数据库 IOPS 降低近一个数量级。**第二**，`tool_result`、`pause`、`done` 三类事件被强制刷盘，因为它们承载了"实际发生过的事实"——其它中间事件即使丢失也可由结果反推。**第三**，`pause` 事件需要双写：除写入审计表外，确认载荷还需写入 `agent_confirmations` 表并通过部分索引暴露给后续的待确认扫描任务。

**确认续接的前后端协同**遵循一条简单约定：续接是一个**新的 chat 请求**而不是某条长连接的"恢复"。前端在用户点击批准/拒绝后，组装一个携带 `continuation.workflow`（来自 pause 事件的 checkpoint）与 `continuation.confirmation_results`（用户的决定）的新请求发往 `/v1/chat`；网关拿到该请求后透传给 Runtime，Runtime 从 checkpoint 反序列化 MASState、把 confirmation_results 合并入 actions 列表、然后从 EXECUTE 阶段继续推进。这种"无状态续接"使续接请求可以被任意 Runtime 副本承接、可以跨域名跨 Pod 完成、甚至可以在原 Pod 已被回收后由新副本拾起——这是 Runtime 水平扩展的实际体现。

**异常路径排查约定**：链路异常时，网关自身连接 Runtime 超时或被拒绝会生成状态码 500 而非透传 502。前端若观察到 502，通常意味着 Ingress/Vite Proxy 到网关的链路出现问题；若是 500，则需优先排查 Runtime 进程是否存活或 LLM 远端推理是否超时。

**前端页面与组件**按职能划分为三组：原生控制台（Crater 既有的集群总览、作业列表、节点详情等页面，并在其中埋入路由监听器以捕获页面上下文）；AIOps 对话副屏（侧边栏形式，承载 SSE 渲染、`thinking/tool_call/tool_result` 折叠时间线与消息气泡）；人机协同卡片（`ConfirmActionCard`、`ParameterReviewCard`、`BatchConfirmCard` 等，针对 Agent 返回参数长度不可控的问题统一实施"限制最大高度 + 内部滚动"的安全布局：文本采用 `[overflow-wrap:anywhere] break-words` 强制换行，弹窗采用 `flex flex-col max-h-[85vh]` 限高，中间参数容器 `flex-1 overflow-y-auto`，确保 Header 与 Footer 始终固定可见）。前端响应包与后端约定了 `msgKey` 字段协议，国际化语言以 `msgKey` 在本地包匹配，避免后端硬编码中文。Admin Console 侧新增"在线审计页"（按时间倒序展示会话列表，可下钻查看完整推理轨迹，支持按工具、错误码、用户检索）与"智能巡检页"（按时间展示每小时巡检报告，下钻查看异常作业与建议）；AIOps 看板复用巡检页的批处理能力，仅做不同的聚合视图。

## 4.3 智能体服务实现

### 4.3.1 记忆与上下文实现

第 3 章用 MASState 与 StateView 描述了记忆的"理念"，本节给出"工程实现"。Runtime 的记忆能力由三个步骤组成：上下文装载、视图投影、token 预算压缩。**上下文装载**把网关传来的对话历史按时间逆序转换为 LangChain 的标准消息序列（HumanMessage / AIMessage / ToolMessage），按总 token 预算贪心装箱：在默认 4000 token 预算下，保留最近 3 轮完整对话，更早的对话以"用户问 X，Agent 答 Y"的摘要替代；工具结果按 1200/1200 字符的头尾截断保留，错误结果留更多文本（1600 字符）以利诊断。这种"近详远略 + 头尾保留"的策略遵循信息论意义上的最小损失装载——开头交代意图、结尾交代结论，最容易被截断的是中间冗余。

**视图投影**在每一个 Agent 被调用前发生，由 `BuildView` 函数承担，按角色把 MASState 投影成局部子集：Planner 仅含 Goal 与 capabilities，约 20% 预算；Explorer 增加 Plan 与已有 Observation，约 45% 预算；Executor 再增加 actions 草稿，约 25% 预算；Verifier 视图涵盖 Plan、Observation 与 Execution，约 10% 预算；Coordinator 看到全状态但以轻量摘要形态出现，控制在 1000 token 以内。预算 $\alpha$ 系数在配置文件中可调，但落地遵循"Explorer 最厚、Verifier 最薄"的经验法则——前者承担证据搜集需要更多空间，后者承担结构性判断对噪声敏感反而要保持简洁。**token 预算压缩**是工程实现的真正难点：系统全局维护一个 token 计数器，对历史消息使用 tiktoken 实时估算，对工具结果使用三档处理——原样保留（≤预算）、结构化头尾截断（结构化数据）、LLM 摘要（非结构化文本）。摘要由轻量 Flash 模型完成，仅在结构化截断不可行时触发，避免每条结果都做一次 LLM 调用。

页面上下文与用户上下文不进入历史消息序列，而是作为系统提示词字段直接注入：每个 Agent 的系统提示词都包含 `<actor_role>` 与 `<page_context>` 两个字段，由 MASState.goal 渲染得到。这一约定让 Runtime 内部任意角色都能直接读取上下文，而不必沿着历史消息追溯。

### 4.3.2 多智能体编排与任务状态实现

Runtime 启动时实例化三种编排器：单 Agent ReAct、Plan-Execute 与完整 MAS。编排器之间通过工具集合与 MASState 字段子集保持兼容，使得它们可以在第 5 章的对比实验中公平比较。**编排器选择**由 IntentRouter 与 `get_orchestration_mode` 协同完成：IntentRouter 是一个独立的轻量 LLM Agent，使用 Qwen 3 Flash 这类小模型完成两级路由——第一级是确定性启发式（页面命中、greeting 检测、resume 标志等），第二级才进入 LLM 分类，输出结构化的 JSON `{entry_mode, op_mode, action, confidence}` 对应"对话/帮助/续接"的入口模式与"读/写"的操作模式。两级路由的合并策略是高置信启发式直通、中置信加权合并、低置信 LLM 优先，目的是把意图分类的 token 成本压在 200 token 以内，同时保持 90% 以上的分类准确率。

**MAS 主循环**是 Runtime 内最复杂的子系统。它围绕 MASState 与 Coordinator 状态机展开：每一轮循环首先由 Coordinator 决定下一阶段，然后调度对应角色把结果写回 MASState，最后判断终止条件。每个角色 Agent 都继承自公共基类 `BaseRoleAgent`，其调用流程被固化为三步：以 BuildView 构造该角色的局部视图、渲染模板化的角色提示词并调用配置好的 LLM（带超时与重试）、解析 LLM 输出并写回 MASState。各角色的核心差异体现在提示词模板、LLM 模型选择与允许写入的 MASState 字段三个维度。Planner 与 Explorer 使用低延时的 Flash 模型；Executor 与 Verifier 使用 Thinking 思考型模型——这种"重决策用大、轻执行用小"的异构 LLM 路由是 MOps 成本可控目标的工程基础。

**Explorer 的迭代式证据收集**值得单独描述。其内层循环遵循"每轮最多两个工具、新颖度阈值守护"原则：每次从候选工具队列中选择两个可调用工具（受角色权限过滤）；执行后计算证据新颖度（与已有 Observation 的差异性）；若连续两轮新颖度低于阈值则提前终止；同时检查 `attempted_tool_signatures` 集合避免对相同工具用相同参数重复调用。**Verifier 在收尾阶段对 Observation 做"挑战式验证"**：分别从证据—结论一致性、遗漏排查路径、建议风险、合规四个维度打分，加权得到总分；任一安全维度低于阈值则直接返回 `risk`，证据/遗漏维度低则返回 `missing_evidence` 并附带 replan 提示。这种"显式化失败模态"是 Verifier 区别于一般"质量评分器"的根本差异，也是第 3.3.2 节四类回退分派的工程基础。

**任务状态序列化**采用 dataclass + JSONB 的混合策略：MASState 的所有字段被深度序列化为 JSON 树，时间戳归一化为 ISO 8601，受信任的 Python 对象（如 Action、ToolRecord）实现 `to_dict`/`from_dict` 双向方法。序列化版本号目前是 5，每一次字段不兼容变更都递增版本号；反序列化时若版本号低于当前则触发迁移函数链 `migrate_v3_to_v4 → migrate_v4_to_v5`，确保线上数据库中遗留的旧版本 checkpoint 可在新版 Runtime 上继续续接。

### 4.3.3 工具与安全模块实现

**工具声明解析**采用装饰器模式：每个工具以 `@tool` 形式声明，装饰器除常规 schema 外额外接受 `permission`、`risk`、`confirm` 三个元字段。装饰器在装载时把工具元信息注册到全局注册表，并自动生成 LangChain 标准的 BaseTool 子类。Runtime 在每次请求构造可用工具列表时，从注册表中按 actor role 与 page 过滤——这一过滤即第 3 章 SanitizeTools 的工程落地，结果作为白名单交给 LLM。开发人员只需编写如 `terminate_crater_job(job_id: str)` 这样的 Python 函数，函数内部利用网关代理的方式发出实际 API 调用；服务启动时系统会自动利用 Pydantic 扫描函数类型注解与 docstring，生成符合 OpenAI Function Calling 规范的 JSON Schema，大模型只需理解"终止作业"这一语义而无需关心 Crater API 的认证头组装与重试逻辑。

**工具执行**通过抽象的 `ToolExecutor` 接口完成，生产流量由 `GoBackendToolExecutor` 承担：把工具调用打包为 HTTPS 请求发往网关 `/v1/agent/tools/execute`，按响应状态做三态处理。`success` 是常规读类工具的成功返回，结果写入 MASState 的 tool_records 与 Observation.evidence；`confirmation_required` 是写类工具命中确认流的情形——网关返回该状态码并附带 confirm_id，Executor 据此把对应 action 标记为 `awaiting_confirmation` 并触发 Coordinator 进入 PAUSE 阶段；`permission_denied` 与底层异常被归并为 `error`，Verifier 据此决定是否进入 retry 分支。审计落库由网关在工具执行时同步完成，Runtime 不重复写入——这一约定是"审计真实性"的保证：如果 Runtime 自行写审计，恶意 Runtime 可能伪造记录，因此审计写入主体必须是处于安全边界之内的网关而不是 LLM 推理服务。

**跨平台适配**通过 ToolExecutor 接口的多实现完成。除 GoBackend 之外，Runtime 还实例化两类适配器：LocalExecutor 仅在开发期使用，把工具调用映射到本地函数与内置 Mock；MockExecutor 在评测期使用，按场景快照回放工具返回。三种适配器对 MASState 完全透明——同一个角色 Agent 在生产/开发/评测三种模式下的代码路径一致，仅注入的 Executor 不同。这是工具声明—执行分离的工程红利：迁移到新智算平台时，只需要实现一个新的 Executor 子类，三个编排器、若干角色、几十种工具声明都不必修改。

**权限校验在双侧执行**：Runtime 侧通过 SanitizeTools 在 LLM 注入前裁剪工具白名单；网关侧在每一次工具执行请求到来时再次校验角色与目标对象的所有权（例如普通用户不能停止他人的作业）。两侧权限校验是有意冗余设计——任何一侧的失败都不会让无权操作泄漏到平台。这种"信任内层、防御外层"的双重防线借鉴了网络安全中的纵深防御思想。

**大模型接入与配置适配**：Python Agent 通过 `llm-clients.json` 配置文件管理多个异构 LLM 客户端。实际部署中针对不同模型与网络环境存在两类必要的工程化处理：其一，当对接部署在校园网或内网的远端节点（使用自签名证书）时，通过在默认配置中读取 `verify_ssl: false` 解决 SSL 握手失败；其二，部分兼容模式（如 Dashscope）下的模型严格限制必须使用 stream 模式，否则会返回 `400 Bad Request`，针对这类特定模型在解析配置时动态注入 `streaming: true` 并在代理层接管数据块合并，保障了框架层面接口的统一。**巡检流水线**复用 MAS 主循环，仅替换 Coordinator 的起始提示词与目标对象来源：cron 定时器按 scope（失败作业 / 空闲 GPU / 长时空闲 Jupyter）查询目标集合，对每个目标构造一个 admin 上下文然后调用 MAS 主循环，最终把所有目标的分析结果聚合为巡检报告并通知管理员。巡检与交互对话共用一套编排代码，差别仅在入口与上下文构造——这是"一套核心、多种场景"的工程典范。

## 4.4 质量评估模块

### 4.4.1 评测数据集与场景构造

为防止代码迭代或 Prompt 调优过程中出现能力退化，系统专门实现了离线的质量评估模块（Evaluation Harness）。评测数据集是 MOps 系统迭代的"指南针"——它把"什么是好的多智能体运维助手"从模糊感受转化为可度量的指标。本文的评测数据集由若干个真实场景构成，每个场景以 JSON 描述，包含场景 ID、类别（诊断 / 运维 / 查询 / 提交）、难度等级（1–5）、轮次数、ground_truth（期望工具序列、根因关键词、答复关键要素、期望确认工具）、score_profile（六维评分权重）与难度权重等字段。

**数据采集流程**分两步。第一步通过 SQL 与 shell 脚本分别从 PostgreSQL 业务库与 Kubernetes 集群拉取真实快照——后端 API 真实返回、作业元数据、事件流、告警时序、容器日志等都被原样保留，这些快照构成评测的"地面真相"。第二步通过转换脚本（如 `transform.py`）把快照转化为 MockExecutor 可回放的格式：每个工具调用被映射为一个固定的快照查询键，相同入参在回放时返回完全相同的输出，使得评测结果可严格复现。**标注流程**遵循"三轮收敛"原则：作者根据场景类别与典型问询样本手工编写期望工具序列（区分 must 与 nice-to-have）、答复关键要素与风险动作清单；运行 MOps 系统 3–5 次试跑记录完整推理轨迹；最后由两名研究人员对每条轨迹做盲审打分，分歧场景由第三人裁决。三轮收敛后的数据集既反映"模型应当做对的事"，也反映"模型容易做错的事"，两者结合才能成为驱动迭代的有效信号。这一数据集本身也作为 *harness* 约束智能体开发的优化方向。

### 4.4.2 评分 Profile 与评测执行器

评分系统采用六维加权评分，其中四维由规则计算、两维由 LLM-as-Judge 完成。**Tool F1** 比较实际工具序列与期望序列的集合 F1 分数；**RootCauseHit** 检查根因关键词是否在最终答复中出现；**ChatJudge** 由外部评测 LLM 对答复的连贯性、领域准确性、可操作性打分；**PermissionCompliance** 检查是否调用了无权工具，违规一票否决；**Efficiency** 衡量工具调用数与期望数的偏离；**MultiTurnPass** 用于包含确认环节的多轮场景，检查所有轮次是否合规通过。每个场景的最终得分按下式计算：

$$
\mathrm{OS}_i^{\mathrm{norm}} = \frac{w_{d(i)}\cdot \sum_{m \in \mathcal{M}} w_m\, s_{i,m}}{w_{d(i)}\cdot \sum_{m \in \mathcal{M}} w_m},
\qquad w_{d(i)} = \begin{cases} 1.0,& \mathrm{easy}\\ 1.2,& \mathrm{medium}\\ 1.5,& \mathrm{hard} \end{cases}
$$

其中 $w_m$ 为六维评分权重、$w_{d(i)}$ 为难度权重。难度归一化的目的是避免数据集中难易场景分布的偏置——若 hard 场景占少数却能拉高总分，则评分将过度奖励"避难"的策略；通过 1.0/1.2/1.5 的递增权重，难场景在加权平均中获得更高占比，使指标更贴近"难任务做好"的设计目标。

**评测执行器**同时提供顺序与并行两种执行模式。顺序模式逐场景串行执行，便于调试；并行模式以 asyncio 并发执行多个场景，吞吐率提升约 6 倍，适合大规模回归。每个场景执行完毕后被写入一组 CSV 详细日志，包括场景级摘要与轮次级详情，便于事后下钻。整个评测流水线在第 5 章被反复使用，是支撑实验结论的方法学基础。**质量反馈闭环**通过 `POST /eval/quality/feedback` 端点回收 Portal 中用户的 👍/👎 反馈，触发 ChatJudge（评对话质量）与 ChainJudge（评任务完成度）并行打分，结果写入 `agent_quality_eval` 表形成质量证据库。质量证据库与离线评测数据集相互印证：前者反映"用户认为做得好不好"，后者反映"评测口径认为做得好不好"，二者长期偏差若超出阈值则触发数据集更新——这是评测自身演化的元闭环。

## 4.5 本章小结

本章把第 3 章的设计落地为可在 Crater 智算平台上运行的工程系统。4.1 节确立前端、后端、Runtime 三服务架构与"凭据隔离、状态无栈"两条部署约束；4.2 节给出业务网关层的七张持久化表与三组路由能力，并通过 SSE 流式中继实现"事件转发、异步审计、暂停—续接"三件事一体化，同时阐明前端 AIOps 副屏与限制最大高度的确认卡片在工程上的关键细节；4.3 节给出 Agent Runtime 内部的记忆装载、多智能体编排状态机、工具执行三态返回、跨平台适配与异构 LLM 网络层适配机制；4.4 节给出真实场景评测数据集的构造与六维加权评分的具体实现。整个系统在"安全可控、可水平扩展、跨平台可迁移、可观测"四个维度上达到工程落地要求，并通过质量反馈与离线评测的双闭环驱动持续迭代。下一章将基于本章实现进行大规模离线评测与线上案例分析，验证 MOps 设计的有效性。
