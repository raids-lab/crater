package crclient

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	recommenddljobapi "github.com/raids-lab/crater/pkg/apis/recommenddljob/v1"
)

type RecommendDLJobController struct {
	client.Client
}

// todo: add more volumes, args etc..
func (c *RecommendDLJobController) CreateRecommendDLJob(ctx context.Context, job *recommenddljobapi.RecommendDLJob) error {
	err := c.Create(ctx, job)
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

func (c *RecommendDLJobController) GetRecommendDLJob(ctx context.Context, name, namespace string) (
	*recommenddljobapi.RecommendDLJob, error) {
	job := &recommenddljobapi.RecommendDLJob{}
	if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (c *RecommendDLJobController) ListRecommendDLJob(ctx context.Context, namespace string) ([]*recommenddljobapi.RecommendDLJob, error) {
	jobList := &recommenddljobapi.RecommendDLJobList{}
	if err := c.List(ctx, jobList, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}
	ret := make([]*recommenddljobapi.RecommendDLJob, 0, len(jobList.Items))
	for i := range jobList.Items {
		ret = append(ret, &jobList.Items[i])
	}
	return ret, nil
}

func (c *RecommendDLJobController) GetRecommendDLJobPodList(ctx context.Context, name, namespace string) ([]*v1.Pod, error) {
	job, err := c.GetRecommendDLJob(ctx, name, namespace)
	if err != nil {
		return nil, err
	}
	podList := make([]*v1.Pod, 0, len(job.Status.PodNames))
	for _, podName := range job.Status.PodNames {
		pod := &v1.Pod{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: podName}, pod); err != nil {
			return nil, err
		}
		podList = append(podList, pod)
	}
	return podList, nil
}

func (c *RecommendDLJobController) DeleteRecommendDLJob(ctx context.Context, name, namespace string) error {
	job, err := c.GetRecommendDLJob(ctx, name, namespace)
	if err != nil {
		return err
	}
	err = c.Delete(ctx, job)
	if err != nil {
		return err
	}
	return nil
}
