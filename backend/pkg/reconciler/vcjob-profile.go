package reconciler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/monitor"
)

const (
	observedProfilePods    = "crater.raids.io/profile-pods"
	observedProfilePodStop = "crater.raids.io/profile-pod-stop"
	observedProfileJobStop = "crater.raids.io/profile-job-stop"
)

func (r *VcJobReconciler) listVCJobPods(ctx context.Context, key client.ObjectKey, job *batch.Job) ([]v1.Pod, error) {
	var pods v1.PodList
	if err := r.List(ctx, &pods, client.InNamespace(key.Namespace), client.MatchingLabels{batch.JobNameKey: key.Name}); err != nil {
		return nil, fmt.Errorf("list pods for VCJob %s: %w", key, err)
	}
	if job != nil && job.UID != "" {
		pods.Items = slices.DeleteFunc(pods.Items, func(pod v1.Pod) bool {
			owner := metav1.GetControllerOfNoCopy(&pod)
			return owner != nil && owner.Kind == "Job" && owner.UID != job.UID
		})
	}
	return pods.Items, nil
}

func keepVCJobProfilePods(key client.ObjectKey, saved, live *batch.Job, pods []v1.Pod) *batch.Job {
	observed, stopped := observeProfilePods(saved, live, pods)
	if saved == nil && observed == "" {
		return nil
	}
	deletion := observedProfileDeletion(saved, live)
	if saved != nil && saved.Annotations[observedProfilePods] == observed &&
		saved.Annotations[observedProfilePodStop] == stopped && saved.Annotations[observedProfileJobStop] == deletion {
		return saved
	}
	snapshot := saved.DeepCopy()
	if snapshot == nil {
		snapshot = &batch.Job{ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace}}
	}
	if snapshot.Annotations == nil {
		snapshot.Annotations = make(map[string]string)
	}
	setProfileAnnotation(snapshot.Annotations, observedProfilePods, observed)
	setProfileAnnotation(snapshot.Annotations, observedProfilePodStop, stopped)
	setProfileAnnotation(snapshot.Annotations, observedProfileJobStop, deletion)
	return snapshot
}

func observedProfileDeletion(saved, live *batch.Job) string {
	if live != nil && live.DeletionTimestamp != nil {
		return live.DeletionTimestamp.UTC().Format(time.RFC3339Nano)
	}
	if saved != nil && saved != live {
		return saved.Annotations[observedProfileJobStop]
	}
	return ""
}

func observeProfilePods(saved, live *batch.Job, pods []v1.Pod) (names, stopped string) {
	observed := make([]string, 0, len(pods))
	for i := range pods {
		observed = append(observed, pods[i].Name)
	}
	if live != nil {
		observed = append(observed, getPodNamesFromJobTemplate(live)...)
	}
	// Only accept profile metadata previously written by the reconciler.
	if saved != nil && saved != live {
		observed = append(observed, strings.Fields(saved.Annotations[observedProfilePods])...)
		stopped = saved.Annotations[observedProfilePodStop]
	}
	if len(pods) > 0 {
		if finished := lastVCJobPodFinished(pods); !finished.IsZero() {
			if existing, err := time.Parse(time.RFC3339Nano, stopped); err != nil || finished.After(existing) {
				stopped = finished.UTC().Format(time.RFC3339Nano)
			}
		} else {
			stopped = ""
		}
	}
	return strings.Join(compactProfilePodNames(observed), " "), stopped
}

func setProfileAnnotation(annotations map[string]string, key, value string) {
	if value == "" {
		delete(annotations, key)
	} else {
		annotations[key] = value
	}
}

func profilePodNames(job *batch.Job) []string {
	names := getPodNamesFromJobTemplate(job)
	if job != nil {
		names = append(names, strings.Fields(job.Annotations[observedProfilePods])...)
	}
	return compactProfilePodNames(names)
}

func compactProfilePodNames(names []string) []string {
	slices.Sort(names)
	return slices.Compact(names)
}

func isProfileDataMissing(profile *datatypes.JSONType[*monitor.ProfileData]) bool {
	return profile == nil || profile.Data() == nil
}

func missingJobProfilePods(record *model.Job) []string {
	if !isProfileDataMissing(record.ProfileData) || record.RunningTimestamp.IsZero() ||
		!record.CompletedTimestamp.After(record.RunningTimestamp) {
		return nil
	}
	return profilePodNames(record.Attributes.Data())
}

func (r *VcJobReconciler) enqueueJobProfile(namespace string, record *model.Job) bool {
	if len(missingJobProfilePods(record)) == 0 {
		return true
	}
	if !r.profileQueue.enqueue(vcJobProfileTask{
		namespace: namespace, jobName: record.JobName,
	}) {
		r.log.Error(errors.New("profile queue full or stopped"), "profile backfill must be requeued", "job", record.JobName)
		return false
	}
	return true
}

func (r *VcJobReconciler) retryJobProfile(ctx context.Context, task *vcJobProfileTask) time.Duration {
	j := query.Job
	record, err := j.WithContext(ctx).Unscoped().Where(j.JobName.Eq(task.jobName)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		r.log.Info("job not found in database; dropping profile backfill", "job", task.jobName)
		return 0
	}
	var empty bool
	if err == nil {
		empty, err = r.updateMissingJobProfile(ctx, task.namespace, record)
	}
	if ctx.Err() != nil {
		return 0
	}
	q := r.profileQueue
	if err != nil {
		r.log.Error(err, "job profile backfill failed", "job", task.jobName, "retries", task.errorRetries)
		if task.errorRetries < len(q.errorBackoff) {
			delay := q.errorBackoff[task.errorRetries]
			task.errorRetries++
			return delay
		}
		r.log.Info("profile retries exhausted; dropping backfill", "job", task.jobName)
	} else if empty && q.clock.Since(record.CompletedTimestamp) < profileRecentWindow && task.emptyRetries < len(q.emptyBackoff) {
		delay := q.emptyBackoff[task.emptyRetries]
		task.emptyRetries++
		r.log.Info("recent job profile has no samples; retrying", "job", task.jobName, "delay", delay)
		return delay
	}
	return 0
}

func (r *VcJobReconciler) updateMissingJobProfile(ctx context.Context, namespace string, record *model.Job) (bool, error) {
	podNames := missingJobProfilePods(record)
	if len(podNames) == 0 {
		return false, nil
	}
	profile, err := r.prometheusClient.QueryProfileDataContext(ctx, namespace, podNames, record.RunningTimestamp, record.CompletedTimestamp)
	if err != nil {
		return false, fmt.Errorf("collect profile for job %s: %w", record.JobName, err)
	}
	if profile == nil {
		return true, nil
	}

	// Validate before GORM binds the value. JSONType.GormValue discards marshal errors.
	payload, err := json.Marshal(profile)
	if err != nil {
		r.log.Error(err, "discarding invalid job profile data", "job", record.JobName)
		return false, nil
	}
	j := query.Job
	info, err := j.WithContext(ctx).Unscoped().Where(j.ID.Eq(record.ID), j.JobName.Eq(record.JobName)).
		Where(j.WithContext(ctx).Where(j.ProfileData.IsNull()).Or(j.ProfileData.Eq(datatypes.JSON("null")))).
		Update(j.ProfileData, datatypes.JSON(payload))
	if err != nil {
		return false, err
	}
	if info.RowsAffected == 0 {
		r.log.Info("job removed or profile already present; skipping profile write", "job", record.JobName)
	}
	return false, nil
}
