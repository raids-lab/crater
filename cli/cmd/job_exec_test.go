package cmd

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/spf13/cobra"
)

func TestJobExecArgvValidation(t *testing.T) {
	for _, tt := range []struct {
		name  string
		argv  []string
		valid bool
	}{
		{"CPU command", []string{"cpu-job", "--", "uname", "-m"}, true},
		{"container environment", []string{"cpu-job", "--env-mode=container", "--", "env"}, true},
		{"invalid environment", []string{"cpu-job", "--env-mode=invalid", "--", "env"}, false},
		{"shell without local terminals", []string{"cpu-job"}, false},
		{"shell explicit", []string{"job", "--", "sh", "-c", "printf '%s' '$HOME'"}, true},
		{"remote flags untouched", []string{"job", "--", "program", "--json", "--help", ""}, true},
		{"separator missing", []string{"job", "uname"}, false},
		{"job missing", []string{"--", "uname"}, false},
		{"command missing", []string{"job", "--"}, false},
		{"NUL in argv", []string{"job", "--", "a\x00b"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool("stdin", false, "")
			cmd.Flags().Bool("tty", false, "")
			cmd.Flags().String("pod", "", "")
			cmd.Flags().String("env-mode", "user", "")
			cmd.Flags().String("container", "", "")
			if err := cmd.ParseFlags(tt.argv); err != nil {
				t.Fatal(err)
			}
			err := validateJobExecArgs(cmd, cmd.Flags().Args())
			if (err == nil) != tt.valid {
				t.Fatalf("error=%v, want valid=%v", err, tt.valid)
			}
		})
	}
}

func TestJobSSHArgsPreserveBoundaries(t *testing.T) {
	args, err := jobSSHArgs(&api.SSHInfo{IP: "2001:db8::1", Port: "30222"}, "alice", "/tmp/key with spaces")
	want := []string{"-p", "30222", "-l", "alice", "-i", "/tmp/key with spaces", "--", "2001:db8::1"}
	if err != nil || !reflect.DeepEqual(args, want) {
		t.Fatalf("args=%q error=%v, want %q", args, err, want)
	}
	for _, address := range []*api.SSHInfo{nil, {IP: "-oProxyCommand=bad", Port: "22"}, {IP: "host\nother", Port: "22"}, {IP: "localhost", Port: "0"}, {IP: "localhost", Port: "65536"}} {
		if _, err := jobSSHArgs(address, "alice", ""); err == nil {
			t.Fatalf("accepted invalid endpoint %+v", address)
		}
	}
}

func TestJobSSHUsesSharedRemoteSession(t *testing.T) {
	info := &api.SSHInfo{IP: "cpu.example.test", Port: "30222", User: "task-user", SessionCommand: "/bin/sh /run/crater/session.sh"}
	args, err := jobSSHArgs(info, info.User, "")
	want := []string{"-p", "30222", "-l", "task-user", "-tt", "--", "cpu.example.test", info.SessionCommand}
	if err != nil || !reflect.DeepEqual(args, want) {
		t.Fatalf("args=%q error=%v, want %q", args, err, want)
	}
}

func TestExecJSONCaptureBoundsCombinedStreams(t *testing.T) {
	remaining := 3
	var out, errOut bytes.Buffer
	stdout, stderr := &execCapture{dst: &out, remaining: &remaining}, &execCapture{dst: &errOut, remaining: &remaining}
	if _, err := stdout.Write([]byte("ab")); err != nil {
		t.Fatal(err)
	}
	if _, err := stderr.Write([]byte("cd")); err == nil {
		t.Fatal("combined output limit was ignored")
	}
	if out.String() != "ab" || errOut.Len() != 0 {
		t.Fatalf("stdout=%q, stderr=%q", out.String(), errOut.String())
	}
}
