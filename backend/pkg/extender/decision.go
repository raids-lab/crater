package extender

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/utils"
)

// maxQueueHierarchyDepth guards the parent walk against a mis-configured queue cycle.
const maxQueueHierarchyDepth = 8

// Event reasons recorded on the vcjob, the only object crater's job detail page lists events for
// before the job has pods. They mirror the log causes below.
const (
	reasonQueueQuotaExceeded     = "QueueQuotaExceeded"
	reasonBlockedByTimedOutJob   = "BlockedByTimedOutJob"
	reasonExceedsQueueCapability = "ExceedsQueueCapability"
)

// vote carries the answer plus the structured log fields that are the only diagnosis channel volcano
// leaves available for extender decisions. err marks an abstention forced by a failure rather than
// by scope, which is the one kind worth surfacing at the default log level.
type vote struct {
	status int
	fields []any
	err    error
}

// decide runs the design's vote order: per-user queue quota, then timeout blocking, then the capability
// guard, which only decides whether to reserve. Every exit is either Reject or Abstain.
func (s *Server) decide(ctx context.Context, req *requestJobInfo) vote {
	jobName := req.ownerJobName()
	if jobName == "" {
		return vote{status: voteAbstain, fields: []any{"podGroup", req.podGroupName(), "skip", "not owned by a vcjob"}}
	}

	if req.Namespace != config.GetConfig().Namespaces.Job {
		return vote{status: voteAbstain, fields: []any{
			"job", jobName, "namespace", req.Namespace, "skip", "outside the job namespace",
		}}
	}

	state, err := s.currentSession(ctx)
	if err != nil {
		return vote{status: voteAbstain, err: err, fields: []any{"job", jobName, "skip", "session state unavailable"}}
	}
	if state.snap == nil {
		return vote{status: voteAbstain}
	}
	snap := state.snap

	// Reject rather than abstain: volcano asks again next session, an abstention admits the job unchecked.
	candidate, ok := snap.byName[jobName]
	if !ok {
		return vote{status: voteReject, fields: []any{"job", jobName, "cause", "vcjob not in cache yet"}}
	}

	if result := snap.quotaVerdict(candidate, s.accumulator); result != nil && result.Exceeded {
		quota := formatQuotaDetails(result.Details)
		s.recorder.Eventf(candidate.reference(), v1.EventTypeWarning, reasonQueueQuotaExceeded,
			"per-user quota of queue %s would be exceeded with this job: %s", candidate.queue, quota)
		return vote{status: voteReject, fields: []any{
			"job", jobName,
			"cause", "queue quota exceeded",
			"queue", candidate.queue,
			"userID", candidate.userID,
			"quota", quota,
		}}
	}

	if blocker := snap.blockerFor(candidate, s.accumulator); blocker != nil {
		s.recorder.Eventf(candidate.reference(), v1.EventTypeWarning, reasonBlockedByTimedOutJob,
			"waiting behind timed-out job %s in queue %s", blocker.name, candidate.queue)
		return vote{status: voteReject, fields: []any{
			"job", jobName,
			"cause", "blocked by timed out job",
			"queue", candidate.queue,
			"blocker", blocker.name,
			"blockerPodGroupPhase", blocker.podGroupPhase,
			"blockerResources", utils.ResourceListSummary(blocker.resources),
		}}
	}

	// Capacity rejects it every round: the minimum demand exceeds the queue's or an ancestor's
	// capability. Abstain without reserving, so no standing reservation shrinks the owner's headroom
	// for as long as the job exists. Capacity records the reason only on the pod group, which crater
	// does not show, so the event repeats it on the vcjob.
	if exceeded := snap.exceededCapability(candidate); exceeded != "" {
		s.recorder.Eventf(candidate.reference(), v1.EventTypeWarning, reasonExceedsQueueCapability,
			"minimum demand exceeds the capability of queue %s", exceeded)
		return vote{status: voteAbstain, fields: []any{"job", jobName, "skip", "exceeds queue capability"}}
	}

	s.accumulator.reserve(candidate)
	return vote{status: voteAbstain}
}

// quotaVerdict returns nil when no limit applies, including for jobs created outside crater, which
// carry no user annotation and therefore belong to no user's ledger.
func (snap *snapshot) quotaVerdict(candidate *jobView, acc *sessionAccumulator) *service.ResourceLimitCheckResult {
	if candidate.userID == 0 {
		return nil
	}
	resolved := snap.quotas.Resolve(candidate.queue)
	if !resolved.Enabled {
		return nil
	}
	used := utils.SumResources(
		snap.usageByOwner[ownerKey{userID: candidate.userID, queue: candidate.queue}],
		acc.reservedExcluding(candidate.userID, candidate.queue, candidate.name),
	)
	return service.EvaluateQueueQuota(resolved, used, candidate.resources)
}

// blockerFor applies the asymmetry between the two blocker classes: an admitted timed-out job already
// queued ahead of everyone blocks every candidate, while a still unadmitted one lets equally timed-out
// peers through so a whole batch cannot deadlock on itself.
func (snap *snapshot) blockerFor(candidate *jobView, acc *sessionAccumulator) *jobView {
	candidateTimedOut := snap.isTimedOut(candidate)
	for _, view := range snap.views {
		if view.name == candidate.name || !snap.isTimedOut(view) {
			continue
		}
		if !inBlockingScope(view, candidate) {
			continue
		}
		if utils.IsPodGroupAdmitted(view.podGroupPhase) {
			if utils.IsPodGroupWaitingForNodes(view.podGroupPhase) {
				return view
			}
			continue
		}
		if candidateTimedOut {
			continue
		}
		if snap.hasBlockingPower(view, acc) {
			return view
		}
	}
	return nil
}

func inBlockingScope(blocker, candidate *jobView) bool {
	return blocker.queue == candidate.queue &&
		utils.CanResourceDomainBlock(blocker.resourceDomain, candidate.resourceDomain) &&
		utils.NodeConstraintsOverlap(blocker.nodes, candidate.nodes)
}

// hasBlockingPower is computed fresh every round, never remembered. A job waiting on its own user's
// quota would otherwise deadlock the queue on itself, and a job larger than its queue capacity would
// freeze the queue forever because it can never be admitted.
func (snap *snapshot) hasBlockingPower(blocker *jobView, acc *sessionAccumulator) bool {
	result := snap.quotaVerdict(blocker, acc)
	if result != nil && result.Exceeded {
		return false
	}
	return snap.exceededCapability(blocker) == ""
}

// exceededCapability mirrors capacity's hierarchical check: the minimum demand must fit the leaf
// queue and every ancestor. It describes the first queue that does not fit, or returns "".
func (snap *snapshot) exceededCapability(view *jobView) string {
	name := view.queue
	for depth := 0; name != "" && depth < maxQueueHierarchyDepth; depth++ {
		queue, ok := snap.queues[name]
		if !ok {
			return ""
		}
		if resource := exceededResource(view.minimum, queue.Spec.Capability); resource != "" {
			requested, limit := view.minimum[resource], queue.Spec.Capability[resource]
			return fmt.Sprintf("%s: %s requested %s capability %s", name, resource, requested.String(), limit.String())
		}
		name = queue.Spec.Parent
	}
	return ""
}

// exceededResource matches volcano's capacity plugin: absent resources and zero cpu or memory limits
// are unlimited. It returns the first resource over its limit, or "".
func exceededResource(required, capability v1.ResourceList) v1.ResourceName {
	for _, name := range slices.Sorted(maps.Keys(capability)) {
		limit := capability[name]
		if (name == v1.ResourceCPU || name == v1.ResourceMemory) && limit.Sign() <= 0 {
			continue
		}
		if requested, ok := required[name]; ok && requested.Cmp(limit) > 0 {
			return name
		}
	}
	return ""
}

func formatQuotaDetails(details []service.ResourceLimitDetail) string {
	parts := make([]string, 0, len(details))
	for i := range details {
		detail := &details[i]
		if !detail.Exceeded {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s used %s limit %s", detail.Resource, detail.Used, detail.Limit))
	}
	sort.Strings(parts)
	return strings.Join(parts, "; ")
}
