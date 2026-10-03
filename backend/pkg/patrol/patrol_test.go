/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package patrol

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

func TestUpdateUserSpaceSizeCache(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user_space_size_cache?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.Exec(`CREATE TABLE users (
		id integer primary key,
		created_at datetime,
		updated_at datetime,
		deleted_at datetime,
		name text,
		space text
	)`).Error; err != nil {
		t.Fatalf("create users table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE user_space_sizes (
		id integer primary key autoincrement,
		user_id integer not null unique,
		size integer,
		created_at datetime,
		updated_at datetime
	)`).Error; err != nil {
		t.Fatalf("create user space sizes table: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (id, name, space) VALUES (?, ?, ?)`, 7, "alice", "alice-home").Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := UpdateUserSpaceSizeCache(context.Background(), db, 7, 1024); err != nil {
		t.Fatalf("create usage cache: %v", err)
	}
	if err := UpdateUserSpaceSizeCache(context.Background(), db, 7, 2048); err != nil {
		t.Fatalf("update usage cache: %v", err)
	}

	var cached model.UserSpaceSize
	if err := db.Where("user_id = ?", 7).First(&cached).Error; err != nil {
		t.Fatalf("read usage cache: %v", err)
	}
	if cached.Size != 2048 {
		t.Fatalf("unexpected usage cache size: %d", cached.Size)
	}

	var count int64
	if err := db.Model(&model.UserSpaceSize{}).Where("user_id = ?", 7).Count(&count).Error; err != nil {
		t.Fatalf("count usage cache rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one usage cache row, got %d", count)
	}
}
