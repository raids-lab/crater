package reconciler

import (
	"context"
	"fmt"
	"sort"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
)

const MaxJobEvents = 20

func getPodNameFromJobTemplate(job *batch.Job) string {
	for i := range job.Spec.Tasks {
		task := &job.Spec.Tasks[i]
		if task.Replicas > 0 {
			podName := fmt.Sprintf("%s-%s-%d", job.Name, task.Name, 0)
			return podName
		}
	}
	return ""
}

func getPodNamesFromJobTemplate(job *batch.Job) []string {
	podNames := make([]string, 0)
	for i := range job.Spec.Tasks {
		task := &job.Spec.Tasks[i]
		for j := int32(0); j < task.Replicas; j++ {
			podName := fmt.Sprintf("%s-%s-%d", job.Name, task.Name, j)
			podNames = append(podNames, podName)
		}
	}
	return podNames
}

func (r *VcJobReconciler) getNewEventsForJob(c context.Context, job *batch.Job, oldRecord *model.Job) []v1.Event {
	podNames := getPodNamesFromJobTemplate(job)
	if len(podNames) == 0 {
		return nil
	}

	// get old events
	var oldEvents []v1.Event
	if oldRecord.Events != nil {
		oldEvents = oldRecord.Events.Data()
	}

	var allEvents []*v1.Event
	for _, podName := range podNames {
		events, err := r.kubeClient.CoreV1().Events(job.Namespace).List(c, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", podName),
			TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
		})
		if err != nil {
			continue
		}
		for i := range events.Items {
			event := &events.Items[i]
			event.ManagedFields = nil
			allEvents = append(allEvents, event)
		}
	}

	if len(allEvents) == 0 {
		return nil
	}

	eventsMap := make(map[types.UID]*v1.Event)
	for i := range oldEvents {
		event := &oldEvents[i]
		eventsMap[event.UID] = event
	}
	for i := range allEvents {
		event := allEvents[i]
		if _, ok := eventsMap[event.UID]; !ok {
			eventsMap[event.UID] = event
		}
	}

	events := make([]v1.Event, 0)
	for _, event := range eventsMap {
		events = append(events, *event)
	}

	// sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].LastTimestamp.After(events[j].LastTimestamp.Time)
	})

	// 最大保留 20 条最近的事件，避免存储过多的事件
	if len(events) > MaxJobEvents {
		klog.Warningf(
			"Job %s/%s has too many events (%d), only keep the latest %d events", job.Namespace, job.Name, len(events), MaxJobEvents,
		)
		events = events[:MaxJobEvents]
	}

	return events
}

func (r *VcJobReconciler) getTerminatedStates(c context.Context, job *batch.Job, oldRecord *model.Job) []v1.ContainerStateTerminated {
	if oldRecord.TerminatedStates != nil {
		return nil
	}
	podNames := getPodNamesFromJobTemplate(job)
	if len(podNames) == 0 {
		return nil
	}

	var allTerminatedStates []v1.ContainerStateTerminated
	for _, podName := range podNames {
		pod, err := r.kubeClient.CoreV1().Pods(job.Namespace).Get(c, podName, metav1.GetOptions{})
		if err != nil {
			continue
		}
		for i := range pod.Status.ContainerStatuses {
			status := &pod.Status.ContainerStatuses[i]
			if status.State.Terminated != nil {
				allTerminatedStates = append(allTerminatedStates, *status.State.Terminated)
			}
		}
	}

	if len(allTerminatedStates) == 0 {
		return nil
	}

	return allTerminatedStates
}
