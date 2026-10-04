package handler

import (
	"context"
	"sort"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

// KthenaReconciler repairs routing and admits queued servings. It runs under the
// controller manager's leader election and shuts down with its context.
type KthenaReconciler struct{ mgr *KthenaMgr }

func NewKthenaReconciler(conf *RegisterConfig) *KthenaReconciler {
	return &KthenaReconciler{mgr: NewKthenaMgr(conf).(*KthenaMgr)}
}
func (*KthenaReconciler) NeedLeaderElection() bool { return true }
func (r *KthenaReconciler) Start(ctx context.Context) error {
	const reconcileInterval = 15 * time.Second
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.reconcileBilling(ctx); err != nil {
				klog.Errorf("reconcile serving billing: %v", err)
			}
			if r.mgr.configService == nil || !r.mgr.configService.IsKthenaInferenceEnabled(ctx) {
				continue
			}
			if err := r.reconcile(ctx); err != nil {
				klog.Errorf("reconcile inference deployments: %v", err)
			}
		}
	}
}
func (r *KthenaReconciler) reconcile(ctx context.Context) error {
	objects,
		err := kthena.ListDeployments(ctx,
		r.mgr.client,
		r.mgr.namespace,
		client.MatchingLabels{kthena.ManagedByLabel: kthena.ManagedByValue})
	if err != nil {
		return err
	}
	sort.SliceStable(objects, func(i, j int) bool {
		return objects[i].GetCreationTimestamp().Time.Before(objects[j].GetCreationTimestamp().Time)
	})
	for i := range objects {
		obj := &objects[i]
		if obj.GetDeletionTimestamp() != nil {
			continue
		}
		if err := r.mgr.reconcileKthenaNetworking(ctx, obj); err != nil {
			klog.Warningf("repair inference routing %s: %v", obj.GetName(), err)
			continue
		}
		desired := queuedServingReplicaCount(obj)
		if desired <= 0 {
			continue
		}
		if err := r.admit(ctx, obj, desired); err != nil {
			klog.Warningf("admit queued inference %s: %v", obj.GetName(), err)
		}
	}
	return nil
}
func (r *KthenaReconciler) admit(ctx context.Context, obj *unstructured.Unstructured, desired int64) error {
	user, _ := strconv.ParseUint(obj.GetLabels()[inferenceServiceLabelUserID], 10, 64)
	account, _ := strconv.ParseUint(obj.GetLabels()[inferenceServiceLabelAccountID], 10, 64)
	return service.WithWorkloadAdmission(ctx, uint(user), uint(account), func() error { return r.admitLocked(ctx, obj, desired) })
}
func (r *KthenaReconciler) admitLocked(ctx context.Context, obj *unstructured.Unstructured, desired int64) error {
	labels, annotations := obj.GetLabels(), obj.GetAnnotations()
	userID, err := strconv.ParseUint(labels[inferenceServiceLabelUserID], 10, 64)
	if err != nil {
		return err
	}
	accountID, err := strconv.ParseUint(labels[inferenceServiceLabelAccountID], 10, 64)
	if err != nil {
		return err
	}
	token := util.JWTMessage{UserID: uint(userID),
		AccountID:   uint(accountID),
		Username:    annotations[inferenceServiceAnnotationUsername],
		AccountName: annotations[inferenceServiceAnnotationAccount]}
	if err := r.mgr.validateStoredServing(ctx, obj, token); err != nil {
		return err
	}
	if err := service.CheckWorkloadSubmission(ctx,
		r.mgr.admissionBans,
		r.mgr.billing,
		r.mgr.quota,
		token.UserID,
		token.AccountID,
		model.ScheduleTypeNormal); err != nil {
		return err
	}
	if err := r.mgr.prepareKthenaScheduling(ctx, token); err != nil {
		return err
	}
	candidate := obj.DeepCopy()
	if err := unstructured.SetNestedField(candidate.Object, desired, "spec", "replicas"); err != nil {
		return err
	}
	queued, err := r.mgr.shouldQueueServing(ctx, candidate, token)
	if err != nil || queued {
		return err
	}
	delete(annotations, kthena.QueuedReplicasAnnotation)
	candidate.SetAnnotations(annotations)
	return r.mgr.client.Update(ctx, candidate)
}
