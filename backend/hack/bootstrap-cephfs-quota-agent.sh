#!/usr/bin/env bash

set -euo pipefail

APP_NAMESPACE="${APP_NAMESPACE:-crater-workspace}"
ROOK_NAMESPACE="${ROOK_NAMESPACE:-rook-ceph}"
SOURCE_PVC="${SOURCE_PVC:-crater-rw-storage}"
QUOTA_CLIENT="${QUOTA_CLIENT:-crater-quota}"
QUOTA_PV="${QUOTA_PV:-crater-quota-storage-pv}"
QUOTA_PVC="${QUOTA_PVC:-crater-quota-storage}"
CSI_SECRET="${CSI_SECRET:-crater-quota-csi}"
GENERATED_CEPH_CLIENT_SECRET="${GENERATED_CEPH_CLIENT_SECRET:-rook-ceph-client-${QUOTA_CLIENT}}"

for command in kubectl grep; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf 'required command not found: %s\n' "$command" >&2
    exit 1
  fi
done

if ! kubectl api-resources --api-group=ceph.rook.io -o name | grep -qx 'cephclients.ceph.rook.io'; then
  echo "CephClient CRD is unavailable; install or enable the Rook operator first." >&2
  exit 1
fi

source_pv=$(kubectl -n "$APP_NAMESPACE" get pvc "$SOURCE_PVC" -o jsonpath='{.spec.volumeName}')
if [[ -z "$source_pv" ]]; then
  echo "PVC $APP_NAMESPACE/$SOURCE_PVC is not bound." >&2
  exit 1
fi

capacity=$(kubectl get pv "$source_pv" -o jsonpath='{.spec.capacity.storage}')
storage_class=$(kubectl get pv "$source_pv" -o jsonpath='{.spec.storageClassName}')
driver=$(kubectl get pv "$source_pv" -o jsonpath='{.spec.csi.driver}')
volume_handle=$(kubectl get pv "$source_pv" -o jsonpath='{.spec.csi.volumeHandle}')
cluster_id=$(kubectl get pv "$source_pv" -o jsonpath='{.spec.csi.volumeAttributes.clusterID}')
fs_name=$(kubectl get pv "$source_pv" -o jsonpath='{.spec.csi.volumeAttributes.fsName}')
subvolume_name=$(kubectl get pv "$source_pv" -o jsonpath='{.spec.csi.volumeAttributes.subvolumeName}')
subvolume_path=$(kubectl get pv "$source_pv" -o jsonpath='{.spec.csi.volumeAttributes.subvolumePath}')

if [[ "$driver" != *cephfs.csi.ceph.com || -z "$subvolume_path" || -z "$fs_name" ]]; then
  echo "PVC $APP_NAMESPACE/$SOURCE_PVC is not a supported CephFS CSI volume." >&2
  exit 1
fi

# Rook recommends retaining the original dynamically provisioned PV while a
# static PV points at the same CephFS subvolume.
kubectl patch pv "$source_pv" --type=merge \
  -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}' >/dev/null

cat <<EOF | kubectl apply -f -
apiVersion: ceph.rook.io/v1
kind: CephClient
metadata:
  name: ${QUOTA_CLIENT}
  namespace: ${ROOK_NAMESPACE}
spec:
  caps:
    mds: "allow rwp fsname=${fs_name} path=${subvolume_path}"
    mon: "allow r fsname=${fs_name}"
    osd: "allow rw tag cephfs data=${fs_name}"
EOF

generated_secret="$GENERATED_CEPH_CLIENT_SECRET"
for _ in $(seq 1 30); do
  status_secret=$(kubectl -n "$ROOK_NAMESPACE" get cephclient "$QUOTA_CLIENT" \
    -o jsonpath='{.status.info.secretName}' 2>/dev/null || true)
  if [[ -n "$status_secret" ]]; then
    generated_secret="$status_secret"
  fi
  if kubectl -n "$ROOK_NAMESPACE" get secret "$generated_secret" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

if ! kubectl -n "$ROOK_NAMESPACE" get secret "$generated_secret" >/dev/null 2>&1; then
  echo "Rook did not create a Secret for CephClient $QUOTA_CLIENT." >&2
  exit 1
fi

client_key=$(kubectl -n "$ROOK_NAMESPACE" get secret "$generated_secret" \
  -o jsonpath='{.data.userKey}')
if [[ -z "$client_key" ]]; then
  client_key=$(kubectl -n "$ROOK_NAMESPACE" get secret "$generated_secret" \
    -o "go-template={{ index .data \"${QUOTA_CLIENT}\" }}")
fi
if [[ -z "$client_key" ]]; then
  echo "The generated CephClient Secret contains neither userKey nor $QUOTA_CLIENT." >&2
  exit 1
fi

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: ${CSI_SECRET}
  namespace: ${ROOK_NAMESPACE}
type: Opaque
stringData:
  userID: ${QUOTA_CLIENT}
data:
  userKey: ${client_key}
EOF
unset client_key

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: ${QUOTA_PV}
spec:
  capacity:
    storage: ${capacity}
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  storageClassName: ${storage_class}
  volumeMode: Filesystem
  csi:
    driver: ${driver}
    volumeHandle: ${volume_handle}-quota-agent
    nodeStageSecretRef:
      name: ${CSI_SECRET}
      namespace: ${ROOK_NAMESPACE}
    volumeAttributes:
      clusterID: ${cluster_id}
      fsName: ${fs_name}
      subvolumeName: ${subvolume_name}
      subvolumePath: ${subvolume_path}
      rootPath: ${subvolume_path}
      staticVolume: "true"
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ${QUOTA_PVC}
  namespace: ${APP_NAMESPACE}
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: ${capacity}
  storageClassName: ${storage_class}
  volumeMode: Filesystem
  volumeName: ${QUOTA_PV}
EOF

echo
echo "Quota-agent storage is ready:"
echo "  CephClient: ${ROOK_NAMESPACE}/${QUOTA_CLIENT}"
echo "  PVC:        ${APP_NAMESPACE}/${QUOTA_PVC}"
echo
echo "Enable the Helm component with:"
printf '  --set quotaAgent.enabled=true \\\n'
printf '  --set quotaAgent.existingClaim=%s \\\n' "$QUOTA_PVC"
printf '  --set backendConfig.storage.quota.rookNamespace=%s \\\n' "$ROOK_NAMESPACE"
printf '  --set-string backendConfig.storage.quota.cephFSCSIDriver=%s\n' "$driver"
