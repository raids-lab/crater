<div align="center">

<img src="./website/content/docs/admin/assets/icon.webp" alt="Crater logo" width="120" />

# Crater

### A Kubernetes-native control plane for shared AI computing clusters

Operate shared **GPU clusters**, **LLM training and serving workloads**, **developer environments**, and **data/model assets** across research, education, and enterprise teams.

<br/>

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](LICENSE)
[![Stars](https://img.shields.io/github/stars/raids-lab/crater?style=flat-square&logo=github&color=f5b301)](https://github.com/raids-lab/crater/stargazers)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](./CONTRIBUTING.md)
[![Docs](https://img.shields.io/badge/Docs-raids--lab.github.io-brightgreen?style=flat-square)](https://raids-lab.github.io/crater/zh)
[![Backend Build](https://img.shields.io/github/actions/workflow/status/raids-lab/crater/backend-build.yml?style=flat-square&label=backend)](https://github.com/raids-lab/crater/actions/workflows/backend-build.yml)
[![Helm Chart Validate](https://img.shields.io/github/actions/workflow/status/raids-lab/crater/helm-chart-validate.yml?style=flat-square&label=helm)](https://github.com/raids-lab/crater/actions/workflows/helm-chart-validate.yml)

![Kubernetes](https://img.shields.io/badge/Kubernetes-326CE5?style=flat-square&logo=kubernetes&logoColor=white)
![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-20232A?style=flat-square&logo=react&logoColor=61DAFB)
![Helm](https://img.shields.io/badge/Helm-0F1689?style=flat-square&logo=helm&logoColor=white)

**English** · [简体中文](docs/zh-CN/README.md)

[**Documentation**](https://raids-lab.github.io/crater/zh/docs/admin/) ·
[Helm Chart](./charts/crater) ·
[Backend](./backend/README.md) ·
[Frontend](./frontend/README.md) ·
[CLI](./cli/README.md)

<br/>

🏢 **Multi-Tenant Governance** &nbsp;·&nbsp; ⚙️ **Policy-Aware Scheduling** &nbsp;·&nbsp; 🚀 **LLM Training & Serving** &nbsp;·&nbsp; 🧩 **Heterogeneous Accelerators** &nbsp;·&nbsp; 🤖 **AI-Assisted Operations**

</div>

---

<details>
<summary><b>📖 Table of Contents</b></summary>

- [Overview](#-overview)
- [Why Crater](#-why-crater)
- [Designed For](#-designed-for)
- [Features](#-features)
- [Architecture](#-architecture)
- [Getting Started](#-getting-started)
- [Documentation](#-documentation)
- [Repository Structure](#-repository-structure)
- [Community & Support](#-community--support)
- [Contributing](#-contributing)
- [License](#-license)

</details>

## ✨ Overview

**Crater** is a Kubernetes-native platform for operating shared AI computing clusters. It helps organizations manage heterogeneous compute resources, submit and govern AI workloads, quickly deploy large-model training and inference environments, and observe cluster health from a unified web console, CLI, and AI-assisted operations interface.

Crater is designed for environments where different teams and workloads share the same GPU cluster: long-running training jobs, bursty lab workloads, interactive notebooks, online AI services, LLM inference services, and offline data processing pipelines. It builds an operational control plane on top of Kubernetes and Volcano, connecting users, accounts, queues, quotas, images, datasets, models, jobs, services, and observability into one workflow.

<div align="center">

|  |  |
| :---: | :---: |
| <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/jupyter.gif" alt="Jupyter Lab" width="420" /><br/>**🧪 Interactive Development** — Jupyter, WebIDE, terminals, and external access | <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/ray.gif" alt="Batch Jobs" width="420" /><br/>**🚀 AI Workloads** — training, serving, templates, and batch jobs |
| <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/monitor.gif" alt="Monitor" width="420" /><br/>**📈 Monitoring** — real-time metrics & logs | <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/datasets.gif" alt="Models" width="420" /><br/>**📦 Models & Datasets** — manage assets in one place |

</div>

## 💡 Why Crater

Kubernetes and Volcano provide powerful low-level scheduling, but operating a **shared** GPU cluster for many teams still requires a lot of glue. Crater fills that gap:

| Without a control plane | With Crater |
| :--- | :--- |
| Raw `kubectl` / YAML access, easy to misuse | Web console, CLI, and APIs with role-based, multi-tenant access |
| GPU usage is hard to attribute and bound | Accounts, queues, quotas, approvals, and cost visibility |
| Everyone rebuilds training/serving manifests by hand | Reusable job templates and one-click LLM deployment |
| Datasets, models, and images scattered across nodes | Managed datasets, models, images, and shared storage |
| Operators and users debug from different tools | Unified metrics, logs, GPU analysis, and AI-assisted operations |

## 🌐 Designed For

Crater fits shared AI computing environments in universities, research institutes, enterprise AI teams, and internal platform teams.

| Scenario | Typical workloads | What Crater provides |
| :--- | :--- | :--- |
| **Research & engineering** | Model fine-tuning, simulation, scientific computing, large experiments | Long-running GPU jobs, reusable environments, data/model mounting, logs, monitoring, and lifecycle controls |
| **Teaching & training** | Course labs, student projects, virtual experiments, workshops | Account and quota management, job templates, burst handling, fair access, and simple web-based submission |
| **LLM training & serving** | Fine-tuning, evaluation, inference endpoints, model demos, mixed training/serving clusters | Fast deployment templates, GPU-aware placement, data/model assets, service access, and train/serve resource governance |
| **Enterprise AI services** | Internal assistants, document intelligence, multimodal services, inference backends | Managed runtime environments, service access, operational visibility, and resource governance |
| **Data processing** | Dataset preparation, image analysis, batch pipelines, offline preprocessing | Storage integration, dataset/model management, schedulable batch jobs, and observability |

## 🎯 Features

<table>
  <tr>
    <td width="50%" valign="top">
      <h3>🏢 Multi-Tenant Governance</h3>
      Manage users, accounts, queues, quotas, approvals, and billing-oriented resource visibility. Crater turns a raw GPU cluster into an accountable shared service for teams and projects.
    </td>
    <td width="50%" valign="top">
      <h3>⚙️ Policy-Aware Scheduling</h3>
      Build on Kubernetes and Volcano to support queue-based admission, priority-aware execution, prequeue policies, and workload placement across heterogeneous resources, including mixed training and serving workloads.
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h3>🚀 Workload Lifecycle</h3>
      Submit, clone, monitor, stop, and inspect AI workloads through Kubernetes-native jobs and reusable templates, from interactive sessions and LLM fine-tuning to long-running batch jobs.
    </td>
    <td width="50%" valign="top">
      <h3>🧪 Interactive Development</h3>
      Launch containerized Jupyter, WebIDE, web terminals, SSH access, and custom environments without manual cluster setup, giving users a reproducible workspace close to the data and GPUs.
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h3>📦 Data, Model &amp; Image Assets</h3>
      Organize datasets, models, shared files, custom images, registry entries, and platform-side model or dataset downloads so workloads can reuse managed artifacts.
    </td>
    <td width="50%" valign="top">
      <h3>🧩 Heterogeneous Accelerators</h3>
      Represent GPUs and accelerator models as schedulable resources, supporting NVIDIA GPUs, domestic accelerator cards, vGPU-style resources, and DRA/CDI-based device integration.
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h3>📈 Observability &amp; Operations</h3>
      Troubleshoot with metrics, logs, Grafana dashboards, node status, operation logs, GPU analysis, and runtime inspection, reducing the gap between platform operators and workload owners.
    </td>
    <td width="50%" valign="top">
      <h3>⌨️ Web, CLI &amp; Agent Interfaces</h3>
      Operate Crater through a web console, command-line interface, HTTP APIs, and agent-oriented command skills for automation, scripted workflows, and AI-assisted operations.
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h3>🤖 LLM &amp; AI Service Platform</h3>
      Support large-model quick deployment, LLM training and inference, inference gateways, model-serving integrations, trusted service integrations, and platform-managed runtime templates.
    </td>
    <td width="50%" valign="top">
      <h3>☸️ Kubernetes-Native Deployment</h3>
      Deploy with Helm and integrate with Kubernetes, Volcano, Prometheus/Grafana, persistent storage, and cluster add-ons while keeping workloads portable.
    </td>
  </tr>
</table>

## 🏗️ Architecture

<div align="center">
  <img src="./website/content/docs/admin/assets/architecture.webp" alt="Crater architecture" width="90%" />
  <br/>
  <sub>High-level architecture of Crater and its major components.</sub>
</div>

Crater is organized around four layers:

- **User interfaces**: web console, CLI, HTTP APIs, and agent-friendly command skills.
- **Control plane**: authentication, accounts, quotas, scheduling policies, jobs, services, templates, images, datasets, models, approvals, and operations.
- **Execution layer**: Kubernetes workloads, Volcano scheduling, accelerator resources, Pods, Services, PVCs, and external access rules for training, serving, and interactive environments.
- **Observability and AI operations layer**: metrics, logs, Grafana dashboards, operation records, runtime diagnostics, AI assistant workflows, and admin-side intelligent operations.

## 🚀 Getting Started

### 1. Prerequisites

- A running Kubernetes cluster
- [`kubectl`](https://kubernetes.io/docs/tasks/tools/)
- [Helm v3](https://helm.sh/docs/intro/install/)

### 2. Set up a cluster

Pick the option that fits your scenario:

| Option | Best for | Reference |
| :--- | :--- | :--- |
| 🐳 **Kind** | Local clusters in Docker | [kind.sigs.k8s.io](https://kind.sigs.k8s.io/) |
| 🧱 **Minikube** | Single-node local dev & testing | [minikube.sigs.k8s.io](https://minikube.sigs.k8s.io/) |
| ☁️ **Production K8s** | Production or large-scale deployments | [kubernetes.io/docs/setup](https://kubernetes.io/docs/setup/) |

### 3. Install via Helm (OCI)

```bash
helm registry login ghcr.io
helm install crater oci://ghcr.io/raids-lab/crater --version <chart-version>
```

> 💡 The chart version is in `charts/crater/Chart.yaml` (field `version`) or in the GitHub releases.

**Deployment guides:**

- 📄 [Minimal Deployment (Kind)](https://raids-lab.github.io/crater/en/docs/admin/kind-start/) — quickly spin up a basic Crater
- 📄 [Cluster Deployment Guide](https://raids-lab.github.io/crater/en/docs/admin/deploy-on-cluster/) — deploy a full Crater on a cluster

## 📚 Documentation

- 📘 Admin guide (English): https://raids-lab.github.io/crater/en/docs/admin/
- 📗 Admin guide (中文): https://raids-lab.github.io/crater/zh/docs/admin/

## 📁 Repository Structure

| Path | Description |
| :--- | :--- |
| `backend/` | Backend services |
| `frontend/` | Web UI |
| `cli/` | Command-line interface |
| `charts/` | Helm charts for deploying Crater |
| `website/` | Documentation website source |
| `grafana-dashboards/` | Grafana dashboards used by Crater |
| `docs/` | Documentation entrypoints and localization resources |
| `hack/` | Developer tooling and scripts |

## 💬 Community & Support

- 🐛 **Issues** — report bugs or request features: [GitHub Issues](https://github.com/raids-lab/crater/issues)
- 💡 **Discussions** — ask questions and share ideas: [GitHub Discussions](https://github.com/raids-lab/crater/discussions)
- 📚 **Docs** — admin and user guides: [raids-lab.github.io/crater](https://raids-lab.github.io/crater/en/docs/admin/)
- ⭐ **Star the project** if you find Crater useful — it helps others discover it.

## 🤝 Contributing

We welcome community contributions! The complete development and contribution specification lives in [CONTRIBUTING.md](./CONTRIBUTING.md): global rules, environment setup (fork, hooks, unified config), workflow, commit convention, PR description template, and per-module entry points.

**Per-module specs:**

- Backend — [backend/CONTRIBUTING.md](./backend/CONTRIBUTING.md)
- Frontend — [frontend/CONTRIBUTING.md](./frontend/CONTRIBUTING.md)
- Website / Docs — [website/CONTRIBUTING.md](./website/CONTRIBUTING.md)
- CLI — [cli/CONTRIBUTING.md](./cli/CONTRIBUTING.md)

## 📝 License

Crater is licensed under the **Apache License 2.0**. See [LICENSE](LICENSE).

<div align="center"><sub>Copyright 2023-2026 The Crater Project Team, RAIDS-Lab.</sub></div>
