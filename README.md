# ![crater](./crater-website/content/docs/admin/assets/icon.webp) Crater

<table>
  <tr>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/jupyter.gif"><br>
      <em>Jupyter Lab</em>
    </td>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/ray.gif"><br>
      <em>Ray Job</em>
    </td>
  </tr>
  <tr>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/monitor.gif"><br>
      <em>Monitor</em>
    </td>
    <td align="center" width="45%">
      <img src="https://github.com/raids-lab/crater-frontend/blob/main/docs/images/datasets.gif"><br>
      <em>Models</em>
    </td>
  </tr>
</table>

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
![crater architecture](./crater-website/content/docs/admin/assets/architecture.webp)

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

See [Deployment Guide](https://raids-lab.github.io/crater/zh/docs/admin/) for more details.