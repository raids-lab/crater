package handler

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/raids-lab/crater/internal/kthena"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
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
	return KthenaRuntimePod{
		Group:           pod.Labels[kthena.GroupLabel],
		Role:            pod.Labels[kthena.RoleLabel],
		Instance:        pod.Labels["modelserving.volcano.sh/role-id"],
		Entry:           pod.Labels[kthena.EntryLabel] == kthena.EntryValue,
		Name:            pod.Name,
		Namespace:       pod.Namespace,
		NodeName:        pod.Spec.NodeName,
		PodIP:           pod.Status.PodIP,
		HostIP:          pod.Status.HostIP,
		Phase:           string(pod.Status.Phase),
		Ready:           isPodReady(pod),
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

func (mgr *KthenaMgr) diagnosticsFromPod(ctx context.Context, pod *corev1.Pod) []KthenaDiagnostic {
	if pod == nil {
		return nil
	}
	diagnostics := make([]KthenaDiagnostic, 0)
	resource := "Pod/" + pod.Namespace + "/" + pod.Name
	if pod.Status.Phase == corev1.PodPending {
		for _, condition := range pod.Status.Conditions {
			if condition.Type != corev1.PodScheduled || condition.Status != corev1.ConditionFalse {
				continue
			}
			diagnostics = append(diagnostics, KthenaDiagnostic{
				Level:     kthenaDiagnosticLevelWarning,
				Reason:    nonEmpty(condition.Reason, "PodSchedulingFailed"),
				Message:   condition.Message,
				Resource:  resource,
				Pod:       pod.Name,
				Timestamp: condition.LastTransitionTime.Time,
			})
		}
	}
	for i := range pod.Status.InitContainerStatuses {
		status := &pod.Status.InitContainerStatuses[i]
		logTail := ""
		if status.State.Waiting != nil || (status.State.Terminated != nil && status.State.Terminated.ExitCode != 0) {
			logTail = mgr.containerLogTail(ctx, pod, status.Name, status.RestartCount)
		}
		diagnostics = append(diagnostics, diagnosticsFromContainerStatus(pod, status, true, logTail)...)
	}
	for i := range pod.Status.ContainerStatuses {
		status := &pod.Status.ContainerStatuses[i]
		logTail := ""
		if status.State.Waiting != nil || (status.State.Terminated != nil && status.State.Terminated.ExitCode != 0) {
			logTail = mgr.containerLogTail(ctx, pod, status.Name, status.RestartCount)
		}
		diagnostics = append(diagnostics, diagnosticsFromContainerStatus(pod, status, false, logTail)...)
	}
	return diagnostics
}

func diagnosticsFromContainerStatus(
	pod *corev1.Pod,
	status *corev1.ContainerStatus,
	initContainer bool,
	logTail string,
) []KthenaDiagnostic {
	if pod == nil || status == nil {
		return nil
	}
	resource := "Pod/" + pod.Namespace + "/" + pod.Name
	containerKind := "container"
	reasonPrefix := "Runtime"
	if initContainer {
		containerKind = "initContainer"
		reasonPrefix = "InitContainer"
	}
	if waiting := status.State.Waiting; waiting != nil {
		return []KthenaDiagnostic{{
			Level:     kthenaDiagnosticLevelWarning,
			Reason:    nonEmpty(waiting.Reason, reasonPrefix+"Waiting"),
			Message:   nonEmpty(waiting.Message, fmt.Sprintf("%s %q is waiting", containerKind, status.Name)),
			Details:   logTail,
			Resource:  resource,
			Pod:       pod.Name,
			Container: status.Name,
		}}
	}
	if terminated := status.State.Terminated; terminated != nil && terminated.ExitCode != 0 {
		return []KthenaDiagnostic{{
			Level: "error",
			Reason: nonEmpty(
				terminated.Reason,
				reasonPrefix+"Failed",
			),
			Message: nonEmpty(
				terminated.Message,
				fmt.Sprintf("%s %q terminated with exit code %d", containerKind, status.Name, terminated.ExitCode),
			),
			Details:   logTail,
			Resource:  resource,
			Pod:       pod.Name,
			Container: status.Name,
			Timestamp: terminated.FinishedAt.Time,
		}}
	}
	return nil
}

func (mgr *KthenaMgr) containerLogTail(
	ctx context.Context,
	pod *corev1.Pod,
	container string,
	restartCount int32,
) string {
	if mgr.kubeClient == nil || pod == nil || container == "" {
		return ""
	}
	if logs := mgr.readContainerLogTail(ctx, pod.Namespace, pod.Name, container, false); logs != "" {
		return logs
	}
	if restartCount > 0 {
		return mgr.readContainerLogTail(ctx, pod.Namespace, pod.Name, container, true)
	}
	return ""
}

func (mgr *KthenaMgr) readContainerLogTail(
	ctx context.Context,
	namespace string,
	pod string,
	container string,
	previous bool,
) string {
	tailLines := kthenaDefaultLogTailLines
	req := mgr.kubeClient.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{
		Container: container,
		Previous:  previous,
		TailLines: &tailLines,
	})
	stream, err := req.Stream(ctx)
	if err != nil {
		klog.V(kthenaLogVerbosity).Infof(
			"read logs for kthena pod %s/%s container %s previous=%t failed: %v",
			namespace,
			pod,
			container,
			previous,
			err,
		)
		return "Container logs are currently unavailable; check pod state and logs permission."
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			klog.V(kthenaLogVerbosity).Infof(
				"close log stream for kthena pod %s/%s container %s failed: %v",
				namespace,
				pod,
				container,
				closeErr,
			)
		}
	}()
	const maxLogBytes = 64 << 10
	data, err := io.ReadAll(io.LimitReader(stream, maxLogBytes))
	if err != nil {
		klog.V(kthenaLogVerbosity).Infof(
			"read log stream for kthena pod %s/%s container %s failed: %v",
			namespace,
			pod,
			container,
			err,
		)
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (mgr *KthenaMgr) diagnosticsFromPodEvents(ctx context.Context, pod *corev1.Pod) []KthenaDiagnostic {
	events, err := mgr.kubeClient.CoreV1().Events(pod.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod.Name),
	})
	if err != nil {
		klog.V(kthenaLogVerbosity).Infof("list events for kthena pod %s/%s failed: %v", pod.Namespace, pod.Name, err)
		return []KthenaDiagnostic{{Level: kthenaDiagnosticLevelWarning,
			Reason:  "EventsUnavailable",
			Message: "Could not read pod events; check cluster access and retry.",
			Pod:     pod.Name}}
	}
	diagnostics := make([]KthenaDiagnostic, 0, len(events.Items))
	for i := range events.Items {
		event := &events.Items[i]
		// Pod names are reused when a ModelServing is recreated. Events from a
		// previous pod with the same name can remain in the namespace and must
		// not make the replacement deployment appear unhealthy.
		if event.InvolvedObject.UID != pod.UID {
			continue
		}
		if event.Type != corev1.EventTypeWarning {
			continue
		}
		diagnostics = append(diagnostics, KthenaDiagnostic{
			Level:     kthenaDiagnosticLevelWarning,
			Reason:    nonEmpty(event.Reason, "PodWarning"),
			Message:   event.Message,
			Resource:  "Pod/" + pod.Namespace + "/" + pod.Name,
			Pod:       pod.Name,
			Timestamp: event.LastTimestamp.Time,
		})
	}
	return diagnostics
}

func runtimeAwareInferencePhase(phase string, roles []kthena.RoleSpec, replicas int64, pods []KthenaRuntimePod) string {
	if phase != kthenaPhaseReady && phase != kthenaPhaseActive {
		return phase
	}
	if replicas < 1 || len(roles) == 0 {
		return kthenaPhaseDegraded
	}
	groups := map[string]map[string]map[string][]KthenaRuntimePod{}
	for i := range pods {
		pod := &pods[i]
		if !pod.Ready {
			return kthenaPhaseDegraded
		}
		if groups[pod.Group] == nil {
			groups[pod.Group] = map[string]map[string][]KthenaRuntimePod{}
		}
		if groups[pod.Group][pod.Role] == nil {
			groups[pod.Group][pod.Role] = map[string][]KthenaRuntimePod{}
		}
		groups[pod.Group][pod.Role][pod.Instance] = append(groups[pod.Group][pod.Role][pod.Instance], *pod)
	}
	if int64(len(groups)) != replicas {
		return kthenaPhaseDegraded
	}
	for _, group := range groups {
		if !servingGroupReady(group, roles) {
			return kthenaPhaseDegraded
		}
	}
	return phase
}

func servingGroupReady(group map[string]map[string][]KthenaRuntimePod, roles []kthena.RoleSpec) bool {
	if len(group) != len(roles) {
		return false
	}
	for i := range roles {
		role := &roles[i]
		instances := group[role.Name]
		if int64(len(instances)) != role.Instances {
			return false
		}
		for _, instance := range instances {
			entries := 0
			for i := range instance {
				if instance[i].Entry {
					entries++
				}
			}
			if entries != 1 || int64(len(instance)) != role.NodesPerInstance {
				return false
			}
		}
	}
	return true
}
