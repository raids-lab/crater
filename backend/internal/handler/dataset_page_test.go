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
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

func newDatasetPageTestRouter(t *testing.T, visibleIDs []uint) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open dataset page test database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Dataset{}, &model.ModelDownload{}, &model.UserDataset{}, &model.AccountDataset{}); err != nil {
		t.Fatalf("migrate dataset page test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get dataset page test database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	query.SetDefault(db)
	users := []model.User{
		{Model: gorm.Model{ID: 1}, Name: "alice", Space: "/tmp/dataset-page-alice", Role: model.RoleUser, Status: model.StatusActive},
		{Model: gorm.Model{ID: 2}, Name: "bob", Space: "/tmp/dataset-page-bob", Role: model.RoleUser, Status: model.StatusActive},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("create dataset page users: %v", err)
	}
	createdAt := time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)
	datasets := []model.Dataset{
		{Model: gorm.Model{ID: 1, CreatedAt: createdAt}, Name: "Repo", Type: model.DataTypeDataset, UserID: 1},
		{Model: gorm.Model{ID: 2, CreatedAt: createdAt}, Name: "Repo", Describe: "duplicate-only", Type: model.DataTypeDataset, UserID: 2},
		{Model: gorm.Model{ID: 3, CreatedAt: createdAt}, Name: "Other", Type: model.DataTypeDataset, UserID: 1},
		{Model: gorm.Model{ID: 4, CreatedAt: createdAt}, Name: "Repo", Type: model.DataTypeModel, UserID: 1},
		{Model: gorm.Model{ID: 5, CreatedAt: createdAt}, Name: "Repo", Type: model.DataTypeModel, UserID: 2},
		{Model: gorm.Model{ID: 6, CreatedAt: createdAt}, Name: "experiment", Type: model.DataTypeDataset, UserID: 1},
		{Model: gorm.Model{ID: 7, CreatedAt: createdAt}, Name: "experiment", Type: model.DataTypeDataset, UserID: 1},
		{Model: gorm.Model{ID: 8, CreatedAt: createdAt}, Name: "Repo", Type: model.DataTypeShareFile, UserID: 1},
		{Model: gorm.Model{ID: 9, CreatedAt: createdAt}, Name: "Repo", Type: model.DataTypeShareFile, UserID: 2},
	}
	if err := db.Create(&datasets).Error; err != nil {
		t.Fatalf("create dataset page rows: %v", err)
	}
	for index, id := range visibleIDs {
		// Exercise both direct sharing and the public account's sharing scope.
		if index%2 == 0 {
			if err := db.Create(&model.UserDataset{UserID: 1, DatasetID: id}).Error; err != nil {
				t.Fatalf("share dataset with user: %v", err)
			}
		} else if err := db.Create(&model.AccountDataset{AccountID: model.DefaultAccountID, DatasetID: id}).Error; err != nil {
			t.Fatalf("share dataset with public account: %v", err)
		}
	}
	for _, category := range []model.DownloadCategory{model.DownloadCategoryDataset, model.DownloadCategoryModel} {
		if err := db.Create(&model.ModelDownload{
			Name: "Repo", Source: model.ModelSourceHuggingFace, Category: category,
			Status: model.ModelDownloadStatusReady, Path: "/tmp/dataset-page-repo", CreatorID: 1,
		}).Error; err != nil {
			t.Fatalf("create dataset page download: %v", err)
		}
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		role := model.RoleUser
		if strings.HasPrefix(c.Request.URL.Path, "/v1/admin/") {
			role = model.RoleAdmin
		}
		util.SetJWTContext(c, util.JWTMessage{UserID: 1, Username: "alice", AccountID: 2, RolePlatform: role})
		c.Next()
	})
	mgr := &DatasetMgr{}
	mgr.RegisterProtected(router.Group("/v1/dataset"))
	mgr.RegisterAdmin(router.Group("/v1/admin/dataset"))
	return router
}

func readDatasetTestPage(t *testing.T, router *gin.Engine, path string) resputil.Page[DatasetResp] {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, http.NoBody))
	if recorder.Code != http.StatusOK {
		t.Fatalf("dataset page HTTP %d: %s", recorder.Code, recorder.Body.String())
	}
	var result resputil.Response[resputil.Page[DatasetResp]]
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode dataset page: %v", err)
	}
	if result.Code != resputil.OK || result.Data.Items == nil {
		t.Fatalf("unexpected dataset page envelope: %+v", result)
	}
	return result.Data
}

func assertDatasetPageIDs(t *testing.T, page resputil.Page[DatasetResp], total int64, wantIDs []uint) {
	t.Helper()
	ids := make([]uint, 0, len(page.Items))
	for i := range page.Items {
		ids = append(ids, page.Items[i].ID)
	}
	if page.Total != total || !reflect.DeepEqual(ids, wantIDs) {
		t.Fatalf("total=%d ids=%v, want total=%d ids=%v", page.Total, ids, total, wantIDs)
	}
}

func TestDatasetPageRoutesDeduplicateBeforePagination(t *testing.T) {
	router := newDatasetPageTestRouter(t, []uint{1, 2, 3, 4, 5, 6, 7, 8, 9})
	for _, prefix := range []string{"/v1/dataset/mydataset/page", "/v1/admin/dataset/alldataset/page"} {
		t.Run(prefix, func(t *testing.T) {
			wantDatasetIDs := []uint{1, 3, 6, 7}
			wantSharedFileIDs := []uint{8, 9}
			if strings.HasPrefix(prefix, "/v1/admin/") {
				// Administrator sorts retain their existing descending ID tie-breaker.
				wantDatasetIDs = []uint{7, 6, 3, 1}
				wantSharedFileIDs = []uint{9, 8}
			}
			page := readDatasetTestPage(t, router, prefix+"?page_size=10&type=dataset&sort=createdAt")
			assertDatasetPageIDs(t, page, 4, wantDatasetIDs)
			for index, query := range []string{"page=1", "page=2", "page=3", "page=4"} {
				page = readDatasetTestPage(t, router, prefix+"?"+query+"&page_size=1&type=dataset&sort=createdAt")
				assertDatasetPageIDs(t, page, 4, []uint{wantDatasetIDs[index]})
				if page.Page != index+1 || page.PageSize != 1 {
					t.Fatalf("requested page was not preserved: %+v", page)
				}
			}
			page = readDatasetTestPage(t, router, prefix+"?page=5&page_size=1&type=dataset&sort=createdAt")
			assertDatasetPageIDs(t, page, 4, []uint{})
			if page.Page != 5 {
				t.Fatalf("empty page changed requested page: %+v", page)
			}
			page = readDatasetTestPage(t, router, prefix+"?type=model")
			assertDatasetPageIDs(t, page, 1, []uint{4})
			page = readDatasetTestPage(t, router, prefix+"?type=sharefile&sort=createdAt")
			assertDatasetPageIDs(t, page, 2, wantSharedFileIDs)
			page = readDatasetTestPage(t, router, prefix+"?type=dataset&search=repo")
			assertDatasetPageIDs(t, page, 1, []uint{1})
			page = readDatasetTestPage(t, router, prefix+"?type=dataset&search=duplicate-only")
			assertDatasetPageIDs(t, page, 0, []uint{})
		})
	}
	page := readDatasetTestPage(t, router, "/v1/dataset/mydataset/page?type=dataset&search=repo&owner=mine")
	assertDatasetPageIDs(t, page, 1, []uint{1})
	page = readDatasetTestPage(t, router, "/v1/dataset/mydataset/page?type=dataset&owner=others")
	assertDatasetPageIDs(t, page, 0, []uint{})
	page = readDatasetTestPage(t, router, "/v1/admin/dataset/alldataset/page?sort=createdAt")
	assertDatasetPageIDs(t, page, 7, []uint{9, 8, 7, 6, 4, 3, 1})
}

func TestDatasetPageCanonicalizationUsesVisibleRows(t *testing.T) {
	router := newDatasetPageTestRouter(t, []uint{2, 3})
	page := readDatasetTestPage(t, router, "/v1/dataset/mydataset/page?type=dataset&sort=createdAt")
	assertDatasetPageIDs(t, page, 2, []uint{2, 3})
	page = readDatasetTestPage(t, router, "/v1/dataset/mydataset/page?type=dataset&search=repo&owner=others")
	assertDatasetPageIDs(t, page, 1, []uint{2})
	page = readDatasetTestPage(t, router, "/v1/dataset/mydataset/page?type=model")
	assertDatasetPageIDs(t, page, 0, []uint{})
	page = readDatasetTestPage(t, router, "/v1/admin/dataset/alldataset/page?type=dataset&sort=createdAt")
	assertDatasetPageIDs(t, page, 4, []uint{7, 6, 3, 1})
}

func TestDatasetPageCanonicalizationWithNoAccess(t *testing.T) {
	router := newDatasetPageTestRouter(t, nil)
	page := readDatasetTestPage(t, router, "/v1/dataset/mydataset/page?type=dataset")
	assertDatasetPageIDs(t, page, 0, []uint{})
}

func TestDatasetLegacyRoutesRemainUnpaginated(t *testing.T) {
	router := newDatasetPageTestRouter(t, []uint{1, 2, 3, 4, 5, 6, 7, 8, 9})
	for _, path := range []string{"/v1/dataset/mydataset", "/v1/admin/dataset/alldataset"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path+"?page=99&page_size=1", http.NoBody))
		if recorder.Code != http.StatusOK {
			t.Fatalf("legacy dataset HTTP %d: %s", recorder.Code, recorder.Body.String())
		}
		var result resputil.Response[[]DatasetResp]
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode legacy dataset array: %v", err)
		}
		ids := make([]uint, 0, len(result.Data))
		for i := range result.Data {
			ids = append(ids, result.Data[i].ID)
		}
		slices.Sort(ids)
		if result.Code != resputil.OK || !reflect.DeepEqual(ids, []uint{1, 3, 4, 6, 7, 8, 9}) {
			t.Fatalf("legacy array changed: code=%d ids=%v", result.Code, ids)
		}
	}
}
