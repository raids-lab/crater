## Installation

We recommend installing `metrics-server` using Helm with Crater’s preconfigured `values.yaml`.

### Step 1: Add the Helm Chart Repository

```bash
helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/
helm repo update
```
### Step 2: Install Metrics Server with Insecure TLS
In clusters where kubelet does not have properly signed certificates, insecure TLS is required.

```bash
helm upgrade --install metrics-server metrics-server/metrics-server \
  --namespace metrics-server \
  --create-namespace \
  -f values.yaml
```

⚠️ --kubelet-insecure-tls is enabled by default in our values.yaml to allow communication with kubelet on most test or local clusters.

