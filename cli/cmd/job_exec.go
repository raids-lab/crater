package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/muesli/cancelreader"
	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const maxExecJSONBytes = 16 << 20

var jobExecCmd = &cobra.Command{
	Use: "exec <name> [flags] [-- COMMAND [ARG...]]", Short: "Open a user shell or execute a command in a running job",
	Args: validateJobExecArgs, RunE: runJobExec,
}

func validateJobExecArgs(cmd *cobra.Command, args []string) error {
	issues := []usageIssue{}
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		issues = append(issues, missingIssue("name", "job_label_name"))
	}
	if (len(args) > 1 && (cmd.ArgsLenAtDash() != 1 || args[1] == "")) ||
		(len(args) == 1 && cmd.ArgsLenAtDash() >= 0) {
		issues = append(issues, invalidIssue("command", i18n.T("err_job_exec_command")))
	}
	for _, arg := range args {
		if strings.ContainsRune(arg, 0) {
			issues = append(issues, invalidIssue("command", i18n.T("err_job_exec_nul")))
			break
		}
	}
	stdin, tty := jobExecStreams(cmd, args)
	if tty && !stdin {
		issues = append(issues, invalidIssue("tty", i18n.T("err_job_exec_tty_stdin")))
	}
	if tty && (outputJSON || noInteractive || !localTerminals()) {
		issues = append(issues, invalidIssue("tty", i18n.T("err_job_connect_tty")))
	}
	mode, _ := cmd.Flags().GetString("env-mode")
	if mode != "user" && mode != "container" {
		issues = append(issues, invalidIssue("env-mode", i18n.T("err_job_exec_env_mode")))
	}
	if len(args) == 1 {
		for _, flag := range []string{"stdin", "tty"} {
			value, _ := cmd.Flags().GetBool(flag)
			if cmd.Flags().Changed(flag) && !value {
				issues = append(issues, invalidIssue(flag, i18n.T("err_job_exec_shell_streams")))
			}
		}
	}
	for _, flag := range []string{"pod", "container"} {
		value, _ := cmd.Flags().GetString(flag)
		if cmd.Flags().Changed(flag) && strings.TrimSpace(value) == "" {
			issues = append(issues, invalidIssue(flag, i18n.T("err_job_exec_empty_target", flag)))
		}
	}
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	return nil
}

func jobExecStreams(cmd *cobra.Command, args []string) (stdin, tty bool) {
	if len(args) == 1 {
		return true, true
	}
	stdin, _ = cmd.Flags().GetBool("stdin")
	tty, _ = cmd.Flags().GetBool("tty")
	return stdin, tty
}

func runJobExec(cmd *cobra.Command, args []string) (runErr error) {
	client, err := activeAPIClient()
	if err != nil {
		return err
	}
	pod, _ := cmd.Flags().GetString("pod")
	container, _ := cmd.Flags().GetString("container")
	stdin, tty := jobExecStreams(cmd, args)
	envMode, _ := cmd.Flags().GetString("env-mode")
	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()
	options := api.JobExecOptions{Pod: strings.TrimSpace(pod), Container: strings.TrimSpace(container), Command: args[1:], EnvMode: envMode, TTY: tty}
	if tty {
		state, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return execLocalError(err)
		}
		defer func() {
			if err := term.Restore(int(os.Stdin.Fd()), state); err != nil && runErr == nil {
				runErr = execLocalError(err)
			}
		}()
		options.Size = currentTerminalSize()
		options.Resize = watchTerminalSize(ctx)
	}
	if stdin {
		// cancelreader's Windows implementation opens CONIN$ and only handles
		// console input. Redirected files/pipes use os.File's cancellable Close.
		if runtime.GOOS == "windows" && !term.IsTerminal(int(os.Stdin.Fd())) {
			options.Input = os.Stdin
		} else {
			reader, err := cancelreader.NewReader(os.Stdin)
			if err != nil {
				return execLocalError(err)
			}
			options.Input = &execInput{CancelReader: reader}
		}
	}
	var stdout, stderr bytes.Buffer
	var out, errOut io.Writer = os.Stdout, os.Stderr
	if outputJSON {
		remaining := maxExecJSONBytes
		out, errOut = &execCapture{dst: &stdout, remaining: &remaining}, &execCapture{dst: &stderr, remaining: &remaining}
	}
	result, err := client.StreamJobExec(ctx, args[0], options, out, errOut)
	if ctx.Err() != nil {
		return errOperationCancelled()
	}
	if err != nil {
		var streamErr *api.ExecStreamError
		if errors.As(err, &streamErr) && streamErr.Local {
			return execLocalError(err)
		}
		return cliErrFromAPI(err)
	}
	if result.ExitCode != 0 {
		facts := map[string]interface{}{"namespace": result.Namespace, "pod": result.Pod, "container": result.Container, "exit_code": result.ExitCode}
		if outputJSON {
			facts["stdout"], facts["stderr"] = stdout.String(), stderr.String()
		}
		return &clierror.Error{Category: errorcodes.CategoryAPI, Code: errorcodes.ErrRemoteCommandFailed,
			Message: i18n.T("err_job_exec_exit", result.ExitCode), Context: facts}
	}
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"exec": struct {
			api.JobExecResult
			Stdout string `json:"stdout"`
			Stderr string `json:"stderr"`
		}{result, stdout.String(), stderr.String()}}))
	}
	return nil
}

type execInput struct{ cancelreader.CancelReader }

func (r *execInput) Close() error {
	r.Cancel()
	return r.CancelReader.Close()
}

type execCapture struct {
	dst       *bytes.Buffer
	remaining *int
}

func (w *execCapture) Write(data []byte) (int, error) {
	if len(data) > *w.remaining {
		return 0, errors.New(i18n.T("err_job_exec_output_limit"))
	}
	*w.remaining -= len(data)
	return w.dst.Write(data)
}

func execLocalError(err error) *clierror.Error {
	return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrCommandExecution,
		Message: i18n.T("err_job_exec_local", err), Context: map[string]interface{}{"msg": err.Error()}}
}

func localTerminals() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func currentTerminalSize() api.TerminalSize {
	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || cols <= 0 || rows <= 0 || cols > 65535 || rows > 65535 {
		return api.TerminalSize{Cols: 80, Rows: 24}
	}
	return api.TerminalSize{Cols: uint16(cols), Rows: uint16(rows)}
}

func watchTerminalSize(ctx context.Context) <-chan api.TerminalSize {
	sizes := make(chan api.TerminalSize, 1)
	go func() {
		defer close(sizes)
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		previous := currentTerminalSize()
		for {
			select {
			case <-ticker.C:
				size := currentTerminalSize()
				if size != previous {
					select {
					case sizes <- size:
						previous = size
					case <-ctx.Done():
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return sizes
}

func init() {
	jobExecCmd.Flags().String("pod", "", "Select a pod by name")
	jobExecCmd.Flags().StringP("container", "c", "", "Select a container by name")
	jobExecCmd.Flags().BoolP("stdin", "i", false, "Forward standard input")
	jobExecCmd.Flags().BoolP("tty", "t", false, "Allocate a terminal")
	jobExecCmd.Flags().String("env-mode", "user", "Load user configuration or use the container environment")
	_ = jobExecCmd.RegisterFlagCompletionFunc("env-mode", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"user", "container"}, cobra.ShellCompDirectiveNoFileComp
	})
	jobCmd.AddCommand(jobExecCmd)
}
