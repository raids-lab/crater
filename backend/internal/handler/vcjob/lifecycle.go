package vcjob

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/utils"
	vcjobadmission "github.com/raids-lab/crater/pkg/vcjob/admission"
)

func (mgr *VolcanojobMgr) submitJob(
	ctx context.Context,
	token util.JWTMessage,
	job *batch.Job,
) error {
	scheduleType, err := parseSubmissionScheduleType(job)
	if err != nil {
		return err
	}

	jobResources := vcjobservice.CalculateJobResources(job)
	jobResourceStringMap := utils.ToStringMap(jobResources)

	if err := mgr.ensureJobAdmitted(ctx, job); err != nil {
		return err
	}

	if mgr.prequeueWatcher == nil {
		return vcjobservice.ActivateJob(ctx, mgr.client, mgr.serviceManager, job)
	}

	quotaExceeded, err := mgr.checkSubmissionQuota(ctx, token, job, scheduleType, jobResourceStringMap)
	if err != nil {
		return err
	}

	hasTimedOutPendingJob, err := mgr.hasBlockingTimedOutPendingNormalJob(ctx, token, jobResources)
	if err != nil {
		return err
	}

	if shouldPrequeueSubmittedJob(scheduleType, quotaExceeded, hasTimedOutPendingJob) {
		return mgr.createPrequeueRecord(ctx, token, job)
	}

	if err := vcjobservice.ActivateJob(ctx, mgr.client, mgr.serviceManager, job); err != nil {
		return err
	}
	return nil
}

func parseSubmissionScheduleType(job *batch.Job) (model.ScheduleType, error) {
	if job == nil {
		return 0, fmt.Errorf("invalid job: nil")
	}
	scheduleTypeInt, err := strconv.ParseInt(
		job.Annotations[vcjobservice.AnnotationKeyScheduleType], 10, 64,
	)
	if err != nil {
		return 0, fmt.Errorf("invalid schedule type annotation value: %w", err)
	}
	return model.ScheduleType(scheduleTypeInt), nil
}

func (mgr *VolcanojobMgr) ensureJobAdmitted(ctx context.Context, job *batch.Job) error {
	admission, err := vcjobadmission.CheckJobAdmission(ctx, mgr.client, job)
	if err != nil {
		return err
	}
	if !admission.Accepted {
		return fmt.Errorf("job admission failed: %s", admission.Reason)
	}
	return nil
}

func (mgr *VolcanojobMgr) checkSubmissionQuota(
	ctx context.Context,
	token util.JWTMessage,
	job *batch.Job,
	scheduleType model.ScheduleType,
	jobResourceStringMap map[string]string,
) (bool, error) {
	if scheduleType != model.ScheduleTypeNormal || mgr.queueQuotaSvc == nil {
		return false, nil
	}

	requestLimitCheck, err := mgr.queueQuotaSvc.CheckRequestedResourceLimit(
		ctx,
		token.UserID,
		token.AccountID,
		job.Spec.Queue,
		jobResourceStringMap,
	)
	if err != nil {
		return false, err
	}
	if requestLimitCheck.Enabled && requestLimitCheck.Exceeded {
		return false, fmt.Errorf("requested resources exceed user queue quota: %s", formatExceededResourceLimitDetails(requestLimitCheck.Details))
	}

	limitCheck, err := mgr.queueQuotaSvc.CheckUserResourceLimit(
		ctx,
		token.UserID,
		token.AccountID,
		job.Spec.Queue,
		jobResourceStringMap,
	)
	if err != nil {
		return false, err
	}
	return limitCheck.Enabled && limitCheck.Exceeded, nil
}

func (mgr *VolcanojobMgr) hasBlockingTimedOutPendingNormalJob(
	ctx context.Context,
	token util.JWTMessage,
	jobResources v1.ResourceList,
) (bool, error) {
	if mgr.prequeueWatcher == nil {
		return false, nil
	}
	hasTimedOutPendingJob, err := mgr.prequeueWatcher.HasBlockingTimedOutPendingNormalJob(
		ctx,
		token.AccountID,
		jobResources,
	)
	if err != nil {
		klog.Errorf("failed to check timed out pending normal job for account %v: %v", token.AccountID, err)
		return false, err
	}
	return hasTimedOutPendingJob, nil
}

func (mgr *VolcanojobMgr) createPrequeueRecord(
	ctx context.Context,
	token util.JWTMessage,
	job *batch.Job,
) error {
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

func formatExceededResourceLimitDetails(details []service.ResourceLimitDetail) string {
	parts := make([]string, 0, len(details))
	for _, detail := range details {
		if !detail.Exceeded {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s requested %s exceeds limit %s", detail.Resource, detail.Used, detail.Limit))
	}
	if len(parts) == 0 {
		return "requested resources exceed user queue quota"
	}
	return strings.Join(parts, "; ")
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
