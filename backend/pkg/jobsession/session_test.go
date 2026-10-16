// Copyright 2026 The Crater Project Team, RAIDS-Lab

package jobsession

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// localSession runs the production POSIX wrapper with an isolated account
// database, HOME and shell. It never provisions or changes a host account.
func localSession(t *testing.T, shell string) (run Runner, home string, env []string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("session scripts execute on Linux containers")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	entry := fmt.Sprintf("alice:x:%d:%d::%s:%s", os.Getuid(), os.Getgid(), dir, shell)
	writeScript(t, filepath.Join(bin, "getent"), "#!/bin/sh\nprintf '%s\\n' "+quote(entry)+"\n")
	env = []string{"PATH=" + bin + ":/usr/bin:/bin:/usr/sbin:/sbin", "NB_USER=alice", "HOME=/incorrect", "SHELL=/incorrect", "USER=incorrect", "LOGNAME=incorrect", "CRATER_VALUE=a'b\nsecond=line", "CRATER_REMOVE=container", "ASCEND_HOME_PATH=/opt/ascend", "LD_LIBRARY_PATH=/vendor/lib"}
	run = environmentRunner(env, dir)
	return run, dir, env
}

func writeScript(t *testing.T, path, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestCommandPreservesAccountEnvironmentAndArgv(t *testing.T) {
	for _, mode := range []string{UserEnvironment, ContainerEnvironment} {
		t.Run(mode, func(t *testing.T) {
			run, home, env := localSession(t, "/bin/sh")
			argv := []string{"/bin/sh", "-c", `printf '%s\0' "$HOME" "$USER" "$LOGNAME" "$SHELL" "$CRATER_VALUE" "$ASCEND_HOME_PATH" "$LD_LIBRARY_PATH" "$@"`, "fixture", "a b", "$(touch ignored)", "", "--json"}
			prepared, err := Command(t.Context(), run, argv, mode, false)
			if err != nil {
				t.Fatal(err)
			}
			var output, stderr bytes.Buffer
			if err = runPrepared(t.Context(), prepared, env, home, &output, &stderr); err != nil {
				t.Fatalf("command error=%v stderr=%q", err, stderr.String())
			}
			want := strings.Join([]string{home, "alice", "alice", "/bin/sh", "a'b\nsecond=line", "/opt/ascend", "/vendor/lib", "a b", "$(touch ignored)", "", "--json", ""}, "\x00")
			if output.String() != want {
				t.Fatalf("output=%q, want %q", output.String(), want)
			}
		})
	}
}

func TestUserConfigurationIsProbedOnceWithoutStartupNoise(t *testing.T) {
	_, home, env := localSession(t, "/bin/sh")
	profile := filepath.Join(home, "profile")
	writeScript(t, profile, "#!/bin/sh\necho startup-noise\nexport TOOLCHAIN_VALUE=loaded\nunset CRATER_REMOVE\nexport CRATER_VALUE=\"from user's config\"\n")
	// POSIX sh reads ENV for interactive shells, without consulting host rc files.
	env = append(env, "ENV="+profile)
	run := environmentRunner(env, home)
	for _, tt := range []struct{ mode, want string }{
		{UserEnvironment, "loaded|from user's config|unset"},
		{ContainerEnvironment, "|a'b\nsecond=line|container"},
	} {
		t.Run(tt.mode, func(t *testing.T) {
			prepared, err := Command(t.Context(), run, []string{"/bin/sh", "-c", `printf '%s|%s|%s' "${TOOLCHAIN_VALUE:-}" "$CRATER_VALUE" "${CRATER_REMOVE:-unset}"`}, tt.mode, false)
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err = runPrepared(t.Context(), prepared, env, home, &output, io.Discard); err != nil || output.String() != tt.want {
				t.Fatalf("output=%q error=%v, want %q", output.String(), err, tt.want)
			}
		})
	}
}

func TestAccountShellAndHomeAreUsedForInteractiveSession(t *testing.T) {
	for _, kind := range []string{"bash", "zsh", "fish", "sh"} {
		t.Run(kind, func(t *testing.T) {
			// Observe the exact account shell selected, independent of installed shells.
			shell := filepath.Join(t.TempDir(), kind)
			writeScript(t, shell, "#!/bin/sh\nprintf '%s|%s|%s|%s' \"$HOME\" \"$USER\" \"$SHELL\" \"$*\"\n")
			run, home, env := localSession(t, shell)
			prepared, err := Command(t.Context(), run, nil, UserEnvironment, false)
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err = runPrepared(t.Context(), prepared, env, home, &output, io.Discard); err != nil {
				t.Fatal(err)
			}
			want := home + "|alice|" + shell + "|-il"
			if output.String() != want {
				t.Fatalf("session=%q, want %q", output.String(), want)
			}
		})
	}
}

func TestProbeFailuresAreBoundedAndActionable(t *testing.T) {
	for _, kind := range []string{"cancel", "overflow", "missing marker", "execution error"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
			defer cancel()
			run := func(ctx context.Context, _ []string, stdout, _ io.Writer) error {
				switch kind {
				case "cancel":
					<-ctx.Done()
					return ctx.Err()
				case "overflow":
					_, err := stdout.Write(bytes.Repeat([]byte("x"), maxProbeBytes+1))
					return err
				case "missing marker":
					_, err := io.WriteString(stdout, "startup banner only")
					return err
				default:
					return errors.New("private fixture diagnostic")
				}
			}
			_, err := Command(ctx, run, []string{"true"}, UserEnvironment, false)
			if err == nil || !strings.Contains(err.Error(), "--env-mode container") || strings.Contains(err.Error(), "private fixture") {
				t.Fatalf("error=%v, want safe recovery guidance", err)
			}
		})
	}
}

func TestMissingOrDifferentUserNeverFallsBack(t *testing.T) {
	for _, kind := range []string{"missing", "different uid"} {
		t.Run(kind, func(t *testing.T) {
			_, dir, env := localSession(t, "/bin/sh")
			getent := filepath.Join(dir, "bin", "getent")
			body := "#!/bin/sh\nexit 2\n"
			if kind == "different uid" {
				if os.Geteuid() == 0 {
					t.Skip("root can legitimately switch uid")
				}
				body = "#!/bin/sh\nprintf '%s\\n' " + quote("alice:x:0:0::"+dir+":/bin/sh")
			}
			writeScript(t, getent, body)
			run := environmentRunner(env, dir)
			prepared, err := Command(t.Context(), run, []string{"/bin/sh", "-c", "echo should-not-run"}, ContainerEnvironment, false)
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err = runPrepared(t.Context(), prepared, env, dir, &output, io.Discard); err == nil || output.Len() != 0 {
				t.Fatalf("error=%v output=%q, want failure without execution", err, output.String())
			}
		})
	}
}

func environmentRunner(env []string, dir string) Runner {
	return func(ctx context.Context, argv []string, stdout, stderr io.Writer) error {
		//nolint:gosec // Executes only production argv against isolated test fixtures.
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		cmd.Env, cmd.Dir, cmd.Stdout, cmd.Stderr = env, dir, stdout, stderr
		cmd.WaitDelay = 100 * time.Millisecond
		return cmd.Run()
	}
}

func TestShellQuoteAndEnvironmentNames(t *testing.T) {
	for _, value := range []string{"", "a b", "a'b\nline=value", "$(false)", "\\", "中文"} {
		t.Run(strconv.Quote(value), func(t *testing.T) {
			//nolint:gosec // Verifies quoting of fixed adversarial test data.
			out, err := exec.CommandContext(t.Context(), "/bin/sh", "-c", "printf '%s' "+quote(value)).Output()
			if err != nil || string(out) != value {
				t.Fatalf("output=%q error=%v, want %q", out, err, value)
			}
		})
	}
	for _, name := range []string{"HOME", "SHELL", "crater_home", "PWD"} {
		if !identityVariable(name) {
			t.Errorf("identity variable %q was accepted", name)
		}
	}
	for _, name := range []string{"", "1NAME", "BASH_FUNC_f%%", "a;b", "a-b"} {
		if validName(name) {
			t.Errorf("unsafe export name %q was accepted", name)
		}
	}
}

func runPrepared(ctx context.Context, prepared Prepared, env []string, dir string, stdout, stderr io.Writer) error {
	//nolint:gosec // The command is produced by the session implementation under test.
	cmd := exec.CommandContext(ctx, prepared.Argv[0], prepared.Argv[1:]...)
	cmd.Env, cmd.Dir, cmd.Stdout, cmd.Stderr = env, dir, stdout, stderr
	cmd.Stdin = bytes.NewReader(prepared.Input)
	cmd.WaitDelay = 100 * time.Millisecond
	return cmd.Run()
}

func TestPrivateEnvironmentPrefixPreservesCommandStdin(t *testing.T) {
	run, home, env := localSession(t, "/bin/sh")
	prepared, err := Command(t.Context(), run, []string{"cat"}, UserEnvironment, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(prepared.Argv, " "), "second=line") {
		t.Fatal("private environment value appeared in Kubernetes argv")
	}
	input := []byte{0, 0xff, 'a', '\n'}
	prepared.Input = append(prepared.Input, input...)
	var output bytes.Buffer
	if err = runPrepared(t.Context(), prepared, env, home, &output, io.Discard); err != nil || !bytes.Equal(output.Bytes(), input) {
		t.Fatalf("output=%q error=%v, want original stdin=%q", output.Bytes(), err, input)
	}
}

func TestTruncatedEnvironmentPreventsCommandExecution(t *testing.T) {
	run, home, env := localSession(t, "/bin/sh")
	prepared, err := Command(t.Context(), run, []string{"/bin/sh", "-c", "echo should-not-run"}, UserEnvironment, false)
	if err != nil {
		t.Fatal(err)
	}
	prepared.Input = prepared.Input[:len(prepared.Input)-10]
	var output bytes.Buffer
	if err = runPrepared(t.Context(), prepared, env, home, &output, io.Discard); err == nil || output.Len() != 0 {
		t.Fatalf("error=%v output=%q, want rejection before execution", err, output.String())
	}
}

func TestPrivilegeHelpersRestoreLoaderEnvironmentAndArgv(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("the fixture needs a target uid different from root")
	}
	for _, helper := range []string{"runuser", "sudo", "su"} {
		t.Run(helper, func(t *testing.T) {
			_, home, env := localSession(t, "/bin/sh")
			bin := filepath.Join(home, "bin")
			// Model a privilege helper clearing loader variables, without changing
			// any host credentials. Restrict PATH to exercise each fallback.
			writeScript(t, filepath.Join(bin, "id"), `#!/bin/sh
if [ "$*" = -u ] && [ -z "${CRATER_TEST_DROPPED:-}" ]; then echo 0; else exec /usr/bin/id "$@"; fi
`)
			for _, tool := range []string{"sed"} {
				if err := os.Symlink("/usr/bin/"+tool, filepath.Join(bin, tool)); err != nil {
					t.Fatal(err)
				}
			}
			body := `#!/bin/sh
export CRATER_TEST_DROPPED=1
unset LD_LIBRARY_PATH
while [ "$#" -gt 0 ]; do
    if [ "$1" = -- ]; then shift; exec "$@"; fi
    if [ "$1" = -c ]; then shift; exec /bin/sh -c "$1"; fi
    shift
done
exit 1
`
			writeScript(t, filepath.Join(bin, helper), body)
			env = append(env, "PATH="+bin)
			run := environmentRunner(env, home)
			argv := []string{"/bin/sh", "-c", `printf '%s\0' "$LD_LIBRARY_PATH" "$@"`, "fixture", "a'b\n", "$(false)", ""}
			prepared, err := Command(t.Context(), run, argv, ContainerEnvironment, false)
			if err != nil {
				t.Fatal(err)
			}
			var output, stderr bytes.Buffer
			if err = runPrepared(t.Context(), prepared, env, home, &output, &stderr); err != nil {
				t.Fatalf("helper=%s error=%v stderr=%q", helper, err, stderr.String())
			}
			want := "/vendor/lib\x00a'b\n\x00$(false)\x00\x00"
			if output.String() != want {
				t.Fatalf("helper=%s output=%q, want %q", helper, output.String(), want)
			}
		})
	}
}

func TestSymlinkSnapshotDirectoryIsNeverSourced(t *testing.T) {
	_, home, env := localSession(t, "/bin/sh")
	bin := filepath.Join(home, "bin")
	writeScript(t, filepath.Join(bin, "id"), "#!/bin/sh\necho 0\n")
	writeScript(t, filepath.Join(bin, "getent"), "#!/bin/sh\nprintf '%s\\n' "+quote("alice:x:0:0::"+home+":/bin/sh")+"\n")
	target := filepath.Join(home, "untrusted")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, "session")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(home, "snapshot-was-sourced")
	writeScript(t, filepath.Join(target, "environment.sh"), "printf unsafe > "+quote(marker)+"\nexport UNTRUSTED_VALUE=unsafe\n")
	prepared, err := Command(t.Context(), nil, []string{"/bin/sh", "-c", `printf '%s' "${UNTRUSTED_VALUE:-clean}"`}, ContainerEnvironment, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		argv []string
		fail bool
	}{
		{"exec ignores snapshot", prepared.Argv, false},
		{"SSH setup rejects before reading", InstallSSHCommand(), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			argv := append([]string(nil), tt.argv...)
			for i := range argv {
				argv[i] = strings.ReplaceAll(argv[i], "/run/crater", link)
			}
			var output, stderr bytes.Buffer
			err := environmentRunner(env, home)(t.Context(), argv, &output, &stderr)
			if (err != nil) != tt.fail || (!tt.fail && output.String() != "clean") {
				t.Fatalf("output=%q stderr=%q error=%v, want failure=%v", output.String(), stderr.String(), err, tt.fail)
			}
			if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("untrusted snapshot was sourced: marker error=%v", err)
			}
		})
	}
}

func TestTTYCommandLoadsUserConfigurationWithoutPrivateInput(t *testing.T) {
	run, home, env := localSession(t, "/bin/sh")
	profile := filepath.Join(home, "profile")
	writeScript(t, profile, "#!/bin/sh\necho startup-noise\nexport TOOLCHAIN_VALUE=loaded\n")
	env = append(env, "ENV="+profile)
	private := "a'b\nsecond=line"
	argv := []string{"/bin/sh", "-c", `printf '%s|%s' "$TOOLCHAIN_VALUE" "$1"`, "fixture", "a'b\n$(false)"}
	prepared, err := Command(t.Context(), run, argv, UserEnvironment, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.Input) != 0 || strings.Contains(strings.Join(prepared.Argv, " "), private) {
		t.Fatal("private environment would be sent through a TTY or Kubernetes URL")
	}
	var output, stderr bytes.Buffer
	if err = runPrepared(t.Context(), prepared, env, home, &output, &stderr); err != nil {
		t.Fatalf("error=%v stderr=%q", err, stderr.String())
	}
	want := "startup-noise\nloaded|a'b\n$(false)"
	if output.String() != want {
		t.Fatalf("output=%q, want %q", output.String(), want)
	}
}
