package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
)

func TestFileDownloadRejectedResponseDoesNotPublish(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusPartialContent} {
		for _, overwrite := range []bool{false, true} {
			t.Run(fmt.Sprintf("status=%d/overwrite=%t", status, overwrite), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(status)
					if status == http.StatusPartialContent {
						_, _ = io.WriteString(w, "partial")
					}
				}))
				t.Cleanup(server.Close)
				target := filepath.Join(t.TempDir(), "result.bin")
				if overwrite {
					if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				var stdout bytes.Buffer
				err := runFileDownloadWith(testFileDownloadCommand(t, overwrite),
					[]string{"user/result.bin", target}, fileDownloadDeps{
						client: func() (api.FileDownloadClient, error) { return api.NewClient(server.URL), nil },
						stdout: &stdout,
						json:   true,
					})
				if err == nil || stdout.Len() != 0 {
					t.Fatalf("rejected download = %v, stdout=%q", err, stdout.String())
				}
				assertNoDownloadTemps(t, target)
				if overwrite {
					assertFileContent(t, target, []byte("old"))
				} else if _, err := os.Lstat(target); !os.IsNotExist(err) {
					t.Fatalf("target published after rejected response: %v", err)
				}
			})
		}
	}
}
