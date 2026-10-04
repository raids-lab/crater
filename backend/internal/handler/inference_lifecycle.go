package handler

import (
	"context"
	"errors"
	"strconv"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

// Create the workload first so networking owner references contain its actual
// UID. On failure, delete only this new workload; never adopt/overwrite a
// conflicting resource.
func (mgr *KthenaMgr) createNativeKthenaService(
	ctx context.Context, req *CreateKthenaReq, token util.JWTMessage,
) (*unstructured.Unstructured, error) {
	var result *unstructured.Unstructured
	err := service.WithWorkloadAdmission(ctx, token.UserID, token.AccountID, func() error {
		if mgr.quota != nil {
			checkCtx := ctx
			existing, err := kthena.GetDeployment(ctx, mgr.client, client.ObjectKey{Namespace: mgr.namespace, Name: req.Name})
			if err == nil {
				checkCtx = service.ExcludeServingReservation(ctx, string(existing.GetUID()))
			}
			if err := service.CheckWorkloadSubmission(
				checkCtx, mgr.admissionBans, mgr.billing, mgr.quota, token.UserID, token.AccountID, model.ScheduleTypeNormal,
			); err != nil {
				return err
			}
		}
		var err error
		result, err = mgr.createNativeKthenaServiceLocked(ctx, req, token)
		return err
	})
	return result, err
}

func (mgr *KthenaMgr) createNativeKthenaServiceLocked(
	ctx context.Context,
	req *CreateKthenaReq,
	token util.JWTMessage,
) (*unstructured.Unstructured, error) {
	req.BillingGate = mgr.billing != nil && mgr.billing.IsFeatureEnabled(ctx)
	serving, err := buildModelServing(req, token, mgr.namespace, mgr.downloaderImage)
	if err != nil {
		return nil, err
	}
	hash, err := kthenaRequestHash(req)
	if err != nil {
		return nil, err
	}
	annotations := serving.GetAnnotations()
	annotations[kthenaRequestHashAnnotation] = hash
	queued, err := mgr.shouldQueueServing(ctx, serving, token)
	if err != nil {
		return nil, err
	}
	if queued {
		annotations[kthena.QueuedReplicasAnnotation] = strconv.FormatInt(req.Replicas, 10)
		if err := unstructured.SetNestedField(serving.Object, int64(0), "spec", "replicas"); err != nil {
			return nil, err
		}
	}
	serving.SetAnnotations(annotations)
	if err := mgr.client.Create(ctx, serving); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}
		return mgr.existingServingSubmission(ctx, serving, hash, err)
	}
	for _, obj := range buildKthenaNetworking(serving, req) {
		if err := mgr.client.Create(ctx, obj); err != nil {
			return nil, mgr.rollbackKthenaCreation(ctx, serving, err)
		}
	}

	return serving, nil
}

func (mgr *KthenaMgr) rollbackKthenaCreation(ctx context.Context, serving *unstructured.Unstructured, cause error) error {
	const cleanupTimeout = 10 * time.Second
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	uid := serving.GetUID()
	cleanupErr := mgr.client.Delete(cleanupCtx, serving,
		client.Preconditions{UID: &uid}, client.PropagationPolicy(metav1.DeletePropagationBackground))
	if cleanupErr != nil && !apierrors.IsNotFound(cleanupErr) {
		return errors.Join(cause, bizerr.Internal.K8sServiceError.Wrap(cleanupErr, "failed to roll back inference ModelServing"))
	}
	return cause
}

func (mgr *KthenaMgr) rawKthenaResources(ctx context.Context, obj *unstructured.Unstructured) (map[string]any, error) {
	items := []any{obj.Object}
	for _, gvk := range []schema.GroupVersionKind{kthena.ServerGVK, kthena.RouteGVK} {
		child := &unstructured.Unstructured{}
		child.SetGroupVersionKind(gvk)
		if err := mgr.client.Get(ctx, client.ObjectKeyFromObject(obj), child); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		if isRelatedKthenaObject(child, obj) {
			items = append(items, child.Object)
		}
	}
	return map[string]any{"apiVersion": "v1", "kind": "List", "items": items}, nil
}

func (mgr *KthenaMgr) existingServingSubmission(
	ctx context.Context,
	serving *unstructured.Unstructured,
	hash string,
	cause error,
) (*unstructured.Unstructured, error) {
	existing, getErr := kthena.GetDeployment(ctx, mgr.client, client.ObjectKeyFromObject(serving))
	if getErr != nil ||
		existing.GetAnnotations()[kthenaRequestHashAnnotation] != hash ||
		existing.GetLabels()[inferenceServiceLabelUserID] != serving.GetLabels()[inferenceServiceLabelUserID] ||
		existing.GetLabels()[inferenceServiceLabelAccountID] != serving.GetLabels()[inferenceServiceLabelAccountID] {
		return nil, cause
	}
	if err := mgr.reconcileKthenaNetworking(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}
