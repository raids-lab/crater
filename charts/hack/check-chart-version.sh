#!/usr/bin/env bash

set -euo pipefail

BASE_REF=""
STAGED=false

usage() {
    echo "Usage: bash hack/check-chart-version.sh [--base <ref>] [--staged]"
    echo ""
    echo "Checks whether release-impacting Helm chart changes bumped"
    echo "charts/crater/Chart.yaml version and appVersion together."
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --base)
            BASE_REF="${2:-}"
            if [ -z "$BASE_REF" ]; then
                usage
                exit 2
            fi
            shift 2
            ;;
        --staged)
            STAGED=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown argument: $1"
            usage
            exit 2
            ;;
    esac
done

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [ -z "$REPO_ROOT" ]; then
    echo "Unable to find the git repository root." >&2
    exit 1
fi
cd "$REPO_ROOT"

find_base_ref() {
    if [ -n "$BASE_REF" ]; then
        if git rev-parse --verify "$BASE_REF^{commit}" >/dev/null 2>&1; then
            echo "$BASE_REF"
            return 0
        fi
        echo "Base ref not found: $BASE_REF" >&2
        return 1
    fi

    for candidate in origin/main upstream/main main; do
        if git rev-parse --verify "$candidate^{commit}" >/dev/null 2>&1; then
            echo "$candidate"
            return 0
        fi
    done

    echo "Unable to find a base ref. Fetch main or pass --base <ref>." >&2
    return 1
}

changed_files() {
    if [ "$STAGED" = true ]; then
        git diff --cached --name-only -- charts
    else
        local base_ref="$1"
        git diff --name-only "$base_ref"...HEAD -- charts
    fi
}

release_changed_files() {
    grep -E '^charts/crater/(Chart\.yaml|values\.yaml|templates/|charts/|Chart\.lock)$' || true
}

read_chart_field() {
    local field="$1"
    awk -v key="$field" '$1 == key ":" { print $2; exit }' | tr -d '"' | tr -d "'"
}

read_current_chart_field() {
    local field="$1"
    if [ "$STAGED" = true ]; then
        git show ":charts/crater/Chart.yaml" | read_chart_field "$field"
    else
        read_chart_field "$field" < charts/crater/Chart.yaml
    fi
}

read_base_chart_field() {
    local base_ref="$1"
    local field="$2"
    git show "$base_ref:charts/crater/Chart.yaml" | read_chart_field "$field"
}

semver_core() {
    local version="$1"
    version="${version#v}"
    version="${version%%-*}"
    version="${version%%+*}"
    echo "$version"
}

semver_gt() {
    local current base
    current="$(semver_core "$1")"
    base="$(semver_core "$2")"

    IFS=. read -r current_major current_minor current_patch <<EOF
$current
EOF
    IFS=. read -r base_major base_minor base_patch <<EOF
$base
EOF

    for part in current_major current_minor current_patch base_major base_minor base_patch; do
        case "${!part:-}" in
            ''|*[!0-9]*)
                echo "Invalid semantic version core: current=$1 base=$2" >&2
                return 2
                ;;
        esac
    done

    if [ "$current_major" -gt "$base_major" ]; then return 0; fi
    if [ "$current_major" -lt "$base_major" ]; then return 1; fi
    if [ "$current_minor" -gt "$base_minor" ]; then return 0; fi
    if [ "$current_minor" -lt "$base_minor" ]; then return 1; fi
    if [ "$current_patch" -gt "$base_patch" ]; then return 0; fi
    return 1
}

if [ "$STAGED" != true ]; then
    BASE_REF="$(find_base_ref)"
fi

CHANGED_FILES="$(changed_files "${BASE_REF:-}")"
RELEASE_CHANGED_FILES="$(printf '%s\n' "$CHANGED_FILES" | release_changed_files)"

if [ -z "$RELEASE_CHANGED_FILES" ]; then
    echo "No release-impacting Helm chart files changed; skipping chart version check."
    exit 0
fi

echo "Release-impacting Helm chart files changed:"
printf '%s\n' "$RELEASE_CHANGED_FILES" | sed 's/^/  - /'

if [ "$STAGED" = true ]; then
    BASE_REF="$(find_base_ref)"
fi

CURRENT_VERSION="$(read_current_chart_field version)"
CURRENT_APP_VERSION="$(read_current_chart_field appVersion)"
BASE_VERSION="$(read_base_chart_field "$BASE_REF" version)"

echo "Base chart version ($BASE_REF): $BASE_VERSION"
echo "Current chart version: $CURRENT_VERSION"
echo "Current appVersion: $CURRENT_APP_VERSION"

if [ -z "$CURRENT_VERSION" ] || [ -z "$CURRENT_APP_VERSION" ] || [ -z "$BASE_VERSION" ]; then
    echo "Unable to read version/appVersion from charts/crater/Chart.yaml."
    exit 1
fi

if [ "$CURRENT_VERSION" != "$CURRENT_APP_VERSION" ]; then
    echo "charts/crater/Chart.yaml version and appVersion must be identical."
    exit 1
fi

if semver_gt "$CURRENT_VERSION" "$BASE_VERSION"; then
    echo "Chart version check passed: $BASE_VERSION -> $CURRENT_VERSION"
    exit 0
fi

echo "Chart version must be incremented for release-impacting chart changes."
echo "Update both version and appVersion in charts/crater/Chart.yaml to the same higher semver value."
exit 1
