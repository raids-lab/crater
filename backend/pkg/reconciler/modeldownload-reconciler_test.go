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

package reconciler

import (
	"context"
	"strings"
	"testing"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

func TestClassifyDownloadFailure(t *testing.T) {
	tests := []struct {
		name string
		logs string
		want string
	}{
		{name: "gated", logs: "Access to model is restricted", want: "gated"},
		{name: "authentication", logs: "HTTP 401 Unauthorized", want: "access denied"},
		{name: "missing revision", logs: "revision not found (404)", want: "repository or revision not found"},
		{name: "validated missing revision", logs: "[ERROR] revision_not_found: 'main'", want: "requested revision does not exist"},
		{name: "storage", logs: "write failed: no space left on device", want: "no space left"},
		{name: "network", logs: "connection reset by peer", want: "network error"},
		{name: "fallback", logs: "trace\ncustom downloader error\n", want: "custom downloader error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyDownloadFailure(tt.logs); !strings.Contains(got, tt.want) {
				t.Fatalf("classifyDownloadFailure() = %q, want substring %q", got, tt.want)
			}
		})
	}
}

func TestStoredLogTailAndDescription(t *testing.T) {
	logs := strings.Repeat("old line\n", 20) + "[DESC] Useful repository summary.\n"
	truncated := truncateLogTail(logs, 64)

	if len(truncated) > 64 {
		t.Fatalf("stored log tail has %d bytes, want at most 64", len(truncated))
	}
	if got := parseDescriptionFromLogs(truncated); got != "Useful repository summary." {
		t.Fatalf("parseDescriptionFromLogs() = %q", got)
	}
}

func TestParseRepositoryMetadata(t *testing.T) {
	metadata := parseRepositoryMetadata(`noise
[META] {"downloads":4235273,"likes":333,"updated_at":"2025-07-26T16:12:41Z","tags":["text-generation"]}
`)

	if metadata.Downloads != 4235273 || metadata.Likes != 333 || metadata.UpdatedAt != "2025-07-26T16:12:41Z" {
		t.Fatalf("unexpected repository metadata: %#v", metadata)
	}
	if strings.Join(metadata.Tags, ",") != "text-generation" {
		t.Fatalf("unexpected repository tags: %#v", metadata.Tags)
	}
}

func TestDatasetExtraForDownloadPreservesTags(t *testing.T) {
	url := "https://modelscope.cn/models/Qwen/Qwen3-32B"
	extra := datasetExtraForDownload(model.ExtraContent{Tags: []string{"llm"}}, &model.ModelDownload{
		Source: model.ModelSourceModelScope,
	}, url, []string{"text-generation"})

	if strings.Join(extra.Tags, ",") != "llm,modelscope,text-generation" {
		t.Fatalf("unexpected tags: %#v", extra.Tags)
	}
	if extra.WebURL == nil || *extra.WebURL != url || extra.Editable {
		t.Fatalf("unexpected dataset extra: %#v", extra)
	}
}

func TestUpdateDownloadStatusSettlesQuotaReservations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:model_download_quota_reconciler?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ModelDownload{}, &model.ModelDownloadSubmission{}); err != nil {
		t.Fatal(err)
	}
	query.SetDefault(db)
	reconciler := &ModelDownloadReconciler{}

	successful := model.ModelDownload{
		Name: "owner/success", Source: model.ModelSourceModelScope,
		Category: model.DownloadCategoryModel, Revision: "main", Path: "public/Models/owner/success",
		Status: model.ModelDownloadStatusDownloading, CreatorID: 7,
	}
	if err := db.Create(&successful).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ModelDownloadSubmission{
		UserID: 7, ModelDownloadID: successful.ID,
		Action: model.ModelDownloadSubmissionRetry, Status: model.ModelDownloadSubmissionReserved,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := reconciler.updateDownloadStatus(t.Context(), &successful, model.ModelDownloadStatusReady); err != nil {
		t.Fatal(err)
	}
	assertQuotaSubmissionSettlement(t, db, successful.ID, model.ModelDownloadSubmissionSucceeded, true)

	failed := model.ModelDownload{
		Name: "owner/failed", Source: model.ModelSourceModelScope,
		Category: model.DownloadCategoryModel, Revision: "main", Path: "public/Models/owner/failed",
		Status: model.ModelDownloadStatusDownloading, CreatorID: 7,
	}
	if err := db.Create(&failed).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ModelDownloadSubmission{
		UserID: 7, ModelDownloadID: failed.ID,
		Action: model.ModelDownloadSubmissionCreate, Status: model.ModelDownloadSubmissionReserved,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := reconciler.updateDownloadStatus(t.Context(), &failed, model.ModelDownloadStatusFailed); err != nil {
		t.Fatal(err)
	}
	assertQuotaSubmissionSettlement(t, db, failed.ID, model.ModelDownloadSubmissionReleased, false)
}

func assertQuotaSubmissionSettlement(
	t *testing.T,
	db *gorm.DB,
	downloadID uint,
	wantStatus model.ModelDownloadSubmissionStatus,
	wantCompletion bool,
) {
	t.Helper()
	var submission model.ModelDownloadSubmission
	if err := db.Where("model_download_id = ?", downloadID).First(&submission).Error; err != nil {
		t.Fatal(err)
	}
	if submission.Status != wantStatus {
		t.Fatalf("submission status = %s, want %s", submission.Status, wantStatus)
	}
	if (submission.CompletedAt != nil) != wantCompletion {
		t.Fatalf("submission completion = %v, want present=%t", submission.CompletedAt, wantCompletion)
	}
}

func TestCreateDatasetForModelDoesNotRepurposeSameNameDataset(t *testing.T) {
	db := newModelDownloadDatasetTestDB(t)

	manual := model.Dataset{
		Name: "owner/resource", URL: "homes/user/private-resource", Type: model.DataTypeModel,
		Describe: "user-created dataset", UserID: 9,
		Extra: datatypes.NewJSONType(model.ExtraContent{Editable: true}),
	}
	if err := db.Create(&manual).Error; err != nil {
		t.Fatal(err)
	}

	download := &model.ModelDownload{
		Name: "owner/resource", Source: model.ModelSourceModelScope,
		Category: model.DownloadCategoryModel, Revision: "master",
		Path: "storage/Models/owner/resource", SizeBytes: 42, CreatorID: 1,
	}
	reconciler := &ModelDownloadReconciler{}
	reconcileDownloadedDataset(t, db, reconciler, download)
	reconcileDownloadedDataset(t, db, reconciler, download)

	assertDatasetCount(t, db, 2)
	assertManualDatasetUnchanged(t, db, &manual)
	assertDownloadedDataset(t, db, download, manual.ID)
}

func newModelDownloadDatasetTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:model_download_dataset_identity?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Dataset{}, &model.UserDataset{}, &model.AccountDataset{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func reconcileDownloadedDataset(
	t *testing.T, db *gorm.DB, reconciler *ModelDownloadReconciler, download *model.ModelDownload,
) {
	t.Helper()

	if err := reconciler.createDatasetForModelTx(
		context.Background(), query.Use(db), download, "downloaded model", []string{"llm"},
	); err != nil {
		t.Fatal(err)
	}
}

func assertDatasetCount(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()

	var datasetCount int64
	if err := db.Model(&model.Dataset{}).Count(&datasetCount).Error; err != nil {
		t.Fatal(err)
	}
	if datasetCount != want {
		t.Fatalf("dataset count = %d, want one manual and one downloaded Dataset", datasetCount)
	}
}

func assertManualDatasetUnchanged(t *testing.T, db *gorm.DB, manual *model.Dataset) {
	t.Helper()

	var unchanged model.Dataset
	if err := db.First(&unchanged, manual.ID).Error; err != nil {
		t.Fatal(err)
	}
	if unchanged.URL != manual.URL || unchanged.Describe != manual.Describe || unchanged.Type != manual.Type {
		t.Fatalf("same-name user Dataset was modified: %#v", unchanged)
	}
	var manualPublicLinks int64
	if err := db.Model(&model.AccountDataset{}).Where("dataset_id = ?", manual.ID).Count(&manualPublicLinks).Error; err != nil {
		t.Fatal(err)
	}
	if manualPublicLinks != 0 {
		t.Fatalf("same-name user Dataset received %d public associations", manualPublicLinks)
	}
}

func assertDownloadedDataset(t *testing.T, db *gorm.DB, download *model.ModelDownload, manualDatasetID uint) {
	t.Helper()

	var downloaded model.Dataset
	if err := db.Where("url = ? AND type = ?", download.Path, model.DataTypeModel).First(&downloaded).Error; err != nil {
		t.Fatal(err)
	}
	if downloaded.ID == manualDatasetID || downloaded.Name != download.Name || downloaded.SizeBytes != download.SizeBytes {
		t.Fatalf("unexpected downloaded Dataset: %#v", downloaded)
	}
	var publicLink model.AccountDataset
	if err := db.Where("dataset_id = ? AND account_id = ?", downloaded.ID, model.DefaultAccountID).
		First(&publicLink).Error; err != nil {
		t.Fatalf("downloaded Dataset is not public: %v", err)
	}
}
