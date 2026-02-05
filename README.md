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
- `storage/`: Storage service
- `charts/`: Helm charts for deploying Crater
- `website/`: Documentation website source
- `grafana-dashboards/`: Grafana dashboards used by Crater
- `docs/`: Documentation entrypoints and localization resources
- `hack/`: Developer tooling and scripts

## Contributing

We welcome community contributions! If you would like to contribute to the Crater project, please follow the workflow below.

### 1) Fork and clone

1. **Fork the repository**
   - Visit the [Crater main repository](https://github.com/raids-lab/crater)
   - Click **Fork** in the top right corner

2. **Clone your fork**

   ```bash
   # Replace YOUR_USERNAME with your GitHub username
   git clone https://github.com/YOUR_USERNAME/crater.git
   cd crater
   ```

3. **Add upstream (optional)**

   ```bash
   # Add upstream repository to sync latest changes
   git remote add upstream https://github.com/raids-lab/crater.git

   # Verify remote repository configuration
   git remote -v
   ```

   If you configure it this way, `origin` points to your fork repository, and `upstream` points to the upstream main repository.

### 2) Create a branch

It's recommended to create a new feature branch from the latest main branch. If you need to sync upstream changes, please first refer to the [Sync Upstream Changes](#-sync-upstream-changes) section to update your local main branch, then create a new feature branch:

```bash
# Create and switch to a new feature branch
git checkout -b feature/your-feature-name
# Or use this when fixing bugs
git checkout -b fix/your-bug-fix
```

### 3) Install hooks and set up environments

Install Git pre-commit hooks (required for the repository workflow):

```bash
make install-hooks
```

Then set up the environment for the component you want to work on:

- Backend: [Backend Development Guide](./backend/README.md)
- Frontend: [Frontend Development Guide](./frontend/README.md)
- Storage: [Storage Service Development Guide](./storage/README.md)
- Website: [Documentation Website Development Guide](./website/README.md)

### 4) Configuration files (optional)

Crater provides a unified configuration management workflow to centralize configs and create symlinks per component.

Example structure:

```
config/
├── backend/
│   ├── .debug.env              # Backend debug environment variables
│   ├── kubeconfig              # Kubernetes config file (optional)
│   └── debug-config.yaml       # Backend debug configuration
├── frontend/
│   └── .env.development        # Frontend development environment variables
└── storage/
    ├── .env                    # Storage service environment variables
    └── config.yaml             # Storage service configuration
```

Make targets:

- `make config-link`: Create symlinks for config files (backs up existing files with `.bak`)

  ```bash
  make config-link CONFIG_DIR=~/develop/crater/config
  ```

- `make config-status`: Show config file status
- `make config-unlink`: Remove symlinks only
- `make config-restore`: Restore files from `.bak`

### 5) Run checks before commit

The pre-commit hook checks staged files and runs checks for the affected sub-projects:

```bash
git add <your-files>
make pre-commit-check
```

You can also run checks within a sub-project (checks all files in that sub-project):

```bash
cd frontend && make pre-commit-check
cd backend && make pre-commit-check
cd storage && make pre-commit-check
cd website && make pre-commit-check
```

### 6) Commit and open a PR

```bash
git status

# Add changed files (please specify specific files or directories, avoid using git add .)
git add backend/pkg/handler/user.go
git add frontend/src/components/

git commit -m "feat: add new feature description"
git push origin feature/your-feature-name
```

Commit message convention:

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation updates
- `style:` Code style changes
- `refactor:` Code refactoring
- `test:` Test related
- `chore:` Build/tool related

Then create a Pull Request on GitHub and include:

- What changed and why
- How to test
- Screenshots (if UI changes)

### 7) (Optional) squash commits

```bash
# Assuming you have 3 commits to squash
git rebase -i HEAD~3
```

<a id="-sync-upstream-changes"></a>

### 🔄 Sync Upstream Changes

If you have added `upstream` and there are new changes in the upstream repository, you can update your local main branch as follows:

```bash
git checkout main
git fetch upstream
git merge upstream/main
# Or use the shortcut (one step)
# git pull upstream main
```

After your local main branch is updated, you can create new feature branches based on it.

If you already have a feature branch, merge updates into it:

```bash
git checkout feature/your-feature-name
git merge main
```

If you haven't configured `upstream`, you can use GitHub's **Sync fork** feature to sync changes from upstream, then run `git pull origin main` locally.

## License

Crater is licensed under the Apache License 2.0. See [LICENSE](LICENSE).
