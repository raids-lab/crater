package prequeuewatcher

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/pkg/utils"
)

type blockingScope struct {
	accountID uint
	queue     string
}

type timedOutNormalBlocker struct {
	domain string
	nodes  sets.Set[string]
}

type timedOutNormalBlockers map[blockingScope][]timedOutNormalBlocker

func (w *PrequeueWatcher) activateNextPrequeueBatch(ctx context.Context, remaining int) (bool, error) {
	cfg, err := w.configService.GetPrequeueConfig(ctx)
	if err != nil {
		return true, err
	}
	prequeueCandidateSize := cfg.PrequeueCandidateSize
	limit := max(0, min(remaining, int(prequeueCandidateSize)))
	candidates, hasMore, err := w.selectActivationCandidates(ctx, limit+1)
	if err != nil {
		return true, err
	}
	if len(candidates) == 0 {
		return hasMore, nil
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
		hasMore = true
	}

	for _, candidate := range candidates {
		activated, err := w.claimAndActivatePrequeueJob(ctx, candidate)
		if err != nil {
			return true, err
		}
		if !activated {
			hasMore = true
		}
	}

	return hasMore, nil
}

// claimAndActivatePrequeueJob atomically claims a prequeue row before submitting it to Volcano.
func (w *PrequeueWatcher) claimAndActivatePrequeueJob(ctx context.Context, candidate *model.Job) (activated bool, err error) {
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

		job, err := vcjobservice.RestoreJobFromRecord(candidate)
		if err != nil {
			return err
		}
		err = vcjobservice.ActivateJob(ctx, w.k8sClient, w.serviceMgr, job)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}

		activated = true
		return nil
	})
	return activated, err
}

// selectActivationCandidates applies quota and timeout blockers while preserving FCFS order.
//
//nolint:gocyclo // Candidate filtering keeps quota and timeout ordering together.
func (w *PrequeueWatcher) selectActivationCandidates(
	ctx context.Context,
	limit int,
) ([]*model.Job, bool, error) {
	if limit <= 0 {
		return nil, false, nil
	}

	now := utils.GetLocalTime()
	cfg := w.currentRuntimeConfig()
	pendingBlockers := timedOutNormalBlockers{}
	prequeueBlockers := timedOutNormalBlockers{}
	if cfg.ShouldBlockByTimedOutPendingNormalJob() {
		loaded, err := w.loadTimedOutNormalBlockersByStatus(ctx, batch.Pending, now)
		if err != nil {
			return nil, false, err
		}
		pendingBlockers = loaded

		loaded, err = w.loadTimedOutNormalBlockersByStatus(ctx, model.Prequeue, now)
		if err != nil {
			return nil, false, err
		}
		prequeueBlockers = loaded
	}

	pageSize := limit
	if pageSize < defaultPageSize {
		pageSize = defaultPageSize
	}

	selected := make([]*model.Job, 0, limit)
	selectedResources := make(map[string]v1.ResourceList)
	offset := 0
	for {
		page, err := w.listPrequeueJobPage(ctx, offset, pageSize)
		if err != nil {
			return nil, false, err
		}
		if len(page) == 0 {
			return selected, false, nil
		}

		for _, candidate := range page {
			fitsQuota, err := w.candidateFitsQueueQuota(ctx, candidate, selectedResources)
			if err != nil {
				return nil, true, err
			}
			if !fitsQuota {
				continue
			}

			if isCandidateBlockedByTimedOutBlockers(
				candidate,
				cfg.BackfillEnabled,
				pendingBlockers,
				prequeueBlockers,
				now,
			) {
				continue
			}

			selected = append(selected, candidate)
			if candidate.ScheduleType != nil && *candidate.ScheduleType == model.ScheduleTypeNormal {
				key := fmt.Sprintf("%d:%d:%s", candidate.AccountID, candidate.UserID, candidate.Queue)
				selectedResources[key] = utils.SumResources(
					selectedResources[key],
					candidate.Resources.Data(),
				)
			}
			if len(selected) >= limit {
				return selected, true, nil
			}
		}
		if len(page) < pageSize {
			return selected, false, nil
		}
		offset += len(page)
	}
}

func isBlockedByTimedOutBlockers(blockersByScope timedOutNormalBlockers, candidate *model.Job) bool {
	if len(blockersByScope) == 0 {
		return false
	}
	if candidate == nil {
		return false
	}
	blockers := blockersByScope[blockingScope{accountID: candidate.AccountID, queue: candidate.Queue}]
	if len(blockers) == 0 {
		return false
	}
	candidateDomain := utils.GetJobResourceDomain(candidate)
	candidateNodes := utils.GetJobRecordExplicitNodeNames(candidate)
	for _, blocker := range blockers {
		if utils.CanResourceDomainBlock(blocker.domain, candidateDomain) &&
			nodeConstraintsOverlap(blocker.nodes, candidateNodes) {
			return true
		}
	}
	return false
}

func isCandidateBlockedByTimedOutBlockers(
	candidate *model.Job,
	backfillEnabled bool,
	pendingBlockers timedOutNormalBlockers,
	prequeueBlockers timedOutNormalBlockers,
	now time.Time,
) bool {
	if candidate == nil {
		return false
	}
	// Already-prequeued backfill jobs are not held by fairness blockers when backfill is disabled.
	if candidate.ScheduleType != nil && *candidate.ScheduleType == model.ScheduleTypeBackfill && !backfillEnabled {
		return false
	}

	// Timed-out pending normal jobs have priority over all overlapping prequeue candidates.
	if isBlockedByTimedOutBlockers(pendingBlockers, candidate) {
		return true
	}
	// A timed-out prequeue normal job should not block itself or peers that already reached tolerance.
	return !isTimedOutNormalJob(candidate, now) && isBlockedByTimedOutBlockers(prequeueBlockers, candidate)
}

func nodeConstraintsOverlap(left, right sets.Set[string]) bool {
	if left.Len() == 0 || right.Len() == 0 {
		return true
	}
	return left.Intersection(right).Len() > 0
}

func (w *PrequeueWatcher) listPrequeueJobPage(
	ctx context.Context,
	offset int,
	limit int,
) ([]*model.Job, error) {
	records := make([]*model.Job, 0, limit)
	err := w.q.Job.WithContext(ctx).UnderlyingDB().
		Model(&model.Job{}).
		Where("status = ?", model.Prequeue).
		Order("creation_timestamp ASC").
		Offset(offset).
		Limit(limit).
		Find(&records).Error
	return records, err
}

func (w *PrequeueWatcher) loadTimedOutNormalBlockersByStatus(
	ctx context.Context,
	status batch.JobPhase,
	now time.Time,
) (timedOutNormalBlockers, error) {
	pageSize := defaultPageSize
	offset := 0
	blockers := make(timedOutNormalBlockers)

	for {
		page, err := w.listNormalJobsByStatusPage(ctx, status, nil, nil, offset, pageSize)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return blockers, nil
		}

		for _, record := range page {
			if !isTimedOutNormalJob(record, now) {
				continue
			}
			scope := blockingScope{accountID: record.AccountID, queue: record.Queue}
			blockers[scope] = append(blockers[scope], timedOutNormalBlocker{
				domain: utils.GetJobResourceDomain(record),
				nodes:  utils.GetJobRecordExplicitNodeNames(record),
			})
		}

		if len(page) < pageSize {
			return blockers, nil
		}
		offset += len(page)
	}
}

func (w *PrequeueWatcher) listNormalJobsByStatusPage(
	ctx context.Context,
	status batch.JobPhase,
	accountID *uint,
	jobTypes []model.JobType,
	offset int,
	limit int,
) ([]*model.Job, error) {
	queryBuilder := w.q.Job.WithContext(ctx).UnderlyingDB().
		Model(&model.Job{}).
		Where(
			"status = ? AND schedule_type = ? AND waiting_tolerance_seconds IS NOT NULL",
			status,
			model.ScheduleTypeNormal,
		).
		Order("creation_timestamp ASC").
		Offset(offset).
		Limit(limit)
	if accountID != nil {
		queryBuilder = queryBuilder.Where("account_id = ?", *accountID)
	}
	if len(jobTypes) > 0 {
		queryBuilder = queryBuilder.Where("job_type IN ?", jobTypes)
	}

	page := make([]*model.Job, 0, limit)
	if err := queryBuilder.Find(&page).Error; err != nil {
		return nil, err
	}
	return page, nil
}

func (w *PrequeueWatcher) candidateFitsQueueQuota(
	ctx context.Context,
	candidate *model.Job,
	selectedResources map[string]v1.ResourceList,
) (bool, error) {
	if candidate == nil {
		return true, nil
	}
	if (candidate.ScheduleType != nil && *candidate.ScheduleType == model.ScheduleTypeBackfill) ||
		!w.currentRuntimeConfig().QueueQuotaEnabled || w.queueQuotaSvc == nil {
		return true, nil
	}

	key := fmt.Sprintf("%d:%d:%s", candidate.AccountID, candidate.UserID, candidate.Queue)
	projected := utils.SumResources(selectedResources[key], candidate.Resources.Data())
	resourceMap := utils.ToStringMap(projected)
	limitCheck, err := w.queueQuotaSvc.CheckUserResourceLimit(
		ctx,
		candidate.UserID,
		candidate.AccountID,
		candidate.Queue,
		resourceMap,
	)
	if err != nil {
		return false, err
	}
	return !limitCheck.Enabled || !limitCheck.Exceeded, nil
}
