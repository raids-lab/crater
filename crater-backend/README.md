<h1 align="center">Crater Web Backend</h1>

 [![Pipeline Status](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/badges/main/pipeline.svg) ](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/-/commits/main)
 [![Develop Version](https://img.shields.io/badge/Develop-0.1.0-blue) ](https://crater.***REMOVED***/)
 [![Release Version](https://img.shields.io/badge/Release-0.1.0-blue) ](https://***REMOVED***/)

Crater 是一个基�? Kubernetes �? GPU 集群管理系统，提供了�?站式�? GPU 集群管理解决方案�?

- 网站访问：https://***REMOVED***/
- �?求分析：[GPU 集群管理与作业调�? Portal 设计和任务分解](***REMOVED***)
- 任务排期：[Crater Group Milestone](***REMOVED***)


## 1. 环境准备

### 1.1 安装 Go �? Kubectl

> 您不�?要在本地安装 MiniKube �? Kind 集群，我们将使用 ACT 实验室的 GPU 小集群开�?

在开始之前，请确保您的开发环境中已安�? Go �? Kubectl。如果尚未安装，请参考官方文档：

- Go v1.22.1: [Download and install](https://go.dev/doc/install)
- Kubectl v1.22.1: [Install Tools | Kubernetes](https://kubernetes.io/docs/tasks/tools/)

```bash
# Ubuntu 如果安装 Go 时报错，很可能是缺失 build-essential
sudo apt-get install build-essential

# 设置 Go 中国源，否则无法拉取 Github 的包
go env -w GOPROXY=https://goproxy.cn,direct
```

### 1.2 获取集群访问权限

之后�?要获�? K8s 集群的访问权限�?�申请�?�过后，集群管理员会提供 `kubeconfig.yaml` 文件，创�? `~/.kube` 目录，并�? `kubeconfig.yaml` 文件重命名后放置在该路径下，仍以 Ubuntu 系统为例�?

```bash
mkdir -p ~/.kube
# Kubectl 默认配置文件路径位于 `~/.kube/config`
cp ./${kubeconfig.yaml} ~/.kube/config
```

### 1.3 环境�?�?

�?�? Go �? Kubectl 是否安装成功，版本是否与项目推荐配置匹配，Kubectl 是否连接集群（如果您未进�? 1.2，则 Kubectl 将仅显示 Client 版本，这是预期行为）�?

```bash
go version
# go version go1.22.1 linux/amd64

kubectl version
# Client Version: version.Info{Major:"1", Minor:"26", GitVersion:"v1.26.9", GitCommit:"d1483fdf7a0578c83523bc1e2212a606a44fd71d", GitTreeState:"clean", BuildDate:"2023-09-13T11:32:41Z", GoVersion:"go1.20.8", Compiler:"gc", Platform:"linux/amd64"}
# Kustomize Version: v4.5.7
# Server Version: version.Info{Major:"1", Minor:"26", GitVersion:"v1.26.9", GitCommit:"d1483fdf7a0578c83523bc1e2212a606a44fd71d", GitTreeState:"clean", BuildDate:"2023-09-13T11:25:26Z", GoVersion:"go1.20.8", Compiler:"gc", Platform:"linux/amd64"}
```

## 2. 项目�?�?

### 2.1 系统概况

Crater 目前部署�? [K8s 小集群](https://gitlab.***REMOVED***/raids/resource-scheduling/gpu-cluster-portal/-/wikis/home) 中，�? Web Backend 下游，集群中还有以下组件�?

- [](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/-/tree/main/deploy/mysql?ref_type=heads) ：Web Backend �?使用的数据库
- [AI Job Controller](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/aijob-controller) ：连�? Web Backend 与调度层的中间层
  1. 维护 AIJob 队列，进�? AI Job �? Pod 的转换工作，�? Pod 提交到调度层
  2. 监控 Pod 生命周期，将 Pod 的状态同步到 AI Job 里，反馈�?  Web Backend
- [AI Job Scheduler](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/aijob-scheduler) ：Crater 的调度层，实现了 Best Effort 作业抢占等机�?

为便于开发人员测试，目前�? MySQL 数据库的 3306 端口暴露到集群外�? 30306 端口（见 `deploy/mysql/mysql-hack.yaml` ），数据库的密码�? `etc/debug-config.yaml`�?

### 2.2 本地�?�?

- **VSCode**：可导入 `.vscode` 文件夹中�? Profile 设置
- **Goland**：[Wikis �? JetBrain configuration](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/-/wikis/JetBrain-configuration)

配置�? IDE 后，您需要下载项目所使用的依赖：

```bash
go mod download
```

#### 2.2.1 在命令行窗口运行后端

如果您在使用 Linux �? MacOS 系统，可使用 `make run` 命令，在本地 `8099` 端口手动运行 Web 后端�?

```bash
#!/bin/bash
export KUBECONFIG=${PWD}/kubeconfig
go run main.go \
    --config-file ./etc/debug-config.yaml \
    --server-port :8099
```

如果您在使用 Windows 系统，请继续阅读�?

#### 2.2.2 通过 IDE 运行后端

- 如果您使�? VSCode，可通过 `Run` 选项卡下�? `Run without Debugging` (Ctrl + F5) 启动后端
- 如果您在使用 Goland，请参�?? [Wikis �? JetBrain configuration](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/-/wikis/JetBrain-configuration) 进行配置

### 2.3 代码风格�? Lint

项目使用 `golangci-lint` 工具规范代码格式�?

- [如何安装 `golangci-lint`](https://golangci-lint.run/welcome/install/#local-installation)
- [�? `golangci-lint` �? IDE 集成](https://golangci-lint.run/welcome/integrations/)

安装后，您可能需要将 `GOPATH` 添加到系统变量中，才可以在命令行中使�? `golangci-lint` 工具。以 Linux 系统为例�?

```bash
# 打印 GOPATH 位置
go env GOPATH
# /Users/xxx/go

# �? .zshrc �? .bashrc 的最后，更新系统变量
export PATH="/Users/xxx/go/bin:$PATH"

# 测试 `golangci-lint` 是否安装成功
golangci-lint --version
# golangci-lint has version 1.57.1 built with go1.22.1 from cd890db2 on 2024-03-20T16:34:34Z

# 运行 Lint
golangci-lint run
```

为了避免手动运行，建议您配置 Git Hooks，从而允许在每次 commit 之前，自动检查代码是否符合规范�?�将位于项目根目录的 `.githook/pre-commit` 脚本复制�? `.git/` 文件夹下，并提供执行权限�?

�? Linux 系统为例�?

```bash
cp .githook/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

�? Windows 系统下，您可能需要修�?  `.githook/pre-commit` 脚本内容，如将脚本中 `golangci-lint` 替换�? `golangci-lint.exe`。（如果您完成了配置，请联系 LYL 补充文档�?

提交到仓库后，Gitlab CI 将自动运行代码检查，只允许�?�过 Lint 的代码合入主分支�?

此外，在代码中传递错误信息时�?

> > [Go standards and style guidelines](https://docs.gitlab.com/ee/development/go_guide/)
>
> A few things to keep in mind when adding context:
>
> 添加上下文时要记住以下几点：
>
> Don’t use words like failed, error, didn't. As it’s an error, the user already knows that something failed and this might lead to having strings like failed xx failed xx failed xx. Explain what failed instead.
>
> 不要使用 failed �? error �? didn't 等词语�?�由于这是一个错误，用户已经知道某些事情失败了，这可能会导致出现�? failed xx failed xx failed xx 这样的字符串。解释一下失败的原因�?

Lint 还不能检查错误信息的内容，因此您应该尽量遵守这一点�??

### 2.4 单步调试

- **VSCode**: 通过 Start Debugging (F5) 的默认配置，可以启动调试模式。此时，您可以在代码中添加断点，进行单步调试
- **Goland**: 应该更简�?


### 2.5 如何测试接口

#### 2.5.1 通过本地运行前端

可以在本地运�? [Web Frontend](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-frontend) 进行测试�?

由于调试时前后端不同域，�? `pkg/server/middleware/cors.go` 中，允许了来�? `http://localhost:5173` 的跨域请求�??

前端可能会有 `http://localhost:5173`, `http://127.0.0.1:5173` 这两�? URL，视操作系统的不同，前端 Vite 程序可能会引导至二�?�之�?，建议您使用 `http://localhost:5173` 访问前端，避免跨域问题�??

## 3. 部署

1. �? Kubeconfig 指向生产集群，检查一下节点是否是生产集群的机器�??

```shell
export KUBECONFIG=$HOME/.kube/config_actgpu
```

2. 由于�? Gitlab Pipeline 中，我们不会使用 Helm 去更新，导致生产环境的前端�?�后端�?�存储后端的版本�?�?和当前的 chart 中的版本不同，为此提供了�?个脚本，用于将镜像版本写入到 Values 中：

```shell
$ ./hack/helm-current-version.sh             
web-backend 镜像：crater-***REMOVED***/crater/web-backend:02b20b9f
web-frontend 镜像：crater-***REMOVED***/crater/web-frontend:ad7b3722
webdav 镜像：crater-***REMOVED***/crater/webdav:lc9v6p5s
镜像信息已在 charts/crater/values.yaml 中替换完成�??
```

3. 准备好证书，�? `*.***REMOVED***` 的证书解压到项目根目录，已经忽略了解压后的文件夹�?

```shell
$ unzip ***REMOVED***-certs-until-2025-06-11.zip
解压�? ***REMOVED*** 文件�?

$ tree ***REMOVED***
***REMOVED***
├─�? ***REMOVED***.cer
├─�? ***REMOVED***.conf
├─�? ***REMOVED***.csr
├─�? ***REMOVED***.csr.conf
├─�? ***REMOVED***.key
├─�? ca.cer
└─�? fullchain.cer
```

4. 确认 `***REMOVED***/fullchain.cer` �? `***REMOVED***/***REMOVED***.key` 文件存在，先 Dry Run 查看结果�?

```shell
helm upgrade --install crater ./charts/crater \
--namespace crater \
--create-namespace \
--set-string tls.base.cert="$(cat ***REMOVED***/fullchain.cer)" \
--set-string tls.base.key="$(cat ***REMOVED***/***REMOVED***.key)" \
--set-string tls.forward.cert="$(cat ***REMOVED***/fullchain.cer)" \
--set-string tls.forward.key="$(cat ***REMOVED***/***REMOVED***.key)" \
--set tls.base.enabled=true \
--set tls.forward.enabled=true \
--dry-run
```

5. 移除 `--dry-run` 参数，正式安装�??

```shell
helm upgrade --install crater ./charts/crater \
--namespace crater \
--create-namespace \
--set-string tls.base.cert="$(cat ***REMOVED***/fullchain.cer)" \
--set-string tls.base.key="$(cat ***REMOVED***/***REMOVED***.key)" \
--set-string tls.forward.cert="$(cat ***REMOVED***/fullchain.cer)" \
--set-string tls.forward.key="$(cat ***REMOVED***/***REMOVED***.key)" \
--set tls.base.enabled=true \
--set tls.forward.enabled=true
```

6. 如果不需要更新证书，则直接执行：

```shell
helm upgrade --install crater ./charts/crater \
--namespace crater \
--create-namespace 
```

### 3.3 证书过期

ACT �? HTTPS 证书�? 3 个月更新�?次，证书更新方法�? Web Frontend 项目�?

## 4. 项目结构（过时）

> [Wiki 代码架构](https://gitlab.***REMOVED***/raids/resource-scheduling/crater/web-backend/-/wikis/%E4%BB%A3%E7%A0%81%E6%9E%B6%E6%9E%84)

主要代码逻辑在pkg文件夹下�?

* apis：crd的定义�??
* control：提供接口，负责在集群创建具体的对象，例如pod、aijob等�??
* **controller**：负责同步各crd的状�?
  * job_controller.go：控制job的状态变�?
  * pod.go：监听pod的状态变化�??
  * quota_controller
  * quota_info.go
* db：数据库相关存储
  * internal：db的底层操�?
  * task
  * quota
  * user
* generated：k8s生成的clientset
* models：数据模�?
  * aitask
  * quota
  * user
* **server**：服务端接口和响�?
  * handlers：具体响应，操作数据�?
  * payload：外部请求接口的定义
* **taskqueue**：维护用户的任务队列，检查什么时候应该调度作�?
* profiler：负责对任务进行profile

## 5. �?发注意事�?

1. 在编�? Gin API 时，Gin 会先�? JWT 验证中间件先�?�? JWT Token 中包含的用户信息，并存入 Context 中�?�要获取该请求对应的用户信息，可通过 `util.GetToken` 获取
2. 数据�? CURD 代码通过 Gorm Gen 生成，见 `cmd/gorm_gen` 内文�?
