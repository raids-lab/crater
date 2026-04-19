package prequeuewatcher

import (
	"context"
	"sort"

	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/utils"
)

type preemptionPlan struct {
	nodeName string
	jobs     []*model.Job
}

type singleNodeJobRequirements struct {
	podSpec  *v1.PodSpec
	requests v1.ResourceList
}

func isPreemptableBackfillJobType(jobType model.JobType) bool {
	switch jobType {
	case model.JobTypeJupyter, model.JobTypeCustom, model.JobTypeWebIDE:
		return true
	default:
		return false
	}
}

// findPendingNormalJobPreemptionPlan picks the first timed-out pending normal job with a backfill plan.
func (w *PrequeueWatcher) findPendingNormalJobPreemptionPlan(ctx context.Context) (*preemptionPlan, error) {
	pageSize := defaultPageSize
	offset := 0
	now := utils.GetLocalTime()

	for {
		page, err := w.listNormalJobsByStatusPage(ctx, batch.Pending, nil, nil, offset, pageSize)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return nil, nil
		}

		for _, record := range page {
			if !isTimedOutNormalJob(record, now) || !utils.IsSingleNodeJob(record) {
				continue
			}

			plan, err := w.findSingleNodePreemptionPlan(ctx, record)
			if err != nil {
				return nil, err
			}
			if plan != nil && len(plan.jobs) > 0 {
				return plan, nil
			}
		}

		if len(page) < pageSize {
			return nil, nil
		}
		offset += len(page)
	}
}

func (w *PrequeueWatcher) hasBlockingTimedOutPendingNormalJob(
	ctx context.Context,
	accountID uint,
	candidateDomain string,
) (bool, error) {
	pageSize := defaultPageSize
	offset := 0
	now := utils.GetLocalTime()

	for {
		page, err := w.listNormalJobsByStatusPage(ctx, batch.Pending, &accountID, nil, offset, pageSize)
		if err != nil {
			return false, err
		}
		if len(page) == 0 {
			return false, nil
		}

		for _, record := range page {
			if !isTimedOutNormalJob(record, now) {
				continue
			}
			recordResourceDomain := utils.GetJobResourceDomain(record)
			if utils.CanResourceDomainBlock(recordResourceDomain, candidateDomain) {
				return true, nil
			}
		}

		if len(page) < pageSize {
			return false, nil
		}
		offset += len(page)
	}
}

func (w *PrequeueWatcher) deletePreemptedBackfillJobs(
	ctx context.Context,
	plan *preemptionPlan,
) (bool, error) {
	if plan == nil || len(plan.jobs) == 0 {
		return false, nil
	}
	for _, record := range plan.jobs {
		job := &batch.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      record.JobName,
				Namespace: config.GetConfig().Namespaces.Job,
			},
		}
		err := w.k8sClient.Delete(ctx, job)
		if err != nil && !apierrors.IsNotFound(err) {
			return true, err
		}
	}

	return true, nil
}

func (w *PrequeueWatcher) findSingleNodePreemptionPlan(
	ctx context.Context,
	timedOutPendingNormalJob *model.Job,
) (*preemptionPlan, error) {
	jobRequirements, err := getSingleNodeJobRequirements(timedOutPendingNormalJob)
	if err != nil {
		return nil, nil
	}

	nodeList := &v1.NodeList{}
	if err := w.k8sClient.List(ctx, nodeList); err != nil {
		return nil, err
	}

	backfillJobsByNode, err := w.listPreemptableRunningBackfillJobsByNode(ctx)
	if err != nil {
		return nil, err
	}

	var best *preemptionPlan
	for i := range nodeList.Items {
		candidatePlan, err := w.buildNodeBackfillPreemptionPlan(
			ctx,
			&nodeList.Items[i],
			jobRequirements,
			backfillJobsByNode,
		)
		if err != nil {
			return nil, err
		}
		if candidatePlan == nil {
			continue
		}
		if best == nil || len(candidatePlan.jobs) < len(best.jobs) {
			best = candidatePlan
			continue
		}
		if len(candidatePlan.jobs) == len(best.jobs) && candidatePlan.nodeName < best.nodeName {
			best = candidatePlan
		}
	}

	return best, nil
}

// buildNodeBackfillPreemptionPlan finds the smallest backfill set that covers one node's deficit.
func (w *PrequeueWatcher) buildNodeBackfillPreemptionPlan(
	ctx context.Context,
	node *v1.Node,
	jobRequirements *singleNodeJobRequirements,
	backfillJobsByNode map[string][]*model.Job,
) (*preemptionPlan, error) {
	canScheduleOnNode := nodeMatchesPodSchedulingConstraints(node, jobRequirements.podSpec)
	if !canScheduleOnNode {
		return nil, nil
	}

	available, err := w.getNodeAvailableResources(ctx, node)
	if err != nil {
		return nil, err
	}
	deficit := calculateResourceDeficit(jobRequirements.requests, available)
	if len(deficit) == 0 {
		return nil, nil
	}

	// Backfill preemption is intentionally domain-agnostic.
	eligibleBackfillJobs := lo.Filter(backfillJobsByNode[node.Name], func(record *model.Job, _ int) bool {
		return record != nil
	})
	jobsToPreempt := selectMinimalPreemptionSubset(eligibleBackfillJobs, deficit)
	if len(jobsToPreempt) == 0 {
		return nil, nil
	}

	return &preemptionPlan{
		nodeName: node.Name,
		jobs:     jobsToPreempt,
	}, nil
}

func (w *PrequeueWatcher) listPreemptableRunningBackfillJobsByNode(ctx context.Context) (map[string][]*model.Job, error) {
	records := make([]*model.Job, 0)
	err := w.q.Job.WithContext(ctx).UnderlyingDB().
		Model(&model.Job{}).
		Where(
			"status = ? AND schedule_type = ?",
			batch.Running,
			model.ScheduleTypeBackfill,
		).
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	now := utils.GetLocalTime()
	result := make(map[string][]*model.Job)
	for _, record := range records {
		if record == nil || !isPreemptableBackfillJobType(record.JobType) ||
			record.LockedTimestamp.After(now) || !utils.IsSingleNodeJob(record) {
			continue
		}

		nodes, err := w.getAssignedNodes(ctx, record)
		if err != nil {
			return nil, err
		}
		if len(nodes) != 1 {
			continue
		}

		nodeName := nodes[0]
		result[nodeName] = append(result[nodeName], record)
	}

	for nodeName := range result {
		sort.Slice(result[nodeName], func(i, j int) bool {
			left := result[nodeName][i]
			right := result[nodeName][j]
			if !left.CreationTimestamp.Equal(right.CreationTimestamp) {
				return left.CreationTimestamp.Before(right.CreationTimestamp)
			}
			return left.JobName < right.JobName
		})
	}

	return result, nil
}
