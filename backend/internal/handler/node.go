package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/server/payload"
	resputil "github.com/raids-lab/crater/pkg/server/response"
)

type NodeMgr struct {
	nodeClient *crclient.NodeClient
}

func NewNodeMgr(nodeClient *crclient.NodeClient) Manager {
	return &NodeMgr{
		nodeClient: nodeClient,
	}
}

func (mgr *NodeMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *NodeMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.ListNode)
}

func (mgr *NodeMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.ListNode)
	g.GET("/pod/:name", mgr.ListNodePod)
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
// @Router /nodes [get]
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

// ListNodePod godoc
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
// @Router /nodes/pod/{name} [get]
func (mgr *NodeMgr) ListNodePod(c *gin.Context) {
	logutils.Log.Infof("Node List, url: %s", c.Request.URL)
	name := c.Param("name")
	if name == "" {
		resputil.Error(c, "name is empty", resputil.NotSpecified)
		return
	}
	nodes, err := mgr.nodeClient.ListNodesPod(name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list nodes pods failed, err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nodes)
}
