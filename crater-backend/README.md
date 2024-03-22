<h1 align="center">Crater Web Backend</h1>

 [![Pipeline Status](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/badges/main/pipeline.svg) ](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/-/commits/main)
 [![Release Version](https://img.shields.io/badge/Release-0.1.0-blue) ](https://crater.***REMOVED***/)

Crater 是一个基于 Kubernetes 的 GPU 集群管理系统，提供了一站式的 GPU 集群管理解决方案。

- 网站访问：https://crater.***REMOVED***/
- 需求分析：[GPU 集群管理与作业调度 Portal 设计和任务分解](***REMOVED***)
- 任务排期：[Crater Group Milestone](https://gitlab.***REMOVED***/groups/raids/resource-scheduling/crater/-/milestones)


## 1. 环境准备

### 1.1 安装 Go 和 Kubectl

> 您不需要在本地安装 MiniKube 或 K3s 集群

在开始之前，请确保您的开发环境中已安装 Go 和 Kubectl。如果尚未安装，请参考以下步骤：

- Go: [Download and install](https://go.dev/doc/install)
- Kubectl: [Install Tools | Kubernetes](https://kubernetes.io/docs/tasks/tools/)

以 Ubuntu 系统为例，使用如下命令安装匹配的版本：

```bash
# build essential
sudo apt-get install build-essential

# install go
rm -rf /usr/local/go
wget -qO- https://go.dev/dl/go1.22.1.linux-amd64.tar.gz | sudo tar xz -C /usr/local

# ~/.zshrc
export PATH=$PATH:/usr/local/go/bin
export GOPROXY=https://goproxy.cn

# install kubectl
curl -LO https://dl.k8s.io/release/v1.22.1/bin/linux/amd64/kubectl
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
```

### ~~1.2 获取集群访问权限~~

> 目前 Crater Backend 直接使用位于项目根目录的 `/kubeconfig` 文件作为 Context，这种方式并不正规，但因此，您可以忽略这一步。
>
> 请不要在 Crater 外，使用项目根目录的 `/kubeconfig` 文件连接到集群。

之后需要获取 K8s 集群的访问权限。申请通过后，集群管理员会提供 `user-xxx.kubeconfig` 文件，创建 `~/.kube` 目录，并将 `user-xxx.kubeconfig` 文件放置在该路径下，仍以 Ubuntu 系统为例：

```bash
mkdir -p ~/.kube
# Kubectl 默认配置文件路径位于 `~/.kube/config`
cp ./${user-xxx.kubeconfig} ~/.kube/config
```

### 1.3 环境检查

检查 Go 和 Kubectl 是否安装成功，Kubectl 是否连接集群（如果您未进行 1.2，则 Kubectl 将仅显示 Client 版本，这是预期行为）：

```bash
go version
# v1.22.1

kubectl version
# Client Version: version.Info{Major:"1", Minor:"22", GitVersion:"v1.22.1", ...}
# Server Version: version.Info{Major:"1", Minor:"22", GitVersion:"v1.22.1", ...}
```

## 2. 开发

### 2.1 系统概况

Crater 目前部署于 [K8s 小集群](https://gitlab.***REMOVED***/raids/resource-scheduling/gpu-cluster-portal/-/wikis/home) 中，在 Web Backend 下游，集群中还有以下组件：

- [MySQL](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/-/tree/main/deploy/mysql?ref_type=heads) ：Web Backend 所使用的数据库
- [AI Job Controller](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/aijob-controller) ：连接 Web Backend 与调度层的中间层
  1. 维护 AIJob 队列，进行 AI Job 到 Pod 的转换工作，将 Pod 提交到调度层
  2. 监控 Pod 生命周期，将 Pod 的状态同步到 AI Job 里，反馈给  Web Backend
- [AI Job Scheduler](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/aijob-scheduler) ：Crater 的调度层，实现了 Best Effort 作业抢占等机制

为便于开发人员测试，目前将 MySQL 数据库的 3306 端口暴露到集群外的 30306 端口（见 `deploy/mysql/mysql-hack.yaml` ）。

### 2.2 本地开发

首先，您需要下载项目所使用的依赖：

```bash
go mod download
```

如果您在使用 Linux 或 MacOS 系统，可使用 `./debug.sh` 脚本，在本地 `8099` 端口运行 Web 后端：

```bash
#!/bin/bash
export KUBECONFIG=${PWD}/kubeconfig
go run main.go \
    --db-config-file ./debug-dbconf.yaml \
    --config-file ./etc/debug-config.yaml \
    --metrics-bind-address :8097 \
    --health-probe-bind-address :8096 \
    --server-port :8099
```

如果您在使用 Windows 系统，上述脚本可能需要修改为适用于 Windows 的版本（等待一位好心人！）

### 2.3 单步调试

Crater Web Backend 已经为 VSCode 配置好了单步调试设置，通过点击 VSCode 左侧的 Run and Debug (Ctrl + Shift + D) 按钮，并点击 `Debug Server` 左侧的 Start Debugging (F5) 按钮，可以启动调试模式。此时，您可以在代码中添加断点，进行单步调试。

### 2.4 如何测试接口

完成新功能开发后，可以用 Postman 自测。可以在 Header 中添加 `X-Debug-Username` 指定用户名绕过登录认证，直接测试接口功能。

```json
{
  "X-Debug-Username": "YOUR_USERNAME"
}
```

也可以在本地运行 [Web Frontend](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-frontend) 进行测试。

由于调试时前后端不同域，在 `pkg/server/middleware/cors.go` 中，允许了来自 `http://localhost:5173` 的跨域请求。

前端可能会有 `http://localhost:5173`, `http://127.0.0.1:5173` 这两种 URL，视操作系统的不同，前端 Vite 程序可能会引导至二者之一，建议您使用 `http://localhost:5173` 访问前端，避免跨域问题。


## 3. 部署

### 3.1 首次部署

与部署相关的文件位于 deploy/ 文件夹下。

```bash
deploy/
├── backend
│   ├── crater-backend-ingress.yaml # 后端 Ingress
│   ├── deploy.yaml                 # 部署后端到集群
│   └── libs
│       ├── backend-config.yaml     # 基本配置 ConfigMap
│       └── share-dir.yaml          # 共享目录 ConfigMap
└── mysql
    ├── mysql-cluster
    │   ├── cluster-ceph.yaml       # MySQL Cluster
    │   ├── mysql-hack.yaml         # MySQL NodePort
    │   └── secret.yaml             # MySQL Secret
    └── mysql-operator
        ├── deploy-crds.yaml
        └── deploy-operator.yaml
```

### 3.2 GitLab CI/CD

完成部署后，要更新代码变动到集群中时，只需打上相应的标签。

```bash
git tag v0.x.x
git push origin v0.x.x
```

使用命令行，或在 Gitlab 网页端操作，GitLab CI/CD 会根据标签自动部署。

### 3.3 证书过期

ACT 的 HTTPS 证书每 3 个月更新一次，证书更新方法见 Web Frontend 项目。



## 4. 项目结构

> [Wiki 代码架构](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/-/wikis/%E4%BB%A3%E7%A0%81%E6%9E%B6%E6%9E%84)

主要代码逻辑在pkg文件夹下：

* apis：crd的定义。
* control：提供接口，负责在集群创建具体的对象，例如pod、aijob等。
* **controller**：负责同步各crd的状态
  * job_controller.go：控制job的状态变化
  * pod.go：监听pod的状态变化。
  * quota_controller
  * quota_info.go
* db：数据库相关存储
  * internal：db的底层操作
  * task
  * quota
  * user
* generated：k8s生成的clientset
* models：数据模型
  * aitask
  * quota
  * user
* **server**：服务端接口和响应
  * handlers：具体响应，操作数据库
  * payload：外部请求接口的定义
* **taskqueue**：维护用户的任务队列，检查什么时候应该调度作业
* profiler：负责对任务进行profile

## 5. 其他

- 旧文档位于 `docs/README.md`。
