package cleaner

import (
	"context"
	"errors"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/config"
)

type CancelWaitingJupyterJobsRequest struct {
	WaitMinitues int `form:"waitMinitues" binding:"required"`
}

func CleanWaitingJupyterJobs(c context.Context, clients *Clients, req *CancelWaitingJupyterJobsRequest) (map[string][]string, error) {
	if req == nil {
		err := errors.New("invalid request")
		return nil, err
	}
	deletedJobs := deleteUnscheduledJupyterJobs(c, clients, req.WaitMinitues)
	ret := map[string][]string{
		"deleted": deletedJobs,
	}
	return ret, nil
}

func deleteUnscheduledJupyterJobs(c context.Context, clients *Clients, waitMinitues int) []string {
	jobDB := query.Job
	jobs, err := jobDB.WithContext(c).Where(
		jobDB.Status.Eq(string(batch.Pending)),
		jobDB.JobType.Eq(string(model.JobTypeJupyter)),
		jobDB.CreationTimestamp.Lt(time.Now().Add(-time.Duration(waitMinitues)*time.Minute)),
	).Find()

	if err != nil {
		klog.Errorf("Failed to get unscheduled jupyter jobs: %v", err)
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

// 如果VCJob还没有创建Pod，返回false
// 所有Pod被schedule，返回true；否则返回false
func isJobscheduled(c context.Context, clients *Clients, jobName string) bool {
	namespace := config.GetConfig().Namespaces.Job
	pods, err := clients.KubeClient.CoreV1().Pods(namespace).List(c, metav1.ListOptions{
		// 目前仅考虑vcjob
		LabelSelector: fmt.Sprintf("volcano.sh/job-name=%s", jobName),
	})
	if err != nil {
		klog.Errorf("Failed to get pods: %v", err)
		return false
	}

	if len(pods.Items) == 0 {
		// 没有Pod，说明VCJob还没有创建Pod
		return false
	}

	result := true
	for i := range pods.Items {
		pod := &pods.Items[i]
		// check pod's conditions: type==PodScheduled && status==True
		thisPodScheduled := false
		for _, condition := range pod.Status.Conditions {
			if condition.Type != "PodScheduled" {
				continue
			}
			if condition.Status == "True" {
				thisPodScheduled = true
				break
			}
			if condition.Status == "False" {
				return false
			}
		}
		result = result && thisPodScheduled
	}
	return result
}
