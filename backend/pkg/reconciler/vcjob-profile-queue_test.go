package reconciler

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
	clocktesting "k8s.io/utils/clock/testing"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/monitor"
)

type asyncProfileClient struct {
	monitor.PrometheusInterface
	query func(context.Context) (*monitor.ProfileData, error)
}

func (p *asyncProfileClient) QueryProfileDataContext(ctx context.Context, _ string, _ []string, _, _ time.Time) (*monitor.ProfileData, error) {
	return p.query(ctx)
}

func startProfileWorkers(t *testing.T, r *VcJobReconciler) (cancel context.CancelFunc, stopped <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := r.profileQueue.run(ctx, r.retryJobProfile); err != nil {
			t.Error(err)
		}
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("profile workers did not stop")
		}
	})
	return cancel, done
}

func waitProfileSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(10 * time.Second):
		t.Fatal("profile worker signal timed out")
	}
}

func waitProfileCondition(t *testing.T, condition func() bool) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for !condition() {
		select {
		case <-timer.C:
			t.Fatal("profile worker condition timed out")
		default:
			runtime.Gosched()
		}
	}
}

func waitProfileIdle(t *testing.T, q *vcJobProfileQueue) {
	t.Helper()
	waitProfileCondition(t, func() bool {
		q.mu.Lock()
		defer q.mu.Unlock()
		return len(q.pending) == 0
	})
}

func readUnscopedProfileJob(t *testing.T, q *query.Query, name string) *model.Job {
	t.Helper()
	record, err := q.Job.WithContext(t.Context()).Unscoped().Where(q.Job.JobName.Eq(name)).First()
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestReconcileAndDeletionDoNotWaitForProfile(t *testing.T) {
	for _, deletion := range []bool{false, true} {
		t.Run(map[bool]string{false: "completed", true: "deleted"}[deletion], func(t *testing.T) {
			q := useJobProfileDB(t)
			end := time.Now()
			job := newProfileJob(batch.Completed, end)
			insertProfileJob(t, q, job, end.Add(-time.Hour))
			r := jobProfileReconciler(t, &profileCapture{}, job)
			if deletion {
				if err := r.Delete(t.Context(), job); err != nil {
					t.Fatal(err)
				}
			}
			started, release := make(chan struct{}), make(chan struct{})
			var calls atomic.Int32
			r.prometheusClient = &asyncProfileClient{query: func(ctx context.Context) (*monitor.ProfileData, error) {
				if calls.Add(1) == 1 {
					close(started)
				}
				select {
				case <-release:
					return &monitor.ProfileData{CPUUsageAvg: ptr.To(float32(2))}, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}}
			request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}
			returned := make(chan error, 1)
			go func() {
				_, err := r.Reconcile(t.Context(), request)
				returned <- err
			}()
			select {
			case err := <-returned:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("reconcile waited for the blocked profile query")
			}
			waitProfileSignal(t, started)
			wantStatus := batch.Completed
			if deletion {
				wantStatus = model.Freed
			}
			if record := readProfileJob(t, q, job.Name); record.Status != wantStatus || record.CompletedTimestamp.IsZero() {
				t.Fatalf("job status was not persisted before profile collection: %+v", record)
			}
			repeatProfileReconcile(t, r, request)
			if calls.Load() != 1 {
				t.Fatalf("duplicate in-flight queries: %d", calls.Load())
			}
			close(release)
			waitProfileIdle(t, r.profileQueue)
			assertStoredCPU(t, readProfileJob(t, q, job.Name), 2)
			reconcileProfileJob(t, r, request)
			if calls.Load() != 1 {
				t.Fatalf("stored profile was fetched again: %d", calls.Load())
			}
		})
	}
}

func repeatProfileReconcile(t *testing.T, r *VcJobReconciler, request ctrl.Request) {
	t.Helper()
	for range 8 {
		if _, err := r.Reconcile(t.Context(), request); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRecentEmptyProfileRetriesIncludingSoftDeletedJob(t *testing.T) {
	for _, softDeleted := range []bool{false, true} {
		t.Run(map[bool]string{false: "completed", true: "soft deleted"}[softDeleted], func(t *testing.T) {
			q := useJobProfileDB(t)
			end := time.Now().UTC()
			job := newProfileJob(batch.Completed, end)
			insertProfileJob(t, q, job, end.Add(-time.Hour))
			r := jobProfileReconciler(t, &profileCapture{}, job)
			fakeClock := clocktesting.NewFakeClock(end)
			r.profileQueue.clock = fakeClock
			if softDeleted {
				j := q.Job
				if _, err := j.WithContext(t.Context()).Where(j.JobName.Eq(job.Name)).Updates(map[string]any{
					"status": batch.Completed, "completed_timestamp": end,
				}); err != nil {
					t.Fatal(err)
				}
				if _, err := j.WithContext(t.Context()).Where(j.JobName.Eq(job.Name)).Delete(); err != nil {
					t.Fatal(err)
				}
				if err := r.Delete(t.Context(), job); err != nil {
					t.Fatal(err)
				}
			}
			var calls atomic.Int32
			r.prometheusClient = &asyncProfileClient{query: func(context.Context) (*monitor.ProfileData, error) {
				if calls.Add(1) == 1 {
					return nil, nil
				}
				return &monitor.ProfileData{CPUUsageAvg: ptr.To(float32(3))}, nil
			}}
			request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}
			if _, err := r.Reconcile(t.Context(), request); err != nil {
				t.Fatal(err)
			}
			waitProfileCondition(t, fakeClock.HasWaiters)
			before := readUnscopedProfileJob(t, q, job.Name)
			for range 8 {
				r.enqueueJobProfile(job.Namespace, before)
			}
			if calls.Load() != 1 {
				t.Fatalf("scheduled retry was duplicated: %d", calls.Load())
			}
			fakeClock.Step(30 * time.Second)
			waitProfileIdle(t, r.profileQueue)
			after := readUnscopedProfileJob(t, q, job.Name)
			assertStoredCPU(t, after, 3)
			if calls.Load() != 2 || after.Status != before.Status || after.DeletedAt != before.DeletedAt ||
				!after.UpdatedAt.After(before.UpdatedAt) || !after.CompletedTimestamp.Equal(before.CompletedTimestamp) {
				t.Fatalf("backfill changed job state or retry count: before=%+v, after=%+v, calls=%d", before, after, calls.Load())
			}
		})
	}
}

func TestProfileRetryBounds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		age    time.Duration
		err    error
		delays []time.Duration
	}{
		{"old empty", 3 * time.Minute, nil, nil},
		{"recent empty", 0, nil, []time.Duration{30 * time.Second, time.Minute}},
		{"query error", 3 * time.Minute, errors.New("prometheus unavailable"), []time.Duration{30 * time.Second, time.Minute, 2 * time.Minute}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := useJobProfileDB(t)
			end := time.Now().UTC()
			job := newProfileJob(batch.Completed, end)
			insertProfileJob(t, q, job, end.Add(-time.Hour))
			r := jobProfileReconciler(t, &profileCapture{}, job)
			fakeClock := clocktesting.NewFakeClock(end.Add(tc.age))
			r.profileQueue.clock = fakeClock
			var calls atomic.Int32
			r.prometheusClient = &asyncProfileClient{query: func(context.Context) (*monitor.ProfileData, error) {
				calls.Add(1)
				return nil, tc.err
			}}
			if _, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}); err != nil {
				t.Fatal(err)
			}
			for _, delay := range tc.delays {
				waitProfileCondition(t, fakeClock.HasWaiters)
				fakeClock.Step(delay)
			}
			waitProfileIdle(t, r.profileQueue)
			if int(calls.Load()) != len(tc.delays)+1 || fakeClock.HasWaiters() {
				t.Fatalf("retry bound was not respected: calls=%d, active timer=%v", calls.Load(), fakeClock.HasWaiters())
			}
		})
	}
}

func TestProfileDatabaseErrorRetries(t *testing.T) {
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
	r := jobProfileReconciler(t, capture)
	fakeClock := clocktesting.NewFakeClock(end)
	r.profileQueue.clock = fakeClock
	var fail atomic.Bool
	fail.Store(true)
	db := q.Job.WithContext(t.Context()).UnderlyingDB()
	if err := db.Callback().Query().Before("gorm:query").Register("profile_transient_error", func(tx *gorm.DB) {
		if fail.Load() {
			_ = tx.AddError(errors.New("database temporarily unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	r.profileQueue.enqueue(vcJobProfileTask{namespace: job.Namespace, jobName: job.Name})
	waitProfileCondition(t, fakeClock.HasWaiters)
	fail.Store(false)
	fakeClock.Step(30 * time.Second)
	waitProfileIdle(t, r.profileQueue)
	assertStoredCPU(t, readProfileJob(t, q, job.Name), 2)
	if capture.calls != 1 {
		t.Fatalf("database failure unexpectedly queried Prometheus: calls=%d", capture.calls)
	}
	if fakeClock.HasWaiters() {
		t.Fatal("successful database retry left a timer")
	}
}

func TestProfileQueueCancellation(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run(map[bool]string{false: "active query", true: "pending retry"}[retry], func(t *testing.T) {
			q := useJobProfileDB(t)
			end := time.Now().UTC()
			job := newProfileJob(batch.Completed, end)
			insertProfileJob(t, q, job, end.Add(-time.Hour))
			r := newJobProfileReconciler(t, &profileCapture{}, job)
			cancel, done := startProfileWorkers(t, r)
			queue := r.profileQueue
			fakeClock := clocktesting.NewFakeClock(end)
			queue.clock = fakeClock
			started := make(chan struct{})
			r.prometheusClient = &asyncProfileClient{query: func(ctx context.Context) (*monitor.ProfileData, error) {
				close(started)
				if retry {
					return nil, errors.New("unavailable")
				}
				<-ctx.Done()
				return nil, ctx.Err()
			}}
			if _, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}); err != nil {
				t.Fatal(err)
			}
			waitProfileSignal(t, started)
			if retry {
				waitProfileCondition(t, fakeClock.HasWaiters)
			}
			cancel()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("workers or retry timers did not stop on cancellation")
			}
			if fakeClock.HasWaiters() || queue.enqueue(vcJobProfileTask{jobName: "after stop"}) {
				t.Fatal("stopped queue retained a timer or accepted new work")
			}
			waitProfileIdle(t, queue)
		})
	}
}

func TestProfileQueueCapacityAndQueuedDeduplication(t *testing.T) {
	q := newVCJobProfileQueue()
	q.jobs = make(chan vcJobProfileTask, 1)
	first := vcJobProfileTask{jobName: "first"}
	if !q.enqueue(first) {
		t.Fatal("queue rejected its first task")
	}
	if !q.enqueue(first) || q.enqueue(vcJobProfileTask{jobName: "second"}) || len(q.jobs) != 1 {
		t.Fatal("queue did not deduplicate or enforce its capacity")
	}
}
