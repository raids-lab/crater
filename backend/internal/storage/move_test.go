package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
)

func TestMoveStorageEntryMovesFileAndDirectory(t *testing.T) {
	for _, test := range []struct {
		name      string
		makeEntry func(*testing.T, string)
		assert    func(*testing.T, string)
	}{
		{
			name: "file",
			makeEntry: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte("payload"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			assert: func(t *testing.T, path string) {
				t.Helper()
				assertStoredFile(t, path, []byte("payload"))
			},
		},
		{
			name: "directory",
			makeEntry: func(t *testing.T, path string) {
				t.Helper()
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(path, "nested.txt"), []byte("nested"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			assert: func(t *testing.T, path string) {
				t.Helper()
				assertStoredFile(t, filepath.Join(path, "nested.txt"), []byte("nested"))
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			storage := t.TempDir()
			sourceDirectory := filepath.Join(storage, "source")
			destinationDirectory := filepath.Join(storage, "destination")
			if err := os.Mkdir(sourceDirectory, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(destinationDirectory, 0o700); err != nil {
				t.Fatal(err)
			}
			test.makeEntry(t, filepath.Join(sourceDirectory, "entry"))

			sourceRoot, err := os.OpenRoot(sourceDirectory)
			if err != nil {
				t.Fatal(err)
			}
			defer sourceRoot.Close()
			destinationRoot, err := os.OpenRoot(destinationDirectory)
			if err != nil {
				t.Fatal(err)
			}
			defer destinationRoot.Close()

			if err := moveStorageEntry(sourceRoot, "entry", destinationRoot, "renamed"); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(filepath.Join(sourceDirectory, "entry")); !os.IsNotExist(err) {
				t.Fatalf("source still exists: %v", err)
			}
			test.assert(t, filepath.Join(destinationDirectory, "renamed"))
		})
	}
}

func TestMoveStorageEntryDoesNotOverwrite(t *testing.T) {
	storage := t.TempDir()
	if err := os.WriteFile(filepath.Join(storage, "source"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storage, "destination"), []byte("destination"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(storage)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := moveStorageEntry(root, "source", root, "destination"); !errors.Is(err, errMoveTargetExists) {
		t.Fatalf("error = %v, want errMoveTargetExists", err)
	}
	assertStoredFile(t, filepath.Join(storage, "source"), []byte("source"))
	assertStoredFile(t, filepath.Join(storage, "destination"), []byte("destination"))
}

func TestMoveStorageEntryReportsMissingSource(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := moveStorageEntry(root, "missing", root, "destination"); !errors.Is(err, errMoveSourceNotFound) {
		t.Fatalf("error = %v, want errMoveSourceNotFound", err)
	}
}

//nolint:gocyclo // This concurrency test counts winners, conflicts, and remaining sources in one assertion.
func TestMoveStorageEntryRaceHasOneWinner(t *testing.T) {
	storage := t.TempDir()
	const contenders = 8
	for index := 0; index < contenders; index++ {
		name := "source-" + string(rune('a'+index))
		if err := os.WriteFile(filepath.Join(storage, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	roots := make([]*os.Root, contenders)
	t.Cleanup(func() {
		for _, root := range roots {
			if root != nil {
				_ = root.Close()
			}
		}
	})
	for index := range roots {
		root, err := os.OpenRoot(storage)
		if err != nil {
			t.Fatal(err)
		}
		roots[index] = root
	}

	var wait sync.WaitGroup
	wait.Add(contenders)
	start := make(chan struct{})
	errorsSeen := make(chan error, contenders)
	for index := 0; index < contenders; index++ {
		go func() {
			defer wait.Done()
			<-start
			name := "source-" + string(rune('a'+index))
			errorsSeen <- moveStorageEntry(roots[index], name, roots[index], "destination")
		}()
	}
	close(start)
	wait.Wait()
	close(errorsSeen)

	successes := 0
	conflicts := 0
	for err := range errorsSeen {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, errMoveTargetExists):
			conflicts++
		default:
			t.Fatalf("unexpected move error: %v", err)
		}
	}
	if successes != 1 || conflicts != contenders-1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
	if _, err := os.Stat(filepath.Join(storage, "destination")); err != nil {
		t.Fatal(err)
	}

	remaining := 0
	for index := 0; index < contenders; index++ {
		name := "source-" + string(rune('a'+index))
		if _, err := os.Stat(filepath.Join(storage, name)); err == nil {
			remaining++
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	if remaining != contenders-1 {
		t.Fatalf("remaining sources = %d, want %d", remaining, contenders-1)
	}
}

func TestMoveFileHandlerHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("moves file", func(t *testing.T) {
		storage := newMoveHandlerStorage(t)
		recorder := serveMove(t, testMoveHandlerDeps(storage), "user/source.txt", "user/archive/result.txt")
		assertMoveEnvelope(t, recorder, http.StatusOK, 0)
		var envelope struct {
			Data string `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Data != "move files successfully" {
			t.Fatalf("success data = %q", envelope.Data)
		}
		assertStoredFile(t, filepath.Join(storage, "users", "alice", "archive", "result.txt"), []byte("source"))
		if _, err := os.Stat(filepath.Join(storage, "users", "alice", "source.txt")); !os.IsNotExist(err) {
			t.Fatalf("source still exists: %v", err)
		}
	})

	t.Run("keeps admin path compatibility", func(t *testing.T) {
		storage := t.TempDir()
		if err := os.MkdirAll(filepath.Join(storage, "users", "alice"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(storage, "users", "bob", "archive"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(storage, "users", "alice", "source.txt"), []byte("admin"), 0o600); err != nil {
			t.Fatal(err)
		}
		deps := testMoveHandlerDeps(storage)
		deps.redirect = func(_ *gin.Context, logicalPath string, _ util.JWTMessage) (string, error) {
			if logicalPath == "admin-user" {
				return "users", nil
			}
			return "users/" + strings.TrimPrefix(logicalPath, "admin-user/"), nil
		}
		recorder := serveMove(
			t,
			deps,
			"admin-user/alice/source.txt",
			"admin-user/bob/archive/result.txt",
		)
		assertMoveEnvelope(t, recorder, http.StatusOK, 0)
		assertStoredFile(t, filepath.Join(storage, "users", "bob", "archive", "result.txt"), []byte("admin"))
	})

	tests := []struct {
		name        string
		source      string
		destination string
		wantStatus  int
		wantCode    int
		mutate      func(*testing.T, string, *moveFileHandlerDeps)
	}{
		{
			name: "invalid source", source: testLogicalUserRoot, destination: "user/archive/result.txt",
			wantStatus: http.StatusBadRequest, wantCode: 40004,
		},
		{
			name: "invalid destination", source: "user/source.txt", destination: "../public/result.txt",
			wantStatus: http.StatusBadRequest, wantCode: 40004,
		},
		{
			name: "same path", source: "user/source.txt", destination: "/user//source.txt",
			wantStatus: http.StatusBadRequest, wantCode: 40004,
		},
		{
			name: "destination below source", source: "user/source.txt", destination: "user/source.txt/nested",
			wantStatus: http.StatusBadRequest, wantCode: 40004,
		},
		{
			name: "unauthorized", source: "user/source.txt", destination: "user/archive/result.txt",
			wantStatus: http.StatusUnauthorized, wantCode: 40102,
			mutate: func(_ *testing.T, _ string, deps *moveFileHandlerDeps) {
				deps.authenticate = func(*gin.Context) (util.JWTMessage, error) {
					return util.JWTMessage{}, errors.New("invalid token")
				}
			},
		},
		{
			name: "source forbidden", source: "user/source.txt", destination: "user/archive/result.txt",
			wantStatus: http.StatusForbidden, wantCode: 40301,
			mutate: func(_ *testing.T, _ string, deps *moveFileHandlerDeps) {
				deps.permission = func(path string, _ util.JWTMessage, _ *gin.Context) model.FilePermission {
					if path == "user/source.txt" {
						return model.ReadOnly
					}
					return model.ReadWrite
				}
			},
		},
		{
			name: "destination forbidden", source: "user/source.txt", destination: "public/result.txt",
			wantStatus: http.StatusForbidden, wantCode: 40301,
			mutate: func(_ *testing.T, _ string, deps *moveFileHandlerDeps) {
				deps.permission = func(path string, _ util.JWTMessage, _ *gin.Context) model.FilePermission {
					if strings.HasPrefix(path, "public/") {
						return model.ReadOnly
					}
					return model.ReadWrite
				}
			},
		},
		{
			name: "missing source", source: "user/missing.txt", destination: "user/archive/result.txt",
			wantStatus: http.StatusNotFound, wantCode: 40404,
		},
		{
			name: "missing source parent", source: "user/missing/source.txt", destination: "user/archive/result.txt",
			wantStatus: http.StatusNotFound, wantCode: 40404,
		},
		{
			name: "destination exists", source: "user/source.txt", destination: "user/archive/existing.txt",
			wantStatus: http.StatusConflict, wantCode: 40901,
			mutate: func(t *testing.T, storage string, _ *moveFileHandlerDeps) {
				target := filepath.Join(storage, "users", "alice", "archive", "existing.txt")
				if err := os.WriteFile(target, []byte("existing"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "missing destination parent", source: "user/source.txt", destination: "user/missing/result.txt",
			wantStatus: http.StatusConflict, wantCode: 40902,
		},
		{
			name: "destination redirect failure", source: "user/source.txt", destination: "user/archive/result.txt",
			wantStatus: http.StatusInternalServerError, wantCode: 50005,
			mutate: func(_ *testing.T, storage string, deps *moveFileHandlerDeps) {
				redirect := deps.redirect
				deps.redirect = func(c *gin.Context, logicalPath string, token util.JWTMessage) (string, error) {
					if logicalPath == "user/archive/result.txt" {
						return "", errors.New(filepath.Join(storage, "private-path"))
					}
					return redirect(c, logicalPath, token)
				}
				deps.openTarget = func(string, string, string) (*os.Root, string, error) {
					t.Fatal("openTarget called after destination redirect failed")
					return nil, "", nil
				}
				deps.move = func(*os.Root, string, *os.Root, string) error {
					t.Fatal("move called after destination redirect failed")
					return nil
				}
			},
		},
		{
			name: "filesystem failure", source: "user/source.txt", destination: "user/archive/result.txt",
			wantStatus: http.StatusInternalServerError, wantCode: 50005,
			mutate: func(_ *testing.T, _ string, deps *moveFileHandlerDeps) {
				deps.move = func(*os.Root, string, *os.Root, string) error {
					return errors.New("disk unavailable")
				}
			},
		},
		{
			name: "source parent access failure", source: "user/source.txt", destination: "user/archive/result.txt",
			wantStatus: http.StatusInternalServerError, wantCode: 50005,
			mutate: func(_ *testing.T, _ string, deps *moveFileHandlerDeps) {
				deps.openTarget = func(string, string, string) (*os.Root, string, error) {
					return nil, "", &uploadParentAccessError{cause: os.ErrPermission}
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage := newMoveHandlerStorage(t)
			deps := testMoveHandlerDeps(storage)
			if test.mutate != nil {
				test.mutate(t, storage, &deps)
			}
			recorder := serveMove(t, deps, test.source, test.destination)
			assertMoveEnvelope(t, recorder, test.wantStatus, test.wantCode)
			if strings.Contains(recorder.Body.String(), storage) {
				t.Fatalf("response leaked physical storage path: %s", recorder.Body.String())
			}
		})
	}
}

func TestMoveFileHandlerRejectsInvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, body := range []string{`{`, `{}`} {
		storage := newMoveHandlerStorage(t)
		deps := testMoveHandlerDeps(storage)
		router := gin.New()
		router.POST("/move/*path", func(c *gin.Context) {
			moveFileWithDeps(c, deps)
		})
		request := httptest.NewRequest(http.MethodPost, "/move/user/source.txt", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		assertMoveEnvelope(t, recorder, http.StatusBadRequest, 40001)
	}
}

func TestRegisterRoutesServesMoveEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/ss/move/user/source.txt",
		bytes.NewBufferString(`{"dst":"user/result.txt"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assertMoveEnvelope(t, recorder, http.StatusUnauthorized, 40102)
}

func TestRegisterRoutesServesMKCOLEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router)

	request := httptest.NewRequest("MKCOL", "/api/ss/user/new-directory", http.NoBody)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assertMoveEnvelope(t, recorder, http.StatusUnauthorized, 40102)
}

func newMoveHandlerStorage(t *testing.T) string {
	t.Helper()
	storage := t.TempDir()
	userRoot := filepath.Join(storage, "users", "alice")
	if err := os.MkdirAll(filepath.Join(userRoot, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userRoot, "source.txt"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	return storage
}

func testMoveHandlerDeps(storage string) moveFileHandlerDeps {
	return moveFileHandlerDeps{
		authenticate: func(*gin.Context) (util.JWTMessage, error) {
			return util.JWTMessage{}, nil
		},
		permission: func(string, util.JWTMessage, *gin.Context) model.FilePermission {
			return model.ReadWrite
		},
		redirect: func(_ *gin.Context, logicalPath string, _ util.JWTMessage) (string, error) {
			if logicalPath == testLogicalUserRoot {
				return testRealUserRoot, nil
			}
			if strings.HasPrefix(logicalPath, testLogicalUserRoot+"/") {
				return testRealUserRoot + "/" + strings.TrimPrefix(logicalPath, testLogicalUserRoot+"/"), nil
			}
			if logicalPath == "public" {
				return "public", nil
			}
			return logicalPath, nil
		},
		openTarget:  openUploadTarget,
		move:        moveStorageEntry,
		storageRoot: storage,
	}
}

func serveMove(
	t *testing.T,
	deps moveFileHandlerDeps,
	sourcePath string,
	destinationPath string,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(MoveFileReq{Dst: destinationPath})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.POST("/move/*path", func(c *gin.Context) {
		moveFileWithDeps(c, deps)
	})
	request := httptest.NewRequest(http.MethodPost, "/move/"+sourcePath, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertMoveEnvelope(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus, wantCode int) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, wantStatus, recorder.Body.String())
	}
	var envelope struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, recorder.Body.String())
	}
	if envelope.Code != wantCode {
		t.Fatalf("code = %d, want %d; body=%s", envelope.Code, wantCode, recorder.Body.String())
	}
}
