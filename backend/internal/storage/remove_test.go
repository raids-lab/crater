package storage

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
)

func TestParseRemoveRecursiveRequiresExplicitBoolean(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want bool
		ok   bool
	}{
		{raw: "false", want: false, ok: true},
		{raw: "true", want: true, ok: true},
		{raw: "", ok: false},
		{raw: "TRUE", ok: false},
		{raw: "False", ok: false},
		{raw: "0", ok: false},
		{raw: "1", ok: false},
	} {
		t.Run(test.raw, func(t *testing.T) {
			got, err := parseRemoveRecursive(test.raw)
			if test.ok {
				if err != nil || got != test.want {
					t.Fatalf("parseRemoveRecursive(%q) = %v, %v", test.raw, got, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("parseRemoveRecursive(%q) unexpectedly succeeded", test.raw)
			}
		})
	}
}

func TestNormalizeRemoveLogicalPathIsStrictAndUserScoped(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want string
	}{
		{raw: "user/file.txt", want: "user/file.txt"},
		{raw: "/user/folder/file.txt", want: "user/folder/file.txt"},
		{raw: "public/shared", want: "public/shared"},
		{raw: "account/project", want: "account/project"},
	} {
		t.Run("accepts "+test.raw, func(t *testing.T) {
			got, err := normalizeRemoveLogicalPath(test.raw)
			if err != nil || got != test.want {
				t.Fatalf("normalizeRemoveLogicalPath(%q) = %q, %v", test.raw, got, err)
			}
		})
	}

	for _, raw := range []string{
		"",
		"/",
		"user",
		"/user",
		"user/",
		"user//file.txt",
		"/user//file.txt",
		"//user/file.txt",
		"user/./file.txt",
		"user/../file.txt",
		"user/file.txt/..",
		"admin-user/alice/file.txt",
		"admin-public/file.txt",
		"admin-account/team/file.txt",
		"dataset/file.txt",
		"model/file.txt",
		"unknown/file.txt",
		`user\file.txt`,
		"user/control\ncharacter",
	} {
		t.Run("rejects "+raw, func(t *testing.T) {
			if got, err := normalizeRemoveLogicalPath(raw); err == nil {
				t.Fatalf("normalizeRemoveLogicalPath(%q) = %q, want error", raw, got)
			}
		})
	}
}

func TestRemoveStorageEntryRequiresRecursiveForDirectory(t *testing.T) {
	storage := t.TempDir()
	target := filepath.Join(storage, "directory")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "nested.txt"), []byte("nested"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(storage)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := removeStorageEntry(root, "directory", false); !errors.Is(err, errRemoveRecursiveRequired) {
		t.Fatalf("error = %v, want errRemoveRecursiveRequired", err)
	}
	if _, err := os.Stat(filepath.Join(target, "nested.txt")); err != nil {
		t.Fatalf("directory changed after non-recursive refusal: %v", err)
	}

	if err := removeStorageEntry(root, "directory", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("directory still exists: %v", err)
	}
}

func TestRemoveStorageEntryReportsMissingAndRejectsInvalidNames(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if err := removeStorageEntry(root, "missing", false); !errors.Is(err, errRemoveTargetNotFound) {
		t.Fatalf("missing target error = %v", err)
	}
	for _, name := range []string{"", ".", "..", "nested/file"} {
		if err := removeStorageEntry(root, name, false); !errors.Is(err, errUploadParentInvalid) {
			t.Fatalf("name %q error = %v, want errUploadParentInvalid", name, err)
		}
	}
}

func TestRemoveStorageEntryDoesNotUpgradeFileRaceToRecursiveDelete(t *testing.T) {
	storage := t.TempDir()
	filePath := filepath.Join(storage, "file")
	directoryPath := filepath.Join(storage, "directory")
	if err := os.WriteFile(filePath, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(directoryPath, 0o700); err != nil {
		t.Fatal(err)
	}
	fileInfo, err := os.Lstat(filePath)
	if err != nil {
		t.Fatal(err)
	}
	directoryInfo, err := os.Lstat(directoryPath)
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(storage)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	lstatCalls := 0
	recursiveCalled := false
	deps := removeStorageEntryDeps{
		lstat: func(*os.Root, string) (os.FileInfo, error) {
			lstatCalls++
			if lstatCalls == 1 {
				return fileInfo, nil
			}
			return directoryInfo, nil
		},
		unlinkNonDirectory: func(*os.Root, string) error {
			return syscall.EISDIR
		},
		removeDirectory: func(*os.Root, string) error {
			recursiveCalled = true
			return nil
		},
	}
	err = removeStorageEntryWithDeps(root, "target", true, deps)
	if !errors.Is(err, errRemoveTargetChanged) {
		t.Fatalf("error = %v, want errRemoveTargetChanged", err)
	}
	if recursiveCalled {
		t.Fatal("file-to-directory race was upgraded to recursive removal")
	}
}

func TestRemoveStorageEntryPreservesRegularFilePermissionError(t *testing.T) {
	storage := t.TempDir()
	path := filepath.Join(storage, "file")
	if err := os.WriteFile(path, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	fileInfo, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(storage)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	deps := removeStorageEntryDeps{
		lstat: func(*os.Root, string) (os.FileInfo, error) {
			return fileInfo, nil
		},
		unlinkNonDirectory: func(*os.Root, string) error {
			return syscall.EPERM
		},
		removeDirectory: func(*os.Root, string) error {
			t.Fatal("recursive removal called for a regular file")
			return nil
		},
	}
	if err := removeStorageEntryWithDeps(root, "file", true, deps); !errors.Is(err, syscall.EPERM) {
		t.Fatalf("error = %v, want EPERM", err)
	}
}

func TestRemoveFileHandlerHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("removes file and returns metadata", func(t *testing.T) {
		storage := newRemoveHandlerStorage(t)
		recorder := serveRemove(t, testRemoveHandlerDeps(storage), "user/file.txt", "recursive=false")
		assertRemoveEnvelope(t, recorder, http.StatusOK, 0)
		var envelope struct {
			Data removeFileResponse `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Data.RemotePath != "user/file.txt" || envelope.Data.Recursive {
			t.Fatalf("success data = %+v", envelope.Data)
		}
		if _, err := os.Lstat(filepath.Join(storage, "users", "alice", "file.txt")); !os.IsNotExist(err) {
			t.Fatalf("file still exists: %v", err)
		}
	})

	t.Run("removes directory only with recursive opt in", func(t *testing.T) {
		storage := newRemoveHandlerStorage(t)
		recorder := serveRemove(t, testRemoveHandlerDeps(storage), "user/tree", "recursive=true")
		assertRemoveEnvelope(t, recorder, http.StatusOK, 0)
		if _, err := os.Lstat(filepath.Join(storage, "users", "alice", "tree")); !os.IsNotExist(err) {
			t.Fatalf("directory still exists: %v", err)
		}
	})

	tests := []struct {
		name       string
		path       string
		query      string
		wantStatus int
		wantCode   int
		mutate     func(*testing.T, string, *removeFileHandlerDeps)
		assert     func(*testing.T, string)
	}{
		{
			name: "unauthorized", path: "user/file.txt", query: "recursive=false",
			wantStatus: http.StatusUnauthorized, wantCode: 40102,
			mutate: func(_ *testing.T, _ string, deps *removeFileHandlerDeps) {
				deps.authenticate = func(*gin.Context) (util.JWTMessage, error) {
					return util.JWTMessage{}, errors.New("invalid token")
				}
			},
		},
		{
			name: "missing recursive", path: "user/file.txt", query: "",
			wantStatus: http.StatusBadRequest, wantCode: 40003,
		},
		{
			name: "ambiguous recursive", path: "user/file.txt", query: "recursive=false&recursive=true",
			wantStatus: http.StatusBadRequest, wantCode: 40004,
		},
		{
			name: "invalid recursive", path: "user/file.txt", query: "recursive=TRUE",
			wantStatus: http.StatusBadRequest, wantCode: 40004,
		},
		{
			name: "empty recursive", path: "user/file.txt", query: "recursive=",
			wantStatus: http.StatusBadRequest, wantCode: 40004,
		},
		{
			name: "logical root target", path: "user", query: "recursive=true",
			wantStatus: http.StatusBadRequest, wantCode: 40004,
		},
		{
			name: "reserved admin path", path: "admin-user/alice/file.txt", query: "recursive=false",
			wantStatus: http.StatusBadRequest, wantCode: 40004,
		},
		{
			name: "forbidden", path: "user/file.txt", query: "recursive=false",
			wantStatus: http.StatusForbidden, wantCode: 40301,
			mutate: func(_ *testing.T, _ string, deps *removeFileHandlerDeps) {
				deps.permission = func(string, util.JWTMessage, *gin.Context) model.FilePermission {
					return model.ReadOnly
				}
			},
		},
		{
			name: "missing target", path: "user/missing.txt", query: "recursive=false",
			wantStatus: http.StatusNotFound, wantCode: 40404,
		},
		{
			name: "missing parent", path: "user/missing/file.txt", query: "recursive=false",
			wantStatus: http.StatusNotFound, wantCode: 40404,
		},
		{
			name: "directory without recursive", path: "user/tree", query: "recursive=false",
			wantStatus: http.StatusConflict, wantCode: 40902,
			assert: func(t *testing.T, storage string) {
				t.Helper()
				if _, err := os.Stat(filepath.Join(storage, "users", "alice", "tree", "nested.txt")); err != nil {
					t.Fatalf("directory changed after refusal: %v", err)
				}
			},
		},
		{
			name: "redirect failure", path: "user/file.txt", query: "recursive=false",
			wantStatus: http.StatusInternalServerError, wantCode: 50005,
			mutate: func(_ *testing.T, storage string, deps *removeFileHandlerDeps) {
				deps.redirect = func(*gin.Context, string, util.JWTMessage) (string, error) {
					return "", errors.New(filepath.Join(storage, "private-path"))
				}
			},
		},
		{
			name: "parent access failure", path: "user/file.txt", query: "recursive=false",
			wantStatus: http.StatusInternalServerError, wantCode: 50005,
			mutate: func(_ *testing.T, _ string, deps *removeFileHandlerDeps) {
				deps.openTarget = func(string, string, string) (*os.Root, string, error) {
					return nil, "", &uploadParentAccessError{cause: os.ErrPermission}
				}
			},
		},
		{
			name: "target changed", path: "user/file.txt", query: "recursive=false",
			wantStatus: http.StatusConflict, wantCode: 40902,
			mutate: func(_ *testing.T, _ string, deps *removeFileHandlerDeps) {
				deps.remove = func(*os.Root, string, bool) error {
					return errRemoveTargetChanged
				}
			},
		},
		{
			name: "cross device", path: "user/tree", query: "recursive=true",
			wantStatus: http.StatusConflict, wantCode: 40902,
			mutate: func(_ *testing.T, _ string, deps *removeFileHandlerDeps) {
				deps.remove = func(*os.Root, string, bool) error {
					return errRemoveCrossDevice
				}
			},
		},
		{
			name: "safe operation unsupported", path: "user/file.txt", query: "recursive=false",
			wantStatus: http.StatusInternalServerError, wantCode: 50005,
			mutate: func(_ *testing.T, _ string, deps *removeFileHandlerDeps) {
				deps.remove = func(*os.Root, string, bool) error {
					return errRemoveOperationUnsupported
				}
			},
		},
		{
			name: "filesystem failure", path: "user/file.txt", query: "recursive=false",
			wantStatus: http.StatusInternalServerError, wantCode: 50005,
			mutate: func(_ *testing.T, _ string, deps *removeFileHandlerDeps) {
				deps.remove = func(*os.Root, string, bool) error {
					return errors.New("disk unavailable")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage := newRemoveHandlerStorage(t)
			deps := testRemoveHandlerDeps(storage)
			if test.mutate != nil {
				test.mutate(t, storage, &deps)
			}
			recorder := serveRemove(t, deps, test.path, test.query)
			assertRemoveEnvelope(t, recorder, test.wantStatus, test.wantCode)
			if strings.Contains(recorder.Body.String(), storage) {
				t.Fatalf("response leaked physical storage path: %s", recorder.Body.String())
			}
			if test.assert != nil {
				test.assert(t, storage)
			}
		})
	}
}

func TestRegisterRoutesServesSafeRemoveAndPreservesLegacyDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router)

	for _, path := range []string{
		"/api/ss/files/user/file.txt?recursive=false",
		"/api/ss/delete/user/file.txt",
	} {
		request := httptest.NewRequest(http.MethodDelete, path, http.NoBody)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code == http.StatusNotFound {
			t.Fatalf("DELETE %s was not registered", path)
		}
	}

	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/ss/files/user/file.txt?recursive=false",
		http.NoBody,
	)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assertRemoveEnvelope(t, recorder, http.StatusUnauthorized, 40102)
}

func newRemoveHandlerStorage(t *testing.T) string {
	t.Helper()
	storage := t.TempDir()
	userRoot := filepath.Join(storage, "users", "alice")
	if err := os.MkdirAll(filepath.Join(userRoot, "tree"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userRoot, "file.txt"), []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userRoot, "tree", "nested.txt"), []byte("nested"), 0o600); err != nil {
		t.Fatal(err)
	}
	return storage
}

func testRemoveHandlerDeps(storage string) removeFileHandlerDeps {
	return removeFileHandlerDeps{
		authenticate: func(*gin.Context) (util.JWTMessage, error) {
			return util.JWTMessage{}, nil
		},
		permission: func(string, util.JWTMessage, *gin.Context) model.FilePermission {
			return model.ReadWrite
		},
		redirect: func(_ *gin.Context, logicalPath string, _ util.JWTMessage) (string, error) {
			if logicalPath == "user" {
				return "users/alice", nil
			}
			return "users/alice/" + strings.TrimPrefix(logicalPath, "user/"), nil
		},
		openTarget:  openUploadTarget,
		remove:      removeStorageEntry,
		storageRoot: storage,
	}
}

func serveRemove(
	t *testing.T,
	deps removeFileHandlerDeps,
	logicalPath string,
	rawQuery string,
) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.DELETE("/remove/*path", func(c *gin.Context) {
		removeFileWithDeps(c, deps)
	})
	url := "/remove/" + logicalPath
	if rawQuery != "" {
		url += "?" + rawQuery
	}
	request := httptest.NewRequest(http.MethodDelete, url, http.NoBody)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertRemoveEnvelope(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus, wantCode int) {
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
