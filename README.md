[English](README.md) | [简体中文](docs/zh-CN/README.md)

# ![crater](./website/content/docs/admin/assets/icon.webp) Crater

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

**Crater** is a university-developed cluster management platform designed to provide users with an efficient and user-friendly solution for managing computing clusters. It offers unified scheduling and management of computing, storage, and other resources within a cluster, ensuring stable operation and optimal resource utilization.

## Features

### 🎛️ Intuitive Interface Design

Crater features a clean and easy-to-use graphical user interface that enables users to perform various cluster management tasks effortlessly. The resource dashboard provides real-time insights into key metrics such as CPU utilization, memory usage, and storage capacity.

The job management interface allows users to monitor running jobs, view job queues, and access job history, making it easy to track and control task execution.

### ⚙️ Intelligent Resource Scheduling

The platform employs smart scheduling algorithms to automatically allocate the most suitable resources to each job based on priority, resource requirements, and other factors. For example, when multiple jobs request resources simultaneously, Crater can quickly analyze the situation and prioritize critical and time-sensitive tasks to improve overall efficiency.

### 📈 Comprehensive Monitoring

Crater offers detailed monitoring data and logging capabilities, empowering users with deep visibility into cluster operations. These features facilitate quick troubleshooting and performance tuning, helping maintain system stability and responsiveness.

---
## Overall Architecture
![crater architecture](./website/content/docs/admin/assets/architecture.webp)

## Installation

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

---

## Deployment (via Helm)

If you want to quickly deploy a basic Crater using Kind, please refer to [Minimal Deployment](https://raids-lab.github.io/crater/zh/docs/admin/kind-start/).

If you want to deploy a full Crater in a cluster, please refer to [Cluster Deployment Guide](https://raids-lab.github.io/crater/zh/docs/admin/deploy-on-cluster/).

---

## Development

We welcome community contributions! If you would like to contribute to the Crater project, please follow the development workflow below.

### 🔀 Fork and Clone Repository

1. **Fork the Repository**
   - Visit the [Crater main repository](https://github.com/raids-lab/crater)
   - Click the "Fork" button in the top right corner to fork the repository to your GitHub account

2. **Clone Your Fork**
   ```bash
   # Replace YOUR_USERNAME with your GitHub username
   git clone https://github.com/YOUR_USERNAME/crater.git
   cd crater
   ```

3. **Add Upstream Repository**
   ```bash
   # Add upstream repository to sync latest changes
   git remote add upstream https://github.com/raids-lab/crater.git
   
   # Verify remote repository configuration, you should see both the main repository and your Fork
   git remote -v
   ```
   
   If you configure it this way, `origin` points to your Fork repository, and `upstream` points to the upstream main repository.

   **Alternative:** You can also skip `git remote add upstream` and only connect to your Fork repository. In this case, when you need to sync upstream changes, use the Sync Fork feature on GitHub to sync changes from the upstream main repository.

### 🌿 Create Development Branch

It's recommended to create a new feature branch from the latest main branch. If you need to sync upstream changes, please first refer to the [Sync Upstream Changes](#-sync-upstream-changes) section to update your local main branch, then create a new feature branch:

```bash
# Create and switch to a new feature branch
git checkout -b feature/your-feature-name
# Or use this when fixing bugs
git checkout -b fix/your-bug-fix
```

### ⚙️ Environment Setup

Before starting development, first install Git pre-commit hooks, which will automatically check modified files when you commit code:

```bash
# Execute in the repository root directory
make install-hooks
```

Then configure the development environment for the component you want to develop. For detailed environment setup instructions, please refer to the README of each submodule:

- **Backend Development Environment**: Please refer to [Backend Development Guide](./backend/README.md)
- **Frontend Development Environment**: Please refer to [Frontend Development Guide](./frontend/README.md)
- **Storage Service Development Environment**: Please refer to [Storage Service Development Guide](./storage/README.md)
- **Documentation Website Development Environment**: Please refer to [Documentation Website Development Guide](./website/README.md)

### 📁 Configuration File Management

Crater provides a unified configuration file management system to help developers manage configuration files across different components. This system allows you to centralize all configuration files in a single directory and create symlinks in each project directory.

**Configuration Directory Structure:**

The configuration directory should have the following structure:

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

**Available Make Targets:**

- `make config-link`: Create symlinks for configuration files. If a configuration file already exists as a regular file, it will be backed up with a `.bak` suffix. If it exists as a symlink, it will be replaced.

  ```bash
  make config-link CONFIG_DIR=~/develop/crater/config
  ```

- `make config-status`: Display the status of all configuration files, showing whether they exist, are symlinks, or are missing.

- `make config-unlink`: Remove configuration symlinks (only symlinks, regular files are preserved).

- `make config-restore`: Restore configuration files from `.bak` backups.

### 💻 Development

Enter the corresponding directory for the component you want to modify:

- **Backend Development**: `backend/` directory
- **Frontend Development**: `frontend/` directory
- **Storage Service**: `storage/` directory
- **Documentation**: `website` directory

**Pre-commit Testing:**

Before committing, you can use the following command to run checks in advance to ensure your code meets the standards:

```bash
# Execute in the repository root directory, will check all modified directories
# Note: You must first stage files with 'git add' before running this command
# The hook only checks staged files to determine which sub-projects to check
git add <your-files>
make pre-commit-check
```

Alternatively, you can run checks directly in sub-project directories, which will check all files in that project (not just staged files):

```bash
# Check frontend (checks all files in frontend/)
cd frontend && make pre-commit-check

# Check backend (checks all files in backend/)
cd backend && make pre-commit-check

# Check storage (checks all files in storage/)
cd storage && make pre-commit-check

# Check website (checks all files in website/)
cd website && make pre-commit-check
```

This helps you discover and fix issues early, avoiding being blocked by hooks during commit.

### 📝 Commit Changes

After completing development, commit your changes:

```bash
# View changed files
git status

# Add changed files (please specify specific files or directories, avoid using git add .)
git add backend/pkg/handler/user.go
# Or add an entire directory
git add frontend/src/components/

# Commit changes (please use clear English commit messages)
git commit -m "feat: add new feature description"
```

**Commit Message Convention:**

- **Commit Types**:
  - `feat:` - New feature
  - `fix:` - Bug fix
  - `docs:` - Documentation updates
  - `style:` - Code style changes
  - `refactor:` - Code refactoring
  - `test:` - Test related
  - `chore:` - Build/tool related

**Git Hook Checks:**

If you have installed Git hooks (see Environment Setup section), checks will be automatically triggered when you commit. The hooks will execute corresponding checks based on the directories you modified (`backend/`, `frontend/`, `storage/`, `website/`) (such as lint, format checks, etc.).

If checks fail, the commit will be blocked. Please fix the issues according to the error messages and commit again.

**Squash Multiple Commits:**

If you have multiple commits for one feature, it's recommended to squash them into one commit before pushing to your Fork repository. You can use interactive rebase:

```bash
# Assuming you have 3 commits to squash
git rebase -i HEAD~3
# In the editor, change "pick" to "squash" or "s" for the last two commits
# After saving, Git will prompt you to edit the merged commit message
```

### 🚀 Push to Fork Repository

Push your changes to your Fork repository:

```bash
# Push to your Fork repository
git push origin feature/your-feature-name
```

### 📤 Create Pull Request

1. **Create PR on GitHub**
   - Visit your Fork repository page
   - Click the "Compare & pull request" button
   - Or visit the main repository, click "New pull request", and select your Fork and branch

   Actually, after you push your changes, you'll see a prominent prompt to create a PR, and you can also click the prompt directly to create a PR.

2. **Self-review Changes**

   Before creating the PR, please review your own changes first, carefully check all changes (Changes), and ensure each modification is as expected, without including files or code that shouldn't be committed.

3. **Fill in PR Information**
   - **Title**: GitHub will set the PR title to the first line of the latest commit message. Please ensure it conforms to conventions and is accurate. If not, please modify it before creating the PR.
   - **Description**: Provide detailed information:
     - The reason and content of the changes
     - How to test these changes
     - Screenshots of the modified effects (if frontend changes are involved)
     - References to related Issues (if any)

4. **Review and Modify**

   Maintainers will review your PR. Please make necessary modifications based on feedback, and push to the same branch after modifications. The PR will be automatically updated.

### 🔄 Sync Upstream Changes

If you have added `upstream` and there are new changes in the upstream repository, you can update your local main branch as follows:

```bash
# Switch to main branch
git checkout main

# Fetch and merge upstream changes
git fetch upstream
git merge upstream/main
# Or use the shortcut (one step)
# git pull upstream main
```

After completing the above steps, your local main branch is updated, and you can create new feature branches based on it for development.

**Additionally, if you already have a development branch, you need to merge updates into it:**

```bash
# After updating the local main branch, switch to your feature branch
git checkout feature/your-feature-name

# Merge main branch changes into your feature branch
git merge main
```

Additionally, if you haven't configured `upstream` remote, you can use the Sync Fork feature on GitHub to sync upstream changes. After the main branch of your Fork repository is updated, use `git pull origin main` to update it locally.

---

Thank you for contributing to the Crater project! 🎉