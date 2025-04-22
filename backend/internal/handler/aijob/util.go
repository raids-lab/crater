package aijob

import batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

func convertJobPhase(aijobStatus string) batch.JobPhase {
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
