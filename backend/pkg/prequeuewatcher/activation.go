package prequeuewatcher

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
)

// submitTimeout bounds the row lock the claim holds: the transaction spans a call to the apiserver,
// and the rest config carries no client-side deadline of its own.
const submitTimeout = 30 * time.Second

// claimAndActivatePrequeueJob atomically claims a prequeue row before submitting it to Volcano.
func (w *PrequeueWatcher) claimAndActivatePrequeueJob(
	ctx context.Context,
	candidate *model.Job,
) (activated bool, err error) {
	err = w.q.Transaction(func(tx *query.Query) error {
		info, err := tx.Job.WithContext(ctx).
			Where(tx.Job.ID.Eq(candidate.ID), tx.Job.Status.Eq(string(model.Prequeue))).
			Updates(model.Job{Status: batch.Pending})
		if err != nil {
			return err
		}
		if info.RowsAffected == 0 {
			return nil
		}

		job, err := w.restoreJobForActivation(ctx, candidate)
		if err != nil {
			return err
		}
		submitCtx, cancel := context.WithTimeout(ctx, submitTimeout)
		defer cancel()
		err = vcjobservice.ActivateJob(submitCtx, w.k8sClient, w.serviceMgr, job)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}

		activated = true
		return nil
	})
	return activated, err
}

func (w *PrequeueWatcher) restoreJobForActivation(ctx context.Context, candidate *model.Job) (*batch.Job, error) {
	job, err := vcjobservice.RestoreJobFromRecord(candidate)
	if err != nil {
		return nil, err
	}
	if err := service.ApplyJobPodBandwidth(ctx, w.configService, w.kubeClient, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (w *PrequeueWatcher) listPrequeueJobs(ctx context.Context, limit int) ([]*model.Job, error) {
	records := make([]*model.Job, 0, limit)
	err := w.q.Job.WithContext(ctx).UnderlyingDB().
		Model(&model.Job{}).
		Where("status = ?", model.Prequeue).
		Order("creation_timestamp ASC").
		Limit(limit).
		Find(&records).Error
	return records, err
}
