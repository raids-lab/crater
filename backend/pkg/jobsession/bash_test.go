// Copyright 2026 The Crater Project Team, RAIDS-Lab

package jobsession

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBashReadOnlyOptionsDoNotBreakExec(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("Bash is required for the read-only shell option regression")
	}
	for _, tt := range []struct {
		name, mode, prelude, want string
	}{
		{"container exports", ContainerEnvironment, "export SHELLOPTS BASHOPTS\n", "a'b\nsecond=line|"},
		{"login shell exports", UserEnvironment, "", "a'b\nsecond=line|loaded"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			shell := filepath.Join(t.TempDir(), "login-shell")
			writeScript(t, shell, "#!/bin/sh\nexec "+quote(bash)+" --posix -c "+
				quote("export SHELLOPTS BASHOPTS\nexport CRATER_PROBE_VALUE=loaded\neval \"$1\"")+" fixture \"$2\"\n")
			run, home, env := localSession(t, shell)
			// Model an image whose /bin/sh is Bash, including the unprivileged stage.
			bashArgv := func(argv []string) []string {
				script := strings.Replace(argv[2], "set -- /bin/sh -c", "set -- "+quote(bash)+" --posix -c", 1)
				return append([]string{bash, "--posix", "-c", tt.prelude + script}, argv[3:]...)
			}
			probe := func(ctx context.Context, argv []string, out, errOut io.Writer) error {
				return run(ctx, bashArgv(argv), out, errOut)
			}
			argv := []string{"/bin/sh", "-c", "printf '%s|%s' \"$CRATER_VALUE\" \"$CRATER_PROBE_VALUE\""}
			prepared, err := Command(t.Context(), probe, argv, tt.mode, false)
			if err != nil {
				t.Fatal(err)
			}
			prepared.Argv = bashArgv(prepared.Argv)
			var stdout, stderr bytes.Buffer
			if err := runPrepared(t.Context(), prepared, env, home, &stdout, &stderr); err != nil {
				t.Fatalf("command failed: %v, stderr=%q", err, stderr.String())
			}
			if stdout.String() != tt.want {
				t.Fatalf("output=%q, want %q", stdout.String(), tt.want)
			}
		})
	}
}
