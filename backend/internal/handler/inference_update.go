package handler

import (
	"context"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const servingPausedAnnotation = "crater.raids.io/paused-replicas"

type ServingScaleRequest struct {
	Replicas *int64 `json:"replicas" binding:"required"`
}

// UserScaleServing godoc
// @Summary Scale serving groups; zero suspends the deployment
// @Tags kthena
// @Security Bearer
// @Accept json
// @Produce json
// @Param name path string true "Deployment"
// @Param request body ServingScaleRequest true "Groups"
// @Success 200 {object} resputil.Response[string]
// @Router /v1/kthena/inference-services/{name}/scale [patch]
func (mgr *KthenaMgr) UserScaleServing(c *gin.Context) {
	var req ServingScaleRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Replicas == nil || *req.Replicas < 0 || *req.Replicas > 10000 {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("replicas must be between 0 and 10000"))
		return
	}
	obj, ok := mgr.loadKthenaService(c, false)
	if !ok {
		return
	}
	token := util.GetToken(c)
	ctx := c.Request.Context()
	err := service.WithWorkloadAdmission(ctx, token.UserID, token.AccountID, func() error {
		if !obj.GetDeletionTimestamp().IsZero() {
			return bizerr.Conflict.ResourceStatusError.New("deployment is deleting")
		}
		current, _, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas")
		annotations := obj.GetAnnotations()
		if *req.Replicas == 0 {
			if current > 0 {
				annotations[servingPausedAnnotation] = strconv.FormatInt(current, 10)
			} else if desired := annotations[kthena.QueuedReplicasAnnotation]; desired != "" {
				annotations[servingPausedAnnotation] = desired
			}
			delete(annotations, kthena.QueuedReplicasAnnotation)
		} else {
			var err error
			*req.Replicas, err = mgr.prepareServingScale(ctx, obj, token, *req.Replicas, current, annotations)
			if err != nil {
				return err
			}
		}
		if err := unstructured.SetNestedField(obj.Object, *req.Replicas, "spec", "replicas"); err != nil {
			return err
		}
		obj.SetAnnotations(annotations)
		return mgr.client.Update(ctx, obj)
	})
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, "serving group count updated")
}

// UserUpdateServing godoc
// @Summary Update native serving templates with group rolling recovery
// @Tags kthena
// @Security Bearer
// @Accept json
// @Produce json
// @Param name path string true "Deployment"
// @Param request body CreateKthenaReq true "Deployment"
// @Success 200 {object} resputil.Response[string]
// @Router /v1/kthena/inference-services/{name} [put]
func (mgr *KthenaMgr) UserUpdateServing(c *gin.Context) {
	var req CreateKthenaReq
	if err := bindServingRequest(c, &req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid deployment"))
		return
	}
	obj, ok := mgr.loadKthenaService(c, false)
	if !ok {
		return
	}
	req.Name = obj.GetName()
	token := util.GetToken(c)
	ctx := c.Request.Context()
	if err := validateCreateKthenaReq(ctx, &req, token); err != nil {
		resputil.HandleError(c, err)
		return
	}
	if err := mgr.validateServingSecrets(c.Request.Context(), &req, util.GetToken(c)); err != nil {
		resputil.HandleError(c, err)
		return
	}
	if err := validateServingCapability(&req); err != nil {
		resputil.HandleError(c, err)
		return
	}
	err := service.WithWorkloadAdmission(
		ctx,
		token.UserID,
		token.AccountID,
		func() error { return mgr.updateServing(ctx, obj, &req, token) },
	)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, "serving templates updated")
}

func (mgr *KthenaMgr) updateServing(
	ctx context.Context,
	obj *unstructured.Unstructured,
	req *CreateKthenaReq,
	token util.JWTMessage,
) error {
	if err := mgr.prepareServingUpdate(ctx, obj, req, token); err != nil {
		return err
	}
	return mgr.client.Update(ctx, obj)
}

func (mgr *KthenaMgr) prepareServingUpdate(
	ctx context.Context,
	obj *unstructured.Unstructured,
	req *CreateKthenaReq,
	token util.JWTMessage,
) error {
	if !obj.GetDeletionTimestamp().IsZero() {
		return bizerr.Conflict.ResourceStatusError.New("deployment is deleting")
	}
	old, err := kthena.ReadDeployment(obj)
	if err != nil {
		return err
	}
	if old.Engine != req.BackendType || old.Layout != req.Layout || old.Port != req.Port || old.ServedModel != req.ServedModel {
		return bizerr.BadRequest.ParameterError.New(
			"engine, layout, port and served model are immutable; clone for a new routing contract",
		)
	}
	req.BillingGate = mgr.billing != nil && mgr.billing.IsFeatureEnabled(ctx)
	desired, err := buildModelServing(req, token, mgr.namespace, mgr.downloaderImage)
	if err != nil {
		return err
	}
	newDeployment, err := kthena.ReadDeployment(desired)
	if err != nil {
		return err
	}
	if err := validateServingUpdate(obj, desired, old, newDeployment, req); err != nil {
		return err
	}
	// Retain the deployment identity and its reservation. Template UUIDs must not
	// change during rolling updates or the existing ModelServer would lose Pods.
	if err := retainServingIdentity(obj, desired); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(desired.Object, old.Replicas, "spec", "replicas"); err != nil {
		return err
	}
	obj.Object["spec"] = desired.Object["spec"]
	updateServingAnnotations(obj, desired, old.Replicas, req.Replicas)
	return nil
}

func validateServingUpdate(
	obj, desired *unstructured.Unstructured,
	old, newDeployment *kthena.Deployment,
	req *CreateKthenaReq,
) error {
	oldGang, _, _ := unstructured.NestedMap(obj.Object, "spec", "template", "gangPolicy")
	newGang, _, _ := unstructured.NestedMap(desired.Object, "spec", "template", "gangPolicy")
	if !reflect.DeepEqual(oldGang, newGang) {
		return bizerr.BadRequest.ParameterError.New("role instances are immutable; scale serving groups or clone the deployment")
	}
	if old.Replicas > 0 {
		if req.Replicas != old.Replicas {
			return bizerr.BadRequest.ParameterError.New("use the scale endpoint to change serving groups")
		}
		newDeployment.Replicas = old.Replicas
		if !equalServingResources(old.TotalResources(), newDeployment.TotalResources()) {
			return bizerr.Conflict.ResourceStatusError.New("suspend deployment before changing resource allocation")
		}
	}
	return nil
}
func retainServingIdentity(obj, desired *unstructured.Unstructured) error {
	labels := desired.GetLabels()
	labels[kthena.PodDeploymentLabel] = obj.GetLabels()[kthena.PodDeploymentLabel]
	desired.SetLabels(labels)
	roles, _, _ := unstructured.NestedSlice(desired.Object, "spec", "template", "roles")
	for _, raw := range roles {
		role := raw.(map[string]any)
		for _, key := range []string{"entryTemplate", "workerTemplate"} {
			if _, ok := role[key]; ok {
				if err := unstructured.SetNestedField(
					role, labels[kthena.PodDeploymentLabel], key, "metadata", "labels", kthena.PodDeploymentLabel,
				); err != nil {
					return err
				}
			}
		}
	}
	if err := unstructured.SetNestedSlice(desired.Object, roles, "spec", "template", "roles"); err != nil {
		return err
	}
	return nil
}
func equalServingResources(a, b corev1.ResourceList) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		other, ok := b[key]
		if !ok || value.Cmp(other) != 0 {
			return false
		}
	}
	return true
}

func (mgr *KthenaMgr) prepareServingScale(
	ctx context.Context,
	obj *unstructured.Unstructured,
	token util.JWTMessage,
	replicas, current int64,
	annotations map[string]string,
) (int64, error) {
	if err := mgr.validateStoredServing(ctx, obj, token); err != nil {
		return 0, err
	}
	ctx = service.ExcludeServingReservation(ctx, string(obj.GetUID()))
	if err := service.CheckWorkloadSubmission(
		ctx, mgr.admissionBans, mgr.billing, mgr.quota, token.UserID, token.AccountID, model.ScheduleTypeNormal,
	); err != nil {
		return 0, err
	}
	if err := mgr.prepareKthenaScheduling(ctx, token); err != nil {
		return 0, err
	}
	if err := unstructured.SetNestedField(obj.Object, replicas, "spec", "replicas"); err != nil {
		return 0, err
	}
	queued, err := mgr.shouldQueueServing(ctx, obj, token)
	if err != nil {
		return 0, err
	}
	delete(annotations, servingPausedAnnotation)
	delete(annotations, kthena.QueuedReplicasAnnotation)
	if queued {
		if current > 0 {
			return 0, bizerr.Conflict.ResourceStatusError.New("insufficient quota to scale; existing deployment retained")
		}
		annotations[kthena.QueuedReplicasAnnotation] = strconv.FormatInt(replicas, 10)
		replicas = 0
	}
	return replicas, nil
}

func updateServingAnnotations(obj, desired *unstructured.Unstructured, replicas, desiredReplicas int64) {
	annotations := obj.GetAnnotations()
	for k, v := range desired.GetAnnotations() {
		annotations[k] = v
	}
	if replicas == 0 {
		if annotations[servingPausedAnnotation] != "" {
			annotations[servingPausedAnnotation] = strconv.FormatInt(desiredReplicas, 10)
		} else if annotations[kthena.QueuedReplicasAnnotation] != "" {
			annotations[kthena.QueuedReplicasAnnotation] = strconv.FormatInt(desiredReplicas, 10)
		}
	}
	delete(annotations, kthenaRequestHashAnnotation)
	obj.SetAnnotations(annotations)
}
