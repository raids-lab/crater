#!/bin/bash

# Check if at least one node name is provided
if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <node-name> [<node-name>...]"
  exit 1
fi

# Prepare the JSON patch data
PATCH_DATA=$(cat <<EOF
[
  {"op": "remove", "path": "/status/capacity/nvidia.com~1gpu"}
]
EOF
)

# Iterate over each node name provided as an argument
for NODE_NAME in "$@"
do
  # Execute the PATCH request
  curl --header "Content-Type: application/json-patch+json" \
       --request PATCH \
       --data "$PATCH_DATA" \
       http://127.0.0.1:8001/api/v1/nodes/$NODE_NAME/status

  echo "Patch request sent for node $NODE_NAME"
done
