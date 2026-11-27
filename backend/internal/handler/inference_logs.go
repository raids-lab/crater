package handler

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/resputil"
)

// UserServingLogs godoc
// @Summary Read or follow logs from a Pod belonging to this deployment
// @Description Set follow=true for a plain-text stream with recent log lines and up to two minutes of updates.
// @Description The default returns recent log lines in a JSON response.
// @Tags kthena
// @Security Bearer
// @Produce json,plain
// @Param name path string true "Deployment"
// @Param pod query string true "Pod name"
// @Param container query string false "Container name"
// @Param follow query bool false "Stream new log entries after the recent lines"
// @Success 200 {object} resputil.Response[string] "JSON snapshot schema. Follow mode returns plain text."
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
	const (
		logTailLines = int64(200)
		logLimit     = int64(256 << 10)
	)
	follow, _ := strconv.ParseBool(c.Query("follow"))
	timeout := 8 * time.Second
	options := &corev1.PodLogOptions{Container: container, TailLines: ptr.To(logTailLines), Follow: follow}
	if follow {
		timeout = 2 * time.Minute
	} else {
		options.LimitBytes = ptr.To(logLimit)
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()
	stream, err := mgr.kubeClient.CoreV1().Pods(obj.GetNamespace()).GetLogs(c.Query("pod"), options).Stream(ctx)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "read pod logs"))
		return
	}
	defer stream.Close()
	if follow {
		streamServingLogs(c, stream)
		return
	}
	data, err := io.ReadAll(io.LimitReader(stream, logLimit))
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, string(data))
}

func streamServingLogs(c *gin.Context, stream io.Reader) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "no-store")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Writer.Flush()
	const logChunkSize = 32 << 10
	buffer := make([]byte, logChunkSize)
	c.Stream(func(w io.Writer) bool {
		n, readErr := stream.Read(buffer)
		if n > 0 {
			if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
				return false
			}
		}
		return readErr == nil
	})
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
