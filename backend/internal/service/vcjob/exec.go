package vcjob

import (
	"fmt"
	"slices"
	"strings"

	v1 "k8s.io/api/core/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/internal/bizerr"
)

// SelectExecTarget selects one running container owned by the live Volcano job.
// Selection is independent of node architecture and accelerator resources.
func SelectExecTarget(job *batch.Job, pods []v1.Pod, podName, containerName string) (*v1.Pod, string, error) {
	selected, err := selectExecPod(job, pods, podName)
	if err != nil {
		return nil, "", err
	}
	if selected.Status.Phase != v1.PodRunning || selected.DeletionTimestamp != nil {
		return nil, "", bizerr.Conflict.ResourceStatusError.New("selected pod is not running")
	}
	container, err := selectExecContainer(selected, containerName)
	return selected, container, err
}

func selectExecPod(job *batch.Job, pods []v1.Pod, podName string) (*v1.Pod, error) {
	var candidates []*v1.Pod
	for i := range pods {
		pod := &pods[i]
		if pod.Namespace != job.Namespace || !ownedByJob(pod, job) {
			continue
		}
		candidates = append(candidates, pod)
	}
	slices.SortFunc(candidates, func(a, b *v1.Pod) int { return strings.Compare(a.Name, b.Name) })
	if len(candidates) == 0 {
		return nil, bizerr.NotFound.K8sResourceNotFound.New("job has no pods")
	}
	var selected *v1.Pod
	for _, pod := range candidates {
		if pod.Name == podName || (podName == "" && len(candidates) == 1) {
			selected = pod
			break
		}
	}
	if selected == nil {
		names := make([]string, len(candidates))
		for i, pod := range candidates {
			names[i] = pod.Name
		}
		return nil, bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("select --pod from: %s", strings.Join(names, ", ")),
		)
	}
	return selected, nil
}

func selectExecContainer(selected *v1.Pod, containerName string) (string, error) {
	if containerName == "" {
		if len(selected.Spec.Containers) != 1 {
			return "", bizerr.BadRequest.ParameterError.New("select --container for this pod")
		}
		containerName = selected.Spec.Containers[0].Name
	}
	for _, statuses := range [][]v1.ContainerStatus{selected.Status.ContainerStatuses, selected.Status.InitContainerStatuses} {
		for i := range statuses {
			if statuses[i].Name == containerName && statuses[i].State.Running != nil {
				return containerName, nil
			}
		}
	}
	return "", bizerr.Conflict.ResourceStatusError.New("selected container does not exist or is not running")
}

func ownedByJob(pod *v1.Pod, job *batch.Job) bool {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "Job" && owner.APIVersion == batch.SchemeGroupVersion.String() &&
			owner.Name == job.Name && owner.UID == job.UID && owner.Controller != nil && *owner.Controller {
			return true
		}
	}
	return false
}
