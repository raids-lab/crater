#!/bin/sh
set -eu

ARGS_JSON="${1:-{}}"
if command -v jq >/dev/null 2>&1; then
  NS="$(echo "$ARGS_JSON" | jq -r '.namespace // "default"')"
  FIELD_SELECTOR="$(echo "$ARGS_JSON" | jq -r '.field_selector // ""')"
  LIMIT="$(echo "$ARGS_JSON" | jq -r '.limit // 200')"
else
  NS="default"
  FIELD_SELECTOR=""
  LIMIT="200"
fi

echo "Collecting events in namespace '$NS' (limit=$LIMIT)"
if [ -n "$FIELD_SELECTOR" ]; then
  kubectl get events -n "$NS" --field-selector "$FIELD_SELECTOR" --sort-by=.metadata.creationTimestamp | tail -n "$LIMIT"
else
  kubectl get events -n "$NS" --sort-by=.metadata.creationTimestamp | tail -n "$LIMIT"
fi
