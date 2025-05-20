# Storage Architecture

Crater utilizes a hybrid storage architecture to address both high-performance local workloads and persistent shared data access across pods and nodes. This document outlines the storage solutions used in the cluster.

---

## 1. Local Persistent Volumes (LocalPV via OpenEBS)

We use [OpenEBS LocalPV](https://openebs.io/docs/user-guides/localpv) to manage node-local storage for workloads that require high throughput and data-locality.

### Why LocalPV?

- **CRD-Based Management**: Allows Kubernetes-native declarative management of local disks.
- **Performance**: Data remains on the same node, minimizing network overhead.
- **Use Case in Crater**:
  - Job cache directories
  - Local dataset staging areas
  - Per-node inference or training scratch space

### StorageClass

A dedicated `StorageClass` is configured for local disk provisioning. This class is referenced by other charts like:
- `cloudnative-pg` for database storage
- Distributed job outputs that do not require replication

See: [`deployments/openebs`](../deployments/openebs)

---

## 2. Shared Block Storage (Ceph RBD via Rook)

For persistent, multi-node accessible volumes, Crater uses [Rook-Ceph RBD](https://rook.io/docs/rook/latest/ceph-block.html). RBD volumes are dynamically provisioned and support:

- ReadWriteOnce access with migration support
- Replication and failure recovery
- Suitable for:
  - Prometheus TSDB storage
  - Crater internal services requiring persistence
  - Datasets shared across nodes

Ceph was selected due to its:

- Kubernetes-native provisioning via Rook
- Strong community support
- Scalability and fault tolerance

> 📌 Most of our stateful components such as Prometheus use Ceph RBD volumes via a custom `StorageClass`.

---

## Storage Usage Matrix

| Component          | Type           | Storage Backend       | Notes                                  |
|-------------------|----------------|------------------------|----------------------------------------|
| PostgreSQL (Crater DB) | Persistent    | LocalPV (OpenEBS)     | High-speed, node-specific              |
| Prometheus TSDB   | Persistent    | Ceph RBD (Rook)       | Multi-node, highly durable             |
| User Jobs         | Ephemeral / Persistent | LocalPV / Ceph RBD | Based on configuration                 |
| Grafana Dashboards| Ephemeral / Persistent | Ceph RBD            | Optional, depending on dashboard config|

---

## Notes

- Always ensure nodes offering LocalPV have labeled and mounted disk paths.
- Ceph RBD requires pre-provisioned block storage devices on cluster nodes.
- Use node affinity and tolerations with LocalPV to bind pods to correct storage locations.
- See each chart’s documentation for the appropriate `StorageClass` override.

---

## Related Modules

- [`openebs`](./openebs.md)
- [`cloudnative-pg`](./cloudnative-pg.md)
- [`prometheus`](./prometheus.md)

## Installation

We recommend leveraging NFS as the storage provisioner and using official Helm chart with Crater's preconfigured values.
 
📖 Detailed guide: [`deployments/nfs/README.md`](../deployments/nfs/README.md)