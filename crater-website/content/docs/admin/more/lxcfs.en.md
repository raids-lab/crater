---
title: "LXCFS Configuration"
description: "Configure LXCFS to support resource view isolation"
---

## Background Introduction

Over the past two years, we have built a cloud-native machine learning platform based on Kubernetes, gradually replacing the original cluster scheduling tool based on Slurm.

In order to maintain as much compatibility as possible between the original approach and the container-based approach, we have made some attempts, but there are still some issues, such as resource visibility within containers—

### User Story

Xiaoming is a graduate student in the field of deep learning and a user of the cloud-native machine learning platform.

One day, he applied for a Jupyter debug job on the platform. When starting the job, Xiaoming needed to select the number and type of CPU, Memory, and GPU. After that, the platform would render these limits into Kubernetes Pod Resources Requests and Limits:

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

After the job started, Xiaoming ran the `nvidia-smi` command in the job, and it displayed one GPU normally. However, when running commands like `lscpu` and `top`, he saw CPU cores and memory capacity far exceeding the 16C 32G he applied for (which was actually the host machine's resource count):

```bash
$ top
MiB Mem : 385582.0 total, 258997.6 free,  24158.2 used, 105203.0 buff/cache
```

Xiaoming is not familiar with container technology. He thought the machine learning platform allocated resources similar to virtual machines, so he was a bit confused by this behavior.

### Solution

The above issue not only affects user experience, but may also impact program performance. For programs such as Java and Go, for example, a Go program sets the `GOMAXPROCS` variable when starting, indicating the maximum number of threads that can be executed[^2]. However, in the container environment, the value of this variable is still the host machine's value. If too many threads are started on a few CPUs, it may cause frequent thread switching overhead, thereby slowing down the program's runtime speed.

We have two solutions for this:

1. **User-aware**: In Slurm, the following environment variables are injected into the job to indicate the actual resources requested by the job[^1]:

| Variable Name                | Explanation                      |
| --------------------------- | -------------------------------- |
| `SLURM_CPUS_ON_NODE`        | Number of CPUs on the allocated node |
| `SLURM_CPUS_PER_TASK`       | Number of CPUs per task          |
| `SLURM_GPUS_PER_NODE`       | Number of GPUs required per node |
| `SLURM_MEM_PER_NODE`        | Amount of Mem required per node  |

Similarly, we can inject related environment variables when starting the Pod and make an agreement with the user.

2. **User-unaware** (but still has some limitations): For example, the LXCFS introduced below.

## Introduction to LXCFS

LXCFS (Linux Container Filesystem) is a user-space filesystem implementation based on the FUSE filesystem, aiming to solve the inherent limitations of the proc filesystem (procfs) in Linux container environments.

Specifically, it provides two main features:

1. A set of files that can be bind-mounted to their original `/proc` files to provide cgroup-aware values.
2. A container-aware tree similar to cgroupfs.

With LXCFS, when we query information such as `/proc/cpuinfo` in the container, the content will be "hijacked" by LXCFS using the FUSE method. LXCFS will combine the container's `cgroup` information to provide the correct result.

## Limitations of Existing LXCFS for Kubernetes Solutions

> [!quote]
>
> - [Technical Sharing: Implementing Pod Resource View Isolation \| DongJiang Blog](https://kubeservice.cn/2021/04/27/k8s-lxcfs-overview/)

The above idea is not difficult, and there are already many open-source solutions for LXCFS for Kubernetes:

| Project                                                                                      | Notes                                      |
|---------------------------------------------------------------------------------------------|-------------------------------------------|
| [denverdino/lxcfs-admission-webhook](https://github.com/denverdino/lxcfs-admission-webhook) | Most starred, but incomplete, and not maintained for a long time |
| [kubeservice-stack/lxcfs-webhook](https://github.com/kubeservice-stack/lxcfs-webhook)       | Updated frequently, but has some errors (plans to submit a PR later) |
| [cndoit18/lxcfs-on-kubernetes](https://github.com/cndoit18/lxcfs-on-kubernetes)             | Less maintained                            |

(TODO: Provide a simple introduction to the principles of the above solutions; skip for now, readers can refer to the relevant blog)

However, after in-depth study and use of the above solutions, I found that these solutions have some issues to varying degrees:

### 1. Pod Resource Information Abnormal After Node Restart

> [!quote]
>
> - [TIPS: After Kubernetes LXCFS failure recovery, remount operation for existing Pod \| DongJiang Blog](https://kubeservice.cn/2022/04/13/k8s-lxcfs-remount/)
> - [LXCFS Practice in Kubernetes - Personal Blog of Liao Sirui](https://blog.liaosirui.com/%E7%B3%BB%E7%BB%9F%E8%BF%90%E7%BB%B4/E.%E5%AE%B9%E5%99%A8%E4%B8%8E%E5%AE%B9%E5%99%A8%E7%BC%96%E6%8E%92/%E5%AE%B9%E5%99%A8%E6%8A%80%E6%9C%AF%E7%9A%84%E5%9F%BA%E7%9F%B3/lxcfs/lxcfs%E7%9A%84%E4%BD%BF%E7%94%A8/lxcfs%E7%9A%84Kubernetes%E5%AE%9E%E8%B7%B5.html)
> - [lxcfs-admission-webhook/lxcfs-image/start.sh at 23298354a1d3cd6eaeb76417aa3fea75df5cdd63 · ThinkBlue1991/lxcfs-admission-webhook · GitHub](https://github.com/ThinkBlue1991/lxcfs-admission-webhook/blob/23298354a1d3cd6eaeb76417aa3fea75df5cdd63/lxcfs-image/start.sh)

[kubeservice-lxcfs-webhook 1.4.0 · kubeservice/kubservice-charts](https://artifacthub.io/packages/helm/kubservice-charts/kubeservice-lxcfs-webhook?modal=values)

[lxcfs-on-kubernetes/charts/lxcfs-on-kubernetes/values.yaml at master · cndoit18/lxcfs-on-kubernetes · GitHub](https://github.com/cndoit18/lxcfs-on-kubernetes/blob/master/charts/lxcfs-on-kubernetes/values.yaml)

[Container Lifecycle Hooks \| Kubernetes](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#:~:text=This%20hook%20is%20called%20immediately%20before%20a%20container,liveness%2Fstartup%20probe%20failure%2C%20preemption%2C%20resource%20contention%20and%20others.)

When LXCFS is running normally, the Pod can view the rewritten Uptime and other information:

```bash
$ top
top - 07:47:52 up 9 min,  0 users,  load average: 0.00, 0.00, 0.00
```

However, if the node restarts, by default, LXCFS does not continue to rewrite the relevant information inside the Pod:

```bash
$ top
top: failed /proc/stat open: Transport endpoint is not connected
```

To solve this problem, the community has proposed corresponding solutions[^3]. We can leverage the Kubernetes Container Lifecycle Hooks mechanism to remount the current Pod after the node restarts when LXCFS starts. 

The above solution requires installing LXCFS on the node and configuring LXCFS to start automatically via Systemd. This is very uncloud-native. Therefore, we can mount the containerd-related socket in the LXCFS container, thus not relying on the host's capabilities.

### 2. LXCFS Container Exit and Re-creation Failure (Deadlock)

During my debugging, I found that if the LXCFS DaemonSet exits, then before the node restarts, re-installing the LXCFS DaemonSet will definitely fail:

```bash
$ kubectl get pods
NAME         READY  STATUS                RESTARTS  AGE
lxcfs-77c87  0/1    CreateContainerError  0         18m
```

This is because when LXCFS is created, it needs to mount the host's `/var/lib/lxcfs` directory, but this directory is only successfully mounted after LXCFS is created, causing a deadlock.

To address this, we can use Kubernetes's Container Lifecycle Hooks mechanism to delete the relevant mount points before LXCFS exits.

```yaml
preStop:
  exec:
	command:
	  - bash
	  - -c
	  - nsenter -m/proc/1/ns/mnt fusermount -u /var/lib/lxc/lxcfs 2> /dev/null || true
```

The above method is not foolproof. If the cleanup still fails, a node restart is required.

To solve this, we can create another volume declaration pointing to the parent directory of lxcfs and perform the unmounting of residual mounts in the init container, ensuring this is foolproof.

### 3. LXCFS Version is Relatively Outdated

Currently, LXCFS has been updated to version 6.0, but the mainstream version in the community is still 4.0.

However, higher versions of LXCFS require higher versions of glibc and other libraries, and the version to use should be selected based on the actual situation of the cluster.

### 4. Depends on the Host's `libfuse.so`

> [!quote] [LXCFS Practice in Docker and Kubernetes](https://zhuanlan.zhihu.com/p/348625551)

When deploying a DaemonSet in Kubernetes, there may be an error:

```text
/usr/local/bin/lxcfs: error while loading shared libraries: libfuse.so.2: cannot open shared object file: No such file or directory
```

To resolve the above issue, the first method is to install `libfuse2` on the node (different for CentOS), and ensure that `libfuse2` is installed on all nodes using Ansible:

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

Another method is to modify the Dockerfile build method and startup script so that the final LXCFS container includes the required dynamic libraries when it runs.

## Install LXCFS Webhook

[raids-lab/lxcfs-webhook](https://github.com/raids-lab/lxcfs-webhook)

To address the above issues, we have integrated and optimized multiple solutions to provide Yet Another LXCFS Webhook.

### 1. Dependencies

First, install Cert Manager (if not already installed):

```bash
helm repo add jetstack https://charts.jetstack.io --force-update
```

To install the cert-manager Helm chart, use the Helm install command as follows.

```bash
helm install \
cert-manager jetstack/cert-manager \
--namespace cert-manager \
--create-namespace \
--version v1.17.2 \
--set crds.enabled=true
```

### 2. Install via Helm

[raids-lab/lxcfs-webhook](https://github.com/raids-lab/lxcfs-webhook)

After cloning the code, install via Helm:

```bash
helm upgrade --install lxcfs-webhook ./dist/chart -n lxcfs
```

This includes the LXCFS DaemonSet, Webhook, and solves issues such as node restarts and Daemon restarts.

### 3. Specify Scope

Then, you can add labels to the namespace:

```bash
kubectl label namespace <namespace-name> lxcfs-admission-webhook:enabled
```

Pods within the corresponding namespace will automatically mount LXCFS when created.

## LXCFS Webhook Design

### 1. LXCFS DaemonSet Image Building

To build an image not dependent on the host's `libfuse.go`, we first check the location of `ldconfig -p | grep libfuse.so.2`:

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

Then, for the Ubuntu operating system, we perform a two-stage build:

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

Here, we move the related dynamic libraries to the `/lxcfs` temporary directory first, otherwise they may be overwritten by HostPath. Then, we write a startup script, in which we move the related dynamic libraries back:

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

### 2. LXCFS Webhook Function Design

The functionality of the Webhook is simple. Based on the Kubebuilder framework, we can quickly build a Webhook. We implemented Mutation and Validate Webhooks. In Validate, we mainly check if the Pod and LXCFS rules ignore-related Annotations have the correct values.

In Mutation, we first check if the Pod needs to be mutated. If yes, we label the Pod as mutated and add LXCFS Volumes and VolumeMounts to the Pod.

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

## Verification

Apply for 1c 2G, and check CPU and Memory in the container:

```bash
$ cat /proc/meminfo | grep MemTotal:
MemTotal:        2097152 kB

$ cat /proc/cpuinfo | grep processor
processor       : 0

$ cat /proc/cpuinfo | grep processor | wc -l
1
```

## Summary

Through the above solutions, we can make the debug jobs in the machine learning platform more like virtual machines, reducing the cognitive burden on users. However, the LXCFS solution still has some limitations, such as the commonly used `nproc` command still displaying the host machine's information[^4].

Users of the machine learning platform usually have limited knowledge of container technology. How to make them understand the reasons for these inconsistencies and the solutions remains a problem that troubles us.

[^1]: [Slurm Job Scheduling System User Guide | Supercomputing Center of USTC](https://scc.ustc.edu.cn/hmli/doc/userguide/slurm-userguide.pdf)
[^2]: [Container Resource Visibility Issues and GOMAXPROCS Configuration · Issue #216 · islishude/blog](https://github.com/islishude/blog/issues/216)
[^3]: [lxcfs-admission-webhook/lxcfs-image/start.sh at 23298354a1d3cd6eaeb76417aa3fea75df5cdd63 · ThinkBlue1991/lxcfs-admission-webhook · GitHub](https://github.com/ThinkBlue1991/lxcfs-admission-webhook/blob/23298354a1d3cd6eaeb76417aa3fea75df5cdd63/lxcfs-image/start.sh)
[^4]: [lscpu shows all cpu cores of physical server](https://github.com/lxc/lxcfs/issues/181#issuecomment-290458686)