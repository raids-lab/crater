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

package service

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
)

//nolint:gocyclo // The permission matrix covers setup, revocation and denied access in each subtest.
func TestImageBuildDetailVisibility(t *testing.T) {
	tests := []struct {
		name         string
		owner        bool
		admin        bool
		public       bool
		account      uint
		sharedUser   uint
		withoutImage bool
		deletedImage bool
		deletedShare bool
		deletedBuild bool
		wantCode     bizerr.BizCode
		wantManage   bool
	}{
		{name: "owner private", owner: true, wantManage: true},
		{name: "owner public", owner: true, public: true, wantManage: true},
		{name: "owner unfinished", owner: true, withoutImage: true, wantManage: true},
		{name: "legacy public", public: true},
		{name: "public account", account: model.DefaultAccountID},
		{name: "current account", account: 20},
		{name: "direct share", sharedUser: 2},
		{name: "private", wantCode: bizerr.Forbidden.PermissionDenied},
		{name: "other account", account: 30, wantCode: bizerr.Forbidden.PermissionDenied},
		{name: "other user", sharedUser: 3, wantCode: bizerr.Forbidden.PermissionDenied},
		{name: "unpublished build", withoutImage: true, wantCode: bizerr.Forbidden.PermissionDenied},
		{name: "deleted image", public: true, deletedImage: true, wantCode: bizerr.Forbidden.PermissionDenied},
		{name: "revoked user share", sharedUser: 2, deletedShare: true, wantCode: bizerr.Forbidden.PermissionDenied},
		{name: "revoked account share", account: 20, deletedShare: true, wantCode: bizerr.Forbidden.PermissionDenied},
		{name: "deleted build", public: true, deletedBuild: true, wantCode: bizerr.NotFound.DataBaseNotFound},
		{name: "admin private", admin: true, wantManage: true},
		{name: "admin unfinished", admin: true, withoutImage: true, wantManage: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := imageDetailTestDB(t)
			build := model.Kaniko{UserID: 1, ImagePackName: "build", ImageLink: "image:tag", Status: model.BuildJobPending}
			if err := db.Create(&build).Error; err != nil {
				t.Fatal(err)
			}
			image := model.Image{UserID: 1, ImagePackName: ptr.To("build"), ImageLink: "image:tag", IsPublic: tt.public}
			if !tt.withoutImage {
				if err := db.Create(&image).Error; err != nil {
					t.Fatal(err)
				}
			}
			if tt.account != 0 {
				share := model.ImageAccount{ImageID: image.ID, AccountID: tt.account}
				if err := db.Create(&share).Error; err != nil {
					t.Fatal(err)
				}
				if tt.deletedShare {
					if err := db.Delete(&share).Error; err != nil {
						t.Fatal(err)
					}
				}
			}
			if tt.sharedUser != 0 {
				share := model.ImageUser{ImageID: image.ID, UserID: tt.sharedUser}
				if err := db.Create(&share).Error; err != nil {
					t.Fatal(err)
				}
				if tt.deletedShare {
					if err := db.Delete(&share).Error; err != nil {
						t.Fatal(err)
					}
				}
			}
			if tt.deletedImage {
				if err := db.Delete(&image).Error; err != nil {
					t.Fatal(err)
				}
			}
			if tt.deletedBuild {
				if err := db.Delete(&build).Error; err != nil {
					t.Fatal(err)
				}
			}
			userID := uint(2)
			if tt.owner {
				userID = 1
			}
			result, manage, err := NewImageBuildDetailService(query.Use(db)).Get(context.Background(), "build", userID, 20, tt.admin)
			if tt.wantCode != 0 {
				var be *bizerr.BizError
				if !errors.As(err, &be) || be.Code != tt.wantCode {
					t.Fatalf("error = %v, want code %d", err, tt.wantCode)
				}
				if result != nil || manage {
					t.Fatal("denied read returned build access")
				}
				return
			}
			if err != nil || result == nil || result.ID != build.ID || manage != tt.wantManage {
				t.Fatalf("build=%v manage=%v err=%v", result, manage, err)
			}
		})
	}
}

func TestImageBuildDetailMissingAndDatabaseFailure(t *testing.T) {
	db := imageDetailTestDB(t)
	s := NewImageBuildDetailService(query.Use(db))
	_, _, err := s.Get(context.Background(), "missing", 2, 20, false)
	var be *bizerr.BizError
	if !errors.As(err, &be) || be.Code != bizerr.NotFound.DataBaseNotFound {
		t.Fatalf("missing: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	_, _, err = s.Get(context.Background(), "missing", 2, 20, false)
	if !errors.As(err, &be) || be.Code != bizerr.Internal.DatabaseError {
		t.Fatalf("database: %v", err)
	}
}

func imageDetailTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(&model.Kaniko{}, &model.Image{}, &model.ImageAccount{}, &model.ImageUser{}); err != nil {
		t.Fatal(err)
	}
	return db
}
