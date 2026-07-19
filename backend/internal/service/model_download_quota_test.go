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
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
)

func TestModelDownloadQuotaServiceEnforcesConfiguredLimits(t *testing.T) {
	db, configService, quotaService := newModelDownloadQuotaTestServices(t, "configured_limits")
	userID := uint(7)
	limit := ModelDownloadLimitConfig{
		Enabled: true, MaxConcurrent: 2, WindowHours: 2, MaxSuccessfulDownloads: 2,
	}
	updateModelDownloadQuotaTestConfig(t, configService, limit)

	downloads := make([]model.ModelDownload, 0, 2)
	for i, status := range []model.ModelDownloadStatus{
		model.ModelDownloadStatusPending, model.ModelDownloadStatusDownloading,
	} {
		download := createModelDownloadQuotaTestRecord(t, db, userID, fmt.Sprintf("active-%d", i), status)
		createModelDownloadQuotaTestSubmission(t, db, userID, download.ID, model.ModelDownloadSubmissionReserved)
		downloads = append(downloads, download)
	}
	target := createModelDownloadQuotaTestRecord(t, db, userID, "concurrent-target", model.ModelDownloadStatusPending)
	err := quotaService.Reserve(t.Context(), db, userID, target.ID, model.ModelDownloadSubmissionCreate)
	if !errors.Is(err, bizerr.RateLimit.Base) || !strings.Contains(err.Error(), "This account") {
		t.Fatalf("configured concurrent limit should return an English rate-limit error, got %v", err)
	}

	if err := db.Model(&model.ModelDownload{}).Where("creator_id = ?", userID).
		Update("status", model.ModelDownloadStatusReady).Error; err != nil {
		t.Fatal(err)
	}
	for i := range downloads {
		if err := CompleteModelDownloadQuotaReservation(t.Context(), db, downloads[i].ID, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	err = quotaService.Reserve(t.Context(), db, userID, target.ID, model.ModelDownloadSubmissionRetry)
	if !errors.Is(err, bizerr.RateLimit.Base) || !strings.Contains(err.Error(), "rolling 2-hour window") {
		t.Fatalf("configured rolling-window limit should return an English rate-limit error, got %v", err)
	}

	if err := db.Model(&model.ModelDownloadSubmission{}).
		Where("user_id = ? AND status = ?", userID, model.ModelDownloadSubmissionSucceeded).
		Update("completed_at", time.Now().Add(-3*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := quotaService.Reserve(
		t.Context(), db, userID, target.ID, model.ModelDownloadSubmissionResume,
	); err != nil {
		t.Fatalf("successful downloads outside the configured window should not count: %v", err)
	}
}

func TestModelDownloadQuotaServiceTracksJobsAcrossConfigurationChanges(t *testing.T) {
	db, configService, quotaService := newModelDownloadQuotaTestServices(t, "configuration_changes")
	limit := ModelDownloadLimitConfig{
		Enabled: false, MaxConcurrent: 1, WindowHours: 2, MaxSuccessfulDownloads: 10,
	}
	updateModelDownloadQuotaTestConfig(t, configService, limit)

	userID := uint(7)
	disabledJob := createModelDownloadQuotaTestRecord(
		t, db, userID, "disabled-job", model.ModelDownloadStatusDownloading,
	)
	if err := quotaService.Reserve(
		t.Context(), db, userID, disabledJob.ID, model.ModelDownloadSubmissionCreate,
	); err != nil {
		t.Fatalf("disabled quotas should skip checks but record the Job: %v", err)
	}
	assertModelDownloadQuotaTestSubmission(t, db, userID, disabledJob.ID)

	limit.Enabled = true
	updateModelDownloadQuotaTestConfig(t, configService, limit)
	afterEnable := createModelDownloadQuotaTestRecord(t, db, userID, "after-enable", model.ModelDownloadStatusPending)
	if err := quotaService.Reserve(
		t.Context(), db, userID, afterEnable.ID, model.ModelDownloadSubmissionCreate,
	); !errors.Is(err, bizerr.RateLimit.Base) {
		t.Fatalf("a Job started while disabled must count after quotas are enabled, got %v", err)
	}

	whitelistedUserID := uint(8)
	limit.WhitelistUserIDs = []uint{whitelistedUserID}
	updateModelDownloadQuotaTestConfig(t, configService, limit)
	whitelistedJob := createModelDownloadQuotaTestRecord(
		t, db, whitelistedUserID, "whitelisted-job", model.ModelDownloadStatusDownloading,
	)
	if err := quotaService.Reserve(
		t.Context(), db, whitelistedUserID, whitelistedJob.ID, model.ModelDownloadSubmissionCreate,
	); err != nil {
		t.Fatalf("whitelisted users should skip checks but record the Job: %v", err)
	}
	assertModelDownloadQuotaTestSubmission(t, db, whitelistedUserID, whitelistedJob.ID)

	limit.WhitelistUserIDs = nil
	updateModelDownloadQuotaTestConfig(t, configService, limit)
	afterRemoval := createModelDownloadQuotaTestRecord(
		t, db, whitelistedUserID, "after-whitelist-removal", model.ModelDownloadStatusPending,
	)
	if err := quotaService.Reserve(
		t.Context(), db, whitelistedUserID, afterRemoval.ID, model.ModelDownloadSubmissionCreate,
	); !errors.Is(err, bizerr.RateLimit.Base) {
		t.Fatalf("a whitelisted Job must count after the user is removed from the whitelist, got %v", err)
	}
}

func TestModelDownloadQuotaServiceReserveUsesSubmissionTableFromScopedDB(t *testing.T) {
	db, _, quotaService := newModelDownloadQuotaTestServices(t, "scoped_db_submission_table")
	userID := uint(7)
	download := createModelDownloadQuotaTestRecord(
		t, db, userID, "scoped-db", model.ModelDownloadStatusPending,
	)

	err := query.Use(db).Transaction(func(tx *query.Query) error {
		return quotaService.Reserve(
			t.Context(), tx.ModelDownload.WithContext(t.Context()).UnderlyingDB(),
			userID, download.ID, model.ModelDownloadSubmissionCreate,
		)
	})
	if err != nil {
		t.Fatalf("reserve with a model-download-scoped DB should use the submission table: %v", err)
	}
	assertModelDownloadQuotaTestSubmission(t, db, userID, download.ID)
}

func TestModelDownloadQuotaServiceReservesAndSettlesByOperator(t *testing.T) {
	db, configService, quotaService := newModelDownloadQuotaTestServices(t, "settlement_by_operator")
	updateModelDownloadQuotaTestConfig(t, configService, ModelDownloadLimitConfig{
		Enabled: true, MaxConcurrent: 10, WindowHours: 2, MaxSuccessfulDownloads: 1,
	})
	originalCreatorID := uint(7)
	retrierID := uint(8)

	failed := createModelDownloadQuotaTestRecord(t, db, originalCreatorID, "failed", model.ModelDownloadStatusFailed)
	createModelDownloadQuotaTestSubmission(
		t, db, originalCreatorID, failed.ID, model.ModelDownloadSubmissionReleased,
	)
	if err := quotaService.Reserve(
		t.Context(), db, retrierID, failed.ID, model.ModelDownloadSubmissionRetry,
	); err != nil {
		t.Fatalf("a released failed attempt should not consume the rolling window: %v", err)
	}
	if err := db.Model(&failed).Update("status", model.ModelDownloadStatusDownloading).Error; err != nil {
		t.Fatal(err)
	}

	next := createModelDownloadQuotaTestRecord(t, db, retrierID, "next", model.ModelDownloadStatusPending)
	if err := quotaService.Reserve(
		t.Context(), db, retrierID, next.ID, model.ModelDownloadSubmissionCreate,
	); !errors.Is(err, bizerr.RateLimit.Base) {
		t.Fatalf("an active reservation should prevent rolling-window overbooking, got %v", err)
	}
	creatorNext := createModelDownloadQuotaTestRecord(
		t, db, originalCreatorID, "creator-next", model.ModelDownloadStatusPending,
	)
	if err := quotaService.Reserve(
		t.Context(), db, originalCreatorID, creatorNext.ID, model.ModelDownloadSubmissionCreate,
	); err != nil {
		t.Fatalf("another user's retry must not consume the original creator's quota: %v", err)
	}

	if err := ReleaseModelDownloadQuotaReservation(t.Context(), db, failed.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&failed).Update("status", model.ModelDownloadStatusFailed).Error; err != nil {
		t.Fatal(err)
	}
	if err := quotaService.Reserve(
		t.Context(), db, retrierID, next.ID, model.ModelDownloadSubmissionCreate,
	); err != nil {
		t.Fatalf("failure should release the reserved rolling-window slot: %v", err)
	}
	if err := CompleteModelDownloadQuotaReservation(t.Context(), db, next.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&next).Update("status", model.ModelDownloadStatusReady).Error; err != nil {
		t.Fatal(err)
	}

	third := createModelDownloadQuotaTestRecord(t, db, retrierID, "third", model.ModelDownloadStatusPending)
	if err := quotaService.Reserve(
		t.Context(), db, retrierID, third.ID, model.ModelDownloadSubmissionCreate,
	); !errors.Is(err, bizerr.RateLimit.Base) {
		t.Fatalf("a successful download should consume the window from completion, got %v", err)
	}
}

func newModelDownloadQuotaTestServices(
	t *testing.T, name string,
) (*gorm.DB, *ConfigService, *ModelDownloadQuotaService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.SystemConfig{}, &model.PrequeueConfig{}, &model.ModelDownload{}, &model.ModelDownloadSubmission{},
	); err != nil {
		t.Fatal(err)
	}
	configService := NewConfigService(query.Use(db))
	return db, configService, NewModelDownloadQuotaService(configService)
}

func updateModelDownloadQuotaTestConfig(
	t *testing.T, configService *ConfigService, limit ModelDownloadLimitConfig,
) {
	t.Helper()
	if err := configService.UpdateModelDownloadLimitConfig(t.Context(), limit); err != nil {
		t.Fatal(err)
	}
}

func createModelDownloadQuotaTestRecord(
	t *testing.T, db *gorm.DB, creatorID uint, name string, status model.ModelDownloadStatus,
) model.ModelDownload {
	t.Helper()
	download := model.ModelDownload{
		Name: "owner/" + name, Source: model.ModelSourceModelScope,
		Category: model.DownloadCategoryModel, Revision: "main", Path: "public/Models/owner/" + name,
		Status: status, CreatorID: creatorID,
	}
	if err := db.Create(&download).Error; err != nil {
		t.Fatal(err)
	}
	return download
}

func createModelDownloadQuotaTestSubmission(
	t *testing.T,
	db *gorm.DB,
	userID uint,
	downloadID uint,
	status model.ModelDownloadSubmissionStatus,
) {
	t.Helper()
	if err := db.Create(&model.ModelDownloadSubmission{
		UserID: userID, ModelDownloadID: downloadID,
		Action: model.ModelDownloadSubmissionCreate, Status: status,
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func assertModelDownloadQuotaTestSubmission(t *testing.T, db *gorm.DB, userID, downloadID uint) {
	t.Helper()
	var count int64
	if err := db.Model(&model.ModelDownloadSubmission{}).
		Where("user_id = ? AND model_download_id = ? AND status = ?", userID, downloadID,
			model.ModelDownloadSubmissionReserved).
		Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("reserved submission count = %d, want 1", count)
	}
}
