package storage

import (
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

const (
	testLogicalUserRoot = "user"
	testRealUserRoot    = "users/alice"
)

func TestCreateStorageDirectoryCreatesOneDirectoryWithRequestedMode(t *testing.T) {
	storage := t.TempDir()
	root, err := os.OpenRoot(storage)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := createStorageDirectory(root, "new-directory", 0o777); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(storage, "new-directory"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0o777 {
		t.Fatalf("directory mode = %v", info.Mode())
	}
	if err := createStorageDirectory(root, "new-directory", 0o777); !os.IsExist(err) {
		t.Fatalf("existing directory error = %v, want exists", err)
	}
}

func TestCreateStorageDirectoryRaceHasOneWinner(t *testing.T) {
	storage := t.TempDir()
	const contenders = 8
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
	for index := range roots {
		go func() {
			defer wait.Done()
			<-start
			errorsSeen <- createStorageDirectory(roots[index], "new-directory", 0o777)
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
		case os.IsExist(err):
			conflicts++
		default:
			t.Fatalf("unexpected mkdir error: %v", err)
		}
	}
	if successes != 1 || conflicts != contenders-1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
	info, err := os.Stat(filepath.Join(storage, "new-directory"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o777 {
		t.Fatalf("directory mode = %v, want 0777", info.Mode().Perm())
	}
}

func TestCreateDirectoryHandlerHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("created", func(t *testing.T) {
		storage := newCreateDirectoryHandlerStorage(t)
		recorder := serveCreateDirectory(t, testCreateDirectoryHandlerDeps(storage), "user/parent/new-directory")
		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body=%s", recorder.Code, recorder.Body.String())
		}
		info, err := os.Stat(filepath.Join(storage, "users", "alice", "parent", "new-directory"))
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() || info.Mode().Perm() != 0o777 {
			t.Fatalf("directory mode = %v", info.Mode())
		}
	})

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantCode   int
		mutate     func(*testing.T, string, *createDirectoryHandlerDeps)
	}{
		{
			name: "invalid root path", path: testLogicalUserRoot,
			wantStatus: http.StatusBadRequest, wantCode: 40004,
		},
		{
			name: "invalid traversal", path: "user/../public/new-directory",
			wantStatus: http.StatusBadRequest, wantCode: 40004,
		},
		{
			name: "unauthorized", path: "user/parent/new-directory",
			wantStatus: http.StatusUnauthorized, wantCode: 40102,
			mutate: func(_ *testing.T, _ string, deps *createDirectoryHandlerDeps) {
				deps.authenticate = func(*gin.Context) (util.JWTMessage, error) {
					return util.JWTMessage{}, errors.New("invalid token")
				}
			},
		},
		{
			name: "forbidden", path: "public/new-directory",
			wantStatus: http.StatusForbidden, wantCode: 40301,
			mutate: func(_ *testing.T, _ string, deps *createDirectoryHandlerDeps) {
				deps.permission = func(string, util.JWTMessage, *gin.Context) model.FilePermission {
					return model.ReadOnly
				}
			},
		},
		{
			name: "already exists", path: "user/parent/existing",
			wantStatus: http.StatusConflict, wantCode: 40901,
			mutate: func(t *testing.T, storage string, _ *createDirectoryHandlerDeps) {
				if err := os.Mkdir(filepath.Join(storage, "users", "alice", "parent", "existing"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "missing parent", path: "user/missing/new-directory",
			wantStatus: http.StatusConflict, wantCode: 40902,
		},
		{
			name: "redirect failure", path: "user/parent/new-directory",
			wantStatus: http.StatusInternalServerError, wantCode: 50005,
			mutate: func(_ *testing.T, storage string, deps *createDirectoryHandlerDeps) {
				deps.redirect = func(*gin.Context, string, util.JWTMessage) (string, error) {
					return "", errors.New(filepath.Join(storage, "private-path"))
				}
			},
		},
		{
			name: "filesystem failure", path: "user/parent/new-directory",
			wantStatus: http.StatusInternalServerError, wantCode: 50005,
			mutate: func(_ *testing.T, _ string, deps *createDirectoryHandlerDeps) {
				deps.mkdir = func(*os.Root, string, os.FileMode) error {
					return errors.New("disk unavailable")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage := newCreateDirectoryHandlerStorage(t)
			deps := testCreateDirectoryHandlerDeps(storage)
			if test.mutate != nil {
				test.mutate(t, storage, &deps)
			}
			recorder := serveCreateDirectory(t, deps, test.path)
			assertMoveEnvelope(t, recorder, test.wantStatus, test.wantCode)
			if strings.Contains(recorder.Body.String(), storage) {
				t.Fatalf("response leaked physical storage path: %s", recorder.Body.String())
			}
		})
	}
}

func newCreateDirectoryHandlerStorage(t *testing.T) string {
	t.Helper()
	storage := t.TempDir()
	if err := os.MkdirAll(filepath.Join(storage, "users", "alice", "parent"), 0o755); err != nil {
		t.Fatal(err)
	}
	return storage
}

func testCreateDirectoryHandlerDeps(storage string) createDirectoryHandlerDeps {
	return createDirectoryHandlerDeps{
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
			return testRealUserRoot + "/" + strings.TrimPrefix(logicalPath, testLogicalUserRoot+"/"), nil
		},
		openTarget:  openUploadTarget,
		mkdir:       createStorageDirectory,
		storageRoot: storage,
	}
}

func serveCreateDirectory(
	t *testing.T,
	deps createDirectoryHandlerDeps,
	logicalPath string,
) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.Handle("MKCOL", "/mkdir/*path", func(c *gin.Context) {
		createDirectoryWithDeps(c, deps)
	})
	request := httptest.NewRequest("MKCOL", "/mkdir/"+logicalPath, http.NoBody)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestCreateStorageDirectoryRollsBackChmodFailure(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	sentinel := errors.New("chmod denied")
	err = createStorageDirectoryWithChmod(root, "new-dir", 0o777,
		func(*os.Root, string, os.FileMode) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("got %v, want chmod error", err)
	}
	if _, err := root.Lstat("new-dir"); !os.IsNotExist(err) {
		t.Fatalf("failed mkdir left an entry: %v", err)
	}
	if err := createStorageDirectory(root, "new-dir", 0o777); err != nil {
		t.Fatalf("retry failed: %v", err)
	}
}
