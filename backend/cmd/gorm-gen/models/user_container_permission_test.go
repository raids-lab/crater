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
)

//nolint:gocyclo // One integration test verifies defaults, existing rows, idempotency and rollback.
func TestUserContainerSysAdminMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("CREATE TABLE users (id integer primary key, name text)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO users (id, name) VALUES (?, ?)", 1, "existing-user").Error; err != nil {
		t.Fatal(err)
	}
	migration := userContainerSysAdminMigration()
	for range 2 {
		if err := migration.Migrate(db); err != nil {
			t.Fatal(err)
		}
	}
	var user struct{ ContainerSysAdminEnabled bool }
	if err := db.Table("users").Where("id = ?", 1).First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.ContainerSysAdminEnabled {
		t.Fatal("granted existing user by default")
	}
	if err := db.Exec("INSERT INTO users (id, name) VALUES (?, ?)", 2, "new-user").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("users").Where("id = ?", 2).First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.ContainerSysAdminEnabled {
		t.Fatal("granted new user by default")
	}
	for range 2 {
		if err := migration.Rollback(db); err != nil {
			t.Fatal(err)
		}
	}
	if db.Table("users").Migrator().HasColumn("users", "container_sys_admin_enabled") {
		t.Fatal("column remains after rollback")
	}
	var count int64
	if err := db.Table("users").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatal("rollback removed users")
	}
}
