package handler

import (
	"maps"
	"slices"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/raids-lab/crater/internal/kthena"
)

func kthenaRuntimePodFromPod(pod *corev1.Pod) KthenaRuntimePod {
	readyContainers := 0
	totalContainers := len(pod.Status.ContainerStatuses)
	restarts := int32(0)
	for i := range pod.Status.ContainerStatuses {
		status := &pod.Status.ContainerStatuses[i]
		if status.Ready {
			readyContainers++
		}
		restarts += status.RestartCount
	}
	ready := isPodReady(pod)
	stage, reason, message := runtimePodStage(pod, ready)
	return KthenaRuntimePod{
		Group:     pod.Labels[kthena.GroupLabel],
		Role:      pod.Labels[kthena.RoleLabel],
		Instance:  pod.Labels["modelserving.volcano.sh/role-id"],
		Entry:     pod.Labels[kthena.EntryLabel] == kthena.EntryValue,
		Name:      pod.Name,
		Namespace: pod.Namespace,
		NodeName:  pod.Spec.NodeName,
		PodIP:     pod.Status.PodIP,
		HostIP:    pod.Status.HostIP,
		Phase:     string(pod.Status.Phase),
		Ready:     ready,
		Stage:     stage, Reason: reason, Message: message,
		Restarts:        restarts,
		ReadyContainers: readyContainers,
		TotalContainers: totalContainers,
	}
}

func isPodReady(pod *corev1.Pod) bool {
	if !pod.DeletionTimestamp.IsZero() || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return false
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

const (
	runtimePodBlocked     = "blocked"
	runtimePodFailed      = "failed"
	runtimePodScheduling  = "scheduling"
	runtimePodTerminating = "terminating"
)

func runtimePodStage(pod *corev1.Pod, ready bool) (stage, reason, message string) {
	switch {
	case !pod.DeletionTimestamp.IsZero():
		return runtimePodTerminating, "", ""
	case pod.Status.Phase == corev1.PodFailed:
		return runtimePodFailed, pod.Status.Reason, pod.Status.Message
	case ready:
		return "ready", "", ""
	case pod.Spec.NodeName == "":
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse {
				return runtimePodScheduling, condition.Reason, condition.Message
			}
		}
		return runtimePodScheduling, "", ""
	}
	for _, statuses := range [][]corev1.ContainerStatus{pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses} {
		if reason, message := blockedContainer(statuses); reason != "" {
			return runtimePodBlocked, reason, message
		}
	}
	for i := range pod.Status.InitContainerStatuses {
		status := &pod.Status.InitContainerStatuses[i]
		if status.Name == "model-downloader" && status.State.Running != nil {
			return "downloading", "", ""
		}
	}
	return "starting", "", ""
}

func blockedContainer(statuses []corev1.ContainerStatus) (reason, message string) {
	for i := range statuses {
		state := &statuses[i].State
		if terminated := state.Terminated; terminated != nil && terminated.ExitCode != 0 {
			return nonEmpty(terminated.Reason, "ContainerExited"), terminated.Message
		}
		if waiting := state.Waiting; waiting != nil {
			switch waiting.Reason {
			case "", "PodInitializing", "ContainerCreating":
			default:
				return waiting.Reason, waiting.Message
			}
		}
	}
	return "", ""
}

func observedReadyReplicas(obj *unstructured.Unstructured) *int64 {
	observed, _, _ := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	if observed < obj.GetGeneration() {
		return nil
	}
	ready, _, _ := unstructured.NestedInt64(obj.Object, "status", "availableReplicas")
	return &ready
}

func applyServingObservation(response *KthenaServiceResp) {
	groups, ready, active := servingRuntimeGroups(response.Roles, response.RuntimePods)
	response.RuntimeGroups = groups
	response.ReadyReplicas = &ready

	if response.Access.RouteName == "" || response.Access.ServerName == "" {
		response.Diagnostics = append(response.Diagnostics, KthenaDiagnostic{
			Level:   kthenaDiagnosticLevelWarning,
			Reason:  "RoutingResourcesMissing",
			Message: "Routing resources are incomplete; retry deployment repair",
		})
		if response.Phase != "Suspended" && response.Phase != "Prequeue" {
			response.Phase = kthenaPhaseProgressing
		}
		return
	}
	switch response.Phase {
	case kthenaPhaseReady, kthenaPhaseActive:
		if ready != response.Replicas || active != response.Replicas {
			response.Phase = kthenaPhaseDegraded
		}
	case kthenaPhasePending, kthenaPhaseProgressing:
		response.Phase = currentServingStartupPhase(response.Phase, response.RuntimePods)
	}
}

func currentServingStartupPhase(phase string, pods []KthenaRuntimePod) string {
	for i := range pods {
		pod := &pods[i]
		switch pod.Stage {
		case runtimePodFailed, runtimePodBlocked:
			return kthenaPhaseDegraded
		case runtimePodScheduling, runtimePodTerminating:
		default:
			phase = kthenaPhaseProgressing
		}
	}
	return phase
}

type runtimeRolePods map[string][]*KthenaRuntimePod // instance name to Pods

func servingRuntimeGroups(roles []kthena.RoleSpec, pods []KthenaRuntimePod) (groups []KthenaRuntimeGroup, ready, active int64) {
	groupPods := make(map[string]map[string]runtimeRolePods)
	for i := range pods {
		pod := &pods[i]
		if groupPods[pod.Group] == nil {
			groupPods[pod.Group] = make(map[string]runtimeRolePods)
		}
		if groupPods[pod.Group][pod.Role] == nil {
			groupPods[pod.Group][pod.Role] = make(runtimeRolePods)
		}
		instances := groupPods[pod.Group][pod.Role]
		instances[pod.Instance] = append(instances[pod.Instance], pod)
	}

	groups = make([]KthenaRuntimeGroup, 0, len(groupPods))
	for _, name := range slices.Sorted(maps.Keys(groupPods)) {
		group, hasActivePods := runtimeGroup(name, roles, groupPods[name])
		if group.Ready {
			ready++
		}
		if hasActivePods {
			active++
		}
		groups = append(groups, group)
	}
	return groups, ready, active
}

func runtimeGroup(name string, roles []kthena.RoleSpec, pods map[string]runtimeRolePods) (KthenaRuntimeGroup, bool) {
	group := KthenaRuntimeGroup{Name: name, Ready: name != "" && len(roles) > 0}
	var active bool
	for i := range roles {
		role := &roles[i]
		status, activeInstances := runtimeRole(role.Name, role.Instances, role.NodesPerInstance, pods[role.Name])
		group.Ready = group.Ready && status.ReadyInstances == role.Instances && activeInstances == role.Instances
		active = active || activeInstances > 0
		group.Roles = append(group.Roles, status)
		delete(pods, role.Name)
	}
	// Pods from a retired role can still be terminating after an update.
	for _, role := range slices.Sorted(maps.Keys(pods)) {
		status, activeInstances := runtimeRole(role, 0, 0, pods[role])
		group.Ready = group.Ready && activeInstances == 0
		active = active || activeInstances > 0
		group.Roles = append(group.Roles, status)
	}
	return group, active
}

func runtimeRole(name string, desired, nodes int64, pods runtimeRolePods) (role KthenaRuntimeRole, active int64) {
	role = KthenaRuntimeRole{Name: name, DesiredInstances: desired, Instances: make([]KthenaRuntimeInstance, 0, len(pods))}
	for _, name := range slices.Sorted(maps.Keys(pods)) {
		instance, hasActivePods := runtimeInstance(name, nodes, pods[name])
		if instance.Ready {
			role.ReadyInstances++
		}
		if hasActivePods {
			active++
		}
		role.Instances = append(role.Instances, instance)
	}
	return role, active
}

func runtimeInstance(name string, nodes int64, pods []*KthenaRuntimePod) (KthenaRuntimeInstance, bool) {
	instance := KthenaRuntimeInstance{Name: name, PodNames: make([]string, 0, len(pods))}
	var active, ready, entries int64
	for _, pod := range pods {
		instance.PodNames = append(instance.PodNames, pod.Name)
		if pod.Stage == runtimePodTerminating {
			continue
		}
		active++
		if pod.Entry {
			entries++
		}
		if pod.Ready {
			ready++
		}
	}
	instance.Ready = name != "" && nodes > 0 && entries == 1 && active == nodes && ready == nodes
	return instance, active > 0
}
