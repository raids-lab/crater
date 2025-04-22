package aijob

import (
	"github.com/raids-lab/crater/dao/model"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

func convertJobPhase(aijob *model.AITask) batch.JobPhase {
	if aijob.IsDeleted {
		return model.Deleted
	}
	aijobStatus := aijob.Status
	switch aijobStatus {
	case "Pending":
		return batch.Pending
	case "Running":
		return batch.Running
	case "Succeeded":
		return batch.Completed
	case "Failed":
		return batch.Failed
	case "Preempted":
		return batch.Aborted
	default:
		return batch.Pending
	}
}
