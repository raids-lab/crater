# PR 495：模型部署接入实现与验证

日期：2026-10-04。工作树：`fix/pr495-review`。原生 API 基线：Kthena v1.0.0。

## 已实现的部署组合

| 引擎 | 布局 | 每个角色的执行方式 | KV 传输 |
| --- | --- | --- | --- |
| vLLM | 合并 | 单节点 | — |
| SGLang | 合并 | 单节点 | — |
| vLLM | 合并 | Ray 多节点 | — |
| vLLM | Prefill/Decode | 单节点 | NixlConnector |
| SGLang | Prefill/Decode | 单节点 | Mooncake |
| vLLM | Prefill/Decode | Ray 多节点 | NixlConnector |

这些组合共享 roles 契约。`replicas` 表示服务组数，`roles[].instances` 表示每组角色实例数，`nodesPerInstance` 包含一个 entry 与 N−1 个 worker。Ray profile 使用 TP=每节点 GPU 数、PP=节点数；SGLang 使用自身启动参数。合并单节点默认可选，四种分布式组合由管理员通过 `backendConfig.kthena.enabledProfiles` 逐项启用。Capabilities 返回配置证据及三类 API 的可访问性。

## 前端与接口

创建页按模型、拓扑、角色资源、预检查四步组织，底部保留操作按钮，桌面显示资源摘要。复用 ImageFormField、ResourceFormFields、EnvFormCard、Combobox、FormImportButton/FormExportButton；表单状态、模板 CRUD、校验、序列化分别归属独立文件。四种语言同步更新。

Entry/Worker 支持 CPU、内存、GPU、RDMA、镜像、节点约束、普通环境变量及 Secret 引用。Worker 默认继承 Entry，允许显式覆盖。前后端均校验实例数、节点数、TP/PP、重复参数及模型来源；后端禁止通过高级参数覆盖模型、端口和分布式拓扑。模型查询失败提供重试，隐藏步骤的字段错误集中显示。编辑时锁定名称与模型路由身份。

新 DTO 以 `backendType`、`layout`、`replicas`、`roles` 和模型字段为基础。模板 `schemaVersion=2`，克隆与模板使用相同序列化器。旧 worker DTO 与旧部署兼容路径已移除。Swagger 同步生成。

接口增加 capabilities、preview、scale、update、logs。Preview 返回归一化配置、总请求、队列、排队状态及 ModelServing/ModelServer/ModelRoute。提交重新校验。详情显示角色配置和 group/role/instance/entry-worker 层级、就绪数量、日志、暂停/恢复/扩缩容与编辑入口。

## 原生资源与运行时

ModelServing 管理角色及 Pod；ModelServer 使用部署 UUID、entry=true 和 P/D group 标签选择 HTTP 后端；ModelRoute 使用部署名作为请求路由。后台修复从同一角色模型重新生成网络资源，保留引擎、P/D、connector 和端口。

启动器随 Crater 二进制嵌入生成的 Pod command，使用 argv 启动引擎。Ray entry 启动 head 并等待所需节点/GPU，worker 连接 Kthena 注入的 ENTRY_ADDRESS；形成集群超时会退出并清理 Ray。worker 探针检查本机 Ray 节点是否存活，entry 使用引擎 HTTP 健康检查。恢复策略为 ServingGroupRecreate。

同一 Ray 实例通过包含 group/role-id 的 `matchLabelKeys` 反亲和约束分布到不同物理节点。所有角色使用 Volcano 调度器和 Crater 账户队列，完整角色实例数量参与 gang。账户 toleration、镜像拉取配置、模型权限和存储挂载应用到所有 entry/worker。

平台模型挂载授权 PVC 子目录，外部模型每 Pod 下载到 emptyDir。外部 hf/ms 分布式部署要求固定 40 位 commit revision，S3 分布式部署先保存为平台模型。凭据只持有 Secret name/key，后端检查用户、账户及 key；Hugging Face 下载使用 HF_AUTH_TOKEN。

## 资源准入与生命周期

资源汇总为 `groups × Σ[instances × (entry + workers × worker)]`，每个资源名称独立求和。Pod 资源遵循 Kubernetes app/init/restartable-sidecar/overhead 请求计算。统一 Workload、配额、列表摘要、预排队和活跃部署计数均消费该模型。

原生部署提交、普通任务提交和预排队唤醒使用同一账户/用户准入锁。PostgreSQL 使用事务 advisory lock 跨副本串行化；竞争者使用 try-lock 并在事务外重试，避免等待锁耗尽连接池。成功写入 Kubernetes、尚未被数据库观察器记录的 Volcano Job 也计入预留，避免观察延迟重复放行。

扩容与恢复排除自身旧预留后重新准入；已有运行部署遇到扩容额度不足时保留原状态。暂停以零组保存期望副本数。活动模板更新保持部署 UUID、路由身份及资源总量，Quantity 按数值比较；资源预算改变需先暂停。引擎、布局、端口、servedModel 和 gang 角色实例数为不可变字段，需变更时克隆。

计费启用时使用计量 gate/finalizer；关闭计费的部署不依赖计量门控。推理请求率、错误比例和端到端延迟以 Kthena Router 为边界；vLLM/SGLang 引擎指标分别按角色采集，P/D 总 token 吞吐取 Decode。Chart 提供 entry PodMonitor 和 Router ServiceMonitor。

## 实际验证结果

- Go 定向测试通过：handler、kthena、service、vcjob、prequeuewatcher、monitor；使用 fake Kubernetes 客户端及 SQLite overlay。
- 新增六组合 render/read/repair 与资源汇总回归，更新身份保留、活动资源变更拒绝、暂停更新、跨账户 Secret/key 校验、SGLang 与 Router 指标边界测试。
- Python 启动器 4 项测试通过：参数保持、Ray head 等待、超时清理、worker 定位和终止信号转发。
- 六组合的 18 个原生资源及基础契约的 3 个资源均通过 Kthena v1.0.0 CRD OpenAPI schema 校验。
- 增量 Go lint 为 0 issues；后端编译通过。前端 TypeScript、定向 ESLint、四语言格式检查及生产构建通过。
- Helm lint 通过；启用两类 monitor 的 Helm template 生成成功。
- Chrome + Playwright fixture：四步 P/D 创建、预检查和确认框通过；页面切换 P/D + Ray、启用独立 Worker、切换 SGLang P/D 均提交正确配置；六组合模板/克隆 codec 往返保持配置、重复参数被拒绝。390px 窄屏无横向溢出、底部操作按钮位于可见视口，浏览器无未处理异常；确认框与摘要使用同一资源总量。

本地浏览器记录位于 `frontend/output/playwright/serving-topology-report.json`。这些验证使用接口 fixture；分布式 profile 的启用由管理员在目标 GPU 集群完成模型加载、Ray 通信、KV 传输、故障恢复及与训练任务混合调度检查后配置。部署使用方式见用户文档与 Chart README。

## 本轮审查问题修复（2026-10-04）

1. 配额使用 `max(期望资源, 未终止 Pod 资源) + 正在终止的 Pod 资源`；按部署 UUID 汇总，根对象已删除的残留 Pod 继续占用配额。替换部署预留时只抵扣可复用的活动资源，正在终止的资源保持独立占用；作业数量不重复计算同一部署。
2. Ready 要求所有服务组、角色、实例的 Entry 与 Worker 齐备且就绪。缺失 Decode、缺失 Ray Entry/Worker、部分组未创建、Pod 正在终止均不能显示 Ready。
3. GPU 和扩展资源统一按 Kubernetes Quantity 校验并规范化。`1`、`1e0`、`1000m` 均作为一个 GPU 传入 Ray，拒绝非整数分配。
4. 克隆/编辑恢复节点 include/exclude 控件；额外 selector 独立保留并可编辑删除。根据 GPU 对应的注册网络资源恢复 RDMA 型号和数量，关闭或替换会移除原请求；其他扩展资源显示在可编辑列表中。复用资源控件的费用预览同时采用 RDMA 实际数量。
5. 预检查通过 `mode=create|update` 明确操作。新建检查名称、账户/用户队列、提交策略和配额；编辑复用实际更新的不可变字段、资源变更及身份保留规则，并保持暂停状态。预检查不创建队列、不写入部署；新建提交仍在准入锁内重查容量。
6. `kthena.runtimeImages` 按 engine/layout/execution 配置允许的固定版本镜像或摘要。配置项替换该 profile 的默认镜像；空列表禁用该 profile。创建、编辑、恢复和预排队激活检查 Entry/Worker 镜像。能力接口返回镜像列表和固定的 NixlConnector/Mooncake 契约，前端显示相应限制。

验证结果：相关六个 Go 包通过回归；增量 Go lint 0 issues；后端编译和 Swagger 生成通过。四个 Python 启动器测试通过；21 个生成资源通过 Kthena v1.0.0 CRD OpenAPI schema 校验。前端 TypeScript、定向 ESLint、四语言格式检查和生产构建通过。新增三项表单测试通过，可在 `frontend` 运行 `node hack/test-serving-form.mjs`。Chrome fixture 覆盖六种拓扑往返、P/D/Ray/SGLang 预检查，以及克隆后关闭节点约束和 RDMA：提交移除对应请求，保留其他约束；390px 视口无横向溢出，浏览器无未处理异常。

## 迁移程序构建修复

CI、GitHub 发布流程、GitLab 构建和本地 Makefile 统一以 `./cmd/gorm-gen/models` 包作为构建或运行入口，使迁移注册和拆分后的实现一并参与编译。`CGO_ENABLED=0` 下 Linux amd64、arm64 的迁移程序和控制器均构建成功，迁移包回归测试通过。

## 上游依据

- [Kthena v1.0.0](https://github.com/volcano-sh/kthena/releases/tag/v1.0.0)：2026-10-04 GitHub API 核对为 latest，发布于 2026-07-16。
- [ModelServer 类型](https://github.com/volcano-sh/kthena/blob/v1.0.0/pkg/apis/networking/v1alpha1/modelserver_types.go)：vLLM/SGLang、PDGroup、KVConnector、单 workloadPort。
- [Role 与 entry/worker](https://github.com/volcano-sh/kthena/blob/v1.0.0/pkg/apis/workload/v1alpha1/servinggroup_types.go)。
- [Ray 示例](https://github.com/volcano-sh/kthena/blob/v1.0.0/examples/model-serving/multi-node.yaml)。
- [vLLM P/D + Ray 示例](https://github.com/volcano-sh/kthena/blob/9434282/examples/model-serving/vllm-pd-multinode.yaml)：上游 main 的组合示例；使用 v1.0.0 已有 CRD 字段。
- [SGLang P/D](https://github.com/volcano-sh/kthena/blob/v1.0.0/examples/model-serving/sglang-pd-disaggregation.yaml)。
