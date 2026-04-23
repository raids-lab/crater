#!/bin/sh
set -eu

ARGS_JSON="${1:-{}}"
if command -v jq >/dev/null 2>&1; then
  NS="$(echo "$ARGS_JSON" | jq -r '.namespace // "default"')"
  POD_NAME="$(echo "$ARGS_JSON" | jq -r '.pod_name // ""')"
else
  NS="default"
  POD_NAME=""
fi

if [ -n "$POD_NAME" ]; then
  echo "Inspecting pod mounts for '$POD_NAME' in namespace '$NS'"
  kubectl get pod "$POD_NAME" -n "$NS" -o yaml
  kubectl describe pod "$POD_NAME" -n "$NS"
else
  echo "Listing pods and PVC claims in namespace '$NS'"
  kubectl get pods -n "$NS" -o wide
  kubectl get pvc -n "$NS" -o wide
fi
