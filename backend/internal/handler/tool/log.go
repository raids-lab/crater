package tool

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

// 实现流式日志函数
func (mgr *APIServerMgr) StreamPodContainerLog(c *gin.Context) {
	var req PodContainerLogURIReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var param PodContainerLogQueryReq
	if err := c.ShouldBindQuery(&param); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	if token.RolePlatform != model.RoleAdmin && !strings.Contains(req.PodName, token.Username) {
		resputil.Error(c, "user not allowed to visit pod log", resputil.UserNotAllowed)
		return
	}

	// 设置流式响应头
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// 创建日志请求，强制设置Follow为true
	logReq := mgr.kubeClient.CoreV1().Pods(req.Namespace).GetLogs(req.PodName, &v1.PodLogOptions{
		Container:  req.ContainerName,
		Follow:     true, // 强制为true
		Timestamps: param.Timestamps,
		TailLines:  param.TailLines,
	})

	// 使用流式方式获取日志
	stream, err := logReq.Stream(c)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to get log stream: %v", err), resputil.NotSpecified)
		return
	}
	defer stream.Close()

	// 设置连接关闭检测
	ctx := c.Request.Context()
	go func() {
		<-ctx.Done()
		stream.Close()
	}()

	// 读取并发送日志
	reader := bufio.NewReader(stream)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				// 发送错误信息（可选）
				fmt.Fprintf(c.Writer, "ERROR: %v\n", err)
			}
			break
		}

		// 将日志行编码为base64并添加换行符，方便前端解析
		encoded := base64.StdEncoding.EncodeToString(line)
		_, _ = c.Writer.WriteString(encoded + "\n")
		c.Writer.Flush()
	}
}

// GetPodContainerLog godoc
//
//	@Summary		获取Pod容器日志
//	@Description	获取Pod容器日志
//	@Tags			Pod
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			namespace	path		string					true	"命名空间"
//	@Param			name		path		string					true	"Pod名称"
//	@Param			container	path		string					true	"容器名称"
//	@Param			page		query		int						true	"页码"
//	@Param			size		query		int						true	"每页数量"
//	@Success		200			{object}	resputil.Response[any]	"Pod容器日志"
//	@Failure		400			{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500			{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/namespaces/{namespace}/pods/{name}/containers/{container}/log [get]
func (mgr *APIServerMgr) GetPodContainerLog(c *gin.Context) {
	// Implementation for fetching and returning the pod container log
	var req PodContainerLogURIReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	var param PodContainerLogQueryReq
	if err := c.ShouldBindQuery(&param); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// 获取指定 Pod 的日志请求
	logReq := mgr.kubeClient.CoreV1().Pods(req.Namespace).GetLogs(req.PodName, &v1.PodLogOptions{
		Container:  req.ContainerName,
		Follow:     param.Follow,
		TailLines:  param.TailLines,
		Timestamps: param.Timestamps,
		Previous:   param.Previous,
	})

	// 获取日志内容
	logData, err := logReq.DoRaw(c)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to get log: %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, logData)
}
