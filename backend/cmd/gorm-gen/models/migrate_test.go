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

package main

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

//nolint:gocyclo // One integration test verifies forward, repeated, rollback, and repeated rollback behavior.
func TestModelDatasetSourceMigrationAndRollback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:model_dataset_source_migration?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE datasets (id integer primary key, name text, url text, type text)`,
		`CREATE TABLE model_downloads (id integer primary key, name text, path text, category text)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create legacy table: %v", err)
		}
	}

	migration := modelDatasetSourceMigration()
	if err := migration.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := migration.Migrate(db); err != nil {
		t.Fatalf("idempotent migrate: %v", err)
	}
	for _, table := range []any{&model.ModelDatasetSource{}, &model.ModelDatasetDiscovery{}} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("missing migrated table %T", table)
		}
	}
	for table, value := range map[string]any{
		"datasets":        &model.Dataset{},
		"model_downloads": &model.ModelDownload{},
	} {
		if !db.Table(table).Migrator().HasColumn(value, "ModelDatasetSourceID") {
			t.Fatalf("%s is missing model_dataset_source_id", table)
		}
	}

	if err := migration.Rollback(db); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if err := migration.Rollback(db); err != nil {
		t.Fatalf("idempotent rollback: %v", err)
	}
	if db.Migrator().HasTable(&model.ModelDatasetSource{}) || db.Migrator().HasTable(&model.ModelDatasetDiscovery{}) {
		t.Fatal("source or discovery table remains after rollback")
	}
	if db.Table("datasets").Migrator().HasColumn(&model.Dataset{}, "ModelDatasetSourceID") {
		t.Fatal("datasets.model_dataset_source_id remains after rollback")
	}
	if db.Table("model_downloads").Migrator().HasColumn(&model.ModelDownload{}, "ModelDatasetSourceID") {
		t.Fatal("model_downloads.model_dataset_source_id remains after rollback")
	}
}
