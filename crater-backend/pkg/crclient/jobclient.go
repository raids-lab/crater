package crclient

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aijobapi "github.com/aisystem/ai-protal/pkg/apis/aijob/v1alpha1"
	"github.com/aisystem/ai-protal/pkg/models"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobControl struct {
	client.Client
}

func (c *JobControl) GetJobStatus(task *models.AITask) (aijobapi.JobPhase, error) {
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

//
func (c *JobControl) CreateJobFromTask(task *models.AITask) error {
	resourceRequest, err := models.JSONToResourceList(task.ResourceRequest)
	if err != nil {
		return fmt.Errorf("resource request is not valid: %v", err)
	}

	// convert metadata to lower case
	taskName := strings.ToLower(task.TaskName)
	jobname := fmt.Sprintf("%s-%d", taskName, task.ID)
	taskID := strconv.Itoa(int(task.ID))

	// set labels and annotations
	labels := map[string]string{
		aijobapi.LabeKeyTaskID:    taskID,
		aijobapi.LabelKeyTaskUser: task.UserName,
		aijobapi.LabelKeyTaskType: task.TaskType,
		aijobapi.LabelKeyTaskSLO:  strconv.FormatUint(uint64(task.SLO), 10),
		aijobapi.JobNameLabel:     jobname,
	}

	annotations := make(map[string]string)
	if task.ProfileStatus == models.ProfileFinish {
		annotations[aijobapi.AnnotationKeyProfileStat] = task.ProfileStat
		annotations[aijobapi.AnnotationKeyPreemptCount] = "0"
	}

	// set priority class
	var priorityClassName string
	if task.SLO == 0 {
		priorityClassName = models.PriorityClassBestEffort
	} else {
		priorityClassName = models.PriorityClassGauranteed
	}

	volumes, volumeMounts, err := GenVolumeAndMountsFromAITask(task)
	if err != nil {
		return fmt.Errorf("gen volumes and mounts failed: %v", err)
	}
	job := &aijobapi.AIJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobname,
			Namespace:   task.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: aijobapi.JobSpec{
			Replicas: 1,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        jobname,
					Labels:      labels,
					Namespace:   task.Namespace,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					PriorityClassName: priorityClassName,
					RestartPolicy:     corev1.RestartPolicyNever,
					NodeSelector: map[string]string{
						"v100": "true", // todo: for test
					},
					Containers: []corev1.Container{
						{
							Image:   task.Image,
							Name:    "main",
							Command: []string{"/bin/bash", "-c", task.Command}, // todo:
							// Args:    models.JSONStringToList(task.Args),        // todo:
							Resources: corev1.ResourceRequirements{
								Limits:   resourceRequest,
								Requests: resourceRequest,
							},
							WorkingDir:   task.WorkingDir,
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
			ResourceRequest: resourceRequest,
		},
	}
	err = c.Create(context.Background(), job)
	if err != nil {
		return fmt.Errorf("create job %s failed: %v", task.TaskName, err)
	}
	return nil
}

func GenVolumeAndMountsFromAITask(task *models.AITask) ([]corev1.Volume, []corev1.VolumeMount, error) {

	// set volumes
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "user-volume",
			MountPath: "/home/" + task.UserName,
		},
		{
			Name:      "cache-volume",
			MountPath: "/dev/shm",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "user-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: fmt.Sprintf(UserHomePVC, task.UserName),
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
	}
	if task.ShareDirs != "" {
		taskShareDir := models.JSONStringToVolumes(task.ShareDirs)
		if taskShareDir == nil {
			logrus.Errorf("parse task share dir failed： %v", task.ShareDirs)
			return nil, nil, fmt.Errorf("parse task share dir failed： %v", task.ShareDirs)
		}
		for pvc, mounts := range taskShareDir {

			volumes = append(volumes, corev1.Volume{
				Name: pvc,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvc,
					},
				},
			})
			for _, mount := range mounts {
				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      pvc,
					MountPath: mount.MountPath,
					SubPath:   mount.SubPath,
				})
			}
		}
	}
	return volumes, volumeMounts, nil
}
