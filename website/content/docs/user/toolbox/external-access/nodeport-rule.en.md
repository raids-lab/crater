---
title: NodePort Access Rules
description: NodePort rules can expose service ports and allow external users to access them via the IP address of a cluster node and a specified port number.
---

## 2.1 Feature Introduction

**NodePort rules** allow you to directly access services in a Kubernetes cluster through an external IP address. Unlike Ingress rules, NodePort rules expose service ports and allow external users to access them through the IP address of a cluster node and a specified port number. NodePort rules are suitable for applications that require external access, such as **SSH connections**.

In NodePort rules, **Kubernetes automatically assigns a port number from the range 30000 to 32767 for the service**. For example, if you want to connect to a node in the cluster via SSH, Kubernetes will assign a port for the service, and you can connect from the outside using that port number.

**Advantages**:

- Suitable for applications requiring direct external access, such as SSH connections.
- Automatically assigns a port for the service, simplifying configuration.
- Does not depend on HTTP/HTTPS protocols.

**Use Cases**:

- Connecting to nodes in the cluster via SSH (port 22).
- Other applications that require access through specific ports.

**Forwarding Path**: Accessible via `{nodeIP}:{nodePort}`. Here, `nodeIP` is the address of any node in the cluster, and `nodePort` is the assigned port number.

![nodeport-intro](./img/nodeport-intro.webp)

After configuration, the following content will be visible in the corresponding Pod's `Annotations`, using `nodeport.crater.raids.io` as the `key`:

```yaml
metadata:
  annotations:
    crater.raids.io/task-name: tensorboard-example
    nodeport.crater.raids.io/smtp: '{"name":"smtp","containerPort":25,"address":"192.168.5.82","nodePort":30631}'
    nodeport.crater.raids.io/ssh: '{"name":"ssh","containerPort":22,"address":"192.168.5.82","nodePort":32513}'
    nodeport.crater.raids.io/telnet: '{"name":"telnet","containerPort":23,"address":"192.168.5.82","nodePort":32226}'
```

## 2.2 Usage Example

When you want to access certain applications (such as SSH connections) via an external IP address, you can use **NodePort rules**. For example, you can configure a NodePort rule to expose the SSH port (port 22) to enable remote development via tools such as VSCode.

**Steps to set up a NodePort external access rule**:

1. Click **"Set External Access Rule"** on the job details page.

   ![ingress-entrance](./img/ingress-entrance.webp)

2. Click **"Add NodePort Rule"** in the popped-up dialog, enter the corresponding **rule name** (only lowercase letters, no more than 20 characters, and must be unique), as well as the **container port**, and click Save.

   ![nodeport-new](./img/nodeport-new.webp)

3. After successfully saving, you will see the **corresponding NodePort rule**.

   ![nodeport-ssh](./img/nodeport-ssh.webp)

**Example Configuration**:

```json
{
  "name": "ssh",
  "containerPort": 22,
  "address": "192.168.5.82",
  "nodePort": 32513
}
```

**Field Description**:

- **Container port number** (`containerPort`): Select port **22**, usually used for SSH services.
- **Cluster node address** (`address`): The IP address of any node in the cluster.
- **Assigned NodePort port** (`nodePort`): Kubernetes automatically assigns a port number for the service from the range 30000 to 32767.

**Access Method**:

- Kubernetes automatically assigns a port number for the SSH service, and you can use this port to establish an SSH connection via an external IP (e.g., for remote development using VSCode).
- For example, Kubernetes assigns a port for this service (such as `32513`), and you can connect using `ssh user@<node-ip>:32513`.

The effect of connecting to a remote Jupyter Notebook via NodePort in VSCode is as follows:

![vscode-nodeport-ssh](./img/vscode-nodeport-ssh.webp)