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
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

func setupListTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Dataset{}, &model.ModelDownload{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	query.SetDefault(db)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func createTestUser(t *testing.T, db *gorm.DB, id uint, name string) {
	t.Helper()
	if err := db.Create(&model.User{
		Model:  gorm.Model{ID: id},
		Name:   name,
		Role:   model.RoleUser,
		Status: model.StatusActive,
		Space:  "/tmp/" + name,
	}).Error; err != nil {
		t.Fatalf("create user %q: %v", name, err)
	}
}

func TestListPaginatesAndFiltersBeforePagination(t *testing.T) {
	db := setupListTestDB(t)
	createTestUser(t, db, 1, "alice")
	createdAt := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	datasets := []model.Dataset{
		{Model: gorm.Model{ID: 1, CreatedAt: createdAt}, Name: "测试 Alpha", Type: model.DataTypeShareFile, UserID: 1},
		{Model: gorm.Model{ID: 2, CreatedAt: createdAt}, Name: "unrelated", Type: model.DataTypeShareFile, UserID: 1},
		{Model: gorm.Model{ID: 3, CreatedAt: createdAt}, Name: "测试 Beta", Type: model.DataTypeShareFile, UserID: 1},
	}
	if err := db.Create(&datasets).Error; err != nil {
		t.Fatalf("create datasets: %v", err)
	}

	firstPage, total, err := List(t.Context(), &ListOptions{
		AccessibleIDs: []uint{1, 2, 3},
		Offset:        0,
		Limit:         1,
		Search:        "测试",
		Type:          model.DataTypeShareFile,
		Sort:          "createdAt",
	})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	secondPage, secondTotal, err := List(t.Context(), &ListOptions{
		AccessibleIDs: []uint{1, 2, 3},
		Offset:        1,
		Limit:         1,
		Search:        "测试",
		Type:          model.DataTypeShareFile,
		Sort:          "createdAt",
	})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}

	if total != 2 || secondTotal != 2 {
		t.Fatalf("search must count filtered rows: first=%d second=%d", total, secondTotal)
	}
	if len(firstPage) != 1 || firstPage[0].ID != 1 || len(secondPage) != 1 || secondPage[0].ID != 3 {
		t.Fatalf("unexpected pages: first=%#v second=%#v", firstPage, secondPage)
	}
}

func TestListStableSortAndOwnerFilter(t *testing.T) {
	db := setupListTestDB(t)
	createTestUser(t, db, 1, "alice")
	createTestUser(t, db, 2, "bob")
	createdAt := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	datasets := []model.Dataset{
		{Model: gorm.Model{ID: 1, CreatedAt: createdAt}, Name: "one", Type: model.DataTypeShareFile, UserID: 1},
		{Model: gorm.Model{ID: 2, CreatedAt: createdAt}, Name: "two", Type: model.DataTypeShareFile, UserID: 2},
		{Model: gorm.Model{ID: 3, CreatedAt: createdAt}, Name: "three", Type: model.DataTypeDataset, UserID: 1},
	}
	if err := db.Create(&datasets).Error; err != nil {
		t.Fatalf("create datasets: %v", err)
	}

	items, total, err := List(t.Context(), &ListOptions{
		AccessibleIDs: []uint{1, 2, 3},
		Offset:        0,
		Limit:         10,
		Owner:         "others",
		Type:          model.DataTypeShareFile,
		UserID:        1,
		Sort:          "-createdAt",
	})
	if err != nil {
		t.Fatalf("list owner-filtered page: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != 2 {
		t.Fatalf("unexpected owner-filtered result: total=%d items=%#v", total, items)
	}

	allItems, _, err := List(t.Context(), &ListOptions{
		AccessibleIDs: []uint{1, 2},
		Offset:        0,
		Limit:         1,
		Type:          model.DataTypeShareFile,
		Sort:          "-createdAt",
	})
	if err != nil {
		t.Fatalf("list stable page: %v", err)
	}
	if len(allItems) != 1 || allItems[0].ID != 2 {
		t.Fatalf("stable descending order must use ID tie-breaker: %#v", allItems)
	}
}

func TestListReturnsEmptyForNoAccessibleIDs(t *testing.T) {
	items, total, err := List(t.Context(), &ListOptions{Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("list empty accessible set: %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("unexpected empty result: total=%d items=%#v", total, items)
	}
}

func TestListReturnsEmptyPageWhenOffsetIsOutOfRange(t *testing.T) {
	db := setupListTestDB(t)
	createTestUser(t, db, 1, "alice")
	for id := uint(1); id <= 2; id++ {
		if err := db.Create(&model.Dataset{
			Model:  gorm.Model{ID: id},
			Name:   fmt.Sprintf("sharefile-%d", id),
			Type:   model.DataTypeShareFile,
			UserID: 1,
		}).Error; err != nil {
			t.Fatalf("create dataset %d: %v", id, err)
		}
	}

	items, total, err := List(t.Context(), &ListOptions{
		AccessibleIDs: []uint{1, 2},
		Offset:        10,
		Limit:         10,
		Type:          model.DataTypeShareFile,
		Sort:          "-createdAt",
	})
	if err != nil {
		t.Fatalf("list out-of-range page: %v", err)
	}
	if total != 2 || len(items) != 0 {
		t.Fatalf("out-of-range page must be empty while retaining total: total=%d items=%#v", total, items)
	}
}
