#!/usr/bin/env bash

# Copyright 2026 The Crater Project Team, RAIDS-Lab
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.

set -euo pipefail

export PATH="${HOME}/.local/bin:${PATH}"

readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly COMPOSE_FILE="$ROOT_DIR/hack/local/compose.yaml"
readonly RUNTIME_DIR="${TMPDIR:-/tmp}/crater-agent-local-${UID}"
readonly RUNTIME_ENV="$RUNTIME_DIR/compose.env"
readonly FLAT_KUBECONFIG="$RUNTIME_DIR/kubeconfig"
readonly RUNTIME_BACKEND_CONFIG="$RUNTIME_DIR/backend-config.yaml"
readonly SECRET_ENV="${CRATER_LOCAL_SECRET_FILE:-${XDG_STATE_HOME:-$HOME/.local/state}/crater/crater-agent-local.env}"

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

prepare_runtime_config() {
  local source_kubeconfig="${KUBECONFIG:-}"
  local backend_config="${CRATER_LOCAL_BACKEND_CONFIG:-$ROOT_DIR/backend/etc/debug-config.yaml}"

  if [[ -z "$source_kubeconfig" || "$source_kubeconfig" == *:* ]]; then
    printf 'Set KUBECONFIG to one kubeconfig file for the external cluster.\n' >&2
    exit 1
  fi
  if [[ "${CRATER_LOCAL_ALLOW_EXTERNAL_CLUSTER:-}" != "true" ]]; then
    printf '%s\n' \
      'This mode starts Crater controllers against the external cluster.' \
      'Stop any conflicting local backend, then set CRATER_LOCAL_ALLOW_EXTERNAL_CLUSTER=true.' >&2
    exit 1
  fi
  if [[ ! -f "$source_kubeconfig" ]]; then
    printf 'Kubeconfig does not exist: %s\n' "$source_kubeconfig" >&2
    exit 1
  fi
  if [[ ! -f "$backend_config" ]]; then
    printf 'Backend configuration does not exist: %s\n' "$backend_config" >&2
    exit 1
  fi

  mkdir -p "$RUNTIME_DIR"
  kubectl --kubeconfig "$source_kubeconfig" config view \
    --raw --flatten --minify >"$FLAT_KUBECONFIG"
  chmod 600 "$FLAT_KUBECONFIG"

  if ! awk '
    BEGIN { updated = 0 }
    updated == 0 && /^port:[[:space:]]*/ { print "port: :8098"; updated = 1; next }
    { print }
    END { if (updated == 0) exit 1 }
  ' "$backend_config" >"$RUNTIME_BACKEND_CONFIG"; then
    printf 'Backend configuration must contain a top-level port field: %s\n' "$backend_config" >&2
    exit 1
  fi
  chmod 600 "$RUNTIME_BACKEND_CONFIG"

  {
    printf 'CRATER_AGENT_INTERNAL_TOKEN=%s\n' "$CRATER_AGENT_INTERNAL_TOKEN"
    printf 'CRATER_AGENT_CONTEXT_SIGNING_KEY=%s\n' "$CRATER_AGENT_CONTEXT_SIGNING_KEY"
    printf 'SYSTEM_SECRET_KEY=%s\n' "$SYSTEM_SECRET_KEY"
    if [[ -n "${SYSTEM_SECRET_KEY_PREVIOUS:-}" ]]; then
      printf 'SYSTEM_SECRET_KEY_PREVIOUS=%s\n' "$SYSTEM_SECRET_KEY_PREVIOUS"
    fi
    printf 'CRATER_LOCAL_BACKEND_CONFIG=%s\n' "$RUNTIME_BACKEND_CONFIG"
    printf 'CRATER_LOCAL_KUBECONFIG=%s\n' "$FLAT_KUBECONFIG"
    printf 'CRATER_LOCAL_HTTP_PORT=%s\n' "${CRATER_LOCAL_HTTP_PORT:-5180}"
    printf 'CRATER_LOCAL_BACKEND_PORT=%s\n' "${CRATER_LOCAL_BACKEND_PORT:-8098}"
  } >"$RUNTIME_ENV"
  chmod 600 "$RUNTIME_ENV"
}

prepare_runtime_secrets() {
  if [[ ! -f "$SECRET_ENV" ]]; then
    mkdir -p "$(dirname "$SECRET_ENV")"
    {
      printf 'CRATER_AGENT_INTERNAL_TOKEN=%s\n' \
        "${CRATER_AGENT_INTERNAL_TOKEN:-$(openssl rand -hex 32)}"
      printf 'CRATER_AGENT_CONTEXT_SIGNING_KEY=%s\n' \
        "${CRATER_AGENT_CONTEXT_SIGNING_KEY:-$(openssl rand -hex 32)}"
      printf 'SYSTEM_SECRET_KEY=%s\n' "${SYSTEM_SECRET_KEY:-$(openssl rand -hex 16)}"
      if [[ -n "${SYSTEM_SECRET_KEY_PREVIOUS:-}" ]]; then
        printf 'SYSTEM_SECRET_KEY_PREVIOUS=%s\n' "$SYSTEM_SECRET_KEY_PREVIOUS"
      fi
    } >"$SECRET_ENV"
  fi
  chmod 600 "$SECRET_ENV"

  set -a
  # shellcheck disable=SC1090
  source "$SECRET_ENV"
  set +a
  : "${CRATER_AGENT_INTERNAL_TOKEN:?missing CRATER_AGENT_INTERNAL_TOKEN in $SECRET_ENV}"
  : "${CRATER_AGENT_CONTEXT_SIGNING_KEY:?missing CRATER_AGENT_CONTEXT_SIGNING_KEY in $SECRET_ENV}"
  : "${SYSTEM_SECRET_KEY:?missing SYSTEM_SECRET_KEY in $SECRET_ENV}"
}

build_agent_image() {
  pnpm --dir "$ROOT_DIR/crater-agent" install --frozen-lockfile
  pnpm --dir "$ROOT_DIR/crater-agent" build
  docker build --tag crater-agent:local "$ROOT_DIR/crater-agent"
}

build_backend_image() {
  local go_arch="$1"
  local target_platform="linux/${go_arch}"
  local binary_dir="$ROOT_DIR/backend/bin/local/bin-linux_${go_arch}"

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
    --build-arg TARGETARCH="$go_arch" \
    --build-arg BIN_DIR=bin/local \
    --tag crater-backend:local \
    "$ROOT_DIR/backend"
}

build_frontend_image() {
  pnpm --dir "$ROOT_DIR/frontend" install --frozen-lockfile
  pnpm --dir "$ROOT_DIR/frontend" build
  docker build --tag crater-frontend:local "$ROOT_DIR/frontend"
}

run_database_migrations() {
  if [[ "${CRATER_LOCAL_RUN_MIGRATIONS:-false}" != "true" ]]; then
    return
  fi
  docker run --rm \
    --entrypoint /migrate \
    --env-file "$RUNTIME_ENV" \
    --volume "$RUNTIME_BACKEND_CONFIG:/etc/config/config.yaml:ro" \
    crater-backend:local
}

main() {
  for command_name in docker kubectl openssl go pnpm; do
    require_command "$command_name"
  done
  docker compose version >/dev/null
  docker buildx version >/dev/null
  docker info >/dev/null

  prepare_runtime_secrets
  prepare_runtime_config

  local go_arch
  go_arch="$(detect_go_arch)"
  build_agent_image
  build_backend_image "$go_arch"
  build_frontend_image
  run_database_migrations

  docker compose --env-file "$RUNTIME_ENV" --file "$COMPOSE_FILE" \
    up --detach --remove-orphans
  "$ROOT_DIR/hack/local/crater-agent-smoke.sh"

  printf '\nCrater local deployment: http://localhost:%s\n' "${CRATER_LOCAL_HTTP_PORT:-5180}"
  printf 'Backend health endpoint: http://localhost:%s/healthz\n' \
    "${CRATER_LOCAL_BACKEND_PORT:-8098}"
  printf 'Persistent local Agent secrets: %s\n' "$SECRET_ENV"
  printf 'The backend is using cluster context: '
  kubectl --kubeconfig "$FLAT_KUBECONFIG" config current-context
}

main "$@"
