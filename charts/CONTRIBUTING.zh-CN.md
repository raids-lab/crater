[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

# 为 Crater Helm Charts 贡献

本文档覆盖 `charts/`，尤其是 `charts/crater` Helm Chart。请先阅读仓库根 [CONTRIBUTING](../docs/zh-CN/CONTRIBUTING.md)，了解全仓库流程、分支规则、PR 要求和跨模块约束。

修改 Chart 元数据、`values.yaml`、Helm 模板、依赖、生成的 Chart README 或发布版本时，使用本文档。

## Chart 文件

- `charts/crater/Chart.yaml`：Chart 元数据与发布版本字段。
- `charts/crater/values.yaml`：面向用户的 Chart 配置。
- `charts/crater/templates/`：Helm 渲染出的 Kubernetes manifests。
- `charts/crater/README.md`：由 Chart 元数据与 values 生成的参数文档。

## 改动流程

1. **判断是否影响用户配置**。新增、删除、重命名配置项，或改变配置项行为 / 默认值，都视为配置变更。
2. **同步修改 Chart 与文档**。修改 `values.yaml` 参数后，必须同步反映到 `charts/crater/README.md`。
3. **按需提升发布版本字段**。模板、依赖、配置逻辑或配置行为变更都需要提升版本。
4. **验证渲染与文档**。PR 会触发 `.github/workflows/helm-chart-validate.yml`（`helm lint`、`helm template`、对实际影响发布的 Chart 变更执行版本递增检查、打包 smoke test）；合入 `main` 后由 `.github/workflows/helm-chart-publish.yml` 自动发布 Chart 到 GHCR OCI。本地可按需运行 `helm lint` / `helm template` 预检。

## 版本管理

`charts/crater/Chart.yaml` 使用一套共享发布版本。`version` 与 `appVersion` 在 Crater 发布中必须始终保持完全相同的值，不要拆分理解为 Chart 版本和应用版本两套编号。

- 修改 Chart 模板、依赖或配置逻辑时，必须同时将两个字段提升到同一个新值。
- 新增、删除、重命名配置项，或改变配置项行为 / 默认值时，必须同时将两个字段提升到同一个新值，并同步更新 `charts/crater/README.md`。
- `version` / `appVersion` 共用同一个语义化版本值。
- 仅修改 `values.yaml` 配置项时可提升 patch 版本，除非该变更同时影响前后端 API 兼容性。
- 前端 / 后端 API 发生变化并影响 Chart 部署出的应用契约时，必须提升 minor 版本。
- 不要主动提升 major 版本，除非维护者明确判断这是一次 major breaking release。
- 发布版本变化时，提醒开发者在 GitHub 上创建对应 tag，通常为 `v<version>`。

## Values 与文档

- 新增配置参数必须包含清晰的含义描述、默认值，以及必要时的使用说明 / 示例。
- `values.yaml` 中新增配置项必须包含准确、清晰、能帮助用户理解用途与期望值的英文注释。
- 推荐通过 `helm-docs` 维护 `charts/crater/README.md`。
- 不得让生成的 Chart 文档与 `Chart.yaml` 或 `values.yaml` 漂移。

## 质量与安全

- 配置项命名保持语义化，并与既有命名风格一致。
- 模板应简洁、稳健、可读；当 Helm helpers / pipelines 能让渲染结果更安全或更清晰时，优先使用。
- 关注资源 requests / limits、安全敏感默认值，以及与 Chart 已管理 Kubernetes 对象的兼容性。
- 严禁提交真实密钥、Token、密码、证书、kubeconfig、内部专用 endpoint 或生产凭据。
- 示例中的占位密钥必须明显是假的，例如 `<MUSTEDIT>` 或 `<MASKED>`。

## 提交 Chart 改动前

- PR 改动 `charts/**` 时会自动运行 **Validate Helm Chart** workflow；合入 `main` 后 **Publish Helm Chart** workflow 会自动打包并推送到 `oci://ghcr.io/raids-lab/crater`。
- 根 pre-commit hook 会在暂存 `charts/**` 变更时委托执行 `cd charts && make pre-commit-check`。只有实际影响 Chart 发布内容的文件变更时，才要求提升共享 `version` / `appVersion`（`Chart.yaml`、`values.yaml`、模板、依赖或 `Chart.lock`）。
- 本地可按需运行 `helm lint crater/`、`helm template crater crater/ --dry-run` 预检；`charts/` 当前没有专属 `make` target。
- 若运行了本地检查，在 PR 描述中写清具体命令与结果。
- 任一版本字段变化时，确认 `version` 与 `appVersion` 完全相同。
- 确认 `charts/crater/README.md` 已反映 `values.yaml` 变更。
