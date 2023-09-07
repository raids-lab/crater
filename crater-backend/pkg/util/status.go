package util

import (
	aijobapi "github.com/aisystem/ai-protal/pkg/apis/aijob/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// JobCreatedReason is added in a job when it is created.
	JobCreatedReason = "JobCreated"
	// JobSucceededReason is added in a job when it is succeeded.
	JobSucceededReason = "JobSucceeded"
	// JobRunningReason is added in a job when it is running.
	JobRunningReason = "JobRunning"
	// JobFailedReason is added in a job when it is failed.
	JobFailedReason = "JobFailed"
	// JobSuspended is added in a job when it is restarting.
	JobRestartingReason = "JobSuspended"
	// JobFailedValidationReason is added in a job when it failed validation
	JobFailedValidationReason = "JobFailedValidation"

	// labels for pods and servers.

)

// IsSucceeded checks if the job is succeeded
func IsSucceeded(status aijobapi.JobStatus) bool {
	return hasCondition(status, aijobapi.JobSucceeded)
}

// IsFailed checks if the job is failed
func IsFailed(status aijobapi.JobStatus) bool {
	return hasCondition(status, aijobapi.JobFailed)
}

// UpdateJobConditions adds to the jobStatus a new condition if needed, with the conditionType, reason, and message
func UpdateJobConditionsAndStatus(jobStatus *aijobapi.JobStatus, phase aijobapi.JobPhase, conditionType aijobapi.JobConditionType, reason, message string) error {
	condition := newCondition(conditionType, reason, message)
	setCondition(jobStatus, condition)
	jobStatus.Phase = phase
	return nil
}

func hasCondition(status aijobapi.JobStatus, condType aijobapi.JobConditionType) bool {
	for _, condition := range status.Conditions {
		if condition.Type == condType && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

// newCondition creates a new job condition.
func newCondition(conditionType aijobapi.JobConditionType, reason, message string) aijobapi.JobCondition {
	return aijobapi.JobCondition{
		Type:               conditionType,
		Status:             v1.ConditionTrue,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// getCondition returns the condition with the provided type.
func getCondition(status aijobapi.JobStatus, condType aijobapi.JobConditionType) *aijobapi.JobCondition {
	for _, condition := range status.Conditions {
		if condition.Type == condType {
			return &condition
		}
	}
	return nil
}

// setCondition updates the job to include the provided condition.
// If the condition that we are about to add already exists
// and has the same status and reason then we are not going to update.
func setCondition(status *aijobapi.JobStatus, condition aijobapi.JobCondition) {
	// Do nothing if JobStatus have failed condition
	if IsFailed(*status) {
		return
	}

	currentCond := getCondition(*status, condition.Type)

	// Do nothing if condition doesn't change
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}

	// Append the updated condition to the conditions
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of job conditions without conditions with the provided type.
func filterOutCondition(conditions []aijobapi.JobCondition, condType aijobapi.JobConditionType) []aijobapi.JobCondition {
	var newConditions []aijobapi.JobCondition
	for _, c := range conditions {
		// if condType == aijobapi.JobRestarting && c.Type == aijobapi.JobRunning{
		// 	continue
		// }
		// if condType == aijobapi.JobRunning && c.Type == aijobapi.JobRestarting {
		// 	continue
		// }

		if c.Type == condType {
			continue
		}

		// Set the running condition status to be false when current condition failed or succeeded
		if (condType == aijobapi.JobFailed || condType == aijobapi.JobSucceeded) && c.Type == aijobapi.JobRunning {
			c.Status = v1.ConditionFalse
		}

		newConditions = append(newConditions, c)
	}
	return newConditions
}

func IsCompletedStatus(status string) bool {
	if status == string(aijobapi.JobFailed) || status == string(aijobapi.JobSucceeded) {
		return true
	}
	return false
}
