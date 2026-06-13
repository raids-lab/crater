<p align="center">
  <a href="README.md">English</a> · <a href="docs/zh-CN/README.md">简体中文</a>
</p>

<p align="center">
  <img src="./website/content/docs/admin/assets/icon.webp" alt="Crater logo" width="120" />
</p>

<h1 align="center">Crater</h1>

<p align="center">
  A comprehensive AI development platform for Kubernetes that provides GPU resource management, containerized development environments, and workflow orchestration.
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License" /></a>
  <a href="https://raids-lab.github.io/crater/zh"><img src="https://img.shields.io/badge/Docs-raids--lab.github.io-brightgreen" alt="Docs" /></a>
  <a href="https://github.com/raids-lab/crater/actions/workflows/backend-build.yml"><img src="https://github.com/raids-lab/crater/actions/workflows/backend-build.yml/badge.svg" alt="Backend Build" /></a>
  <a href="https://github.com/raids-lab/crater/actions/workflows/helm-chart-validate.yml"><img src="https://github.com/raids-lab/crater/actions/workflows/helm-chart-validate.yml/badge.svg" alt="Helm Chart Validate" /></a>
</p>

<p align="center">
  <a href="https://raids-lab.github.io/crater/zh/docs/admin/">Documentation</a> ·
  <a href="./charts/crater">Helm Chart</a> ·
  <a href="./backend/README.md">Backend</a> ·
  <a href="./frontend/README.md">Frontend</a>
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

Crater is a Kubernetes-based platform that helps teams manage heterogeneous compute resources (e.g., GPUs) and run AI workloads through unified scheduling, development environments, and observability.

## Features

- 🎛️ **Intuitive UI**: Manage clusters, jobs, and resources through a clean web interface.
- ⚙️ **Intelligent scheduling**: Allocate resources based on priority and requirements to improve utilization.
- 📈 **Monitoring & logs**: Observe cluster status and troubleshoot with metrics and logs.

## Architecture

![crater architecture](./website/content/docs/admin/assets/architecture.webp)

High-level architecture of Crater and its major components.

## Documentation

- Admin guide (中文): https://raids-lab.github.io/crater/zh/docs/admin/
- Admin guide (English): https://raids-lab.github.io/crater/en/docs/admin/

Deployment guides:

If you want to quickly deploy a basic Crater using Kind, please refer to [Minimal Deployment](https://raids-lab.github.io/crater/zh/docs/admin/kind-start/).

If you want to deploy a full Crater in a cluster, please refer to [Cluster Deployment Guide](https://raids-lab.github.io/crater/zh/docs/admin/deploy-on-cluster/).

English versions:

- [Minimal Deployment](https://raids-lab.github.io/crater/en/docs/admin/kind-start/)
- [Cluster Deployment Guide](https://raids-lab.github.io/crater/en/docs/admin/deploy-on-cluster/)

## Getting Started

### Prerequisites

- A running Kubernetes cluster
- `kubectl`
- Helm v3

To get started with **Crater**, you first need to have a running Kubernetes cluster. You can set up a cluster using one of the following methods:

### 🐳 1. Local Cluster with Kind

Kind (Kubernetes IN Docker) is a lightweight tool for running local Kubernetes clusters using Docker containers.

📖 [https://kind.sigs.k8s.io/](https://kind.sigs.k8s.io/)

### 🧱 2. Local Cluster with Minikube

Minikube runs a single-node Kubernetes cluster locally, ideal for development and testing.

📖 [https://minikube.sigs.k8s.io/](https://minikube.sigs.k8s.io/)

### ☁️ 3. Production-grade Kubernetes Cluster

For deploying Crater in a production or large-scale test environment, you can use any standard Kubernetes setup.

📖 [https://kubernetes.io/docs/setup/](https://kubernetes.io/docs/setup/)

### Install via Helm (OCI)

> Use the docs above for a full guide. The chart version can be found in `charts/crater/Chart.yaml` (field `version`) or GitHub releases.

```bash
helm registry login ghcr.io
helm install crater oci://ghcr.io/raids-lab/crater --version <chart-version>
```

## Repository Structure

- `backend/`: Backend services
- `frontend/`: Web UI
- `backend/internal/storage/`: Storage service (integrated in backend module)
- `charts/`: Helm charts for deploying Crater
- `website/`: Documentation website source
- `grafana-dashboards/`: Grafana dashboards used by Crater
- `docs/`: Documentation entrypoints and localization resources
- `hack/`: Developer tooling and scripts

## Contributing

We welcome community contributions. The complete development and contribution specification lives in [CONTRIBUTING.md](./CONTRIBUTING.md): global rules, environment setup (fork, hooks, unified config), workflow, commit convention, PR description template, and per-module entry points.

Per-module specs:

- Backend: [backend/CONTRIBUTING.md](./backend/CONTRIBUTING.md)
- Frontend: [frontend/CONTRIBUTING.md](./frontend/CONTRIBUTING.md)
- Website / Docs: [website/CONTRIBUTING.md](./website/CONTRIBUTING.md)
- CLI: [cli/CONTRIBUTING.md](./cli/CONTRIBUTING.md)

## License

Crater is licensed under the Apache License 2.0. See [LICENSE](LICENSE).

Copyright 2023-2026 The Crater Project Team, RAIDS-Lab.
