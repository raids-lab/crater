[English](../../README.md) | [简体中文](README.md)

# ![crater](../../website/content/docs/admin/assets/icon.webp) Crater

<table>
  <tr>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/jupyter.gif"><br>
      <em>Jupyter Lab</em>
    </td>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/ray.gif"><br>
      <em>Ray Job</em>
    </td>
  </tr>
  <tr>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/monitor.gif"><br>
      <em>Monitor</em>
    </td>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/datasets.gif"><br>
      <em>Models</em>
    </td>
  </tr>
</table>

**Crater** 是一个由大学开发的集群管理平台，旨在为用户提供高效且易用的计算集群管理解决方案。它提供集群内计算、存储和其他资源的统一调度和管理，确保稳定运行和资源的最优利用。

## 功能特性

### 🎛️ 直观的界面设计

Crater 具有简洁易用的图形用户界面，使用户能够轻松执行各种集群管理任务。资源仪表板提供关键指标的实时洞察，如 CPU 利用率、内存使用情况和存储容量。

作业管理界面允许用户监控运行中的作业、查看作业队列和访问作业历史，便于跟踪和控制任务执行。

### ⚙️ 智能资源调度

该平台采用智能调度算法，根据优先级、资源需求和其他因素自动为每个作业分配最合适的资源。例如，当多个作业同时请求资源时，Crater 可以快速分析情况并优先处理关键和时间敏感的任务，以提高整体效率。

### 📈 全面监控

Crater 提供详细的监控数据和日志记录功能，使用户能够深入了解集群操作。这些功能有助于快速故障排除和性能调优，帮助维护系统的稳定性和响应性。

---

## 整体架构
![crater architecture](../../website/content/docs/admin/assets/architecture.webp)

## 安装

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

---

## 部署（通过 Helm）

如果您希望使用 Kind 快速部署一个基本的 Crater，请参照[最小化部署](https://raids-lab.github.io/crater/zh/docs/admin/kind-start/)。

如果您希望在集群中部署一个完整的 Crater，请参照[集群部署指南](https://raids-lab.github.io/crater/zh/docs/admin/deploy-on-cluster/)。

---

## 开发

我们欢迎社区贡献！如果您想为 Crater 项目做出贡献，请遵循以下开发流程。

### 🔀 Fork 和克隆仓库

1. **Fork 仓库**
   - 访问 [Crater 主仓库](https://github.com/raids-lab/crater)
   - 点击右上角的 "Fork" 按钮，将仓库 Fork 到您的 GitHub 账户

2. **克隆您的 Fork**
   ```bash
   # 将 YOUR_USERNAME 替换为您的 GitHub 用户名
   git clone https://github.com/YOUR_USERNAME/crater.git
   cd crater
   ```

3. **添加上游仓库**
   ```bash
   # 添加上游仓库以便同步最新更改
   git remote add upstream https://github.com/raids-lab/crater.git
   
   # 验证远程仓库配置，您应该可以看到主仓库和您的 Fork 仓库
   git remote -v
   ```
   
   如果您按照这种方式配置，那么 `origin` 指向您的 Fork 仓库，`upstream` 指向上游主仓库。

   **替代方式：** 您也可以不使用 `git remote add upstream`，只连接 Fork 仓库。在这种情况下，当需要同步上游更改时，请在 GitHub 上使用 Sync Fork 功能来同步上游主仓库的更改。

### 🌿 创建开发分支

建议从最新的主分支创建一个新的功能分支。如果您需要同步上游更改，请先参考[同步上游更改](#-同步上游更改)部分更新本地 main 分支，然后创建新的功能分支：

```bash
# 创建并切换到新的功能分支
git checkout -b feature/your-feature-name
# 或修复 bug 时使用
git checkout -b fix/your-bug-fix
```

### ⚙️ 环境配置

在开始开发之前，请先安装 Git 预提交钩子，它会在您提交代码时自动检查修改的文件：

```bash
# 在仓库根目录执行
make install-hooks
```

然后根据您要开发的组件配置相应的开发环境。详细的环境配置说明请参考各子模块的 README：

- **后端开发环境**: 请参考 [后端开发指南](../../backend/README.zh-CN.md)
- **前端开发环境**: 请参考 [前端开发指南](../../frontend/README.zh-CN.md)
- **存储服务开发环境**: 请参考 [存储服务开发指南](../../storage/README_CN.md)
- **文档网站开发环境**: 请参考 [文档网站开发指南](../../website/README.zh-CN.md)

### 💻 进行开发

根据您要修改的组件，进入相应的目录进行开发：

- **后端开发**: `backend/` 目录
- **前端开发**: `frontend/` 目录
- **存储服务**: `storage/` 目录
- **文档**: `website` 目录

**提前测试检查：**

在提交之前，您可以使用以下命令提前执行检查，确保代码符合规范：

```bash
# 在仓库根目录执行，会检查所有修改的目录
# 注意：必须先使用 'git add' 暂存文件，然后才能运行此命令
# 钩子只会检查暂存的文件来决定需要检查哪些子项目
git add <your-files>
make pre-commit-check
```

或者，您也可以直接在子项目目录中运行检查，这会检查该项目的所有文件（不仅仅是暂存的文件）：

```bash
# 检查前端（检查 frontend/ 中的所有文件）
cd frontend && make pre-commit-check

# 检查后端（检查 backend/ 中的所有文件）
cd backend && make pre-commit-check

# 检查存储服务（检查 storage/ 中的所有文件）
cd storage && make pre-commit-check

# 检查文档网站（检查 website/ 中的所有文件）
cd website && make pre-commit-check
```

这样可以提前发现问题并修复，避免在提交时被钩子阻止。

### 📝 提交更改

完成开发后，提交您的更改：

```bash
# 查看更改的文件
git status

# 添加更改的文件（请指定具体的文件或目录，避免使用 git add .）
git add backend/pkg/handler/user.go
# 或添加整个目录
git add frontend/src/components/

# 提交更改（请使用清晰的英文提交信息）
git commit -m "feat: add new feature description"
```

**提交信息规范：**

- **提交类型**：
  - `feat:` - 新功能
  - `fix:` - 修复 bug
  - `docs:` - 文档更新
  - `style:` - 代码格式调整
  - `refactor:` - 代码重构
  - `test:` - 测试相关
  - `chore:` - 构建/工具相关

**Git 钩子检查：**

如果您已安装 Git 钩子（见环境配置部分），在提交时会自动触发检查。钩子会根据您修改的目录（`backend/`、`frontend/`、`storage/`、`website/`）执行相应的检查（如 lint、格式检查等）。

如果检查未通过，提交会被阻止。请根据错误信息修复问题后重新提交。

**合并多个提交：**

如果对于一个功能有多个提交，建议在推送到 Fork 仓库前将它们合并成一个提交。可以使用交互式 rebase：

```bash
# 假设您有 3 个提交需要合并
git rebase -i HEAD~3
# 在编辑器中，将后两个提交的 "pick" 改为 "squash" 或 "s"
# 保存后，Git 会提示您编辑合并后的提交信息
```

### 🚀 推送到 Fork 仓库

将您的更改推送到您的 Fork 仓库：

```bash
# 推送到您的 Fork 仓库
git push origin feature/your-feature-name
```

### 📤 创建 Pull Request

1. **在 GitHub 上创建 PR**
   - 访问您的 Fork 仓库页面
   - 点击 "Compare & pull request" 按钮
   - 或访问主仓库，点击 "New pull request"，选择您的 Fork 和分支

   实际上，在您推送修改后，您会看到明显的创建 PR 的提示，也可以直接点击提示来创建 PR。

2. **自我审查修改**

   在创建 PR 之前，请先自己进行 review，仔细查看所有的更改（Changes），确保每个修改都是符合预期的，没有包含不应该提交的文件或代码。

3. **填写 PR 信息**
   - **标题**: GitHub 会将 PR 标题设置为最新提交信息的第一行，请确保它是符合规范和准确的，如果不是，请在创建 PR 前修改
   - **描述**: 详细说明：
     - 更改的原因和内容
     - 如何测试这些更改
     - 修改后效果的截图（如果涉及前端的修改）
     - 对相关 Issue 的引用（如有）

4. **审查与修改**

   维护者会审查您的 PR，请根据反馈进行必要的修改，修改后推送到同一分支，PR 会自动更新。

### 🔄 同步上游更改

如果您添加了 `upstream`，且上游仓库有新的更改，您可以使用如下方式更新本地的 main 分支：

```bash
# 切换到主分支
git checkout main

# 获取并合并上游更改
git fetch upstream
git merge upstream/main
# 或者使用快捷方式（一步完成）
# git pull upstream main
```

完成以上步骤后，您的本地 main 分支已更新，可以基于它创建新的功能分支进行开发。

**另外，如果您已有开发分支，需要将更新合并到开发分支：**

```bash
# 在更新本地 main 分支后，切换到您的功能分支
git checkout feature/your-feature-name

# 将主分支的更改合并到您的功能分支
git merge main
```

此外，如果您没有配置 `upstream` remote，可以在 GitHub 上使用 Sync Fork 功能同步上游更改，Fork 仓库的主分支更新后，使用 `git pull origin main` 将其更新到本地。

感谢您对 Crater 项目的贡献！🎉
