# Prometheus Stack Deployment Guide

This deployment uses the official `kube-prometheus-stack` Helm chart with customized values and subcharts tailored for Crater's observability needs.

We pin the version to ensure compatibility with our configurations and dependencies.

---

## Installation

We recommend using Helm to deploy the stack with our preconfigured values.

### Step 1: Add and Pull the Chart

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm search repo prometheus-community/kube-prometheus-stack

# Pull version 66.2.1 (recommended)
helm pull prometheus-community/kube-prometheus-stack --version 66.2.1
```
### Step 2: Install the Chart

```bash
helm install prometheus prometheus-community/kube-prometheus-stack \
  --version 66.2.1 \
  -f values.yaml \
  -n monitoring
```

## Optional: Customize Subcharts
If you need to override configurations in subcharts like grafana or kube-state-metrics, you can modify them locally.

### Step 1: Edit Subchart Values
Update charts/grafana/values.yaml and charts/kube-state-metrics/values.yaml.

For example, to pin these components to master nodes:

```yaml
nodeSelector:
  role: master
tolerations:
  - key: "node-role.kubernetes.io/control-plane"
    operator: "Exists"
    effect: "NoSchedule"
```

### Step 2: Link Custom Subcharts
Edit Chart.yaml to use local subcharts:
```yaml
dependencies:
  - name: grafana
    repository: file://charts/grafana
  - name: kube-state-metrics
    repository: file://charts/kube-state-metrics
```

### Step 3: Rebuild and Install
```bash
helm dependency update
helm install prometheus ./ --version 66.2.1 -f values.yaml -n monitoring
```

## Expose Prometheus and Grafana
To access Prometheus and Grafana outside the cluster, patch or create NodePort services:

### 1. Expose Prometheus on NodePort 31110
```bash
kubectl patch service prometheus-kube-prometheus-prometheus -n monitoring \
  -p '{"spec": {"type": "NodePort", "ports": [{"port": 9090, "targetPort": 9090, "nodePort": 31110}]}}'
```

### 2. Expose Grafana on NodePort 31120
Grafana's default service may not work for this. Delete the existing service and recreate:
```bash
apiVersion: v1
kind: Service
metadata:
  name: prometheus-grafana
  namespace: monitoring
  labels:
    app.kubernetes.io/instance: prometheus
    app.kubernetes.io/name: grafana
spec:
  selector:
    app.kubernetes.io/instance: prometheus
    app.kubernetes.io/name: grafana
  ports:
    - protocol: TCP
      port: 80
      targetPort: 3000
      nodePort: 31120
  type: NodePort
```

Apply with:
```bash
kubectl apply -f grafana-nodeport-service.yaml
```
# References
* [kube-prometheus-stack Helm Chart](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)

* [NVIDIA DCGM Exporter](https://github.com/NVIDIA/dcgm-exporter)

* [Grafana Documentation](https://grafana.com/docs/)

# Troubleshooting 
Grafana initial user and password:can be found in charts
* admin
* prome-operator

