#!/bin/sh
set -eu

ARGS_JSON="${1:-{}}"
if command -v jq >/dev/null 2>&1; then
  NS="$(echo "$ARGS_JSON" | jq -r '.namespace // "default"')"
  PVC_NAME="$(echo "$ARGS_JSON" | jq -r '.pvc_name // ""')"
else
  NS="default"
  PVC_NAME=""
fi

if [ -n "$PVC_NAME" ]; then
  echo "Inspecting PVC '$PVC_NAME' in namespace '$NS'"
  kubectl get pvc "$PVC_NAME" -n "$NS" -o yaml
  kubectl describe pvc "$PVC_NAME" -n "$NS"
  kubectl get events -n "$NS" --field-selector "involvedObject.kind=PersistentVolumeClaim,involvedObject.name=${PVC_NAME}" --sort-by=.metadata.creationTimestamp
else
  echo "Listing PVCs in namespace '$NS'"
  kubectl get pvc -n "$NS" -o wide
fi
