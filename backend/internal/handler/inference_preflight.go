package handler

import (
	"context"
	"fmt"

	"github.com/raids-lab/crater/dao/model"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/vcqueue"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const servingPreviewCreate = "create"

type ServingCapability struct {
	Images    []string `json:"images"`
	Connector string   `json:"connector,omitempty"`
	ID        string   `json:"id"`
	Engine    string   `json:"engine"`
	Layout    string   `json:"layout"`
	Execution string   `json:"execution"`
	Enabled   bool     `json:"enabled"`
	Evidence  string   `json:"evidence"`
	Reason    string   `json:"reason,omitempty"`
}

type ServingPreflight struct {
	Spec           CreateKthenaReq     `json:"spec"`
	TotalResources corev1.ResourceList `json:"totalResources"`
	Pods           int64               `json:"pods"`
	Queue          string              `json:"queue"`
	Queued         bool                `json:"queued"`
	Warnings       []string            `json:"warnings"`
	Resources      []map[string]any    `json:"resources"`
}

func servingCapabilities() []ServingCapability {
	out := []ServingCapability{}
	for _, engine := range []string{"vLLM", "SGLang"} {
		for _, layout := range []string{"combined", kthena.LayoutPD} {
			for _, execution := range []string{"single", kthena.ExecutionRay} {
				if engine == "SGLang" && execution == kthena.ExecutionRay {
					continue
				}
				id := engine + "/" + layout + "/" + execution
				enabled := layout == "combined" && execution == "single"
				for _, allowed := range config.GetConfig().Kthena.EnabledProfiles {
					if id == allowed {
						enabled = true
					}
				}
				if len(servingProfileImages(id, engine)) == 0 {
					enabled = false
				}
				reason := ""
				if !enabled {
					reason = "Administrator must enable this profile after validating images, GPU communication and KV transfer in this cluster"
				}
				out = append(
					out,
					ServingCapability{
						Images:    servingProfileImages(id, engine),
						Connector: servingProfileConnector(engine, layout),
						ID:        id,
						Engine:    engine,
						Layout:    layout,
						Execution: execution,
						Enabled:   enabled,
						Evidence:  "configured",
						Reason:    reason,
					},
				)
			}
		}
	}
	return out
}

func validateServingCapability(req *CreateKthenaReq) error {
	for i := range req.Roles {
		r := &req.Roles[i]
		id := req.BackendType + "/" + req.Layout + "/" + r.Execution
		allowed := false
		capabilities := servingCapabilities()
		for i := range capabilities {
			c := &capabilities[i]
			if c.ID == id && c.Enabled {
				allowed = true
				if err := validateServingProfileImages(r, c.Images); err != nil {
					return err
				}
			}
		}
		if !allowed {
			return bizerr.Conflict.ResourceStatusError.New("serving profile is not enabled: " + id)
		}
	}
	return nil
}

func (mgr *KthenaMgr) checkServingAPIs(ctx context.Context) error {
	// Listing exercises namespace RBAC and the served API for each resource,
	// instead of treating one CRD's discovery entry as proof of all three APIs.
	for _, gvk := range []schema.GroupVersionKind{kthena.ServingGVK, kthena.ServerGVK, kthena.RouteGVK} {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
		if err := mgr.client.List(ctx, list, client.InNamespace(mgr.namespace), client.Limit(1)); err != nil {
			return bizerr.Internal.K8sServiceError.Wrap(err, fmt.Sprintf("%s %s unavailable", gvk.GroupVersion(), gvk.Kind))
		}
	}
	return nil
}

// UserServingCapabilities godoc
// @Summary List configured serving profiles and API availability
// @Tags kthena
// @Security Bearer
// @Produce json
// @Success 200 {object} resputil.Response[[]ServingCapability]
// @Router /v1/kthena/inference-services/capabilities [get]
func (mgr *KthenaMgr) UserServingCapabilities(c *gin.Context) {
	capabilities := servingCapabilities()
	if err := mgr.checkServingAPIs(c.Request.Context()); err != nil {
		for i := range capabilities {
			capabilities[i].Enabled = false
			capabilities[i].Evidence = "unknown"
			capabilities[i].Reason = err.Error()
		}
	}
	resputil.Success(c, capabilities)
}

// UserPreviewServing godoc
// @Summary Validate deployment and preview its native resources without reserving capacity
// @Tags kthena
// @Security Bearer
// @Accept json
// @Produce json
// @Param mode query string false "Operation: create or update"
// @Param request body CreateKthenaReq true "Deployment"
// @Success 200 {object} resputil.Response[ServingPreflight]
// @Router /v1/kthena/inference-services/preview [post]
func (mgr *KthenaMgr) UserPreviewServing(c *gin.Context) {
	var req CreateKthenaReq
	if err := bindServingRequest(c, &req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid deployment"))
		return
	}
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
	if err := mgr.checkServingAPIs(ctx); err != nil {
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.Wrap(err, "serving APIs unavailable"))
		return
	}
	req.BillingGate = mgr.billing != nil && mgr.billing.IsFeatureEnabled(ctx)
	serving, err := buildModelServing(&req, token, mgr.namespace, mgr.downloaderImage)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	serving, ctx, err = mgr.prepareServingPreview(c, &req, serving)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	queued := serving.GetAnnotations()[kthena.QueuedReplicasAnnotation] != ""
	if c.DefaultQuery("mode", servingPreviewCreate) == servingPreviewCreate {
		queued, err = mgr.shouldQueueServing(ctx, serving, token)
		if err != nil {
			resputil.HandleError(c, err)
			return
		}
	}
	deployment, err := kthena.ReadDeployment(serving)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	// Resource totals describe the requested configuration; a paused update keeps
	// spec.replicas at zero in the actual manifest until explicit resume.
	deployment.Replicas = req.Replicas
	result := ServingPreflight{
		Spec:           req,
		TotalResources: deployment.TotalResources(),
		Queue:          vcqueue.ResolveJobQueueName(token),
		Queued:         queued,
		Resources:      []map[string]any{serving.Object},
		Warnings: []string{
			"Preview does not reserve capacity; submission revalidates quota.",
			"Runtime image and GPU/network compatibility require cluster acceptance.",
		},
	}
	for i := range req.Roles {
		r := &req.Roles[i]
		result.Pods += req.Replicas * r.Instances * r.NodesPerInstance
	}
	for _, obj := range buildKthenaNetworking(serving, &req) {
		result.Resources = append(result.Resources, obj.Object)
	}
	resputil.Success(c, result)
}

// Revalidate a queued or suspended deployment against current administrator policy.
func (mgr *KthenaMgr) validateStoredServing(ctx context.Context, obj *unstructured.Unstructured, token util.JWTMessage) error {
	d, err := kthena.ReadDeployment(obj)
	if err != nil {
		return err
	}
	req := &CreateKthenaReq{BackendType: d.Engine, Layout: d.Layout, Roles: d.Roles}
	if err := validateServingCapability(req); err != nil {
		return err
	}
	return mgr.validateServingSecrets(ctx, req, token)
}

func (mgr *KthenaMgr) prepareServingPreview(
	c *gin.Context,
	req *CreateKthenaReq,
	serving *unstructured.Unstructured,
) (*unstructured.Unstructured, context.Context, error) {
	ctx, token := c.Request.Context(), util.GetToken(c)
	mode := c.DefaultQuery("mode", servingPreviewCreate)
	if mode != servingPreviewCreate && mode != "update" {
		return nil, ctx, bizerr.BadRequest.ParameterError.New("preview mode must be create or update")
	}
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(kthena.ServingGVK)
	lookupErr := mgr.client.Get(ctx, client.ObjectKeyFromObject(serving), existing)
	if lookupErr != nil && !apierrors.IsNotFound(lookupErr) {
		return nil, ctx, lookupErr
	}
	if mode == "update" {
		if lookupErr != nil || !kthena.IsNative(existing) || !canAccessKthenaService(c, existing) {
			return nil, ctx, bizerr.NotFound.K8sResourceNotFound.New("deployment not found")
		}
		serving = existing.DeepCopy()
		if err := mgr.prepareServingUpdate(ctx, serving, req, token); err != nil {
			return nil, ctx, err
		}
		ctx = service.ExcludeServingReservation(ctx, string(existing.GetUID()))
	} else if lookupErr == nil {
		return nil, ctx, bizerr.Conflict.ResourceStatusError.New("deployment name already exists")
	}
	if mode == servingPreviewCreate {
		if err := mgr.checkKthenaScheduling(ctx, token, true); err != nil {
			return nil, ctx, err
		}

		if err := service.CheckWorkloadSubmission(
			ctx, mgr.admissionBans, mgr.billing, mgr.quota, token.UserID, token.AccountID, model.ScheduleTypeNormal,
		); err != nil {
			return nil, ctx, err
		}
	}
	return serving, ctx, nil
}
