package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

type fakeFileMutationClient struct {
	createDirectory func(context.Context, string) error
	moveFile        func(context.Context, string, string) error
}

func (client fakeFileMutationClient) CreateDirectory(ctx context.Context, remotePath string) error {
	return client.createDirectory(ctx, remotePath)
}

func (client fakeFileMutationClient) MoveFile(
	ctx context.Context,
	sourcePath string,
	destinationPath string,
) error {
	return client.moveFile(ctx, sourcePath, destinationPath)
}

func TestFileMutationArgs(t *testing.T) {
	mkdir := &cobra.Command{Use: "mkdir"}
	if err := fileMkdirArgs(mkdir, nil); err == nil {
		t.Fatal("missing mkdir path should fail")
	}
	if err := fileMkdirArgs(mkdir, []string{"user/results"}); err != nil {
		t.Fatalf("valid mkdir args: %v", err)
	}
	if err := fileMkdirArgs(mkdir, []string{"a", "b"}); err == nil {
		t.Fatal("extra mkdir arg should fail")
	}

	move := &cobra.Command{Use: "mv"}
	if err := fileMoveArgs(move, nil); err == nil {
		t.Fatal("missing move source should fail")
	}
	if err := fileMoveArgs(move, []string{"user/source"}); err == nil {
		t.Fatal("missing move destination should fail")
	}
	if err := fileMoveArgs(move, []string{"user/source", "user/destination"}); err != nil {
		t.Fatalf("valid move args: %v", err)
	}
	if err := fileMoveArgs(move, []string{"a", "b", "c"}); err == nil {
		t.Fatal("extra move arg should fail")
	}
}

func TestRunFileMkdirNormalizesPathAndWritesHumanOutput(t *testing.T) {
	var stdout bytes.Buffer
	var gotPath string
	err := runFileMkdirWith(
		context.Background(),
		[]string{"/user//runs/./new directory/"},
		fileMutationDeps{
			client: func() (api.FileMutationClient, error) {
				return fakeFileMutationClient{
					createDirectory: func(_ context.Context, remotePath string) error {
						gotPath = remotePath
						return nil
					},
				}, nil
			},
			stdout: &stdout,
		},
	)
	if err != nil {
		t.Fatalf("runFileMkdirWith: %v", err)
	}
	if gotPath != "user/runs/new directory" {
		t.Fatalf("remote path = %q", gotPath)
	}
	if !strings.Contains(stdout.String(), gotPath) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunFileMkdirWritesJSON(t *testing.T) {
	var stdout bytes.Buffer
	err := runFileMkdirWith(
		context.Background(),
		[]string{"account/results"},
		fileMutationDeps{
			client: func() (api.FileMutationClient, error) {
				return fakeFileMutationClient{
					createDirectory: func(context.Context, string) error { return nil },
				}, nil
			},
			stdout: &stdout,
			json:   true,
		},
	)
	if err != nil {
		t.Fatalf("runFileMkdirWith: %v", err)
	}
	var envelope struct {
		Status string `json:"status"`
		Data   struct {
			RemotePath string `json:"remote_path"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
	}
	if envelope.Status != "OK" || envelope.Data.RemotePath != "account/results" {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestRunFileMoveNormalizesPathsAndWritesJSON(t *testing.T) {
	var stdout bytes.Buffer
	var gotSource, gotDestination string
	err := runFileMoveWith(
		context.Background(),
		[]string{"/user//runs/./source", "account/results/目标 file"},
		fileMutationDeps{
			client: func() (api.FileMutationClient, error) {
				return fakeFileMutationClient{
					moveFile: func(_ context.Context, sourcePath, destinationPath string) error {
						gotSource = sourcePath
						gotDestination = destinationPath
						return nil
					},
				}, nil
			},
			stdout: &stdout,
			json:   true,
		},
	)
	if err != nil {
		t.Fatalf("runFileMoveWith: %v", err)
	}
	if gotSource != "user/runs/source" || gotDestination != "account/results/目标 file" {
		t.Fatalf("source=%q destination=%q", gotSource, gotDestination)
	}
	var envelope struct {
		Status string `json:"status"`
		Data   struct {
			Source      string `json:"source_path"`
			Destination string `json:"destination_path"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
	}
	if envelope.Status != "OK" ||
		envelope.Data.Source != gotSource ||
		envelope.Data.Destination != gotDestination {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestFileMutationRejectsInvalidPathsBeforeClient(t *testing.T) {
	for _, test := range []struct {
		name string
		run  func(fileMutationDeps) error
	}{
		{
			name: "mkdir logical root",
			run: func(deps fileMutationDeps) error {
				return runFileMkdirWith(context.Background(), []string{"user"}, deps)
			},
		},
		{
			name: "move same path",
			run: func(deps fileMutationDeps) error {
				return runFileMoveWith(
					context.Background(),
					[]string{"user/source", "/user//./source"},
					deps,
				)
			},
		},
		{
			name: "move below source",
			run: func(deps fileMutationDeps) error {
				return runFileMoveWith(
					context.Background(),
					[]string{"user/source", "user/source/nested"},
					deps,
				)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			factoryCalls := 0
			err := test.run(fileMutationDeps{
				client: func() (api.FileMutationClient, error) {
					factoryCalls++
					return nil, errors.New("must not be called")
				},
				stdout: io.Discard,
			})
			if err == nil {
				t.Fatal("invalid path should fail")
			}
			if factoryCalls != 0 {
				t.Fatalf("factory calls = %d, want 0", factoryCalls)
			}
		})
	}
}

func TestRunFileMoveAggregatesInvalidOperandsBeforeClient(t *testing.T) {
	factoryCalls := 0
	err := runFileMoveWith(
		context.Background(),
		[]string{"user/../public/source", `account\destination`},
		fileMutationDeps{
			client: func() (api.FileMutationClient, error) {
				factoryCalls++
				return nil, errors.New("must not be called")
			},
			stdout: io.Discard,
		},
	)
	if factoryCalls != 0 {
		t.Fatalf("factory calls = %d, want 0", factoryCalls)
	}
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) || cliErr.Category != errorcodes.CategoryUsage {
		t.Fatalf("error = %T %v, want usage error", err, err)
	}
	issues, ok := cliErr.Context["issues"].([]map[string]interface{})
	if !ok || len(issues) != 2 {
		t.Fatalf("issues = %#v, want two", cliErr.Context["issues"])
	}
	if issues[0]["field"] != "source-path" || issues[1]["field"] != "destination-path" {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestFileMutationAPIFailureWritesNoSuccessOutput(t *testing.T) {
	var stdout bytes.Buffer
	err := runFileMoveWith(
		context.Background(),
		[]string{"user/source", "user/destination"},
		fileMutationDeps{
			client: func() (api.FileMutationClient, error) {
				return fakeFileMutationClient{
					moveFile: func(context.Context, string, string) error {
						return &api.RequestError{
							HTTPStatus: 409,
							CraterCode: 40901,
							Msg:        "destination path already exists",
						}
					},
				}, nil
			},
			stdout: &stdout,
		},
	)
	if err == nil {
		t.Fatal("API conflict should fail")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) || cliErr.Category != errorcodes.CategoryAPI {
		t.Fatalf("error = %T %v, want API error", err, err)
	}
}

func TestFileMutationRootCompletion(t *testing.T) {
	i18n.SetLanguage("en")
	candidates, err := fileRootCompleter(completion.Context{
		Words:   []string{"crater", "file", "mv", "user/source", "p"},
		Current: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(candidates))
	for index := range candidates {
		got[index] = candidates[index].Value
	}
	if !reflect.DeepEqual(got, []string{"public"}) {
		t.Fatalf("candidate values = %#v", got)
	}
}
