package cleaner

import (
	"context"
	"errors"
	"time"

	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/samber/lo"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/config"
)

type CancelWaitingJobsRequest struct {
	WaitMinitues int             `json:"waitMinitues" form:"waitMinitues" binding:"required"`
	JobTypes     []model.JobType `json:"jobTypes" form:"jobTypes"`
}

func CleanWaitingJobs(c context.Context, clients *Clients, req *CancelWaitingJobsRequest) (map[string][]string, error) {
	if req == nil {
		err := errors.New("invalid request")
		return nil, err
	}
	if len(req.JobTypes) == 0 {
		req.JobTypes = []model.JobType{model.JobTypeJupyter} // 向后兼容
	}
	if req.WaitMinitues <= 0 {
		err := errors.New("waitMinitues must be greater than 0")
		return nil, err
	}
	deletedJobs := deleteUnscheduledJobs(c, clients, req.WaitMinitues, req.JobTypes)
	ret := map[string][]string{
		"deleted": deletedJobs,
	}
	return ret, nil
}

func deleteUnscheduledJobs(c context.Context, clients *Clients, waitMinitues int, jobTypes []model.JobType) []string {
	jobTypeStrs := lo.Map(jobTypes, func(jobType model.JobType, _ int) string {
		return string(jobType)
	})
	jobDB := query.Job
	jobs, err := jobDB.WithContext(c).Where(
		jobDB.Status.Eq(string(batch.Pending)),
		jobDB.JobType.In(jobTypeStrs...),
		jobDB.CreationTimestamp.Lt(time.Now().Add(-time.Duration(waitMinitues)*time.Minute)),
	).Find()

	if err != nil {
		klog.Errorf("Failed to get unscheduled jobs: %v", err)
		return nil
	}

	deletedJobs := []string{}
	for _, job := range jobs {
		if isJobscheduled(c, clients, job.JobName) {
			continue
		}

		// delete job
		vcjob := &batch.Job{}
		namespace := config.GetConfig().Namespaces.Job
		if err := clients.Client.Get(c, client.ObjectKey{Name: job.JobName, Namespace: namespace}, vcjob); err != nil {
			klog.Errorf("Failed to get job %s: %v", job.JobName, err)
			continue
		}

		if err := clients.Client.Delete(c, vcjob); err != nil {
			klog.Errorf("Failed to delete job %s: %v", job.JobName, err)
			continue
		}

		deletedJobs = append(deletedJobs, job.JobName)
	}

	return deletedJobs
}
