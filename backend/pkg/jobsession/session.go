// Copyright 2026 The Crater Project Team, RAIDS-Lab

// Package jobsession prepares the same Linux user environment for exec and SSH.
package jobsession

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// UserEnvironment loads the account's exported login/interactive environment.
	UserEnvironment = "user"
	// ContainerEnvironment skips configuration probing, retaining user identity.
	ContainerEnvironment = "container"
	probeTimeout         = 10 * time.Second
	maxProbeBytes        = 1 << 20
)

//go:embed bootstrap.sh
var bootstrap string

//go:embed user.sh
var userStage string

// Runner executes argv in the selected container without a terminal or stdin.
type Runner func(context.Context, []string, io.Writer, io.Writer) error

// Prepared keeps private environment data out of the Kubernetes exec URL.
// Input must precede user stdin; the wrapper consumes exactly that many bytes.
type Prepared struct {
	Argv  []string
	Input []byte
}

// Command builds an argv-preserving command or an interactive login shell.
// Probing is bounded and never forwards startup output to the command streams.
func Command(ctx context.Context, run Runner, argv []string, mode string, tty bool) (Prepared, error) {
	if mode == "" {
		mode = UserEnvironment
	}
	if mode != UserEnvironment && mode != ContainerEnvironment {
		return Prepared{}, errors.New("env_mode must be user or container")
	}
	if len(argv) == 0 {
		return Prepared{Argv: command("shell")}, nil
	}
	if tty && mode == UserEnvironment {
		// A PTY can echo stdin before the wrapper starts. Load configuration in
		// the user's shell instead of sending private environment through a PTY.
		return Prepared{Argv: command("terminal-command", "exec "+shellCommand(argv))}, nil
	}
	var exports string
	if mode == UserEnvironment {
		var err error
		exports, err = probe(ctx, run)
		if err != nil {
			return Prepared{}, err
		}
	}
	return Prepared{Argv: command("command", append([]string{strconv.Itoa(len(exports))}, argv...)...), Input: []byte(exports)}, nil
}

func command(mode string, args ...string) []string {
	script := strings.Replace(bootstrap, "__USER_SCRIPT__", quote(userStage), 1)
	return append([]string{"/bin/sh", "-c", script, "crater-session", mode}, args...)
}

func probe(ctx context.Context, run Runner) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	marker := "\x1e" + rand.Text() + "\x1f"
	baseName := rand.Text()
	baseMarker := "\x1e" + baseName + "\x1f"
	probeScript := fmt.Sprintf("printf '\\036%%s\\037' %s; env -0 && printf '\\036%%s\\037' %s",
		quote(marker[1:len(marker)-1]), quote(marker[1:len(marker)-1]))
	var output limitedBuffer
	if err := run(ctx, command("probe", probeScript, baseName), &output, io.Discard); err != nil {
		return "", errors.New("user environment preparation failed; check the task account and shell configuration, or use --env-mode container")
	}
	env, err := readEnvironment(output.Bytes(), marker)
	if err != nil {
		return "", errors.New("user environment probe returned no environment; check shell startup scripts or use --env-mode container")
	}
	base, err := readEnvironment(output.Bytes(), baseMarker)
	if err != nil {
		return "", errors.New("user environment probe returned no base environment; use --env-mode container")
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var result strings.Builder
	var removed []string
	for key := range base {
		if _, exists := env[key]; !exists {
			removed = append(removed, key)
		}
	}
	sort.Strings(removed)
	for _, key := range removed {
		fmt.Fprintf(&result, "unset %s\n", key)
	}
	for _, key := range keys {
		fmt.Fprintf(&result, "export %s=%s\n", key, quote(env[key]))
	}
	return result.String(), nil
}

func readEnvironment(output []byte, marker string) (map[string]string, error) {
	start := bytes.Index(output, []byte(marker))
	end := bytes.LastIndex(output, []byte(marker))
	if start < 0 || end <= start {
		return nil, errors.New("missing environment markers")
	}
	env := map[string]string{}
	for _, entry := range bytes.Split(output[start+len(marker):end], []byte{0}) {
		name, value, ok := strings.Cut(string(entry), "=")
		if ok && validName(name) && !identityVariable(name) {
			env[name] = value
		}
	}
	return env, nil
}

func quote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }

func identityVariable(name string) bool {
	switch name {
	case "HOME", "USER", "LOGNAME", "SHELL", "PWD", "OLDPWD", "SHLVL", "_":
		return true
	}
	return strings.HasPrefix(name, "crater_")
}

func validName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		letter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		digit := r >= '0' && r <= '9'
		if r == '_' || letter || (i > 0 && digit) {
			continue
		}
		return false
	}
	return true
}

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) > maxProbeBytes-b.Len() {
		return 0, errors.New("user environment exceeds 1 MiB")
	}
	return b.Buffer.Write(p)
}
