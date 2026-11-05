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

if [[ -f "$RUNTIME_ENV" ]]; then
  docker compose --env-file "$RUNTIME_ENV" --file "$COMPOSE_FILE" down --remove-orphans
fi
rm -f "$RUNTIME_ENV" "$RUNTIME_DIR/kubeconfig"
rmdir "$RUNTIME_DIR" 2>/dev/null || true
