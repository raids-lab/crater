package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/raids-lab/crater/cli/internal/api"
)

// Exercise SIGINT in a child process so a regression cannot terminate the test runner.
func TestFileDownloadInterruptCleansTemporaryFile(t *testing.T) {
	if os.Getenv("CRATER_DOWNLOAD_INTERRUPT_HELPER") == "1" {
		target := os.Getenv("CRATER_DOWNLOAD_INTERRUPT_TARGET")
		var stdout bytes.Buffer
		err := runFileDownloadWith(testFileDownloadCommand(t, true),
			[]string{"user/result.bin", target}, fileDownloadDeps{
				client: func() (api.FileDownloadClient, error) {
					return fakeFileDownloadClient{download: func(ctx context.Context, _ string, dst io.Writer) (int64, error) {
						n, err := io.WriteString(dst, "partial")
						if err != nil {
							return int64(n), err
						}
						if _, err := fmt.Fprintln(os.Stdout, "ready"); err != nil {
							return int64(n), err
						}
						<-ctx.Done()
						return int64(n), &api.NetworkError{Cause: ctx.Err()}
					}}, nil
				},
				stdout: &stdout,
				json:   true,
			})
		if err == nil || stdout.Len() != 0 {
			t.Fatalf("interrupted download = %v, stdout = %q", err, stdout.String())
		}
		assertNoDownloadTemps(t, target)
		return
	}
	if runtime.GOOS == "windows" {
		t.Skip("os.Process.Signal(os.Interrupt) is unsupported on Windows")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing=%t", existing), func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "result.bin")
			if existing {
				if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, executable, "-test.run=^TestFileDownloadInterruptCleansTemporaryFile$")
			child.Env = append(os.Environ(), "CRATER_DOWNLOAD_INTERRUPT_HELPER=1", "CRATER_DOWNLOAD_INTERRUPT_TARGET="+target)
			var stderr bytes.Buffer
			child.Stderr = &stderr
			pipe, err := child.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := child.Start(); err != nil {
				t.Fatal(err)
			}
			ready, readErr := bufio.NewReader(pipe).ReadString('\n')
			if readErr != nil || ready != "ready\n" {
				cancel()
				waitErr := child.Wait()
				t.Fatalf("child readiness = %q, %v; exit=%v; stderr=%s", ready, readErr, waitErr, stderr.String())
			}
			if err := child.Process.Signal(os.Interrupt); err != nil {
				cancel()
				_ = child.Wait()
				t.Fatal(err)
			}
			if err := child.Wait(); err != nil {
				t.Fatalf("child failed to unwind SIGINT: %v; stderr=%s", err, stderr.String())
			}
			assertNoDownloadTemps(t, target)
			if existing {
				assertFileContent(t, target, []byte("old"))
			} else if _, err := os.Lstat(target); !os.IsNotExist(err) {
				t.Fatalf("target published after interrupt: %v", err)
			}
		})
	}
}

func TestFileDownloadCanceledBeforePublishKeepsOldTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "result.bin")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := downloadToLocal(ctx, func() (api.FileDownloadClient, error) {
		return fakeFileDownloadClient{download: func(_ context.Context, _ string, dst io.Writer) (int64, error) {
			n, err := io.WriteString(dst, "new")
			cancel()
			return int64(n), err
		}}, nil
	}, fileDownloadInput{remotePath: "user/result.bin", localPath: target, overwrite: true})
	if err == nil {
		t.Fatal("canceled download unexpectedly succeeded")
	}
	assertFileContent(t, target, []byte("old"))
	assertNoDownloadTemps(t, target)
}

func TestResolveLocalDownloadPathRejectsDirectorySyntax(t *testing.T) {
	for _, path := range []string{"", ".", "./", "missing/", "missing" + string(filepath.Separator)} {
		if _, err := resolveLocalDownloadPath("user/result.bin", []string{"user/result.bin", path}); err == nil {
			t.Errorf("directory path %q accepted", path)
		}
	}
	// A backslash is a valid filename character on Unix, and a separator on Windows.
	_, err := resolveLocalDownloadPath("user/result.bin", []string{"user/result.bin", "missing\\"})
	if (err != nil) != (runtime.GOOS == "windows") {
		t.Errorf("backslash path error = %v on %s", err, runtime.GOOS)
	}
}
