#!/bin/sh
# Copyright 2026 The Crater Project Team, RAIDS-Lab
# Resolve the account before entering the shared, unprivileged session stage.
set -eu
# Only the root-managed startup snapshot is read before dropping privileges.
if [ "$(id -u)" = 0 ] && [ ! -L /run/crater ] && [ -f /run/crater/environment.sh ] && [ ! -L /run/crater/environment.sh ]; then
    . /run/crater/environment.sh
fi
crater_user_script=__USER_SCRIPT__
crater_target=${NB_USER:-$(id -un)}
if command -v getent >/dev/null 2>&1; then
    crater_entry=$(getent passwd "$crater_target") || {
        echo 'crater: task user is missing from the container account database' >&2; exit 126;
    }
else
    crater_entry=$(awk -F: -v name="$crater_target" '$1 == name { print; exit }' /etc/passwd)
fi
IFS=: read -r crater_name crater_password crater_uid crater_gid crater_gecos crater_home crater_shell <<EOF
$crater_entry
EOF
case "$crater_uid" in ''|*[!0-9]*) echo 'crater: task user has no valid uid' >&2; exit 126;; esac
case "$crater_home" in /*) ;; *) echo 'crater: task user has no absolute HOME' >&2; exit 126;; esac
if [ -z "$crater_shell" ]; then
    crater_shell=/bin/sh
fi
crater_snapshot=$(export -p)
set -- /bin/sh -c "$crater_user_script" crater-session "$crater_snapshot" \
    "$crater_name" "$crater_home" "$crater_shell" "$PWD" "$@"
if [ "$(id -u)" = "$crater_uid" ]; then
    exec "$@"
fi
if [ "$(id -u)" != 0 ]; then
    echo 'crater: container user cannot switch to the task user' >&2; exit 126
fi
if command -v runuser >/dev/null 2>&1; then
    exec runuser -u "$crater_name" -- "$@"
elif command -v sudo >/dev/null 2>&1; then
    exec sudo -n -H -u "$crater_name" -- "$@"
elif command -v su >/dev/null 2>&1; then
    crater_command='exec'
    for crater_arg do
        crater_quoted=$(printf '%s' "$crater_arg" | sed "s/'/'\\\\''/g"; printf '.')
        crater_quoted=${crater_quoted%.}
        crater_command="$crater_command '$crater_quoted'"
    done
    exec su -m -s /bin/sh "$crater_name" -c "$crater_command"
fi
echo 'crater: switching to the task user requires runuser, sudo, or su' >&2
exit 126
