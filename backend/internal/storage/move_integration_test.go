//go:build integration && linux

package storage

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/sys/unix"
)

// Run with a Linux CLI binary and TMPDIR on the shared filesystem under test:
// CRATER_TEST_CLI_BIN=/path/to/crater TMPDIR=/mounted/pvc/scratch go test -tags integration ./internal/storage -run TestMoveClientsOnSharedFilesystem -v
// Authentication is isolated through the existing test fixtures; requests run
// the real move handler and filesystem implementation without deploying a server.
func TestMoveClientsOnSharedFilesystem(t *testing.T) {
	cliBinary := os.Getenv("CRATER_TEST_CLI_BIN")
	if cliBinary == "" {
		t.Skip("set CRATER_TEST_CLI_BIN to a Linux CLI binary")
	}
	storage := newMoveHandlerStorage(t)
	var filesystem unix.Statfs_t
	if err := unix.Statfs(storage, &filesystem); err != nil {
		t.Fatal(err)
	}
	t.Logf("shared filesystem type: %#x", filesystem.Type)
	probeMoveNoReplaceSupport(t, storage)
	deps := testMoveHandlerDeps(storage)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/ss/move/*path", func(c *gin.Context) { moveFileWithDeps(c, deps) })
	server := httptest.NewServer(router)
	defer server.Close()

	home := t.TempDir()
	environment := []string{
		"HOME=" + home, "XDG_CONFIG_HOME=" + home,
		"CRATER_LANG=en", "CRATER_TEST_SANDBOX=1",
		"CRATER_TEST_SANDBOX_HTTP=passthrough",
		"CRATER_TEST_SANDBOX_PLATFORM_URL=" + server.URL,
	}
	userRoot := filepath.Join(storage, "users", "alice")
	for _, client := range []string{"cli", "web"} {
		for _, directory := range []bool{false, true} {
			kind := "file"
			if directory {
				kind = "directory"
			}
			t.Run(client+"/"+kind, func(t *testing.T) {
				name := client + "-" + kind
				source := filepath.Join(userRoot, name)
				if directory {
					makeRemoveIdentityTestDirectory(t, source, "payload")
				} else if err := os.WriteFile(source, []byte("payload"), 0o600); err != nil {
					t.Fatal(err)
				}
				sourcePath, destinationPath := "user/"+name, "user/archive/"+name
				destination := filepath.Join(userRoot, "archive", name)
				moveWithIntegrationClient(t, cliBinary, environment, server.URL, client, sourcePath, destinationPath, false)
				if _, err := os.Lstat(source); !os.IsNotExist(err) {
					t.Fatalf("source still exists: %v", err)
				}
				contentPath := destination
				if directory {
					contentPath = filepath.Join(destination, "keep.txt")
				}
				assertStoredFile(t, contentPath, []byte("payload"))
				if err := os.WriteFile(source, []byte("do not overwrite"), 0o600); err != nil {
					t.Fatal(err)
				}
				moveWithIntegrationClient(t, cliBinary, environment, server.URL, client, sourcePath, destinationPath, true)
				assertStoredFile(t, source, []byte("do not overwrite"))
				assertStoredFile(t, contentPath, []byte("payload"))
			})
		}
	}
}

func moveWithIntegrationClient(t *testing.T, binary string, environment []string, serverURL, client, source, destination string, conflict bool) {
	t.Helper()
	if client == "cli" {
		command := exec.Command(binary, "file", "mv", source, destination, "--json", "--no-interactive")
		command.Env = environment
		output, err := command.CombinedOutput()
		var envelope struct {
			Status  string         `json:"status"`
			Data    map[string]any `json:"data"`
			Context struct {
				HTTPStatus int `json:"http_status"`
			} `json:"context"`
		}
		if decodeErr := json.Unmarshal(output, &envelope); decodeErr != nil {
			t.Fatalf("CLI response: %v output=%s", decodeErr, output)
		}
		if conflict {
			if err == nil || envelope.Context.HTTPStatus != http.StatusConflict {
				t.Fatalf("CLI conflict: error=%v output=%s", err, output)
			}
		} else if err != nil || envelope.Status != "OK" || envelope.Data["source_path"] != source || envelope.Data["destination_path"] != destination {
			t.Fatalf("CLI move: error=%v output=%s", err, output)
		}
		return
	}
	requestBody, err := json.Marshal(MoveFileReq{Dst: destination})
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(serverURL+"/api/ss/move/"+source, "application/json", strings.NewReader(string(requestBody)))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	wantStatus := http.StatusOK
	if conflict {
		wantStatus = http.StatusConflict
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("Web move: status=%d want=%d body=%s", response.StatusCode, wantStatus, body)
	}
}

func probeMoveNoReplaceSupport(t *testing.T, storage string) {
	t.Helper()
	source, destination := filepath.Join(storage, "probe-source"), filepath.Join(storage, "probe-destination")
	if err := os.WriteFile(source, []byte("probe"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := unix.Renameat2(unix.AT_FDCWD, source, unix.AT_FDCWD, destination, unix.RENAME_NOREPLACE)
	t.Logf("native RENAME_NOREPLACE probe: %v", err)
}
