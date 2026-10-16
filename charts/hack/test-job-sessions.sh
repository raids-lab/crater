#!/usr/bin/env bash
# Copyright 2026 The Crater Project Team, RAIDS-Lab
set -euo pipefail

chart_dir="$(cd "$(dirname "$0")/../crater" && pwd)"
scratch="$(mktemp -d "${TMPDIR:-/tmp}/crater-job-sessions.XXXXXX")"
trap 'rm -rf "$scratch"' EXIT

for kind in custom unified; do
    template="job/custom-start-configmap.yaml"
    key="start.sh"
    if [ "$kind" = unified ]; then
        template="job/unified-start-configmap.yaml"
        key="unified-start.sh"
    fi
    helm template crater "$chart_dir" --show-only "templates/$template" > "$scratch/$kind.yaml"
    awk -v key="$key" '
        $0 == "  " key ": |" { copying=1; next }
        copying && /^    / { sub(/^    /, ""); print }
    ' "$scratch/$kind.yaml" > "$scratch/$kind.sh"
    test -s "$scratch/$kind.sh"
    bash -n "$scratch/$kind.sh"
    # Existing account preferences take precedence over the image's root SHELL.
    mkdir -p "$scratch/bin"
    printf '#!/bin/sh\nexit 0\n' > "$scratch/preferred-shell"
    chmod 700 "$scratch/preferred-shell"
    printf '#!/bin/sh\nprintf "%%s\\n" "alice:x:1234:5678::%s/custom-home:%s/preferred-shell"\n' \
        "$scratch" "$scratch" > "$scratch/bin/getent"
    chmod 700 "$scratch/bin/getent"
    sed -n '/^detect_shell() {/,/^}/p' "$scratch/$kind.sh" > "$scratch/detect-$kind.sh"
    printf '\ndetect_shell\n' >> "$scratch/detect-$kind.sh"
    actual="$(PATH="$scratch/bin:$PATH" NB_USER=alice SHELL=/bin/sh bash "$scratch/detect-$kind.sh")"
    test "$actual" = "$scratch/preferred-shell"
    # Execute only the snapshot hook; never provision users or alter host paths.
    awk '
        /# Preserve generated toolchain/ { copying=1 }
        copying { print }
        /unset -f crater_save_session_environment/ { exit }
    ' "$scratch/$kind.sh" > "$scratch/$kind-hook.sh"
    test -s "$scratch/$kind-hook.sh"
    sed "s|/run/crater|$scratch/$kind-session|g" "$scratch/$kind-hook.sh" > "$scratch/$kind-test.sh"
    expected=$'single\x27quote\n$(must-not-execute)\nkey=value=more'
    CRATER_TEST_VALUE="$expected" LD_LIBRARY_PATH=/opt/ascend/lib:/vendor/lib \
        bash "$scratch/$kind-test.sh"
    snapshot="$scratch/$kind-session/environment.sh"
    actual="$(sh -c '. "$1"; printf "%s" "$CRATER_TEST_VALUE"' fixture "$snapshot")"
    test "$actual" = "$expected"
    actual="$(sh -c '. "$1"; printf "%s" "$LD_LIBRARY_PATH"' fixture "$snapshot")"
    test "$actual" = /opt/ascend/lib:/vendor/lib
    mode="$(LC_ALL=C ls -ld "$snapshot" | cut -c2-10)"
    test "$mode" = rw-------
done

mkdir "$scratch/link-target"
ln -s "$scratch/link-target" "$scratch/link"
sed "s|/run/crater|$scratch/link|g" "$scratch/custom-hook.sh" > "$scratch/link-test.sh"
if ! bash "$scratch/link-test.sh" > "$scratch/link-output" 2>&1; then
    echo "snapshot failure stopped startup" >&2
    exit 1
fi
if [ -e "$scratch/link-target/environment.sh" ] || ! grep -q 'must not be a symlink' "$scratch/link-output"; then
    echo "snapshot hook wrote through a symlink session directory" >&2
    exit 1
fi
bash "$chart_dir/../hack/test-job-startup-compatibility.sh" "$scratch/custom.sh" "$scratch/unified.sh"
echo "Job startup scripts: render, syntax, private environment snapshots and symlink checks passed."
