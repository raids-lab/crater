package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/server/payload"
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

func NewNodeMgr(conf RegisterConfig) Manager {
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
	g.GET("/:name", mgr.GetNode)
	g.GET("/:name/pods", mgr.GetPodsForNode)
	g.GET("/:name/gpu", mgr.ListNodeGPUUtil)
}

func (mgr *NodeMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.ListNode)
	g.GET("/:name/pods", mgr.GetPodsForNode)
	g.GET("/:name/gpu", mgr.ListNodeGPUUtil)
}

// ListNode godoc
// @Summary 获取节点基本信息
// @Description kubectl + prometheus获取节点基本信息
// @Tags Node
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[string] "成功返回值描述，注意这里返回Json字符串，swagger无法准确解析"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/nodes [get]
func (mgr *NodeMgr) ListNode(c *gin.Context) {
	logutils.Log.Infof("Node List, url: %s", c.Request.URL)
	nodes, err := mgr.nodeClient.ListNodes()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list nodes failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.ListNodeResp{
		Rows: nodes,
	}
	resputil.Success(c, resp)
}

// GetNode godoc
// @Summary 获取节点详细信息
// @Description kubectl + prometheus获取节点详细信息
// @Tags Node
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "节点名称"
// @Success 200 {object} resputil.Response[payload.ClusterNodeDetail] "成功返回值"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/nodes/{name} [get]
func (mgr *NodeMgr) GetNode(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		logutils.Log.Infof("Bind URI failed, err: %v", err)
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

// GetPodsForNode godoc
// @Summary 获取节点Pod信息
// @Description kubectl + prometheus获取节点Pod信息
// @Tags Node
// @Accept json
// @Produce json
// @Security Bearer
// @Param name query string false "节点名称"
// @Success 200 {object} resputil.Response[payload.ClusterNodePodInfo] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/nodes/{name}/pod/ [get]
func (mgr *NodeMgr) GetPodsForNode(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		logutils.Log.Infof("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}

	logutils.Log.Infof("Node List Pod, name: %s", req.Name)
	pods, err := mgr.nodeClient.GetPodsForNode(c, req.Name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("List nodes pods failed, err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, pods)
}

// ListNodeGPUUtil godoc
// @Summary 获取GPU各节点的利用率
// @Description 查询prometheus获取GPU各节点的利用率
// @Tags Node
// @Accept json
// @Produce json
// @Security Bearer
// @Param name query string false "节点名称"
// @Success 200 {object} resputil.Response[payload.GPUInfo] "成功返回值描述"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/nodes/{name}/gpu/ [get]
func (mgr *NodeMgr) ListNodeGPUUtil(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		return
	}

	logutils.Log.Infof("List Node GPU Util, name: %s", req.Name)
	gpuInfo, err := mgr.nodeClient.GetNodeGPUInfo(req.Name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get nodes GPU failed, err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, gpuInfo)
}
