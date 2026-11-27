package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/resputil"
)

// KthenaDeploymentOverview is the cluster-visible summary used by the portal.
type KthenaDeploymentOverview struct {
	Name              string              `json:"name"`
	Model             string              `json:"model"`
	AccountName       string              `json:"accountName"`
	Phase             string              `json:"phase"`
	ReadyGroups       *int64              `json:"readyGroups"`
	DesiredGroups     int64               `json:"desiredGroups"`
	Resources         corev1.ResourceList `json:"resources"`
	RoutingConfigured bool                `json:"routingConfigured"`
	CanOpenDetail     bool                `json:"canOpenDetail"`
	CreatedAt         time.Time           `json:"createdAt"`
}

// UserListKthenaDeploymentOverview godoc
// @Summary List current model deployments for the platform overview
// @Description Returns cluster-visible summaries of existing deployments, including paused ones, without a creation-date filter.
// @Tags kthena
// @Security Bearer
// @Produce json
// @Success 200 {object} resputil.Response[[]KthenaDeploymentOverview]
// @Failure 500 {object} resputil.Response[any]
// @Router /v1/kthena/deployment-overview [get]
func (mgr *KthenaMgr) UserListKthenaDeploymentOverview(c *gin.Context) {
	ctx := c.Request.Context()
	managed := client.MatchingLabels{inferenceServiceLabelManagedBy: inferenceServiceManagedByValue}
	deployments, err := kthena.ListDeployments(ctx, mgr.client, mgr.namespace, managed)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "list model deployments failed"))
		return
	}
	result := make([]KthenaDeploymentOverview, 0, len(deployments))
	if len(deployments) == 0 {
		resputil.Success(c, result)
		return
	}

	runtimePods, err := mgr.listKthenaRuntimePods(ctx, deployments)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "list model deployment pods failed"))
		return
	}
	servers, err := mgr.kthenaOverviewRoutingObjects(ctx, kthena.ServerGVK, managed)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "list model servers failed"))
		return
	}
	routes, err := mgr.kthenaOverviewRoutingObjects(ctx, kthena.RouteGVK, managed)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "list model routes failed"))
		return
	}
	accounts, err := kthenaOverviewAccountNames(ctx, deployments)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "look up deployment accounts failed"))
		return
	}

	for i := range deployments {
		serving := &deployments[i]
		name := serving.GetName()
		row := KthenaDeploymentOverview{
			Name: name, AccountName: accounts[serving.GetLabels()[inferenceServiceLabelAccountID]],
			Phase:             kthena.Phase(serving),
			DesiredGroups:     desiredServingReplicas(serving),
			RoutingConfigured: kthenaOverviewRoutingConfigured(serving, servers[name], routes[name]),
			CanOpenDetail:     canAccessKthenaService(c, serving),
			CreatedAt:         serving.GetCreationTimestamp().Time,
		}
		deployment, err := kthena.ReadDeployment(serving)
		if err != nil {
			row.Phase = "Invalid"
		} else {
			row.Model, row.Resources = deployment.ServedModel, deployment.TotalResources()
			pods := runtimePods[name]
			_, ready, active := servingRuntimeGroups(deployment.Roles, pods)
			row.ReadyGroups = &ready
			row.Phase = observedServingPhase(row.Phase, row.DesiredGroups, ready, active, pods)
		}
		result = append(result, row)
	}
	resputil.Success(c, result)
}

func (mgr *KthenaMgr) kthenaOverviewRoutingObjects(
	ctx context.Context, gvk schema.GroupVersionKind, managed client.MatchingLabels,
) (map[string]*unstructured.Unstructured, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
	if err := mgr.client.List(ctx, list, client.InNamespace(mgr.namespace), managed); err != nil {
		return nil, err
	}
	byName := make(map[string]*unstructured.Unstructured, len(list.Items))
	for i := range list.Items {
		byName[list.Items[i].GetName()] = &list.Items[i]
	}
	return byName, nil
}

func kthenaOverviewRoutingConfigured(serving, server, route *unstructured.Unstructured) bool {
	if !isRelatedKthenaObject(server, serving) || !isRelatedKthenaObject(route, serving) {
		return false
	}
	model, _, _ := unstructured.NestedString(route.Object, "spec", "modelName")
	if model != serving.GetName() {
		return false
	}
	rules, _, _ := unstructured.NestedSlice(route.Object, "spec", "rules")
	for _, rule := range rules {
		fields, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		targets, _, _ := unstructured.NestedSlice(fields, "targetModels")
		for _, target := range targets {
			if fields, ok := target.(map[string]any); ok && fields["modelServerName"] == server.GetName() {
				return true
			}
		}
	}
	return false
}

func kthenaOverviewAccountNames(ctx context.Context, deployments []unstructured.Unstructured) (map[string]string, error) {
	ids := make([]uint, 0, len(deployments))
	for i := range deployments {
		if id, err := strconv.ParseUint(deployments[i].GetLabels()[inferenceServiceLabelAccountID], 10, strconv.IntSize); err == nil {
			ids = append(ids, uint(id))
		}
	}
	names := make(map[string]string)
	if len(ids) == 0 {
		return names, nil
	}
	account := query.Account
	rows, err := account.WithContext(ctx).Select(account.ID, account.Name, account.Nickname).Where(account.ID.In(ids...)).Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		name := row.Nickname
		if name == "" {
			name = row.Name
		}
		names[strconv.FormatUint(uint64(row.ID), 10)] = name
	}
	return names, nil
}
