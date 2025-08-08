package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/crclient"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewNodeMgr)
}

type NodeMgr struct {
	name       string
	nodeClient *crclient.NodeClient
}

// 接收 URI 中的参数
type NodePodRequest struct {
	Name string `uri:"name" binding:"required"`
}
type NodeTaint struct {
	Name  string `json:"name" binding:"required"`
	Taint string `json:"taint" binding:"required"`
}

func NewNodeMgr(conf *RegisterConfig) Manager {
	return &NodeMgr{
		name: "nodes",
		nodeClient: &crclient.NodeClient{
			Client:           conf.Client,
			KubeClient:       conf.KubeClient,
			PrometheusClient: conf.PrometheusClient,
		},
	}
}

func (mgr *NodeMgr) GetName() string { return mgr.name }

func (mgr *NodeMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *NodeMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.ListNode)
	g.DELETE("/:name/taint", mgr.DeleteNodetaint)
	g.POST("/:name/taint", mgr.AddNodetaint)
	g.PUT("/:name", mgr.UpdateNodeunschedule)
	g.GET("/:name", mgr.GetNode)
	g.GET("/:name/pods", mgr.GetPodsForNode)
	g.GET("/:name/gpu", mgr.ListNodeGPUInfo)
}

func (mgr *NodeMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.ListNode)
	g.GET("/:name/pods", mgr.GetPodsForNode)
	g.GET("/:name/gpu", mgr.ListNodeGPUInfo)
}

// ListNode godoc
//
//	@Summary		获取节点基本信息
//	@Description	kubectl + prometheus获取节点基本信息
//	@Tags			Node
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[string]	"成功返回值描述，注意这里返回Json字符串，swagger无法准确解析"
//	@Failure		400	{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500	{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/nodes [get]
func (mgr *NodeMgr) ListNode(c *gin.Context) {
	klog.Infof("Node List, url: %s", c.Request.URL)
	nodes, err := mgr.nodeClient.ListNodes(c)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list nodes failed, err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nodes)
}

// GetNode godoc
//
//	@Summary		获取节点详细信息
//	@Description	kubectl + prometheus获取节点详细信息
//	@Tags			Node
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string											true	"节点名称"
//	@Success		200		{object}	resputil.Response[crclient.ClusterNodeDetail]	"成功返回值"
//	@Failure		400		{object}	resputil.Response[any]							"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]							"其他错误"
//	@Router			/v1/nodes/{name} [get]
func (mgr *NodeMgr) GetNode(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		klog.Infof("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}
	nodeInfo, err := mgr.nodeClient.GetNode(c, req.Name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("List nodes pods failed, err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nodeInfo)
}

// UpdataNodeunschedule godoc
//
//	@Summary		更新节点调度状态
//	@Description	介绍函数的主要实现逻辑
//	@Tags			接口对应的标签
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string						true	"节点名称"
//	@Success		200		{object}	resputil.Response[string]	"成功返回值"
//	@Failure		400		{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/nodes/{name}  [put]
func (mgr *NodeMgr) UpdateNodeunschedule(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		klog.Infof("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}
	err := mgr.nodeClient.UpdateNodeunschedule(c, req.Name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Update Node Unschedulable failed , err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, fmt.Sprintf("update %s unschedulable ", req.Name))
}

// addNodetaint godoc
//
//	@Summary		添加节点污点
//	@Description	通过nodeclient调用k8s接口添加节点污点
//	@Tags			接口对应的标签
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			req	body		NodeTaint					true	"节点名称+污点"
//	@Success		200	{object}	resputil.Response[string]	"成功返回值描述"
//	@Failure		400	{object}	resputil.Response[any]		"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]		"Other errors"
//	@Router			/v1/nodes/{name}/taint  [post]
//
//nolint:dupl// 重复代码
func (mgr *NodeMgr) AddNodetaint(c *gin.Context) {
	var req NodeTaint
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.Infof("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}

	err := mgr.nodeClient.AddNodetaint(c, req.Name, req.Taint)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Add Node taint failed , err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, fmt.Sprintf("Add %s taint %s ", req.Name, req.Taint))
}

// DeleteNodeTaint godoc
//
//	@Summary		删除节点的污点
//	@Description	匹配是否存在该污点，存在则删除
//	@Tags			接口对应的标签
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			req	body		NodeTaint					true	"节点名称+污点"
//	@Success		200	{object}	resputil.Response[string]	"成功返回值描述"
//	@Failure		400	{object}	resputil.Response[any]		"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]		"Other errors"
//	@Router			/v1/nodes/{name}/taint  [delete]
//
//nolint:dupl// 重复代码
func (mgr *NodeMgr) DeleteNodetaint(c *gin.Context) {
	var req NodeTaint
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.Infof("Delete Bind URI failed, err: %v", err)
		resputil.Error(c, "Delete Invalid request parameter", resputil.NotSpecified)
		return
	}

	err := mgr.nodeClient.DeleteNodetaint(c, req.Name, req.Taint)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Delete Node taint failed , err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, fmt.Sprintf("Delet %s taint %s ", req.Name, req.Taint))
}

// GetPodsForNode godoc
//
//	@Summary		获取节点Pod信息
//	@Description	kubectl + prometheus获取节点Pod信息
//	@Tags			Node
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	query		string					false	"节点名称"
//	@Success		200		{object}	resputil.Response[any]	"成功返回值描述"
//	@Failure		400		{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]	"其他错误"
//	@Router			/v1/nodes/{name}/pod/ [get]
func (mgr *NodeMgr) GetPodsForNode(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		klog.Infof("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}

	klog.Infof("Node List Pod, name: %s", req.Name)
	pods, err := mgr.nodeClient.GetPodsForNode(c, req.Name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("List nodes pods failed, err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, pods)
}

// ListNodeGPUUtil godoc
//
//	@Summary		获取GPU各节点的利用率
//	@Description	查询prometheus获取GPU各节点的利用率
//	@Tags			Node
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	query		string								false	"节点名称"
//	@Success		200		{object}	resputil.Response[crclient.GPUInfo]	"成功返回值描述"
//	@Failure		400		{object}	resputil.Response[any]				"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]				"其他错误"
//	@Router			/v1/nodes/{name}/gpu/ [get]
func (mgr *NodeMgr) ListNodeGPUInfo(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		return
	}

	klog.Infof("List Node GPU Util, name: %s", req.Name)
	gpuInfo, err := mgr.nodeClient.GetNodeGPUInfo(req.Name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get nodes GPU failed, err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, gpuInfo)
}
