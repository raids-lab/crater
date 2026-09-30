package vcjob

import (
	"context"
	"fmt"
	"strings"

	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/internal/bizerr"
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
		return fmt.Errorf("job admission failed: %s", admission.Reason)
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
