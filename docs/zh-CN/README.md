<div align="center">

<img src="../../website/content/docs/admin/assets/icon.webp" alt="Crater logo" width="120" />

# Crater

### 面向 Kubernetes 的 AI 开发平台

一站式提供 **GPU 资源管理**、**容器化开发环境**与**工作流编排**能力。

<br/>

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](../../LICENSE)
[![Docs](https://img.shields.io/badge/Docs-raids--lab.github.io-brightgreen?style=flat-square)](https://raids-lab.github.io/crater/zh)
[![Backend Build](https://img.shields.io/github/actions/workflow/status/raids-lab/crater/backend-build.yml?style=flat-square&label=backend)](https://github.com/raids-lab/crater/actions/workflows/backend-build.yml)
[![Helm Chart Validate](https://img.shields.io/github/actions/workflow/status/raids-lab/crater/helm-chart-validate.yml?style=flat-square&label=helm)](https://github.com/raids-lab/crater/actions/workflows/helm-chart-validate.yml)

![Kubernetes](https://img.shields.io/badge/Kubernetes-326CE5?style=flat-square&logo=kubernetes&logoColor=white)
![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-20232A?style=flat-square&logo=react&logoColor=61DAFB)
![Helm](https://img.shields.io/badge/Helm-0F1689?style=flat-square&logo=helm&logoColor=white)

[English](../../README.md) · **简体中文**

[**文档**](https://raids-lab.github.io/crater/zh/docs/admin/) ·
[Helm Chart](../../charts/crater) ·
[后端](../../backend/README.zh-CN.md) ·
[前端](../../frontend/README.zh-CN.md) ·
[CLI](../../cli/README.zh-CN.md)

</div>

---

## ✨ 项目简介

**Crater** 帮助团队管理异构算力资源（例如 GPU），并通过**统一调度**、**开箱即用的开发环境**与**端到端可观测性**在 Kubernetes 上运行 AI 工作负载——这一切都在一个简洁的 Web 界面中完成。

<div align="center">

|  |  |
| :---: | :---: |
| <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/jupyter.gif" alt="Jupyter Lab" width="420" /><br/>**🧪 Jupyter Lab** — 交互式开发环境 | <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/ray.gif" alt="Ray Job" width="420" /><br/>**🚀 Ray 作业** — 分布式训练与推理 |
| <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/monitor.gif" alt="Monitor" width="420" /><br/>**📈 监控** — 实时指标与日志 | <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/datasets.gif" alt="Models" width="420" /><br/>**📦 模型与数据集** — 集中管理资产 |

</div>

## 🎯 功能特性

<table>
  <tr>
    <td width="50%" valign="top">
      <h3>🎛️ 直观的界面</h3>
      通过简洁的 Web 界面管理集群、作业与资源，并实时查看 CPU、内存与存储等关键指标。
    </td>
    <td width="50%" valign="top">
      <h3>⚙️ 智能调度</h3>
      根据优先级与资源需求进行分配，优先处理时间敏感的任务，提升集群整体利用率。
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h3>🧪 开发环境</h3>
      数秒内启动容器化、开箱即用的开发环境，如 Jupyter Lab 与 Ray 作业，无需手动配置。
    </td>
    <td width="50%" valign="top">
      <h3>📈 监控与日志</h3>
      借助内置指标、日志与 Grafana Dashboard 掌握集群状态并快速排障。
    </td>
  </tr>
</table>

## 🏗️ 整体架构

<div align="center">
  <img src="../../website/content/docs/admin/assets/architecture.webp" alt="crater architecture" width="90%" />
  <br/>
  <sub>Crater 的整体架构与主要组件概览。</sub>
</div>

## 🚀 快速开始

### 1. 前置条件

- 一个可用的 Kubernetes 集群
- [`kubectl`](https://kubernetes.io/docs/tasks/tools/)
- [Helm v3](https://helm.sh/docs/intro/install/)

### 2. 准备集群

根据您的场景选择合适的方式：

| 方式 | 适用场景 | 参考 |
| :--- | :--- | :--- |
| 🐳 **Kind** | 在 Docker 中运行本地集群 | [kind.sigs.k8s.io](https://kind.sigs.k8s.io/) |
| 🧱 **Minikube** | 单节点本地开发与测试 | [minikube.sigs.k8s.io](https://minikube.sigs.k8s.io/) |
| ☁️ **生产级 K8s** | 生产或大规模部署 | [kubernetes.io/docs/setup](https://kubernetes.io/docs/setup/) |

### 3. 通过 Helm（OCI）安装

```bash
helm registry login ghcr.io
helm install crater oci://ghcr.io/raids-lab/crater --version <chart-version>
```

> 💡 Chart 版本可在 `charts/crater/Chart.yaml`（字段 `version`）或 GitHub releases 中查看。

**部署指南：**

- 📄 [最小化部署（Kind）](https://raids-lab.github.io/crater/zh/docs/admin/kind-start/) — 快速部署一个基本的 Crater
- 📄 [集群部署指南](https://raids-lab.github.io/crater/zh/docs/admin/deploy-on-cluster/) — 在集群中部署完整的 Crater

## 📚 文档

- 📗 管理员指南（中文）: https://raids-lab.github.io/crater/zh/docs/admin/
- 📘 管理员指南（English）: https://raids-lab.github.io/crater/en/docs/admin/

## 📁 仓库结构

| 路径 | 说明 |
| :--- | :--- |
| `backend/` | 后端服务 |
| `frontend/` | Web 前端 |
| `cli/` | 命令行工具 |
| `charts/` | 用于部署 Crater 的 Helm Chart |
| `website/` | 文档网站源码 |
| `grafana-dashboards/` | Crater 使用的 Grafana Dashboard |
| `docs/` | 文档入口与多语言资源 |
| `hack/` | 开发工具与脚本 |

## 🤝 贡献指南

欢迎社区贡献！开发与贡献的完整规范见 [CONTRIBUTING.md](../../CONTRIBUTING.md)：全局守则、环境准备（fork、hooks、统一配置）、开发流程、Commit 规范、PR 描述模板与各模块入口。

**各模块规范：**

- 后端 — [backend/CONTRIBUTING.zh-CN.md](../../backend/CONTRIBUTING.zh-CN.md)
- 前端 — [frontend/CONTRIBUTING.zh-CN.md](../../frontend/CONTRIBUTING.zh-CN.md)
- 文档 / 文档站 — [website/CONTRIBUTING.zh-CN.md](../../website/CONTRIBUTING.zh-CN.md)
- CLI — [cli/CONTRIBUTING.zh-CN.md](../../cli/CONTRIBUTING.zh-CN.md)

## 📝 许可证

Crater 使用 **Apache License 2.0** 许可证，详见 [LICENSE](../../LICENSE)。

<div align="center"><sub>Copyright 2023-2026 The Crater Project Team, RAIDS-Lab.</sub></div>
