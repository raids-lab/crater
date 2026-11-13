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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
)

const (
	testDownloadResourceName  = "owner/repository"
	testDownloadResourcePath  = "public/Models/owner/repository"
	testUnrelatedResourcePath = "user/alice/repository"
)

func newDownloadResourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "resources.db")), &gorm.Config{
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
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.AutoMigrate(&model.Dataset{}, &model.ModelDownload{}, &model.User{}, &model.UserModelDownload{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestDownloadResourceUsesStorageIdentityInsteadOfName(t *testing.T) {
	for _, category := range []model.DownloadCategory{model.DownloadCategoryModel, model.DownloadCategoryDataset} {
		t.Run(string(category), func(t *testing.T) {
			t.Parallel()
			db := newDownloadResourceTestDB(t)
			mgr := &ModelDownloadMgr{}
			download := model.ModelDownload{
				Name: testDownloadResourceName, Category: category, Status: model.ModelDownloadStatusReady,
				Path: modelDownloadStoragePath(modelDownloadBasePath(category), testDownloadResourceName),
			}
			physicalPath := mgr.convertToPhysicalPath(download.Path)
			otherType := model.DataTypeModel
			if category == model.DownloadCategoryModel {
				otherType = model.DataTypeDataset
			}
			resources := []model.Dataset{
				{Name: download.Name, URL: testUnrelatedResourcePath, Type: model.DataType(category)},
				{Name: download.Name, URL: physicalPath, Type: otherType},
				{Name: "renamed resource", URL: physicalPath, Type: model.DataType(category)},
			}
			if err := db.Create(&resources).Error; err != nil {
				t.Fatal(err)
			}
			id, err := mgr.findDownloadResourceID(t.Context(), query.Use(db), &download)
			if err != nil {
				t.Fatal(err)
			}
			if id == nil || *id != resources[2].ID {
				t.Fatalf("linked resource = %v, want renamed resource ID %d", id, resources[2].ID)
			}
		})
	}
}

func TestDownloadResourceOmitsUnavailableResource(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status model.ModelDownloadStatus
		path   string
	}{
		{name: "pending", status: model.ModelDownloadStatusPending, path: testDownloadResourcePath},
		{name: "downloading", status: model.ModelDownloadStatusDownloading, path: testDownloadResourcePath},
		{name: "paused", status: model.ModelDownloadStatusPaused, path: testDownloadResourcePath},
		{name: "failed", status: model.ModelDownloadStatusFailed, path: testDownloadResourcePath},
		{name: "missing path", status: model.ModelDownloadStatusReady},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			db := newDownloadResourceTestDB(t)
			mgr := &ModelDownloadMgr{}
			resource := model.Dataset{Name: testDownloadResourceName, URL: mgr.convertToPhysicalPath(tc.path), Type: model.DataTypeModel}
			if err := db.Create(&resource).Error; err != nil {
				t.Fatal(err)
			}
			download := model.ModelDownload{
				Name: resource.Name, Category: model.DownloadCategoryModel, Status: tc.status, Path: tc.path,
			}
			id, err := mgr.findDownloadResourceID(t.Context(), query.Use(db), &download)
			if err != nil || id != nil {
				t.Fatalf("unavailable resource = %v, error = %v, want no link", id, err)
			}
		})
	}
}

func TestDownloadResourceDoesNotFallbackToSameNameAfterDeletion(t *testing.T) {
	db := newDownloadResourceTestDB(t)
	mgr := &ModelDownloadMgr{}
	download := model.ModelDownload{
		Name: testDownloadResourceName, Category: model.DownloadCategoryDataset,
		Status: model.ModelDownloadStatusReady, Path: "public/Datasets/owner/repository/huggingface/old-revision",
	}
	resources := []model.Dataset{
		{Name: download.Name, URL: testUnrelatedResourcePath, Type: model.DataTypeDataset},
		{Name: download.Name, URL: mgr.convertToPhysicalPath(download.Path), Type: model.DataTypeDataset},
	}
	if err := db.Create(&resources).Error; err != nil {
		t.Fatal(err)
	}
	id, err := mgr.findDownloadResourceID(t.Context(), query.Use(db), &download)
	if err != nil || id == nil || *id != resources[1].ID {
		t.Fatalf("historical download resource = %v, error = %v, want ID %d", id, err, resources[1].ID)
	}
	if err := db.Delete(&resources[1]).Error; err != nil {
		t.Fatal(err)
	}
	id, err = mgr.findDownloadResourceID(t.Context(), query.Use(db), &download)
	if err != nil || id != nil {
		t.Fatalf("deleted resource = %v, error = %v, want no link", id, err)
	}
}

func TestGetDownloadReturnsResourceIDAndOmitsDeletedResource(t *testing.T) {
	db := newDownloadResourceTestDB(t)
	// This handler uses the application's default query. Keep this test serial
	// and restore it before closing its database so other tests stay isolated.
	previous := *query.Q
	query.SetDefault(db)
	t.Cleanup(func() { *query.Q = previous })

	mgr := &ModelDownloadMgr{}
	download := model.ModelDownload{
		Name: testDownloadResourceName, Category: model.DownloadCategoryModel,
		Status: model.ModelDownloadStatusReady, Path: testDownloadResourcePath, CreatorID: 7,
	}
	if err := db.Create(&download).Error; err != nil {
		t.Fatal(err)
	}
	resources := []model.Dataset{
		{Name: download.Name, URL: testUnrelatedResourcePath, Type: model.DataTypeModel},
		{Name: download.Name, URL: mgr.convertToPhysicalPath(download.Path), Type: model.DataTypeModel},
	}
	if err := db.Create(&resources).Error; err != nil {
		t.Fatal(err)
	}
	data := requestDownloadResourceDetail(t, mgr, download.ID)
	var resourceID uint
	if err := json.Unmarshal(data["resourceId"], &resourceID); err != nil {
		t.Fatal(err)
	}
	if resourceID != resources[1].ID {
		t.Fatalf("detail resourceId = %d, want %d", resourceID, resources[1].ID)
	}
	if err := db.Delete(&resources[1]).Error; err != nil {
		t.Fatal(err)
	}
	data = requestDownloadResourceDetail(t, mgr, download.ID)
	if value, ok := data["resourceId"]; ok {
		t.Fatalf("detail returned resourceId %s for a deleted resource", value)
	}
}

func requestDownloadResourceDetail(t *testing.T, mgr *ModelDownloadMgr, id uint) map[string]json.RawMessage {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		util.SetJWTContext(c, util.JWTMessage{UserID: 7, Username: "alice", RolePlatform: model.RoleUser})
		c.Next()
	})
	mgr.RegisterProtected(router.Group("/v1/model-download"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/model-download/models/downloads/%d", id), http.NoBody))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET download returned HTTP %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response.Data
}
