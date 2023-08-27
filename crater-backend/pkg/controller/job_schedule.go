package controller

import (
	aijobapi "github.com/aisys/ai-task-controller/pkg/apis/aijob/v1alpha1"
)

func (jc *JobController) addJobToQueue(jobKey string, aijob *aijobapi.AIJob) {
	jc.jobPendingQueue.Store(jobKey, aijob)
}

func (jc *JobController) removeJobFromQueue(jobKey string) {
	jc.jobPendingQueue.Delete(jobKey)
}
