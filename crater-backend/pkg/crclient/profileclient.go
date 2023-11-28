package crclient

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	aijobapi "github.com/aisystem/ai-protal/pkg/apis/aijob/v1alpha1"
	"github.com/aisystem/ai-protal/pkg/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ProfilingPodLabels = map[string]string{
		"aisystem.github.com/profile": "true",
	}
)

type ProfilingPodControl struct {
	client.Client
}

func (c *ProfilingPodControl) ListProflingPods() ([]corev1.Pod, error) {
	pods := &corev1.PodList{}
	err := c.List(context.Background(), pods, client.MatchingLabels(ProfilingPodLabels))
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func (c *ProfilingPodControl) DeleteProfilePodFromTask(task *models.AITask) error {
	podName := fmt.Sprintf("%s-%d-profiling", task.TaskName, task.ID)
	podName = strings.ToLower(podName)
	podName = strings.Replace(podName, "_", "-", -1)
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

func (c *ProfilingPodControl) GetTaskIDFromPod(pod *corev1.Pod) (uint, error) {
	id, ok := pod.Labels[aijobapi.LabeKeyTaskID]
	if !ok {
		return 0, fmt.Errorf("taskID not found in pod: %v/%v", pod.Namespace, pod.Name)
	}
	taskID, _ := strconv.Atoi(id)
	return uint(taskID), nil
}

//
func (c *ProfilingPodControl) CreateProfilePodFromTask(task *models.AITask) error {

	resourceRequest, err := models.JSONToResourceList(task.ResourceRequest)
	if err != nil {
		return fmt.Errorf("resource request is not valid: %v", err)
	}
	podName := fmt.Sprintf("%s-%d-profiling", task.TaskName, task.ID)
	podName = strings.ToLower(podName)
	podName = strings.Replace(podName, "_", "-", -1)
	taskID := strconv.Itoa(int(task.ID))
	labels := map[string]string{
		aijobapi.LabeKeyTaskID: taskID,
	}
	for k, v := range ProfilingPodLabels {
		labels[k] = v
	}

	volumes, volumeMounts, err := GenVolumeAndMountsFromAITask(task)
	if err != nil {
		return fmt.Errorf("gen volumes and mounts failed: %v", err)
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: task.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"profiling-pod": "true",
			},
		},

		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			NodeSelector: map[string]string{
				"profile": "true", // todo: for test
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
			// add toleration that can be scheduled to profile node
			Tolerations: []corev1.Toleration{
				{
					Key:      "profile",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
		},
	}
	err = c.Create(context.Background(), pod)
	if err != nil {
		return fmt.Errorf("create pod %s failed: %v", task.TaskName, err)
	}
	return nil
}
