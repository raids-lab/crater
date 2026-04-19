package vcjob

import (
	"context"
	"fmt"
	"strconv"

	"k8s.io/klog/v2"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/utils"
)

func (mgr *VolcanojobMgr) submitJob(
	ctx context.Context,
	token util.JWTMessage,
	job *batch.Job,
) error {
	if mgr.billingService != nil {
		if err := mgr.billingService.OnJobCreateCheck(ctx, token.UserID, token.AccountID); err != nil {
			return err
		}
	}

	scheduleTypeInt, err := strconv.ParseInt(
		job.Annotations[vcjobservice.AnnotationKeyScheduleType], 10, 64,
	)
	if err != nil {
		return fmt.Errorf("invalid schedule type annotation value: %w", err)
	}
	scheduleType := model.ScheduleType(scheduleTypeInt)

	if mgr.prequeueWatcher == nil {
		return vcjobservice.ActivateJob(ctx, mgr.client, mgr.serviceManager, job)
	}

	jobResources := vcjobservice.CalculateJobResources(job)
	jobResourceStringMap := utils.ToStringMap(jobResources)
	quotaExceeded := false
	if scheduleType == model.ScheduleTypeNormal && mgr.queueQuotaSvc != nil {
		limitCheck, err := mgr.queueQuotaSvc.CheckUserResourceLimit(
			ctx,
			token.UserID,
			token.AccountID,
			job.Spec.Queue,
			jobResourceStringMap,
		)
		if err != nil {
			return err
		}
		if limitCheck.Enabled && limitCheck.Exceeded {
			quotaExceeded = true
		}
	}
	hasTimedOutPendingJob := false
	hasTimedOutPendingJob, err = mgr.prequeueWatcher.HasBlockingTimedOutPendingNormalJob(
		ctx,
		token.AccountID,
		jobResources,
	)
	if err != nil {
		klog.Errorf("failed to check timed out pending normal job for account %v: %v", token.AccountID, err)
		return err
	}
	if shouldPrequeueSubmittedJob(scheduleType, quotaExceeded, hasTimedOutPendingJob) {
		record, err := vcjobservice.GenerateJobRecord(job, token.UserID, token.AccountID, model.Prequeue)
		if err != nil {
			return err
		}
		if err := query.Job.WithContext(ctx).Create(record); err != nil {
			return fmt.Errorf("failed to create prequeue job record: %w", err)
		}
		mgr.prequeueWatcher.RequestFullScan()
		return nil
	}

	if err := vcjobservice.ActivateJob(ctx, mgr.client, mgr.serviceManager, job); err != nil {
		return err
	}
	return nil
}

func shouldPrequeueSubmittedJob(
	scheduleType model.ScheduleType,
	quotaExceeded bool,
	hasTimedOutPendingNormalJob bool,
) bool {
	// 队列内每个用户的资源配额
	if scheduleType == model.ScheduleTypeNormal && quotaExceeded {
		return true
	}
	// volcano队列中存在等待时间超过阈值的pending normal job
	if hasTimedOutPendingNormalJob {
		return scheduleType == model.ScheduleTypeNormal || scheduleType == model.ScheduleTypeBackfill
	}

	return false
}
