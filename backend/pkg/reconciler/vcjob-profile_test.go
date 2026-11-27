package reconciler

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/monitor"
)

type profileCapture struct {
	monitor.PrometheusInterface
	namespace string
	pods      []string
	from, to  time.Time
	result    *monitor.ProfileData
	err       error
	onQuery   func()
	calls     int
}

func (p *profileCapture) QueryProfileDataContext(_ context.Context, namespace string, pods []string, from, to time.Time) (*monitor.ProfileData, error) {
	p.namespace, p.pods, p.from, p.to = namespace, pods, from, to
	p.calls++
	if p.onQuery != nil {
		p.onQuery()
	}
	return p.result, p.err
}

func useJobProfileDB(t *testing.T) *query.Query {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, IgnoreRelationshipsWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	})
	if err = db.AutoMigrate(&model.Job{}, &model.ApprovalOrder{}); err != nil {
		t.Fatal(err)
	}
	q := query.Use(db)
	previousJob, previousOrder := query.Job, query.ApprovalOrder
	query.Job, query.ApprovalOrder = &q.Job, &q.ApprovalOrder
	t.Cleanup(func() { query.Job, query.ApprovalOrder = previousJob, previousOrder })
	return q
}

func jobProfileReconciler(t *testing.T, capture *profileCapture, objects ...client.Object) *VcJobReconciler {
	t.Helper()
	r := newJobProfileReconciler(t, capture, objects...)
	startProfileWorkers(t, r)
	return r
}

func newJobProfileReconciler(t *testing.T, capture *profileCapture, objects ...client.Object) *VcJobReconciler {
	t.Helper()
	scheme := podGroupScheme()
	if err := batch.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return &VcJobReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build(),
		log:    logr.Discard(), kubeClient: kubefake.NewClientset(), prometheusClient: capture,
		profileQueue: newVCJobProfileQueue(),
	}
}

func newProfileJob(phase batch.JobPhase, timeObserved time.Time) *batch.Job {
	return &batch.Job{
		ObjectMeta: metav1.ObjectMeta{Name: vcjobTestName, Namespace: vcjobTestNamespace, UID: vcjobTestUID},
		Spec:       batch.JobSpec{Tasks: []batch.TaskSpec{{Name: "master", Replicas: 1}, {Name: "worker", Replicas: 1}}},
		Status:     batch.JobStatus{State: batch.JobState{Phase: phase, LastTransitionTime: metav1.NewTime(timeObserved)}},
	}
}

func insertProfileJob(t *testing.T, q *query.Query, job *batch.Job, start time.Time) {
	t.Helper()
	if err := q.Job.WithContext(t.Context()).Create(&model.Job{
		Name: job.Name, JobName: job.Name, UserID: 1, AccountID: 1, JobType: model.JobTypeCustom,
		Status: batch.Running, RunningTimestamp: start, Attributes: datatypes.NewJSONType(job),
	}); err != nil {
		t.Fatal(err)
	}
}

func readProfileJob(t *testing.T, q *query.Query, jobName string) *model.Job {
	t.Helper()
	record, err := q.Job.WithContext(t.Context()).Where(q.Job.JobName.Eq(jobName)).First()
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func assertProfileLifecycle(t *testing.T, capture *profileCapture, start, end time.Time, pods []string) {
	t.Helper()
	if capture.namespace != vcjobTestNamespace || !capture.from.Equal(start) || !capture.to.Equal(end) ||
		!reflect.DeepEqual(capture.pods, pods) {
		t.Fatalf("got namespace %q, bounds [%v, %v], pods %v; want %q [%v, %v] %v",
			capture.namespace, capture.from, capture.to, capture.pods, vcjobTestNamespace, start, end, pods)
	}
}

func TestCompletedJobRetriesProfileAfterStatusIsStored(t *testing.T) {
	q := useJobProfileDB(t)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(48 * time.Hour)
	job := newProfileJob(batch.Completed, end)
	insertProfileJob(t, q, job, start)
	promErr := errors.New("remote read temporarily unavailable")
	capture := &profileCapture{err: promErr}
	capture.onQuery = func() {
		record := readProfileJob(t, q, job.Name)
		if record.Status != batch.Completed || !record.CompletedTimestamp.Equal(end) {
			t.Fatalf("profiling before terminal state is persisted: %s, %v", record.Status, record.CompletedTimestamp)
		}
	}
	r := jobProfileReconciler(t, capture, job)
	r.profileQueue.errorBackoff = nil
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}
	reconcileProfileJob(t, r, req)
	assertProfileLifecycle(t, capture, start, end, []string{job.Name + "-master-0", job.Name + "-worker-0"})
	if record := readProfileJob(t, q, job.Name); record.ProfileData != nil {
		t.Fatal("failed request stored a profile")
	}

	capture.err = nil
	capture.result = &monitor.ProfileData{CPUUsageAvg: ptr.To(float32(math.NaN()))}
	reconcileProfileJob(t, r, req)
	if record := readProfileJob(t, q, job.Name); record.ProfileData != nil || record.Status != batch.Completed {
		t.Fatal("invalid profile was persisted or state did not transition cleanly")
	}

	j := q.Job
	if _, err := j.WithContext(t.Context()).Where(j.JobName.Eq(job.Name)).Updates(map[string]any{
		"profile_data": datatypes.JSON("null"),
	}); err != nil {
		t.Fatal(err)
	}
	record := readProfileJob(t, q, job.Name)
	if record.ProfileData == nil || record.ProfileData.Data() != nil {
		t.Fatalf("fixture should represent a JSON null wrapper, got %+v", record.ProfileData)
	}
	capture.result = &monitor.ProfileData{CPUUsageAvg: ptr.To(float32(2))}
	reconcileProfileJob(t, r, req)
	assertStoredCPU(t, readProfileJob(t, q, job.Name), 2)
}

func TestCompletedJobBackfillsProfileAfterSoftDeletion(t *testing.T) {
	q := useJobProfileDB(t)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	job := newProfileJob(batch.Completed, end)
	insertProfileJob(t, q, job, start)
	promErr := errors.New("remote read temporarily unavailable")
	capture := &profileCapture{err: promErr}
	r := jobProfileReconciler(t, capture, job)
	r.profileQueue.errorBackoff = nil
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}
	reconcileProfileJob(t, r, req)
	record := readProfileJob(t, q, job.Name)
	if err := r.Delete(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	j := q.Job
	if _, err := j.WithContext(t.Context()).Where(j.ID.Eq(record.ID)).Delete(); err != nil {
		t.Fatal(err)
	}
	deletedJob := j.WithContext(t.Context()).Unscoped().Where(j.ID.Eq(record.ID))
	before, err := deletedJob.First()
	if err != nil {
		t.Fatal(err)
	}
	if !before.DeletedAt.Valid || before.ProfileData != nil {
		t.Fatalf("expected a soft-deleted job without a profile: %+v", before)
	}

	capture.err = nil
	capture.result = &monitor.ProfileData{Version: monitor.ProfileDataVersion, CPUUsageAvg: ptr.To(float32(2))}
	reconcileProfileJob(t, r, req)
	after, err := deletedJob.First()
	if err != nil {
		t.Fatal(err)
	}
	if !after.DeletedAt.Valid || !after.DeletedAt.Time.Equal(before.DeletedAt.Time) ||
		after.Status != batch.Completed || !after.CompletedTimestamp.Equal(end) {
		t.Fatalf("backfill changed the soft-deleted job's state: %+v", after)
	}
	assertStoredCPU(t, after, 2)
	if after.ProfileData.Data().Version != monitor.ProfileDataVersion {
		t.Fatalf("backfilled profile has version %d", after.ProfileData.Data().Version)
	}
}

func TestFreedJobUsesDeletionTimeAndRetriesProfile(t *testing.T) {
	q := useJobProfileDB(t)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	job := newProfileJob(batch.Running, start)
	insertProfileJob(t, q, job, start)
	j := q.Job
	if _, err := j.WithContext(t.Context()).Where(j.JobName.Eq(job.Name)).
		Updates(map[string]any{"completed_timestamp": start.Add(10 * time.Minute)}); err != nil {
		t.Fatal(err)
	}

	capture := &profileCapture{result: &monitor.ProfileData{CPUUsageAvg: ptr.To(float32(math.Inf(1)))}}
	r := jobProfileReconciler(t, capture)
	job.DeletionTimestamp = ptr.To(metav1.NewTime(end))
	r.observeVCJobDeletion(event.DeleteEvent{Object: job})
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}
	reconcileProfileJob(t, r, req)
	freed := readProfileJob(t, q, job.Name)
	if freed.Status != model.Freed || freed.ProfileData != nil || !freed.CompletedTimestamp.Equal(end) {
		t.Fatalf("invalid profile affected Freed transition: %s, %v, %+v", freed.Status, freed.CompletedTimestamp, freed.ProfileData)
	}
	assertProfileLifecycle(t, capture, start, end, []string{job.Name + "-master-0", job.Name + "-worker-0"})
	capture.result = &monitor.ProfileData{CPUUsageAvg: ptr.To(float32(3))}
	reconcileProfileJob(t, r, req)
	assertProfileLifecycle(t, capture, start, end, []string{job.Name + "-master-0", job.Name + "-worker-0"})
	assertStoredCPU(t, readProfileJob(t, q, job.Name), 3)
}

func TestFreedJobWithoutAttributes(t *testing.T) {
	q := useJobProfileDB(t)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	job := newProfileJob(batch.Running, start)
	insertProfileJob(t, q, job, start)
	j := q.Job
	if _, err := j.WithContext(t.Context()).Where(j.JobName.Eq(job.Name)).Updates(map[string]any{
		"attributes": datatypes.JSON("null"),
	}); err != nil {
		t.Fatal(err)
	}

	capture := &profileCapture{}
	r := jobProfileReconciler(t, capture)
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}
	if _, err := r.Reconcile(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	waitProfileIdle(t, r.profileQueue)
	record := readProfileJob(t, q, job.Name)
	if capture.calls != 0 || record.Status != model.Freed || record.Attributes.Data() != nil {
		t.Fatalf("expected Freed without requesting a profile, got %s, calls=%d", record.Status, capture.calls)
	}
}

func TestScaledPodsAreRememberedAfterDisappearing(t *testing.T) {
	for _, status := range []batch.JobPhase{batch.Completed, model.Freed} {
		t.Run(string(status), func(t *testing.T) {
			q := useJobProfileDB(t)
			start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
			end := start.Add(time.Hour)
			job := newProfileJob(batch.Running, start)
			insertProfileJob(t, q, job, start)
			scaledName := job.Name + "-worker-3"
			scaledPod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: scaledName, Namespace: job.Namespace,
				Labels: map[string]string{batch.JobNameKey: job.Name}}}
			unrelated := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: job.Name + "-other", Namespace: job.Namespace,
				Labels: map[string]string{batch.JobNameKey: "other-job"}}}
			capture := &profileCapture{result: &monitor.ProfileData{CPUUsageAvg: ptr.To(float32(4))}}
			r := jobProfileReconciler(t, capture, job, scaledPod, unrelated)
			request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}
			reconcileProfileJob(t, r, request)
			if err := r.Delete(t.Context(), scaledPod); err != nil {
				t.Fatal(err)
			}
			if status == batch.Completed {
				if err := r.Get(t.Context(), request.NamespacedName, job); err != nil {
					t.Fatal(err)
				}
				job.Status.State = batch.JobState{Phase: batch.Completed, LastTransitionTime: metav1.NewTime(end)}
				if err := r.Update(t.Context(), job); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := r.Delete(t.Context(), job); err != nil {
					t.Fatal(err)
				}
				job.DeletionTimestamp = ptr.To(metav1.NewTime(end))
				r.observeVCJobDeletion(event.DeleteEvent{Object: job})
			}
			reconcileProfileJob(t, r, request)
			assertProfileLifecycle(t, capture, start, end, []string{job.Name + "-master-0", job.Name + "-worker-0", scaledName})
			record := readProfileJob(t, q, job.Name)
			if record.Status != status {
				t.Fatalf("got status %s, want %s", record.Status, status)
			}
			assertStoredCPU(t, record, 4)
		})
	}
}

func reconcileProfileJob(t *testing.T, r *VcJobReconciler, request ctrl.Request) {
	t.Helper()
	if _, err := r.Reconcile(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	waitProfileIdle(t, r.profileQueue)
}

func assertStoredCPU(t *testing.T, record *model.Job, want float32) {
	t.Helper()
	if isProfileDataMissing(record.ProfileData) || record.ProfileData.Data().CPUUsageAvg == nil ||
		*record.ProfileData.Data().CPUUsageAvg != want {
		t.Fatalf("profile was not stored with CPU usage %v: %+v", want, record.ProfileData)
	}
}

func TestFreedJobUsesPreviouslyObservedPodStop(t *testing.T) {
	q := useJobProfileDB(t)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	stop := start.Add(time.Hour)
	job := newProfileJob(batch.Running, start)
	insertProfileJob(t, q, job, start)
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: job.Name + "-worker-0", Namespace: job.Namespace, Labels: map[string]string{batch.JobNameKey: job.Name}},
		Status: v1.PodStatus{Phase: v1.PodSucceeded, ContainerStatuses: []v1.ContainerStatus{
			{State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{FinishedAt: metav1.NewTime(stop)}}},
		}},
	}
	capture := &profileCapture{}
	r := jobProfileReconciler(t, capture, job, pod)
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}
	reconcileProfileJob(t, r, request)
	if err := r.Delete(t.Context(), pod); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	reconcileProfileJob(t, r, request)
	record := readProfileJob(t, q, job.Name)
	if record.Status != model.Freed || !record.CompletedTimestamp.Equal(stop) {
		t.Fatalf("observed pod stop was lost: %s at %v", record.Status, record.CompletedTimestamp)
	}
	assertProfileLifecycle(t, capture, start, stop, []string{job.Name + "-master-0", job.Name + "-worker-0"})
}

func TestFreedJobUsesPersistedVCJobDeletionTime(t *testing.T) {
	q := useJobProfileDB(t)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	stop := start.Add(time.Hour)
	job := newProfileJob(batch.Running, start)
	insertProfileJob(t, q, job, start)
	job.DeletionTimestamp = ptr.To(metav1.NewTime(stop))
	job.Finalizers = []string{"test.example.com/cleanup"}

	capture := &profileCapture{}
	r := jobProfileReconciler(t, capture, job)
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}
	reconcileProfileJob(t, r, request)
	snapshot := readProfileJob(t, q, job.Name).Attributes.Data()
	if snapshot.DeletionTimestamp != nil || snapshot.Annotations[observedProfileJobStop] != stop.Format(time.RFC3339Nano) {
		t.Fatalf("expected original job attributes plus the recorded stop time, got %v, %v",
			snapshot.DeletionTimestamp, snapshot.Annotations[observedProfileJobStop])
	}

	// Simulate a controller restart after Kubernetes removed the finalized Job.
	r = jobProfileReconciler(t, capture)
	reconcileProfileJob(t, r, request)
	record := readProfileJob(t, q, job.Name)
	if record.Status != model.Freed || !record.CompletedTimestamp.Equal(stop) {
		t.Fatalf("observed job deletion time was lost: %s at %v", record.Status, record.CompletedTimestamp)
	}
	assertProfileLifecycle(t, capture, start, stop, []string{job.Name + "-master-0", job.Name + "-worker-0"})
}

func TestScaledOutReplicaTemplateIsRetainedWithoutObservingPod(t *testing.T) {
	q := useJobProfileDB(t)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	stop := start.Add(time.Hour)
	job := newProfileJob(batch.Running, start)
	insertProfileJob(t, q, job, start)

	// Scale worker from 1 to 2 before worker-1 Pod appears, then complete without observing worker-1 Pod.
	scaled := job.DeepCopy()
	scaled.Spec.Tasks[1].Replicas = 2
	capture := &profileCapture{}
	r := jobProfileReconciler(t, capture, scaled)
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}
	reconcileProfileJob(t, r, request)

	scaled.Status.State = batch.JobState{Phase: batch.Completed, LastTransitionTime: metav1.NewTime(stop)}
	if err := r.Update(t.Context(), scaled); err != nil {
		t.Fatal(err)
	}
	reconcileProfileJob(t, r, request)
	assertProfileLifecycle(t, capture, start, stop, []string{
		job.Name + "-master-0", job.Name + "-worker-0", job.Name + "-worker-1",
	})
}

func TestFreedJobRetainsLatestPodStopAfterLaterPodIsRemoved(t *testing.T) {
	q := useJobProfileDB(t)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	earlier := start.Add(10 * time.Minute)
	later := start.Add(20 * time.Minute)
	job := newProfileJob(batch.Running, start)
	insertProfileJob(t, q, job, start)

	pod1 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: job.Name + "-master-0", Namespace: job.Namespace, Labels: map[string]string{batch.JobNameKey: job.Name}},
		Status: v1.PodStatus{Phase: v1.PodSucceeded, ContainerStatuses: []v1.ContainerStatus{
			{State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{FinishedAt: metav1.NewTime(earlier)}}},
		}},
	}
	pod2 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: job.Name + "-worker-0", Namespace: job.Namespace, Labels: map[string]string{batch.JobNameKey: job.Name}},
		Status: v1.PodStatus{Phase: v1.PodSucceeded, ContainerStatuses: []v1.ContainerStatus{
			{State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{FinishedAt: metav1.NewTime(later)}}},
		}},
	}
	capture := &profileCapture{}
	r := jobProfileReconciler(t, capture, job, pod1, pod2)
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(job)}
	reconcileProfileJob(t, r, request)

	// Delete the later-finishing pod and the VCJob while the earlier-finishing pod still exists.
	if err := r.Delete(t.Context(), pod2); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	reconcileProfileJob(t, r, request)
	record := readProfileJob(t, q, job.Name)
	if record.Status != model.Freed || !record.CompletedTimestamp.Equal(later) {
		t.Fatalf("expected Freed at latest pod stop %v, got %s at %v", later, record.Status, record.CompletedTimestamp)
	}
	assertProfileLifecycle(t, capture, start, later, []string{job.Name + "-master-0", job.Name + "-worker-0"})
}
