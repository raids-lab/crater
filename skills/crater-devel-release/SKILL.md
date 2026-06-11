---
name: crater-devel-release
version: 0.1.0
description: "Crater 发布（开发者侧）：charts/ Helm Chart 开发与版本管理、values/README 同步，以及经 CI 产出的镜像与 Chart 发布物。用户修改 charts/、Helm 模板/values、Chart 版本或准备发布产物时使用；开始前须应用 crater-devel-shared。运维内部集群（部署/rollout/重启/镜像 digest 校验）面向集群管理员，不属于本 Skill。"
---

# Crater 开发 · 发布

**开始前先应用 [`crater-devel-shared`](../crater-devel-shared/SKILL.md)。** 本 Skill 面向**开发者侧的发布准备**：开发 `charts/` 的 Helm Chart、管理 Chart 版本、产出可发布的镜像与 Chart。覆盖 `charts/` 与 `grafana-dashboards/`。

**完整开发规范见 `charts/CONTRIBUTING.md`**（版本管理、配置文档、注释、安全与验证均以其为权威），并同时遵守根 `CONTRIBUTING.md` 的全局流程。下文只保留发布准备中最容易漏掉的提醒。

**边界**：把 Crater 部署 / rollout 到具体（生产 / 内部）集群、`kubectl` 重启、核对 GHCR 镜像 digest、`act-gpu-cluster` 等**集群运维**面向集群管理员，**不属于本 Skill 也不属于 `crater-devel-*` 这套开发者 Skill**；那类任务用面向管理员的 `crater-rollout` Skill。

## 高优先提醒

- 改 `values.yaml` 配置逻辑、模板、依赖、配置项行为或默认值时，按 `charts/CONTRIBUTING.md` 提升 `charts/crater/Chart.yaml` 的 `version` 与 `appVersion`，并保持二者完全相同。
- 仅 values 字段变更通常提升 patch；前后端 API 变化并影响 Chart 部署出的应用契约时必须提升 minor；不要主动提升 major。
- 改 Chart 配置须同步 `charts/crater/README.md`；新增 values 项要有准确英文注释。
- 版本为发布而变化时，提醒创建对应 GitHub tag（通常为 `v<version>`）。文档引用版本号遵循 `website/CONTRIBUTING.md` 的占位约定，不硬编码。
- 配置中含密钥、证书、kubeconfig、内部 endpoint 或生产凭据时严禁上传公网。
- Chart 发布由 CI 完成；开发者通常无需本地手动打包或推送 Chart。
