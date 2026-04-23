#!/bin/sh
set -eu

ARGS_JSON="${1:-{}}"
if command -v jq >/dev/null 2>&1; then
  NODE_NAME="$(echo "$ARGS_JSON" | jq -r '.node_name // ""')"
else
  NODE_NAME=""
fi

if [ -n "$NODE_NAME" ]; then
  echo "Inspecting RDMA-relevant data for node '$NODE_NAME'"
  kubectl get node "$NODE_NAME" -o yaml
  kubectl describe node "$NODE_NAME"
else
  echo "Listing nodes (RDMA labels can be checked in output)"
  kubectl get nodes -o wide
fi

echo "Searching for common RDMA/IB labels"
kubectl get nodes -o json | jq -r '
  .items[] |
  {
    name: .metadata.name,
    labels: (.metadata.labels // {})
  } |
  select(
    (.labels | to_entries[]?.key | test("rdma|infiniband|mlx|roce"; "i"))
    or
    (.labels | to_entries[]?.value | test("rdma|infiniband|mlx|roce"; "i"))
  )
' 2>/dev/null || true
