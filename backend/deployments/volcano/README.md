# Volcano

## Installation

We recommend installing Volcano using Helm with preconfigured values from Crater.

```bash
helm repo add volcano-sh https://volcano-sh.github.io/helm-charts
helm repo update
```

Install Volcano using Helm (recommended version: 1.10.0):

```bash
helm upgrade --install volcano volcano-sh/volcano \
  --namespace volcano-system \
  --create-namespace \
  --version 1.10.0 \
  -f volcano/values.yaml
```

You can test the configuration with:
```bash
helm upgrade --install volcano volcano-sh/volcano \
  --namespace volcano-system \
  --create-namespace \
  --version 1.10.0 \
  -f volcano/values.yaml --dry-run
```

📌 Note: The provided values.yaml contains customized configurations for Crater, including:

* Enabling queue-based scheduling

* Custom scheduler plugins (e.g., gang, priority, capacity)

* Support for preemption policies

# References
* [Volcano GitHub](https://github.com/volcano-sh/volcano)