package reconciler

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	clocktesting "k8s.io/utils/clock/testing"
	"k8s.io/utils/ptr"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/pkg/monitor"
)

func TestProfileDatabaseWriteErrorRetries(t *testing.T) {
	q := useJobProfileDB(t)
	end := time.Now().UTC()
	job := newProfileJob(batch.Completed, end)
	insertProfileJob(t, q, job, end.Add(-time.Hour))
	if _, err := q.Job.WithContext(t.Context()).Where(q.Job.JobName.Eq(job.Name)).Updates(map[string]any{
		"status": batch.Completed, "completed_timestamp": end,
	}); err != nil {
		t.Fatal(err)
	}
	capture := &profileCapture{result: &monitor.ProfileData{CPUUsageAvg: ptr.To(float32(2))}}
	r := newJobProfileReconciler(t, capture)
	fakeClock := clocktesting.NewFakeClock(end)
	r.profileQueue.clock = fakeClock
	var writes atomic.Int32
	db := q.Job.WithContext(t.Context()).UnderlyingDB()
	if err := db.Callback().Update().Before("gorm:update").Register("profile_write_error", func(tx *gorm.DB) {
		if writes.Add(1) == 1 {
			_ = tx.AddError(errors.New("database write temporarily unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	startProfileWorkers(t, r)
	if !r.profileQueue.enqueue(vcJobProfileTask{namespace: job.Namespace, jobName: job.Name}) {
		t.Fatal("write-error fixture was rejected")
	}
	waitProfileCondition(t, fakeClock.HasWaiters)
	if record := readProfileJob(t, q, job.Name); !isProfileDataMissing(record.ProfileData) || writes.Load() != 1 {
		t.Fatal("failed write stored a profile or was not attempted")
	}
	fakeClock.Step(30 * time.Second)
	waitProfileIdle(t, r.profileQueue)
	assertStoredCPU(t, readProfileJob(t, q, job.Name), 2)
	if capture.calls != 2 || writes.Load() != 2 || fakeClock.HasWaiters() {
		t.Fatalf("write retry did not finish: queries=%d, writes=%d, timer=%v", capture.calls, writes.Load(), fakeClock.HasWaiters())
	}
}

func TestProfileWritePreservesConcurrentProfile(t *testing.T) {
	for _, jsonNull := range []bool{false, true} {
		name := "SQL NULL"
		if jsonNull {
			name = "JSON null"
		}
		t.Run(name, func(t *testing.T) {
			q := useJobProfileDB(t)
			end := time.Now().UTC()
			job := newProfileJob(batch.Completed, end)
			insertProfileJob(t, q, job, end.Add(-time.Hour))
			updates := map[string]any{"completed_timestamp": end}
			if jsonNull {
				updates["profile_data"] = datatypes.JSON("null")
			}
			if _, err := q.Job.WithContext(t.Context()).Where(q.Job.JobName.Eq(job.Name)).Updates(updates); err != nil {
				t.Fatal(err)
			}
			record := readProfileJob(t, q, job.Name)
			capture := &profileCapture{result: &monitor.ProfileData{CPUUsageAvg: ptr.To(float32(2))}}
			var concurrentUpdatedAt time.Time
			capture.onQuery = func() {
				profile := datatypes.NewJSONType(&monitor.ProfileData{CPUUsageAvg: ptr.To(float32(9))})
				if _, err := q.Job.WithContext(t.Context()).Where(q.Job.ID.Eq(record.ID)).Update(q.Job.ProfileData, profile); err != nil {
					t.Fatal(err)
				}
				concurrentUpdatedAt = readProfileJob(t, q, job.Name).UpdatedAt
			}
			r := newJobProfileReconciler(t, capture)
			empty, err := r.updateMissingJobProfile(t.Context(), job.Namespace, record)
			if err != nil || empty || capture.calls != 1 {
				t.Fatalf("unexpected profile result: empty=%v, error=%v, calls=%d", empty, err, capture.calls)
			}
			after := readProfileJob(t, q, job.Name)
			assertStoredCPU(t, after, 9)
			if !after.UpdatedAt.Equal(concurrentUpdatedAt) {
				t.Fatal("skipped profile write changed updated_at")
			}
		})
	}
}
