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

if [[ ! -f "$RUNTIME_ENV" ]]; then
  printf 'Local deployment runtime configuration is missing. Run make agent-local-up first.\n' >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$RUNTIME_ENV"
set +a

readonly HTTP_PORT="${CRATER_LOCAL_HTTP_PORT:-5180}"
readonly BACKEND_PORT="${CRATER_LOCAL_BACKEND_PORT:-8098}"

for _ in $(seq 1 60); do
  if curl --fail --silent --show-error "http://127.0.0.1:${BACKEND_PORT}/healthz" >/dev/null &&
    curl --fail --silent --show-error "http://127.0.0.1:${HTTP_PORT}/" >/dev/null; then
    break
  fi
  sleep 2
done

curl --fail --silent --show-error "http://127.0.0.1:${BACKEND_PORT}/healthz" >/dev/null
curl --fail --silent --show-error "http://127.0.0.1:${HTTP_PORT}/" >/dev/null
docker compose --env-file "$RUNTIME_ENV" --file "$COMPOSE_FILE" exec --no-TTY agent \
  node -e "fetch('http://127.0.0.1:8000/healthz').then(r=>{if(!r.ok)process.exit(1)}).catch(()=>process.exit(1))"

docker compose --env-file "$RUNTIME_ENV" --file "$COMPOSE_FILE" ps
printf 'Local Crater-Agent deployment smoke test passed.\n'
