package reconciler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr/funcr"
	"gorm.io/datatypes"
	clocktesting "k8s.io/utils/clock/testing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/monitor"
)

func runTestProfileQueue(
	t *testing.T, ctx context.Context, q *vcJobProfileQueue,
	collect func(context.Context, *vcJobProfileTask) time.Duration,
) {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := q.run(ctx, collect); err != nil {
			t.Error(err)
		}
	}()
	t.Cleanup(func() {
		cancel()
		waitProfileSignal(t, done)
	})
}

func TestProfileQueueWorkerRecoversAndFinishesPanickedTask(t *testing.T) {
	q := newVCJobProfileQueue()
	logged := make(chan string, 1)
	logger := funcr.New(func(_, line string) { logged <- line }, funcr.Options{})
	runTestProfileQueue(t, log.IntoContext(t.Context(), logger), q,
		func(_ context.Context, task *vcJobProfileTask) time.Duration {
			if task.jobName == "panic-job" {
				panic("broken profile")
			}
			return 0
		})
	if !q.enqueue(vcJobProfileTask{jobName: "panic-job"}) {
		t.Fatal("panic fixture was rejected")
	}
	waitProfileIdle(t, q)
	select {
	case line := <-logged:
		if !strings.Contains(line, `"job"="panic-job"`) || !strings.Contains(line, `"error"=`) ||
			!strings.Contains(line, "broken profile") {
			t.Fatalf("panic log lost its error or job: %s", line)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("worker panic was not logged")
	}
	if !q.enqueue(vcJobProfileTask{jobName: "next-job"}) {
		t.Fatal("queue rejected work after a panic")
	}
	waitProfileIdle(t, q)
}

func TestProfileQueueRetryTimersCountTowardCapacity(t *testing.T) {
	q := newVCJobProfileQueue()
	q.jobs = make(chan vcJobProfileTask, 1)
	fakeClock := clocktesting.NewFakeClock(time.Now())
	q.clock = fakeClock
	runTestProfileQueue(t, t.Context(), q,
		func(_ context.Context, task *vcJobProfileTask) time.Duration {
			if task.errorRetries == 0 {
				task.errorRetries++
				return 30 * time.Second
			}
			return 0
		})
	first := vcJobProfileTask{jobName: "retrying"}
	if !q.enqueue(first) {
		t.Fatal("retry fixture was rejected")
	}
	waitProfileCondition(t, fakeClock.HasWaiters)
	if len(q.jobs) != 0 || !q.enqueue(first) || q.enqueue(vcJobProfileTask{jobName: "overflow"}) {
		t.Fatal("waiting retry did not reserve capacity or deduplicate")
	}
	fakeClock.Step(30 * time.Second)
	waitProfileIdle(t, q)
	if !q.enqueue(vcJobProfileTask{jobName: "after-retry", errorRetries: 1}) {
		t.Fatal("finished retry did not release capacity")
	}
	waitProfileIdle(t, q)
}

func TestProfileEnqueueSkipsIneligibleJobs(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name   string
		change func(*model.Job)
	}{
		{"not started", func(job *model.Job) { job.RunningTimestamp = time.Time{} }},
		{"not completed", func(job *model.Job) { job.CompletedTimestamp = time.Time{} }},
		{"zero duration", func(job *model.Job) { job.CompletedTimestamp = job.RunningTimestamp }},
		{"negative duration", func(job *model.Job) { job.CompletedTimestamp = job.RunningTimestamp.Add(-time.Second) }},
		{"profile present", func(job *model.Job) {
			profile := datatypes.NewJSONType(&monitor.ProfileData{})
			job.ProfileData = &profile
		}},
		{"no pods", func(job *model.Job) { job.Attributes = datatypes.NewJSONType[*batch.Job](nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newJobProfileReconciler(t, &profileCapture{})
			job := &model.Job{
				JobName: "ineligible", RunningTimestamp: now.Add(-time.Hour), CompletedTimestamp: now,
				Attributes: datatypes.NewJSONType(newProfileJob(batch.Completed, now)),
			}
			tc.change(job)
			if !r.enqueueJobProfile(vcjobTestNamespace, job) || len(r.profileQueue.pending) != 0 || len(r.profileQueue.jobs) != 0 {
				t.Fatal("ineligible job was enqueued or requested a retry")
			}
		})
	}
}

func TestProfileQueueRejectionRequeuesReconcileAndDeletion(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		for _, deletion := range []bool{false, true} {
			name := "full"
			if stopped {
				name = "stopped"
			}
			if deletion {
				name += "/deleted"
			} else {
				name += "/completed"
			}
			t.Run(name, func(t *testing.T) {
				q := useJobProfileDB(t)
				end := time.Now()
				job := newProfileJob(batch.Completed, end)
				insertProfileJob(t, q, job, end.Add(-time.Hour))
				r := newJobProfileReconciler(t, &profileCapture{}, job)
				rejections := make(chan string, 1)
				r.log = funcr.New(func(_, line string) { rejections <- line }, funcr.Options{})
				if deletion {
					if err := r.Delete(t.Context(), job); err != nil {
						t.Fatal(err)
					}
				}
				r.profileQueue.stopped = stopped
				if !stopped {
					r.profileQueue.jobs = make(chan vcJobProfileTask, 1)
					if !r.profileQueue.enqueue(vcJobProfileTask{jobName: "blocking-job"}) {
						t.Fatal("could not fill queue")
					}
				}
				result, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)})
				if err != nil || result.RequeueAfter != time.Minute {
					t.Fatalf("rejected profile lost its controller retry: result=%+v, error=%v", result, err)
				}
				select {
				case line := <-rejections:
					if !strings.Contains(line, `"error"=`) || !strings.Contains(line, job.Name) {
						t.Fatalf("rejection was not logged as a job error: %s", line)
					}
				default:
					t.Fatal(errors.New("queue rejection was not logged"))
				}
			})
		}
	}
}
