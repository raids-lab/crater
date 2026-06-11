---
applyTo: "charts/**"
---

# Helm Chart 开发与审查规范

此文档针对 `charts/` 目录下的 Helm Chart 配置及相关文档变更提供**审查指引**。

**完整开发规范以 [`charts/CONTRIBUTING.md`](../../charts/CONTRIBUTING.md) 为权威来源**（版本管理、配置文档、注释、安全与验证）；本文档不重复其全部内容，仅固化最重要的拦截项与审查特有补充。

## 核心规范 (Core Requirements)

以下为高优先拦截项（完整规则见 `charts/CONTRIBUTING.md`）：

- **版本同步**: 凡涉及模板、依赖、配置逻辑，或新增 / 删除 / 重命名 / 改变配置项行为或默认值，必须同步更新 `charts/crater/Chart.yaml` 中的 `version` 与 `appVersion`，并保持二者为完全相同的值。
- **版本级别**: 仅 `values.yaml` 配置项变更可提升 patch；前后端 API 变化并影响应用契约时必须提升 minor；不要主动提升 major，除非维护者明确决定。
- **GitHub tag**: `version` / `appVersion` 为发布而变化时，须提醒开发者创建对应 GitHub tag（因两者相同，通常为 `v<version>`）。
- **配置文档同步**: 修改 `values.yaml` 参数后，必须同步更新 `charts/crater/README.md`；新增配置参数须包含含义描述、默认值与必要的使用说明 / 示例。
- **注释规范**: `values.yaml` 新增配置项必须包含准确、清晰、能辅助用户理解用途与期望值的**英文注释**。
- **安全**: 严禁提交真实密钥、Token、密码、证书、kubeconfig、内部专用 endpoint 或生产凭据。

## 优化建议 (Optimization Suggestions)

文档自动化（`helm-docs`）、配置命名、注释清晰度、resource requests / limits、安全敏感默认值与模板稳健性等建议，详见 `charts/CONTRIBUTING.md`；审查时据此提出 SHOULD 级建议。
