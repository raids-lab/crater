package service

import (
	"context"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
)

// CheckWorkloadSubmission applies identity, billing and workload-count policy
// before either a Volcano job or a ModelServing is submitted.
func CheckWorkloadSubmission(ctx context.Context,
	bans *UserBanService,
	billing *BillingService,
	quota *PrequeueService,
	userID,
	accountID uint,
	schedule model.ScheduleType) error {
	if bans != nil {
		if err := bans.RequireCapability(ctx, userID, UserBanCapabilityJobSubmission); err != nil {
			return err
		}
	}
	if billing != nil {
		if err := billing.OnJobCreateCheck(ctx, userID, accountID, &schedule); err != nil {
			return bizerr.Conflict.ResourceStatusError.Wrap(err, "insufficient balance or unavailable billing account")
		}
	}
	var jobs []*model.Job
	if err := query.GetDB().WithContext(ctx).Model(&model.Job{}).
		Where("user_id = ? AND account_id = ? AND status IN ?",
			userID,
			accountID,
			[]string{"Running",
				"Pending"}).Find(&jobs).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "read workload submission limits failed")
	}
	count := int64(len(jobs))
	if quota != nil {
		allJobs, err := quota.liveJobReservations(ctx, userID, accountID, jobs, true)
		if err != nil {
			return bizerr.Internal.ServiceError.Wrap(err, "read pending workload submissions failed")
		}
		count = int64(len(allJobs))
		usages, err := quota.servingUsage(ctx, userID, accountID)
		if err != nil {
			return bizerr.Internal.ServiceError.Wrap(err, "read serving submission limits failed")
		}
		excluded, _ := ctx.Value(servingExclusionKey{}).(string)
		for _, usage := range usages {
			if excluded == "" || usage.JobName != excluded {
				count++
			}
		}
	}
	const maxActiveWorkloads = 100
	if count >= maxActiveWorkloads {
		return bizerr.Conflict.ResourceStatusError.New("workload submission limit reached")
	}

	return nil
}
