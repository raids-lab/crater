package extender

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	schedulinginternal "volcano.sh/apis/pkg/apis/scheduling"
)

// Verbs the volcano configmap must point at. onSessionClose is optional: without it the cluster view
// falls back to a time-based rebuild, which behaves the same, only a little more stale.
const (
	JobEnqueueableVerb = "jobEnqueueable"
	OnSessionCloseVerb = "onSessionClose"
)

// Vote values mirror volcano's util.Reject/Abstain. Permit is never returned: it decides the tier and
// would skip every later tier on crater's behalf.
const (
	voteReject  = -1
	voteAbstain = 0
)

const vcJobKind = "Job"

// jobEnqueueableRequest mirrors volcano's payload. api.JobInfo carries no json tags, so its Go field
// names are the wire names; only the JobInfo fields crater reads are declared here.
type jobEnqueueableRequest struct {
	Job *requestJobInfo `json:"job"`
}

type requestJobInfo struct {
	Name      string           `json:"Name"`
	Namespace string           `json:"Namespace"`
	PodGroup  *requestPodGroup `json:"PodGroup"`
}

// requestPodGroup is volcano's api.PodGroup; the embedded internal type has no json tags, so ObjectMeta is inlined.
type requestPodGroup struct {
	schedulinginternal.PodGroup
	Version string `json:"Version"`
	// Metadata catches the tagged v1beta1 shape, which volcano does not currently send.
	Metadata metav1.ObjectMeta `json:"metadata"`
}

func (pg *requestPodGroup) name() string {
	if pg.Metadata.Name != "" {
		return pg.Metadata.Name
	}
	return pg.Name
}

func (pg *requestPodGroup) ownerReferences() []metav1.OwnerReference {
	if len(pg.Metadata.OwnerReferences) > 0 {
		return pg.Metadata.OwnerReferences
	}
	return pg.OwnerReferences
}

type jobEnqueueableResponse struct {
	Status int `json:"status"`
}

// ownerJobName resolves the vcjob behind the pod group. Pod groups created by volcano's
// podgroup-controller for plain pods have no vcjob owner and are left alone.
func (req *requestJobInfo) ownerJobName() string {
	if req == nil || req.PodGroup == nil {
		return ""
	}
	refs := req.PodGroup.ownerReferences()
	for i := range refs {
		ref := &refs[i]
		if ref.Controller != nil && *ref.Controller &&
			ref.Kind == vcJobKind && ref.APIVersion == batch.SchemeGroupVersion.String() {
			return ref.Name
		}
	}
	return ""
}

func (req *requestJobInfo) podGroupName() string {
	if req == nil {
		return ""
	}
	if req.PodGroup != nil && req.PodGroup.name() != "" {
		return req.PodGroup.name()
	}
	return req.Name
}
