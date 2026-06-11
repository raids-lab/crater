---
applyTo: "backend/**,frontend/**"
---

# 前后端及存储模块开发与审查规范

此文档针对 `backend/`、`frontend/` 及 `backend/internal/storage/` 目录的代码变更提供**审查指引**。

**完整开发规范以 CONTRIBUTING 为权威来源**，审查时据此判断；本文档不重复其全部内容，仅固化最重要的拦截项与审查特有补充。

- 后端：[`backend/CONTRIBUTING.md`](../../backend/CONTRIBUTING.md)
- 前端：[`frontend/CONTRIBUTING.md`](../../frontend/CONTRIBUTING.md)
- 全仓库流程、核心工程规则与跨模块规则：根 [`CONTRIBUTING.md`](../../CONTRIBUTING.md)

## 核心规范 (Core Requirements)

以下为高优先拦截项（完整规则见上述 CONTRIBUTING）：

- **敏感信息**: 严禁硬编码密钥、Token、密码或内网 IP。
- **后端权限路由**: 管理员接口注册到 `Admin` 路由，用户接口注册到 `Protected` 路由。
- **后端存储安全**: 严禁在 `backend/internal/storage/` 或 DAO 层拼接 SQL 字符串，必须参数化查询。
- **后端 API / 错误**: 变更外部 API 必须同步 `swag` 注释；使用 RESTful HTTP 状态码，错误信息为清晰准确的英文。
- **后端验证与运行环境**: 后端构建 / 测试前确认本地 `go version` 符合 `backend/go.mod` / `backend/CONTRIBUTING.md`；`make run` 依赖真实配置、集群和网络环境，缺少配置或管理员凭据时应要求开发者检查，不要让 Agent 反复尝试。
- **作业模板兼容性**: 修改作业创建配置字段、模板序列化或克隆作业相关 payload 时，必须判断是否应阻断旧模板 / 旧导出配置；需要阻断时必须提升对应前端 `MetadataForm*` version，旧配置仍需可用时补充兼容处理。
- **前端公共组件**: 修改被广泛引用的公共组件（尤其 `src/components/ui-custom/`）必须经风险评估，严禁草率变更。
- **前端多语言 / 身份**: 禁止硬编码文本，必须接入多语言方案；新增 / 修改翻译 key 必须使用英文语义 key，并放入合适 domain，不得新增中文 key；新增 / 修改可翻译文案必须同步所有语言的 `translation.json`，保证翻译准确和专有名词一致；身份判断必须用 `useIsAdmin()`。
- **前端非幂等操作确认**: 创建、更新、删除、停止、锁定 / 解锁、配额变更等非幂等操作执行前必须有确认弹出框，说明操作对象和后果；耗时请求还须有 Loading 并禁用重复提交。
- **前端配置说明**: 不易理解的输入项、开关或配置项须在标签 / 标题旁提供帮助图标和 hover tooltip，解释作用、适用场景、关键机制或影响。
- **前端截图**: 前端 / UI 改动的 PR 必须附相应界面截图，展示受影响页面、角色、关键状态或操作结果。
- **文档与配置同步**: 新功能或重大变更同步 `website/` 文档；配置结构变更同步 `charts/`，提升 `charts/crater/Chart.yaml` 的 `version` 与 `appVersion`，并保持二者为完全相同的值，同时同步 `charts/crater/README.md`；前后端 API 变化影响 Chart 应用契约时按 `charts/CONTRIBUTING.md` 提升 minor。

## 优化建议 (Optimization Suggestions)

优化建议（命名、架构解耦、业务码精准化、性能、移动端适配、防重复提交等）详见各模块 CONTRIBUTING；审查时据此提出 SHOULD 级建议。

## 补充说明 (Supplemental Instructions)

### `code-review` (审查助手)
- **高风险组件监控**: 凡涉及前端 `src/components/ui-custom/` 或其他被广泛复用的基础组件变更，必须在 Review 评论中将其列为“核心规范”项。要求开发者在评论中明确确认该变更对现有引用页面的影响范围及兼容性。
- **后端审查力度控制**: 在针对后端代码提出优化建议时应保持审慎（保守态度）。仅在发现明显的架构违规、重大性能隐患或具有高度参考价值的最佳实践时才进行评论，避免由于微小的个人风格差异或非关键性优化导致 Review 评论过于冗余。
