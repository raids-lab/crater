package controller

import (
	aijobapi "k8s.io/ai-task-controller/pkg/apis/aijob/v1alpha1"
)

func (jc *JobController) addJobToQueue(jobKey string, aijob *aijobapi.AIJob) {
	jc.jobPendingQueue.Store(jobKey, aijob)
}

func (jc *JobController) removeJobFromQueue(jobKey string) {
	jc.jobPendingQueue.Delete(jobKey)
}
