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
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/raids-lab/crater/dao/model"
)

func TestStaleDownloadWritesPreserveCurrentAttempt(t *testing.T) {
	for _, state := range []string{"paused", "resumed"} {
		for _, operation := range []string{"status", "progress", "result", "logs", "metadata", "dataset"} {
			t.Run(state+"/"+operation, func(t *testing.T) {
				reconciler, job, current, stale := newStaleDownloadTestFixture(t, state)
				err := runStaleDownloadWrite(t, reconciler, job, &stale, operation)
				if !errors.Is(err, errDownloadStateChanged) {
					t.Fatalf("stale %s error = %v, want state conflict", operation, err)
				}
				assertStaleDownloadWriteRejected(t, reconciler.database(), &current)
			})
		}
	}
}

func newStaleDownloadTestFixture(t *testing.T, state string) (
	reconciler *ModelDownloadReconciler, job *batchv1.Job, current, stale model.ModelDownload,
) {
	t.Helper()
	db := newMissingJobTransitionTestDB(t)
	if err := db.AutoMigrate(&model.ModelDatasetSource{}, &model.Dataset{}); err != nil {
		t.Fatal(err)
	}
	download := model.ModelDownload{
		Name: "owner/model", Source: model.ModelSourceModelScope, Category: model.DownloadCategoryModel,
		Revision: "main", Path: "public/Models/owner/model", CreatorID: 7,
		Status: model.ModelDownloadStatusDownloading, JobName: "old-job",
		Message: "current attempt", Logs: "current logs", SourceReadme: "current README",
		DownloadedBytes: 11, SizeBytes: 100,
	}
	if err := db.Create(&download).Error; err != nil {
		t.Fatal(err)
	}
	stale = download
	if state == "paused" {
		download.Status = model.ModelDownloadStatusPaused
	} else {
		download.JobName = "new-job"
	}
	if err := db.Model(&download).Updates(map[string]any{
		"status": download.Status, "job_name": download.JobName,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ModelDownloadSubmission{
		UserID: 7, ModelDownloadID: download.ID, Action: model.ModelDownloadSubmissionResume,
		Status: model.ModelDownloadSubmissionReserved,
	}).Error; err != nil {
		t.Fatal(err)
	}

	scheme := runtime.NewScheme()
	if err := v1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "old-pod", Namespace: "jobs", Labels: map[string]string{"job-name": "old-job"}},
		Status:     v1.PodStatus{Phase: v1.PodRunning},
	}
	reconciler = &ModelDownloadReconciler{
		db: db, Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build(),
		podLogs: func(context.Context, *v1.Pod) (string, error) {
			return "[PROGRESS] downloaded_bytes=99\n[RESULT] size_bytes=99\n", nil
		},
	}
	job = &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "old-job", Namespace: "jobs"}}
	return reconciler, job, download, stale
}

func runStaleDownloadWrite(t *testing.T, reconciler *ModelDownloadReconciler, job *batchv1.Job, stale *model.ModelDownload, operation string) error {
	t.Helper()
	db := reconciler.database()
	var err error
	switch operation {
	case "status":
		err = reconciler.updateDownloadStatus(t.Context(), stale, model.ModelDownloadStatusReady)
	case "progress":
		err = reconciler.updateProgress(t.Context(), job, stale)
	case "result":
		err = reconciler.extractFinalResult(t.Context(), job, stale)
	case "logs":
		_, err = reconciler.persistFinalLogs(t.Context(), job, stale)
	case "metadata":
		metadata, parseErr := parseRepositoryMetadata(capturedReadmeLogs(t, "stale README", 8))
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		err = persistRepositoryMetadataWithDB(t.Context(), db, stale, &metadata, "", nil, "")
	case "dataset":
		err = reconciler.createDatasetForModel(t.Context(), stale, "stale description", nil)
	}
	return err
}

func assertStaleDownloadWriteRejected(t *testing.T, db *gorm.DB, download *model.ModelDownload) {
	t.Helper()
	var stored model.ModelDownload
	if err := db.First(&stored, download.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != download.Status || stored.JobName != download.JobName ||
		stored.Message != download.Message || stored.Logs != download.Logs ||
		stored.SourceReadme != download.SourceReadme || stored.DownloadedBytes != 11 || stored.SizeBytes != 100 {
		t.Fatalf("stale write overwrote current attempt: %#v", stored)
	}
	assertQuotaSubmissionSettlement(t, db, download.ID, model.ModelDownloadSubmissionReserved, false)
	for _, table := range []any{&model.Dataset{}, &model.ModelDatasetSource{}} {
		var count int64
		if err := db.Model(table).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("stale write published %d %T rows", count, table)
		}
	}
}

func TestStaleCompletionRequeuesWithoutPublishing(t *testing.T) {
	db := newMissingJobTransitionTestDB(t)
	download := model.ModelDownload{
		Name: "owner/model", Source: model.ModelSourceModelScope, Category: model.DownloadCategoryModel,
		Revision: "main", Status: model.ModelDownloadStatusPaused, JobName: "old-job", Message: "Download paused by user",
	}
	if err := db.Create(&download).Error; err != nil {
		t.Fatal(err)
	}
	stale := download
	stale.Status = model.ModelDownloadStatusDownloading
	scheme := runtime.NewScheme()
	if err := v1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: "old-pod", Namespace: "jobs", Labels: map[string]string{"job-name": "old-job"},
	}}
	reconciler := &ModelDownloadReconciler{
		db: db, Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build(),
		podLogs: func(context.Context, *v1.Pod) (string, error) { return "[RESULT] size_bytes=99\n", nil },
	}
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "old-job", Namespace: "jobs"},
		Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{
			Type: batchv1.JobComplete, Status: v1.ConditionTrue,
		}}},
	}
	result, err := reconciler.syncDownloadWithJob(t.Context(), job, &stale, logr.Discard())
	if err != nil || result.RequeueAfter != progressRequeueInterval {
		t.Fatalf("stale completion result = %#v, error = %v, want state refresh", result, err)
	}
	var stored model.ModelDownload
	if err := db.First(&stored, download.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.ModelDownloadStatusPaused || stored.Message != download.Message || stored.SizeBytes != 0 {
		t.Fatalf("stale completion overwrote paused download: %#v", stored)
	}
}
