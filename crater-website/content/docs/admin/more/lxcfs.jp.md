---
title: "LXCFS 設定"
description: "LXCFS をリソースビューの分離をサポートするために設定する"
---

## 背景紹介

過去2年間、Kubernetesに基づいてクラウドネイティブな機械学習プラットフォームを構築しており、もともとSlurmに基づくクラスタスケジューラーを段階的に置き換えています。

もともとの方法とコンテナベースの方法との互換性をできるだけ保つために、いくつかの試行錯誤を行いましたが、いくつかの問題が残っています。例えばコンテナ内のリソースの可視性がーー

### ユーザーストーリー

小明は深層学習の修士課程の学生であり、クラウドネイティブな機械学習プラットフォームのユーザーでもあります。

この日、彼はプラットフォームでJupyterデバッグジョブを申請しました。ジョブを起動する際、小明はCPU、Memory、GPUの数とモデルを選択し、その後プラットフォームはこれらの制限をKubernetes Pod ResourcesのRequestsとLimitsとしてレンダリングします：

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

ジョブが起動した後、小明は`nvidia-smi`コマンドをジョブ内で実行し、正常に1枚のGPUが表示されます。しかし、`lscpu`、`top`などのコマンドを実行すると、CPUコアやメモリ容量は彼が申請した16C 32G（実際にはホストマシンのリソース数）を大きく超えています：

```bash
$ top
MiB Mem : 385582.0 total, 258997.6 free,  24158.2 used, 105203.0 buff/cache
```

小明自身はコンテナ技術に詳しくなく、機械学習プラットフォームが仮想マシンのようなリソースを割り当てていると誤解しているため、このような挙動に困惑しています。

### 解決策

上記の問題はユーザー体験だけでなく、プログラムの性能にも影響を与える可能性があります。JavaやGoなどのプログラムでは、Goのプログラムを例にすると、起動時にCPU数に基づいて`GOMAXPROCS`変数を設定し、実行可能な最大スレッド数を示します[^2]。しかしコンテナ環境では、この変数の値はホストマシンの値であり、少量のCPUで多くのスレッドを起動すると、頻繁なスレッド切り替えのオーバーヘッドによりプログラムの実行速度が遅くなる可能性があります。

これに対し、私たちは2つの解決策を持っています：

1. **ユーザー認識**：Slurmでは、ジョブに以下のような環境変数を注入し、実際に申請されたリソースを示します[^1]：

| 変数名                | 説明                      |
| --------------------- | ------------------------- |
| `SLURM_CPUS_ON_NODE`  | 分配されたノード上のCPU数 |
| `SLURM_CPUS_PER_TASK` | 各タスクのCPU数           |
| `SLURM_GPUS_PER_NODE` | 各ノードで必要なGPU数     |
| `SLURM_MEM_PER_NODE`  | 各ノードで必要なメモリ数  |

同様に、Podを起動する際に関連する環境変数を注入し、ユーザーとの約束を行います。

2. **ユーザー認識なし**（ただし、ある程度の限界があります）：以下で紹介するLXCFS。

## LXCFS 介紹

LXCFS（Linux Container Filesystem）は、ユーザー空間のファイルシステム実装で、FUSEファイルシステムに基づいており、Linuxコンテナ環境におけるprocファイルシステム（procfs）の固有の制限を解決することを目的としています。

具体的には、以下の2つの主な内容を提供します：

1. 一連のファイルが、オリジナルの`/proc`ファイルにバインドマウントされ、CGroup認識値を提供します。
2. コンテナ認識型のcgroupfsのようなツリー。

LXCFSを用いることで、コンテナ内で`/proc/cpuinfo`などの情報をクエリする際、LXCFSがFUSE方式で「ハイジャック」し、コンテナの`cgroup`情報に基づいて正しい結果を提供します。

## 現在のLXCFS for Kubernetes方案の不足

> [!quote]
>
> - [技術共有：Pod リソースビューの分離の実装 \| 董江ブログ \| DongJiang Blog](https://kubeservice.cn/2021/04/27/k8s-lxcfs-overview/)

上記のアイデアは難しくありません。すでに多くのLXCFS for Kubernetesのオープンソース方案があります：

| プロジェクト                                                                                        | 备考                                      |
| ------------------------------------------------------------------------------------------- | ----------------------------------------- |
| [denverdino/lxcfs-admission-webhook](https://github.com/denverdino/lxcfs-admission-webhook) | Star数が最多ですが、機能が不完全で、長期間メンテナンスされていない |
| [kubeservice-stack/lxcfs-webhook](https://github.com/kubeservice-stack/lxcfs-webhook)       | 更新が速いが、いくつかのエラーがある（後ほどPRを提出する予定） |
| [cndoit18/lxcfs-on-kubernetes](https://github.com/cndoit18/lxcfs-on-kubernetes)             | メンテナンスが少ない                      |

（TODO：上記の方案の原理についても簡単な紹介を行うが、ここでは飛ばす。読者は関連ブログを参照してください）

しかし、これらの方案を深く理解して使用した後、これらの方案にはいくつかの問題があることを発見しました：

### 1. ノード再起動後のPodリソース情報の異常

> [!quote]
>
> - [TIPS：Kubernetes LXCFS故障復旧後、既存のPodに対してremount操作を実行 \| 董江ブログ \| DongJiang Blog](https://kubeservice.cn/2022/04/13/k8s-lxcfs-remount/)
> - [lxcfsのKubernetes実践 - 廖思睿の個人ブログ](https://blog.liaosirui.com/%E7%B3%BB%E7%BB%9F%E8%BF%90%E7%BB%B4/E.%E5%AE%B9%E5%99%A8%E4%B8%8E%E5%AE%B9%E5%99%A8%E7%BC%96%E6%8E%92/%E5%AE%B9%E5%99%A8%E6%8A%80%E6%9C%AF%E7%9A%84%E5%9F%BA%E7%9F%B3/lxcfs/lxcfs%E7%9A%84%E4%BD%BF%E7%94%A8/lxcfs%E7%9A%84Kubernetes%E5%AE%9E%E8%B7%B5.html)
> - [lxcfs-admission-webhook/lxcfs-image/start.sh at 23298354a1d3cd6eaeb76417aa3fea75df5cdd63 · ThinkBlue1991/lxcfs-admission-webhook · GitHub](https://github.com/ThinkBlue1991/lxcfs-admission-webhook/blob/23298354a1d3cd6eaeb76417aa3fea75df5cdd63/lxcfs-image/start.sh)

[kubeservice-lxcfs-webhook 1.4.0 · kubeservice/kubservice-charts](https://artifacthub.io/packages/helm/kubservice-charts/kubeservice-lxcfs-webhook?modal=values)

[lxcfs-on-kubernetes/charts/lxcfs-on-kubernetes/values.yaml at master · cndoit18/lxcfs-on-kubernetes · GitHub](https://github.com/cndoit18/lxcfs-on-kubernetes/blob/master/charts/lxcfs-on-kubernetes/values.yaml)

[コンテナライフサイクルフック \| Kubernetes](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#:~:text=This%20hook%20is%20called%20immediately%20before%20a%20container,liveness%2Fstartup%20probe%20failure%2C%20preemption%2C%20resource%20contention%20and%20others.)

LXCFSが正常に動作している場合、Podは上書きされたUptimeなどの情報を閲覧できます：

```bash
$ top
top - 07:47:52 up 9 min,  0 users,  load average: 0.00, 0.00, 0.00
```

しかし、ノードが再起動された場合、デフォルトではLXCFSがPod内の情報を再書き込みしません：

```bash
$ top
top: failed /proc/stat open: Transport endpoint is not connected
```

この問題に対して、コミュニティも対応策を提案しています[^3]。KubernetesのContainer Lifecycle Hooksメカニズムを活用し、ノード再起動時にLXCFSを起動する際に、現在の各Podに対してマウントを再追加します。

この方案ではノードにLXCFSをインストールし、SystemdでLXCFSの自動起動を設定する必要があります。これは非常にクラウドネイティブではありません。そのため、LXCFSコンテナ内にcontainerdの関連ソケットをマウントすることで、ホストマシンの能力に依存しないようにできます。

### 2. LXCFSコンテナ終了後に再作成できない（死鎖）

デバッグ中に発見したように、LXCFS DaemonSetが終了した場合、ノード再起動前でLXCFS Daemonsetを再インストールする際、必ず失敗します：

```bash
$ kubectl get pods
NAME         READY  STATUS                RESTARTS  AGE
lxcfs-77c87  0/1    CreateContainerError  0         18m
```

これは、LXCFSが作成時にホストマシン上の`/var/lib/lxcfs`ディレクトリをマウントする必要があり、このディレクトリはLXCFSが作成した後にのみ成功してマウントされるため、死鎖が発生しているためです。

これを解決するために、LXCFSが終了する際にKubernetesのContainer Lifecycle Hooksメカニズムを活用し、終了前に関連マウントポイントを削除します。

```yaml
preStop:
  exec:
	command:
	  - bash
	  - -c
	  - nsenter -m/proc/1/ns/mnt fusermount -u /var/lib/lxc/lxcfs 2> /dev/null || true
```

この方法も万全ではありません。もしもまだクリーンアップされていない場合は、ノードを再起動する必要があります。

この問題を解決するために、LXCFSの親ディレクトリを指す別のvolumes宣言を作成し、init Container内で残留マウントをアンマウントするようにする必要があります。これにより、万全になります。

### 3. サポートされているLXCFSバージョンが比較的古く

現在、LXCFSは6.0バージョンまで更新されていますが、コミュニティの主流バージョンは4.0です。

ただし、高バージョンのLXCFSはglibcなどにもより高い要件があります。そのため、クラスタの実際の状況に応じて使用するバージョンを選択する必要があります。

### 4. ホストマシンの`libfuse.so`に依存

> [!quote] [LXCFSのDockerとKubernetesでの実践](https://zhuanlan.zhihu.com/p/348625551)

KubernetesでDaemonSetをデプロイする際、エラーが発生することがあります：

```text
/usr/local/bin/lxcfs: error while loading shared libraries: libfuse.so.2: cannot open shared object file: No such file or directory
```

この問題を解決するための第一の方法は、ノードに`libfuse2`をインストールすること（CentOSでは異なる）であり、Ansibleでノード上の`libfuse2`がインストールされていることを保証します：

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

別の方法では、Dockerfileの構築方法と起動スクリプトを変更し、LXCFSコンテナが実行されるときに必要な動的リンクライブラリを含むようにします。

## LXCFS Webhookのインストール

[raids-lab/lxcfs-webhook](https://github.com/raids-lab/lxcfs-webhook)

上記の問題に対応して、複数の方案を統合・最適化し、Yet Another LXCFS Webhookを提供しています。

### 1. 依存

まず、Cert Managerをインストールします（まだインストールしていない場合）：

```bash
helm repo add jetstack https://charts.jetstack.io --force-update
```

cert-manager Helmチャートをインストールするには、以下の通りHelm installコマンドを使用します。

```bash
helm install \
cert-manager jetstack/cert-manager \
--namespace cert-manager \
--create-namespace \
--version v1.17.2 \
--set crds.enabled=true
```

### 2. Helm経由でのインストール

[raids-lab/lxcfs-webhook](https://github.com/raids-lab/lxcfs-webhook)

上記コードをクローンした後、Helmでインストールします：

```bash
helm upgrade --install lxcfs-webhook ./dist/chart -n lxcfs
```

これにより、LXCFS DaemonSet、Webhookを含み、ノード再起動やDaemon再起動などの問題を解決します。

### 3. 作用域の指定

その後、名前空間にラベルを追加します：

```bash
kubectl label namespace <namespace-name> lxcfs-admission-webhook:enabled
```

対応する名前空間内のPodは、作成時に自動的にLXCFSのマウントを行います。

## LXCFS Webhookの設計

### 1. LXCFS DaemonSetイメージの構築

ホストマシンの`libfuse.go`に依存しないイメージを構築するために、まず`ldconfig -p | grep libfuse.so.2`に対応する位置を確認します：

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

その後、Ubuntuオペレーティングシステムを対象として、二段階構築を行います：

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

ここでは、関連する動的リンクライブラリを一時的に`/lxcfs`ディレクトリに移動させ、ホストパスが上書きしないようにします。その後、起動スクリプトを編成し、関連する動的リンクライブラリを再び移動します：

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

### 2. LXCFS Webhookの機能設計

Webhookの機能は比較的単純です。Kubebuilderのフレームワークに基づいて、Webhookを簡単に構築できます。私たちはMutationとValidateのWebhookを実装しており、Validateでは、PodとLXCFSルール無視に関するアノテーションが正しい値を持つかを主にチェックします。

Mutationでは、まずPodがMutateする必要があるかをチェックし、必要であればPodにMutateされたタグを追加し、PodにLXCFSのVolumesとVolumeMountsを追加します。

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

## 検証

1c 2Gを申請し、コンテナ内でCPUとMemoryを確認する方法：

```bash
$ cat /proc/meminfo | grep MemTotal:
MemTotal:        2097152 kB

$ cat /proc/cpuinfo | grep processor
processor       : 0

$ cat /proc/cpuinfo | grep processor | wc -l
1
```

## 結論

上記の方案により、機械学習プラットフォームのデバッグジョブは仮想マシンのように見え、ユーザーの認識の負荷を軽減できます。しかし、LXCFSの方案にはいくつかの限界があります。例えば、よく使われる`nproc`コマンドは、ホストマシンの情報を依然として表示しています[^4]。

機械学習プラットフォームのユーザーは、通常コンテナ技術について詳しくなく、これらの不一致の原因と解決策を知る方法は、我々にとって依然として難しい問題です。

[^1]: [Slurmジョブスケジューラーの使用ガイド | 中国科大スーパーコンピューティングセンター](https://scc.ustc.edu.cn/hmli/doc/userguide/slurm-userguide.pdf)
[^2]: [コンテナリソース可視性の問題とGOMAXPROCSの設定 · Issue #216 · islishude/blog](https://github.com/islishude/blog/issues/216)
[^3]: [lxcfs-admission-webhook/lxcfs-image/start.sh at 23298354a1d3cd6eaeb76417aa3fea75df5cdd63 · ThinkBlue1991/lxcfs-admission-webhook · GitHub](https://github.com/ThinkBlue1991/lxcfs-admission-webhook/blob/23298354a1d3cd6eaeb76417aa3fea75df5cdd63/lxcfs-image/start.sh)
[^4]: [lscpuは物理サーバーのすべてのCPUコアを表示](https://github.com/lxc/lxcfs/issues/181#issuecomment-290458686)