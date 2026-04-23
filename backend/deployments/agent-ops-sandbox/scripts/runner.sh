#!/bin/sh
set -eu

SCRIPT_NAME="${1:-}"
SCRIPT_ARGS_JSON="${2:-{}}"

if [ -z "$SCRIPT_NAME" ]; then
  echo "SCRIPT_NAME is required" >&2
  exit 2
fi

case "$SCRIPT_NAME" in
  inspect_pvc|inspect_mounts|collect_events|inspect_rdma_node|diagnose_nccl_job)
    ;;
  *)
    echo "script '$SCRIPT_NAME' is not allowlisted" >&2
    exit 3
    ;;
esac

SCRIPT_PATH="/opt/ops/scripts/${SCRIPT_NAME}.sh"
if [ ! -f "$SCRIPT_PATH" ]; then
  echo "script file not found: $SCRIPT_PATH" >&2
  exit 4
fi

echo "Running allowlisted script: $SCRIPT_NAME"
/bin/sh "$SCRIPT_PATH" "$SCRIPT_ARGS_JSON"
