package crclient

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	aijobapi "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
	"github.com/raids-lab/crater/pkg/models"
	v1 "k8s.io/api/core/v1"
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

func (c *ProfilingPodControl) ListProflingPods() ([]v1.Pod, error) {
	pods := &v1.PodList{}
	err := c.List(context.Background(), pods, client.MatchingLabels(ProfilingPodLabels))
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func (c *ProfilingPodControl) DeleteProfilePodFromTask(task *models.AITask) error {
	podName := fmt.Sprintf("%s-%d-profiling", task.TaskName, task.ID)
	podName = strings.ToLower(podName)
	podName = strings.ReplaceAll(podName, "_", "-")
	ns := task.Namespace
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
	}
	err := c.Delete(context.Background(), pod)
	return err
}

func (c *ProfilingPodControl) GetTaskIDFromPod(pod *v1.Pod) (uint, error) {
	id, ok := pod.Labels[aijobapi.LabelKeyTaskID]
	if !ok {
		return 0, fmt.Errorf("taskID not found in pod: %v/%v", pod.Namespace, pod.Name)
	}
	taskID, _ := strconv.Atoi(id)
	return uint(taskID), nil
}

func (c *ProfilingPodControl) CreateProfilePodFromTask(ctx context.Context, task *models.AITask) error {
	podSpec := task.PodTemplate.Data()
	if podSpec.Containers == nil || len(podSpec.Containers) == 0 {
		return fmt.Errorf("no container in pod spec")
	}

	// 1. rewrite resourceRequest nvidia.com/* to nvidia.com/p100
	resourceRequest := make(v1.ResourceList)
	rawResourceRequest := podSpec.Containers[0].Resources.Requests
	for k, v := range rawResourceRequest {
		if strings.Contains(string(k), "nvidia.com") {
			resourceRequest["nvidia.com/p100"] = v
		} else {
			resourceRequest[k] = v
		}
	}
	podSpec.Containers[0].Resources.Requests = resourceRequest
	podSpec.Containers[0].Resources.Limits = resourceRequest

	// 2. pod meta
	podName := fmt.Sprintf("%s-%d-profiling", task.TaskName, task.ID)
	podName = strings.ToLower(podName)
	podName = strings.ReplaceAll(podName, "_", "-")
	taskID := strconv.Itoa(int(task.ID))
	labels := map[string]string{
		aijobapi.LabelKeyTaskID: taskID,
	}
	for k, v := range ProfilingPodLabels {
		labels[k] = v
	}

	// 3. pod node selector and toleration
	podSpec.NodeSelector = map[string]string{
		"profile": "true", // todo: for test
	}
	podSpec.Tolerations = []v1.Toleration{
		{
			Key:      "profile",
			Operator: v1.TolerationOpExists,
			Effect:   v1.TaintEffectNoSchedule,
		},
	}

	// 4. pod template
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: task.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"profiling-pod": "true",
			},
		},
		Spec: podSpec,
	}

	if err := c.Create(ctx, pod); err != nil {
		return fmt.Errorf("create pod %s failed: %w", task.TaskName, err)
	}

	return nil
}
