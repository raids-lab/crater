package service

import (
	"context"
	"slices"
	"strconv"

	"gorm.io/datatypes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/pkg/utils"
)

// Include accepted Volcano Jobs before the asynchronous DB observer sees them.
// The uncached API reader makes a successful submission an immediate quota
// reservation; existing DB records are retained to cover deletion/observer lag.
func (s *QueueQuotaService) liveJobReservations(
	ctx context.Context,
	userID, accountID uint,
	records []*model.Job,
	includeWaiting bool,
) ([]*model.Job, error) {
	if s.servingClient == nil {
		return records, nil
	}
	var list batch.JobList
	if err := s.servingClient.List(ctx, &list, client.InNamespace(s.servingNamespace)); err != nil {
		return nil, err
	}
	groups := &scheduling.PodGroupList{}
	if !includeWaiting {
		if err := s.servingClient.List(ctx, groups, client.InNamespace(s.servingNamespace)); err != nil {
			return nil, err
		}
	}
	phases := livePodGroupPhases(groups)
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
		phase := phases[job.Name+"/"+string(job.UID)]
		if !includeWaiting && !slices.Contains(QueueQuotaOccupiedJobPhases(), string(job.Status.State.Phase)) &&
			!utils.IsJobInqueue(job.Status.State.Phase, phase) {
			continue
		}
		if utils.IsJobPhaseTerminal(job.Status.State.Phase) {
			continue
		}
		records = append(
			records,
			&model.Job{
				UserID:        userID,
				AccountID:     accountID,
				JobName:       job.Name,
				Queue:         job.Spec.Queue,
				Status:        job.Status.State.Phase,
				PodGroupPhase: phase,
				Resources:     datatypes.NewJSONType(vcjobservice.CalculateJobResources(job)),
			},
		)
	}
	return records, nil
}

func livePodGroupPhases(groups *scheduling.PodGroupList) map[string]scheduling.PodGroupPhase {
	phases := make(map[string]scheduling.PodGroupPhase, len(groups.Items))
	for i := range groups.Items {
		for _, owner := range groups.Items[i].OwnerReferences {
			if owner.Controller != nil && *owner.Controller && owner.UID != "" && owner.Kind == "Job" &&
				owner.APIVersion == batch.SchemeGroupVersion.String() {
				phases[owner.Name+"/"+string(owner.UID)] = groups.Items[i].Status.Phase
			}
		}
	}
	return phases
}
