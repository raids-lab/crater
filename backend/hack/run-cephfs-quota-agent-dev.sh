#!/usr/bin/env bash

set -euo pipefail

APP_NAMESPACE="${APP_NAMESPACE:-crater-workspace}"
QUOTA_PVC="${QUOTA_PVC:-crater-quota-storage}"
DEV_POD="${DEV_POD:-crater-quota-agent-dev}"
DEV_SECRET="${DEV_SECRET:-crater-quota-agent-dev-auth}"
TARGET_ARCH="${TARGET_ARCH:-amd64}"
DEV_IMAGE="${DEV_IMAGE:-ghcr.io/raids-lab/storage-server:latest}"
CONFIG_FILE="${CONFIG_FILE:-backend/etc/debug-config.yaml}"
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
BUILD_OUTPUT="$ROOT_DIR/backend/.gocache/storage-server-linux-${TARGET_ARCH}"

if [[ "${1:-}" == "--cleanup" ]]; then
  kubectl -n "$APP_NAMESPACE" delete pod "$DEV_POD" --ignore-not-found
  kubectl -n "$APP_NAMESPACE" delete secret "$DEV_SECRET" --ignore-not-found
  exit 0
fi

for command in kubectl go awk base64 sha256sum; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf 'required command not found: %s\n' "$command" >&2
    exit 1
  fi
done

REQUIRED_GO_VERSION=$(awk '$1 == "go" { print $2; exit }' "$ROOT_DIR/backend/go.mod")
if [[ -z "$REQUIRED_GO_VERSION" ]]; then
  echo "Go version was not found in backend/go.mod" >&2
  exit 1
fi

if [[ ! -f "$ROOT_DIR/$CONFIG_FILE" ]]; then
  echo "config file not found: $ROOT_DIR/$CONFIG_FILE" >&2
  exit 1
fi

pvc_status=$(kubectl -n "$APP_NAMESPACE" get pvc "$QUOTA_PVC" \
  -o jsonpath='{.status.phase}' 2>/dev/null || true)
if [[ "$pvc_status" != "Bound" ]]; then
  echo "PVC $APP_NAMESPACE/$QUOTA_PVC is not Bound." >&2
  echo "Run backend/hack/bootstrap-cephfs-quota-agent.sh first." >&2
  exit 1
fi

access_token_secret=$(awk '
  /^[[:space:]]*accessTokenSecret:[[:space:]]*/ {
    value = $0
    sub(/^[[:space:]]*accessTokenSecret:[[:space:]]*/, "", value)
    gsub(/^["'\'' ]+|["'\'' ]+$/, "", value)
    print value
    exit
  }
' "$ROOT_DIR/$CONFIG_FILE")
if [[ -z "$access_token_secret" ]]; then
  echo "auth.token.accessTokenSecret was not found in $CONFIG_FILE" >&2
  exit 1
fi

mkdir -p "$ROOT_DIR/backend/.gocache"
(
  cd "$ROOT_DIR/backend"
  GOCACHE="$ROOT_DIR/backend/.gocache" \
    GOTOOLCHAIN="${GOTOOLCHAIN:-go${REQUIRED_GO_VERSION}}" \
    CGO_ENABLED=0 GOOS=linux GOARCH="$TARGET_ARCH" \
    go build -o "$BUILD_OUTPUT" ./cmd/storage-server
)

internal_token=$(printf 'crater-storage-quota:%s' "$access_token_secret" | sha256sum | awk '{print $1}')
unset access_token_secret
secret_base64=$(printf '%s' "$internal_token" | base64 | tr -d '\r\n')
unset internal_token
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: ${DEV_SECRET}
  namespace: ${APP_NAMESPACE}
type: Opaque
data:
  internal-token: ${secret_base64}
EOF
unset secret_base64

kubectl -n "$APP_NAMESPACE" delete pod "$DEV_POD" --ignore-not-found --wait=true
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: ${DEV_POD}
  namespace: ${APP_NAMESPACE}
  labels:
    app: crater-quota-agent-dev
spec:
  nodeSelector:
    kubernetes.io/arch: ${TARGET_ARCH}
  containers:
    - name: quota-agent
      image: ${DEV_IMAGE}
      imagePullPolicy: IfNotPresent
      command: ["sh", "-c", "trap : TERM INT; sleep infinity & wait"]
      env:
        - name: CRATER_STORAGE_INTERNAL_TOKEN
          valueFrom:
            secretKeyRef:
              name: ${DEV_SECRET}
              key: internal-token
      ports:
        - name: http
          containerPort: 7320
      volumeMounts:
        - name: storage
          mountPath: /crater
  volumes:
    - name: storage
      persistentVolumeClaim:
        claimName: ${QUOTA_PVC}
EOF

kubectl -n "$APP_NAMESPACE" wait --for=condition=Ready "pod/$DEV_POD" --timeout=120s
(
  cd "$ROOT_DIR"
  MSYS_NO_PATHCONV=1 kubectl -n "$APP_NAMESPACE" cp \
    "backend/.gocache/storage-server-linux-${TARGET_ARCH}" \
    "$DEV_POD:/tmp/storage-server" -c quota-agent
)
kubectl -n "$APP_NAMESPACE" exec "$DEV_POD" -c quota-agent -- sh -c \
  'chmod 0700 /tmp/storage-server; CRATER_STORAGE_MODE=quota-agent GIN_MODE=release nohup /tmp/storage-server >/tmp/quota-agent.log 2>&1 &'
sleep 2
kubectl -n "$APP_NAMESPACE" exec "$DEV_POD" -c quota-agent -- sh -c \
  'pid=$(pidof storage-server || true); if [ -z "$pid" ]; then cat /tmp/quota-agent.log; exit 1; fi; kill -0 "$pid"; tail -n 20 /tmp/quota-agent.log'

echo
echo "Development quota-agent is running from the local binary."
echo "Start the local tunnel in another terminal:"
echo "  kubectl -n ${APP_NAMESPACE} port-forward pod/${DEV_POD} 7330:7320"
echo
echo "Cleanup:"
echo "  bash backend/hack/run-cephfs-quota-agent-dev.sh --cleanup"
