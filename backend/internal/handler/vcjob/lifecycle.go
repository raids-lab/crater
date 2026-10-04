package vcjob

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/utils"
	vcjobadmission "github.com/raids-lab/crater/pkg/vcjob/admission"
)

// submitJob hands the job straight to volcano. Queue quota and timeout blocking are decided by the
// extender endpoint during volcano's scheduling session, not here.
func (mgr *VolcanojobMgr) submitJob(
	ctx context.Context,
	token util.JWTMessage,
	job *batch.Job,
) error {
	return service.WithWorkloadAdmission(ctx, token.UserID, token.AccountID, func() error {
		if mgr.queueQuotaSvc != nil {
			if err := service.CheckWorkloadSubmission(ctx, mgr.userBanService, mgr.billingService,
				mgr.queueQuotaSvc, token.UserID, token.AccountID); err != nil {
				return err
			}
		}
		return mgr.submitJobLocked(ctx, token, job)
	})
}

func (mgr *VolcanojobMgr) submitJobLocked(ctx context.Context, token util.JWTMessage, job *batch.Job) error {
	if err := mgr.ensureJobAdmitted(ctx, job); err != nil {
		return err
	}
	if err := mgr.checkRequestedResourceLimit(ctx, token, job); err != nil {
		return err
	}
	return mgr.activateJob(ctx, job)
}

func (mgr *VolcanojobMgr) activateJob(ctx context.Context, job *batch.Job) error {
	if err := service.ApplyJobPodBandwidth(ctx, mgr.configService, mgr.kubeClient, job); err != nil {
		return err
	}
	return vcjobservice.ActivateJob(ctx, mgr.client, mgr.serviceManager, job)
}

func (mgr *VolcanojobMgr) ensureJobAdmitted(ctx context.Context, job *batch.Job) error {
	admission, err := vcjobadmission.CheckJobAdmission(ctx, mgr.client, job)
	if err != nil {
		return err
	}
	if !admission.Accepted {
		return bizerr.Conflict.JobAdmissionFailed.New(admission.Reason)
	}
	return nil
}

// checkRequestedResourceLimit rejects a job whose own request already exceeds the user's queue quota:
// no amount of waiting would ever let it run, so queuing it would only be misleading.
func (mgr *VolcanojobMgr) checkRequestedResourceLimit(
	ctx context.Context,
	token util.JWTMessage,
	job *batch.Job,
) error {
	if mgr.queueQuotaSvc == nil {
		return nil
	}

	check, err := mgr.queueQuotaSvc.CheckRequestedResourceLimit(
		ctx,
		token.UserID,
		token.AccountID,
		job.Spec.Queue,
		utils.ToStringMap(vcjobservice.CalculateJobResources(job)),
	)
	if err != nil {
		return err
	}
	if check.Enabled && check.Exceeded {
		return bizerr.BadRequest.ParameterError.New(
			"requested resources exceed user queue quota: " +
				formatExceededResourceLimitDetails(check.Details),
		)
	}
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

// handleJobSubmissionError preserves business codes and classifies other submission errors.
func handleJobSubmissionError(c *gin.Context, err error) {
	if _, ok := bizerr.FromError(err); ok {
		resputil.HandleError(c, err)
		return
	}
	resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(err, err.Error()))
}
