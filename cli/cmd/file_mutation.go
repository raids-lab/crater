package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

var fileMkdirCmd = &cobra.Command{
	Use:   "mkdir <remote-path>",
	Short: "Create one remote directory",
	Long:  "Create exactly one directory below user, public, or account storage without creating missing parents.",
	Args:  fileMkdirArgs,
	RunE:  runFileMkdir,
}

var fileMoveCmd = &cobra.Command{
	Use:   "mv <source-path> <destination-path>",
	Short: "Move one remote file or directory",
	Long:  "Move one remote storage entry to an exact destination path without replacing an existing entry.",
	Args:  fileMoveArgs,
	RunE:  runFileMove,
}

type fileMutationDeps struct {
	client func() (api.FileMutationClient, error)
	stdout io.Writer
	json   bool
}

func fileMkdirArgs(cmd *cobra.Command, args []string) error {
	return exactFileMutationArgs(cmd, args, []fileArgument{{
		field: "remote-path",
		label: i18n.T("file_label_remote_directory"),
	}})
}

func fileMoveArgs(cmd *cobra.Command, args []string) error {
	return exactFileMutationArgs(cmd, args, []fileArgument{
		{field: "source-path", label: i18n.T("file_label_source_path")},
		{field: "destination-path", label: i18n.T("file_label_destination_path")},
	})
}

type fileArgument struct {
	field string
	label string
}

func exactFileMutationArgs(cmd *cobra.Command, args []string, required []fileArgument) error {
	if len(args) > len(required) {
		return errTooManyArgs(cmd, len(args), len(required))
	}
	if len(args) < len(required) {
		missing := required[len(args)]
		return errUsageFromIssues([]usageIssue{{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_missing_required_arg", missing.label, missing.field),
			Field:   missing.field,
		}})
	}
	return nil
}

func runFileMkdir(cmd *cobra.Command, args []string) error {
	return runFileMkdirWith(cmd.Context(), args, fileMutationDeps{
		client: activeFileMutationClient,
		stdout: os.Stdout,
		json:   outputJSON,
	})
}

func runFileMove(cmd *cobra.Command, args []string) error {
	return runFileMoveWith(cmd.Context(), args, fileMutationDeps{
		client: activeFileMutationClient,
		stdout: os.Stdout,
		json:   outputJSON,
	})
}

func activeFileMutationClient() (api.FileMutationClient, error) {
	return activeAPIClient()
}

func runFileMkdirWith(ctx context.Context, args []string, deps fileMutationDeps) error {
	remotePath, issue := normalizeMutationPath(args[0], "remote-path")
	if issue != nil {
		return errUsageFromIssues([]usageIssue{*issue})
	}
	client, err := deps.client()
	if err != nil {
		return err
	}
	if err := client.CreateDirectory(ctx, remotePath); err != nil {
		return cliErrFromAPI(err)
	}
	return writeFileMutationResult(
		deps.stdout,
		deps.json,
		map[string]interface{}{"remote_path": remotePath},
		i18n.T("file_mkdir_success", remotePath),
	)
}

func runFileMoveWith(ctx context.Context, args []string, deps fileMutationDeps) error {
	sourcePath, sourceIssue := normalizeMutationPath(args[0], "source-path")
	destinationPath, destinationIssue := normalizeMutationPath(args[1], "destination-path")
	issues := make([]usageIssue, 0, 2)
	if sourceIssue != nil {
		issues = append(issues, *sourceIssue)
	}
	if destinationIssue != nil {
		issues = append(issues, *destinationIssue)
	}
	if len(issues) > 0 {
		return errUsageFromIssues(issues)
	}
	switch {
	case sourcePath == destinationPath:
		return errUsageFromIssues([]usageIssue{invalidIssue(
			"destination-path",
			i18n.T("err_file_move_same", destinationPath),
		)})
	case strings.HasPrefix(destinationPath, sourcePath+"/"):
		return errUsageFromIssues([]usageIssue{invalidIssue(
			"destination-path",
			i18n.T("err_file_move_descendant", destinationPath, sourcePath),
		)})
	}

	client, err := deps.client()
	if err != nil {
		return err
	}
	if err := client.MoveFile(ctx, sourcePath, destinationPath); err != nil {
		return cliErrFromAPI(err)
	}
	return writeFileMutationResult(
		deps.stdout,
		deps.json,
		map[string]interface{}{
			"source_path":      sourcePath,
			"destination_path": destinationPath,
		},
		i18n.T("file_move_success", sourcePath, destinationPath),
	)
}

func normalizeMutationPath(rawPath, field string) (string, *usageIssue) {
	normalized, err := normalizeRemotePath(rawPath)
	if err != nil {
		var cliErr *clierror.Error
		code := errorcodes.ErrInvalidFlagValue
		message := err.Error()
		if errors.As(err, &cliErr) {
			code = cliErr.Code
			message = cliErr.Message
		}
		issue := usageIssue{Code: code, Message: message, Field: field}
		return "", &issue
	}
	if len(strings.Split(normalized, "/")) < 2 {
		issue := invalidIssue(field, i18n.T("err_file_path_not_entry", rawPath))
		return "", &issue
	}
	return normalized, nil
}

func writeFileMutationResult(
	writer io.Writer,
	jsonOutput bool,
	data map[string]interface{},
	humanMessage string,
) error {
	if jsonOutput {
		return output.WriteSuccessJSON(writer, output.SuccessEnvelope(data))
	}
	if _, err := fmt.Fprintln(writer, humanMessage); err != nil {
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrCommandExecution,
			Message:  i18n.T("err_file_output", err.Error()),
			Context:  map[string]interface{}{"msg": err.Error()},
		}
	}
	return nil
}

func init() {
	fileCmd.AddCommand(fileMkdirCmd, fileMoveCmd)
	completion.RegisterPositional([]string{"file", "mkdir"}, 0, fileRootCompleter)
	completion.RegisterPositional([]string{"file", "mv"}, 0, fileRootCompleter)
	completion.RegisterPositional([]string{"file", "mv"}, 1, fileRootCompleter)
}
