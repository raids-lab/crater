package crclient

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aijobapi "github.com/aisystem/ai-protal/pkg/apis/aijob/v1alpha1"
	"github.com/aisystem/ai-protal/pkg/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobControl struct {
	client.Client
}

func (c *JobControl) DeleteJobFromTask(task *models.AITask) error {
	jobname := fmt.Sprintf("%s-%d", task.TaskName, task.ID)
	ns := task.Namespace
	job := &aijobapi.AIJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobname,
			Namespace: ns,
		},
	}
	err := c.Delete(context.Background(), job)
	return err
}

// todo: add more volumes, args etc..
func (c *JobControl) CreateJobFromTask(task *models.TaskAttr) error {
	args := []string{}
	for k, v := range task.Args {
		args = append(args, k, v)
	}
	pvcname := fmt.Sprintf(PVCFormat, task.UserName)
	jobname := fmt.Sprintf("%s-%d", task.TaskName, task.ID)
	job := &aijobapi.AIJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobname,
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
							Command: []string{"/bin/bash", "-c", task.Command}, // todo:
							Args:    args,                                      // todo:
							Resources: corev1.ResourceRequirements{
								Limits:   task.ResourceRequest.DeepCopy(),
								Requests: task.ResourceRequest.DeepCopy(),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "user-volume",
									MountPath: "/home/" + task.UserName,
								},
								{
									Name:      "cache-volume",
									MountPath: "/dev/shm",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "user-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcname,
								},
							},
						},
						{
							Name: "cache-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									Medium: corev1.StorageMediumMemory,
								},
							},
						},
					},
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
