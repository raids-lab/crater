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
	"time"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

// CompleteModelDownloadQuotaReservation converts the active reservation for a
// download into a successful rolling-window entry. The window starts when the
// download completes, rather than when the Job was submitted.
func CompleteModelDownloadQuotaReservation(
	ctx context.Context, db *gorm.DB, downloadID uint, completedAt time.Time,
) error {
	return db.WithContext(ctx).Exec(
		`UPDATE model_download_submissions
		 SET status = ?, completed_at = ?
		 WHERE model_download_id = ? AND status = ?`,
		model.ModelDownloadSubmissionSucceeded, completedAt,
		downloadID, model.ModelDownloadSubmissionReserved,
	).Error
}

// ReleaseModelDownloadQuotaReservation releases active reservations after a
// failed, paused, canceled, or otherwise unsuccessful download attempt.
func ReleaseModelDownloadQuotaReservation(
	ctx context.Context, db *gorm.DB, downloadID uint,
) error {
	return db.WithContext(ctx).Exec(
		`UPDATE model_download_submissions
		 SET status = ?, completed_at = NULL
		 WHERE model_download_id = ? AND status = ?`,
		model.ModelDownloadSubmissionReleased,
		downloadID, model.ModelDownloadSubmissionReserved,
	).Error
}
