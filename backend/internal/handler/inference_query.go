package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

func (mgr *KthenaMgr) listKthenaServices(
	ctx context.Context,
	labels client.MatchingLabels,
) ([]KthenaServiceResp, error) {
	items, err := kthena.ListDeployments(ctx, mgr.client, mgr.namespace, labels)
	if err != nil {
		return nil, err
	}

	services := make([]KthenaServiceResp, 0, len(items))
	for i := range items {
		resp, err := inferenceServiceSummary(&items[i])
		if err != nil {
			resp = &KthenaServiceResp{Name: items[i].GetName(),
				Namespace: items[i].GetNamespace(),
				Phase:     "Invalid",
				Diagnostics: []KthenaDiagnostic{{Level: "error",
					Reason:  "InvalidSpecification",
					Message: "Deployment configuration is invalid; inspect its resources"}}}
		}
		services = append(services, *resp)
	}
	return services, nil
}

func (mgr *KthenaMgr) getKthenaService(c *gin.Context, admin, raw bool) {
	obj, ok := mgr.loadKthenaService(c, admin)
	if !ok {
		return
	}
	if raw {
		raw, err := mgr.rawKthenaResources(c.Request.Context(), obj)
		if err != nil {
			resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "read inference resources failed"))
			return
		}
		resputil.Success(c, raw)
		return
	}
	resp, err := mgr.inferenceServiceToResp(c.Request.Context(), obj)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(err, "failed to parse inference service"))
		return
	}
	resputil.Success(c, resp)
}

func (mgr *KthenaMgr) deleteKthenaService(c *gin.Context, admin bool) {
	obj, ok := mgr.loadKthenaService(c, admin)
	if !ok {
		return
	}
	uid := obj.GetUID()
	if err := mgr.client.Delete(c.Request.Context(), obj,
		client.Preconditions{UID: &uid}, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
		klog.Errorf("delete inference service failed: %v", err)
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "delete inference service failed"))
		return
	}
	resputil.Success(c, "inference service deleted")
}

func (mgr *KthenaMgr) loadKthenaService(c *gin.Context, admin bool) (*unstructured.Unstructured, bool) {
	var req struct {
		Name string `uri:"name" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid service name"))
		return nil, false
	}

	obj, err := kthena.GetDeployment(c.Request.Context(), mgr.client, client.ObjectKey{Namespace: mgr.namespace, Name: req.Name})
	if err != nil {
		if errors.IsNotFound(err) {
			resputil.HandleError(c, bizerr.NotFound.K8sResourceNotFound.Wrap(err, "inference service not found"))
			return nil, false
		}
		klog.Errorf("get inference service failed: %v", err)
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "get inference service failed"))
		return nil, false
	}
	if !admin && !canAccessKthenaService(c, obj) {
		resputil.HandleError(c, bizerr.NotFound.K8sResourceNotFound.New("inference service not found"))
		return nil, false
	}

	return obj, true
}

func canAccessKthenaService(c *gin.Context, obj *unstructured.Unstructured) bool {
	token := util.GetToken(c)
	labels := obj.GetLabels()
	return labels[inferenceServiceLabelUserID] == strconv.FormatUint(uint64(token.UserID), 10) &&
		labels[inferenceServiceLabelAccountID] == strconv.FormatUint(uint64(token.AccountID), 10)
}

func (mgr *KthenaMgr) inferenceServiceToResp(ctx context.Context, obj *unstructured.Unstructured) (*KthenaServiceResp, error) {
	response, err := inferenceServiceSummary(obj)
	if err != nil {
		return nil, err
	}
	resources, err := mgr.readKthenaResources(ctx, obj)
	if err != nil {
		return nil, err
	}
	pods, err := mgr.readKthenaPods(ctx, obj)
	if err != nil {
		return nil, err
	}
	response.Resources = resources
	for i := range pods {
		response.RuntimePods = append(response.RuntimePods, kthenaRuntimePodFromPod(&pods[i]))
	}
	response.Access = mgr.buildKthenaAccess(ctx, obj, resources)
	if response.Access.RouteName == "" || response.Access.ServerName == "" {
		response.Phase = kthenaPhaseProgressing
		response.Diagnostics = []KthenaDiagnostic{{Level: "warning",
			Reason:  "RoutingResourcesMissing",
			Message: "Routing resources are incomplete; retry deployment repair"}}
	}
	response.Phase = runtimeAwareInferencePhase(response.Phase, response.Roles, response.Replicas, response.RuntimePods)
	return response, nil
}

func inferenceServiceSummary(obj *unstructured.Unstructured) (*KthenaServiceResp, error) {
	deployment, err := kthena.ReadDeployment(obj)
	if err != nil {
		return nil, err
	}
	if queued := queuedServingReplicaCount(obj); queued > 0 {
		deployment.Replicas = queued
	}
	conditions, _ := normalizeConditions(obj)
	phase := kthena.Phase(obj)
	owner, userInfo := kthenaServiceOwner(obj)

	return &KthenaServiceResp{
		Name:            obj.GetName(),
		Namespace:       obj.GetNamespace(),
		Owner:           owner,
		UserInfo:        userInfo,
		ModelSource:     modelSourceFromObject(obj),
		PlatformModelID: platformModelIDFromObject(obj),
		ModelURI:        deployment.ModelURI,
		ModelRevision:   deployment.ModelRevision,
		ServedModel:     deployment.ServedModel,
		BackendType:     deployment.Engine,
		CacheURI:        deployment.CacheURI,
		Replicas:        deployment.Replicas,
		DesiredReplicas: desiredServingReplicas(obj),
		Layout:          deployment.Layout, Port: deployment.Port, Roles: deployment.Roles, TotalResources: deployment.TotalResources(),
		Queue:      obj.GetAnnotations()["scheduling.volcano.sh/queue-name"],
		Phase:      phase,
		Conditions: conditions,

		Labels:            obj.GetLabels(),
		CreationTimestamp: obj.GetCreationTimestamp().Time,
	}, nil
}

// kthenaServiceOwner reads the creator identity persisted on the ModelServing
// at creation time. Keeping this data on the Kubernetes object lets a list
// response expose UserInfo without an additional database lookup for each
// deployment.
func kthenaServiceOwner(obj *unstructured.Unstructured) (string, model.UserInfo) {
	if obj == nil {
		return "", model.UserInfo{}
	}
	username := strings.TrimSpace(obj.GetAnnotations()[inferenceServiceAnnotationUsername])
	return username, model.UserInfo{Username: username}
}

func isRelatedKthenaObject(obj, serving *unstructured.Unstructured) bool {
	if obj == nil || serving == nil || obj.GetNamespace() != serving.GetNamespace() {
		return false
	}
	for _, owner := range obj.GetOwnerReferences() {
		if owner.Kind == serving.GetKind() && owner.APIVersion == serving.GetAPIVersion() && owner.Name == serving.GetName() &&
			owner.UID != "" && owner.UID == serving.GetUID() {
			return true
		}
	}
	return false
}

func kthenaResourceFromObject(kind string, obj *unstructured.Unstructured) KthenaResource {
	conditions, _ := normalizeConditions(obj)
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	if nativePhase := kthena.Phase(obj); nativePhase != "" {
		phase = nativePhase
	}
	if phase == "" {
		phase = conditionPhase(conditions)
	}
	return KthenaResource{
		Kind:       kind,
		Name:       obj.GetName(),
		Namespace:  obj.GetNamespace(),
		Phase:      phase,
		Ready:      phase == kthenaPhaseReady || phase == kthenaPhaseActive,
		Conditions: conditions,
	}
}

func (mgr *KthenaMgr) buildKthenaAccess(
	ctx context.Context,
	serving *unstructured.Unstructured,
	resources []KthenaResource,
) KthenaAccess {
	access := KthenaAccess{
		ModelName:       serving.GetName(),
		ProxyBaseURL:    fmt.Sprintf("/v1/kthena/inference-services/%s/%s/v1", serving.GetName(), kthenaProxyPrefix),
		InternalBaseURL: fmt.Sprintf("http://%s.%s.svc.cluster.local/v1", kthenaRouterService, kthenaNamespace),
		RouterService:   fmt.Sprintf("%s/%s", kthenaNamespace, kthenaRouterService),
	}
	for _, resource := range resources {
		switch resource.Kind {
		case kthenaKindModelRoute:
			access.RouteName = resource.Name
		case kthenaKindModelServer:
			access.ServerName = resource.Name
		}
	}
	if mgr.kubeClient != nil {
		if svc, err := mgr.kubeClient.CoreV1().Services(kthenaNamespace).Get(ctx, kthenaRouterService, metav1.GetOptions{}); err == nil {
			access.NodePortURL = nodePortURL(svc)
		}
	}
	return access
}

func normalizeConditions(obj *unstructured.Unstructured) ([]map[string]any, error) {
	raw, ok, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !ok {
		return []map[string]any{}, err
	}
	conditions := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if condition, ok := item.(map[string]any); ok {
			conditions = append(conditions, condition)
		}
	}
	return conditions, nil
}

func conditionPhase(conditions []map[string]any) string {
	for _, condition := range conditions {
		if stringValue(condition[kthenaSpecTypeKey]) == kthenaPhaseActive && stringValue(condition["status"]) == "True" {
			return kthenaPhaseReady
		}
	}
	if len(conditions) > 0 {
		return kthenaPhaseProgressing
	}
	return kthenaPhasePending
}

func servedModelFromDeployment(obj *unstructured.Unstructured) string {
	deployment, err := kthena.ReadDeployment(obj)
	if err != nil {
		return ""
	}
	return nonEmpty(deployment.ServedModel, inferServedModelName(deployment.ModelURI))
}

func modelSourceFromObject(obj *unstructured.Unstructured) string {
	source := obj.GetAnnotations()[inferenceServiceAnnotationSource]
	if source == "" {
		return inferenceModelSourceExternal
	}
	return source
}

func platformModelIDFromObject(obj *unstructured.Unstructured) uint {
	value := obj.GetAnnotations()[inferenceServiceAnnotationModelID]
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return uint(parsed)
}

func nodePortURL(svc *corev1.Service) string {
	if svc == nil || svc.Spec.Type != corev1.ServiceTypeLoadBalancer && svc.Spec.Type != corev1.ServiceTypeNodePort {
		return ""
	}
	for _, port := range svc.Spec.Ports {
		if port.NodePort <= 0 {
			continue
		}
		nodeHost := firstExternalOrInternalIP(svc)
		if nodeHost == "" {
			return fmt.Sprintf("http://<node-ip>:%d/v1", port.NodePort)
		}
		return fmt.Sprintf("http://%s:%d/v1", nodeHost, port.NodePort)
	}
	return ""
}

func firstExternalOrInternalIP(svc *corev1.Service) string {
	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			return ingress.IP
		}
		if ingress.Hostname != "" {
			return ingress.Hostname
		}
	}
	return ""
}

func desiredServingReplicas(obj *unstructured.Unstructured) int64 {
	for _, key := range []string{servingPausedAnnotation, kthena.QueuedReplicasAnnotation} {
		if raw := obj.GetAnnotations()[key]; raw != "" {
			n, _ := strconv.ParseInt(raw, 10, 64)
			return n
		}
	}
	n, _, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	return n
}
