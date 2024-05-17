package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type LabelMgr struct {
	KubeClient kubernetes.Interface
}

func NewLabelMgr(kc kubernetes.Interface) Manager {
	return &LabelMgr{
		KubeClient: kc,
	}
}

func (mgr *LabelMgr) RegisterPublic(_ *gin.RouterGroup) {
}

func (mgr *LabelMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.ListLabels)
}

func (mgr *LabelMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.POST("", mgr.CreateLabel)
	g.POST("/sync/nvidia", mgr.SyncNvidiaLabels)
	g.PUT("/:id", mgr.UpdateLabel)
	g.DELETE("/:id", mgr.DeleteLabel)
}

type (
	CreateLabelReq struct {
		Name     string           `json:"name" binding:"required"`
		Priority int              `json:"priority" binding:"required"`
		Type     model.WorkerType `json:"type" binding:"required"`
	}

	UpdateLabelID struct {
		ID int `uri:"id" binding:"required"`
	}
	UpdateLabelReq struct {
		Name     string           `json:"name"`
		Priority int              `json:"priority"`
		Type     model.WorkerType `json:"type"`
	}
	DeleteLabelID struct {
		ID uint `uri:"id" binding:"required"`
	}

	LabelResp struct {
		ID       uint             `json:"id"`
		Label    string           `json:"label"`
		Name     string           `json:"name"`
		Type     model.WorkerType `json:"type"`
		Count    int              `json:"count"`
		Priority int              `json:"priority"`
	}
)

// ListLabels godoc
// @Summary list labels
// @Description show all labels, decs by priority
// @Tags label
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[LabelResp] "label struct"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/labels [get]
func (mgr *LabelMgr) ListLabels(c *gin.Context) {
	label := query.Label
	var labels []LabelResp
	err := label.WithContext(c).Select(label.ID, label.Label, label.Name, label.Type, label.Count, label.Priority).
		Order(label.Priority.Desc()).Scan(&labels)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list labels: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, labels)
}

// CreateLabel godoc
// @Summary 创建标签
// @Description 创建标签
// @Tags label
// @Accept json
// @Produce json
// @Security Bearer
// @Param data body CreateLabelReq true "创建标签"
// @Success 200 {object} resputil.Response[any] "成功返回"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/labels [post]
func (mgr *LabelMgr) CreateLabel(c *gin.Context) {
	var req CreateLabelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to bind request: %v", err), resputil.NotSpecified)
		return
	}
	var workerType model.WorkerType
	switch req.Type {
	case model.Nvidia:
		workerType = model.Nvidia
	default:
		workerType = model.Nvidia
	}

	label := query.Label
	err := label.WithContext(c).Create(&model.Label{Name: req.Name, Priority: req.Priority, Type: workerType})
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create label: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nil)
}

// UpdateLabel godoc
// @Summary 更新标签
// @Description 更新标签
// @Tags label
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path UpdateLabelID true "标签ID"
// @Param data body UpdateLabelReq true "更新标签"
// @Success 200 {object} resputil.Response[any] "成功返回"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/labels/{id} [put]
func (mgr *LabelMgr) UpdateLabel(c *gin.Context) {
	var req UpdateLabelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to bind request: %v", err), resputil.NotSpecified)
		return
	}
	var reqID UpdateLabelID
	if err := c.ShouldBindUri(&reqID); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to bind request: %v", err), resputil.NotSpecified)
		return
	}

	var workerType model.WorkerType
	switch req.Type {
	case model.Nvidia:
		workerType = model.Nvidia
	default:
		workerType = model.Nvidia
	}

	label := query.Label

	_, err := label.WithContext(c).Where(label.ID.Eq(uint(reqID.ID))).Updates(map[string]any{
		"name":     req.Name,
		"priority": req.Priority,
		"type":     workerType,
	})
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to update label: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nil)
}

// DeleteLabel godoc
// @Summary 删除标签
// @Description 根据ID删除标签
// @Tags label
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path DeleteLabelID true "标签ID"
// @Success 200 {object} resputil.Response[any] "成功返回"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/labels/{id} [delete]
func (mgr *LabelMgr) DeleteLabel(c *gin.Context) {
	var req DeleteLabelID
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to bind request: %v", err), resputil.NotSpecified)
		return
	}

	l := query.Label
	if _, err := l.WithContext(c).Where(l.ID.Eq(req.ID)).Delete(); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to delete label: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nil)
}

func (mgr *LabelMgr) SyncNvidiaLabels(c *gin.Context) {
	nodes, err := mgr.KubeClient.CoreV1().Nodes().List(c, metav1.ListOptions{})
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list nodes: %v", err), resputil.NotSpecified)
		return
	}

	// 创建一个 map 用来存储标签值和出现次数
	labelCounts := make(map[string]int)

	// 遍历每个节点
	for i := range nodes.Items {
		node := &nodes.Items[i]
		// 获取节点的标签
		labels := node.GetLabels()
		// 获取节点的 "nvidia.com/gpu.product" 标签的值
		gpuProduct := labels["nvidia.com/gpu.product"]
		// 将标签值及其出现次数记录到 map 中
		labelCounts[gpuProduct]++
	}

	// 如果标签已存在则更新其数量，否则创建新标签
	l := query.Label
	for label, count := range labelCounts {
		info, err := l.WithContext(c).Where(l.Label.Eq(label)).Update(l.Count, count)
		if err != nil {
			resputil.Error(c, fmt.Sprintf("failed to update label: %v", err), resputil.NotSpecified)
			return
		}
		if info.RowsAffected == 0 {
			var newLabel model.Label
			if label == "" {
				newLabel = model.Label{Label: "", Count: count, Type: model.Unknown}
			} else {
				newLabel = model.Label{Label: label, Name: label, Count: count, Type: model.Nvidia, Priority: 1}
			}
			err := l.WithContext(c).Create(&newLabel)
			if err != nil {
				resputil.Error(c, fmt.Sprintf("failed to create label: %v", err), resputil.NotSpecified)
				return
			}
		}
	}

	resputil.Success(c, labelCounts)
}
