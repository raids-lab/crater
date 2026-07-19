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
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
)

// ModelDownloadQuotaService owns the complete quota reservation state machine.
// Every Kubernetes download Job receives a reservation, including Jobs created
// while quota enforcement is disabled or for an exempt user. Keeping those
// lifecycle records makes later configuration changes take effect immediately
// for already-running Jobs.
type ModelDownloadQuotaService struct {
	configService *ConfigService
}

func NewModelDownloadQuotaService(configService *ConfigService) *ModelDownloadQuotaService {
	return &ModelDownloadQuotaService{configService: configService}
}

// Reserve checks the administrator-managed limits when they apply and records
// the Job attempt in the same transaction as the download state transition.
func (s *ModelDownloadQuotaService) Reserve(
	ctx context.Context,
	db *gorm.DB,
	userID uint,
	downloadID uint,
	action model.ModelDownloadSubmissionAction,
) error {
	if s == nil || s.configService == nil {
		return bizerr.Internal.DatabaseError.New("model download quota service is not initialized")
	}
	cfg, err := s.configService.GetModelDownloadLimitConfig(ctx)
	if err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "get model download quota")
	}

	tx := db.WithContext(ctx).Session(&gorm.Session{NewDB: true})
	exempt := !cfg.Enabled || slices.Contains(cfg.WhitelistUserIDs, userID)
	if !exempt {
		if err := lockModelDownloadQuota(tx, userID); err != nil {
			return err
		}
		if err := checkModelDownloadConcurrentQuota(tx, userID, downloadID, cfg.MaxConcurrent); err != nil {
			return err
		}
		if err := checkModelDownloadWindowQuota(
			tx, userID, cfg.WindowHours, cfg.MaxSuccessfulDownloads,
		); err != nil {
			return err
		}
	}

	submission := &model.ModelDownloadSubmission{
		UserID: userID, ModelDownloadID: downloadID, Action: action,
		Status: model.ModelDownloadSubmissionReserved,
	}
	if err := query.Use(tx).ModelDownloadSubmission.Table("model_download_submissions").
		WithContext(ctx).Create(submission); err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "record model download submission")
	}
	return nil
}

func lockModelDownloadQuota(db *gorm.DB, userID uint) error {
	if db.Name() != "postgres" {
		return nil
	}
	quotaIdentity := fmt.Sprintf("model-download-quota:%d", userID)
	if err := db.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", quotaIdentity).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "lock model download quota")
	}
	return nil
}

func checkModelDownloadConcurrentQuota(
	db *gorm.DB, userID uint, downloadID uint, maxConcurrent int64,
) error {
	activeQuery := db.Table("model_downloads AS download").
		Joins("JOIN model_download_submissions AS submission ON submission.model_download_id = download.id").
		Where("download.status IN ?", []model.ModelDownloadStatus{
			model.ModelDownloadStatusPending, model.ModelDownloadStatusDownloading,
		}).
		Where("submission.user_id = ? AND submission.status = ?", userID, model.ModelDownloadSubmissionReserved)
	if downloadID != 0 {
		activeQuery = activeQuery.Where("download.id <> ?", downloadID)
	}
	var activeCount int64
	if err := activeQuery.Distinct("download.id").Count(&activeCount).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "count concurrent model downloads")
	}
	if activeCount >= maxConcurrent {
		return bizerr.RateLimit.TooManyRequests.New(fmt.Sprintf(
			"This account can have at most %d pending or downloading tasks at the same time. "+
				"Wait for a task to finish or pause one before trying again.",
			maxConcurrent,
		))
	}
	return nil
}

func checkModelDownloadWindowQuota(
	db *gorm.DB, userID uint, windowHours int64, maxSuccessfulDownloads int64,
) error {
	windowStart := time.Now().Add(-time.Duration(windowHours) * time.Hour)
	var windowUsageCount int64
	if err := db.Table("model_download_submissions AS submission").
		Joins("LEFT JOIN model_downloads AS download ON download.id = submission.model_download_id").
		Where("submission.user_id = ?", userID).
		Where(
			"(submission.status = ? AND submission.completed_at >= ?) OR "+
				"(submission.status = ? AND download.deleted_at IS NULL AND download.status IN ?)",
			model.ModelDownloadSubmissionSucceeded, windowStart,
			model.ModelDownloadSubmissionReserved, []model.ModelDownloadStatus{
				model.ModelDownloadStatusPending, model.ModelDownloadStatusDownloading,
			},
		).
		Count(&windowUsageCount).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "count model download rolling-window usage")
	}
	if windowUsageCount >= maxSuccessfulDownloads {
		return bizerr.RateLimit.TooManyRequests.New(fmt.Sprintf(
			"This account can complete at most %d model or dataset downloads in a rolling %d-hour window. "+
				"Active downloads reserve slots; wait for one to finish, fail, or be paused before trying again.",
			maxSuccessfulDownloads, windowHours,
		))
	}
	return nil
}

// CompleteModelDownloadQuotaReservation converts the active reservation for a
// download into a successful rolling-window entry. The window starts when the
// download completes, rather than when the Job was submitted.
func CompleteModelDownloadQuotaReservation(
	ctx context.Context, db *gorm.DB, downloadID uint, completedAt time.Time,
) error {
	q := query.Use(db.WithContext(ctx).Session(&gorm.Session{NewDB: true})).
		ModelDownloadSubmission.Table("model_download_submissions")
	_, err := q.WithContext(ctx).
		Where(q.ModelDownloadID.Eq(downloadID), q.Status.Eq(string(model.ModelDownloadSubmissionReserved))).
		Updates(map[string]any{
			"status":       model.ModelDownloadSubmissionSucceeded,
			"completed_at": completedAt,
		})
	return err
}

// ReleaseModelDownloadQuotaReservation releases active reservations after a
// failed, paused, canceled, or otherwise unsuccessful download attempt.
func ReleaseModelDownloadQuotaReservation(
	ctx context.Context, db *gorm.DB, downloadID uint,
) error {
	q := query.Use(db.WithContext(ctx).Session(&gorm.Session{NewDB: true})).
		ModelDownloadSubmission.Table("model_download_submissions")
	_, err := q.WithContext(ctx).
		Where(q.ModelDownloadID.Eq(downloadID), q.Status.Eq(string(model.ModelDownloadSubmissionReserved))).
		Updates(map[string]any{
			"status":       model.ModelDownloadSubmissionReleased,
			"completed_at": nil,
		})
	return err
}
