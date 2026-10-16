// Copyright 2026 The Crater Project Team, RAIDS-Lab

package jobsession

import (
	"bytes"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSHAndExecShareAccountShellAndContainerEnvironment(t *testing.T) {
	shell := filepath.Join(t.TempDir(), "preferred-shell")
	writeScript(t, shell, "#!/bin/sh\nprintf '%s|%s|%s|%s|%s' \"$HOME\" \"$USER\" \"$SHELL\" \"$CRATER_VALUE\" \"$LD_LIBRARY_PATH\"\n")
	run, home, env := localSession(t, shell)
	// Model the SSH-authenticated account without changing the host's real uid.
	writeScript(t, filepath.Join(home, "bin", "id"), "#!/bin/sh\nif [ \"$*\" = -un ]; then echo alice; else exec /usr/bin/id \"$@\"; fi\n")
	prepared, err := Command(t.Context(), run, nil, UserEnvironment, false)
	if err != nil {
		t.Fatal(err)
	}
	var execOutput bytes.Buffer
	if err = runPrepared(t.Context(), prepared, env, home, &execOutput, io.Discard); err != nil {
		t.Fatal(err)
	}
	// Exercise the production launcher serializer, including quotes/newlines.
	var launcher bytes.Buffer
	argv := []string{"/bin/sh", "-c", writeSSHLauncher + `crater_write_launcher "$(export -p)" "$1"`, "fixture", shellCommand(command("shell"))}
	if err = run(t.Context(), argv, &launcher, io.Discard); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "ssh-session")
	writeScript(t, path, launcher.String())
	// An SSH login starts with a different base environment; the saved task
	// variables must still match exec after the shared launcher runs.
	clean := []string{"PATH=" + filepath.Join(home, "bin") + ":/usr/bin:/bin", "HOME=/incorrect", "USER=incorrect"}
	var sshOutput, stderr bytes.Buffer
	if err = environmentRunner(clean, home)(t.Context(), []string{"/bin/sh", path}, &sshOutput, &stderr); err != nil {
		t.Fatalf("SSH error=%v stderr=%q", err, stderr.String())
	}
	if sshOutput.String() != execOutput.String() {
		t.Fatalf("SSH=%q, want exec=%q", sshOutput.String(), execOutput.String())
	}
	if !strings.Contains(sshOutput.String(), "a'b\nsecond=line|/vendor/lib") {
		t.Fatalf("task environment was lost: %q", sshOutput.String())
	}
}

func TestCommandsPreserveRemoteExitStatus(t *testing.T) {
	run, home, env := localSession(t, "/bin/sh")
	prepared, err := Command(t.Context(), run, []string{"/bin/sh", "-c", "exit 23"}, ContainerEnvironment, false)
	if err != nil {
		t.Fatal(err)
	}
	err = runPrepared(t.Context(), prepared, env, home, io.Discard, io.Discard)
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 23 {
		t.Fatalf("error=%v, want exit 23", err)
	}
}

func TestExplicitInvalidEnvironmentMode(t *testing.T) {
	_, err := Command(t.Context(), nil, []string{"true"}, "invalid", false)
	if err == nil {
		t.Fatal("accepted an invalid environment mode")
	}
}
