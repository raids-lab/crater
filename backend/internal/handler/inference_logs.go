package handler

import (
	"context"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/resputil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

// UserServingLogs godoc
// @Summary Read bounded logs from a Pod belonging to this deployment
// @Tags kthena
// @Security Bearer
// @Produce json
// @Param name path string true "Deployment"
// @Param pod query string true "Pod name"
// @Param container query string false "Container name"
// @Success 200 {object} resputil.Response[string]
// @Router /v1/kthena/inference-services/{name}/logs [get]
func (mgr *KthenaMgr) UserServingLogs(c *gin.Context) {
	obj, ok := mgr.loadKthenaService(c, false)
	if !ok {
		return
	}
	pods, err := mgr.readKthenaPods(c.Request.Context(), obj)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	container := c.DefaultQuery("container", kthena.EngineContainer)
	if !servingLogContainerExists(pods, c.Query("pod"), container) {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("Pod or container does not belong to this deployment"))
		return
	}
	const logTailLines int64 = 200
	const logTimeout = 8 * time.Second
	const logLimit = 256 << 10
	ctx, cancel := context.WithTimeout(c.Request.Context(), logTimeout)
	defer cancel()
	options := &corev1.PodLogOptions{Container: container, TailLines: ptr.To(logTailLines), LimitBytes: ptr.To(int64(logLimit))}
	stream, err := mgr.kubeClient.CoreV1().Pods(obj.GetNamespace()).GetLogs(c.Query("pod"), options).Stream(ctx)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "read pod logs"))
		return
	}
	defer stream.Close()
	data, err := io.ReadAll(io.LimitReader(stream, logLimit))
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, string(data))
}
func servingLogContainerExists(pods []corev1.Pod, name, container string) bool {
	for i := range pods {
		p := &pods[i]
		if p.Name != name {
			continue
		}
		for j := range p.Spec.Containers {
			if p.Spec.Containers[j].Name == container {
				return true
			}
		}
		for j := range p.Spec.InitContainers {
			if p.Spec.InitContainers[j].Name == container {
				return true
			}
		}
	}
	return false
}
