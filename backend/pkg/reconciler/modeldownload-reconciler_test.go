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
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/go-logr/logr"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const existingMetadataDisplayName = "Existing name"

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

func TestShouldSyncReadyArtifacts(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		download   model.ModelDownload
		jobStatus  model.ModelDownloadStatus
		wantToSync bool
	}{
		{
			name: "ready transition", download: model.ModelDownload{Status: model.ModelDownloadStatusDownloading},
			jobStatus: model.ModelDownloadStatusReady, wantToSync: true,
		},
		{
			name: "recover missing readme", download: model.ModelDownload{Status: model.ModelDownloadStatusReady},
			jobStatus: model.ModelDownloadStatusReady, wantToSync: true,
		},
		{
			name: "already persisted", download: model.ModelDownload{
				Status: model.ModelDownloadStatusReady, SourceReadme: "# Model Card",
			},
			jobStatus: model.ModelDownloadStatusReady, wantToSync: false,
		},
		{
			name: "failed job", download: model.ModelDownload{Status: model.ModelDownloadStatusDownloading},
			jobStatus: model.ModelDownloadStatusFailed, wantToSync: false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := shouldSyncReadyArtifacts(&testCase.download, testCase.jobStatus); got != testCase.wantToSync {
				t.Fatalf("shouldSyncReadyArtifacts() = %v, want %v", got, testCase.wantToSync)
			}
		})
	}
}

func TestParseRepositoryMetadata(t *testing.T) {
	logs := `noise
[META] {"downloads":4235273,"likes":333,"updated_at":"2025-07-26T16:12:41Z","tags":["text-generation"]}
` + capturedReadmeLogs(t, "---\nlicense: apache-2.0\n---\n# Model Card\n<script>unsafe()</script>\nUseful details.", 11)
	metadata, err := parseRepositoryMetadata(logs)
	if err != nil {
		t.Fatal(err)
	}

	if metadata.Downloads != 4235273 || metadata.Likes != 333 || metadata.UpdatedAt != "2025-07-26T16:12:41Z" {
		t.Fatalf("unexpected repository metadata: %#v", metadata)
	}
	if strings.Join(metadata.Tags, ",") != "text-generation" {
		t.Fatalf("unexpected repository tags: %#v", metadata.Tags)
	}
	if metadata.Readme != "# Model Card\n\nUseful details." {
		t.Fatalf("unexpected cleaned README: %q", metadata.Readme)
	}
}

func TestParseRepositoryMetadataRejectsMissingPayload(t *testing.T) {
	if _, err := parseRepositoryMetadata("[RESULT] size_bytes=42\n"); !errors.Is(err, errRepositoryPayloadMissing) {
		t.Fatalf("parseRepositoryMetadata() error = %v, want %v", err, errRepositoryPayloadMissing)
	}
}

func TestReadyArtifactSyncRetriesWhenFinalLogsAreUnavailable(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	reconciler := &ModelDownloadReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
	}
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "download-owner-model", Namespace: "jobs"},
		Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{
			Type: batchv1.JobComplete, Status: v1.ConditionTrue,
		}}},
	}
	download := &model.ModelDownload{Status: model.ModelDownloadStatusDownloading}

	result, err := reconciler.syncDownloadWithJob(context.Background(), job, download, logr.Discard())
	if !errors.Is(err, errFinalLogsUnavailable) {
		t.Fatalf("syncDownloadWithJob() error = %v, want %v", err, errFinalLogsUnavailable)
	}
	if result.RequeueAfter != progressRequeueInterval {
		t.Fatalf("syncDownloadWithJob() RequeueAfter = %v, want %v", result.RequeueAfter, progressRequeueInterval)
	}
	if download.Status != model.ModelDownloadStatusDownloading {
		t.Fatalf("download status advanced without final logs: %s", download.Status)
	}
}

func TestReadyArtifactSyncRetriesWhenMetadataPersistenceFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:metadata_persistence_failure?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ModelDownload{}); err != nil {
		t.Fatal(err)
	}
	download := model.ModelDownload{
		Name: "owner/model", Source: model.ModelSourceModelScope, Category: model.DownloadCategoryModel,
		Revision: "master", Path: "public/Models/owner/model", CreatorID: 1,
		Status: model.ModelDownloadStatusDownloading, DisplayName: existingMetadataDisplayName,
	}
	if err := db.Create(&download).Error; err != nil {
		t.Fatal(err)
	}

	scheme := runtime.NewScheme()
	if err := v1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: "download-owner-model-pod", Namespace: "jobs", Labels: map[string]string{"job-name": "download-owner-model"},
	}}
	reconciler := &ModelDownloadReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build(),
		podLogs: func(context.Context, *v1.Pod) (string, error) {
			return "[META] {\"display_name\":\"Fresh name\"}\n", nil
		},
		db: db,
	}
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "download-owner-model", Namespace: "jobs"},
		Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{
			Type: batchv1.JobComplete, Status: v1.ConditionTrue,
		}}},
	}

	result, err := reconciler.syncDownloadWithJob(context.Background(), job, &download, logr.Discard())
	if err == nil || !strings.Contains(err.Error(), "persist repository metadata") {
		t.Fatalf("syncDownloadWithJob() error = %v, want metadata persistence failure", err)
	}
	if result.RequeueAfter != progressRequeueInterval {
		t.Fatalf("syncDownloadWithJob() RequeueAfter = %v, want %v", result.RequeueAfter, progressRequeueInterval)
	}
	var stored model.ModelDownload
	if err := db.First(&stored, download.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.ModelDownloadStatusDownloading || stored.DisplayName != existingMetadataDisplayName {
		t.Fatalf("failed finalization advanced or overwrote the download: %#v", stored)
	}
}

func TestRepositoryReadmePatchPreservesExistingMetadata(t *testing.T) {
	db := newRepositoryMetadataTestDB(t)
	source := model.ModelDatasetSource{
		Provider: model.ModelDatasetProviderModelScope, ResourceType: model.DataTypeModel,
		RepositoryID: "owner/model", DisplayName: existingMetadataDisplayName, Description: "Existing description",
		License: "apache-2.0", Downloads: 99, Readme: "Old README",
	}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	download := model.ModelDownload{
		Name: "owner/model", Source: model.ModelSourceModelScope, Category: model.DownloadCategoryModel,
		Revision: "master", Path: "public/Models/owner/model", CreatorID: 1,
		DisplayName: existingMetadataDisplayName, SourceDescription: "Existing description", License: "apache-2.0",
		SourceDownloads: 99, SourceReadme: "Old README", ModelDatasetSourceID: &source.ID,
	}
	if err := db.Create(&download).Error; err != nil {
		t.Fatal(err)
	}
	metadata, err := parseRepositoryMetadata(capturedReadmeLogs(t, "# New model card", 8))
	if err != nil {
		t.Fatal(err)
	}
	if err := persistRepositoryMetadataWithDB(
		context.Background(), db, &download, &metadata, "", nil, "",
	); err != nil {
		t.Fatal(err)
	}

	assertDownloadMetadataPatch(t, db, download.ID)
	assertSourceMetadataPatch(t, db, source.ID)
}

func assertDownloadMetadataPatch(t *testing.T, db *gorm.DB, downloadID uint) {
	t.Helper()

	var stored model.ModelDownload
	if err := db.First(&stored, downloadID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.DisplayName != existingMetadataDisplayName || stored.SourceDescription != "Existing description" ||
		stored.License != "apache-2.0" || stored.SourceDownloads != 99 {
		t.Fatalf("README patch cleared download metadata: %#v", stored)
	}
	if stored.SourceReadme != "# New model card" {
		t.Fatalf("download README = %q, want updated model card", stored.SourceReadme)
	}
}

func assertSourceMetadataPatch(t *testing.T, db *gorm.DB, sourceID uint) {
	t.Helper()

	var stored model.ModelDatasetSource
	if err := db.First(&stored, sourceID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.DisplayName != existingMetadataDisplayName || stored.Description != "Existing description" ||
		stored.License != "apache-2.0" || stored.Downloads != 99 {
		t.Fatalf("README patch cleared source metadata: %#v", stored)
	}
	if stored.Readme != "# New model card" {
		t.Fatalf("source README = %q, want updated model card", stored.Readme)
	}
}

func newRepositoryMetadataTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:repository_metadata_patch?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ModelDatasetSource{}, &model.ModelDownload{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCapturedReadmeIsBoundedAtUTF8Boundary(t *testing.T) {
	oversized := strings.Repeat("x", maxStoredReadmeBytes+4096)
	if got := parseReadmeFromLogs(capturedReadmeLogs(t, oversized, 128)); len(got) != maxStoredReadmeBytes {
		t.Fatalf("oversized README length = %d, want %d", len(got), maxStoredReadmeBytes)
	}

	prefix := strings.Repeat("a", maxStoredReadmeBytes-1)
	got := parseReadmeFromLogs(capturedReadmeLogs(t, prefix+"界tail", 128))
	if got != prefix {
		t.Fatalf("multibyte boundary kept partial data: length = %d", len(got))
	}
	if !utf8.ValidString(got) {
		t.Fatal("truncated README is not valid UTF-8")
	}
}

func TestCapturedReadmeIsRemovedFromStoredLogs(t *testing.T) {
	logs := "before\n" + capturedReadmeLogs(t, "# Model Card", 4) + "after\n"
	stored := logsWithoutCapturedReadme(logs)

	if strings.Contains(stored, readmeCaptureChunk) || strings.Contains(stored, readmeCaptureBegin) ||
		strings.Contains(stored, readmeCaptureEnd) {
		t.Fatalf("stored logs expose encoded README: %q", stored)
	}
	for _, expected := range []string{"before", readmeCapturedLogNotice, "after"} {
		if !strings.Contains(stored, expected) {
			t.Fatalf("stored logs do not contain %q: %q", expected, stored)
		}
	}
}

func capturedReadmeLogs(t *testing.T, readme string, chunkSize int) string {
	t.Helper()

	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write([]byte(readme)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	payload := base64.StdEncoding.EncodeToString(compressed.Bytes())

	var logs strings.Builder
	logs.WriteString(readmeCaptureBegin + "\n")
	for offset := 0; offset < len(payload); offset += chunkSize {
		end := min(offset+chunkSize, len(payload))
		logs.WriteString(readmeCaptureChunk + payload[offset:end] + "\n")
	}
	logs.WriteString(readmeCaptureEnd + "\n")
	return logs.String()
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
