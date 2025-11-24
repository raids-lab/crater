[English](README.md) | [简体中文](README_CN.md)

# Crater存储服务

Crater 是一个基于 Kubernetes 的 GPU 集群管理系统，提供 GPU 资源编排的全面解决方案。

## 💻 开发指南

在开始开发之前，请确保您的环境已安装以下工具：

- **Go**：推荐版本 `v1.25.4`  
  📖 [Go 安装指南](https://go.dev/doc/install)

- **Kubectl**：推荐版本 `v1.33`  
  📖 [Kubectl 安装指南](https://kubernetes.io/docs/tasks/tools/)

具体的安装方法，请参照后端仓库 [README](../backend/README.md)。

### 📐 代码风格与检查

本项目使用 [`golangci-lint`](https://golangci-lint.run/) 来强制执行 Go 代码约定和最佳实践。为避免手动运行，我们在 Makefile 中添加了 `pre-commit-check` 目标来执行相关操作，这个目标将会被 Crater 主仓库的 Git 预提交钩子使用。要安装这个钩子，请参阅 Crater 主仓库 [README](../docs/zh-CN/README.md)。

本仓库的 `pre-commit-check` 目标优先使用后端仓库中安装的 golangci-lint，如果不可用，则会尝试使用您本地安装的 golangci-lint。您可以通过以下命令来在本地安装它（**不推荐**），在 Linux 上：

```bash
# 检查您的 GOPATH
go env GOPATH
# /Users/your-username/go

# 将路径添加到 .bashrc 或 .zshrc
export PATH="/Users/your-username/go/bin:$PATH"

# 二进制文件将会被安装到 $(go env GOPATH)/bin/golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.6.2

# 重新加载 shell 并验证
golangci-lint --version
# golangci-lint has version 1.64.8
```

### 🛠️ 数据库代码生成
本项目使用 GORM Gen 来生成数据库 CRUD 操作的样板代码。

生成脚本和文档可在以下位置找到：[ `gorm_gen`](./cmd/gorm-gen/README.md)

修改数据库模型或架构定义后，请重新生成代码，而 CI 流水线将自动进行数据库迁移。

### 项目配置
安装依赖和插件：
```bash
go mod download
```

## 🚀 运行代码

本项目支持两种运行方式：**本地开发** 和 **部署到 Kubernetes 集群**。我们 **推荐使用 Kubernetes 部署** 以获得完整功能和更接近生产的行为。

---

### 🧑‍💻 本地运行

> 适用于快速测试和开发阶段。

#### 📄 配置：

确保您有一个 [config.yaml](./etc/config.yaml) 文件，其中包含正确的数据库设置。

在根目录创建 `.env` 文件以自定义本地端口。此文件被 Git 忽略：

```env
PORT=xxxx
ROOTDIR="/crater"
```

#### 📁 目录设置：

**在你熟悉的目录下创建一个名为 `crater`（或者其他名字） 的文件夹，以模拟文件处理行为。**

**或者，您可以修改 .env 文件中的 `ROOTDIR` 并将其用作测试的根目录。**

```bash
mkdir crater
```

此目录将作为文件处理的根目录。

#### 🚀 运行应用程序：

```bash
make run
```

服务将启动并默认监听 `localhost:port`。

---

### ☸️ 部署到 Kubernetes

#### ✅ 先决条件：

- Docker
- 访问 Kubernetes 集群（`kubectl`）
- 已创建名为 `crater-rw-storage` 的 PVC（用于持久文件存储）

#### 📦 构建并推送 Docker 镜像：

```bash
docker build -t your-registry/crater-webdav:latest .
docker push your-registry/crater-webdav:latest
```

> 将 `your-registry` 替换为您的实际容器注册表。

#### 🚀 部署到 Kubernetes：

确保当前目录中存在以下文件：

- `Dockerfile`
- `deployment.yaml`
- `service.yaml`（如果适用）

您可以在 https://github.com/raids-lab/crater/tree/main/charts/crater/templates/storage-server 找到这些文件

应用清单：

```bash
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

> 确保 `deployment.yaml` 正确引用镜像并挂载 PVC `crater-rw-storage`。

### 🚀 快速部署
要在生产环境中部署 Crater 项目，我们提供了一个 Helm Chart，可在 [Crater Helm Chart](https://github.com/raids-lab/crater) 获取。

请参考主文档以获取