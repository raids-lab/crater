#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   bash storage-index-experiment-toolbox.sh /mnt/mycephfs/.../user-space-root
#
# Notes:
# - ROOT must be the actual CephFS path visible inside rook-ceph-tools.
# - Recursive stats come from CephFS virtual xattrs:
#   ceph.dir.rfiles / ceph.dir.rsubdirs / ceph.dir.rentries / ceph.dir.rbytes
# - find is only used to enumerate top-level directory names for applying the
#   same allow/block filter as the service implementation.

ROOT="${1:-}"
if [[ -z "$ROOT" ]]; then
  echo "Usage: bash storage-index-experiment-toolbox.sh <actual_cephfs_user_root>" >&2
  exit 1
fi

if [[ ! -d "$ROOT" ]]; then
  echo "error: ROOT is not a directory: $ROOT" >&2
  exit 1
fi

SKIP_RE='^(conda|\.conda|miniconda3|anaconda3|mambaforge|\.mamba|micromamba|venv|\.venv|\.git|\.svn|\.hg|\.idea|\.vscode|\.vscode-server|\.ipynb_checkpoints|\.npm|\.yarn|\.pnpm-store|\.cargo|\.m2|\.gradle|\.pytest_cache|\.mypy_cache|__pycache__)$'

get_dir_stat() {
  local attr="$1"
  local target="$2"
  getfattr --only-values -n "$attr" "$target" 2>/dev/null || echo 0
}

TOTAL_BYTES="$(get_dir_stat ceph.dir.rbytes "$ROOT")"
TOTAL_FILES="$(get_dir_stat ceph.dir.rfiles "$ROOT")"
TOTAL_DIRS="$(get_dir_stat ceph.dir.rsubdirs "$ROOT")"
TOTAL_ENTRIES="$(get_dir_stat ceph.dir.rentries "$ROOT")"

mapfile -t TOP_LEVEL_DIRS < <(find "$ROOT" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort)

SELECTED=()
SKIPPED=()
for name in "${TOP_LEVEL_DIRS[@]}"; do
  if [[ "$name" =~ $SKIP_RE ]]; then
    SKIPPED+=("$name")
  else
    SELECTED+=("$name")
  fi
done

FILTERED_BYTES=0
FILTERED_FILES=0
FILTERED_DIRS=0
FILTERED_ENTRIES=0

for name in "${SELECTED[@]}"; do
  subtree="$ROOT/$name"
  bytes="$(get_dir_stat ceph.dir.rbytes "$subtree")"
  files="$(get_dir_stat ceph.dir.rfiles "$subtree")"
  subdirs="$(get_dir_stat ceph.dir.rsubdirs "$subtree")"
  entries="$(get_dir_stat ceph.dir.rentries "$subtree")"
  FILTERED_BYTES=$((FILTERED_BYTES + bytes))
  FILTERED_FILES=$((FILTERED_FILES + files))
  # rsubdirs excludes the subtree root itself, while the filtered scan result
  # counts each selected top-level directory as an indexed directory node.
  FILTERED_DIRS=$((FILTERED_DIRS + subdirs + 1))
  FILTERED_ENTRIES=$((FILTERED_ENTRIES + entries + 1))
done

echo "workspace_root=$ROOT"
echo "total_bytes=$TOTAL_BYTES"
echo "total_file_count=$TOTAL_FILES"
echo "total_directory_count=$TOTAL_DIRS"
echo "total_entry_count=$TOTAL_ENTRIES"
echo "filtered_bytes=$FILTERED_BYTES"
echo "filtered_file_count=$FILTERED_FILES"
echo "filtered_directory_count=$FILTERED_DIRS"
echo "filtered_entry_count=$FILTERED_ENTRIES"
echo "top_level_candidate_dir_count=${#TOP_LEVEL_DIRS[@]}"
echo "selected_top_level_dir_count=${#SELECTED[@]}"
echo "skipped_top_level_dir_count=${#SKIPPED[@]}"
echo "selected_top_level_dir_names=$(IFS=,; echo "${SELECTED[*]-}")"
echo "skipped_top_level_dir_names=$(IFS=,; echo "${SKIPPED[*]-}")"
