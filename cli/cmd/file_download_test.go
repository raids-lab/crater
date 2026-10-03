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

type fakeFileDownloadClient struct {
	download func(context.Context, string, io.Writer) (int64, error)
}

func (client fakeFileDownloadClient) DownloadFile(ctx context.Context, remotePath string, destination io.Writer) (int64, error) {
	return client.download(ctx, remotePath, destination)
}

func testFileDownloadCommand(t *testing.T, overwrite bool) *cobra.Command {
	t.Helper()
	command := &cobra.Command{Use: "download"}
	command.SetContext(context.Background())
	command.Flags().Bool("overwrite", false, "")
	if overwrite {
		if err := command.Flags().Set("overwrite", "true"); err != nil {
			t.Fatal(err)
		}
	}
	return command
}

func TestNormalizeRemoteDownloadPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "nested file", input: "user/results/model.bin", want: "user/results/model.bin"},
		{name: "unicode and spaces", input: "/account/实验 data/结果.bin/", want: "account/实验 data/结果.bin"},
		{name: "normalizes safe segments", input: "public//runs/./out.bin", want: "public/runs/out.bin"},
		{name: "empty", wantErr: true},
		{name: "slash root", input: "///", wantErr: true},
		{name: "dot root", input: ".", wantErr: true},
		{name: "normalized dot root", input: "/././", wantErr: true},
		{name: "unknown root", input: "admin/secret.bin", wantErr: true},
		{name: "parent traversal", input: "user/../public/secret.bin", wantErr: true},
		{name: "backslash", input: `user\secret.bin`, wantErr: true},
		{name: "control character", input: "user/file\nname", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeRemoteDownloadPath(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("normalizeRemotePath(%q) error = %v, wantErr %v", test.input, err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("normalizeRemotePath(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestResolveLocalDownloadPath(t *testing.T) {
	if got, err := resolveLocalDownloadPath("user/results/model.bin", []string{"user/results/model.bin"}); err != nil || got != "model.bin" {
		t.Fatalf("default local path = %q, %v", got, err)
	}
	if got, err := resolveLocalDownloadPath("user/results/model.bin", []string{"user/results/model.bin", "downloads/result.bin"}); err != nil || got != filepath.Join("downloads", "result.bin") {
		t.Fatalf("explicit local path = %q, %v", got, err)
	}
	if _, err := resolveLocalDownloadPath("user/results/model.bin", []string{"user/results/model.bin", ""}); err == nil {
		t.Fatal("empty local path should fail")
	}
}

func TestRunFileDownloadRejectsInvalidRemoteBeforeClient(t *testing.T) {
	factoryCalls := 0
	err := runFileDownloadWith(
		testFileDownloadCommand(t, false),
		[]string{"user/../public/secret.bin"},
		fileDownloadDeps{
			client: func() (api.FileDownloadClient, error) {
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

func TestRunFileDownloadRejectsLogicalRootAsFile(t *testing.T) {
	factoryCalls := 0
	err := runFileDownloadWith(
		testFileDownloadCommand(t, false),
		[]string{"user"},
		fileDownloadDeps{
			client: func() (api.FileDownloadClient, error) {
				factoryCalls++
				return nil, errors.New("must not be called")
			},
			stdout: io.Discard,
		},
	)
	if err == nil {
		t.Fatal("logical root should not be downloadable as a file")
	}
	if factoryCalls != 0 {
		t.Fatalf("factory calls = %d, want 0", factoryCalls)
	}
}

func TestRunFileDownloadExistingTargetNeedsOverwriteBeforeClient(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "result.bin")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	factoryCalls := 0
	err := runFileDownloadWith(
		testFileDownloadCommand(t, false),
		[]string{"user/results/result.bin", target},
		fileDownloadDeps{
			client: func() (api.FileDownloadClient, error) {
				factoryCalls++
				return nil, errors.New("must not be called")
			},
			stdout: io.Discard,
		},
	)
	if err == nil {
		t.Fatal("existing target should fail without --overwrite")
	}
	if factoryCalls != 0 {
		t.Fatalf("factory calls = %d, want 0", factoryCalls)
	}
	assertFileContent(t, target, []byte("old"))
	assertNoDownloadTemps(t, target)
}

func TestRunFileDownloadOverwriteAndJSONContainsOnlyMetadata(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "result.bin")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	payload := []byte{0x00, 0xff, 'N', 'E', 'W'}
	var stdout bytes.Buffer
	err := runFileDownloadWith(
		testFileDownloadCommand(t, true),
		[]string{"user/results/result.bin", target},
		fileDownloadDeps{
			client: func() (api.FileDownloadClient, error) {
				return fakeFileDownloadClient{download: func(_ context.Context, remotePath string, destination io.Writer) (int64, error) {
					if remotePath != "user/results/result.bin" {
						t.Fatalf("remote path = %q", remotePath)
					}
					written, err := destination.Write(payload)
					return int64(written), err
				}}, nil
			},
			stdout: &stdout,
			json:   true,
		},
	)
	if err != nil {
		t.Fatalf("runFileDownloadWith: %v", err)
	}
	assertFileContent(t, target, payload)
	assertNoDownloadTemps(t, target)
	if bytes.Contains(stdout.Bytes(), payload) {
		t.Fatalf("stdout contains binary payload: %q", stdout.Bytes())
	}
	var envelope struct {
		Status string `json:"status"`
		Data   struct {
			RemotePath string `json:"remote_path"`
			LocalPath  string `json:"local_path"`
			Bytes      int64  `json:"bytes"`
			Overwrite  bool   `json:"overwrite"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
	}
	if envelope.Status != "OK" || envelope.Data.RemotePath != "user/results/result.bin" ||
		envelope.Data.LocalPath != target || envelope.Data.Bytes != int64(len(payload)) || !envelope.Data.Overwrite {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestRunFileDownloadDefaultPublishesWithoutOverwrite(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "result.bin")
	payload := []byte("new download")
	err := runFileDownloadWith(
		testFileDownloadCommand(t, false),
		[]string{"user/results/result.bin", target},
		fileDownloadDeps{
			client: func() (api.FileDownloadClient, error) {
				return fakeFileDownloadClient{download: func(_ context.Context, _ string, destination io.Writer) (int64, error) {
					written, writeErr := destination.Write(payload)
					return int64(written), writeErr
				}}, nil
			},
			stdout: io.Discard,
		},
	)
	if err != nil {
		t.Fatalf("runFileDownloadWith: %v", err)
	}
	assertFileContent(t, target, payload)
	assertNoDownloadTemps(t, target)
}

func TestRunFileDownloadFailureKeepsOldTargetAndCleansTemp(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "result.bin")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("connection reset")
	err := runFileDownloadWith(
		testFileDownloadCommand(t, true),
		[]string{"user/results/result.bin", target},
		fileDownloadDeps{
			client: func() (api.FileDownloadClient, error) {
				return fakeFileDownloadClient{download: func(_ context.Context, _ string, destination io.Writer) (int64, error) {
					written, writeErr := destination.Write([]byte("partial"))
					if writeErr != nil {
						return int64(written), writeErr
					}
					return int64(written), &api.NetworkError{Cause: sentinel}
				}}, nil
			},
			stdout: io.Discard,
		},
	)
	if err == nil {
		t.Fatal("partial network failure should fail")
	}
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) || cliErr.Category != errorcodes.CategoryAPI {
		t.Fatalf("error = %T %v, want API cli error", err, err)
	}
	assertFileContent(t, target, []byte("old"))
	assertNoDownloadTemps(t, target)
}

func TestRunFileDownloadDestinationWriteErrorIsSystemError(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "result.bin")
	sentinel := errors.New("disk full")
	err := runFileDownloadWith(
		testFileDownloadCommand(t, false),
		[]string{"user/results/result.bin", target},
		fileDownloadDeps{
			client: func() (api.FileDownloadClient, error) {
				return fakeFileDownloadClient{download: func(context.Context, string, io.Writer) (int64, error) {
					return 0, &api.DestinationWriteError{Cause: sentinel}
				}}, nil
			},
			stdout: io.Discard,
		},
	)
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) || cliErr.Category != errorcodes.CategorySystem {
		t.Fatalf("error = %T %v, want system cli error", err, err)
	}
	assertNoDownloadTemps(t, target)
}

func TestRunFileDownloadNoOverwriteDetectsLateConflict(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "result.bin")
	err := runFileDownloadWith(
		testFileDownloadCommand(t, false),
		[]string{"user/results/result.bin", target},
		fileDownloadDeps{
			client: func() (api.FileDownloadClient, error) {
				return fakeFileDownloadClient{download: func(_ context.Context, _ string, destination io.Writer) (int64, error) {
					written, writeErr := destination.Write([]byte("download"))
					if writeErr != nil {
						return int64(written), writeErr
					}
					if err := os.WriteFile(target, []byte("racer"), 0o600); err != nil {
						t.Fatal(err)
					}
					return int64(written), nil
				}}, nil
			},
			stdout: io.Discard,
		},
	)
	if err == nil {
		t.Fatal("late target conflict should fail")
	}
	assertFileContent(t, target, []byte("racer"))
	assertNoDownloadTemps(t, target)
}

func TestFileRemoteRootCompleter(t *testing.T) {
	candidates, err := fileRootCompleter(completion.Context{
		Words:   []string{"crater", "file", "download", "a"},
		Current: 4,
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

func assertFileContent(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s content = %v, want %v", path, got, want)
	}
}

func assertNoDownloadTemps(t *testing.T, target string) {
	t.Helper()
	pattern := filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".crater-*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %#v", matches)
	}
}
