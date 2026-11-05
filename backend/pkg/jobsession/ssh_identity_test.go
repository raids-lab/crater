// Copyright 2026 The Crater Project Team, RAIDS-Lab

package jobsession

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSHAuthenticatedAccountSurvivesStartupSnapshot(t *testing.T) {
	_, dir, env := localSession(t, "/bin/sh")
	bin := filepath.Join(dir, "bin")
	rootHome, aliceHome := filepath.Join(dir, "root"), filepath.Join(dir, "alice")
	for _, home := range []string{rootHome, aliceHome} {
		if err := os.Mkdir(home, 0o700); err != nil {
			t.Fatal(err)
		}
		writeScript(t, filepath.Join(home, "shell"), `#!/bin/sh
printf '%s|%s|%s|%s|%s|%s|%s' "$HOME" "$USER" "$LOGNAME" "$SHELL" "$(id -u)" "$NB_USER" "$STARTUP_VALUE"
`)
	}
	// Model container UIDs without creating or switching any host accounts.
	writeScript(t, filepath.Join(bin, "id"), `#!/bin/sh
case "$*" in
    -un) printf '%s\n' "$SSH_ACCOUNT";;
    -u) if [ "$SSH_ACCOUNT" = root ]; then echo 0; else echo 1000; fi;;
    *) exit 1;;
esac
`)
	writeScript(t, filepath.Join(bin, "runuser"), "#!/bin/sh\necho 'unexpected user switch' >&2\nexit 126\n")
	writeScript(t, filepath.Join(bin, "getent"), fmt.Sprintf(`#!/bin/sh
case "$*" in
    'passwd root') printf 'root:x:0:0::%%s:%%s\n' %s %s;;
    'passwd alice') printf 'alice:x:1000:1000::%%s:%%s\n' %s %s;;
    *) exit 2;;
esac
`, quote(rootHome), quote(filepath.Join(rootHome, "shell")), quote(aliceHome), quote(filepath.Join(aliceHome, "shell"))))
	snapshotDir := filepath.Join(dir, "snapshot")
	if err := os.Mkdir(snapshotDir, 0o700); err != nil {
		t.Fatal(err)
	}
	snapshot := "export NB_USER=alice\nexport STARTUP_VALUE=from-root-snapshot\nexport HOME=/incorrect\nexport USER=incorrect\nexport SHELL=/incorrect\n"
	writeScript(t, filepath.Join(snapshotDir, "environment.sh"), snapshot)
	env = append(env, "STARTUP_VALUE=restored")
	var launcher bytes.Buffer
	// Use the remote command passed by the production installer to its serializer.
	argv := []string{"/bin/sh", "-c", writeSSHLauncher + `crater_write_launcher "$(export -p)" "$1"`, "fixture", InstallSSHCommand()[4]}
	if err := environmentRunner(env, dir)(t.Context(), argv, &launcher, io.Discard); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ssh-session")
	writeScript(t, path, strings.ReplaceAll(launcher.String(), "/run/crater", snapshotDir))
	for _, tt := range []struct{ account, home, uid, startup string }{
		{"root", rootHome, "0", "from-root-snapshot"},
		{"alice", aliceHome, "1000", "restored"},
	} {
		t.Run(tt.account, func(t *testing.T) {
			clean := []string{"PATH=" + bin + ":/usr/bin:/bin", "SSH_ACCOUNT=" + tt.account}
			var stdout, stderr bytes.Buffer
			if err := environmentRunner(clean, dir)(t.Context(), []string{"/bin/sh", path}, &stdout, &stderr); err != nil {
				t.Fatalf("SSH error=%v stderr=%q", err, stderr.String())
			}
			want := strings.Join([]string{tt.home, tt.account, tt.account, filepath.Join(tt.home, "shell"), tt.uid, tt.account, tt.startup}, "|")
			if stdout.String() != want {
				t.Fatalf("session=%q, want %q", stdout.String(), want)
			}
		})
	}
}
