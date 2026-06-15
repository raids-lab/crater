<div align="center">

<img src="./website/content/docs/admin/assets/icon.webp" alt="Crater logo" width="120" />

# Crater

### A Kubernetes-native AI development platform

Unified **GPU resource management**, **containerized development environments**, and **workflow orchestration** — all in one place.

<br/>

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](LICENSE)
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

</div>

---

## ✨ Overview

**Crater** helps teams manage heterogeneous compute resources (e.g., GPUs) and run AI workloads on Kubernetes through **unified scheduling**, **ready-to-use development environments**, and **end-to-end observability** — all from a single, clean web interface.

<div align="center">

|  |  |
| :---: | :---: |
| <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/jupyter.gif" alt="Jupyter Lab" width="420" /><br/>**🧪 Jupyter Lab** — interactive dev environments | <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/ray.gif" alt="Ray Job" width="420" /><br/>**🚀 Ray Jobs** — distributed training & inference |
| <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/monitor.gif" alt="Monitor" width="420" /><br/>**📈 Monitoring** — real-time metrics & logs | <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/datasets.gif" alt="Models" width="420" /><br/>**📦 Models & Datasets** — manage assets in one place |

</div>

## 🎯 Features

<table>
  <tr>
    <td width="50%" valign="top">
      <h3>🎛️ Intuitive UI</h3>
      Manage clusters, jobs, and resources through a clean web interface, with real-time dashboards for CPU, memory, and storage usage.
    </td>
    <td width="50%" valign="top">
      <h3>⚙️ Intelligent Scheduling</h3>
      Allocate resources by priority and requirements, prioritizing time-sensitive jobs to improve overall cluster utilization.
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h3>🧪 Dev Environments</h3>
      Spin up containerized, ready-to-use environments like Jupyter Lab and Ray jobs in seconds — no manual setup required.
    </td>
    <td width="50%" valign="top">
      <h3>📈 Monitoring &amp; Logs</h3>
      Observe cluster status and troubleshoot quickly with built-in metrics, logs, and Grafana dashboards.
    </td>
  </tr>
</table>

## 🏗️ Architecture

<div align="center">
  <img src="./website/content/docs/admin/assets/architecture.webp" alt="Crater architecture" width="90%" />
  <br/>
  <sub>High-level architecture of Crater and its major components.</sub>
</div>

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
