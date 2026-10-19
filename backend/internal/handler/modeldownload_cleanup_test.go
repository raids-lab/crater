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

package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

func TestDownloadStoragePathsOverlap(t *testing.T) {
	for _, testCase := range []struct {
		name, cleanup, history string
		want                   bool
	}{
		{"same", "public/Models/owner/model", "public/Models/owner/model", true},
		{"historical child", "public/Models/owner/model", "public/Models/owner/model/modelscope/old", true},
		{"historical parent", "public/Models/owner/model/hf/main", "public/Models/owner/model", true},
		{"siblings", "public/Models/owner/model/hf/main", "public/Models/owner/model/ms/main", false},
		{"prefix only", "public/Models/owner/model", "public/Models/owner/model-extra", false},
		{"different category", "public/Models/owner/model", "public/Datasets/owner/model", false},
		{"normalized", `public\Models\owner\model\`, "public/Models/owner/model/hf/old", true},
		{"empty history", "public/Models/owner/model", "", false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := downloadStoragePathsOverlap(testCase.cleanup, testCase.history); got != testCase.want {
				t.Fatalf("paths overlap = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestOutputCleanupInvalidatesOnlyAffectedSoftDeletedReadyDownloads(t *testing.T) {
	db := newOutputCleanupTestDB(t)
	current := model.ModelDownload{
		Name: "owner/model", Source: model.ModelSourceHuggingFace, Category: model.DownloadCategoryModel,
		Revision: "new", Path: "public/Models/owner/model", Status: model.ModelDownloadStatusPending,
	}
	if err := db.Create(&current).Error; err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name, resource, storagePath string
		category                    model.DownloadCategory
		status                      model.ModelDownloadStatus
		deleted, invalidated        bool
	}{
		{"same directory", current.Name, current.Path, current.Category, model.ModelDownloadStatusReady, true, true},
		{"historical child", current.Name, current.Path + "/modelscope/old", current.Category,
			model.ModelDownloadStatusReady, true, true},
		{"disjoint path", current.Name, current.Path + "-extra", current.Category, model.ModelDownloadStatusReady, true, false},
		{"different name", "other/model", current.Path, current.Category, model.ModelDownloadStatusReady, true, false},
		{"dataset", current.Name, "public/Datasets/owner/model", model.DownloadCategoryDataset,
			model.ModelDownloadStatusReady, true, false},
		{"failed", current.Name, current.Path, current.Category, model.ModelDownloadStatusFailed, true, false},
		{"paused", current.Name, current.Path, current.Category, model.ModelDownloadStatusPaused, true, false},
		{"active ready", current.Name, current.Path, current.Category, model.ModelDownloadStatusReady, false, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			history := model.ModelDownload{
				Name: testCase.resource, Source: model.ModelSourceModelScope, Category: testCase.category,
				Revision: testCase.name, Path: testCase.storagePath, Status: testCase.status, Message: "previous message",
			}
			if err := db.Create(&history).Error; err != nil {
				t.Fatal(err)
			}
			if testCase.deleted {
				if err := db.Delete(&history).Error; err != nil {
					t.Fatal(err)
				}
			}
			deletedAt := history.DeletedAt
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if err := invalidateSoftDeletedDownloadsForCleanup(c, db, &current); err != nil {
				t.Fatal(err)
			}
			var got model.ModelDownload
			if err := db.Unscoped().First(&got, history.ID).Error; err != nil {
				t.Fatal(err)
			}
			if got.DeletedAt.Valid != deletedAt.Valid || !got.DeletedAt.Time.Equal(deletedAt.Time) {
				t.Fatalf("cleanup changed soft deletion: %#v", got.DeletedAt)
			}
			if testCase.invalidated {
				if got.Status != model.ModelDownloadStatusFailed || !strings.Contains(got.Message, "download this resource again") {
					t.Fatalf("invalidated history = status %s message %q", got.Status, got.Message)
				}
			} else if got.Status != testCase.status || got.Message != history.Message {
				t.Fatalf("unaffected history changed: status %s message %q", got.Status, got.Message)
			}
		})
	}
}

func TestOutputCleanupInvalidationRollsBackOnDatabaseError(t *testing.T) {
	db := newOutputCleanupTestDB(t)
	history := model.ModelDownload{
		Name: "owner/model", Source: model.ModelSourceModelScope, Category: model.DownloadCategoryModel,
		Path: "public/Models/owner/model", Status: model.ModelDownloadStatusReady,
	}
	if err := db.Create(&history).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&history).Error; err != nil {
		t.Fatal(err)
	}
	injectedError := gorm.ErrInvalidDB
	if err := db.Callback().Update().Before("gorm:update").Register("test:reject_cleanup_update", func(tx *gorm.DB) {
		_ = tx.AddError(injectedError)
	}); err != nil {
		t.Fatal(err)
	}
	current := history
	current.ID = history.ID + 1
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if err := invalidateSoftDeletedDownloadsForCleanup(c, db, &current); err == nil {
		t.Fatal("cleanup preparation must fail when history cannot be invalidated")
	}
	var got model.ModelDownload
	if err := db.Unscoped().First(&got, history.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.ModelDownloadStatusReady || !got.DeletedAt.Valid {
		t.Fatalf("failed transaction changed history: status %s deleted %v", got.Status, got.DeletedAt.Valid)
	}
}

func newOutputCleanupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.ModelDownload{}); err != nil {
		t.Fatal(err)
	}
	return db
}
