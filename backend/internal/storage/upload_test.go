package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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

const testUploadMode os.FileMode = 0o640

func TestStageAndPublishFileStreamsBinaryAtomically(t *testing.T) {
	directory := t.TempDir()
	payload := []byte{0x00, 0xff, 'c', 'r', 'a', 't', 'e', 'r'}

	outcome, err := stageInDirectory(t, directory, "data.bin", bytes.NewReader(payload), false)
	if err != nil {
		t.Fatalf("stageAndPublishFile: %v", err)
	}
	if outcome.Bytes != int64(len(payload)) || outcome.Overwritten {
		t.Fatalf("outcome = %#v", outcome)
	}
	target := filepath.Join(directory, "data.bin")
	assertStoredFile(t, target, payload)
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", info.Mode().Perm())
	}
	assertNoUploadTemps(t, directory)
}

func TestStageAndPublishFilePublishesEmptyFile(t *testing.T) {
	directory := t.TempDir()
	outcome, err := stageInDirectory(t, directory, "empty.bin", bytes.NewReader(nil), false)
	if err != nil {
		t.Fatalf("stageAndPublishFile: %v", err)
	}
	if outcome.Bytes != 0 || outcome.Overwritten {
		t.Fatalf("outcome = %#v", outcome)
	}
	assertStoredFile(t, filepath.Join(directory, "empty.bin"), nil)
	assertNoUploadTemps(t, directory)
}

func TestStageAndPublishFileNeedsExplicitOverwrite(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "data.bin")
	if err := os.WriteFile(target, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := stageInDirectory(t, directory, "data.bin", bytes.NewBufferString("replacement"), false)
	if !errors.Is(err, errUploadTargetExists) {
		t.Fatalf("error = %v, want errUploadTargetExists", err)
	}
	assertStoredFile(t, target, []byte("original"))
	assertNoUploadTemps(t, directory)

	outcome, err := stageInDirectory(t, directory, "data.bin", bytes.NewBufferString("replacement"), true)
	if err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	if !outcome.Overwritten {
		t.Fatalf("outcome = %#v, want overwritten", outcome)
	}
	assertStoredFile(t, target, []byte("replacement"))
	assertNoUploadTemps(t, directory)
}

func TestStageAndPublishFileOverwriteCreatesWhenAbsent(t *testing.T) {
	directory := t.TempDir()
	outcome, err := stageInDirectory(t, directory, "new.bin", bytes.NewBufferString("new"), true)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Overwritten {
		t.Fatalf("outcome = %#v, want new file", outcome)
	}
	assertStoredFile(t, filepath.Join(directory, "new.bin"), []byte("new"))
}

func TestStageAndPublishFileKeepsOldTargetDuringTransfer(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "data.bin")
	if err := os.WriteFile(target, []byte("old-complete"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	source, writer := io.Pipe()
	type result struct {
		outcome uploadOutcome
		err     error
	}
	done := make(chan result, 1)
	go func() {
		outcome, err := stageAndPublishFile(source, root, "data.bin", true, 0o640)
		done <- result{outcome: outcome, err: err}
	}()
	if _, err := writer.Write([]byte("new-part-1")); err != nil {
		t.Fatal(err)
	}
	assertStoredFile(t, target, []byte("old-complete"))
	if _, err := writer.Write([]byte("-part-2")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	got := <-done
	if got.err != nil {
		t.Fatal(got.err)
	}
	if !got.outcome.Overwritten {
		t.Fatalf("outcome = %#v", got.outcome)
	}
	assertStoredFile(t, target, []byte("new-part-1-part-2"))
	assertNoUploadTemps(t, directory)
}

func TestStageAndPublishFileCleansSourceFailureWithoutChangingTarget(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "data.bin")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("source failed")
	source := io.MultiReader(bytes.NewBufferString("partial"), uploadErrorReader{err: sentinel})

	_, err := stageInDirectory(t, directory, "data.bin", source, true)
	var sourceErr *uploadSourceError
	if !errors.As(err, &sourceErr) || !errors.Is(sourceErr, sentinel) {
		t.Fatalf("error = %T %v, want uploadSourceError", err, err)
	}
	assertStoredFile(t, target, []byte("old"))
	assertNoUploadTemps(t, directory)
}

func TestStageAndPublishFilePublishesOpenedStageAfterNameReplacement(t *testing.T) {
	directory := t.TempDir()
	var renamedStage string
	var replacementStage string
	source := &callbackEOFReader{
		data: []byte("safe"),
		onEOF: func() {
			matches, err := filepath.Glob(filepath.Join(directory, ".crater-upload-*"))
			if err != nil || len(matches) != 1 {
				t.Fatalf("staging entries = %#v, err=%v", matches, err)
			}
			replacementStage = matches[0]
			renamedStage = replacementStage + "-renamed"
			if err := os.Rename(replacementStage, renamedStage); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(replacementStage, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(replacementStage, uploadStagePayload), []byte("attacker"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	}

	if _, err := stageInDirectory(t, directory, "result.bin", source, false); err != nil {
		t.Fatal(err)
	}
	assertStoredFile(t, filepath.Join(directory, "result.bin"), []byte("safe"))
	if renamedStage == "" || replacementStage == "" {
		t.Fatal("replacement callback did not run")
	}
}

func TestStageAndPublishFileRejectsNonRegularTargets(t *testing.T) {
	directory := t.TempDir()
	targetDirectory := filepath.Join(directory, "target")
	if err := os.Mkdir(targetDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := stageInDirectory(t, directory, "target", bytes.NewBufferString("x"), true); !errors.Is(err, errUploadTargetNotRegular) {
		t.Fatalf("directory error = %v", err)
	}

	if err := os.Symlink(targetDirectory, filepath.Join(directory, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := stageInDirectory(t, directory, "link", bytes.NewBufferString("x"), true); !errors.Is(err, errUploadTargetNotRegular) {
		t.Fatalf("symlink error = %v", err)
	}
}

func TestStageAndPublishFileRaceHasOneWinner(t *testing.T) {
	directory := t.TempDir()
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	const contenders = 8
	var wait sync.WaitGroup
	wait.Add(contenders)
	start := make(chan struct{})
	errorsSeen := make(chan error, contenders)

	for index := 0; index < contenders; index++ {
		go func() {
			defer wait.Done()
			<-start
			_, err := stageAndPublishFile(
				bytes.NewBufferString(string(rune('a'+index))),
				root,
				"race.bin",
				false,
				0o640,
			)
			errorsSeen <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errorsSeen)

	successes := 0
	exists := 0
	for err := range errorsSeen {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, errUploadTargetExists):
			exists++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if successes != 1 || exists != contenders-1 {
		t.Fatalf("successes=%d exists=%d", successes, exists)
	}
	assertNoUploadTemps(t, directory)
}

func TestOpenUploadTargetRejectsEscapingParentSymlink(t *testing.T) {
	storage := t.TempDir()
	authorized := filepath.Join(storage, "users", "alice")
	outside := filepath.Join(storage, "users", "bob")
	if err := os.MkdirAll(authorized, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(authorized, "escape")); err != nil {
		t.Fatal(err)
	}

	_, _, err := openUploadTarget(storage, testRealUserRoot, "users/alice/escape/secret.bin")
	if !errors.Is(err, errUploadParentInvalid) {
		t.Fatalf("error = %v, want errUploadParentInvalid", err)
	}
}

func TestOpenUploadTargetRejectsAuthorizedRootEscapingStorage(t *testing.T) {
	storage := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(storage, "users"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(storage, "users", "alice")); err != nil {
		t.Fatal(err)
	}

	_, _, err := openUploadTarget(storage, testRealUserRoot, "users/alice/secret.bin")
	if !errors.Is(err, errUploadParentInvalid) {
		t.Fatalf("error = %v, want errUploadParentInvalid", err)
	}
}

func TestOpenUploadTargetRejectsRawInternalTraversal(t *testing.T) {
	for _, candidate := range []string{
		"users/../users/alice",
		"users/alice/../alice/result.bin",
		`users\..\users\alice`,
		"users/.",
		"users/alice/",
		"//users/alice",
		"users///alice",
		"users/alice//runs//result.bin",
		"users/alice\n",
	} {
		if _, err := cleanStorageRelativePath(candidate); !errors.Is(err, errUploadParentInvalid) {
			t.Fatalf("cleanStorageRelativePath(%q) error = %v, want errUploadParentInvalid", candidate, err)
		}
	}
}

func TestCleanStorageRelativePathSupportsOneLegacyAbsoluteSpaceMarker(t *testing.T) {
	for input, want := range map[string]string{
		"/public/models":             filepath.Join("public", "models"),
		"users//space/zhouyh25":      filepath.Join("users", "space", "zhouyh25"),
		"users//space/zhouyh25/file": filepath.Join("users", "space", "zhouyh25", "file"),
	} {
		got, err := cleanStorageRelativePath(input)
		if err != nil {
			t.Fatalf("cleanStorageRelativePath(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("cleanStorageRelativePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestOpenUploadTargetSupportsLegacyLeadingSlashSpace(t *testing.T) {
	storage := t.TempDir()
	if err := os.MkdirAll(filepath.Join(storage, "users", "space", "alice", "jobs"), 0o755); err != nil {
		t.Fatal(err)
	}
	parent, targetName, err := openUploadTarget(
		storage,
		"users//space/alice",
		"users//space/alice/jobs/result.bin",
	)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	if targetName != "result.bin" {
		t.Fatalf("targetName = %q", targetName)
	}
}

func TestOpenUploadTargetDoesNotCreateMissingParent(t *testing.T) {
	storage := t.TempDir()
	if err := os.MkdirAll(filepath.Join(storage, "users", "alice"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, _, err := openUploadTarget(storage, testRealUserRoot, "users/alice/missing/secret.bin")
	if !errors.Is(err, errUploadParentInvalid) {
		t.Fatalf("error = %v, want errUploadParentInvalid", err)
	}
	if _, err := os.Stat(filepath.Join(storage, "users", "alice", "missing")); !os.IsNotExist(err) {
		t.Fatalf("missing parent was created: %v", err)
	}
}

func TestOpenedUploadTargetRemainsAnchoredAcrossParentRename(t *testing.T) {
	storage := t.TempDir()
	jobs := filepath.Join(storage, "users", "alice", "jobs")
	original := filepath.Join(storage, "users", "alice", "jobs-original")
	outside := t.TempDir()
	if err := os.MkdirAll(jobs, 0o755); err != nil {
		t.Fatal(err)
	}

	parent, targetName, err := openUploadTarget(storage, testRealUserRoot, "users/alice/jobs/result.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	if err := os.Rename(jobs, original); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, jobs); err != nil {
		t.Fatal(err)
	}

	if _, err := stageAndPublishFile(bytes.NewBufferString("safe"), parent, targetName, false, 0o640); err != nil {
		t.Fatal(err)
	}
	assertStoredFile(t, filepath.Join(original, "result.bin"), []byte("safe"))
	if _, err := os.Stat(filepath.Join(outside, "result.bin")); !os.IsNotExist(err) {
		t.Fatalf("upload escaped through replacement symlink: %v", err)
	}
}

func TestUploadFileHandlerHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Run("created", func(t *testing.T) {
		storage := newUploadHandlerStorage(t)
		recorder := serveUpload(t, testUploadHandlerDeps(storage), "user/result.bin", "overwrite=false", bytes.NewBufferString("data"))
		assertUploadEnvelope(t, recorder, http.StatusCreated, 0)
		assertStoredFile(t, filepath.Join(storage, "users", "alice", "result.bin"), []byte("data"))
		info, err := os.Stat(filepath.Join(storage, "users", "alice", "result.bin"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != uploadFileMode {
			t.Fatalf("uploaded mode = %o, want %o", info.Mode().Perm(), uploadFileMode)
		}
	})

	t.Run("overwritten", func(t *testing.T) {
		storage := newUploadHandlerStorage(t)
		target := filepath.Join(storage, "users", "alice", "result.bin")
		if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
		recorder := serveUpload(t, testUploadHandlerDeps(storage), "user/result.bin", "overwrite=true", bytes.NewBufferString("new"))
		assertUploadEnvelope(t, recorder, http.StatusOK, 0)
		assertStoredFile(t, target, []byte("new"))
	})

	tests := []struct {
		name       string
		path       string
		query      string
		wantStatus int
		wantCode   int
		mutate     func(*testing.T, string, *uploadHandlerDeps)
	}{
		{name: "invalid overwrite", path: "user/result.bin", query: "overwrite=yes", wantStatus: http.StatusBadRequest, wantCode: 40004},
		{name: "invalid path", path: testLogicalUserRoot, query: "overwrite=false", wantStatus: http.StatusBadRequest, wantCode: 40004},
		{
			name: "unauthorized", path: "user/result.bin", query: "overwrite=false",
			wantStatus: http.StatusUnauthorized, wantCode: 40102,
			mutate: func(_ *testing.T, _ string, deps *uploadHandlerDeps) {
				deps.authenticate = func(*gin.Context) (util.JWTMessage, error) {
					return util.JWTMessage{}, errors.New("invalid token")
				}
			},
		},
		{
			name: "forbidden", path: "user/result.bin", query: "overwrite=false",
			wantStatus: http.StatusForbidden, wantCode: 40301,
			mutate: func(_ *testing.T, _ string, deps *uploadHandlerDeps) {
				deps.permission = func(string, util.JWTMessage, *gin.Context) model.FilePermission {
					return model.ReadOnly
				}
			},
		},
		{
			name: "existing target", path: "user/result.bin", query: "overwrite=false",
			wantStatus: http.StatusConflict, wantCode: 40901,
			mutate: func(t *testing.T, storage string, _ *uploadHandlerDeps) {
				if err := os.WriteFile(filepath.Join(storage, "users", "alice", "result.bin"), []byte("old"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "nonregular target", path: "user/result.bin", query: "overwrite=true",
			wantStatus: http.StatusConflict, wantCode: 40902,
			mutate: func(t *testing.T, storage string, _ *uploadHandlerDeps) {
				if err := os.Mkdir(filepath.Join(storage, "users", "alice", "result.bin"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "missing parent", path: "user/missing/result.bin", query: "overwrite=false",
			wantStatus: http.StatusConflict, wantCode: 40902,
		},
		{
			name: "source read failure", path: "user/result.bin", query: "overwrite=false",
			wantStatus: http.StatusBadRequest, wantCode: 40001,
			mutate: func(_ *testing.T, _ string, deps *uploadHandlerDeps) {
				deps.stagePublish = func(io.Reader, *os.Root, string, bool, os.FileMode) (uploadOutcome, error) {
					return uploadOutcome{}, &uploadSourceError{cause: errors.New("broken body")}
				}
			},
		},
		{
			name: "filesystem failure", path: "user/result.bin", query: "overwrite=false",
			wantStatus: http.StatusInternalServerError, wantCode: 50005,
			mutate: func(_ *testing.T, _ string, deps *uploadHandlerDeps) {
				deps.openTarget = func(string, string, string) (*os.Root, string, error) {
					return nil, "", errors.New("disk unavailable")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage := newUploadHandlerStorage(t)
			deps := testUploadHandlerDeps(storage)
			if test.mutate != nil {
				test.mutate(t, storage, &deps)
			}
			recorder := serveUpload(t, deps, test.path, test.query, bytes.NewBufferString("data"))
			assertUploadEnvelope(t, recorder, test.wantStatus, test.wantCode)
		})
	}
}

func TestNormalizeUploadLogicalPath(t *testing.T) {
	if got, err := normalizeUploadLogicalPath("/user//runs/./a.bin"); err != nil || got != "user/runs/a.bin" {
		t.Fatalf("normalize = %q, %v", got, err)
	}
	for _, invalid := range []string{testLogicalUserRoot, "admin/file", "user/../public/file", `user\file`, "user/a\nb"} {
		if _, err := normalizeUploadLogicalPath(invalid); err == nil {
			t.Errorf("normalizeUploadLogicalPath(%q) accepted invalid path", invalid)
		}
	}
}

func TestNormalizeWebDAVMutationLogicalPathKeepsAdminCompatibility(t *testing.T) {
	for _, valid := range []string{
		"user/new-directory",
		"public/new-directory",
		"account/new-directory",
		"admin-user/alice/new-directory",
		"admin-public/new-directory",
		"admin-account/team/new-directory",
	} {
		if _, err := normalizeWebDAVMutationLogicalPath(valid); err != nil {
			t.Errorf("normalizeWebDAVMutationLogicalPath(%q): %v", valid, err)
		}
	}
	for _, invalid := range []string{
		testLogicalUserRoot,
		"admin-user",
		"crater-model/new-directory",
		"user/../public/new-directory",
	} {
		if _, err := normalizeWebDAVMutationLogicalPath(invalid); err == nil {
			t.Errorf("normalizeWebDAVMutationLogicalPath(%q) accepted invalid path", invalid)
		}
	}
}

func TestParseUploadOverwrite(t *testing.T) {
	for _, valid := range []string{"", "false", "true"} {
		if _, err := parseUploadOverwrite(valid); err != nil {
			t.Errorf("parseUploadOverwrite(%q): %v", valid, err)
		}
	}
	for _, invalid := range []string{"1", "TRUE", "yes"} {
		if _, err := parseUploadOverwrite(invalid); err == nil {
			t.Errorf("parseUploadOverwrite(%q) accepted invalid value", invalid)
		}
	}
}

func TestRegisterRoutesServesUploadEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/ss/upload/user/test.bin?overwrite=false",
		bytes.NewBufferString("data"),
	)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assertUploadEnvelope(t, recorder, http.StatusUnauthorized, 40102)
}

type uploadErrorReader struct {
	err error
}

func (reader uploadErrorReader) Read([]byte) (int, error) {
	return 0, reader.err
}

type callbackEOFReader struct {
	data  []byte
	onEOF func()
	read  bool
}

func (reader *callbackEOFReader) Read(buffer []byte) (int, error) {
	if !reader.read {
		reader.read = true
		return copy(buffer, reader.data), nil
	}
	reader.onEOF()
	return 0, io.EOF
}

func stageInDirectory(
	t *testing.T,
	directory string,
	targetName string,
	source io.Reader,
	overwrite bool,
) (uploadOutcome, error) {
	t.Helper()
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	return stageAndPublishFile(source, root, targetName, overwrite, testUploadMode)
}

func newUploadHandlerStorage(t *testing.T) string {
	t.Helper()
	storage := t.TempDir()
	if err := os.MkdirAll(filepath.Join(storage, "users", "alice"), 0o755); err != nil {
		t.Fatal(err)
	}
	return storage
}

func testUploadHandlerDeps(storage string) uploadHandlerDeps {
	return uploadHandlerDeps{
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
		openTarget:   openUploadTarget,
		stagePublish: stageAndPublishFile,
		storageRoot:  storage,
	}
}

func serveUpload(
	t *testing.T,
	deps uploadHandlerDeps,
	logicalPath string,
	query string,
	body io.Reader,
) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.POST("/upload/*path", func(c *gin.Context) {
		uploadFileWithDeps(c, deps)
	})
	target := "/upload/" + logicalPath
	if query != "" {
		target += "?" + query
	}
	request := httptest.NewRequest(http.MethodPost, target, body)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertUploadEnvelope(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus, wantCode int) {
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

func assertStoredFile(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s = %v, want %v", path, got, want)
	}
}

func assertNoUploadTemps(t *testing.T, directory string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(directory, ".crater-upload-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary upload entries remain: %#v", matches)
	}
}
