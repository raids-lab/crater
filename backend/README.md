# Crater
![alt text](docs/image/icon.png)

# About Crater

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
![alt text](docs/image/architecture.png)

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

Crater provides Helm charts for simple and configurable deployment.

### 🔧 Prerequisites

Make sure Helm is installed on your system:  
📖 [https://helm.sh/docs/intro/install/](https://helm.sh/docs/intro/install/)

Before deploying Crater, please make sure your Kubernetes cluster has the following dependencies installed. All components can be installed via Helm. We provide both official documentation links and local step-by-step guides under the [`docs/`](./docs) folder.

#### 📦 Cluster Resource Dependencies

| Component           | Purpose                                  | Official Docs                                              | Local Guide                    |
|---------------------|-------------------------------------------|------------------------------------------------------------|--------------------------------|
| OpenEBS             | Persistent storage management CRDs        | [openebs.io](https://openebs.io/docs/next/installation)    | [docs/openebs.md](./docs/openebs.md) |
| CloudNativePG       | PostgreSQL database service               | [cloudnative-pg.io](https://cloudnative-pg.io/docs/)       | [docs/cloudnative-pg.md](./docs/cloudnative-pg.md) |
| Prometheus Stack    | Monitoring stack (Prometheus, Grafana)    | [prometheus-community](https://github.com/prometheus-community/helm-charts) | [docs/prometheus.md](./docs/prometheus.md) |
| metrics-server      | Metrics API for autoscaling               | [metrics-server](https://github.com/kubernetes-sigs/metrics-server) | [docs/metrics-server.md](./docs/metrics-server.md) |
| NVIDIA GPU Operator | GPU device plugin and monitoring          | [nvidia.com](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/overview.html) | [docs/gpu-operator.md](./docs/gpu-operator.md) |

#### 🌐 Networking & Routing

| Component     | Purpose                             | Official Docs                                          | Local Guide                         |
|---------------|--------------------------------------|--------------------------------------------------------|--------------------------------------|
| MetalLB       | LoadBalancer support for bare metal  | [metallb.universe.tf](https://metallb.universe.tf/installation/) | [docs/metallb.md](./docs/metallb.md) |
| IngressClass  | Ingress traffic routing              | [kubernetes.io](https://kubernetes.io/docs/concepts/services-networking/ingress/) | [docs/ingress.md](./docs/ingress.md) |

#### 🧠 Scheduling & Orchestration

| Component  | Purpose                                 | Official Docs                                              | Local Guide                        |
|------------|------------------------------------------|------------------------------------------------------------|-------------------------------------|
| Volcano    | Base job scheduling framework            | [volcano.sh](https://volcano.sh/en/docs/installation/)     | [docs/volcano.md](./docs/volcano.md) |
| Aische     | Crater's custom intelligent quota scheduler *(coming soon)* | *(To be released)*                                | *(Coming soon)*  |
| Sparse     | Crater's custom sparse-aware scheduler *(coming soon)* | *(To be released)*                                  | *(Coming soon)* |

#### 🗃️ Platform Services

| Component     | Purpose                                | Official Docs                                                | Local Guide                         |
|----------------|-----------------------------------------|--------------------------------------------------------------|--------------------------------------|
| StorageClass (e.g. Ceph, NFS) | Distributed storage backend           | Varies by provider (e.g. [Rook-Ceph](https://rook.io/docs/rook/latest/)) | [docs/storage.md](./docs/storage.md) |
| Harbor         | Container image registry                | [goharbor.io](https://goharbor.io/docs/)                     | [docs/harbor.md](./docs/harbor.md) |


### 🚀 Quick Start

```bash
# Add Crater Helm repository (replace <repo-url> with actual URL)
helm repo add crater <repo-url>
helm repo update

# Install Crater with default settings
helm install crater crater/crater -n crater
```
### ✅ Verify Installation

```bash
kubectl get pods -n crater
```

### 🌐 Access the Dashboard

If using a NodePort service:

```bash
kubectl get svc -n crater
```
Then visit http://<NodeIP>:<NodePort> in your browser.

### 🛠️ Custom Configuration
You can override default values with your own values.yaml file:
```bash
helm install crater crater/crater -f my-values.yaml
```

More documentation coming soon on configuration, architecture, and contributing!