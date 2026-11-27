package vcjob

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
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
