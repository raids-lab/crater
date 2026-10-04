package handler

import (
	"context"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/monitor"
	corev1 "k8s.io/api/core/v1"
)

type inferenceMetricsReader interface {
	QueryInferenceMetrics(context.Context, string, string, string, []string) (monitor.InferenceMetrics, error)
}

type KthenaMetricsResp struct {
	Roles           map[string]monitor.InferenceMetrics `json:"roles"`
	Boundary        string                              `json:"boundary"`
	MonitoringError string                              `json:"monitoringError,omitempty"`
	Metrics         monitor.InferenceMetrics            `json:"metrics"`
	Billing         service.ServingBillingSummary       `json:"billing"`
}

// UserKthenaMetrics godoc
// @Summary Read inference throughput, latency, queue and billing metrics
// @Tags kthena
// @Security Bearer
// @Produce json
// @Param name path string true "Inference service name"
// @Success 200 {object} resputil.Response[KthenaMetricsResp]
// @Failure 500 {object} resputil.Response[any]
// @Router /v1/kthena/inference-services/{name}/metrics [get]
func (mgr *KthenaMgr) UserKthenaMetrics(c *gin.Context) {
	obj, ok := mgr.loadKthenaService(c, false)
	if !ok {
		return
	}
	pods, err := mgr.readKthenaPods(c.Request.Context(), obj)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "read inference pods failed"))
		return
	}
	var result KthenaMetricsResp

	deployment, err := kthena.ReadDeployment(obj)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	result.Roles = map[string]monitor.InferenceMetrics{}
	result.Boundary = "Kthena Router"

	if mgr.metrics != nil {
		mgr.collectServingMetrics(c.Request.Context(), obj.GetName(), obj.GetNamespace(), deployment, pods, &result)
	}
	if mgr.billing != nil {
		token := util.GetToken(c)
		result.Billing, err = mgr.billing.ServingBilling(c.Request.Context(),
			obj.GetLabels()[kthena.PodDeploymentLabel], token.UserID, token.AccountID)
		if err != nil {
			resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "read inference billing failed"))
			return
		}
	}
	resputil.Success(c, result)
}

func (mgr *KthenaMgr) collectServingMetrics(
	ctx context.Context,
	name, namespace string,
	deployment *kthena.Deployment,
	pods []corev1.Pod,
	result *KthenaMetricsResp,
) {
	const metricsTimeout = 8 * time.Second
	ctx, cancel := context.WithTimeout(ctx, metricsTimeout)
	defer cancel()
	snapshots := make([]monitor.InferenceMetrics, len(deployment.Roles)+1)
	failures := make([]error, len(snapshots))
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		snapshots[0], failures[0] = mgr.metrics.QueryInferenceMetrics(
			ctx,
			kthenaNamespace,
			deployment.Engine,
			"router",
			[]string{name},
		)
	}()
	for i := range deployment.Roles {
		role := &deployment.Roles[i]
		names := []string{}
		for j := range pods {
			pod := &pods[j]
			if pod.Labels[kthena.EntryLabel] == kthena.EntryValue && pod.Labels[kthena.RoleLabel] == role.Name {
				names = append(names, pod.Name)
			}
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			snapshots[i+1], failures[i+1] = mgr.metrics.QueryInferenceMetrics(ctx, namespace, deployment.Engine, role.Name, names)
		}()
	}
	wg.Wait()
	result.Metrics = snapshots[0]
	for _, failure := range failures {
		if failure != nil {
			result.MonitoringError = "inference monitoring unavailable"
		}
	}
	for i := range deployment.Roles {
		role := &deployment.Roles[i]
		metrics := snapshots[i+1]
		result.Roles[role.Name] = metrics
		if role.Name == "server" || role.Name == "decode" {
			result.Metrics.TokensPerSecond = metrics.TokensPerSecond
		}
		if role.Name == "server" {
			result.Metrics.TTFTP95Seconds = metrics.TTFTP95Seconds
			result.Metrics.WaitingRequests = metrics.WaitingRequests
			result.Metrics.RunningRequests = metrics.RunningRequests
		}
	}
}
