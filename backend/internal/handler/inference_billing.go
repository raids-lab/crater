package handler

import (
	"context"
	"slices"
	"strconv"
	"time"

	"gorm.io/datatypes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/kthena"
)

const servingBillingFinalizer = "crater.raids.io/serving-billing"
const servingBillingGate = "crater.raids.io/serving-meter"

// Run independently of the serving feature switch: disabling the UI must not
// stop accounting or strand deleting Pods with billing finalizers.
func (r *KthenaReconciler) reconcileBilling(ctx context.Context) error {
	if r.mgr.billing == nil || r.mgr.kubeClient == nil {
		return nil
	}
	pods, err := r.mgr.kubeClient.CoreV1().Pods(r.mgr.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: kthena.ManagedByLabel + "=" + kthena.ManagedByValue,
	})
	if err != nil {
		return err
	}
	for i := range pods.Items {
		pod := &pods.Items[i]
		if err := r.observeBillingPod(ctx, pod); err != nil {
			klog.Errorf("meter serving pod %s: %v", pod.Name, err)
		}
	}
	return nil
}

func (r *KthenaReconciler) observeBillingPod(ctx context.Context, pod *corev1.Pod) error {
	protected := controllerutil.ContainsFinalizer(pod, servingBillingFinalizer) ||
		slices.ContainsFunc(pod.Spec.SchedulingGates, func(g corev1.PodSchedulingGate) bool { return g.Name == servingBillingGate })
	if !protected && (r.mgr.billing == nil || !r.mgr.billing.IsFeatureEnabled(ctx)) {
		return nil
	}
	if pod.DeletionTimestamp == nil {
		base := pod.DeepCopy()
		controllerutil.AddFinalizer(pod, servingBillingFinalizer)
		pod.Spec.SchedulingGates = slices.DeleteFunc(pod.Spec.SchedulingGates, func(gate corev1.PodSchedulingGate) bool {
			return gate.Name == servingBillingGate
		})
		if len(base.Finalizers) != len(pod.Finalizers) || len(base.Spec.SchedulingGates) != len(pod.Spec.SchedulingGates) {
			// Atomically protect the Pod before permitting Volcano to schedule it.
			return r.mgr.client.Patch(ctx, pod, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{}))
		}
	}
	usage, err := servingPodUsage(pod, time.Now())
	if err != nil {
		return err
	}
	if usage != nil {
		if err := r.mgr.billing.ObserveServingUsage(ctx, usage); err != nil {
			return err
		}
	}
	if pod.DeletionTimestamp != nil && controllerutil.ContainsFinalizer(pod, servingBillingFinalizer) {
		base := pod.DeepCopy()
		controllerutil.RemoveFinalizer(pod, servingBillingFinalizer)
		return r.mgr.client.Patch(ctx, pod, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{}))
	}
	return nil
}

func servingPodUsage(pod *corev1.Pod, now time.Time) (*model.ServingUsage, error) {
	if pod.Spec.NodeName == "" || pod.Status.StartTime == nil {
		return nil, nil
	}
	userID, err := strconv.ParseUint(pod.Labels[inferenceServiceLabelUserID], 10, 64)
	if err != nil {
		return nil, err
	}
	accountID, err := strconv.ParseUint(pod.Labels[inferenceServiceLabelAccountID], 10, 64)
	if err != nil {
		return nil, err
	}
	if pod.UID == "" || pod.Labels[kthena.PodDeploymentLabel] == "" {
		return nil, nil
	}
	resources := servingPodResources(pod)
	usage := &model.ServingUsage{PodUID: string(pod.UID), DeploymentID: pod.Labels[kthena.PodDeploymentLabel],
		Namespace: pod.Namespace, ServiceName: pod.Labels["modelserving.volcano.sh/name"], UserID: uint(userID), AccountID: uint(accountID),
		Resources: datatypes.NewJSONType(resources), StartedAt: pod.Status.StartTime.Time, ObservedUntil: now}
	usage.EndedAt = servingPodEnd(pod)
	if usage.EndedAt != nil {
		usage.ObservedUntil = *usage.EndedAt
	}
	return usage, nil
}

func servingPodResources(pod *corev1.Pod) corev1.ResourceList {
	return kthena.PodRequests(&pod.Spec)
}

func servingPodEnd(pod *corev1.Pod) *time.Time {
	var endedAt *time.Time
	if pod.DeletionTimestamp != nil {
		end := pod.DeletionTimestamp.Time
		endedAt = &end
	}
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		end := pod.Status.StartTime.Time
		statuses := append(append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
		for i := range statuses {
			if state := statuses[i].State.Terminated; state != nil && state.FinishedAt.After(end) {
				end = state.FinishedAt.Time
			}
		}
		if endedAt == nil || end.Before(*endedAt) {
			endedAt = &end
		}
	}
	return endedAt
}
