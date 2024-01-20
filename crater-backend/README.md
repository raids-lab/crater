<h1 align="center">Crater Web Backend</h1>

 [![Pipeline Status](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/badges/main/pipeline.svg) ](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/-/commits/main)
 [![Develop Version](https://img.shields.io/badge/Develop-0.1.0-orange) ](http://***REMOVED***:8888/)
 [![Release Version](https://img.shields.io/badge/Release-0.1.0-blue) ](http://***REMOVED***:32088/)

Crater 是一个基于 Kubernetes 的 GPU 集群管理系统，提供了一站式的 GPU 集群管理解决方案。要了解更多信息，请访问 [GPU 集群管理与作业调度 Portal 设计和任务分解](***REMOVED***) 。

## 1. 环境准备

在开始之前，请确保您的开发环境中已安装 Go 和 Kubectl。如果尚未安装，请参考以下步骤：

- Go: [Download and install](https://go.dev/doc/install)
- Kubectl: [Install Tools | Kubernetes](https://kubernetes.io/docs/tasks/tools/)

以 Ubuntu 系统为例，使用如下命令安装匹配的版本：

```bash
# build essential
sudo apt-get install build-essential

# install go
rm -rf /usr/local/go
wget -qO- https://go.dev/dl/go1.19.13.linux-amd64.tar.gz | sudo tar xz -C /usr/local

# ~/.zshrc
export PATH=$PATH:/usr/local/go/bin
export GOPROXY=https://goproxy.cn

# install kubectl
curl -LO https://dl.k8s.io/release/v1.22.1/bin/linux/amd64/kubectl
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
```

之后需要获取 K8s 集群的访问权限。申请通过后，集群管理员会提供 `user-xxx.kubeconfig` 文件，创建 `~/.kube` 目录，并将 `user-xxx.kubeconfig` 文件放置在该路径下，仍以 Ubuntu 系统为例：

```bash
mkdir -p ~/.kube
# Kubectl 默认配置文件路径位于 `~/.kube/config`
cp ./${user-xxx.kubeconfig} ~/.kube/config
```

检查 Go 和 Kubectl 是否安装成功，Kubectl 是否连接集群：

```bash
go version
# v1.19.13

kubectl version
# Client Version: version.Info{Major:"1", Minor:"22", GitVersion:"v1.22.1", ...}
# Server Version: version.Info{Major:"1", Minor:"22", GitVersion:"v1.22.1", ...}
```

## 2. 开发

Crater 目前部署于 [K8s 小集群](https://gitlab.***REMOVED***/raids/resource-scheduling/gpu-cluster-portal/-/wikis/home) 中，在 Web Backend 下游，集群中还有以下组件：

- [MySQL](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/-/tree/main/deploy/mysql?ref_type=heads) ：Web Backend 所使用的数据库
- [AI Job Controller](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/aijob-controller) ：连接 Web Backend 与调度层的中间层
  1. 维护 AIJob 队列，进行 AI Job 到 Pod 的转换工作，将 Pod 提交到调度层
  2. 监控 Pod 生命周期，将 Pod 的状态同步到 AI Job 里，反馈给  Web Backend
- [AI Job Scheduler](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/aijob-scheduler) ：Crater 的调度层，实现了 Best Effort 作业抢占等机制

为便于开发人员测试，目前将 MySQL 数据库的 3306 端口暴露到集群外的 30306 端口（见 `deploy/mysql/mysql-hack.yaml` ），使用 `./debug.sh` 脚本在本地 `8099` 端口运行：

```bash
#!/bin/bash
go run main.go \
    --db-config-file ./debug-dbconf.yaml \
    --config-file ./etc/debug-config.yaml \
    --metrics-bind-address :8097 \
    --health-probe-bind-address :8096 \
    --server-port :8099
```

完成新功能开发后，可以用 Postman 自测。可以在 Header 中添加 `X-Debug-Username` 指定用户名绕过登录认证，直接测试接口功能。

```json
{
  "X-Debug-Username": "lyl"
}
```

也可以在本地运行 [Web Frontend](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-frontend) 进行测试，在 `pkg/server/middleware/cors.go` 中，允许了来自 `http://localhost:5173` 的跨域请求。

合入 `main` 分支后，在 ***REMOVED*** 运行 `docker restart ai-portal-backend`，会启动 main 分支版本，暴露 8078 端口。

## 3. 部署

(WIP)

## 4. 项目结构

(WIP)

## 5. 其他

- 旧文档位于 `docs/README.md`。
