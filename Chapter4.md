# 第4章 面向智算平台的智能运维系统实现

在第3章给出了平台无关的多智能体协作框架（Mops）设计之后，本章将详细介绍该框架在真实的智算平台——Crater 上的工程落地与系统实现。本章首先介绍 Crater 平台底座及其与智能体服务的集成关系，随后阐述系统的前后端微服务架构，并重点对前端交互视图、后端业务模块以及 Python 智能体服务中针对 Crater API 的工具封装、安全控制和配置管理等核心实现细节进行剖析。

## 4.1 基于 Crater 平台的系统运行底座

### 4.1.1 Crater 智算平台简介

Crater 是一个为大规模 AI 训练与推理提供基础设施管理的智算平台。它向下对接底层的异构算力集群（如 GPU 节点、存储集群），向上为算法工程师和平台管理员提供作业调度（Jobs）、镜像管理、资源配额以及全链路的日志与监控服务。在 Crater 平台中，所有的资源与作业状态均通过其原生数据库（如 MySQL/PostgreSQL）进行持久化，并通过一组标准的 RESTful API 向外提供控制面服务。

### 4.1.2 智能体与 Crater 底座的集成模式

本课题所实现的智能运维系统，并非替代 Crater 现有的核心控制面，而是作为一种“智能化副屏（Copilot）”与“后台大脑（Brain）”插件式地挂载于 Crater 平台之上。
智能体服务获取 Crater 平台状态或对其进行变更，完全依赖于 Crater 暴露的 OpenAPI 接口。例如，Agent 规划出的“查询节点资源”操作，在底层会转化为对 Crater `/api/v1/nodes/metrics` 接口的 HTTP GET 请求。这种“旁路集成”模式，既保证了 Crater 核心调度链路的稳定性，又赋予了平台强大的自然语言交互与自动化排障能力。

## 4.2 系统整体微服务架构实现

系统在物理部署上采用了“前后端分离、控制平面与智能代理分离”的三端微服务架构。

1. **Vite/React 前端交互服务：** 
   作为用户直接操作的入口，不仅包含了 Crater 原生的控制台页面，还深度嵌入了 AIOps 智能交互组件。前端通过反向代理将 `/api/v1/agent/*` 的请求统一路由至后端网关。
2. **Go 后端业务网关：**
   作为系统的控制平面，采用高性能的 Go 语言编写。它不仅负责 Crater 平台原有的 CRUD 业务，还作为 Agent 请求的中继网关。所有针对智能体的调用，都必须经过 Go 后端的 JWT 鉴权与 RBAC 权限校验，随后通过 HTTP/SSE 协议透传至下游的 Python 服务。
3. **Python 智能体服务 (Python Agent)：**
   核心的多智能体协作逻辑（如 LangGraph 状态机编排）、大语言模型推理以及对 Crater API 的封装调用均在此服务内实现。Python 服务监听内部端口（如 `8000`），并配置为仅允许来自 Go 后端网关（默认端口 `8098`）的请求访问。

在开发与联调环境中，主仓库通过软链接机制（如执行根目录的 `make config-link` 调用的 `hack/config.sh`）打通了三端的配置文件。例如，Go 后端强依赖于 `backend/.debug.env` 与 `backend/etc/debug-config.yaml`，缺失时会在配置加载阶段直接熔断，从而保证了三端协同开发时环境配置的强一致性。

## 4.3 前端视图与后端模块实现

为了清晰地实现业务逻辑解耦，本节将详细说明前端页面与后端网关的具体模块划分与实现细节。

### 4.3.1 前端视图与组件模块划分

前端部分主要基于 React 框架实现，按照功能边界划分为三大视图模块：

**1. 基础资源与作业管控台 (Native Console)**
这是 Crater 平台原有的业务视图，包括集群总览、作业列表（Job List）、节点详情等页面。为了支持“所见即所问”的上下文注入，我们在这些页面中埋入了路由监听器。当用户在 `job_detail` 页面唤出 Agent 时，前端会自动捕获当前的作业 ID（Entity ID）与页面路径，将其打包在请求 Payload 中发往后端，作为 3.4 节设计的“页面上下文”。

**2. AIOps 智能副屏与对话流模块 (AIOps Chat Panel)**
该模块以侧边栏或悬浮窗的形式存在。为了实现类似 ChatGPT 的丝滑体验，对话接口采用了 Server-Sent Events (SSE) 协议，实时渲染由 Python Agent 推送的 Token 流。
针对多语言（i18n）支持，前端与后端的 `resputil` 响应包约定了 `msgKey` 字段协议。当出现业务异常（如“无法定位该作业”）时，后端返回包含 `code`、英文 `msg` 与 `msgKey` 的响应。前端拦截后，优先利用 `msgKey` 在本地语言包中匹配对应的中文或多语言提示，确保了界面语言的一致性。

**3. 人机协同与安全交互组件 (HITL Components)**
当 Agent 尝试执行中高风险的 Crater API 时，前端会暂停对话流，渲染 `ConfirmActionCard`、`ParameterReviewCard` 或 `BatchConfirmCard` 等交互卡片。
**实现细节：** 由于 Agent 返回的作业参数或配置内容长度不可控，极易撑破侧边栏布局，前端对此类卡片统一实施了“限制最大高度 + 内部滚动”的安全布局策略。文本采用 `[overflow-wrap:anywhere] break-words` 强制换行；在表单模式下，利用 `flex flex-col max-h-[85vh]` 布局限制弹窗整体高度，并为中间的参数配置容器设置 `flex-1 overflow-y-auto`，确保 Header（操作标题）与 Footer（确认/拒绝按钮）始终固定在可视区域内。

### 4.3.2 后端业务网关模块划分

Go 后端在智能运维系统中扮演着“交通枢纽”与“安全门”的角色，主要分为以下三大模块：

**1. 统一鉴权与权限拦截模块 (Auth & RBAC)**
所有发往 Python Agent 的请求在进入网关时，首先经过 JWT 中间件解析出用户身份。随后，权限拦截器会查询 Crater 的原生 `Users` 与 `Role_Permissions` 数据库表，校验当前用户是否具有操作对应集群资源的权限。权限校验失败的请求将直接在 Go 层面被拒绝（HTTP 403），防止恶意指令消耗昂贵的 LLM 推理算力。

**2. Agent 中继与异常处理模块 (Agent Proxy)**
Go 后端负责将前端的 SSE 请求代理转发给 Python Agent（配置中的 `agent.serviceURL` 默认指向 `http://localhost:8000`）。
**实现细节：** 在链路异常排查中，如果 Go 后端自身连接 Python 服务超时或被拒绝，它会生成状态码 500 而非直接透传 502。因此，若前端观察到 502 错误，通常意味着前端代理（Nginx/Vite Proxy）到 Go 后端的链路出现了问题；若是 500 错误，则需优先排查 Python Agent 进程是否存活或 LLM 远端推理是否超时。

**3. 审计日志与快照持久化模块 (Audit & Checkpoint)**
为满足安全生产要求，后端利用 ORM 框架对接 MySQL 数据库，设计了 `AuditLogs` 与 `AgentCheckpoints` 表。当 Python Agent 发出挂起指令（要求用户确认）时，Go 后端会将接收到的多智能体状态对象（MASState）序列化并存入 `AgentCheckpoints` 表，生成唯一的 `ConfirmationID`；当用户在前端点击确认后，后端通过该 ID 唤醒快照，实现断点续接，并将操作记录写入 `AuditLogs` 表供在线审计页面检索。

## 4.4 智能体服务与 Crater 工具集成实现

Python 智能体服务是多智能体协作框架的真正执行者。它通过 LangGraph 框架与 Pydantic 数据验证，将第 3 章的理论设计转化为可运行的代码。

### 4.4.1 针对 Crater 的工具封装与调用实现

在第 3.5 节的工具体系设计指导下，我们通过 Python 的反射机制实现了工具声明与执行的解耦。
我们编写了 `@tool_declare(risk_level="High")` 等自定义装饰器。开发人员只需编写如 `terminate_crater_job(job_id: str)` 的 Python 函数，该函数内部利用 `requests` 库调用 Crater 的原生 REST API。服务启动时，系统会自动利用 Pydantic 扫描函数的类型注解与 Docstring，生成符合 OpenAI Function Calling 规范的 JSON Schema 白名单。
这种封装使得大模型只需理解“终止作业”这一语义，而无需关心 Crater 平台 API 的认证头（Headers）组装与网络重试逻辑，极大降低了模型的认知负担。

### 4.4.2 记忆管理与多智能体编排落地

**1. 状态机与编排实现：**
我们使用 LangGraph 将协调器（Coordinator）、规划器（Planner）等角色实例化为图节点（Nodes），利用条件边（Conditional Edges）实现状态流转。全局上下文被定义为强类型的 `MASState` 字典，确保了轮次（round）、执行计划（plan）与观察证据（observation）在不同节点间传递的类型安全。

**2. 大模型接入与配置适配细节：**
Python Agent 通过 `llm-clients.json` 配置文件管理多个异构的 LLM 客户端。在实际部署中，针对不同的模型与网络环境，我们在初始化 `ChatOpenAI` 客户端时实施了特殊处理：
- **自签名证书处理：** 当对接部署在校园网或内网的远端 Qwen3.5 节点（如 `*.gpu.act.buaa.edu.cn`）时，由于使用了自签名证书，通过在 `default` 配置中读取并注入 `verify_ssl: false`，解决了 SSL 握手失败的问题。
- **流式输出强制适配：** Dashscope 兼容模式下的某些模型（如 `glm-4.5`）严格限制必须使用 Stream 模式，否则会报 `400 Bad Request`。我们在解析配置时，针对这类特定模型动态注入 `"streaming": true` 参数，并在代理层接管了数据块的合并，保障了框架层面的接口统一。

## 4.5 质量评估模块实现

为防止在代码迭代或 Prompt 调优过程中出现能力退化，系统专门实现了离线的质量评估模块（Evaluation Harness）。

### 4.5.1 评测数据集与 Crater_Bench 构造

为了保证评测的真实性，我们建立了独立的 `dataset` 模块。评测数据不依赖人工臆造，而是通过 Python 脚本直连 Crater 平台的真实数据库与 API 进行采集。
**实现细节：** 数据采集脚本会调用 Crater API 获取 `all_jobs` 列表，并根据特定的分布策略（如随机抽取 50 个近期失败作业、20 个排队中作业、10 个运行中作业）抓取其详细日志与节点状态。随后，`transform.py` 脚本会对这些真实数据进行脱敏处理，打包生成标准化的 `crater_bench` 场景数据集，每个样本包含了预设的用户提问（Query）与期望的工具调用链路（Ground Truth）。

### 4.5.2 评测执行与评分机制

评测执行器会隔离运行 Agent 工作流，并根据第 3 章公式 5.1 实现的综合加权评分逻辑（Weighted Score），对智能体的表现进行量化打分。系统预置了多种评分 Profile，包括工具调用的精准度（参数匹配率）、人机协同阻断成功率以及基于 LLM-as-a-Judge 的最终回答相似度。通过这种自动化的回归测试机制，研发团队可以直观地观察到框架调整对不同风险等级任务的影响，为系统的持续迭代提供了科学的数据支撑。

## 4.6 本章小结

本章详细阐述了面向智算平台的智能运维系统在真实 Crater 平台上的工程实现细节。首先介绍了 Crater 底座的集成模式与三端微服务架构。随后，深入拆解了前端视图（AIOps 副屏、限制高度的 ConfirmActionCard）与后端网关（JWT 鉴权、msgKey 映射、502 链路排查、审计持久化）的模块划分。在 Python 智能体服务方面，着重讲解了 Crater API 的工具化封装、LangGraph 状态机编排以及针对远端大模型证书和强制流式要求的网络层适配。最后介绍了基于真实平台数据构建的 `crater_bench` 评测集与自动化打分系统。这些工程实践有效验证了前一章设计方案的合理性与可用性。
