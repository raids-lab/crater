package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

type fakeFileUploadClient struct {
	upload func(context.Context, string, io.Reader, bool) (api.FileUploadResult, error)
}

func (client fakeFileUploadClient) UploadFile(
	ctx context.Context,
	remotePath string,
	source io.Reader,
	overwrite bool,
) (api.FileUploadResult, error) {
	return client.upload(ctx, remotePath, source, overwrite)
}

func testFileUploadCommand(t *testing.T, overwrite bool) *cobra.Command {
	t.Helper()
	command := &cobra.Command{Use: "upload"}
	command.Flags().Bool("overwrite", false, "")
	if overwrite {
		if err := command.Flags().Set("overwrite", "true"); err != nil {
			t.Fatal(err)
		}
	}
	return command
}

func TestFileUploadArgs(t *testing.T) {
	command := &cobra.Command{Use: "upload"}
	if err := fileUploadArgs(command, nil); err == nil {
		t.Fatal("missing local file should fail")
	}
	if err := fileUploadArgs(command, []string{"local.bin"}); err == nil {
		t.Fatal("missing remote file should fail")
	}
	if err := fileUploadArgs(command, []string{"local.bin", "user/local.bin"}); err != nil {
		t.Fatalf("valid args: %v", err)
	}
	if err := fileUploadArgs(command, []string{"a", "b", "c"}); err == nil {
		t.Fatal("extra args should fail")
	}
}

func TestNormalizeRemoteUploadPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "nested file", input: "user/results/model.bin", want: "user/results/model.bin"},
		{name: "unicode and spaces", input: "/account/实验 data/结果.bin/", want: "account/实验 data/结果.bin"},
		{name: "safe normalization", input: "public//runs/./out.bin", want: "public/runs/out.bin"},
		{name: "empty", wantErr: true},
		{name: "unknown root", input: "admin/secret.bin", wantErr: true},
		{name: "parent traversal", input: "user/../public/secret.bin", wantErr: true},
		{name: "backslash", input: `user\secret.bin`, wantErr: true},
		{name: "control character", input: "user/file\nname", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeRemoteUploadPath(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("normalizeRemotePath(%q) error = %v, wantErr %v", test.input, err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("normalizeRemotePath(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestRunFileUploadRejectsInvalidRemoteBeforeClient(t *testing.T) {
	factoryCalls := 0
	err := runFileUploadWith(
		testFileUploadCommand(t, false),
		[]string{"does-not-need-to-exist", "user/../public/secret.bin"},
		fileUploadDeps{
			client: func() (api.FileUploadClient, error) {
				factoryCalls++
				return nil, errors.New("must not be called")
			},
			stdout: io.Discard,
		},
	)
	if err == nil {
		t.Fatal("invalid remote path should fail")
	}
	if factoryCalls != 0 {
		t.Fatalf("factory calls = %d, want 0", factoryCalls)
	}
}

func TestRunFileUploadRejectsLogicalRootBeforeClient(t *testing.T) {
	local := writeUploadFixture(t, []byte("data"))
	factoryCalls := 0
	err := runFileUploadWith(
		testFileUploadCommand(t, false),
		[]string{local, "user"},
		fileUploadDeps{
			client: func() (api.FileUploadClient, error) {
				factoryCalls++
				return nil, errors.New("must not be called")
			},
			stdout: io.Discard,
		},
	)
	if err == nil {
		t.Fatal("logical root should not be an upload target")
	}
	if factoryCalls != 0 {
		t.Fatalf("factory calls = %d, want 0", factoryCalls)
	}
}

func TestRunFileUploadRejectsNonRegularSourceBeforeClient(t *testing.T) {
	factoryCalls := 0
	err := runFileUploadWith(
		testFileUploadCommand(t, false),
		[]string{t.TempDir(), "user/result.bin"},
		fileUploadDeps{
			client: func() (api.FileUploadClient, error) {
				factoryCalls++
				return nil, errors.New("must not be called")
			},
			stdout: io.Discard,
		},
	)
	if err == nil {
		t.Fatal("directory source should fail")
	}
	if factoryCalls != 0 {
		t.Fatalf("factory calls = %d, want 0", factoryCalls)
	}
}

func TestRunFileUploadExistingTargetNeedsOverwrite(t *testing.T) {
	local := writeUploadFixture(t, []byte("new"))
	uploadCalls := 0
	err := runFileUploadWith(
		testFileUploadCommand(t, false),
		[]string{local, "user/results/result.bin"},
		fileUploadDeps{
			client: func() (api.FileUploadClient, error) {
				return fakeFileUploadClient{
					upload: func(_ context.Context, remotePath string, _ io.Reader, overwrite bool) (api.FileUploadResult, error) {
						uploadCalls++
						if remotePath != "user/results/result.bin" || overwrite {
							t.Fatalf("remotePath=%q overwrite=%v", remotePath, overwrite)
						}
						return api.FileUploadResult{}, &api.RequestError{
							HTTPStatus: 409,
							CraterCode: 40901,
							Msg:        "target file already exists",
						}
					},
				}, nil
			},
			stdout: io.Discard,
		},
	)
	if err == nil {
		t.Fatal("existing target should fail without --overwrite")
	}
	if uploadCalls != 1 {
		t.Fatalf("upload calls = %d, want 1", uploadCalls)
	}
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) || cliErr.Category != errorcodes.CategoryAPI {
		t.Fatalf("error = %T %v, want API error", err, err)
	}
}

func TestRunFileUploadRejectsRemoteDirectoryEvenWithOverwrite(t *testing.T) {
	local := writeUploadFixture(t, []byte("new"))
	uploadCalls := 0
	err := runFileUploadWith(
		testFileUploadCommand(t, true),
		[]string{local, "public/results"},
		fileUploadDeps{
			client: func() (api.FileUploadClient, error) {
				return fakeFileUploadClient{
					upload: func(context.Context, string, io.Reader, bool) (api.FileUploadResult, error) {
						uploadCalls++
						return api.FileUploadResult{}, &api.RequestError{
							HTTPStatus: 409,
							CraterCode: 40902,
							Msg:        "target path is not a regular file",
						}
					},
				}, nil
			},
			stdout: io.Discard,
		},
	)
	if err == nil {
		t.Fatal("remote directory should fail")
	}
	if uploadCalls != 1 {
		t.Fatalf("upload calls = %d, want 1", uploadCalls)
	}
}

func TestRunFileUploadStreamsBinaryAndWritesJSONMetadata(t *testing.T) {
	payload := []byte{0x00, 0xff, 'C', 'L', 'I'}
	local := writeUploadFixture(t, payload)
	var stdout bytes.Buffer
	err := runFileUploadWith(
		testFileUploadCommand(t, false),
		[]string{local, "/account//实验 data/./result.bin"},
		fileUploadDeps{
			client: func() (api.FileUploadClient, error) {
				return fakeFileUploadClient{
					upload: func(_ context.Context, remotePath string, source io.Reader, overwrite bool) (api.FileUploadResult, error) {
						if remotePath != "account/实验 data/result.bin" {
							t.Fatalf("remote path = %q", remotePath)
						}
						if overwrite {
							t.Fatal("overwrite = true, want false")
						}
						got, readErr := io.ReadAll(source)
						if readErr != nil {
							return api.FileUploadResult{}, readErr
						}
						if !bytes.Equal(got, payload) {
							t.Fatalf("payload = %v, want %v", got, payload)
						}
						return api.FileUploadResult{
							RemotePath: remotePath,
							Bytes:      int64(len(got)),
						}, nil
					},
				}, nil
			},
			stdout: &stdout,
			json:   true,
		},
	)
	if err != nil {
		t.Fatalf("runFileUploadWith: %v", err)
	}
	if bytes.Contains(stdout.Bytes(), payload) {
		t.Fatalf("stdout contains binary payload: %q", stdout.Bytes())
	}
	var envelope struct {
		Status string `json:"status"`
		Data   struct {
			LocalPath   string `json:"local_path"`
			RemotePath  string `json:"remote_path"`
			Bytes       int64  `json:"bytes"`
			Overwrite   bool   `json:"overwrite"`
			Overwritten bool   `json:"overwritten"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
	}
	if envelope.Status != "OK" || envelope.Data.LocalPath != local ||
		envelope.Data.RemotePath != "account/实验 data/result.bin" ||
		envelope.Data.Bytes != int64(len(payload)) || envelope.Data.Overwrite {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestRunFileUploadOverwritePassesExplicitAuthorization(t *testing.T) {
	local := writeUploadFixture(t, []byte("replacement"))
	overwriteSeen := false
	err := runFileUploadWith(
		testFileUploadCommand(t, true),
		[]string{local, "user/result.bin"},
		fileUploadDeps{
			client: func() (api.FileUploadClient, error) {
				return fakeFileUploadClient{
					upload: func(_ context.Context, remotePath string, source io.Reader, overwrite bool) (api.FileUploadResult, error) {
						overwriteSeen = overwrite
						written, readErr := io.Copy(io.Discard, source)
						return api.FileUploadResult{
							RemotePath:  remotePath,
							Bytes:       written,
							Overwritten: true,
						}, readErr
					},
				}, nil
			},
			stdout: io.Discard,
		},
	)
	if err != nil {
		t.Fatalf("runFileUploadWith: %v", err)
	}
	if !overwriteSeen {
		t.Fatal("overwrite flag was not passed to API client")
	}
}

func TestRunFileUploadSourceReadErrorIsSystemError(t *testing.T) {
	sentinel := errors.New("local read failed")
	_, err := uploadRemoteFile(
		context.Background(),
		func() (api.FileUploadClient, error) {
			return fakeFileUploadClient{
				upload: func(context.Context, string, io.Reader, bool) (api.FileUploadResult, error) {
					return api.FileUploadResult{}, &api.SourceReadError{Cause: sentinel}
				},
			}, nil
		},
		bytes.NewReader(nil),
		fileUploadInput{
			localPath:  "local.bin",
			remotePath: "user/remote.bin",
		},
	)
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) || cliErr.Category != errorcodes.CategorySystem {
		t.Fatalf("error = %T %v, want system cli error", err, err)
	}
}

func TestFileUploadRootCompleter(t *testing.T) {
	candidates, err := fileRootCompleter(completion.Context{
		Words:   []string{"crater", "file", "upload", "local.bin", "a"},
		Current: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(candidates))
	for index := range candidates {
		got[index] = candidates[index].Value
	}
	if !reflect.DeepEqual(got, []string{"account"}) {
		t.Fatalf("candidate values = %#v", got)
	}
}

func TestFileLocalPathCompleter(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "alpha.txt"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, ".hidden"), []byte("h"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(directory, "artifacts"), 0o700); err != nil {
		t.Fatal(err)
	}

	prefix := filepath.Join(directory, "a")
	candidates, err := fileLocalPathCompleter(completion.Context{
		Words:   []string{"crater", "file", "upload", prefix},
		Current: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(candidates))
	for index := range candidates {
		got[index] = candidates[index].Value
	}
	want := []string{
		filepath.Join(directory, "alpha.txt"),
		filepath.Join(directory, "artifacts") + string(filepath.Separator),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate values = %#v, want %#v", got, want)
	}

	hiddenPrefix := filepath.Join(directory, ".h")
	candidates, err = fileLocalPathCompleter(completion.Context{
		Words:   []string{"crater", "file", "upload", hiddenPrefix},
		Current: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Value != filepath.Join(directory, ".hidden") {
		t.Fatalf("hidden candidates = %#v", candidates)
	}
}

func writeUploadFixture(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.bin")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
