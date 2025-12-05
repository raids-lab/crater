package handler

import (
	"context"
	"fmt"
	"sort"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
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

// Node Mark 相关的类型定义
type NodeLabel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type NodeAnnotation struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type NodeTaint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
	// 操作原因
	Reason string `json:"reason"`
}

// 新增结构体用于禁止调度请求
type NodeScheduleRequest struct {
	Reason string `json:"reason"` // 操作原因
}

type NodeMark struct {
	Labels      []NodeLabel      `json:"labels"`
	Annotations []NodeAnnotation `json:"annotations"`
	Taints      []NodeTaint      `json:"taints"`
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
	g.PUT("/:name", mgr.UpdateNodeunschedule)
	g.GET("/:name", mgr.GetNode)
	g.GET("/:name/pods", mgr.GetPodsForNode)
	g.GET("/:name/gpu", mgr.ListNodeGPUInfo)
}

func (mgr *NodeMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.ListNode)
	g.GET("/:name/pods", mgr.AdminGetPodsForNode)
	g.GET("/:name/gpu", mgr.ListNodeGPUInfo)
	g.GET("/:name/mark", mgr.GetNodeMarks)
	g.POST("/:name/label", mgr.AddNodeLabel)
	g.DELETE("/:name/label", mgr.DeleteNodeLabel)
	g.POST("/:name/annotation", mgr.AddNodeAnnotation)
	g.DELETE("/:name/annotation", mgr.DeleteNodeAnnotation)
	g.POST("/:name/taint", mgr.AddNodeTaint)
	g.DELETE("/:name/taint", mgr.DeleteNodeTaint)
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
//	@Param			name	path		string	true	"节点名称"
//	@Param			body	body		NodeScheduleRequest true	"请求体，包含 reason 字段"
//	@Success		200		{object}	resputil.Response[string]	"成功返回值"
//	@Failure		400		{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/nodes/{name}  [put]
func (mgr *NodeMgr) UpdateNodeunschedule(c *gin.Context) {
	var urlReq NodePodRequest
	if err := c.ShouldBindUri(&urlReq); err != nil {
		klog.Infof("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}
	var bodyReq NodeScheduleRequest
	// 绑定 JSON body 获取 reason 字段
	if err := c.ShouldBindJSON(&bodyReq); err != nil {
		klog.Infof("Bind JSON failed, err: %v", err)
		resputil.Error(c, "Invalid request body", resputil.NotSpecified)
		return
	}

	// 从 token 中获取用户名
	token := util.GetToken(c)

	err := mgr.nodeClient.UpdateNodeunschedule(c, urlReq.Name, bodyReq.Reason, token.Username)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Update Node Unschedulable failed , err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, fmt.Sprintf("update %s unschedulable ", urlReq.Name))
}

// handleGetPods 通用的 Pod 获取处理函数
func (mgr *NodeMgr) handleGetPods(c *gin.Context, fetcher func(context.Context, string) ([]crclient.Pod, error), actionLog string) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		klog.Infof("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}

	klog.Infof("%s, name: %s", actionLog, req.Name)
	pods, err := fetcher(c, req.Name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("List nodes pods failed, err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, pods)
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
	mgr.handleGetPods(c, mgr.nodeClient.GetPodsForNode, "Node List Pod")
}

// AdminGetPodsForNode godoc
//
//	@Summary		获取节点Pod信息（管理员）
//	@Description	获取节点Pod信息，包含作业和用户信息
//	@Tags			Node
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string					true	"节点名称"
//	@Success		200		{object}	resputil.Response[[]crclient.Pod]	"成功返回值描述"
//	@Failure		400		{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]	"其他错误"
//	@Router			/admin/nodes/{name}/pods [get]
func (mgr *NodeMgr) AdminGetPodsForNode(c *gin.Context) {
	mgr.handleGetPods(c, mgr.nodeClient.AdminGetPodsForNode, "Admin Node List Pod")
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
	gpuInfo, err := mgr.nodeClient.GetNodeGPUInfo(c.Request.Context(), req.Name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get nodes GPU failed, err %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, gpuInfo)
}

// GetNodeMarks godoc
//
//	@Summary		获取节点标记信息
//	@Description	获取指定节点的Labels、Annotations和Taints信息
//	@Tags			Node
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string									true	"节点名称"
//	@Success		200		{object}	resputil.Response[NodeMark]	"成功返回节点标记信息"
//	@Failure		400		{object}	resputil.Response[any]				"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]				"其他错误"
//	@Router			/v1/nodes/{name}/mark [get]
func (mgr *NodeMgr) GetNodeMarks(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		klog.Errorf("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}

	klog.Infof("Get Node Marks, name: %s", req.Name)
	nodeMarkInfo, err := mgr.nodeClient.GetNodeMarks(c, req.Name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Get node marks failed, err %v", err), resputil.NotSpecified)
		return
	}

	// 转换为前端需要的格式
	var labels []NodeLabel
	for key, value := range nodeMarkInfo.Labels {
		labels = append(labels, NodeLabel{
			Key:   key,
			Value: value,
		})
	}

	var annotations []NodeAnnotation
	for key, value := range nodeMarkInfo.Annotations {
		annotations = append(annotations, NodeAnnotation{
			Key:   key,
			Value: value,
		})
	}

	var taints []NodeTaint
	for _, taint := range nodeMarkInfo.Taints {
		taints = append(taints, NodeTaint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: string(taint.Effect),
		})
	}

	// 按照key升序排序
	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Key < labels[j].Key
	})

	sort.Slice(annotations, func(i, j int) bool {
		return annotations[i].Key < annotations[j].Key
	})

	sort.Slice(taints, func(i, j int) bool {
		return taints[i].Key < taints[j].Key
	})

	result := NodeMark{
		Labels:      labels,
		Annotations: annotations,
		Taints:      taints,
	}

	resputil.Success(c, result)
}

// AddNodeLabel godoc
//
//	@Summary		添加节点标签
//	@Description	为指定节点添加标签
//	@Tags			Node
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string						true	"节点名称"
//	@Param			data	body		NodeLabel					true	"标签信息"
//	@Success		200		{object}	resputil.Response[string]	"成功添加标签"
//	@Failure		400		{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/nodes/{name}/label [post]
//
//nolint:dupl // ignore duplicate code
func (mgr *NodeMgr) AddNodeLabel(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		klog.Errorf("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}

	var labelReq NodeLabel
	if err := c.ShouldBindJSON(&labelReq); err != nil {
		klog.Errorf("Bind JSON failed, err: %v", err)
		resputil.Error(c, "Invalid request body", resputil.NotSpecified)
		return
	}

	klog.Infof("Add Node Label, name: %s, key: %s, value: %s", req.Name, labelReq.Key, labelReq.Value)
	err := mgr.nodeClient.AddNodeLabel(c, req.Name, labelReq.Key, labelReq.Value)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Add node label failed, err %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("Add label %s=%s to node %s successfully", labelReq.Key, labelReq.Value, req.Name))
}

// DeleteNodeLabel godoc
//
//	@Summary		删除节点标签
//	@Description	删除指定节点的标签
//	@Tags			Node
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string						true	"节点名称"
//	@Param			data	body		NodeLabel					true	"标签信息（只需要key）"
//	@Success		200		{object}	resputil.Response[string]	"成功删除标签"
//	@Failure		400		{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/nodes/{name}/label [delete]
func (mgr *NodeMgr) DeleteNodeLabel(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		klog.Errorf("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}

	var labelReq NodeLabel
	if err := c.ShouldBindJSON(&labelReq); err != nil {
		klog.Errorf("Bind JSON failed, err: %v", err)
		resputil.Error(c, "Invalid request body", resputil.NotSpecified)
		return
	}

	klog.Infof("Delete Node Label, name: %s, key: %s", req.Name, labelReq.Key)
	err := mgr.nodeClient.DeleteNodeLabel(c, req.Name, labelReq.Key)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Delete node label failed, err %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("Delete label %s from node %s successfully", labelReq.Key, req.Name))
}

// AddNodeAnnotation godoc
//
//	@Summary		添加节点注解
//	@Description	为指定节点添加注解
//	@Tags			Node
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string						true	"节点名称"
//	@Param			data	body		NodeAnnotation				true	"注解信息"
//	@Success		200		{object}	resputil.Response[string]	"成功添加注解"
//	@Failure		400		{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/nodes/{name}/annotation [post]
//
//nolint:dupl // ignore duplicate code
func (mgr *NodeMgr) AddNodeAnnotation(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		klog.Errorf("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}

	var annotationReq NodeAnnotation
	if err := c.ShouldBindJSON(&annotationReq); err != nil {
		klog.Errorf("Bind JSON failed, err: %v", err)
		resputil.Error(c, "Invalid request body", resputil.NotSpecified)
		return
	}

	klog.Infof("Add Node Annotation, name: %s, key: %s, value: %s", req.Name, annotationReq.Key, annotationReq.Value)
	err := mgr.nodeClient.AddNodeAnnotation(c, req.Name, annotationReq.Key, annotationReq.Value)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Add node annotation failed, err %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("Add annotation %s=%s to node %s successfully", annotationReq.Key, annotationReq.Value, req.Name))
}

// DeleteNodeAnnotation godoc
//
//	@Summary		删除节点注解
//	@Description	删除指定节点的注解
//	@Tags			Node
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string						true	"节点名称"
//	@Param			data	body		NodeAnnotation				true	"注解信息（只需要key）"
//	@Success		200		{object}	resputil.Response[string]	"成功删除注解"
//	@Failure		400		{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/nodes/{name}/annotation [delete]
func (mgr *NodeMgr) DeleteNodeAnnotation(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		klog.Errorf("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}

	var annotationReq NodeAnnotation
	if err := c.ShouldBindJSON(&annotationReq); err != nil {
		klog.Errorf("Bind JSON failed, err: %v", err)
		resputil.Error(c, "Invalid request body", resputil.NotSpecified)
		return
	}

	klog.Infof("Delete Node Annotation, name: %s, key: %s", req.Name, annotationReq.Key)
	err := mgr.nodeClient.DeleteNodeAnnotation(c, req.Name, annotationReq.Key)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Delete node annotation failed, err %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("Delete annotation %s from node %s successfully", annotationReq.Key, req.Name))
}

// AddNodeTaint godoc
//
//	@Summary		添加节点污点
//	@Description	为指定节点添加污点
//	@Tags			Node
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string						true	"节点名称"
//	@Param			data	body		NodeTaint					true	"污点信息"
//	@Success		200		{object}	resputil.Response[string]	"成功添加污点"
//	@Failure		400		{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/nodes/{name}/taint [post]
func (mgr *NodeMgr) AddNodeTaint(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		klog.Errorf("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}

	var taintReq NodeTaint
	if err := c.ShouldBindJSON(&taintReq); err != nil {
		klog.Errorf("Bind JSON failed, err: %v", err)
		resputil.Error(c, "Invalid request body", resputil.NotSpecified)
		return
	}

	// 从 token 中获取用户名
	token := util.GetToken(c)

	err := mgr.nodeClient.AddNodeTaint(c, req.Name, taintReq.Key, taintReq.Value, taintReq.Effect, taintReq.Reason, token.Username)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Add node taint failed, err %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("Add taint %s=%s:%s to node %s successfully", taintReq.Key, taintReq.Value, taintReq.Effect, req.Name))
}

// DeleteNodeTaint godoc
//
//	@Summary		删除节点污点
//	@Description	删除指定节点的污点
//	@Tags			Node
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string						true	"节点名称"
//	@Param			data	body		NodeTaint					true	"污点信息"
//	@Success		200		{object}	resputil.Response[string]	"成功删除污点"
//	@Failure		400		{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]		"其他错误"
//	@Router			/v1/nodes/{name}/taint [delete]
func (mgr *NodeMgr) DeleteNodeTaint(c *gin.Context) {
	var req NodePodRequest
	if err := c.ShouldBindUri(&req); err != nil {
		klog.Errorf("Bind URI failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}

	var taintReq NodeTaint
	if err := c.ShouldBindJSON(&taintReq); err != nil {
		klog.Errorf("Bind JSON failed, err: %v", err)
		resputil.Error(c, "Invalid request body", resputil.NotSpecified)
		return
	}

	klog.Infof("Delete Node Taint, name: %s, key: %s, value: %s, effect: %s", req.Name, taintReq.Key, taintReq.Value, taintReq.Effect)
	err := mgr.nodeClient.DeleteNodeTaint(c, req.Name, taintReq.Key, taintReq.Value, taintReq.Effect)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("Delete node taint failed, err %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, fmt.Sprintf("Delete taint from node %s successfully", req.Name))
}
