package reconciler

import (
	"time"

	v1 "k8s.io/api/core/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/pkg/utils"
)

// resolveJobRunningTimestamp recovers the start of a short job whose Running
// transition was missed. Existing job timestamps retain their original meaning;
// otherwise use the earliest known application container start across replicas.
func resolveJobRunningTimestamp(observed, recorded time.Time, terminated []v1.ContainerStateTerminated) time.Time {
	if !observed.IsZero() {
		return observed
	}
	if !recorded.IsZero() {
		return recorded
	}
	var started time.Time
	for i := range terminated {
		candidate := terminated[i].StartedAt.Time
		if !candidate.IsZero() && (started.IsZero() || candidate.Before(started)) {
			started = candidate
		}
	}
	return started
}

func observedVCJobTimestamps(job *batch.Job) (started, ended time.Time) {
	for _, condition := range job.Status.Conditions {
		if condition.LastTransitionTime == nil {
			continue
		}
		switch condition.Status {
		case batch.Running:
			started = condition.LastTransitionTime.Time
		case batch.Completed, batch.Failed, batch.Aborted, batch.Terminated:
			ended = condition.LastTransitionTime.Time
		}
	}
	if started.IsZero() && job.Status.State.Phase == batch.Running {
		started = job.Status.State.LastTransitionTime.Time
	}
	if ended.IsZero() && utils.IsJobPhaseTerminal(job.Status.State.Phase) {
		ended = job.Status.State.LastTransitionTime.Time
	}
	return started, ended
}

// A pod finish time is a job bound only when every remaining pod is stopped.
func lastVCJobPodFinished(pods []v1.Pod) time.Time {
	var finished time.Time
	for i := range pods {
		pod := &pods[i]
		if pod.Status.Phase != v1.PodSucceeded && pod.Status.Phase != v1.PodFailed {
			return time.Time{}
		}
		var podFinished time.Time
		for j := range pod.Status.ContainerStatuses {
			terminated := pod.Status.ContainerStatuses[j].State.Terminated
			if terminated != nil && terminated.FinishedAt.After(podFinished) {
				podFinished = terminated.FinishedAt.Time
			}
		}
		if podFinished.IsZero() {
			return time.Time{}
		}
		if podFinished.After(finished) {
			finished = podFinished
		}
	}
	return finished
}
