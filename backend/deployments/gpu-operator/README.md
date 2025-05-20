## Helm Repository

### Add NVIDIA Helm Repo

```bash
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia
helm repo update
```

### Download the GPU Operator Chart

```bash
helm pull nvidia/gpu-operator --untar
```
## DCGM Metrics Configuration
To enable DCGM metrics collection:

### 1. Prepare the metrics configuration

```bash
curl https://raw.githubusercontent.com/NVIDIA/dcgm-exporter/main/etc/dcp-metrics-included.csv > dcgm-metrics.csv
kubectl create configmap metrics-config -n nvidia-gpu-operator --from-file=dcgm-metrics.csv
```

### 2. Modify values.yaml
In values.yaml, set the following to enable DCGM metrics:
```yaml
dcgmExporter:
  config:
    name: metrics-config
  env:
    - name: DCGM_EXPORTER_COLLECTORS
      value: /etc/dcgm-exporter/dcgm-metrics.csv
  serviceMonitor:
    enabled: true
    honorLabels: true

image:
  repository: <your-image-repo>
```
Replace <your-image-repo> with your private registry

## Device Plugin Customization
Crater uses a customized version of the NVIDIA Device Plugin (v0.17.0-ubi9) to support resource renaming, allowing advanced GPU allocation strategies.

Refer to Volcano [resource-naming](https://github.com/volcano-sh/devices/blob/release-1.1/docs/resource-naming/README.md) Guide for customization instructions.

You may need to pre-clean drivers from certain nodes:
```bash
chmod +x ./clean-gpu.sh
./clean-gpu.sh dell-gpu-24 dell-gpu-25 ... inspur-gpu-14
```

### Install GPU Operator via Helm
```bash
helm upgrade --install nvdp nvidia/gpu-operator \
  -f values.yaml \
  -n nvidia-gpu-operator \
  --create-namespace
```

### Optional: Enable Time-Slicing

```bash
kubectl label node lenovo-01 nvidia.com/device-plugin.config=default
kubectl label node lenovo-03 nvidia.com/device-plugin.config=default

kubectl rollout restart -n nvidia-gpu-operator daemonset/nvidia-device-plugin-daemonset
```

## ServiceMonitor for Prometheus
To expose GPU metrics to Prometheus, create a ServiceMonitor:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nvidia-dcgm-exporter
  namespace: monitoring
  labels:
    app: nvidia-dcgm-exporter
    release: prometheus
spec:
  endpoints:
    - interval: 1s
      path: /metrics
      port: gpu-metrics
  selector:
    matchLabels:
      app: nvidia-dcgm-exporter
  namespaceSelector:
    matchNames:
      - nvidia-gpu-operator
```

## Troubleshooting
### 1. Node Feature Discovery Image Pull Issue
The node-feature-discovery image may be slow to pull from public sources. Use a mirror site

### 2. Driver Module Stuck
You can manually unload legacy NVIDIA drivers:
```bash
sudo rmmod nvidia_uvm nvidia_drm nvidia_modeset nvidia
```

### 3. nvidia-container-toolkit-daemonset Hangs
If the pod hangs during initialization, simply restart the pod:
```bash
kubectl rollout restart daemonset/nvidia-container-toolkit-daemonset -n nvidia-gpu-operator
```
### 4. Post-Delete Job Pull Secret Fix
If Helm uninstall fails due to a post-delete hook job missing pull secrets, patch the job like below:
```yaml
...
spec:
  template:
    ...
    spec:
      imagePullSecrets:
        - name: <your-secret>
```
Update the post-delete job in the charts/node-feature-discovery/templates/prune-job.yaml.

# References
* [GPU Operator Docs](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/)

* [DCGM Exporter](https://github.com/NVIDIA/dcgm-exporter)

* [Volcano Resource Naming](https://github.com/volcano-sh/devices/blob/release-1.1/docs/resource-naming/README.md)

* [NVIDIA Helm Chart](https://github.com/NVIDIA/gpu-operator/tree/main/deployments/helm)