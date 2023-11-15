package crclient

import (
	"context"
	"fmt"
	"strconv"

	aijobapi "github.com/aisystem/ai-protal/pkg/apis/aijob/v1alpha1"
	"github.com/aisystem/ai-protal/pkg/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodControl struct {
	client.Client
}

func (c *PodControl) GetJobStatus(task *models.AITask) (aijobapi.JobPhase, error) {
	jobname := fmt.Sprintf("%s-%d", task.TaskName, task.ID)
	ns := task.Namespace
	job := &aijobapi.AIJob{}
	err := c.Get(context.Background(), client.ObjectKey{
		Namespace: ns,
		Name:      jobname,
	}, job)
	if err != nil {
		return "", err
	}
	return job.Status.Phase, nil
}

func (c *PodControl) DeleteProfilePodFromTask(task *models.AITask) error {
	podName := fmt.Sprintf("%s-%d", task.TaskName, task.ID)
	ns := task.Namespace
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
	}
	err := c.Delete(context.Background(), pod)
	return err
}

//
func (c *PodControl) CreateProfilePodFromTask(task *models.TaskAttr) error {
	args := []string{}
	for k, v := range task.Args {
		args = append(args, k, v)
	}
	pvcname := fmt.Sprintf(UserHomePVC, task.UserName)
	jobname := fmt.Sprintf("%s-%d", task.TaskName, task.ID)
	taskID := strconv.Itoa(int(task.ID))
	labels := map[string]string{
		aijobapi.TaskIDKey: taskID,
		// todo: add controller ref?

	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobname,
			Namespace: task.Namespace,
			Labels:    labels,
		},

		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			NodeSelector: map[string]string{
				"v100": "true", // todo: for test
			},
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
					WorkingDir: task.WorkingDir,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "user-volume",
							MountPath: "/home/" + task.UserName,
						},
						{
							Name:      "cache-volume",
							MountPath: "/dev/shm",
						},
						{
							Name:      "data-volume",
							MountPath: "/data",
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
				{
					Name: "data-volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: DataPVCName,
						},
					},
				},
			},
		},
	}
	err := c.Create(context.Background(), pod)
	if err != nil {
		return fmt.Errorf("create pod %s failed: %v", task.TaskName, err)
	}
	return nil
}
