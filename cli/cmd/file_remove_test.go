package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

type fakeFileRemoveClient struct {
	removeFile func(context.Context, string, bool) (api.FileRemoveResult, error)
}

func (client fakeFileRemoveClient) RemoveFile(
	ctx context.Context,
	remotePath string,
	recursive bool,
) (api.FileRemoveResult, error) {
	return client.removeFile(ctx, remotePath, recursive)
}

func newFileRemoveTestCommand(t *testing.T, recursive, yes bool) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "rm"}
	cmd.Flags().Bool("recursive", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")
	if recursive {
		if err := cmd.Flags().Set("recursive", "true"); err != nil {
			t.Fatal(err)
		}
	}
	if yes {
		if err := cmd.Flags().Set("yes", "true"); err != nil {
			t.Fatal(err)
		}
	}
	return cmd
}

func TestFileRemoveArgs(t *testing.T) {
	cmd := &cobra.Command{Use: "rm"}
	if err := fileRemoveArgs(cmd, nil); err == nil {
		t.Fatal("missing remote path should fail")
	}
	if err := fileRemoveArgs(cmd, []string{"user/results.txt"}); err != nil {
		t.Fatalf("valid arguments: %v", err)
	}
	if err := fileRemoveArgs(cmd, []string{"user/a", "user/b"}); err == nil {
		t.Fatal("extra argument should fail")
	}
}

func TestRunFileRemoveConfirmsNormalizedPathBeforeCreatingClient(t *testing.T) {
	i18n.SetLanguage("en")
	var stdout bytes.Buffer
	var events []string
	var gotPath string
	var gotRecursive bool
	cmd := newFileRemoveTestCommand(t, true, false)

	err := runFileRemoveWith(
		context.Background(),
		cmd,
		[]string{"/user//runs/result file.txt/"},
		fileRemoveDeps{
			confirm: func(message string) (bool, error) {
				events = append(events, "confirm")
				if message != `Recursively remove remote entry "user/runs/result file.txt"?` {
					t.Fatalf("confirmation = %q", message)
				}
				return true, nil
			},
			client: func() (api.FileRemoveClient, error) {
				events = append(events, "client")
				return fakeFileRemoveClient{
					removeFile: func(
						_ context.Context,
						remotePath string,
						recursive bool,
					) (api.FileRemoveResult, error) {
						events = append(events, "remove")
						gotPath = remotePath
						gotRecursive = recursive
						return api.FileRemoveResult{
							RemotePath: remotePath,
							Recursive:  recursive,
						}, nil
					},
				}, nil
			},
			stdout: &stdout,
		},
	)
	if err != nil {
		t.Fatalf("runFileRemoveWith: %v", err)
	}
	if strings.Join(events, ",") != "confirm,client,remove" {
		t.Fatalf("events = %#v", events)
	}
	if gotPath != "user/runs/result file.txt" || !gotRecursive {
		t.Fatalf("path = %q recursive = %t", gotPath, gotRecursive)
	}
	if !strings.Contains(stdout.String(), gotPath) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFileRemoveYesSkipsConfirmationAndWritesJSON(t *testing.T) {
	var stdout bytes.Buffer
	confirmCalls := 0
	cmd := newFileRemoveTestCommand(t, false, true)

	err := runFileRemoveWith(
		context.Background(),
		cmd,
		[]string{"account/output.txt"},
		fileRemoveDeps{
			confirm: func(string) (bool, error) {
				confirmCalls++
				return false, errors.New("must not be called")
			},
			client: func() (api.FileRemoveClient, error) {
				return fakeFileRemoveClient{
					removeFile: func(
						_ context.Context,
						remotePath string,
						recursive bool,
					) (api.FileRemoveResult, error) {
						return api.FileRemoveResult{
							RemotePath: remotePath,
							Recursive:  recursive,
						}, nil
					},
				}, nil
			},
			stdout: &stdout,
			json:   true,
		},
	)
	if err != nil {
		t.Fatalf("runFileRemoveWith: %v", err)
	}
	if confirmCalls != 0 {
		t.Fatalf("confirmation calls = %d", confirmCalls)
	}
	var envelope struct {
		Status string `json:"status"`
		Data   struct {
			RemotePath string `json:"remote_path"`
			Recursive  bool   `json:"recursive"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
	}
	if envelope.Status != "OK" ||
		envelope.Data.RemotePath != "account/output.txt" ||
		envelope.Data.Recursive {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestRunFileRemoveNonInteractiveRequiresYesBeforeClient(t *testing.T) {
	for _, test := range []struct {
		name          string
		json          bool
		noInteractive bool
	}{
		{name: "JSON", json: true},
		{name: "no interactive", noInteractive: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			clientCalls := 0
			confirmCalls := 0
			err := runFileRemoveWith(
				context.Background(),
				newFileRemoveTestCommand(t, false, false),
				[]string{"user/output.txt"},
				fileRemoveDeps{
					client: func() (api.FileRemoveClient, error) {
						clientCalls++
						return nil, errors.New("must not be called")
					},
					confirm: func(string) (bool, error) {
						confirmCalls++
						return true, nil
					},
					stdout:        io.Discard,
					json:          test.json,
					noInteractive: test.noInteractive,
				},
			)
			var cliErr *clierror.Error
			if !errors.As(err, &cliErr) {
				t.Fatalf("error = %T %v, want *clierror.Error", err, err)
			}
			if cliErr.Category != errorcodes.CategoryUsage ||
				cliErr.Code != errorcodes.ErrMissingRequiredFlag ||
				cliErr.Message != i18n.T("err_confirm_required") {
				t.Fatalf("error = %#v", cliErr)
			}
			if exitCodeFor(err) != errorcodes.ExitUsage {
				t.Fatalf("exit code = %d", exitCodeFor(err))
			}
			if clientCalls != 0 || confirmCalls != 0 {
				t.Fatalf("client calls = %d, confirmation calls = %d", clientCalls, confirmCalls)
			}
		})
	}
}

func TestRunFileRemoveCancellationNeverCreatesClient(t *testing.T) {
	for _, test := range []struct {
		name    string
		confirm func(string) (bool, error)
	}{
		{
			name: "No",
			confirm: func(string) (bool, error) {
				return false, nil
			},
		},
		{
			name: "Ctrl-C",
			confirm: func(string) (bool, error) {
				return false, terminal.InterruptErr
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			clientCalls := 0
			var stdout bytes.Buffer
			err := runFileRemoveWith(
				context.Background(),
				newFileRemoveTestCommand(t, false, false),
				[]string{"public/output.txt"},
				fileRemoveDeps{
					client: func() (api.FileRemoveClient, error) {
						clientCalls++
						return nil, errors.New("must not be called")
					},
					confirm: test.confirm,
					stdout:  &stdout,
				},
			)
			var cliErr *clierror.Error
			if !errors.As(err, &cliErr) ||
				cliErr.Category != errorcodes.CategoryCancelled ||
				cliErr.Code != errorcodes.ErrOperationCancelled {
				t.Fatalf("error = %T %#v, want cancellation", err, cliErr)
			}
			if exitCodeFor(err) != errorcodes.ExitCancelled {
				t.Fatalf("exit code = %d", exitCodeFor(err))
			}
			if clientCalls != 0 {
				t.Fatalf("client calls = %d", clientCalls)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
		})
	}
}

func TestRunFileRemoveRejectsUnsafePathsBeforeConfirmationOrClient(t *testing.T) {
	for _, remotePath := range []string{
		"",
		"/",
		".",
		"..",
		"user",
		"/public/",
		"account/.",
		"user/../public/file",
		"./user/file",
		`user\file`,
		"user/\nfile",
		"admin-user/alice/file",
		"admin-public/file",
		"admin-account/team/file",
	} {
		t.Run(strings.ReplaceAll(remotePath, "/", "_"), func(t *testing.T) {
			confirmCalls := 0
			clientCalls := 0
			err := runFileRemoveWith(
				context.Background(),
				newFileRemoveTestCommand(t, false, true),
				[]string{remotePath},
				fileRemoveDeps{
					confirm: func(string) (bool, error) {
						confirmCalls++
						return true, nil
					},
					client: func() (api.FileRemoveClient, error) {
						clientCalls++
						return nil, errors.New("must not be called")
					},
					stdout: io.Discard,
				},
			)
			var cliErr *clierror.Error
			if !errors.As(err, &cliErr) ||
				cliErr.Category != errorcodes.CategoryUsage ||
				cliErr.Code != errorcodes.ErrInvalidFlagValue {
				t.Fatalf("path %q: error = %T %#v", remotePath, err, cliErr)
			}
			if confirmCalls != 0 || clientCalls != 0 {
				t.Fatalf(
					"path %q: confirmation calls = %d, client calls = %d",
					remotePath,
					confirmCalls,
					clientCalls,
				)
			}
		})
	}
}

func TestRunFileRemoveAPIFailureWritesNoSuccessOutput(t *testing.T) {
	var stdout bytes.Buffer
	err := runFileRemoveWith(
		context.Background(),
		newFileRemoveTestCommand(t, false, true),
		[]string{"user/output.txt"},
		fileRemoveDeps{
			client: func() (api.FileRemoveClient, error) {
				return fakeFileRemoveClient{
					removeFile: func(
						context.Context,
						string,
						bool,
					) (api.FileRemoveResult, error) {
						return api.FileRemoveResult{}, &api.RequestError{
							HTTPStatus: 404,
							CraterCode: 40404,
							Msg:        "storage resource not found",
						}
					},
				}, nil
			},
			confirm: func(string) (bool, error) {
				return true, nil
			},
			stdout: &stdout,
		},
	)
	if err == nil {
		t.Fatal("API failure should fail")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) || cliErr.Category != errorcodes.CategoryAPI {
		t.Fatalf("error = %T %v, want API error", err, err)
	}
}

func TestFileRemoveCommandFlags(t *testing.T) {
	recursive := fileRemoveCmd.Flags().Lookup("recursive")
	if recursive == nil {
		t.Fatal("--recursive is not registered")
	}
	yes := fileRemoveCmd.Flags().Lookup("yes")
	if yes == nil || yes.Shorthand != "y" {
		t.Fatalf("--yes flag = %#v", yes)
	}
}

func TestFileRemoveTranslations(t *testing.T) {
	for _, test := range []struct {
		language  i18n.Language
		short     string
		confirm   string
		recursive string
		success   string
	}{
		{
			language:  i18n.En,
			short:     "Remove one remote file or directory",
			confirm:   `Remove remote entry "user/result.txt"?`,
			recursive: `Recursively remove remote entry "user/result.txt"?`,
			success:   "Removed remote entry user/result.txt",
		},
		{
			language:  i18n.ZhCN,
			short:     "删除单个远端文件或目录",
			confirm:   "确定删除远端条目“user/result.txt”？",
			recursive: "确定递归删除远端条目“user/result.txt”？",
			success:   "已删除远端条目 user/result.txt",
		},
	} {
		t.Run(string(test.language), func(t *testing.T) {
			i18n.SetLanguage(string(test.language))
			if got := i18n.T("file_rm_short"); got != test.short {
				t.Fatalf("short = %q", got)
			}
			if got := i18n.T("file_remove_confirm", "user/result.txt"); got != test.confirm {
				t.Fatalf("confirmation = %q", got)
			}
			if got := i18n.T("file_remove_recursive_confirm", "user/result.txt"); got != test.recursive {
				t.Fatalf("recursive confirmation = %q", got)
			}
			if got := i18n.T("file_remove_success", "user/result.txt"); got != test.success {
				t.Fatalf("success = %q", got)
			}
		})
	}
	i18n.SetLanguage("en")
}
