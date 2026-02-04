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
- `storage/`: 存储服务
- `charts/`: 用于部署 Crater 的 Helm Chart
- `website/`: 文档网站源码
- `grafana-dashboards/`: Crater 使用的 Grafana Dashboard
- `docs/`: 文档入口与多语言资源
- `hack/`: 开发工具与脚本

## 贡献指南

我们欢迎社区贡献！如果您想为 Crater 项目做出贡献，请遵循以下流程。

### 1) Fork 与克隆

1. **Fork 仓库**
   - 访问 [Crater 主仓库](https://github.com/raids-lab/crater)
   - 点击右上角的 **Fork**

2. **克隆您的 Fork**

   ```bash
   # 将 YOUR_USERNAME 替换为您的 GitHub 用户名
   git clone https://github.com/YOUR_USERNAME/crater.git
   cd crater
   ```

3. **添加 upstream（可选）**

   ```bash
   # 添加上游仓库以便同步最新更改
   git remote add upstream https://github.com/raids-lab/crater.git

   # 验证远程仓库配置
   git remote -v
   ```

   如果您按照这种方式配置，那么 `origin` 指向您的 Fork 仓库，`upstream` 指向上游主仓库。

### 2) 创建开发分支

建议从最新的主分支创建一个新的功能分支。如果您需要同步上游更改，请先参考[同步上游更改](#-同步上游更改)部分更新本地 main 分支，然后创建新的功能分支：

```bash
git checkout -b feature/your-feature-name
git checkout -b fix/your-bug-fix
```

### 3) 安装 hook 并配置开发环境

安装 Git 预提交钩子（仓库流程要求）：

```bash
make install-hooks
```

然后根据您要开发的组件配置相应环境：

- 后端： [后端开发指南](../../backend/README.zh-CN.md)
- 前端： [前端开发指南](../../frontend/README.zh-CN.md)
- 存储： [存储服务开发指南](../../storage/README_CN.md)
- 文档网站： [文档网站开发指南](../../website/README.zh-CN.md)

### 4) 配置文件管理（可选）

Crater 提供统一的配置管理方式，可将配置集中到单一目录，并为各组件创建软链接。

示例目录结构：

```
config/
├── backend/
│   ├── .debug.env              # 后端调试环境变量
│   ├── kubeconfig              # Kubernetes 配置文件（可选）
│   └── debug-config.yaml       # 后端调试配置
├── frontend/
│   └── .env.development        # 前端开发环境变量
└── storage/
    ├── .env                    # 存储服务环境变量
    └── config.yaml             # 存储服务配置
```

相关 Make 目标：

- `make config-link`: 创建配置文件软链接（如已存在普通文件会备份为 `.bak`）

  ```bash
  make config-link CONFIG_DIR=~/develop/crater/config
  ```

- `make config-status`: 显示配置文件状态
- `make config-unlink`: 仅删除软链接
- `make config-restore`: 从 `.bak` 恢复

### 5) 提交前检查

pre-commit hook 会检查已暂存文件，并仅对受影响的子项目执行检查：

```bash
git add <your-files>
make pre-commit-check
```

也可以在子项目目录内运行（会检查该子项目全部文件）：

```bash
cd frontend && make pre-commit-check
cd backend && make pre-commit-check
cd storage && make pre-commit-check
cd website && make pre-commit-check
```

### 6) 提交并创建 PR

```bash
git status

# 添加更改的文件（请指定具体文件或目录，避免使用 git add .）
git add backend/pkg/handler/user.go
git add frontend/src/components/

git commit -m "feat: add new feature description"
git push origin feature/your-feature-name
```

提交信息约定：

- `feat:` 新功能
- `fix:` Bug 修复
- `docs:` 文档更新
- `style:` 代码风格调整
- `refactor:` 代码重构
- `test:` 测试相关
- `chore:` 构建/工具相关

然后在 GitHub 上创建 Pull Request，并尽量包含：

- 修改内容与原因
- 测试方式
- 截图（如涉及 UI）

### 7)（可选）Squash 提交

```bash
git rebase -i HEAD~3
```

<a id="-同步上游更改"></a>

### 🔄 同步上游更改

如果您已添加 `upstream` 且上游仓库有新更改，可以按以下方式更新本地 main 分支：

```bash
git checkout main
git fetch upstream
git merge upstream/main
# 或使用快捷方式（一步完成）
# git pull upstream main
```

完成后，本地 main 分支已更新，您可以基于它创建新分支进行开发。

如果您已经有一个开发分支，可以把更新合并到该分支：

```bash
git checkout feature/your-feature-name
git merge main
```

如果您未配置 `upstream`，可以在 GitHub 上使用 **Sync fork** 功能同步上游更改，之后在本地执行 `git pull origin main` 更新。

## 许可证

Crater 使用 Apache License 2.0 许可证，详见 [LICENSE](../../LICENSE)。
