package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

var fileDownloadCmd = &cobra.Command{
	Use:   "download <remote-file> [local-path]",
	Short: "Download one remote file",
	Args:  fileDownloadArgs,
	RunE:  runFileDownload,
}

type fileDownloadDeps struct {
	client func() (api.FileDownloadClient, error)
	stdout io.Writer
	json   bool
}

type fileDownloadInput struct {
	remotePath string
	localPath  string
	overwrite  bool
}

type fileDownloadResult struct {
	RemotePath string
	LocalPath  string
	Bytes      int64
	Overwrite  bool
}

func fileDownloadArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 2 {
		return errTooManyArgs(cmd, len(args), 2)
	}
	if len(args) == 0 {
		return errUsageFromIssues([]usageIssue{{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_missing_required_arg", i18n.T("file_label_remote_file"), "remote-file"),
			Field:   "remote-file",
		}})
	}
	return nil
}

func runFileDownload(cmd *cobra.Command, args []string) error {
	return runFileDownloadWith(cmd, args, fileDownloadDeps{
		client: activeFileDownloadClient,
		stdout: os.Stdout,
		json:   outputJSON,
	})
}

func activeFileDownloadClient() (api.FileDownloadClient, error) {
	return activeAPIClient()
}

func runFileDownloadWith(cmd *cobra.Command, args []string, deps fileDownloadDeps) error {
	remotePath, err := normalizeRemoteDownloadPath(args[0])
	if err != nil {
		return err
	}
	if len(strings.Split(remotePath, "/")) < 2 {
		return invalidFilePathIssue(i18n.T("err_file_path_not_file", args[0]))
	}

	localPath, err := resolveLocalDownloadPath(remotePath, args)
	if err != nil {
		return err
	}
	overwrite, _ := cmd.Flags().GetBool("overwrite")
	// Let cancellation unwind the download and remove its temporary file.
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()
	result, err := downloadToLocal(ctx, deps.client, fileDownloadInput{
		remotePath: remotePath,
		localPath:  localPath,
		overwrite:  overwrite,
	})
	if err != nil {
		return err
	}
	return writeFileDownloadResult(deps.stdout, deps.json, result)
}

func resolveLocalDownloadPath(remotePath string, args []string) (string, error) {
	if len(args) == 1 {
		return pathpkg.Base(remotePath), nil
	}
	raw := args[1]
	if raw == "" || os.IsPathSeparator(raw[len(raw)-1]) {
		return "", invalidLocalPathIssue(i18n.T("err_file_local_path_invalid", raw))
	}
	cleaned := filepath.Clean(raw)
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return "", invalidLocalPathIssue(i18n.T("err_file_local_path_invalid", raw))
	}
	return cleaned, nil
}

func downloadToLocal(
	ctx context.Context,
	clientFactory func() (api.FileDownloadClient, error),
	input fileDownloadInput,
) (fileDownloadResult, error) {
	existed, err := inspectDownloadTarget(input.localPath)
	if err != nil {
		return fileDownloadResult{}, err
	}
	if existed && !input.overwrite {
		return fileDownloadResult{}, invalidLocalPathIssue(i18n.T("err_file_local_exists", input.localPath))
	}

	parent := filepath.Dir(input.localPath)
	prefix := "." + filepath.Base(input.localPath) + ".crater-"
	temporary, err := os.CreateTemp(parent, prefix)
	if err != nil {
		return fileDownloadResult{}, localFileError("err_file_local_temp", input.localPath, err)
	}
	temporaryPath := temporary.Name()
	temporaryOpen := true
	defer func() {
		if temporaryOpen {
			_ = temporary.Close()
		}
		_ = os.Remove(temporaryPath)
	}()

	client, err := clientFactory()
	if err != nil {
		return fileDownloadResult{}, err
	}
	written, err := client.DownloadFile(ctx, input.remotePath, temporary)
	if err != nil {
		var destinationErr *api.DestinationWriteError
		if errors.As(err, &destinationErr) {
			return fileDownloadResult{}, localFileError("err_file_local_write", input.localPath, destinationErr.Cause)
		}
		return fileDownloadResult{}, cliErrFromAPI(err)
	}
	if err := ctx.Err(); err != nil {
		return fileDownloadResult{}, cliErrFromAPI(&api.NetworkError{Cause: err})
	}
	if err := temporary.Sync(); err != nil {
		return fileDownloadResult{}, localFileError("err_file_local_sync", input.localPath, err)
	}
	if err := temporary.Close(); err != nil {
		temporaryOpen = false
		return fileDownloadResult{}, localFileError("err_file_local_close", input.localPath, err)
	}
	temporaryOpen = false

	if err := ctx.Err(); err != nil {
		return fileDownloadResult{}, cliErrFromAPI(&api.NetworkError{Cause: err})
	}
	if input.overwrite {
		if err := os.Rename(temporaryPath, input.localPath); err != nil {
			return fileDownloadResult{}, localFileError("err_file_local_publish", input.localPath, err)
		}
	} else if err := os.Link(temporaryPath, input.localPath); err != nil {
		if os.IsExist(err) {
			return fileDownloadResult{}, invalidLocalPathIssue(i18n.T("err_file_local_exists", input.localPath))
		}
		return fileDownloadResult{}, localFileError("err_file_local_publish", input.localPath, err)
	}

	return fileDownloadResult{
		RemotePath: input.remotePath,
		LocalPath:  input.localPath,
		Bytes:      written,
		Overwrite:  input.overwrite,
	}, nil
}

func inspectDownloadTarget(localPath string) (bool, error) {
	info, err := os.Lstat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, localFileError("err_file_local_stat", localPath, err)
	}
	if info.IsDir() {
		return false, invalidLocalPathIssue(i18n.T("err_file_local_directory", localPath))
	}
	return true, nil
}

func writeFileDownloadResult(writer io.Writer, jsonOutput bool, result fileDownloadResult) error {
	if jsonOutput {
		return output.WriteSuccessJSON(writer, output.SuccessEnvelope(map[string]interface{}{
			"remote_path": result.RemotePath,
			"local_path":  result.LocalPath,
			"bytes":       result.Bytes,
			"overwrite":   result.Overwrite,
		}))
	}
	_, err := fmt.Fprintln(writer, i18n.T("file_download_success", result.RemotePath, result.LocalPath, result.Bytes))
	if err != nil {
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrCommandExecution,
			Message:  i18n.T("err_file_output", err.Error()),
			Context:  map[string]interface{}{"msg": err.Error()},
		}
	}
	return nil
}

func localFileError(key, localPath string, cause error) *clierror.Error {
	return &clierror.Error{
		Category: errorcodes.CategorySystem,
		Code:     errorcodes.ErrCommandExecution,
		Message:  i18n.T(key, localPath, cause.Error()),
		Context: map[string]interface{}{
			"path": localPath,
			"msg":  cause.Error(),
		},
	}
}

func normalizeRemoteDownloadPath(remotePath string) (string, error) {
	normalized, err := normalizeRemotePathWithIssue(remotePath, invalidFilePathIssue)
	if err != nil {
		return "", err
	}
	if normalized == "" {
		return "", invalidFilePathIssue(i18n.T("err_file_path_invalid", remotePath))
	}
	return normalized, nil
}

func invalidFilePathIssue(message string) error {
	return errUsageFromIssues([]usageIssue{{
		Code:    errorcodes.ErrInvalidFlagValue,
		Message: message,
		Field:   "remote-file",
	}})
}

func invalidLocalPathIssue(message string) error {
	return errUsageFromIssues([]usageIssue{{
		Code:    errorcodes.ErrInvalidFlagValue,
		Message: message,
		Field:   "local-path",
	}})
}

func init() {
	fileDownloadCmd.Flags().Bool("overwrite", false, "Replace an existing local file")
	fileCmd.AddCommand(fileDownloadCmd)
	completion.RegisterPositional([]string{"file", "download"}, 0, fileRootCompleter)
}
