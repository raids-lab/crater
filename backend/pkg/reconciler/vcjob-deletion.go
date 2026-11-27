package reconciler

import (
	"context"
	"errors"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/utils"
)

type vcJobDeletion struct {
	uid      types.UID
	exact    time.Time
	observed time.Time
}

func (r *VcJobReconciler) observeVCJobDeletion(e event.DeleteEvent) bool {
	job := e.Object.(*batch.Job)
	_, ended := observedVCJobTimestamps(job)
	if ended.IsZero() && job.DeletionTimestamp != nil {
		ended = job.DeletionTimestamp.Time
	}
	r.deletions.Store(client.ObjectKeyFromObject(job), vcJobDeletion{job.UID, ended, time.Now()})
	return true
}

func (r *VcJobReconciler) reconcileMissingVCJob(ctx context.Context, key client.ObjectKey) (ctrl.Result, error) {
	r.cancelPendingApprovalOrders(ctx, key.Name, "job not found in cluster")
	j := query.Job
	record, err := j.WithContext(ctx).Unscoped().Where(j.JobName.Eq(key.Name)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		r.deletions.Delete(key)
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	terminal := record.DeletedAt.Valid || utils.IsJobPhaseTerminal(record.Status) ||
		record.Status == model.Freed || record.Status == model.Deleted
	if record.Status == model.Prequeue || (terminal && !isProfileDataMissing(record.ProfileData)) {
		r.deletions.Delete(key)
		return ctrl.Result{}, nil
	}

	if !record.DeletedAt.Valid {
		pods, err := r.listVCJobPods(ctx, key, record.Attributes.Data())
		if err != nil {
			return ctrl.Result{}, err
		}
		if err := r.updateMissingVCJobRecord(ctx, key, record, pods, terminal); err != nil {
			return ctrl.Result{}, err
		}
	}
	r.deletions.Delete(key)
	if !r.enqueueJobProfile(key.Namespace, record) {
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}
	return ctrl.Result{}, nil
}

func (r *VcJobReconciler) updateMissingVCJobRecord(
	ctx context.Context, key client.ObjectKey, record *model.Job, pods []v1.Pod, terminal bool,
) error {
	updates := make(map[string]any)
	saved := record.Attributes.Data()
	if attributes := keepVCJobProfilePods(key, saved, nil, pods); attributes != saved {
		record.Attributes = datatypes.NewJSONType(attributes)
		updates["attributes"] = record.Attributes
	}
	if !terminal {
		record.RunningTimestamp = resolveJobRunningTimestamp(time.Time{}, record.RunningTimestamp, terminatedContainerStates(pods))
		record.CompletedTimestamp = r.freedVCJobTimestamp(key, record, pods)
		record.Status = model.Freed
		updates["running_timestamp"] = record.RunningTimestamp
		updates["completed_timestamp"] = record.CompletedTimestamp
		updates["status"] = record.Status
	}
	if len(updates) > 0 {
		j := query.Job
		if _, err := j.WithContext(ctx).Where(j.JobName.Eq(key.Name)).Updates(updates); err != nil {
			return err
		}
	}
	if !terminal && r.billingService != nil {
		if err := r.billingService.OnJobFinishedSettlement(ctx, record); err != nil {
			r.log.Error(err, "billing final settlement hook failed", "job", key.Name)
		}
	}
	return nil
}

func (r *VcJobReconciler) freedVCJobTimestamp(key client.ObjectKey, record *model.Job, pods []v1.Pod) time.Time {
	saved := record.Attributes.Data()
	var deletion vcJobDeletion
	if entry, ok := r.deletions.Load(key); ok {
		observed := entry.(vcJobDeletion)
		if saved == nil || saved.UID == "" || saved.UID == observed.uid {
			deletion = observed
		}
	}
	savedDeletion := r.recordedProfileTime(saved, observedProfileJobStop)
	if savedDeletion.IsZero() && saved != nil && saved.DeletionTimestamp != nil {
		savedDeletion = saved.DeletionTimestamp.Time
	}
	podStop := lastVCJobPodFinished(pods)
	if savedPodStop := r.recordedProfileTime(saved, observedProfilePodStop); savedPodStop.After(podStop) {
		podStop = savedPodStop
	}
	for _, candidate := range []time.Time{
		deletion.exact, savedDeletion, podStop, record.CompletedTimestamp, deletion.observed,
	} {
		if !candidate.IsZero() && !candidate.Before(record.RunningTimestamp) {
			return candidate
		}
	}
	return time.Now()
}

func (r *VcJobReconciler) recordedProfileTime(job *batch.Job, key string) time.Time {
	if job == nil || job.Annotations[key] == "" {
		return time.Time{}
	}
	at, err := time.Parse(time.RFC3339Nano, job.Annotations[key])
	if err != nil {
		r.log.Error(err, "invalid recorded profile timestamp", "job", job.Name, "annotation", key)
	}
	return at
}
