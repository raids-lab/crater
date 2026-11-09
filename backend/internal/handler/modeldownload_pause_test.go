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

package handler

import (
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

const testOldDownloadJobName = "old-job"

func newPauseDownloadTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "pause.db")), &gorm.Config{
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
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.AutoMigrate(&model.ModelDownload{}, &model.ModelDownloadSubmission{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPauseRejectsChangedDownload(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status model.ModelDownloadStatus
		job    string
	}{
		{name: "pending", status: model.ModelDownloadStatusPending, job: testOldDownloadJobName},
		{name: "paused", status: model.ModelDownloadStatusPaused, job: testOldDownloadJobName},
		{name: "ready", status: model.ModelDownloadStatusReady, job: testOldDownloadJobName},
		{name: "failed", status: model.ModelDownloadStatusFailed, job: testOldDownloadJobName},
		{name: "resumed attempt", status: model.ModelDownloadStatusDownloading, job: "new-job"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newPauseDownloadTestDB(t)
			download := model.ModelDownload{
				Name: "owner/model", Source: model.ModelSourceModelScope, Category: model.DownloadCategoryModel,
				Revision: "main", Status: tc.status, JobName: tc.job, Message: "current attempt",
			}
			if err := db.Create(&download).Error; err != nil {
				t.Fatal(err)
			}
			submission := model.ModelDownloadSubmission{
				UserID: 7, ModelDownloadID: download.ID, Action: model.ModelDownloadSubmissionResume,
				Status: model.ModelDownloadSubmissionReserved,
			}
			if err := db.Create(&submission).Error; err != nil {
				t.Fatal(err)
			}
			stale := download
			stale.Status, stale.JobName = model.ModelDownloadStatusDownloading, testOldDownloadJobName
			if err := pauseDownloadAndReleaseQuota(t.Context(), db, &stale); !errors.Is(err, bizerr.Conflict.Base) {
				t.Fatalf("pause error = %v, want state conflict", err)
			}
			var stored model.ModelDownload
			if err := db.First(&stored, download.ID).Error; err != nil {
				t.Fatal(err)
			}
			if stored.Status != tc.status || stored.JobName != tc.job || stored.Message != "current attempt" {
				t.Fatalf("rejected pause changed current attempt: %#v", stored)
			}
			if err := db.First(&submission, submission.ID).Error; err != nil {
				t.Fatal(err)
			}
			if submission.Status != model.ModelDownloadSubmissionReserved {
				t.Fatalf("rejected pause changed reservation to %s", submission.Status)
			}
		})
	}
}

func TestPauseRollsBackWhenQuotaReleaseFails(t *testing.T) {
	db := newPauseDownloadTestDB(t)
	download := model.ModelDownload{
		Name: "owner/model", Source: model.ModelSourceModelScope, Category: model.DownloadCategoryModel,
		Revision: "main", Status: model.ModelDownloadStatusDownloading, JobName: "current-job",
	}
	if err := db.Create(&download).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropTable(&model.ModelDownloadSubmission{}); err != nil {
		t.Fatal(err)
	}
	if err := pauseDownloadAndReleaseQuota(t.Context(), db, &download); err == nil {
		t.Fatal("pause succeeded without the quota reservation table")
	}
	var stored model.ModelDownload
	if err := db.First(&stored, download.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.ModelDownloadStatusDownloading || stored.Message != "" {
		t.Fatalf("failed pause was not rolled back: %#v", stored)
	}
}

func TestFinishedSubmissionPreservesNewerState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status model.ModelDownloadStatus
		job    string
		want   model.ModelDownloadStatus
	}{
		{name: "pending submission", status: model.ModelDownloadStatusPending, job: testOldDownloadJobName, want: model.ModelDownloadStatusDownloading},
		{name: "concurrent pause", status: model.ModelDownloadStatusPaused, job: testOldDownloadJobName, want: model.ModelDownloadStatusPaused},
		{name: "completed job", status: model.ModelDownloadStatusReady, job: testOldDownloadJobName, want: model.ModelDownloadStatusReady},
		{name: "failed job", status: model.ModelDownloadStatusFailed, job: testOldDownloadJobName, want: model.ModelDownloadStatusFailed},
		{name: "new attempt", status: model.ModelDownloadStatusDownloading, job: "new-job", want: model.ModelDownloadStatusDownloading},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newPauseDownloadTestDB(t)
			download := model.ModelDownload{
				Name: "owner/model", Source: model.ModelSourceModelScope, Category: model.DownloadCategoryModel,
				Revision: "main", Status: tc.status, JobName: tc.job,
			}
			if err := db.Create(&download).Error; err != nil {
				t.Fatal(err)
			}
			stale := download
			stale.Status, stale.JobName = model.ModelDownloadStatusPending, testOldDownloadJobName
			current, err := finishDownloadSubmission(t.Context(), db, &stale)
			if err != nil {
				t.Fatal(err)
			}
			if current.Status != tc.want || current.JobName != tc.job {
				t.Fatalf("submission response = %#v, want %s/%s", current, tc.want, tc.job)
			}
		})
	}
}

func TestFailedSubmissionCannotRollbackResumedDownload(t *testing.T) {
	db := newPauseDownloadTestDB(t)
	download := model.ModelDownload{
		Name: "owner/model", Source: model.ModelSourceModelScope, Category: model.DownloadCategoryModel,
		Revision: "main", Status: model.ModelDownloadStatusDownloading, JobName: "new-job",
	}
	if err := db.Create(&download).Error; err != nil {
		t.Fatal(err)
	}
	submission := model.ModelDownloadSubmission{
		UserID: 7, ModelDownloadID: download.ID, Action: model.ModelDownloadSubmissionResume,
		Status: model.ModelDownloadSubmissionReserved,
	}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatal(err)
	}
	stale := download
	stale.JobName = testOldDownloadJobName
	if err := updateDownloadAndReleaseQuota(t.Context(), db, &stale, map[string]any{
		"status": model.ModelDownloadStatusFailed,
	}); err != nil {
		t.Fatal(err)
	}
	var stored model.ModelDownload
	if err := db.First(&stored, download.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.ModelDownloadStatusDownloading || stored.JobName != "new-job" {
		t.Fatalf("stale submission failure changed resumed attempt: %#v", stored)
	}
	if err := db.First(&submission, submission.ID).Error; err != nil {
		t.Fatal(err)
	}
	if submission.Status != model.ModelDownloadSubmissionReserved {
		t.Fatalf("stale submission failure released current reservation: %#v", submission)
	}
}

func TestResumeWaitsForPreviousDownloaderToStop(t *testing.T) {
	for _, tc := range []struct {
		name         string
		jobPresent   bool
		deleting     bool
		podPhase     corev1.PodPhase
		wantConflict bool
	}{
		{name: "existing_job", jobPresent: true, wantConflict: true},
		{name: "deleting_job", jobPresent: true, deleting: true, wantConflict: true},
		{name: "pending_pod", podPhase: corev1.PodPending, wantConflict: true},
		{name: "running_pod", podPhase: corev1.PodRunning, wantConflict: true},
		{name: "terminating_pod", podPhase: corev1.PodRunning, deleting: true, wantConflict: true},
		{name: "succeeded_pod", podPhase: corev1.PodSucceeded},
		{name: "failed_pod", podPhase: corev1.PodFailed},
		{name: "fully_deleted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metadata := metav1.ObjectMeta{Name: testOldDownloadJobName, Namespace: "jobs"}
			if tc.deleting {
				now := metav1.Now()
				metadata.DeletionTimestamp = &now
			}
			var objects []runtime.Object
			if tc.jobPresent {
				objects = append(objects, &batchv1.Job{ObjectMeta: metadata})
			}
			if tc.podPhase != "" {
				metadata.Labels = map[string]string{"job-name": testOldDownloadJobName}
				objects = append(objects, &corev1.Pod{ObjectMeta: metadata, Status: corev1.PodStatus{Phase: tc.podPhase}})
			}
			mgr := ModelDownloadMgr{crClient: fake.NewClientset(objects...), namespace: "jobs"}
			err := mgr.requirePreviousDownloadStopped(t.Context(), &model.ModelDownload{
				Status: model.ModelDownloadStatusPaused, JobName: testOldDownloadJobName,
			})
			if tc.wantConflict {
				if !errors.Is(err, bizerr.Conflict.Base) {
					t.Fatalf("resume error = %v, want a stopping conflict", err)
				}
			} else if err != nil {
				t.Fatalf("stopped downloader blocked resume: %v", err)
			}
		})
	}
}

func TestResumeRejectsUnverifiablePreviousJobState(t *testing.T) {
	for _, resource := range []string{"jobs", "pods"} {
		t.Run(resource, func(t *testing.T) {
			apiErr := k8serrors.NewServiceUnavailable("Kubernetes API unavailable")
			kube := fake.NewClientset()
			kube.PrependReactor("*", resource, func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, apiErr
			})
			mgr := ModelDownloadMgr{crClient: kube, namespace: "jobs"}
			if err := mgr.requirePreviousDownloadStopped(t.Context(), &model.ModelDownload{
				Status: model.ModelDownloadStatusPaused, JobName: testOldDownloadJobName,
			}); !errors.Is(err, apiErr) {
				t.Fatalf("resume error = %v, want the Kubernetes API error", err)
			}
		})
	}
}

func TestPrepareResumePreservesPausedAttemptWhilePreviousJobStops(t *testing.T) {
	db, mgr := newResumePreparationTestManager(t, "resume_waits_for_stopped_job")
	paused := createResumePreparationTestDownload(
		t, db, model.ModelSourceModelScope, "master", model.ModelDownloadStatusPaused, testOldDownloadJobName,
	)
	kube := fake.NewClientset(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: paused.JobName, Namespace: "jobs"}})
	mgr.crClient = kube
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	token := util.JWTMessage{UserID: paused.CreatorID, Username: "alice"}
	if _, err := mgr.prepareResumeDownload(ctx, query.Use(db), paused.ID, token); !errors.Is(err, bizerr.Conflict.Base) {
		t.Fatalf("resume error = %v, want a stopping conflict", err)
	}
	var stored model.ModelDownload
	if err := db.First(&stored, paused.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.ModelDownloadStatusPaused || stored.JobName != paused.JobName {
		t.Fatalf("blocked resume changed the paused attempt: %#v", stored)
	}
	var reservations int64
	if err := db.Model(&model.ModelDownloadSubmission{}).Count(&reservations).Error; err != nil {
		t.Fatal(err)
	}
	if reservations != 0 {
		t.Fatalf("blocked resume reserved quota %d times", reservations)
	}
	if err := kube.BatchV1().Jobs("jobs").Delete(t.Context(), paused.JobName, metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	current, err := mgr.prepareResumeDownload(ctx, query.Use(db), paused.ID, token)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != model.ModelDownloadStatusPending || current.JobName == paused.JobName {
		t.Fatalf("resume did not prepare a pending attempt: %#v", current)
	}
	if _, err := mgr.prepareResumeDownload(ctx, query.Use(db), paused.ID, token); !errors.Is(err, bizerr.Conflict.Base) {
		t.Fatalf("in-flight submission accepted a second resume: %v", err)
	}
}

func TestRecreateSoftDeletedDownload(t *testing.T) {
	for _, tc := range []struct {
		status  model.ModelDownloadStatus
		objects []runtime.Object
	}{
		{status: model.ModelDownloadStatusPaused},
		{status: model.ModelDownloadStatusFailed, objects: []runtime.Object{finishedDownloadJob()}},
	} {
		t.Run(string(tc.status), func(t *testing.T) {
			db, mgr, history := newSoftDeletedDownloadTest(t, tc.status, tc.objects...)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			token := util.JWTMessage{UserID: history.CreatorID, Username: "alice"}
			download, submit, err := mgr.getOrCreateDownload(
				ctx, query.Use(db), &CreateDownloadReq{Name: history.Name}, token,
				history.Source, history.Category, history.Path, history.Revision,
			)
			if err != nil {
				t.Fatal(err)
			}
			if !submit || download.ID != history.ID || download.Status != model.ModelDownloadStatusPending ||
				download.JobName == history.JobName || download.JobName == "" || download.Message != "" {
				t.Fatalf("recreated attempt = id %d status %s job %q message %q, submit %v",
					download.ID, download.Status, download.JobName, download.Message, submit)
			}
			var reservations []model.ModelDownloadSubmission
			if err := db.Where("model_download_id = ?", history.ID).Find(&reservations).Error; err != nil {
				t.Fatal(err)
			}
			if len(reservations) != 1 || reservations[0].Action != model.ModelDownloadSubmissionCreate ||
				reservations[0].Status != model.ModelDownloadSubmissionReserved {
				t.Fatalf("recreated attempt reservations = %+v, want one reserved create submission", reservations)
			}
			finished, err := finishDownloadSubmission(ctx, db, download)
			if err != nil {
				t.Fatal(err)
			}
			if finished.Status != model.ModelDownloadStatusDownloading {
				t.Fatalf("submitted record status = %s, want Downloading", finished.Status)
			}
		})
	}
}

func TestRecreateSoftDeletedDownloadWaitsForPreviousPod(t *testing.T) {
	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "jobs", Name: "old-pod",
			Labels: map[string]string{"job-name": testOldDownloadJobName}, DeletionTimestamp: &now,
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	for _, tc := range []struct {
		status  model.ModelDownloadStatus
		objects []runtime.Object
	}{
		{status: model.ModelDownloadStatusPaused, objects: []runtime.Object{pod}},
		{status: model.ModelDownloadStatusFailed, objects: []runtime.Object{finishedDownloadJob(), pod}},
	} {
		t.Run(string(tc.status), func(t *testing.T) {
			db, mgr, history := newSoftDeletedDownloadTest(t, tc.status, tc.objects...)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			token := util.JWTMessage{UserID: history.CreatorID, Username: "alice"}
			_, submit, err := mgr.getOrCreateDownload(
				ctx, query.Use(db), &CreateDownloadReq{Name: history.Name}, token,
				history.Source, history.Category, history.Path, history.Revision,
			)
			if !errors.Is(err, bizerr.Conflict.Base) || submit {
				t.Fatalf("recreate while a Pod is stopping: submit=%v err=%v, want conflict", submit, err)
			}
			var stored model.ModelDownload
			if err := db.Unscoped().First(&stored, history.ID).Error; err != nil {
				t.Fatal(err)
			}
			if !stored.DeletedAt.Valid || stored.Status != history.Status || stored.JobName != history.JobName {
				t.Fatalf("blocked recreate changed history: deleted=%v status=%s job=%q",
					stored.DeletedAt.Valid, stored.Status, stored.JobName)
			}
			var reservations int64
			if err := db.Model(&model.ModelDownloadSubmission{}).Count(&reservations).Error; err != nil {
				t.Fatal(err)
			}
			if reservations != 0 {
				t.Fatalf("blocked recreate reserved quota %d times", reservations)
			}
		})
	}
}

func finishedDownloadJob() *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Namespace: "jobs", Name: testOldDownloadJobName},
		Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{
			{Type: batchv1.JobFailed, Status: corev1.ConditionTrue},
		}},
	}
}

func newSoftDeletedDownloadTest(
	t *testing.T, status model.ModelDownloadStatus, objects ...runtime.Object,
) (*gorm.DB, *ModelDownloadMgr, model.ModelDownload) {
	t.Helper()
	db := newPauseDownloadTestDB(t)
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.UserModelDownload{}); err != nil {
		t.Fatal(err)
	}
	mgr := &ModelDownloadMgr{
		crClient: fake.NewClientset(objects...), namespace: "jobs",
		quotaService: service.NewModelDownloadQuotaService(service.NewConfigService(query.Use(db))),
	}
	history := createResumePreparationTestDownload(
		t, db, model.ModelSourceModelScope, "master", status, testOldDownloadJobName,
	)
	if err := db.Model(&history).Update("message", "previous download attempt").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&history).Error; err != nil {
		t.Fatal(err)
	}
	return db, mgr, history
}
