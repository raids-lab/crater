<div align="center">

<img src="../../website/content/docs/admin/assets/icon.webp" alt="Crater logo" width="120" />

# Crater

### 面向共享 AI 算力集群的 Kubernetes 原生控制平面

面向科研、教学与企业团队，统一管理共享 **GPU 集群**、**大模型训练与推理工作负载**、**开发环境**与**数据/模型资产**。

<br/>

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](../../LICENSE)
[![Stars](https://img.shields.io/github/stars/raids-lab/crater?style=flat-square&logo=github&color=f5b301)](https://github.com/raids-lab/crater/stargazers)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](../../CONTRIBUTING.md)
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

<br/>

🏢 **多租户资源治理** &nbsp;·&nbsp; ⚙️ **策略化调度** &nbsp;·&nbsp; 🚀 **大模型训练与推理** &nbsp;·&nbsp; 🧩 **异构加速卡** &nbsp;·&nbsp; 🤖 **AI 辅助运维**

</div>

---

<details>
<summary><b>📖 目录</b></summary>

- [项目简介](#-项目简介)
- [为什么选择 Crater](#-为什么选择-crater)
- [适用场景](#-适用场景)
- [功能特性](#-功能特性)
- [整体架构](#️-整体架构)
- [快速开始](#-快速开始)
- [文档](#-文档)
- [仓库结构](#-仓库结构)
- [社区与支持](#-社区与支持)
- [贡献指南](#-贡献指南)
- [许可证](#-许可证)

</details>

## ✨ 项目简介

**Crater** 是一个面向共享 AI 算力集群的 Kubernetes 原生平台。它帮助组织管理异构算力资源，提交和治理 AI 工作负载，快速部署大模型训练与推理环境，并通过统一的 Web 控制台、CLI 与 AI 辅助运维入口观察集群运行状态。

Crater 适用于多个团队和多类工作负载共享同一套 GPU 集群的场景：长时间运行的训练任务、集中爆发的实验课程、交互式 Notebook、在线 AI 服务、大模型推理服务，以及离线数据处理流水线。它在 Kubernetes 与 Volcano 之上构建运维控制平面，将用户、项目、队列、配额、镜像、数据集、模型、作业、服务和可观测性连接成完整工作流。

<div align="center">

|  |  |
| :---: | :---: |
| <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/jupyter.gif" alt="Jupyter Lab" width="420" /><br/>**🧪 交互式开发** — Jupyter、WebIDE、终端与外部访问 | <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/ray.gif" alt="Batch Jobs" width="420" /><br/>**🚀 AI 工作负载** — 训练、推理、模板与批处理作业 |
| <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/monitor.gif" alt="Monitor" width="420" /><br/>**📈 监控** — 实时指标与日志 | <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/datasets.gif" alt="Models" width="420" /><br/>**📦 模型与数据集** — 集中管理资产 |

</div>

## 💡 为什么选择 Crater

Kubernetes 与 Volcano 提供了强大的底层调度能力，但要把一套 GPU 集群**共享**给多个团队使用，仍然需要大量胶水工作。Crater 正是用来补上这一层：

| 没有控制平面 | 使用 Crater |
| :--- | :--- |
| 直接 `kubectl` / YAML 操作，容易误用 | Web 控制台、CLI 与 API，支持基于角色的多租户访问 |
| GPU 用量难以归因和约束 | 账号、队列、配额、审批与成本可见性 |
| 训练/推理清单全靠手写重复 | 可复用作业模板与大模型一键部署 |
| 数据集、模型、镜像散落在各节点 | 统一管理数据集、模型、镜像与共享存储 |
| 管理员与用户用不同工具排障 | 统一指标、日志、GPU 分析与 AI 辅助运维 |

## 🌐 适用场景

Crater 面向高校、科研机构、企业 AI 团队和内部平台团队中的共享 AI 算力环境。

| 场景 | 典型工作负载 | Crater 提供的能力 |
| :--- | :--- | :--- |
| **科研与工程** | 模型微调、仿真计算、科学计算、大规模实验 | 长时间 GPU 作业、可复用环境、数据/模型挂载、日志、监控和生命周期控制 |
| **教学与实训** | 课程实验、学生项目、虚拟仿真、培训工作坊 | 账号与配额管理、作业模板、集中并发承载、公平访问和简单的 Web 提交入口 |
| **大模型训练与推理** | 微调、评测、推理端点、模型演示、训推混部集群 | 快速部署模板、GPU 感知放置、数据/模型资产、服务访问和训推资源治理 |
| **企业 AI 服务** | 内部助手、文档智能、多模态服务、推理后端 | 托管运行环境、服务访问、运维可视性和资源治理 |
| **数据处理** | 数据集准备、图像解析、批处理流水线、离线预处理 | 存储集成、数据/模型管理、可调度批处理作业和可观测能力 |

## 🎯 功能特性

<table>
  <tr>
    <td width="50%" valign="top">
      <h3>🏢 多租户资源治理</h3>
      面向共享集群管理用户、项目、队列、配额、审批，以及面向计费和成本归因的资源可见性，将原始 GPU 集群转化为可治理、可追踪的团队级算力服务。
    </td>
    <td width="50%" valign="top">
      <h3>⚙️ 策略化调度</h3>
      基于 Kubernetes 与 Volcano，支持队列准入、优先级执行、预排队策略，以及跨异构资源的工作负载放置，覆盖训练与推理混部场景。
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h3>🚀 工作负载生命周期</h3>
      基于 Kubernetes 原生作业和可复用模板，提交、克隆、监控、停止和诊断 AI 工作负载，覆盖交互式会话、大模型微调和长时间批处理作业。
    </td>
    <td width="50%" valign="top">
      <h3>🧪 交互式开发环境</h3>
      无需手动配置集群，即可启动容器化 Jupyter、WebIDE、Web 终端、SSH 访问和自定义开发环境，让用户在靠近数据和 GPU 的位置获得可复现工作空间。
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h3>📦 数据、模型与镜像资产</h3>
      统一组织数据集、模型、共享文件、自定义镜像、镜像仓库条目，以及平台侧模型/数据集下载任务，让工作负载复用受管理的资产。
    </td>
    <td width="50%" valign="top">
      <h3>🧩 异构加速卡支持</h3>
      将 GPU 和加速卡型号抽象为可调度资源，支持 NVIDIA GPU、国产加速卡、vGPU 类资源，以及基于 DRA/CDI 的设备集成。
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h3>📈 可观测性与运维</h3>
      通过指标、日志、Grafana Dashboard、节点状态、操作日志、GPU 分析和运行时检查快速定位问题，降低平台管理员与工作负载用户之间的排障成本。
    </td>
    <td width="50%" valign="top">
      <h3>⌨️ Web、CLI 与 Agent 接口</h3>
      支持通过 Web 控制台、命令行工具、HTTP API，以及面向 AI Agent 的命令 Skills 操作平台，便于自动化、脚本化工作流和智能运维。
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h3>🤖 大模型与 AI 服务平台</h3>
      支持大模型快速部署、大模型训练与推理、推理网关、模型服务集成、可信服务集成，以及平台管理的运行时模板。
    </td>
    <td width="50%" valign="top">
      <h3>☸️ Kubernetes 原生部署</h3>
      通过 Helm 部署，并与 Kubernetes、Volcano、Prometheus/Grafana、持久化存储和集群组件集成，同时保持工作负载的可迁移性。
    </td>
  </tr>
</table>

## 🏗️ 整体架构

<div align="center">
  <img src="../../website/content/docs/admin/assets/architecture.webp" alt="crater architecture" width="90%" />
  <br/>
  <sub>Crater 的整体架构与主要组件概览。</sub>
</div>

Crater 由四个层次组成：

- **用户入口**：Web 控制台、CLI、HTTP API，以及面向 AI Agent 的命令 Skills。
- **控制平面**：认证、项目、配额、调度策略、作业、服务、模板、镜像、数据集、模型、审批与运维操作。
- **执行层**：面向训练、推理和交互式环境的 Kubernetes 工作负载、Volcano 调度、加速卡资源、Pod、Service、PVC 和外部访问规则。
- **可观测与 AI 运维层**：指标、日志、Grafana Dashboard、操作记录、运行时诊断、AI 助手工作流和管理员侧智能运维。

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

## 💬 社区与支持

- 🐛 **Issues** — 反馈缺陷或提交需求：[GitHub Issues](https://github.com/raids-lab/crater/issues)
- 💡 **Discussions** — 提问与交流想法：[GitHub Discussions](https://github.com/raids-lab/crater/discussions)
- 📚 **文档** — 管理员与用户指南：[raids-lab.github.io/crater](https://raids-lab.github.io/crater/zh/docs/admin/)
- ⭐ 如果 Crater 对你有帮助，欢迎点亮 Star，让更多人发现它。

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
