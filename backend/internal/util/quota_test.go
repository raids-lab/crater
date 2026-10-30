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

package util

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bytedance/mockey"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/storagequota"
)

func TestCheckStorageQuotaSkipsDependencies(t *testing.T) {
	for _, tt := range []struct {
		name    string
		enabled bool
		mounts  []VolumeMount
	}{
		{name: "disabled with a mount", mounts: []VolumeMount{{Type: FileType, SubPath: "user"}}},
		{name: "enabled with omitted mounts", enabled: true},
		{name: "enabled with empty mounts", enabled: true, mounts: []VolumeMount{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Storage.Quota.Enabled = ptr.To(tt.enabled)
			configMock := mockey.Mock(config.GetConfig).Return(cfg).Build()
			t.Cleanup(func() { configMock.UnPatch() })
			dbMock := mockey.Mock(query.GetDB).To(func() *gorm.DB {
				t.Fatal("quota check accessed the database before skipping")
				return nil
			}).Build()
			t.Cleanup(func() { dbMock.UnPatch() })
			if err := CheckStorageQuota("alice", tt.mounts, nil, nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckStorageQuotaMountedJobs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, space TEXT, deleted_at DATETIME)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO users (id, name, space) VALUES (?, ?, ?)", 1, "alice", "alice-space").Error; err != nil {
		t.Fatal(err)
	}
	dbMock := mockey.Mock(query.GetDB).Return(db).Build()
	t.Cleanup(func() { dbMock.UnPatch() })
	cfg := &config.Config{}
	cfg.Storage.Quota.Enabled = ptr.To(true)
	cfg.Storage.Quota.Provider = storagequota.ProviderStorageServer
	cfg.Storage.Prefix.User = "users"
	configMock := mockey.Mock(config.GetConfig).Return(cfg).Build()
	t.Cleanup(func() { configMock.UnPatch() })

	for _, tt := range []struct {
		name           string
		mount          VolumeMount
		quota          int64
		usage          int64
		failPath       string
		wantCode       bizerr.BizCode
		wantMessage    string
		wantUsageCalls int32
	}{
		{name: "under quota", mount: VolumeMount{Type: FileType, SubPath: "user"}, quota: 100, usage: 60, wantUsageCalls: 1},
		{name: "at quota", mount: VolumeMount{Type: FileType, SubPath: "user"}, quota: 100, usage: 100, wantUsageCalls: 1, wantCode: bizerr.Conflict.ResourceStatusError, wantMessage: "storage usage has reached the quota"},
		{name: "over quota", mount: VolumeMount{Type: FileType, SubPath: "user"}, quota: 100, usage: 101, wantUsageCalls: 1, wantCode: bizerr.Conflict.ResourceStatusError, wantMessage: "storage usage has reached the quota"},
		{name: "unlimited", mount: VolumeMount{Type: FileType, SubPath: "user"}, quota: -1},
		{name: "Ceph unlimited sentinel", mount: VolumeMount{Type: FileType, SubPath: "user"}, quota: 0},
		{name: "personal quota unavailable", mount: VolumeMount{Type: FileType, SubPath: "user"}, failPath: "/internal/storage/quota", wantCode: bizerr.Internal.FileSystemError, wantMessage: "Failed to read the storage quota."},
		{name: "account quota unavailable", mount: VolumeMount{Type: FileType, SubPath: "account/project"}, failPath: "/internal/storage/quota", wantCode: bizerr.Internal.FileSystemError, wantMessage: "Failed to read the storage quota."},
		{name: "public quota unavailable", mount: VolumeMount{Type: FileType, SubPath: "public/models"}, failPath: "/internal/storage/quota", wantCode: bizerr.Internal.FileSystemError, wantMessage: "Failed to read the storage quota."},
		{name: "dataset quota unavailable", mount: VolumeMount{Type: DataType, DatasetID: 42}, failPath: "/internal/storage/quota", wantCode: bizerr.Internal.FileSystemError, wantMessage: "Failed to read the storage quota."},
		{name: "usage unavailable", mount: VolumeMount{Type: FileType, SubPath: "user"}, quota: 100, failPath: "/internal/storage/usage", wantUsageCalls: 1, wantCode: bizerr.Internal.FileSystemError, wantMessage: "Failed to read current storage usage."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var quotaCalls, usageCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("path") != "users/alice-space" {
					t.Errorf("unexpected storage path: %s", r.URL.Query().Get("path"))
				}
				switch r.URL.Path {
				case "/internal/storage/quota":
					quotaCalls.Add(1)
				case "/internal/storage/usage":
					usageCalls.Add(1)
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
				}
				if r.URL.Path == tt.failPath {
					http.Error(w, "private provider failure", http.StatusServiceUnavailable)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				var body any = storagequota.Quota{MaxBytes: tt.quota}
				if r.URL.Path == "/internal/storage/usage" {
					body = storagequota.Usage{Bytes: tt.usage}
				}
				if err := json.NewEncoder(w).Encode(body); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			t.Setenv(storagequota.ServerURLEnv, server.URL)
			err := CheckStorageQuota("alice", []VolumeMount{tt.mount}, nil, nil)
			assertStorageQuotaError(t, err, tt.wantCode, tt.wantMessage, tt.failPath != "")
			if quotaCalls.Load() != 1 || usageCalls.Load() != tt.wantUsageCalls {
				t.Fatalf("provider calls: quota=%d usage=%d", quotaCalls.Load(), usageCalls.Load())
			}
		})
	}
}

func assertStorageQuotaError(t *testing.T, err error, wantCode bizerr.BizCode, wantMessage string, providerFailed bool) {
	t.Helper()
	if wantCode == 0 {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	bErr, ok := bizerr.FromError(err)
	if !ok || bErr.Code != wantCode {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(bErr.Message, wantMessage) {
		t.Fatalf("unexpected message: %s", bErr.Message)
	}
	if !providerFailed {
		return
	}
	if !strings.Contains(bErr.Message, "remove all storage mounts") {
		t.Errorf("missing recovery suggestion: %s", bErr.Message)
	}
	if strings.Contains(bErr.Message, "private provider failure") {
		t.Error("provider details leaked to user message")
	}
	if !strings.Contains(err.Error(), "private provider failure") {
		t.Error("provider cause was lost")
	}
}
