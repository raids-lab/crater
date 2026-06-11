<p align="center">
  <a href="../../README.md">English</a> · <a href="README.md">简体中文</a>
</p>

<p align="center">
  <img src="../../website/content/docs/admin/assets/icon.webp" alt="Crater logo" width="120" />
</p>

<h1 align="center">Crater</h1>

<p align="center">
  A comprehensive AI development platform for Kubernetes that provides GPU resource management, containerized development environments, and workflow orchestration.
</p>

<p align="center">
  <a href="../../LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License" /></a>
  <a href="https://raids-lab.github.io/crater/zh"><img src="https://img.shields.io/badge/Docs-raids--lab.github.io-brightgreen" alt="Docs" /></a>
  <a href="https://github.com/raids-lab/crater/actions/workflows/backend-build.yml"><img src="https://github.com/raids-lab/crater/actions/workflows/backend-build.yml/badge.svg" alt="Backend Build" /></a>
  <a href="https://github.com/raids-lab/crater/actions/workflows/helm-chart-validate.yml"><img src="https://github.com/raids-lab/crater/actions/workflows/helm-chart-validate.yml/badge.svg" alt="Helm Chart Validate" /></a>
</p>

<p align="center">
  <a href="https://raids-lab.github.io/crater/zh/docs/admin/">文档</a> ·
  <a href="../../charts/crater">Helm Chart</a> ·
  <a href="../../backend/README.zh-CN.md">后端</a> ·
  <a href="../../frontend/README.zh-CN.md">前端</a>
</p>

<table>
  <tr>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/jupyter.gif" alt="Jupyter Lab" /><br>
      <em>Jupyter Lab</em>
    </td>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/ray.gif" alt="Ray Job" /><br>
      <em>Ray Job</em>
    </td>
  </tr>
  <tr>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/monitor.gif" alt="Monitor" /><br>
      <em>Monitor</em>
    </td>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/datasets.gif" alt="Models" /><br>
      <em>Models</em>
    </td>
  </tr>
</table>

Crater 是一个基于 Kubernetes 的平台，帮助团队管理异构算力资源（例如 GPU），并通过统一调度、开发环境与可观测性能力运行 AI 工作负载。

## 功能特性

- 🎛️ **直观的界面**：通过清晰的 Web 界面管理集群、作业与资源。
- ⚙️ **智能调度**：根据优先级与资源需求进行分配，提升集群利用率。
- 📈 **监控与日志**：通过指标与日志掌握集群状态并快速排障。

## 整体架构

![crater architecture](../../website/content/docs/admin/assets/architecture.webp)

Crater 的整体架构与主要组件概览。

## 文档

- 管理员指南（中文）: https://raids-lab.github.io/crater/zh/docs/admin/
- 管理员指南（English）: https://raids-lab.github.io/crater/en/docs/admin/

部署文档：

如果您希望使用 Kind 快速部署一个基本的 Crater，请参照[最小化部署](https://raids-lab.github.io/crater/zh/docs/admin/kind-start/)。

如果您希望在集群中部署一个完整的 Crater，请参照[集群部署指南](https://raids-lab.github.io/crater/zh/docs/admin/deploy-on-cluster/)。

英文版本：

- [Minimal Deployment](https://raids-lab.github.io/crater/en/docs/admin/kind-start/)
- [Cluster Deployment Guide](https://raids-lab.github.io/crater/en/docs/admin/deploy-on-cluster/)

## 快速开始

### 前置条件

- 一个可用的 Kubernetes 集群
- `kubectl`
- Helm v3

要开始使用 **Crater**，您首先需要有一个正在运行的 Kubernetes 集群。您可以使用以下方法之一来设置集群：

### 🐳 1. 使用 Kind 的本地集群

Kind (Kubernetes IN Docker) 是一个使用 Docker 容器运行本地 Kubernetes 集群的轻量级工具。

📖 [https://kind.sigs.k8s.io/](https://kind.sigs.k8s.io/)

### 🧱 2. 使用 Minikube 的本地集群

Minikube 在本地运行单节点 Kubernetes 集群，非常适合开发和测试。

📖 [https://minikube.sigs.k8s.io/](https://minikube.sigs.k8s.io/)

### ☁️ 3. 生产级 Kubernetes 集群

要在生产环境或大规模测试环境中部署 Crater，您可以使用任何标准的 Kubernetes 设置。

📖 [https://kubernetes.io/docs/setup/](https://kubernetes.io/docs/setup/)

### 通过 Helm（OCI）安装

> 更完整的步骤请以文档为准。Chart 版本可在 `charts/crater/Chart.yaml`（字段 `version`）或 GitHub releases 中查看。

```bash
helm registry login ghcr.io
helm install crater oci://ghcr.io/raids-lab/crater --version <chart-version>
```

## 仓库结构

- `backend/`: 后端服务
- `frontend/`: Web 前端
- `backend/internal/storage/`: 存储服务（已并入 backend 模块）
- `charts/`: 用于部署 Crater 的 Helm Chart
- `website/`: 文档网站源码
- `grafana-dashboards/`: Crater 使用的 Grafana Dashboard
- `docs/`: 文档入口与多语言资源
- `hack/`: 开发工具与脚本

## 贡献指南

欢迎社区贡献。开发与贡献的完整规范见 [CONTRIBUTING.md](./CONTRIBUTING.md)：全局守则、环境准备（fork、hooks、统一配置）、开发流程、Commit 规范、PR 描述模板与各模块入口。

各模块规范：

- 后端：[backend/CONTRIBUTING.zh-CN.md](../../backend/CONTRIBUTING.zh-CN.md)
- 前端：[frontend/CONTRIBUTING.zh-CN.md](../../frontend/CONTRIBUTING.zh-CN.md)
- 文档 / 文档站：[website/CONTRIBUTING.zh-CN.md](../../website/CONTRIBUTING.zh-CN.md)
- CLI：[cli/CONTRIBUTING.zh-CN.md](../../cli/CONTRIBUTING.zh-CN.md)

## 许可证

Crater 使用 Apache License 2.0 许可证，详见 [LICENSE](../../LICENSE)。
