package vcjob

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

func TestResolveDeleteSettlementTime(t *testing.T) {
	t.Parallel()

	recordCompletedAt := time.Date(2026, 4, 15, 9, 30, 0, 0, time.UTC)
	jobTransitionAt := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		record *model.Job
		job    *batch.Job
		check  func(time.Time) bool
	}{
		{
			name:   "prefers record completed timestamp",
			record: &model.Job{CompletedTimestamp: recordCompletedAt},
			job: &batch.Job{Status: batch.JobStatus{
				State: batch.JobState{LastTransitionTime: metav1.NewTime(jobTransitionAt)},
			}},
			check: func(got time.Time) bool {
				return got.Equal(recordCompletedAt)
			},
		},
		{
			name:   "falls back to job transition timestamp",
			record: &model.Job{},
			job: &batch.Job{Status: batch.JobStatus{
				State: batch.JobState{LastTransitionTime: metav1.NewTime(jobTransitionAt)},
			}},
			check: func(got time.Time) bool {
				return got.Equal(jobTransitionAt)
			},
		},
		{
			name: "falls back to current time when timestamps missing",
			check: func(got time.Time) bool {
				return !got.IsZero()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolveDeleteSettlementTime(tc.record, tc.job)
			if !tc.check(got) {
				t.Fatalf("resolveDeleteSettlementTime() = %v", got)
			}
		})
	}
}

func TestSoftDeletePersistsCompletionWithoutOverwritingRecordedTime(t *testing.T) {
	db := setupRegularJobListDB(t)
	originalQuery := query.Q
	query.Q = query.Use(db)
	t.Cleanup(func() { query.Q = originalQuery })
	transition := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	clusterJob := &batch.Job{Status: batch.JobStatus{
		State: batch.JobState{LastTransitionTime: metav1.NewTime(transition)},
	}}
	mgr := &VolcanojobMgr{}
	for _, tc := range []struct {
		name string
		end  time.Time
		want time.Time
	}{
		{"missing-completion", time.Time{}, transition},
		{"recorded-completion", transition.Add(-time.Hour), transition.Add(-time.Hour)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			job := model.Job{JobName: tc.name, JobType: model.JobTypeCustom, CompletedTimestamp: tc.end}
			if err := db.Create(&job).Error; err != nil {
				t.Fatal(err)
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodDelete, "/", http.NoBody)
			if err := mgr.applyDeleteJobPlan(c, &job, &deleteJobPlan{shouldDeleteRecord: true, clusterJob: clusterJob}); err != nil {
				t.Fatal(err)
			}
			var deleted model.Job
			if err := db.Unscoped().First(&deleted, job.ID).Error; err != nil {
				t.Fatal(err)
			}
			if !deleted.DeletedAt.Valid || !deleted.CompletedTimestamp.Equal(tc.want) {
				t.Fatalf("deleted=%v, completed=%v, want=%v", deleted.DeletedAt.Valid, deleted.CompletedTimestamp, tc.want)
			}
		})
	}
}
