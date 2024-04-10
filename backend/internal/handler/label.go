package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	resputil "github.com/raids-lab/crater/pkg/server/response"
)

type LabelMgr struct {
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
		ID int `uri:"id" binding:"required"`
	}
)

func NewLabelMgr() Manager {
	return &LabelMgr{}
}

func (mgr *LabelMgr) RegisterPublic(_ *gin.RouterGroup) {
}

func (mgr *LabelMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.ListLabels)
}

func (mgr *LabelMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.POST("", mgr.CreateLabel)
	g.PUT("/:id", mgr.UpdateLabel)
	g.DELETE("/:id", mgr.DeleteLabel)
}

// ListLabels godoc
// @Summary list labels
// @Description show all labels, decs by priority
// @Tags label
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[[]model.Label]
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/labels [get]
func (mgr *LabelMgr) ListLabels(c *gin.Context) {
	label := query.Label
	labels, err := label.WithContext(c).Select(label.ID, label.Name, label.Priority, label.Type).Order(label.Priority.Desc()).Find()
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
// @Success 200 {object} resputil.Response[[]model.Label] "成功返回"
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
// @Success 200 {object} resputil.Response[string] "成功返回"
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
// @Success 200 {object} resputil.Response[string] "成功返回"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /v1/admin/labels/{id} [delete]
func (mgr *LabelMgr) DeleteLabel(c *gin.Context) {
	var req DeleteLabelID
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to bind request: %v", err), resputil.NotSpecified)
		return
	}

	label := query.Label
	if _, err := label.WithContext(c).Delete(&model.Label{ID: uint(req.ID)}); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to delete label: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nil)
}
