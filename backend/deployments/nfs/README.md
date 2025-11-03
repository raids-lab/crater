# Setting Up NFS in Kubernetes

In the Crater cluster, NFS (Network File System) is used to provide scalable and shared persistent storage, which is especially useful for AI scenarios such as dataset caching and result sharing. This guide outlines how to install and integrate NFS with Kubernetes.

---

## 1. Install NFS Server

### Step 1: Install Required Packages

Install the NFS service on the designated server node:

```bash
# For RedHat/CentOS:
yum -y install nfs-utils rpcbind

# For Ubuntu/Debian:
apt install nfs-kernel-server
```

### Step 2: Configure Shared Directory
Edit the /etc/exports file and define the shared path and access permissions:

```bash
/data/nfs 10.8.0.0/24(rw,sync,no_root_squash,no_subtree_check) 10.244.0.0/16(rw,sync,no_root_squash,no_subtree_check)
```
Explanation:

* /data/nfs: the directory to be shared

* IP ranges: Pod subnet 10.244.0.0/16 and node subnet 10.8.0.0/24

* Permissions:

* * rw: read/write

* * sync: synchronous writes

* * no_root_squash: preserve root privileges

* * no_subtree_check: disable subtree checking

### Step 3: Start the NFS Service
```bash
systemctl start nfs-server
systemctl enable nfs-server
```

## 2. Install NFS Client on Cluster Nodes (Recommended: Ansible)
You can use Ansible to install the necessary NFS client packages across all cluster nodes:

```yaml
- name: Install NFS support
  hosts: all
  tasks:
    - name: Detect OS
      command: cat /etc/os-release
      register: os_info

    - name: Install on Debian/Ubuntu
      apt:
        name: nfs-common
        state: present
      when: '"Debian" in os_info.stdout or "Ubuntu" in os_info.stdout'

    - name: Install on CentOS/RHEL
      yum:
        name: nfs-utils
        state: present
      when: '"CentOS" in os_info.stdout or "Red Hat Enterprise Linux" in os_info.stdout'
```
Example hosts.ini inventory:
```ini
[ubuntu]
10.8.0.1
10.8.0.6

[centos]
10.8.0.10
10.8.0.11
```

## 3. Test NFS Mount on Clients
On a client node, verify that you can see the shared directory:
```bash
showmount -e <NFS_SERVER_IP>
```

## 4. Integrate with Kubernetes
### Step 1: Install the NFS Subdir External Provisioner via Helm
Add the Helm chart repository:
```bash
helm repo add nfs-subdir-external-provisioner https://kubernetes-sigs.github.io/nfs-subdir-external-provisioner
helm repo update
```

Install the NFS provisioner into the kube-system namespace:

```bash
helm install -n kube-system nfs-client nfs-subdir-external-provisioner/nfs-subdir-external-provisioner \
--set nfs.server=10.8.0.10 \
--set nfs.path=/data/nfs \
--set storageClass.defaultClass=true \
--set image.repository=crater-harbor.act.buaa.edu.cn/registry.k8s.io/sig-storage/nfs-subdir-external-provisioner
```

### Step 2: Create a PVC and Pod to Test NFS
PersistentVolumeClaim (PVC):
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: nfs-client-pvc
spec:
  storageClassName: nfs-client
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 1Gi
```
Test Pod:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nfs-test-pod
spec:
  containers:
    - name: test
      image: crater-harbor.act.buaa.edu.cn/dockerhub/library/busybox:latest
      command: ["sh", "-c", "echo 'Hello from NFS test pod' > /data/test.txt && sleep 3600"]
      volumeMounts:
        - mountPath: /data
          name: nfs-vol
  volumes:
    - name: nfs-vol
      persistentVolumeClaim:
        claimName: nfs-client-pvc
```

Apply the resources:

```bash
kubectl apply -f nfs-client-pvc.yaml
kubectl apply -f nfs-test-pod.yaml
```

## 5. Verify the Result
Check pod status:

```bash
kubectl get pods
kubectl describe pod nfs-test-pod
```
View pod logs:
```bash
kubectl logs nfs-test-pod
```
On the NFS server, check for a new subdirectory in /data/nfs/, and confirm the presence of test.txt:
```bash
ls /data/nfs/*
cat /data/nfs/*/test.txt
```
With the above steps, you will have successfully set up NFS-based persistent storage in the Crater Kubernetes cluster, verified it works, and ensured it supports shared volume access across multiple pods or nodes.
