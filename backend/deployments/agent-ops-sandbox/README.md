# Agent Ops Sandbox (Scaffold)

This directory contains a conservative scaffold for the Crater ops-agent sandbox runtime.

It is designed to support:
- isolated execution in a dedicated namespace
- minimal RBAC for read-diagnosis scripts
- explicit script allowlist execution via a toolbox job template

## Files

- `namespace.yaml`: dedicated namespace for sandbox jobs.
- `serviceaccount.yaml`: service account used by sandbox jobs.
- `role.yaml` + `rolebinding.yaml`: namespace-scoped read permissions.
- `clusterrole.yaml` + `clusterrolebinding.yaml`: cluster-scoped read permissions needed by diagnostics.
- `networkpolicy.yaml`: default deny + limited egress (DNS and API server placeholder).
- `toolbox-job.yaml`: job template for running a whitelisted script.
- `kustomization.yaml`: renders all manifests and generates a script catalog ConfigMap.
- `scripts/*.sh`: script catalog (allowlisted operational scripts).

## Backend Config Mapping

These manifests align with `agent.ops` backend config keys:

- `agent.ops.sandbox.namespace`
  - maps to `metadata.name` in `namespace.yaml`
  - and `metadata.namespace` in all namespaced resources
- `agent.ops.sandbox.serviceAccount`
  - maps to `metadata.name` in `serviceaccount.yaml`
  - and `spec.template.spec.serviceAccountName` in `toolbox-job.yaml`
- `agent.ops.sandbox.image`
  - maps to container `image` in `toolbox-job.yaml`
- `agent.ops.sandbox.defaultTimeoutSeconds`
  - maps to `spec.activeDeadlineSeconds` in `toolbox-job.yaml` (default scaffold value)
- `agent.ops.sandbox.maxTimeoutSeconds`
  - enforced by backend before submitting this template (not enforced by YAML itself)
- `agent.ops.sandbox.scriptAllowlist`
  - maps to `scripts/` catalog + `runner.sh` whitelist switch

Web search config keys (`agent.ops.webSearch.*`) are backend/network concerns and are intentionally not represented as Kubernetes manifests in this folder.

## Script Catalog

Whitelisted scripts in this scaffold:
- `inspect_pvc`
- `inspect_mounts`
- `collect_events`
- `inspect_rdma_node`
- `diagnose_nccl_job`

The generated ConfigMap mounts these scripts at `/opt/ops/scripts` and `runner.sh` dispatches by exact script name.

## Render / Apply

```bash
kubectl apply -k backend/deployments/agent-ops-sandbox
```

## Security Notes

- This scaffold is read-diagnosis oriented and intentionally avoids broad write verbs.
- `networkpolicy.yaml` contains a placeholder API-server egress CIDR (`10.96.0.1/32`).
  Update it to your cluster's Kubernetes API service IP/CIDR.
- If stricter isolation is needed, split script execution into per-script service accounts and separate roles.
