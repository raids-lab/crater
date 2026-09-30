package extender

import (
	"context"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/internal/service"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/utils"
)

// jobView joins the two caches the decision needs: the vcjob supplies demand and constraints, the
// pod group supplies the admission phase. The pair is strictly 1:1.
type jobView struct {
	name           string
	userID         uint
	queue          string
	resources      v1.ResourceList
	minimum        v1.ResourceList
	resourceDomain string
	nodes          sets.Set[string]
	jobPhase       batch.JobPhase
	podGroupPhase  scheduling.PodGroupPhase // empty while the vcjob has no pod group yet
	createdAt      time.Time
	tolerance      time.Duration // zero for a job without the annotation, which then never times out
}

// ownerKey groups the quota ledger by user and queue; the account dimension is implied because a
// queue never spans accounts.
type ownerKey struct {
	userID uint
	queue  string
}

type snapshot struct {
	views        []*jobView
	byName       map[string]*jobView
	queues       map[string]*scheduling.Queue
	quotas       *service.QueueQuotaSet
	usageByOwner map[ownerKey]v1.ResourceList
	now          time.Time
}

func (s *Server) buildSnapshot(ctx context.Context, current *settings) (*snapshot, error) {
	namespace := config.GetConfig().Namespaces.Job
	// Deep copies are skipped because nothing below mutates the cached objects.
	var jobs batch.JobList
	if listErr := s.reader.List(ctx, &jobs, client.InNamespace(namespace), client.UnsafeDisableDeepCopy); listErr != nil {
		return nil, listErr
	}
	var groups scheduling.PodGroupList
	if listErr := s.reader.List(ctx, &groups, client.InNamespace(namespace), client.UnsafeDisableDeepCopy); listErr != nil {
		return nil, listErr
	}
	var queues scheduling.QueueList
	if listErr := s.reader.List(ctx, &queues, client.UnsafeDisableDeepCopy); listErr != nil {
		return nil, listErr
	}

	groupsByJob := podGroupsByOwnerJob(&groups)
	snap := &snapshot{
		views:        make([]*jobView, 0, len(jobs.Items)),
		byName:       make(map[string]*jobView, len(jobs.Items)),
		queues:       make(map[string]*scheduling.Queue, len(queues.Items)),
		quotas:       current.quotas,
		usageByOwner: make(map[ownerKey]v1.ResourceList),
		now:          utils.GetLocalTime(),
	}
	for i := range queues.Items {
		snap.queues[queues.Items[i].Name] = &queues.Items[i]
	}
	for i := range jobs.Items {
		job := &jobs.Items[i]
		view := newJobView(job, groupsByJob[job.Name])
		snap.views = append(snap.views, view)
		snap.byName[view.name] = view
		// Only admitted, unfinished jobs consume a user's limit, so a job held back by its own quota
		// never counts against itself.
		if utils.IsPodGroupAdmitted(view.podGroupPhase) && !utils.IsJobPhaseTerminal(view.jobPhase) {
			key := ownerKey{userID: view.userID, queue: view.queue}
			snap.usageByOwner[key] = utils.SumResources(snap.usageByOwner[key], view.resources)
		}
	}
	return snap, nil
}

func podGroupsByOwnerJob(groups *scheduling.PodGroupList) map[string]*scheduling.PodGroup {
	result := make(map[string]*scheduling.PodGroup, len(groups.Items))
	for i := range groups.Items {
		group := &groups.Items[i]
		if owner := controllerOwnerJobName(group.OwnerReferences); owner != "" {
			result[owner] = group
		}
	}
	return result
}

func controllerOwnerJobName(refs []metav1.OwnerReference) string {
	for i := range refs {
		ref := &refs[i]
		if ref.Controller != nil && *ref.Controller &&
			ref.Kind == vcJobKind && ref.APIVersion == batch.SchemeGroupVersion.String() {
			return ref.Name
		}
	}
	return ""
}

func newJobView(job *batch.Job, group *scheduling.PodGroup) *jobView {
	resources := vcjobservice.CalculateJobResources(job)
	view := &jobView{
		name:           job.Name,
		userID:         annotationUserID(job.Annotations),
		queue:          job.Spec.Queue,
		resources:      resources,
		minimum:        resources,
		resourceDomain: utils.GetResourceDomain(resources),
		nodes:          utils.GetJobExplicitNodeNames(job),
		jobPhase:       job.Status.State.Phase,
		createdAt:      job.CreationTimestamp.Time,
		tolerance:      annotationTolerance(job.Annotations),
	}
	if group != nil {
		view.podGroupPhase = group.Status.Phase
		if group.Spec.MinResources != nil {
			view.minimum = *group.Spec.MinResources
		}
	}
	return view
}

// isWaiting covers the window before volcano admits the job; the empty phase is the state a freshly
// created vcjob sits in until the job controller writes one.
func isWaiting(phase batch.JobPhase) bool {
	return phase == "" || phase == batch.Pending
}

// isTimedOut answers against the session clock, never a live one, so every job asked during one round
// is judged against the same instant.
func (snap *snapshot) isTimedOut(view *jobView) bool {
	return view.tolerance > 0 && isWaiting(view.jobPhase) &&
		snap.now.After(view.createdAt.Add(view.tolerance))
}

func annotationUserID(annotations map[string]string) uint {
	userID, err := strconv.ParseUint(annotations[vcjobservice.AnnotationKeyUserID], 10, 64)
	if err != nil {
		return 0
	}
	return uint(userID)
}

// annotationTolerance returns zero for a job that carries no tolerance, which then never times out and
// never blocks anyone. Crater stamps the annotation at submission, so this only covers jobs created
// outside the platform.
func annotationTolerance(annotations map[string]string) time.Duration {
	seconds, err := strconv.ParseInt(annotations[vcjobservice.AnnotationKeyWaitingToleranceSeconds], 10, 64)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
