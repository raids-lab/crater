// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/webdav"

	"github.com/raids-lab/crater/internal/resputil"
)

func assertFileListResponse(t *testing.T, files []Files) []Files {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resputil.Success(c, files)
	var response struct {
		Code int     `json:"code"`
		Data []Files `json:"data"`
		Msg  string  `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || response.Code != 0 || response.Msg != "" {
		t.Fatalf("unexpected success response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if response.Data == nil {
		t.Fatalf("file list must be a JSON array: %s", recorder.Body.String())
	}
	return response.Data
}

func TestHandleDirsListReturnsArrays(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	filesystem := webdav.Dir(root)
	for _, path := range []string{"empty", "file.txt"} {
		t.Run(path, func(t *testing.T) {
			files, err := handleDirsList(filesystem, path)
			if err != nil {
				t.Fatal(err)
			}
			if got := assertFileListResponse(t, files); len(got) != 0 {
				t.Fatalf("empty listing returned %d entries", len(got))
			}
		})
	}
	t.Run("nonempty directory", func(t *testing.T) {
		files, err := handleDirsList(filesystem, "/")
		if err != nil {
			t.Fatal(err)
		}
		got := assertFileListResponse(t, files)
		if len(got) != 2 {
			t.Fatalf("directory returned %d entries, want 2", len(got))
		}
		for _, file := range got {
			info, err := os.Stat(filepath.Join(root, file.Name))
			if err != nil {
				t.Fatal(err)
			}
			if file.IsDir != info.IsDir() || file.Size != info.Size() || !file.ModifyTime.Equal(info.ModTime()) {
				t.Fatalf("incorrect metadata for %s: %+v", file.Name, file)
			}
		}
	})
}

type fileListReadErrorFS struct {
	webdav.FileSystem
	readErr error
}

func (fs fileListReadErrorFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	file, err := fs.FileSystem.OpenFile(ctx, name, flag, perm)
	if err != nil {
		return nil, err
	}
	return fileListReadErrorFile{File: file, readErr: fs.readErr}, nil
}

type fileListReadErrorFile struct {
	webdav.File
	readErr error
}

func (file fileListReadErrorFile) Readdir(int) ([]os.FileInfo, error) {
	return nil, file.readErr
}

func TestHandleDirsListPreservesErrors(t *testing.T) {
	filesystem := webdav.Dir(t.TempDir())
	readErr := errors.New("directory read failed")
	for _, test := range []struct {
		name       string
		filesystem webdav.FileSystem
		path       string
		wantErr    error
	}{
		{name: "missing path", filesystem: filesystem, path: "missing", wantErr: os.ErrNotExist},
		{name: "read failure", filesystem: fileListReadErrorFS{filesystem, readErr}, path: "/", wantErr: readErr},
	} {
		t.Run(test.name, func(t *testing.T) {
			files, err := handleDirsList(test.filesystem, test.path)
			if !errors.Is(err, test.wantErr) || files != nil {
				t.Fatalf("listing = %v, %v; want nil, %v", files, err, test.wantErr)
			}
		})
	}
}

func TestGetFilesByPathsReturnsArrays(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "user"), 0o700); err != nil {
		t.Fatal(err)
	}
	// This helper uses the shared WebDAV handler, so this test must stay sequential.
	previousFS := fs
	fs = &webdav.Handler{FileSystem: webdav.Dir(root)}
	t.Cleanup(func() { fs = previousFS })
	for _, test := range []struct {
		name  string
		paths []string
		count int
	}{
		{name: "no paths"},
		{name: "unavailable roots", paths: []string{"missing"}},
		{name: "available root", paths: []string{"missing", "user"}, count: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "/api/ss/files", http.NoBody)
			got := assertFileListResponse(t, GetFilesByPaths(test.paths, c))
			if len(got) != test.count {
				t.Fatalf("root listing returned %d entries, want %d", len(got), test.count)
			}
			if test.count > 0 && (got[0].Name != "user" || !got[0].IsDir) {
				t.Fatalf("incorrect root entry: %+v", got[0])
			}
		})
	}
}
