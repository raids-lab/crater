package prequeuewatcher

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/utils"
)

const defaultPageSize = 50

func (w *PrequeueWatcher) runScanIfRequested(ctx context.Context) error {
	if !w.needScan {
		return nil
	}
	cfg := w.currentRuntimeConfig()
	remaining := int(cfg.MaxTotalActivationsPerRound)
	if remaining <= 0 {
		return nil
	}
	needRetry, err := w.runFullScanRound(ctx, remaining)
	w.needScan = needRetry
	return err
}

// runFullScanRound handles pending preemption before activating prequeue candidates.
func (w *PrequeueWatcher) runFullScanRound(ctx context.Context, remaining int) (bool, error) {
	preemptionPlan, err := w.findPendingNormalJobPreemptionPlan(ctx)
	if err != nil {
		return true, err
	}
	if preemptionPlan != nil {
		return w.deletePreemptedBackfillJobs(ctx, preemptionPlan)
	}

	return w.activateNextPrequeueBatch(ctx, remaining)
}

func (w *PrequeueWatcher) HasBlockingTimedOutPendingNormalJob(
	ctx context.Context,
	accountID uint,
	candidateResources v1.ResourceList,
) (bool, error) {
	return w.hasBlockingTimedOutPendingNormalJob(
		ctx,
		accountID,
		utils.GetResourceDomain(candidateResources),
	)
}

func isTimedOutNormalJob(record *model.Job, now time.Time) bool {
	if record == nil {
		return false
	}
	if record.ScheduleType == nil || *record.ScheduleType != model.ScheduleTypeNormal || record.WaitingToleranceSeconds == nil {
		return false
	}

	createdAt := record.CreationTimestamp
	if createdAt.IsZero() {
		createdAt = record.CreatedAt
	}
	if createdAt.IsZero() {
		return false
	}

	return now.After(createdAt.Add(time.Duration(*record.WaitingToleranceSeconds) * time.Second))
}
