package snaptest

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Result holds captured stdout, stderr, and exit code from a CLI subprocess.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Run executes bin with args (argv slice after program name: e.g. {"config","language"}).
// env should typically come from EnvMinimal; stdin is always empty.
func Run(bin string, env, args []string) (*Result, error) {
	if bin == "" {
		return nil, fmt.Errorf("snaptest: empty binary path")
	}
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	cmd.Stdin = strings.NewReader("")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := &Result{
		Stdout: normalizeText(stdout.String()),
		Stderr: normalizeText(stderr.String()),
	}
	if err == nil {
		res.ExitCode = 0
		return res, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		res.ExitCode = ee.ExitCode()
		return res, nil
	}
	return nil, fmt.Errorf("snaptest: run %v: %w", append([]string{bin}, args...), err)
}

func normalizeText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}
