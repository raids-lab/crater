#!/bin/sh
set -eu

ARGS_JSON="${1:-{}}"
if command -v jq >/dev/null 2>&1; then
  NS="$(echo "$ARGS_JSON" | jq -r '.namespace // "default"')"
  JOB_NAME="$(echo "$ARGS_JSON" | jq -r '.job_name // ""')"
  TAIL_LINES="$(echo "$ARGS_JSON" | jq -r '.tail_lines // 200')"
else
  NS="default"
  JOB_NAME=""
  TAIL_LINES="200"
fi

if [ -z "$JOB_NAME" ]; then
  echo "job_name is required for diagnose_nccl_job" >&2
  exit 2
fi

echo "Diagnosing NCCL symptoms for job '$JOB_NAME' in namespace '$NS'"
PODS="$(kubectl get pods -n "$NS" -o name | grep "$JOB_NAME" || true)"

if [ -z "$PODS" ]; then
  echo "No pods found matching job name '$JOB_NAME' in namespace '$NS'" >&2
  exit 5
fi

for POD in $PODS; do
  echo "===== ${POD} : describe ====="
  kubectl describe "$POD" -n "$NS" || true
  echo "===== ${POD} : recent logs (tail=${TAIL_LINES}) ====="
  kubectl logs "$POD" -n "$NS" --tail="$TAIL_LINES" || true
done

echo "===== NCCL/Network keyword scan ====="
for POD in $PODS; do
  kubectl logs "$POD" -n "$NS" --tail="$TAIL_LINES" 2>/dev/null | grep -Ei "nccl|socket|rdma|infiniband|ibv|timeout|unreachable|connection reset" || true
done
