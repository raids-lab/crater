---
title: "LXCFS 配置"
description: "配置 LXCFS 以支持资源视图隔离"
---

## 背景介绍

在过去的两年间，我们基于 Kubernetes 搭建了云原生的机器学习平台，逐步取代了原有的基于 Slurm 的集群调度工具。

为了尽量保持原有方式和基于容器的方式的兼容性，我们进行了一些尝试，但依然存在一些问题，比如容器内的资源可见性——

### 用户故事

小明是深度学习方向的研究生，也是云原生机器学习平台的一名用户。

这一天，他在平台上申请了一个 Jupyter 调试作业，在启动作业时，小明需要选择 CPU、Memory、GPU 的数量和型号，之后平台会将这些限制，渲染成 Kubernetes Pod Resources 的 Requests 和 Limits：

```yaml
resources:
	limits:
		cpu: "16"
		memory: 32Gi
		nvidia.com/a100: "1"
	requests:
		cpu: "16"
		memory: 32Gi
		nvidia.com/a100: "1"
```

作业启动后，小明在作业中运行`nvidia-smi` 命令，正常显示一张显卡。但当运行 `lscpu`、`top` 等命令时，看到的 CPU 核心、内存容量均远超过他所申请的 16C 32G（实际上是宿主机的资源数量）：

```bash
$ top
MiB Mem : 385582.0 total, 258997.6 free,  24158.2 used, 105203.0 buff/cache
```

小明本身对于容器技术并不熟悉，他以为机器学习平台分配的是类似虚拟机的资源，因此对于这样的表现有些困惑。

### 解决方案

上述问题不仅影响着用户体验，还可能对程序性能产生影响。对于 Java、Go 等程序，以 Go 程序为例，Go 程序启动时，会根据 CPU 数量设置 `GOMAXPROCS` 变量，说明可执行的最大线程[^2]。但在容器环境中，这个变量的值依然是宿主机的值，如果在少量 CPU 上启动了过多的线程，可能会造成频繁的线程切换开销，从而拖慢程序运行速度。

对此我们有两种解决方案：

1. **用户感知**：在 Slurm 中，会在作业注入以下环境变量，以说明作业实际申请的资源[^1]：

| 变量名                | 解释                      |
| --------------------- | ------------------------- |
| `SLURM_CPUS_ON_NODE`  | 分配的节点上的 CPU 颗数   |
| `SLURM_CPUS_PER_TASK` | 每个任务的 CPU 颗数       |
| `SLURM_GPUS_PER_NODE` | 需要的每个节点的 GPU 颗数 |
| `SLURM_MEM_PER_NODE`  | 需要的每个节点的 Mem 数量 |

类似的，我们也可以在启动 Pod 时注入相关的环境变量，和用户做好约定。

2. **用户无感知**（但还是有一定的局限性）：比如下文介绍的 LXCFS。

## LXCFS 介绍

LXCFS（Linux Container Filesystem）是一个基于用户空间的文件系统实现，基于 FUSE 文件系统，旨在解决 Linux 容器环境中 proc 文件系统（procfs）的固有局限性。

具体来说，它提供了两个主要内容：

1. 一组文件，可以绑定挂载到其 `/proc` 原始文件上，以提供 CGroup 感知值。
2. 容器感知的类似 cgroupfs 的树。

有了 LXCFS，当我们在容器中查询 `/proc/cpuinfo` 等信息时，查询的内容将被 LXCFS 使用 FUSE 方式注入以 “劫持”，LXCFS 会结合容器的 `cgroup` 信息，给出正确的结果。

## 现有 LXCFS for Kubernetes 方案的不足

> [!quote]
>
> - [技术分享之 实现 Pod 资源视图隔离 \| 董江博客 \| DongJiang Blog](https://kubeservice.cn/2021/04/27/k8s-lxcfs-overview/)

上述思路并不困难，目前也已经有不少 LXCFS for Kubernetes 的开源方案：

| 项目                                                                                        | 备注                                      |
| ------------------------------------------------------------------------------------------- | ----------------------------------------- |
| [denverdino/lxcfs-admission-webhook](https://github.com/denverdino/lxcfs-admission-webhook) | Star 数最多的，但功能不全，很久不维护了   |
| [kubeservice-stack/lxcfs-webhook](https://github.com/kubeservice-stack/lxcfs-webhook)       | 更新较快，但存在一些错误（后续打算提 PR） |
| [cndoit18/lxcfs-on-kubernetes](https://github.com/cndoit18/lxcfs-on-kubernetes)             | 维护较少                                  |

（TODO：给上述方案的原理也做一个简单介绍，这里先跳过，读者可以阅读相关博客）

然而，在深入了解并使用上述方案后，我发现这些方案或多或少存在着一些问题：

### 1. 节点重启后 Pod 资源信息异常

> [!quote]
>
> - [TIPS 之 Kubernetes LXCFS 故障恢复后，对现有 Pod 进行 remount 操作 \| 董江博客 \| DongJiang Blog](https://kubeservice.cn/2022/04/13/k8s-lxcfs-remount/)
> - [lxcfs 的 Kubernetes 实践 - 廖思睿的个人博客](https://blog.liaosirui.com/%E7%B3%BB%E7%BB%9F%E8%BF%90%E7%BB%B4/E.%E5%AE%B9%E5%99%A8%E4%B8%8E%E5%AE%B9%E5%99%A8%E7%BC%96%E6%8E%92/%E5%AE%B9%E5%99%A8%E6%8A%80%E6%9C%AF%E7%9A%84%E5%9F%BA%E7%9F%B3/lxcfs/lxcfs%E7%9A%84%E4%BD%BF%E7%94%A8/lxcfs%E7%9A%84Kubernetes%E5%AE%9E%E8%B7%B5.html)
> - [lxcfs-admission-webhook/lxcfs-image/start.sh at 23298354a1d3cd6eaeb76417aa3fea75df5cdd63 · ThinkBlue1991/lxcfs-admission-webhook · GitHub](https://github.com/ThinkBlue1991/lxcfs-admission-webhook/blob/23298354a1d3cd6eaeb76417aa3fea75df5cdd63/lxcfs-image/start.sh)

[kubeservice-lxcfs-webhook 1.4.0 · kubeservice/kubservice-charts](https://artifacthub.io/packages/helm/kubservice-charts/kubeservice-lxcfs-webhook?modal=values)

[lxcfs-on-kubernetes/charts/lxcfs-on-kubernetes/values.yaml at master · cndoit18/lxcfs-on-kubernetes · GitHub](https://github.com/cndoit18/lxcfs-on-kubernetes/blob/master/charts/lxcfs-on-kubernetes/values.yaml)

[Container Lifecycle Hooks \| Kubernetes](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#:~:text=This%20hook%20is%20called%20immediately%20before%20a%20container,liveness%2Fstartup%20probe%20failure%2C%20preemption%2C%20resource%20contention%20and%20others.)

当 LXCFS 正常运行时，Pod 可以查看被重写的 Uptime 等信息：

```bash
$ top
top - 07:47:52 up 9 min,  0 users,  load average: 0.00, 0.00, 0.00
```

但如果节点发生了重启，那么默认情况下，LXCFS 不会继续重写 Pod 内的相关信息：

```bash
$ top
top: failed /proc/stat open: Transport endpoint is not connected
```

为了解决这个问题，社区也提出了对应方案[^3]，我们可以借助 Kubernetes 的 Container Lifecycle Hooks 机制，在节点重启后 LXCFS 启动时，重新给当前的每个 Pod 添加挂载。

上述方案需要在节点上安装 LXCFS，并通过 Systemd 配置 LXCFS 自启动。这非常不云原生。为此，我们可以在 LXCFS 容器中挂载 containerd 相关的 socket，从而不依赖于宿主机的能力。

### 2. LXCFS 容器退出后无法重新创建（死锁）

在我进行调试的时候我发现，如果 LXCFS DaemonSet 发生了退出，那么在节点重启之前，重新安装 LXCFS Daemonset 必然失败：

```bash
$ kubectl get pods
NAME         READY  STATUS                RESTARTS  AGE
lxcfs-77c87  0/1    CreateContainerError  0         18m
```

这是因为 LXCFS 创建时，需要挂载宿主机上的 `/var/lib/lxcfs` 目录，但这个目录是 LXCFS 创建后，才会成功挂载的，产生了死锁。

为此，我们可以在 LXCFS 退出时，使用 Kubernetes 的 Container Lifecycle Hooks 机制，在退出之前进行相关挂载点的删除。

```yaml
preStop:
  exec:
	command:
	  - bash
	  - -c
	  - nsenter -m/proc/1/ns/mnt fusermount -u /var/lib/lxc/lxcfs 2> /dev/null || true
```

上述方法也不是万无一失的，如果还是没有清理，只能重启节点。

为了解决这个问题，我们可以创建另一个 volumes 声明，指向 lxcfs 的父目录，并在 init Container 中进行残留挂载的卸载，这样就万无一失了。

### 3. 支持 LXCFS 版本较为陈旧

目前 LXCFS 已经更新了 6.0 版本，但社区的主流版本依然是 4.0.

不过高版本的 LXCFS 对 glibc 等也有着更高的要求，需要结合集群实际情况选择使用的版本。

### 4. 依赖于宿主机的 `libfuse.so`

> [!quote] [LXCFS 在 Docker 和 Kubernetes 下的实践](https://zhuanlan.zhihu.com/p/348625551)

在 Kubernetes 部署 DaemonSet 时，可能会报错：

```text
/usr/local/bin/lxcfs: error while loading shared libraries: libfuse.so.2: cannot open shared object file: No such file or directory
```

为了解决上述问题，第一种方法，我们可以在节点上安装 `libfuse2` （CentOS 则不同），通过 Ansible 批量保证节点上的 `libfuse2` 已安装：

```yaml
- name: Ensure libfuse2 is installed
  hosts: all
  become: yes
  gather_facts: yes

  tasks:
    - name: Check if libfuse2 is installed
      apt:
        name: libfuse2
        state: present
      register: libfuse2_installed
      changed_when: libfuse2_installed.changed
```

```bash
$ ansible-playbook -i hosts lxcfs.yaml

PLAY [Ensure libfuse2 is installed]
TASK [Gathering Facts]
ok: [192.168.5.75]
ok: [192.168.5.1]

TASK [Check if libfuse2 is installed]
ok: [192.168.5.1]
changed: [192.168.5.75]

PLAY RECAP
192.168.5.1                : ok=2    changed=0    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
192.168.5.75               : ok=2    changed=1    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
```

另一种方式，我们则可以修改 Dockerfile 的构建方式以及启动脚本，让最终 LXCFS 容器运行的时候，包含所需的动态链接库。

## 安装 LXCFS Webhook

[raids-lab/lxcfs-webhook](https://github.com/raids-lab/lxcfs-webhook)

针对上述问题，我们整合并优化了多个方案，提供了 Yet Another 的 LXCFS Webhook。

### 1. 依赖

首先，安装 Cert Manager（如果还没有安装过）：

```bash
helm repo add jetstack https://charts.jetstack.io --force-update
```

要安装 cert-manager Helm 图表，请使用 Helm install 命令，如下所述。

```bash
helm install \
cert-manager jetstack/cert-manager \
--namespace cert-manager \
--create-namespace \
--version v1.17.2 \
--set crds.enabled=true
```

### 2. 通过 Helm 安装

[raids-lab/lxcfs-webhook](https://github.com/raids-lab/lxcfs-webhook)

克隆上述代码后，通过 Helm 安装：

```bash
helm upgrade --install lxcfs-webhook ./dist/chart -n lxcfs
```

将包含 LXCFS DaemonSet、Webhook，并解决节点重启、Daemon 重启等问题。

### 3. 指定作用域

之后，可以给命名空间加上标签：

```bash
kubectl label namespace <namespace-name> lxcfs-admission-webhook:enabled
```

对应命名空间内的 Pod 在创建时，将自动进行 LXCFS 的挂载。

## LXCFS Webhook 设计

### 1. LXCFS DaemonSet 镜像构建

为了构建不依赖于宿主机 `libfuse.go` 的镜像，我们首先检查 `ldconfig -p | grep libfuse.so.2` 对应的位置：

```bash
$ ldconfig -p | grep libfuse.so.2
        libfuse.so.2 (libc6,x86-64) => /lib/x86_64-linux-gnu/libfuse.so.2

$ ldconfig -p | grep libulockmgr.so
        libulockmgr.so.1 (libc6,x86-64) => /lib/x86_64-linux-gnu/libulockmgr.so.1
        libulockmgr.so (libc6,x86-64) => /lib/x86_64-linux-gnu/libulockmgr.so

$ ls /lxcfs/build/
build.ninja            config.h       lxcfs    lxcfs.spec  meson-private
compile_commands.json  liblxcfs.so    lxcfs.1  meson-info  share
config                 liblxcfs.so.p  lxcfs.p  meson-logs  tests
```

之后针对 Ubuntu 操作系统，我们进行两阶段构建：

```dockerfile
# LXCFS Builder Image
# Builds LXCFS from source on Ubuntu 22.04

FROM crater-harbor.act.buaa.edu.cn/docker.io/ubuntu:22.04 AS build

# Environment configuration
ENV DEBIAN_FRONTEND=noninteractive \
    LXCFS_VERSION=v6.0.4

# Install build dependencies
RUN apt-get update && \
    apt-get --purge remove -y lxcfs && \
    apt-get install -y --no-install-recommends \
    build-essential \
    cmake \
    fuse3 \
    git \
    help2man \
    libcurl4-openssl-dev \
    libfuse-dev \
    libtool \
    libxml2-dev \
    m4 \
    meson \
    mime-support \
    pkg-config \
    python3-pip \
    systemd \
    wget \
    autotools-dev \
    automake && \
    rm -rf /var/lib/apt/lists/*

# Install Python dependencies
RUN pip3 install --no-cache-dir -U jinja2 \
    -i https://mirrors.aliyun.com/pypi/simple/

# Download and build LXCFS from source
RUN wget https://linuxcontainers.org/downloads/lxcfs/lxcfs-${LXCFS_VERSION}.tar.gz && \
    mkdir /lxcfs && \
    tar xzvf lxcfs-${LXCFS_VERSION}.tar.gz -C /lxcfs --strip-components=1 && \
    cd /lxcfs && \
    make && \
    make install && \
    rm -f /lxcfs-${LXCFS_VERSION}.tar.gz

FROM crater-harbor.act.buaa.edu.cn/docker.io/ubuntu:22.04

STOPSIGNAL SIGINT

COPY --from=build /lxcfs/build/lxcfs /lxcfs/lxcfs
COPY --from=build /lxcfs/build/liblxcfs.so /lxcfs/liblxcfs.so
COPY --from=build /lib/x86_64-linux-gnu/libfuse.so.2.9.9 /lxcfs/libfuse.so.2.9.9
COPY --from=build /lib/x86_64-linux-gnu/libulockmgr.so.1.0.1 /lxcfs/libulockmgr.so.1.0.1

CMD ["/bin/false"]
```

这里我们将相关的动态链接库先移动至 `/lxcfs` 暂存目录下，否则会被 HostPath 覆盖，之后编写启动脚本，在脚本中将相关的动态链接库重新移回：

```bash
#!/bin/bash

set -e  # Exit immediately if a command exits with a non-zero status.

# Cleanup
nsenter -m/proc/1/ns/mnt fusermount -u /var/lib/lxc/lxcfs 2> /dev/null || true
nsenter -m/proc/1/ns/mnt [ -L /etc/mtab ] || \
		sed -i "/^lxcfs \/var\/lib\/lxc\/lxcfs fuse.lxcfs/d" /etc/mtab

# Prepare
mkdir -p /usr/local/lib/lxcfs /var/lib/lxc/lxcfs

# Update lxcfs
cp -f /lxcfs/lxcfs /usr/local/bin/lxcfs
cp -f /lxcfs/liblxcfs.so /lib/x86_64-linux-gnu/liblxcfs.so
cp -f /lxcfs/libfuse.so.2.9.9 /lib/x86_64-linux-gnu/libfuse.so.2.9.9
cp -f /lxcfs/libulockmgr.so.1.0.1 /lib/x86_64-linux-gnu/libulockmgr.so.1.0.1

# Remove old links
rm -f /lib/x86_64-linux-gnu/libfuse.so.2 /lib/x86_64-linux-gnu/libulockmgr.so.1 /lib/x86_64-linux-gnu/libulockmgr.so

# Create new links
ln -s /lib/x86_64-linux-gnu/libfuse.so.2.9.9 /lib/x86_64-linux-gnu/libfuse.so.2
ln -s /lib/x86_64-linux-gnu/libulockmgr.so.1.0.1 /lib/x86_64-linux-gnu/libulockmgr.so.1
ln -s /lib/x86_64-linux-gnu/libulockmgr.so.1.0.1 /lib/x86_64-linux-gnu/libulockmgr.so

# Update library cache
nsenter -m/proc/1/ns/mnt ldconfig

# Mount
exec nsenter -m/proc/1/ns/mnt /usr/local/bin/lxcfs /var/lib/lxc/lxcfs/ --enable-cfs -l -o nonempty
```

### 2. LXCFS Webhook 功能设计

Webhook 的功能比较简单，我们基于 Kubebuilder 的框架，可以快速搭建一个 Webhook。我们实现了 Mutation 和 Validate 的 Webhook，在 Validate 中，我们主要检查 Pod 与 LXCFS 规则忽略相关的 Annotation 是否具有正确的值。

而在 Mutation 中，我们首先检查 Pod 是否需要进行 Mutate，如果是，我们就给 Pod 打上已经 Mutate 的标签，并给 Pod 添加 LXCFS 的 Volumes 和 VolumeMounts。

```go
// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Pod.
func (d *PodLxcfsDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)

	if !ok {
		return fmt.Errorf("expected an Pod object but got %T", obj)
	}
	podlog.Info("Defaulting for Pod", "name", pod.GetName(), "namespace", pod.GetNamespace())

	// Check if the Pod should be mutated
	if !mutationRequired(pod) {
		podlog.Info("Skipping mutation for Pod", "name", pod.GetName(), "namespace", pod.GetNamespace())
		return nil
	}

	// If the Pod is not mutated, we need to add the annotation
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations[AdmissionWebhookAnnotationStatusKey] = StatusValueMutated

	// Add LXCFS VolumeMounts to all containers
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		if container.VolumeMounts == nil {
			container.VolumeMounts = make([]corev1.VolumeMount, 0)
		}
		container.VolumeMounts = append(container.VolumeMounts, VolumeMountsTemplate...)
	}

	// Add LXCFS VolumeMounts to pod
	if pod.Spec.Volumes == nil {
		pod.Spec.Volumes = make([]corev1.Volume, 0)
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, VolumesTemplate...)

	return nil
}
```

## 验证

申请 1c 2G，在容器内查看 CPU 和 Memory 的方式：

```bash
$ cat /proc/meminfo | grep MemTotal:
MemTotal:        2097152 kB

$ cat /proc/cpuinfo | grep processor
processor       : 0

$ cat /proc/cpuinfo | grep processor | wc -l
1
```

## 总结

通过以上方案，我们可以让机器学习平台中的调试作业更像一个虚拟机，减少用户的心智负担。但 LXCFS 的方案还是有一些局限性，比如比较常用的 `nproc` 命令，依然是显示宿主机的信息[^4]。

机器学习平台的用户通常对容器技术了解不深，如何让他们知道这些不一致的原因和解决方案，依然是一个困扰着我们的问题。

[^1]: [Slurm 作业调度系统使用指南 | 中国科大超级计算中心](https://scc.ustc.edu.cn/hmli/doc/userguide/slurm-userguide.pdf)
[^2]: [容器资源可见性问题与 GOMAXPROCS 配置 · Issue #216 · islishude/blog](https://github.com/islishude/blog/issues/216)
[^3]: [lxcfs-admission-webhook/lxcfs-image/start.sh at 23298354a1d3cd6eaeb76417aa3fea75df5cdd63 · ThinkBlue1991/lxcfs-admission-webhook · GitHub](https://github.com/ThinkBlue1991/lxcfs-admission-webhook/blob/23298354a1d3cd6eaeb76417aa3fea75df5cdd63/lxcfs-image/start.sh)
[^4]: [lscpu shows all cpu cores of physical server](https://github.com/lxc/lxcfs/issues/181#issuecomment-290458686)
