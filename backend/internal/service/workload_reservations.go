package service

import (
	"context"
	"strconv"

	"github.com/raids-lab/crater/dao/model"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"gorm.io/datatypes"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

// Include accepted Volcano Jobs before the asynchronous DB observer sees them.
// The uncached API reader makes a successful submission an immediate quota
// reservation; existing DB records are retained to cover deletion/observer lag.
func (s *PrequeueService) liveJobReservations(
	ctx context.Context,
	userID, accountID uint,
	records []*model.Job,
	allSchedules bool,
) ([]*model.Job, error) {
	if s.servingClient == nil {
		return records, nil
	}
	var list batch.JobList
	if err := s.servingClient.List(ctx, &list, client.InNamespace(s.servingNamespace)); err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, r := range records {
		known[r.JobName] = true
	}
	for i := range list.Items {
		job := &list.Items[i]
		if known[job.Name] || job.Annotations["crater.raids.io/user-id"] != strconv.FormatUint(uint64(userID), 10) ||
			job.Annotations["crater.raids.io/account-id"] != strconv.FormatUint(uint64(accountID), 10) {
			continue
		}
		schedule, err := model.ParseScheduleType(job.Annotations[vcjobservice.AnnotationKeyScheduleType])
		if err != nil {
			return nil, err
		}
		if !allSchedules && schedule != model.ScheduleTypeNormal {
			continue
		}
		if job.Status.State.Phase == batch.Completed || job.Status.State.Phase == batch.Failed ||
			job.Status.State.Phase == batch.Aborted {
			continue
		}
		records = append(
			records,
			&model.Job{
				UserID:       userID,
				AccountID:    accountID,
				JobName:      job.Name,
				Queue:        job.Spec.Queue,
				Status:       batch.Pending,
				ScheduleType: ptr.To(schedule),
				Resources:    datatypes.NewJSONType(vcjobservice.CalculateJobResources(job)),
			},
		)
	}
	return records, nil
}
