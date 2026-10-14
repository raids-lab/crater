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

package jobtemplate

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
	if err := db.AutoMigrate(&model.User{}, &model.Jobtemplate{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	query.SetDefault(db)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

func createTestUser(t *testing.T, db *gorm.DB, id uint, name string) {
	t.Helper()
	user := model.User{
		Model:  gorm.Model{ID: id},
		Name:   name,
		Role:   model.RoleUser,
		Status: model.StatusActive,
		Space:  "/tmp/" + name,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user %q: %v", name, err)
	}
}

func createTestTemplates(t *testing.T, db *gorm.DB, templates ...model.Jobtemplate) {
	t.Helper()
	for index := range templates {
		if err := db.Create(&templates[index]).Error; err != nil {
			t.Fatalf("create template %q: %v", templates[index].Name, err)
		}
	}
}

func TestListPaginatesWithStableSort(t *testing.T) {
	db := setupListTestDB(t)
	createTestUser(t, db, 1, "alice")

	createdAt := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	for id := uint(1); id <= 5; id++ {
		createTestTemplates(t, db, model.Jobtemplate{
			Model:  gorm.Model{ID: id, CreatedAt: createdAt},
			Name:   fmt.Sprintf("template-%d", id),
			UserID: 1,
		})
	}

	firstPage, total, err := List(t.Context(), ListOptions{Offset: 0, Limit: 2, Sort: "-createdAt"})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	secondPage, secondTotal, err := List(t.Context(), ListOptions{Offset: 2, Limit: 2, Sort: "-createdAt"})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}

	if total != 5 || secondTotal != 5 {
		t.Fatalf("unexpected totals: first=%d second=%d", total, secondTotal)
	}
	if len(firstPage) != 2 || firstPage[0].ID != 5 || firstPage[1].ID != 4 {
		t.Fatalf("unexpected first page order: %#v", firstPage)
	}
	if len(secondPage) != 2 || secondPage[0].ID != 3 || secondPage[1].ID != 2 {
		t.Fatalf("unexpected second page order: %#v", secondPage)
	}
}

func TestListSearchesBeforePagination(t *testing.T) {
	db := setupListTestDB(t)
	createTestUser(t, db, 1, "alice")
	createdAt := time.Date(2026, time.August, 5, 10, 0, 0, 0, time.UTC)
	createTestTemplates(t, db,
		model.Jobtemplate{Model: gorm.Model{ID: 1, CreatedAt: createdAt}, Name: "测试 Alpha", UserID: 1},
		model.Jobtemplate{Model: gorm.Model{ID: 2, CreatedAt: createdAt}, Name: "unrelated", UserID: 1},
		model.Jobtemplate{Model: gorm.Model{ID: 3, CreatedAt: createdAt}, Name: "测试 Beta", UserID: 1},
	)

	firstPage, total, err := List(t.Context(), ListOptions{
		Offset: 0,
		Limit:  1,
		Search: "测试",
		Sort:   "createdAt",
	})
	if err != nil {
		t.Fatalf("list first search page: %v", err)
	}
	secondPage, secondTotal, err := List(t.Context(), ListOptions{
		Offset: 1,
		Limit:  1,
		Search: "测试",
		Sort:   "createdAt",
	})
	if err != nil {
		t.Fatalf("list second search page: %v", err)
	}

	if total != 2 || secondTotal != 2 {
		t.Fatalf("search must count filtered rows: first=%d second=%d", total, secondTotal)
	}
	if len(firstPage) != 1 || firstPage[0].ID != 1 {
		t.Fatalf("unexpected first search page: %#v", firstPage)
	}
	if len(secondPage) != 1 || secondPage[0].ID != 3 {
		t.Fatalf("unexpected second search page: %#v", secondPage)
	}
}

func TestListReturnsNoRowsForEmptyResult(t *testing.T) {
	db := setupListTestDB(t)
	createTestUser(t, db, 1, "alice")

	items, total, err := List(t.Context(), ListOptions{
		Offset: 0,
		Limit:  10,
		Search: "missing",
		Sort:   "-createdAt",
	})
	if err != nil {
		t.Fatalf("list empty result: %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("unexpected empty result: total=%d items=%#v", total, items)
	}
}

func TestListFiltersByOwner(t *testing.T) {
	db := setupListTestDB(t)
	createTestUser(t, db, 1, "alice")
	createTestUser(t, db, 2, "bob")
	createTestTemplates(t, db,
		model.Jobtemplate{Model: gorm.Model{ID: 1}, Name: "alice-template", UserID: 1},
		model.Jobtemplate{Model: gorm.Model{ID: 2}, Name: "bob-template", UserID: 2},
	)

	mine, mineTotal, err := List(t.Context(), ListOptions{
		Limit:  10,
		Owner:  "mine",
		Sort:   "-createdAt",
		UserID: 1,
	})
	if err != nil {
		t.Fatalf("list mine: %v", err)
	}
	others, othersTotal, err := List(t.Context(), ListOptions{
		Limit:  10,
		Owner:  "others",
		Sort:   "-createdAt",
		UserID: 1,
	})
	if err != nil {
		t.Fatalf("list others: %v", err)
	}

	if mineTotal != 1 || len(mine) != 1 || mine[0].ID != 1 {
		t.Fatalf("unexpected mine result: total=%d items=%#v", mineTotal, mine)
	}
	if othersTotal != 1 || len(others) != 1 || others[0].ID != 2 {
		t.Fatalf("unexpected others result: total=%d items=%#v", othersTotal, others)
	}
}
