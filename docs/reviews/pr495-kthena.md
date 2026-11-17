# PR #495：迁移到 Kthena 原生资源

检查日期：2026-10-03。Crater 基线：`1739c3af21fe146ce8192cdf93b15e3d93cb3a55`。
修改保留在本地 `fix/pr495-review` 工作树，未提交、推送或操作集群。

## 最终实现

新部署直接创建以下三个 v1alpha1 资源，不再生成 ModelBooster：

| 资源 | 职责 | 所有权 |
| --- | --- | --- |
| ModelServing | vLLM Pod 模板、模型存储、资源、调度约束及副本数 | 部署根资源，保留 Crater 用户/账户标签 |
| ModelServer | 按部署唯一标识选择 Pod，配置 engine 模型名和端口 | ownerReference 指向 ModelServing UID |
| ModelRoute | 保留以部署名称为键的调用路由 | ownerReference 指向 ModelServing UID |

- 创建顺序为 ModelServing → ModelServer → ModelRoute。发生后续创建失败时，使用 UID 前置条件回滚本次新建的根资源，子资源由 Kubernetes 垃圾回收；不会覆盖或接管同名外部资源。请求取消后仍给回滚保留独立的有限时长上下文。
- 删除以授权后的根资源 UID 为准，使用 Kubernetes 后台级联回收。同名同内容请求可重试恢复；后台在 leader election 下每 15 秒修复缺失的自有网络资源，详情也提供手动修复入口。
- 查询、详情、原始资源、会话、OpenAI 代理、作业列表及门户共享对原生布局的读取。新部署的原始资源接口返回 Kubernetes List，包含根资源及其有明确所有权的网络资源。
- 仅支持 ModelServing、ModelServer、ModelRoute，删除 ModelBooster 的查询、创建检查、关联回退和测试夹具。缺少 ModelServing CRD 时明确返回错误。
- 运行状态读取 ModelServing 的 observedGeneration 和 availableReplicas；观察到旧代状态或仅部分副本可用时，不直接报告完整就绪。

## 与 Volcano 的共同调度

ModelServing 和 Pod 模板均指定 `schedulerName: volcano`。根资源设置 `scheduling.volcano.sh/queue-name`，值来自普通作业使用的 `vcqueue.ResolveJobQueueName`：非公共账户进入 `q-a<账户ID>-u<用户ID>`，公共账户沿用其队列名。提交前复用账户/用户队列创建函数，并拒绝缺少 PodGroup API、关闭的叶队列或父账户不匹配的用户队列。

Kthena v1.0.0 的 [PodGroup manager](https://github.com/volcano-sh/kthena/blob/v1.0.0/pkg/model-serving-controller/podgroupmanager/manager.go) 根据每个 ServingGroup 创建 PodGroup，复制上述 queue 注解，并根据 `gangPolicy.minRoleReplicas`、Pod 数量及容器 requests 计算 minMember/minResources。Crater 显式设置 server 角色的最小副本数，CPU、内存及 GPU 同时写入 requests/limits。

下载 init container 的 CPU/内存预算与 engine 相同，不额外请求 GPU，因此其执行阶段也受资源限制，且 Kubernetes 的 max(init, app) 资源需求与 Kthena 对普通容器 requests 的求和一致。资源竞争由 Volcano 的节点资源与队列策略处理；不足时等待调度，而不是在 Crater 内进行非原子的“剩余资源预占”。最终效果仍取决于实际集群的 Volcano/device plugin 配置。

## Pod 模板与运行范围

每个角色使用单 Pod，支持多个 serving group 和角色副本。API 显式拒绝 `worker.pods > 1`；多节点 Ray 和 P/D 分离未在本次实现。

平台模型经数据集权限校验后，以只读 subPath 挂载授权目录；外部 hf/s3/ms 来源下载到 Pod 独立 emptyDir，不允许用户指定 PVC 或 hostPath。下载器默认固定为 `ghcr.io/volcano-sh/downloader:v1.0.0`，管理员可通过 `backendConfig.kthena.downloaderImage` 设置镜像地址。它与 Crater 的模型下载 Job 镜像是不同配置。

直接启动 vLLM engine，ModelServer 路由到 engine 端口，不需要 ModelBooster 注入的 runtime sidecar。engine、readiness/startup probe 和 ModelServer 使用同一端口，因此支持合法的自定义端口。命令使用参数数组，不通过 shell 拼接。

资源构造代码已拆分：

- `backend/internal/handler/inference_builder.go`：公共请求校验、身份元数据和 worker 资源。
- `backend/internal/handler/inference_native_builder.go`：三个原生资源的构造。
- `backend/internal/handler/inference_lifecycle.go`：创建、回滚和原始资源读取。
- `backend/internal/kthena/resources.go`：ModelServing 查询和规范化读取，供部署与作业接口共用。

保留并扩展了上一轮修复：JSON 兼容的 affinity/tolerations、显式 GPU requests/limits、资源数量/URI/环境变量校验，以及限定模型推理操作的代理端点。前端文案、Swagger、Helm 配置和用户文档同步更新。

## 上游兼容性

最新正式版是 [v1.0.0](https://github.com/volcano-sh/kthena/releases/tag/v1.0.0)，发布于 2026-07-16。检查时主分支为 `943428223c344678d37f365aafab9715af8284ad`（2026-09-30）。实现使用 v1.0.0 已发布的三类 CRD，不依赖未发布主分支功能。

主分支的 [ModelBooster API](https://github.com/volcano-sh/kthena/blob/943428223c344678d37f365aafab9715af8284ad/pkg/apis/workload/v1alpha1/model_booster_types.go) 声明从 v1.1 起弃用、最早 v1.5 移除，并推荐这三类原生资源。主分支还增加了 obs:// 来源、独立 metrics 端口及 tokenizer service 等能力，本次未启用这些未发布能力。

## 验证

- Go 聚焦测试：`internal/kthena`、`internal/handler`、`internal/handler/vcjob`、`cmd/gorm-gen/models` 中 Kthena/Inference/Workload 测试通过。覆盖三资源关联、端口一致性、PVC 路径、原生会话调用、原生删除、冲突回滚、部署名称冲突、缺少 ModelServing CRD、观察失败传播及代际状态。
- handler 测试仍使用临时 Go overlay，仅将 `dao/query.GetDB` 的包初始化连接替换为内存 SQLite；业务代码没有增加测试分支，overlay 不进入仓库。没有验证 PostgreSQL 行锁语义。
- 测试生成的实际三个资源，通过 Kthena v1.0.0 CRD 的 OpenAPI schema 校验。可设置 `CRATER_KTHENA_MANIFEST_DIR` 运行 `TestKthenaNativeResourceContract` 导出同一批对象。这项离线检查不运行 Kubernetes 默认值填充、CEL 或 admission webhook。
- 聚焦 Go 静态检查返回 0 issues；Swagger 已重新生成；Helm lint、前端 TypeScript/ESLint 和四种语言翻译文件格式检查通过。
- 未连接真实集群，未执行 GPU 推理、镜像拉取或 Kubernetes 垃圾回收端到端验证。

## 本轮进一步修复

- Serving 与普通任务共用提交封禁、余额准入和活跃任务数检查；配额计算加入同账户原生 Serving 的全部实际副本。资源不足的部署以零副本留在预队列，后台重试准入。配额观察使用 APIReader，避免 informer 滞后漏算已放行资源。
- Pod 继承平台镜像拉取 Secret、当前账户 toleration、镜像架构约束；禁止客户端伪造账户 toleration 和通配容忍。
- 聊天业务移至 service/inference：短事务领取持久化租约，事务外推理，短事务提交完整结果；请求摘要保证 ID 重放内容一致，过期租约支持恢复，旧请求不能提交。流式、取消、重试接入前端，取消及中断不保存部分输出。
- 列表仅读取 ModelServing 摘要；详情按名称读取子资源和 Pod；事件、日志按需加载，日志限制 64 KiB。读取失败可见。
- 删除 localStorage 会话迁移和不用的前端调用封装；拆分创建页、详情页、聊天状态和诊断组件。修复详情错误态、CPU quantity 克隆、选择器及副本保存、账户查询隔离、curl 引号和破坏性操作确认。
- 新增聊天字段 migration 与生成 DAO；更新 Swagger、四语文案和用户文档。

## 验证边界

聚焦 Go 回归覆盖存储与账户隔离、配额副本汇总、网络修复、单数据库连接推理、幂等重放、取消与过期租约恢复、SSE 截断，以及 migration 升降级。静态检查与后端构建通过。真实集群和 PostgreSQL 并发尚未运行。

持续计费和推理指标现已接入。每 15 秒观察 Pod 的 CPU/内存/GPU 请求与实际开始、结束时间，按 Pod UID 持久化计量；现有周期结算、价格调整、账户额度发放和删除结算共用余额规则。重复观察和结算不会重复扣费，关闭计费不累计欠费。该计费按资源占用时长，请求与 token 指标不另行收费；价格边界精度受观察周期影响。

Pod 模板包含计量 scheduling gate，后端原子安装计费 finalizer 并解除自己的 gate，再允许 Volcano 调度。部署要求 Kubernetes 1.27+（启用 PodSchedulingReadiness）和支持 scheduling gates 的 Volcano；验证依据为 Volcano v1.12.0 的 Pod Scheduling Readiness 设计。后端停止时新 Pod 保持 gated，删除中的 Pod 等待恢复结算，禁止以手工移除 finalizer 代替正常清理。

详情页展示近五分钟成功请求率、生成 token 吞吐、TTFT/e2e P95、等待/执行请求数、HTTP 错误比例和累计结算。Prometheus 查询只选择当前租户部署的 Pod，无样本显示缺失值；监控不可用时仍返回账务摘要。可启用 Helm PodMonitor，或配置 annotation 抓取并保留 namespace/pod 标签。原有 Grafana 资源监控保留。

滚动升级与自动扩缩容仍未实现。PR 与 main 的冲突尚未处理，本地修改未提交或推送。

## 2026-10-04 多引擎与分布式部署

roles 契约、四步前端、vLLM/SGLang、Ray、P/D 和 vLLM P/D+Ray 已落地。具体配置、代码职责与本轮验证结果见 [模型部署实现记录](pr495-serving-design.md)。本次改动保存在本地工作树，尚未提交推送。
