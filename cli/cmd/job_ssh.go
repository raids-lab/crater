package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var sshHostname = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]*$`)

func runJobSSH(cmd *cobra.Command, args []string) (runErr error) {
	name, err := requiredArg(args, "job_label_name", "name")
	if err != nil {
		return err
	}
	connect, _ := cmd.Flags().GetBool("connect")
	user, _ := cmd.Flags().GetString("user")
	identity, _ := cmd.Flags().GetString("identity-file")
	issues := []usageIssue{}
	if connect && (outputJSON || noInteractive || !localTerminals()) {
		issues = append(issues, invalidIssue("connect", i18n.T("err_job_connect_tty")))
	}
	for _, flag := range []string{"user", "identity-file"} {
		value, _ := cmd.Flags().GetString(flag)
		if cmd.Flags().Changed(flag) && (!connect || strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\x00\r\n")) {
			issues = append(issues, invalidIssue(flag, i18n.T("err_job_ssh_option", flag)))
		}
	}
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	binary := ""
	if connect {
		binary, err = exec.LookPath("ssh")
		if err != nil {
			return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrBinaryNotFound, Message: i18n.T("err_job_ssh_binary")}
		}
	}
	client, active, err := activeAPIClientWithContext()
	if err != nil {
		return err
	}
	ssh, err := client.OpenJobSSH(name, connect)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if !connect {
		if outputJSON {
			return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{"ssh": ssh}))
		}
		fmt.Fprintln(os.Stdout, net.JoinHostPort(ssh.IP, ssh.Port))
		return nil
	}
	if user == "" {
		user = ssh.User
		if user == "" {
			user = active.Username
		}
	}
	if strings.TrimSpace(user) == "" || strings.ContainsAny(user, "\x00\r\n") {
		return errUsageFromIssues([]usageIssue{invalidIssue("user", i18n.T("err_job_ssh_option", "user"))})
	}
	sshArgs, err := jobSSHArgs(ssh, user, identity)
	if err != nil {
		return cliErrFromAPI(err)
	}
	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()
	state, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return execLocalError(err)
	}
	defer func() {
		if err := term.Restore(int(os.Stdin.Fd()), state); err != nil && runErr == nil {
			runErr = execLocalError(err)
		}
	}()
	process := exec.CommandContext(ctx, binary, sshArgs...)
	process.Stdin, process.Stdout, process.Stderr = os.Stdin, os.Stdout, os.Stderr
	err = process.Run()
	if ctx.Err() != nil {
		return errOperationCancelled()
	}
	if err != nil {
		facts := map[string]interface{}{}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			facts["exit_code"] = exitErr.ExitCode()
		}
		return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrCommandExecution,
			Message: i18n.T("err_job_ssh_process", err), Context: facts}
	}
	return nil
}

func jobSSHArgs(ssh *api.SSHInfo, user, identity string) ([]string, error) {
	if ssh == nil || (net.ParseIP(ssh.IP) == nil && !sshHostname.MatchString(ssh.IP)) {
		return nil, &api.ExecStreamError{Cause: errors.New(i18n.T("err_job_ssh_address"))}
	}
	port, err := strconv.Atoi(ssh.Port)
	if err != nil || port < 1 || port > 65535 {
		return nil, &api.ExecStreamError{Cause: errors.New(i18n.T("err_job_ssh_address"))}
	}
	args := []string{"-p", strconv.Itoa(port), "-l", user}
	if ssh.SessionCommand != "" {
		if strings.ContainsRune(ssh.SessionCommand, 0) {
			return nil, &api.ExecStreamError{Cause: errors.New(i18n.T("err_job_ssh_address"))}
		}
		args = append(args, "-tt")
	}
	if identity != "" {
		args = append(args, "-i", identity)
	}
	args = append(args, "--", ssh.IP)
	if ssh.SessionCommand != "" {
		args = append(args, ssh.SessionCommand)
	}
	return args, nil
}

func init() {
	jobSSHCmd.Flags().Bool("connect", false, "Connect using local OpenSSH")
	jobSSHCmd.Flags().String("user", "", "SSH username; defaults to the active login")
	jobSSHCmd.Flags().StringP("identity-file", "i", "", "SSH private key file")
}
