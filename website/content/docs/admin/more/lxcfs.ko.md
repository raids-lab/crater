---
title: "LXCFS 구성"
description: "리소스 시각 분리 지원을 위해 LXCFS 구성"
---

## 배경 소개

지난 두 년간 우리는 Kubernetes 기반의 클라우드 네이티브 머신러닝 플랫폼을 구축하여, 기존의 Slurm 기반 클러스터 스케줄링 도구를 점차 대체해 나갔습니다.

기존 방식과 컨테이너 기반 방식의 호환성을 최대한 유지하기 위해 몇 가지 시도를 해보았지만, 여전히 몇 가지 문제가 존재했습니다. 예를 들어, 컨테이너 내의 리소스 가시성 문제 등이 있습니다.

### 사용자 이야기

소밍은 머신러닝 전공 석사 과정 학생이며, 클라우드 네이티브 머신러닝 플랫폼의 사용자입니다.

이 날, 소밍은 플랫폼에서 Jupyter 디버깅 작업을 요청했습니다. 작업 시작 시, 소밍은 CPU, 메모리, GPU의 수량과 모델을 선택해야 했습니다. 이후 플랫폼은 이러한 제한을 Kubernetes Pod Resources의 Requests와 Limits로 렌더링합니다:

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

작업이 시작된 후, 소밍은 `nvidia-smi` 명령어를 실행하여 정상적으로 하나의 그래픽카드를 표시합니다. 하지만 `lscpu`, `top` 등의 명령어를 실행하면 CPU 코어 수와 메모리 용량이 16C 32G보다 훨씬 더 큰 값을 보여줍니다(실제로는 호스트 머신의 리소스 수량입니다):

```bash
$ top
MiB Mem : 385582.0 total, 258997.6 free,  24158.2 used, 105203.0 buff/cache
```

소밍은 컨테이너 기술에 익숙하지 않으며, 머신러닝 플랫폼이 가상 머신과 유사한 리소스를 할당한다고 생각하고, 이러한 결과에 대해 혼란을 느낍니다.

### 해결책

이러한 문제는 사용자 경험에 영향을 주며, 프로그램 성능에도 영향을 줄 수 있습니다. 예를 들어, Java, Go와 같은 프로그램들 중 Go 프로그램의 경우, 시작 시 CPU 수에 따라 `GOMAXPROCS` 변수를 설정하여 실행 가능한 최대 스레드를 지정합니다[^2]. 그러나 컨테이너 환경에서는 이 변수의 값은 여전히 호스트 머신의 값입니다. 소수의 CPU에서 너무 많은 스레드를 시작하면 스레드 전환의 빈도가 증가하여 프로그램 실행 속도가 느려질 수 있습니다.

이에 대해 우리는 두 가지 해결책을 제시합니다:

1. **사용자 인식**: Slurm에서는 작업이 주입될 때 다음 환경 변수를 사용하여 실제 요청된 리소스를 설명합니다[^1]:

| 변수명                | 설명                      |
| --------------------- | ------------------------- |
| `SLURM_CPUS_ON_NODE`  | 할당된 노드의 CPU 수      |
| `SLURM_CPUS_PER_TASK` | 각 작업의 CPU 수          |
| `SLURM_GPUS_PER_NODE` | 각 노드의 필요한 GPU 수   |
| `SLURM_MEM_PER_NODE`  | 각 노드에 필요한 메모리 수 |

동일하게, 우리는 Pod를 시작할 때 관련 환경 변수를 주입하고, 사용자와 협의하여 합의할 수 있습니다.

2. **사용자 인식 없음**(하지만 여전히 일정한 한계가 존재): 예를 들어 아래에서 설명하는 LXCFS.

## LXCFS 소개

LXCFS(Linux Container Filesystem)는 사용자 공간에서 구현된 파일 시스템이며, FUSE 파일 시스템 기반으로 설계되어 Linux 컨테이너 환경에서 proc 파일 시스템(procfs)의 고유한 한계를 해결하려는 목적으로 사용됩니다.

구체적으로, 다음과 같은 두 가지 주요 내용을 제공합니다:

1. 원래 `/proc` 파일에 바인딩 마운트할 수 있는 파일 집합을 통해 CGroup 인식 값을 제공합니다.
2. 컨테이너 인식 가능한 cgroupfs와 유사한 트리 구조.

LXCFS를 사용하면 컨테이너에서 `/proc/cpuinfo` 등의 정보를 조회할 때, LXCFS는 FUSE를 통해 이를 "조작"하여 컨테이너의 `cgroup` 정보를 기반으로 올바른 결과를 제공합니다.

## 현재 LXCFS for Kubernetes 솔루션의 한계

> [!quote]
>
> - [기술 공유: Pod 자원 시각 분리 구현 \| DongJiang Blog](https://kubeservice.cn/2021/04/27/k8s-lxcfs-overview/)

위의 아이디어는 어렵지 않으며, 현재 LXCFS for Kubernetes의 오픈소스 솔루션도 여러 가지가 있습니다:

| 프로젝트                                                                                        | 설명                                      |
| ------------------------------------------------------------------------------------------- | ----------------------------------------- |
| [denverdino/lxcfs-admission-webhook](https://github.com/denverdino/lxcfs-admission-webhook) | 스타 수가 가장 많은데, 기능이 불완전하고 오랫동안 유지보수가 되지 않았습니다.   |
| [kubeservice-stack/lxcfs-webhook](https://github.com/kubeservice-stack/lxcfs-webhook)       | 업데이트가 빠르지만, 오류가 있으며(후에 PR 제안 계획이 있습니다) |
| [cndoit18/lxcfs-on-kubernetes](https://github.com/cndoit18/lxcfs-on-kubernetes)             | 유지보수가 적습니다.                                  |

(TODO: 위의 솔루션 원리에 대한 간단한 소개도 추가할 예정입니다. 지금은 건너뜁니다. 관련 블로그를 읽어보세요.)

그러나 위 솔루션을 깊이 있게 조사하고 사용한 후, 이들 솔루션은或多或少 문제가 있는 것으로 나타났습니다:

### 1. 노드 재시작 후 Pod 자원 정보 이상

> [!quote]
>
> - [TIPS: Kubernetes LXCFS 복구 후, 기존 Pod에 remount 작업 수행 \| DongJiang Blog](https://kubeservice.cn/2022/04/13/k8s-lxcfs-remount/)
> - [LXCFS의 Kubernetes 사용 \|廖思睿의 개인 블로그](https://blog.liaosirui.com/%E7%B3%BB%E7%BB%9F%E8%BF%90%E7%BB%B4/E.%E5%AE%B9%E5%99%A8%E4%B8%8E%E5%AE%B9%E5%99%A8%E7%BC%96%E6%8E%92/%E5%AE%B9%E5%99%A8%E6%8A%80%E6%9C%AF%E7%9A%84%E5%9F%BA%E7%9F%B3/lxcfs/lxcfs%E7%9A%84%E4%BD%BF%E7%94%A8/lxcfs%E7%9A%84Kubernetes%E5%AE%9E%E8%B7%B5.html)
> - [lxcfs-admission-webhook/lxcfs-image/start.sh at 23298354a1d3cd6eaeb76417aa3fea75df5cdd63 · ThinkBlue1991/lxcfs-admission-webhook · GitHub](https://github.com/ThinkBlue1991/lxcfs-admission-webhook/blob/23298354a1d3cd6eaeb76417aa3fea75df5cdd63/lxcfs-image/start.sh)

[kubeservice-lxcfs-webhook 1.4.0 · kubeservice/kubservice-charts](https://artifacthub.io/packages/helm/kubservice-charts/kubeservice-lxcfs-webhook?modal=values)

[lxcfs-on-kubernetes/charts/lxcfs-on-kubernetes/values.yaml at master · cndoit18/lxcfs-on-kubernetes · GitHub](https://github.com/cndoit18/lxcfs-on-kubernetes/blob/master/charts/lxcfs-on-kubernetes/values.yaml)

[컨테이너 라이프사이클 훅 \| Kubernetes](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#:~:text=This%20hook%20is%20called%20immediately%20before%20a%20container,liveness%2Fstartup%20probe%20failure%2C%20preemption%2C%20resource%20contention%20and%20others.)

LXCFS가 정상적으로 작동할 때, Pod는 재작성된 Uptime 등의 정보를 확인할 수 있습니다:

```bash
$ top
top - 07:47:52 up 9 min,  0 users,  load average: 0.00, 0.00, 0.00
```

하지만 노드가 재시작되었을 경우, 기본적으로 LXCFS는 Pod 내의 관련 정보를 계속 재작성하지 않습니다:

```bash
$ top
top: failed /proc/stat open: Transport endpoint is not connected
```

이 문제에 대응하기 위해 커뮤니티에서는 대응 방식[^3]을 제시했습니다. 우리는 Kubernetes의 Container Lifecycle Hooks 메커니즘을 활용하여, 노드 재시작 시 LXCFS가 시작될 때, 현재 각 Pod에 마운트를 다시 추가할 수 있습니다.

이 방식은 노드에 LXCFS를 설치하고 Systemd를 통해 LXCFS를 자동 시작하도록 구성해야 하므로, 매우 클라우드 네이티브적이지 않습니다. 따라서 우리는 LXCFS 컨테이너 내에 containerd 관련 소켓을 마운트하여, 호스트 기능에 의존하지 않도록 할 수 있습니다.

### 2. LXCFS 컨테이너 종료 후 재생성 불가능(데드락)

내가 디버깅을 진행하면서, LXCFS DaemonSet이 종료되면, 노드 재시작 전에 LXCFS Daemonset을 다시 설치하면 반드시 실패합니다:

```bash
$ kubectl get pods
NAME         READY  STATUS                RESTARTS  AGE
lxcfs-77c87  0/1    CreateContainerError  0         18m
```

이는 LXCFS가 생성될 때 호스트의 `/var/lib/lxcfs` 디렉터리를 마운트해야 하며, 이 디렉터리는 LXCFS가 생성된 후에야 성공적으로 마운트되기 때문에 데드락이 발생합니다.

이를 해결하기 위해, LXCFS가 종료될 때 Kubernetes의 Container Lifecycle Hooks 메커니즘을 사용하여, 종료 전에 관련 마운트 지점을 삭제할 수 있습니다.

```yaml
preStop:
  exec:
	command:
	  - bash
	  - -c
	  - nsenter -m/proc/1/ns/mnt fusermount -u /var/lib/lxc/lxcfs 2> /dev/null || true
```

이 방법도 완벽하지는 않으며, 여전히 청소되지 않은 경우, 노드 재시작이 필요합니다.

이 문제를 해결하기 위해, 우리는 LXCFS의 부모 디렉터리를 가리키는 또 다른 volumes 선언을 생성하고, init 컨테이너에서 잔여 마운트를 언마운트함으로써, 완전히 보장할 수 있습니다.

### 3. LXCFS 버전 지원이 상대적으로 오래된 경우

현재 LXCFS는 6.0 버전이 업데이트되었지만, 커뮤니티의 주요 버전은 여전히 4.0입니다.

하지만 고버전의 LXCFS는 glibc 등에 더 높은 요구사항을 가지므로, 클러스터의 실제 상황에 따라 사용할 버전을 선택해야 합니다.

### 4. 호스트의 `libfuse.so`에 의존

> [!quote] [Docker 및 Kubernetes에서의 LXCFS 실무](https://zhuanlan.zhihu.com/p/348625551)

Kubernetes에서 DaemonSet을 배포할 때 다음과 같은 오류가 발생할 수 있습니다:

```text
/usr/local/bin/lxcfs: error while loading shared libraries: libfuse.so.2: cannot open shared object file: No such file or directory
```

위 문제를 해결하기 위해, 첫 번째 방법은 노드에 `libfuse2`를 설치하는 것입니다 (CentOS의 경우는 다릅니다). Ansible을 통해 노드에 `libfuse2`가 설치되었는지 확인할 수 있습니다:

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

다른 방법으로는 Dockerfile의 빌드 방식과 시작 스크립트를 수정하여, 최종적으로 LXCFS 컨테이너가 실행될 때 필요한 동적 링크 라이브러리를 포함시킬 수 있습니다.

## LXCFS Webhook 설치

[raids-lab/lxcfs-webhook](https://github.com/raids-lab/lxcfs-webhook)

위 문제에 대응하여 여러 가지 솔루션을 통합 및 최적화하여, 또 다른 LXCFS Webhook을 제공합니다.

### 1. 의존성

먼저, Cert Manager을 설치하세요 (아직 설치하지 않았다면):

```bash
helm repo add jetstack https://charts.jetstack.io --force-update
```

cert-manager Helm 차트를 설치하려면, 아래와 같이 Helm install 명령어를 사용합니다.

```bash
helm install \
cert-manager jetstack/cert-manager \
--namespace cert-manager \
--create-namespace \
--version v1.17.2 \
--set crds.enabled=true
```

### 2. Helm을 통한 설치

[raids-lab/lxcfs-webhook](https://github.com/raids-lab/lxcfs-webhook)

위 코드를 복제한 후 Helm으로 설치합니다:

```bash
helm upgrade --install lxcfs-webhook ./dist/chart -n lxcfs
```

이렇게 하면 LXCFS DaemonSet과 Webhook을 포함하여, 노드 재시작, Daemon 재시작 등 문제를 해결할 수 있습니다.

### 3. 범위 지정

그 후, 네임스페이스에 라벨을 추가합니다:

```bash
kubectl label namespace <namespace-name> lxcfs-admission-webhook:enabled
```

해당 네임스페이스 내의 Pod이 생성될 때 자동으로 LXCFS 마운트가 수행됩니다.

## LXCFS Webhook 설계

### 1. LXCFS DaemonSet 이미지 빌드

호스트의 `libfuse.go`에 의존하지 않는 이미지를 빌드하기 위해, 먼저 `ldconfig -p | grep libfuse.so.2`에 해당하는 위치를 확인합니다:

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

그 후 Ubuntu 운영체제를 대상으로 두 단계 빌드를 수행합니다:

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

여기서 우리는 관련된 동적 링크 라이브러리를 먼저 `/lxcfs` 임시 디렉터리로 이동합니다. HostPath로 덮어쓰지 않도록 하며, 이후 시작 스크립트를 작성하여 관련 동적 링크 라이브러리를 다시 이동시킵니다:

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

### 2. LXCFS Webhook 기능 설계

Webhook 기능은 간단합니다. Kubebuilder 프레임워크를 기반으로 웹훅을 신속하게 구축할 수 있습니다. 우리는 Mutation과 Validate 웹훅을 구현했으며, Validate에서는 주로 Pod과 LXCFS 규칙 무시 관련 Annotation이 올바른 값을 갖는지 확인합니다.

Mutation에서는 먼저 Pod가 Mutate되어야 하는지 확인하고, 그렇다면 Pod에 Mutate된 라벨을 추가하고 Pod에 LXCFS Volume과 VolumeMounts를 추가합니다.

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

## 검증

1c 2G를 요청하고 컨테이너 내에서 CPU와 Memory를 확인하는 방법:

```bash
$ cat /proc/meminfo | grep MemTotal:
MemTotal:        2097152 kB

$ cat /proc/cpuinfo | grep processor
processor       : 0

$ cat /proc/cpuinfo | grep processor | wc -l
1
```

## 요약

위의 솔루션을 통해 머신러닝 플랫폼의 디버깅 작업이 가상 머신처럼 보이도록 할 수 있어, 사용자의 인지 부담을 줄일 수 있습니다. 그러나 LXCFS 솔루션에는 여전히 한계가 있으며, 예를 들어 `nproc` 명령어 같은 경우, 여전히 호스트 머신의 정보를 표시합니다[^4].

머신러닝 플랫폼의 사용자들은 일반적으로 컨테이너 기술에 대해 깊이 이해하지 못하며, 이러한 불일치의 원인과 해결책에 대해 알리는 것이 여전히 우리에게 고민되는 문제입니다.

[^1]: [Slurm 작업 스케줄링 시스템 사용 가이드 |中科大 슈퍼컴퓨팅 센터](https://scc.ustc.edu.cn/hmli/doc/userguide/slurm-userguide.pdf)
[^2]: [컨테이너 리소스 가시성 문제와 GOMAXPROCS 설정 · Issue #216 · islishude/blog](https://github.com/islishude/blog/issues/216)
[^3]: [lxcfs-admission-webhook/lxcfs-image/start.sh at 23298354a1d3cd6eaeb76417aa3fea75df5cdd63 · ThinkBlue1991/lxcfs-admission-webhook · GitHub](https://github.com/ThinkBlue1991/lxcfs-admission-webhook/blob/23298354a1d3cd6eaeb76417aa3fea75df5cdd63/lxcfs-image/start.sh)
[^4]: [lscpu가 물리 서버의 모든 CPU 코어를 보여주는 문제](https://github.com/lxc/lxcfs/issues/181#issuecomment-290458686)