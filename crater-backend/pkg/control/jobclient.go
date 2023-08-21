package control

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aijobapi "k8s.io/ai-task-controller/pkg/apis/aijob/v1alpha1"
	"k8s.io/ai-task-controller/pkg/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobControl struct {
	client.Client
}

// todo: add more volumes, args etc..
func (c *JobControl) CreateJobFromTask(task models.TaskAttr) error {
	args := []string{}
	for k, v := range task.Args {
		args = append(args, k, v)
	}
	job := &aijobapi.AIJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      task.TaskName,
			Namespace: task.Namespace,
		},
		Spec: aijobapi.JobSpec{
			Replicas: 1,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image:   task.Image,
							Name:    "main",
							Command: []string{task.Command},
							Args:    args,
							Resources: corev1.ResourceRequirements{
								Limits:   task.ResourceRequest.DeepCopy(),
								Requests: task.ResourceRequest.DeepCopy(),
							},
						},
					},
					Volumes: []corev1.Volume{},
				},
			},
			ResourceRequest: task.ResourceRequest.DeepCopy(),
		},
		Status: aijobapi.JobStatus{
			Phase: aijobapi.Pending,
		},
	}
	err := c.Create(context.Background(), job)
	if err != nil {
		return fmt.Errorf("create job %s failed: %v", task.TaskName, err)
	}
	return nil
}
