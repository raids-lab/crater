package reconciler

import (
	"time"

	v1 "k8s.io/api/core/v1"
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
