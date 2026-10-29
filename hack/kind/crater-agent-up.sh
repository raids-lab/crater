#!/usr/bin/env bash

# Copyright 2026 The Crater Project Team, RAIDS-Lab
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.

set -euo pipefail

export PATH="${HOME}/.local/bin:${PATH}"

readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly CLUSTER_NAME="crater-agent-dev"
readonly CONTEXT="kind-${CLUSTER_NAME}"
readonly NAMESPACE="crater-system"
readonly CREDENTIALS_SECRET="crater-agent-kind-credentials"
readonly AGENT_SECRET="crater-agent-internal"

ADMIN_PASSWORD=""
POSTGRES_PASSWORD=""

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'Required command not found: %s\n' "$1" >&2
    exit 1
  fi
}

detect_go_arch() {
  case "$(uname -m)" in
    arm64 | aarch64) printf 'arm64\n' ;;
    x86_64 | amd64) printf 'amd64\n' ;;
    *)
      printf 'Unsupported host architecture: %s\n' "$(uname -m)" >&2
      exit 1
      ;;
  esac
}

ensure_cluster() {
  if ! kind get clusters 2>/dev/null | grep -qx "$CLUSTER_NAME"; then
    kind create cluster \
      --name "$CLUSTER_NAME" \
      --config "$ROOT_DIR/hack/kind/crater-agent-cluster.yaml"
  fi
  kubectl --context "$CONTEXT" cluster-info >/dev/null
}

load_or_create_credentials() {
  if ! kubectl --context "$CONTEXT" get secret "$CREDENTIALS_SECRET" \
    --namespace "$NAMESPACE" >/dev/null 2>&1; then
    ADMIN_PASSWORD="$(openssl rand -hex 24)"
    POSTGRES_PASSWORD="$(openssl rand -hex 24)"
    kubectl --context "$CONTEXT" create secret generic "$CREDENTIALS_SECRET" \
      --namespace "$NAMESPACE" \
      --from-literal="admin-password=$ADMIN_PASSWORD" \
      --from-literal="postgres-password=$POSTGRES_PASSWORD"
    return
  fi

  ADMIN_PASSWORD="$(
    kubectl --context "$CONTEXT" get secret "$CREDENTIALS_SECRET" \
      --namespace "$NAMESPACE" \
      --output=jsonpath='{.data.admin-password}' | openssl base64 -d -A
  )"
  POSTGRES_PASSWORD="$(
    kubectl --context "$CONTEXT" get secret "$CREDENTIALS_SECRET" \
      --namespace "$NAMESPACE" \
      --output=jsonpath='{.data.postgres-password}' | openssl base64 -d -A
  )"
}

ensure_agent_secret() {
  if kubectl --context "$CONTEXT" get secret "$AGENT_SECRET" \
    --namespace "$NAMESPACE" >/dev/null 2>&1; then
    return
  fi

  kubectl --context "$CONTEXT" create secret generic "$AGENT_SECRET" \
    --namespace "$NAMESPACE" \
    --from-literal="internal-token=$(openssl rand -hex 32)" \
    --from-literal="context-signing-key=$(openssl rand -hex 32)" \
    --from-literal="encryption-key=$(openssl rand -hex 16)"
}

install_ingress() {
  kubectl --context "$CONTEXT" apply \
    -f https://kind.sigs.k8s.io/examples/ingress/deploy-ingress-nginx.yaml
  kubectl --context "$CONTEXT" wait \
    --namespace ingress-nginx \
    --for=condition=ready pod \
    --selector=app.kubernetes.io/component=controller \
    --timeout=180s
}

install_dependencies() {
  kubectl --context "$CONTEXT" create namespace "$NAMESPACE" \
    --dry-run=client -o yaml | kubectl --context "$CONTEXT" apply -f -
  load_or_create_credentials
  ensure_agent_secret
  kubectl --context "$CONTEXT" create namespace crater \
    --dry-run=client -o yaml | kubectl --context "$CONTEXT" apply -f -
  kubectl --context "$CONTEXT" create namespace crater-workspace \
    --dry-run=client -o yaml | kubectl --context "$CONTEXT" apply -f -
  kubectl --context "$CONTEXT" create namespace crater-images \
    --dry-run=client -o yaml | kubectl --context "$CONTEXT" apply -f -
  for managed_namespace in crater-workspace crater-images; do
    kubectl --context "$CONTEXT" label namespace "$managed_namespace" \
      app.kubernetes.io/managed-by=Helm --overwrite
    kubectl --context "$CONTEXT" annotate namespace "$managed_namespace" \
      meta.helm.sh/release-name=crater \
      meta.helm.sh/release-namespace="$NAMESPACE" \
      --overwrite
  done

  local current_storage_class
  current_storage_class="$(
    kubectl --context "$CONTEXT" get pvc crater-rw-storage \
      --namespace crater-workspace \
      --output=jsonpath='{.spec.storageClassName}' 2>/dev/null || true
  )"
  if [[ -n "$current_storage_class" && "$current_storage_class" != "crater-kind-hostpath" ]]; then
    kubectl --context "$CONTEXT" delete deployment webdav-deployment \
      --namespace crater-workspace --ignore-not-found --wait=true
    kubectl --context "$CONTEXT" delete pvc crater-rw-storage \
      --namespace crater-workspace --ignore-not-found --wait=true
  fi
  kubectl --context "$CONTEXT" apply \
    -f "$ROOT_DIR/hack/kind/storage.yaml"

  docker pull postgres:16.4-alpine
  kind load docker-image --name "$CLUSTER_NAME" postgres:16.4-alpine
  kubectl --context "$CONTEXT" apply \
    -f "$ROOT_DIR/hack/kind/postgresql.yaml"
  kubectl --context "$CONTEXT" rollout status \
    statefulset/crater-postgresql \
    --namespace "$NAMESPACE" \
    --timeout=5m

  kubectl --context "$CONTEXT" apply \
    -f "$ROOT_DIR/hack/kind/volcano-crds.yaml"
  kubectl --context "$CONTEXT" wait \
    --for=condition=established \
    crd/jobs.batch.volcano.sh \
    crd/podgroups.scheduling.volcano.sh \
    crd/queues.scheduling.volcano.sh \
    --timeout=2m
}

build_agent_image() {
  pnpm --dir "$ROOT_DIR/crater-agent" install --frozen-lockfile
  pnpm --dir "$ROOT_DIR/crater-agent" build
  docker build --tag crater-agent:dev "$ROOT_DIR/crater-agent"
}

build_backend_image() {
  local go_arch="$1"
  local target_platform="linux/${go_arch}"
  local binary_dir="$ROOT_DIR/backend/bin/kind/bin-linux_${go_arch}"

  mkdir -p "$binary_dir"
  (
    cd "$ROOT_DIR/backend"
    CGO_ENABLED=0 GOOS=linux GOARCH="$go_arch" go build \
      -o "$binary_dir/migrate" cmd/gorm-gen/models/migrate.go
    CGO_ENABLED=0 GOOS=linux GOARCH="$go_arch" go build \
      -o "$binary_dir/controller" cmd/crater/main.go
    CGO_ENABLED=0 GOOS=linux GOARCH="$go_arch" go build \
      -o "$binary_dir/model-metadata-refresh" cmd/model-metadata-refresh/main.go
  )
  docker buildx build --load \
    --platform "$target_platform" \
    --build-arg TARGETPLATFORM="$target_platform" \
    --build-arg TARGETARCH="$go_arch" \
    --build-arg BIN_DIR=bin/kind \
    --tag crater-backend:dev \
    "$ROOT_DIR/backend"
}

build_frontend_image() {
  pnpm --dir "$ROOT_DIR/frontend" install --frozen-lockfile
  pnpm --dir "$ROOT_DIR/frontend" build
  docker build --tag crater-frontend:dev "$ROOT_DIR/frontend"
}

load_images() {
  kind load docker-image --name "$CLUSTER_NAME" \
    crater-agent:dev crater-backend:dev crater-frontend:dev
}

install_crater() {
  helm upgrade --install crater "$ROOT_DIR/charts/crater" \
    --kube-context "$CONTEXT" \
    --namespace "$NAMESPACE" \
    --values "$ROOT_DIR/hack/kind/crater-agent-values.yaml" \
    --set-string "agent.existingSecretName=$AGENT_SECRET" \
    --set-string "firstUser.password=$ADMIN_PASSWORD" \
    --set-string "backendConfig.postgres.password=$POSTGRES_PASSWORD" \
    --wait --timeout 15m
}

main() {
  for command_name in docker kind kubectl helm go pnpm openssl; do
    require_command "$command_name"
  done
  docker info >/dev/null

  ensure_cluster
  install_ingress
  install_dependencies

  local go_arch
  go_arch="$(detect_go_arch)"
  build_agent_image
  build_backend_image "$go_arch"
  build_frontend_image
  load_images
  install_crater

  CRATER_AGENT_SMOKE_ADMIN_PASSWORD="$ADMIN_PASSWORD" \
    "$ROOT_DIR/hack/kind/crater-agent-smoke.sh"
  printf '\nCrater is available at http://crater.localhost:8080\n'
  printf 'Local administrator: crater-admin / %s\n' "$ADMIN_PASSWORD"
}

main "$@"
