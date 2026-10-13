package cmd

import (
	"context"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

var fileRemoveCmd = &cobra.Command{
	Use:   "rm <remote-path>",
	Short: "Remove one remote file or directory",
	Long:  "Remove exactly one remote entry below user, public, or account storage. Directories require --recursive.",
	Args:  fileRemoveArgs,
	RunE:  runFileRemove,
}

type fileRemoveDeps struct {
	client        func() (api.FileRemoveClient, error)
	confirm       func(string) (bool, error)
	stdout        io.Writer
	json          bool
	noInteractive bool
}

func fileRemoveArgs(cmd *cobra.Command, args []string) error {
	return exactFileMutationArgs(cmd, args, []fileArgument{{
		field: "remote-path",
		label: i18n.T("file_label_remote_path"),
	}})
}

func runFileRemove(cmd *cobra.Command, args []string) error {
	return runFileRemoveWith(cmd.Context(), cmd, args, fileRemoveDeps{
		client:        activeFileRemoveClient,
		confirm:       confirmFileRemove,
		stdout:        os.Stdout,
		json:          outputJSON,
		noInteractive: noInteractive,
	})
}

func activeFileRemoveClient() (api.FileRemoveClient, error) {
	return activeAPIClient()
}

func runFileRemoveWith(
	ctx context.Context,
	cmd *cobra.Command,
	args []string,
	deps fileRemoveDeps,
) error {
	remotePath, issue := normalizeRemovePath(args[0])
	if issue != nil {
		return errUsageFromIssues([]usageIssue{*issue})
	}

	recursive, _ := cmd.Flags().GetBool("recursive")
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		if deps.json || deps.noInteractive {
			return &clierror.Error{
				Category: errorcodes.CategoryUsage,
				Code:     errorcodes.ErrMissingRequiredFlag,
				Message:  i18n.T("err_confirm_required"),
			}
		}
		confirmKey := "file_remove_confirm"
		if recursive {
			confirmKey = "file_remove_recursive_confirm"
		}
		confirmed, err := deps.confirm(i18n.T(confirmKey, remotePath))
		if err != nil {
			return errSurveyOrSame(err)
		}
		if !confirmed {
			return errOperationCancelled()
		}
	}

	client, err := deps.client()
	if err != nil {
		return err
	}
	result, err := client.RemoveFile(ctx, remotePath, recursive)
	if err != nil {
		return cliErrFromAPI(err)
	}
	return writeFileMutationResult(
		deps.stdout,
		deps.json,
		map[string]interface{}{
			"remote_path": result.RemotePath,
			"recursive":   result.Recursive,
		},
		i18n.T("file_remove_success", result.RemotePath),
	)
}

func confirmFileRemove(message string) (bool, error) {
	var confirmed bool
	prompt := &survey.Confirm{Message: message, Default: false}
	if err := survey.AskOne(prompt, &confirmed); err != nil {
		return false, err
	}
	return confirmed, nil
}

// normalizeRemovePath is intentionally stricter than the shared file path
// normalizer: deletion rejects every raw "." or ".." segment instead of
// silently canonicalizing it.
func normalizeRemovePath(rawPath string) (string, *usageIssue) {
	invalid := func(message string) (string, *usageIssue) {
		issue := invalidIssue("remote-path", message)
		return "", &issue
	}
	if rawPath == "" || strings.ContainsRune(rawPath, '\\') {
		return invalid(i18n.T("err_file_path_invalid", rawPath))
	}
	for _, character := range rawPath {
		if unicode.IsControl(character) {
			return invalid(i18n.T("err_file_path_invalid", rawPath))
		}
	}

	rawSegments := strings.Split(strings.Trim(rawPath, "/"), "/")
	segments := make([]string, 0, len(rawSegments))
	for _, segment := range rawSegments {
		if segment == "." || segment == ".." {
			return invalid(i18n.T("err_file_remove_ambiguous_path", rawPath))
		}
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	if len(segments) == 0 {
		return invalid(i18n.T("err_file_path_invalid", rawPath))
	}
	if !isFileRemoteRoot(segments[0]) {
		return invalid(i18n.T("err_file_path_root", rawPath))
	}
	if len(segments) < 2 {
		return invalid(i18n.T("err_file_remove_root", rawPath))
	}
	return strings.Join(segments, "/"), nil
}

func init() {
	fileCmd.AddCommand(fileRemoveCmd)
	fileRemoveCmd.Flags().Bool("recursive", false, "Remove a directory and all of its contents")
	fileRemoveCmd.Flags().BoolP("yes", "y", false, "Remove without confirmation")
	completion.RegisterPositional([]string{"file", "rm"}, 0, fileRootCompleter)
}
