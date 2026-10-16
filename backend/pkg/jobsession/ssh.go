// Copyright 2026 The Crater Project Team, RAIDS-Lab

package jobsession

import "strings"

// SSHCommand is the fixed remote command returned by the SSH API. OpenSSH runs
// the installed shared session as the authenticated account, with a terminal.
const SSHCommand = "/bin/sh /run/crater/session.sh"

// InstallSSHCommand atomically installs the shared launcher and returns the task
// username on stdout. The private container snapshot is only embedded in this
// user-owned launcher, never returned by the API or written to server logs.
func InstallSSHCommand() []string {
	args := []string{"/bin/sh", "-c", installSSH, "crater-ssh-install", shellCommand(command("shell"))}
	return append(args, command("identity")...)
}

func shellCommand(argv []string) string {
	quoted := make([]string, len(argv))
	for i, arg := range argv {
		quoted[i] = quote(arg)
	}
	return strings.Join(quoted, " ")
}

const writeSSHLauncher = `
crater_quote() { printf "'"; printf '%s' "$1" | sed "s/'/'\\\\''/g"; printf "'"; }
crater_write_launcher() {
    printf '#!/bin/sh\neval '
    crater_quote "$1"
    printf '\nexport NB_USER="$(id -un)"\nexec %s\n' "$2"
}
`

const installSSH = writeSSHLauncher + `set -eu
[ "$(id -u)" = 0 ] || { echo 'crater: SSH setup requires the container root user' >&2; exit 126; }
if [ -f /run/crater/environment.sh ] && [ ! -L /run/crater/environment.sh ]; then
    . /run/crater/environment.sh
fi
crater_snapshot=$(export -p)
[ ! -L /run/crater ] || { echo 'crater: session directory must not be a symlink' >&2; exit 126; }
mkdir -p /run/crater
chown 0:0 /run/crater
chmod 755 /run/crater
umask 077
crater_tmp=$(mktemp /run/crater/.session.XXXXXX)
trap 'rm -f "$crater_tmp"' EXIT HUP INT TERM
crater_write_launcher "$crater_snapshot" "$1" > "$crater_tmp"
shift
# Only the task account may read the environment snapshot in this launcher.
crater_uid=$(id -u "${NB_USER:-root}")
chown "$crater_uid" "$crater_tmp"
chmod 700 "$crater_tmp"
mv -f "$crater_tmp" /run/crater/session.sh
trap - EXIT HUP INT TERM
exec "$@"
`
