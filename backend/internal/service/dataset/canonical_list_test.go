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

package dataset

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

func createCanonicalListFixtures(t *testing.T, db *gorm.DB) {
	t.Helper()
	createTestUser(t, db, 1, "alice")
	createTestUser(t, db, 2, "bob")
	createdAt := time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)
	datasets := []model.Dataset{
		{Model: gorm.Model{ID: 1, CreatedAt: createdAt}, Name: "Repo", Type: model.DataTypeDataset, UserID: 1},
		{Model: gorm.Model{ID: 2, CreatedAt: createdAt}, Name: "\t repo \u3000", Describe: "duplicate-only", Type: model.DataTypeDataset, UserID: 2},
		{Model: gorm.Model{ID: 3, CreatedAt: createdAt}, Name: "Other", Type: model.DataTypeDataset, UserID: 1},
		{Model: gorm.Model{ID: 4, CreatedAt: createdAt}, Name: "Repo", Type: model.DataTypeModel, UserID: 1},
		{Model: gorm.Model{ID: 5, CreatedAt: createdAt}, Name: "Repo", Type: model.DataTypeModel, UserID: 2},
		{Model: gorm.Model{ID: 6, CreatedAt: createdAt}, Name: "experiment", Type: model.DataTypeDataset, UserID: 1},
		{Model: gorm.Model{ID: 7, CreatedAt: createdAt}, Name: "experiment", Type: model.DataTypeDataset, UserID: 1},
		{Model: gorm.Model{ID: 8, CreatedAt: createdAt}, Name: "Repo", Type: model.DataTypeShareFile, UserID: 1},
		{Model: gorm.Model{ID: 9, CreatedAt: createdAt}, Name: "Repo", Type: model.DataTypeShareFile, UserID: 2},
	}
	if err := db.Create(&datasets).Error; err != nil {
		t.Fatalf("create canonical list datasets: %v", err)
	}
	for _, category := range []model.DownloadCategory{model.DownloadCategoryDataset, model.DownloadCategoryModel} {
		if err := db.Create(&model.ModelDownload{
			Name: "Repo", Source: model.ModelSourceHuggingFace, Category: category,
			Status: model.ModelDownloadStatusReady, Path: "/tmp/canonical-repo", CreatorID: 1,
		}).Error; err != nil {
			t.Fatalf("create ready download: %v", err)
		}
	}
}

func datasetIDs(items []*model.Dataset) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestListCanonicalizesBeforeCountAndPagination(t *testing.T) {
	db := setupListTestDB(t)
	createCanonicalListFixtures(t, db)
	options := ListOptions{
		AccessibleIDs: []uint{1, 2, 3, 4, 5, 6, 7, 8, 9},
		Limit:         1, Type: model.DataTypeDataset, Sort: "createdAt",
	}
	for offset, wantID := range []uint{1, 3, 6, 7} {
		options.Offset = offset
		items, total, err := List(t.Context(), &options)
		if err != nil {
			t.Fatalf("list offset %d: %v", offset, err)
		}
		if total != 4 || !reflect.DeepEqual(datasetIDs(items), []uint{wantID}) {
			t.Fatalf("offset %d: total=%d ids=%v, want total=4 id=%d", offset, total, datasetIDs(items), wantID)
		}
	}
	options.Offset = 0
	options.Limit = 10
	items, total, err := List(t.Context(), &options)
	if err != nil || total != 4 || !reflect.DeepEqual(datasetIDs(items), []uint{1, 3, 6, 7}) {
		t.Fatalf("same-page duplicates: total=%d ids=%v err=%v", total, datasetIDs(items), err)
	}
}

func TestListCanonicalFiltersAndPermissionScope(t *testing.T) {
	db := setupListTestDB(t)
	createCanonicalListFixtures(t, db)
	tests := []struct {
		name    string
		options ListOptions
		wantIDs []uint
	}{
		{name: "search and owner", options: ListOptions{Search: "repo", Owner: "mine", UserID: 1, Type: model.DataTypeDataset}, wantIDs: []uint{1}},
		{name: "duplicate metadata is not searchable", options: ListOptions{Search: "duplicate-only", Type: model.DataTypeDataset}, wantIDs: []uint{}},
		{name: "duplicate owner is not counted", options: ListOptions{Owner: "others", UserID: 1, Type: model.DataTypeDataset}, wantIDs: []uint{}},
		{name: "model remains a separate type", options: ListOptions{Type: model.DataTypeModel}, wantIDs: []uint{4}},
		{name: "shared files are not deduplicated", options: ListOptions{Type: model.DataTypeShareFile}, wantIDs: []uint{8, 9}},
		{name: "inaccessible lower ID cannot suppress visible rows", options: ListOptions{AccessibleIDs: []uint{3, 5}, Type: model.DataTypeModel}, wantIDs: []uint{5}},
		{name: "unmatched download names preserve the legacy rule", options: ListOptions{AccessibleIDs: []uint{2, 3}, Type: model.DataTypeDataset}, wantIDs: []uint{2, 3}},
		{name: "all types", options: ListOptions{}, wantIDs: []uint{1, 3, 4, 6, 7, 8, 9}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := tt.options
			if options.AccessibleIDs == nil {
				options.AccessibleIDs = []uint{1, 2, 3, 4, 5, 6, 7, 8, 9}
			}
			options.Sort, options.Limit = "createdAt", 20
			items, total, err := List(t.Context(), &options)
			if err != nil || total != int64(len(tt.wantIDs)) || !reflect.DeepEqual(datasetIDs(items), tt.wantIDs) {
				t.Fatalf("total=%d ids=%v err=%v, want ids=%v", total, datasetIDs(items), err, tt.wantIDs)
			}
		})
	}
}

func TestListCanonicalizationIgnoresDeletedAndUnreadyDownloads(t *testing.T) {
	db := setupListTestDB(t)
	createCanonicalListFixtures(t, db)
	if err := db.Where("category = ?", model.DownloadCategoryDataset).Delete(&model.ModelDownload{}).Error; err != nil {
		t.Fatalf("delete download: %v", err)
	}
	if err := db.Model(&model.ModelDownload{}).Where("category = ?", model.DownloadCategoryModel).
		Update("status", model.ModelDownloadStatusPending).Error; err != nil {
		t.Fatalf("mark download pending: %v", err)
	}
	items, total, err := List(t.Context(), &ListOptions{
		AccessibleIDs: []uint{1, 2, 4, 5}, Limit: 20, Sort: "createdAt",
	})
	if err != nil || total != 4 || !reflect.DeepEqual(datasetIDs(items), []uint{1, 2, 4, 5}) {
		t.Fatalf("deleted/unready downloads: total=%d ids=%v err=%v", total, datasetIDs(items), err)
	}
}

func TestListCanonicalizationIgnoresDeletedDatasetRepresentative(t *testing.T) {
	db := setupListTestDB(t)
	createCanonicalListFixtures(t, db)
	if err := db.Delete(&model.Dataset{}, 4).Error; err != nil {
		t.Fatalf("delete original model: %v", err)
	}
	items, total, err := List(t.Context(), &ListOptions{
		AccessibleIDs: []uint{4, 5}, Limit: 20, Type: model.DataTypeModel, Sort: "createdAt",
	})
	if err != nil || total != 1 || !reflect.DeepEqual(datasetIDs(items), []uint{5}) {
		t.Fatalf("deleted representative: total=%d ids=%v err=%v", total, datasetIDs(items), err)
	}
}

func TestCanonicalListScopeBuildsParameterizedPostgresQuery(t *testing.T) {
	db := setupListTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get isolated database connection: %v", err)
	}
	// Use the PostgreSQL SQL builder without opening a real PostgreSQL connection.
	postgresDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("initialize PostgreSQL dry-run builder: %v", err)
	}
	var items []*model.Dataset
	result := CanonicalListScope(postgresDB.WithContext(t.Context()).Model(&model.Dataset{}).
		Where("id IN ?", []uint{1, 2})).Where("user_id = ?", 1).Limit(1).Find(&items)
	if result.Error != nil {
		t.Fatalf("build PostgreSQL canonical query: %v", result.Error)
	}
	sql := result.Statement.SQL.String()
	for _, fragment := range []string{"NOT EXISTS", "canonical_datasets.id < datasets.id", "LOWER(TRIM", "$1"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("PostgreSQL query is missing %q: %s", fragment, sql)
		}
	}
	if strings.Count(sql, "deleted_at") < 5 || len(result.Statement.Vars) == 0 {
		t.Fatalf("PostgreSQL query lost soft-delete scopes or parameters: %s", sql)
	}
}
