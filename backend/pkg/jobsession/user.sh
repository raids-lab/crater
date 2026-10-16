#!/bin/sh
# Copyright 2026 The Crater Project Team, RAIDS-Lab
# This stage runs as the target user; environment values are shell-quoted data.
set -eu
eval "$1"
shift
crater_name=$1 crater_home=$2 crater_shell=$3 crater_workdir=$4 crater_mode=$5
shift 5
export HOME="$crater_home" USER="$crater_name" LOGNAME="$crater_name" SHELL="$crater_shell"
[ -d "$HOME" ] && [ -x "$HOME" ] || {
    echo 'crater: task HOME is not accessible' >&2; exit 126;
}
case "$crater_mode" in
    identity) printf '%s' "$crater_name"; exit 0;;
    shell)
        cd "$HOME" || { echo 'crater: task HOME is not accessible' >&2; exit 126; }
        [ -x "$SHELL" ] || { echo 'crater: task login shell is not executable' >&2; exit 126; }
        exec "$SHELL" -il;;
    probe)
        cd "$HOME" || { echo 'crater: task HOME is not accessible' >&2; exit 126; }
        [ -x "$SHELL" ] || { echo 'crater: task login shell is not executable' >&2; exit 126; }
        printf '\036%s\037' "$2"
        env -0
        printf '\036%s\037' "$2"
        exec "$SHELL" -lic "$1";;
    terminal-command)
        cd "$crater_workdir" 2>/dev/null || cd "$HOME"
        exec "$SHELL" -lic "$1";;
    command)
        crater_env_bytes=$1
        shift
        if [ "$crater_env_bytes" -gt 0 ]; then
            # Read only the private environment prefix. Remaining stdin belongs
            # to the user's command; values never travel in Kubernetes URLs.
            crater_exports=$(dd bs=1 count="$crater_env_bytes" 2>/dev/null; printf '.')
            crater_received=$(printf '%s' "$crater_exports" | wc -c)
            [ "$crater_received" -eq "$((crater_env_bytes + 1))" ] || {
                echo 'crater: user environment input was truncated' >&2; exit 126;
            }
            eval "${crater_exports%.}"
            unset crater_exports
        fi
        # A user's exported environment must not change the resolved identity.
        export HOME="$crater_home" USER="$crater_name" LOGNAME="$crater_name" SHELL="$crater_shell"
        cd "$crater_workdir" 2>/dev/null || cd "$HOME" || {
            echo 'crater: neither container working directory nor task HOME is accessible' >&2; exit 126;
        }
        exec "$@";;
esac
echo 'crater: unknown session mode' >&2
exit 126
