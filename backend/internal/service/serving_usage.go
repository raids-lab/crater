package service

import (
	"context"
	"strconv"

	"gorm.io/datatypes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/kthena"
)

func (s *PrequeueService) SetServingClient(c client.Reader, namespace string) {
	s.servingClient, s.servingNamespace = c, namespace
}

type servingPodUsage struct {
	active, terminating corev1.ResourceList
}

type servingReservation struct {
	uid, queue string
	resources  corev1.ResourceList
}

// Reserve max(desired, active Pods) plus terminating Pods, which cannot serve
// as replacements for new replicas.
// Pods survive asynchronous scale-down and owner deletion; their allocation must
// remain visible to admission until Kubernetes has actually released it.
func (s *PrequeueService) servingUsage(ctx context.Context, userID, accountID uint) ([]*model.Job, error) {
	if s.servingClient == nil {
		return nil, nil
	}
	labels := client.MatchingLabels{
		"crater.raids.io/user-id":    strconv.FormatUint(uint64(userID), 10),
		"crater.raids.io/account-id": strconv.FormatUint(uint64(accountID), 10),
		kthena.ManagedByLabel:        kthena.ManagedByValue,
	}
	objects, err := kthena.ListDeployments(ctx, s.servingClient, s.servingNamespace, labels)
	if err != nil && !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
		return nil, err
	}
	pods := &corev1.PodList{}
	if err := s.servingClient.List(ctx, pods, client.InNamespace(s.servingNamespace), labels); err != nil {
		return nil, err
	}
	reservations, live := liveServingReservations(pods)
	for i := range objects {
		obj := &objects[i]
		d, err := kthena.ReadDeployment(obj)
		if err != nil {
			return nil, err
		}
		key := obj.GetLabels()[kthena.PodDeploymentLabel]
		if key == "" {
			key = "serving/" + obj.GetName()
		}
		desired := d.TotalResources()
		if !obj.GetDeletionTimestamp().IsZero() {
			desired = corev1.ResourceList{}
		}
		reservations[key] = &servingReservation{
			uid:       string(obj.GetUID()),
			queue:     obj.GetAnnotations()[scheduling.QueueNameAnnotationKey],
			resources: desired,
		}
	}
	return servingReservationJobs(ctx, userID, accountID, reservations, live), nil
}

func liveServingReservations(pods *corev1.PodList) (reservations map[string]*servingReservation, live map[string]*servingPodUsage) {
	reservations = map[string]*servingReservation{}
	live = map[string]*servingPodUsage{}
	for i := range pods.Items {
		p := &pods.Items[i]
		if p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed {
			continue
		}
		key := p.Labels[kthena.PodDeploymentLabel]
		if key == "" {
			key = "pod/" + p.Name
		}
		if live[key] == nil {
			live[key] = &servingPodUsage{active: corev1.ResourceList{}, terminating: corev1.ResourceList{}}
		}
		resources := live[key].active
		if !p.DeletionTimestamp.IsZero() {
			resources = live[key].terminating
		}
		for k, q := range kthena.PodRequests(&p.Spec) {
			value := resources[k]
			value.Add(q)
			resources[k] = value
		}
		reservations[key] = &servingReservation{uid: "pods/" + key, queue: p.Annotations[scheduling.QueueNameAnnotationKey]}
	}
	return reservations, live
}

func servingReservationJobs(
	ctx context.Context,
	userID, accountID uint,
	reservations map[string]*servingReservation,
	live map[string]*servingPodUsage,
) []*model.Job {
	excluded, _ := ctx.Value(servingExclusionKey{}).(string)
	requested, _ := ctx.Value(servingReplacementKey{}).(corev1.ResourceList)
	jobs := make([]*model.Job, 0, len(reservations))
	for key, reservation := range reservations {
		resources := reservation.resources
		if resources == nil || (excluded != "" && reservation.uid == excluded) {
			resources = corev1.ResourceList{}
		}
		pods := live[key]
		if pods == nil {
			pods = &servingPodUsage{}
		}
		for k, q := range pods.active {
			if excluded != "" && reservation.uid == excluded {
				q.Sub(requested[k])
			}
			if value := resources[k]; q.Cmp(value) > 0 {
				resources[k] = q
			}
		}
		for k, q := range pods.terminating {
			value := resources[k]
			value.Add(q)
			resources[k] = value
		}
		if !hasServingResources(resources) {
			continue
		}
		jobs = append(jobs, &model.Job{JobName: reservation.uid, UserID: userID, AccountID: accountID,
			Queue: reservation.queue, Status: batch.Pending,
			Resources: datatypes.NewJSONType(resources), ScheduleType: ptr.To(model.ScheduleTypeNormal)})
	}
	return jobs
}

func hasServingResources(resources corev1.ResourceList) bool {
	for _, q := range resources {
		if q.Sign() > 0 {
			return true
		}
	}
	return false
}

type servingExclusionKey struct{}
type servingReplacementKey struct{}

// Exclude the desired reservation, while retaining live excess over its replacement.
func ExcludeServingReservation(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, servingExclusionKey{}, uid)
}
func WithServingReplacementResources(ctx context.Context, resources corev1.ResourceList) context.Context {
	return context.WithValue(ctx, servingReplacementKey{}, resources)
}
