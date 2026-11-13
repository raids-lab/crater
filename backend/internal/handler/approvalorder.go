package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

const autoApprovalReason = "without review, approved by system"

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewApprovalOrderMgr)
}

type ApprovalOrderMgr struct {
	name string
}

func NewApprovalOrderMgr(_ *RegisterConfig) Manager {
	return &ApprovalOrderMgr{
		name: "approvalorder",
	}
}
func (mgr *ApprovalOrderMgr) GetName() string { return mgr.name }

func (mgr *ApprovalOrderMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *ApprovalOrderMgr) RegisterProtected(g *gin.RouterGroup) {
	// RESTful 风格的路由设计
	g.GET("", mgr.GetMyApprovalOrders)               // 获取我的审批工单列表
	g.GET("/:id", mgr.GetApprovalOrder)              // 通过ID获取审批工单详情
	g.POST("", mgr.CreateApprovalOrder)              // 创建审批工单
	g.PUT("/:id", mgr.UpdateApprovalOrder)           // 更新审批工单
	g.DELETE("/:id", mgr.DeleteApprovalOrder)        // 删除审批工单
	g.GET("/name/:name", mgr.GetApprovalOrderByName) // 通过名字获取审批工单
}

func (mgr *ApprovalOrderMgr) RegisterAdmin(g *gin.RouterGroup) {
	// 管理员接口
	g.GET("", mgr.ListAllApprovalOrders)               // 获取所有审批工单
	g.GET("/:id", mgr.GetApprovalOrderAdmin)           // 管理员通过ID获取审批工单详情
	g.PUT("/:id/review", mgr.ReviewApprovalOrderAdmin) // 管理员审核审批工单
	g.PUT("/:id/approve-and-lock", mgr.ApproveJobOrderAdmin)
	g.PUT("/check", mgr.UpdateApprovalOrderByJobStatus) // 管理员检查待审批的作业锁定工单有效性
}

// swagger
//
//	@Summary		检查待审批作业工单有效性
//	@Description	检查数据库中待审批且类型为job的工单，如果对应作业不在运行状态则将工单状态更新为canceled
//	@Tags			approvalorder
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200 {object} resputil.Response[string] "成功返回检查结果消息"
//	@Failure		500 {object} resputil.Response[any] "服务器错误"
//	@Router			/v1/admin/approvalorder/check [put]
func (mgr *ApprovalOrderMgr) UpdateApprovalOrderByJobStatus(c *gin.Context) {
	klog.Infof("Starting to check pending job approval orders")

	// 1. 查询所有待审批且类型为job的工单
	ao := query.ApprovalOrder
	pendingJobOrders, err := ao.WithContext(c).
		Where(ao.Status.Eq(string(model.ApprovalOrderStatusPending))).
		Where(ao.Type.Eq(string(model.ApprovalOrderTypeJob))).
		Find()

	if err != nil {
		klog.Errorf("failed to query pending job approval orders, err: %v", err)
		resputil.Error(c, "failed to query pending job orders", resputil.NotSpecified)
		return
	}

	if len(pendingJobOrders) == 0 {
		klog.Infof("no pending job approval orders found")
		resputil.Success(c, "no pending job approval orders found")
		return
	}

	// 2. 检查每个工单对应的作业状态
	jobDB := query.Job
	var canceledCount int

	for _, order := range pendingJobOrders {
		// 查询同名作业
		job, err := jobDB.WithContext(c).
			Where(jobDB.JobName.Eq(order.Name)).
			First()

		if err != nil {
			// 如果作业不存在，将工单状态更新为取消
			klog.Warningf("job not found for approval order, orderID: %d, jobName: %s, err: %v",
				order.ID, order.Name, err)

			if updateErr := mgr.cancelApprovalOrder(c, order.ID, "Job not found"); updateErr != nil {
				klog.Errorf("failed to cancel approval order %d: %v", order.ID, updateErr)
				continue
			}

			canceledCount++
			continue
		}

		// 检查作业是否在运行状态
		if !mgr.isJobRunning(job.Status) {
			klog.Infof("job is not running, canceling approval order, orderID: %d, jobName: %s, jobStatus: %s",
				order.ID, order.Name, job.Status)

			reason := fmt.Sprintf("Job is not running (status: %s)", job.Status)
			if updateErr := mgr.cancelApprovalOrder(c, order.ID, reason); updateErr != nil {
				klog.Errorf("failed to cancel approval order %d: %v", order.ID, updateErr)
				continue
			}

			canceledCount++
		} else {
			klog.V(2).Infof("job is running, keeping approval order active, orderID: %d, jobName: %s, jobStatus: %s",
				order.ID, order.Name, job.Status)
		}
	}

	klog.Infof("job approval order check completed, checked: %d, canceled: %d",
		len(pendingJobOrders), canceledCount)

	message := fmt.Sprintf("job approval order check completed: checked %d orders, canceled %d orders",
		len(pendingJobOrders), canceledCount)
	resputil.Success(c, message)
}

// cancelApprovalOrder 取消审批工单
func (mgr *ApprovalOrderMgr) cancelApprovalOrder(c *gin.Context, orderID uint, reason string) error {
	ao := query.ApprovalOrder

	// 更新工单状态为取消，并添加备注说明原因
	info, err := ao.WithContext(c).
		Where(ao.ID.Eq(orderID), ao.Status.Eq(string(model.ApprovalOrderStatusPending))).
		Updates(map[string]any{
			"status":       string(model.ApprovalOrderStatusCancelled),
			"review_notes": reason,
		})

	if err != nil {
		return fmt.Errorf("failed to update approval order status: %w", err)
	}
	if info.RowsAffected == 0 {
		return bizerr.Conflict.ResourceStatusError.New("only pending approval orders can be canceled")
	}

	return nil
}

// isJobRunning 判断作业是否在运行状态
func (mgr *ApprovalOrderMgr) isJobRunning(jobStatus batch.JobPhase) bool {
	runningStatuses := []batch.JobPhase{
		batch.Running,
		batch.Pending,
		model.Prequeue,
	}

	for _, status := range runningStatuses {
		if jobStatus == status {
			return true
		}
	}

	return false
}

type ApprovalOrderResp struct {
	ID          uint                       `json:"id"`
	Name        string                     `json:"name"`
	Type        model.ApprovalOrderType    `json:"type"`
	Status      model.ApprovalOrderStatus  `json:"status"`
	Content     model.ApprovalOrderContent `json:"content"`
	ReviewNotes string                     `json:"reviewNotes"`
	CreatedAt   time.Time                  `json:"createdAt"`
	ReviewedAt  *time.Time                 `json:"reviewedAt,omitempty"`

	CreatorID  uint           `json:"creatorID"`
	Creator    model.UserInfo `json:"creator"`
	ReviewerID uint           `json:"reviewerID"`
	Reviewer   model.UserInfo `json:"reviewer"`
}

// swagger
//
//	@Summary		获取我的审批工单
//	@Description	获取当前用户创建的所有审批工单
//	@Tags			approvalorder
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"成功返回工单列表"
//	@Failure		400	{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		500	{object}	resputil.Response[any]	"服务器错误"
//	@Router			/v1/approvalorder [get]
//
// GetMyApprovalOrders 获取当前用户创建的所有审批工单
func (mgr *ApprovalOrderMgr) GetMyApprovalOrders(c *gin.Context) {
	// 1. 获取当前用户信息
	token := util.GetToken(c)
	if token.UserID == 0 {
		resputil.Error(c, "cannot get user id", resputil.NotSpecified)
		return
	}

	// 2. 查询当前用户作为创建人的所有工单
	ao := query.ApprovalOrder
	creatorOrders, err := ao.WithContext(c).
		Preload(ao.Creator).  // 预加载创建人信息
		Preload(ao.Reviewer). // 预加载审批人信息（可能为空）
		Where(ao.CreatorID.Eq(token.UserID)).
		Find()
	if err != nil {
		klog.Errorf("failed to query created approval orders, userID: %d, err: %v", token.UserID, err)
		resputil.Error(c, "failed to get my approval orders", resputil.NotSpecified)
		return
	}

	// 3. 转换为响应格式
	result := convertToApprovalOrderResps(creatorOrders)
	resputil.Success(c, result)
}

// 辅助函数：将 datatypes.JSONType 转换为 ApprovalOrderContent
func unmarshalApprovalOrderContent(content datatypes.JSONType[model.ApprovalOrderContent]) model.ApprovalOrderContent {
	var result model.ApprovalOrderContent
	jsonBytes, err := content.MarshalJSON()
	if err != nil {
		klog.Warningf("failed to marshal approval order content: %v", err)
		return model.ApprovalOrderContent{}
	}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		klog.Warningf("failed to unmarshal approval order content: %v", err)
		return model.ApprovalOrderContent{}
	}
	return result
}

// 新增辅助函数：将 ApprovalOrder 数组转换为 ApprovalOrderResp 数组
func convertToApprovalOrderResps(orders []*model.ApprovalOrder) []ApprovalOrderResp {
	var result []ApprovalOrderResp
	for _, order := range orders {
		result = append(result, convertToApprovalOrderResp(order))
	}
	return result
}

// swagger
//
//	@Summary		展示所有审批订单
//	@Description	展示所有审批订单
//	@Tags			approvalorder
//	@Accept			json
//	@Produce		json
//	@Success		200 {object} resputil.Response[any] "成功返回值描述"
//	@Failure		400	{object} resputil.Response[any]	"请求参数错误"
//	@Failure		500	{object} resputil.Response[any] "服务器错误"
//	@Security		Bearer
//	@Router			/v1/admin/approvalorder [get]
func (mgr *ApprovalOrderMgr) ListAllApprovalOrders(c *gin.Context) {
	klog.Infof("List all approval orders")

	ao := query.ApprovalOrder
	allOrders, err := ao.WithContext(c).
		Preload(ao.Creator).  // 预加载创建人信息
		Preload(ao.Reviewer). // 预加载审批人信息（可能为空）
		Order(ao.CreatedAt.Desc()).
		Find()
	if err != nil {
		klog.Errorf("failed to query all approval orders, err: %v", err)
		resputil.Error(c, "failed to list all approval orders", resputil.NotSpecified)
		return
	}

	result := convertToApprovalOrderResps(allOrders)
	resputil.Success(c, result)
}

type ApprovalOrderreq struct {
	Name           string                    `json:"name" binding:"required"`
	Type           model.ApprovalOrderType   `json:"type" binding:"required"`
	Status         model.ApprovalOrderStatus `json:"status"`
	TypeID         uint                      `json:"approvalorderTypeID"`         // 关联的ID，可能是数据集或任务ID
	Reason         string                    `json:"approvalOrderReason"`         // 审批原因
	ExtensionHours uint                      `json:"approvalOrderExtensionHours"` // 延长小时数
}

// swagger
//
//	@Summary		创建审批工单
//	@Description	创建新的审批工单
//	@Tags			approvalorder
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			body body ApprovalOrderreq true "审批工单信息"
//	@Success		200 {object} resputil.Response[string] "成功返回值描述"
//	@Failure		400 {object} resputil.Response[any] "请求参数错误"
//	@Failure		500 {object} resputil.Response[any] "服务器错误"
//	@Router			/v1/approvalorder [post]
func (mgr *ApprovalOrderMgr) CreateApprovalOrder(c *gin.Context) {
	var req ApprovalOrderreq
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.Errorf("failed to bind request parameters: %v", err)
		resputil.Error(c, "invalid request parameters", resputil.NotSpecified)
		return
	}

	// 1. 获取当前用户信息
	token := util.GetToken(c)
	if token.UserID == 0 {
		resputil.Error(c, "cannot get user id", resputil.NotSpecified)
		return
	}

	if req.Type == model.ApprovalOrderTypeJob {
		// The job is only looked up to reject orders for jobs that do not exist.
		jobDB := query.Job
		_, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(req.Name)).First()
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				resputil.BadRequestError(c, "job not found")
				return
			}
			klog.Errorf("failed to query job for approval order, userID: %d, jobName: %s, err: %v",
				token.UserID, req.Name, err)
			resputil.Error(c, "failed to query target job", resputil.NotSpecified)
			return
		}
	}

	canAutoApprove, err := mgr.checkAutoApprovalEligibility(c, token.UserID, &req)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to check recent orders"))
		return
	}

	order := model.ApprovalOrder{
		Name:   req.Name,
		Type:   req.Type,
		Status: model.ApprovalOrderStatusPending,
		Content: datatypes.NewJSONType(model.ApprovalOrderContent{
			ApprovalOrderTypeID:         req.TypeID,
			ApprovalOrderExtensionHours: req.ExtensionHours,
			ApprovalOrderReason:         req.Reason,
		}),
		CreatorID: token.UserID,
	}
	if canAutoApprove {
		approved, err := createAutoApprovedJobOrder(c, &order)
		if err != nil {
			resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to create auto-approved order"))
			return
		}
		if approved {
			resputil.Success(c, "create approvalorder successfully and auto-approved with job locked")
			return
		}
	}

	if err := query.ApprovalOrder.WithContext(c).Create(&order); err != nil {
		klog.Errorf("failed to create approval order, userID: %d, err: %v", token.UserID, err)
		resputil.Error(c, "failed to create approval order", resputil.NotSpecified)
		return
	}

	resputil.Success(c, "create approvalorder successfully")
}

type UpdateApprovalOrder struct {
	Name           string                    `json:"name" binding:"required"`
	Type           model.ApprovalOrderType   `json:"type" binding:"required"`
	Status         model.ApprovalOrderStatus `json:"status"`
	TypeID         uint                      `json:"approvalorderTypeID"`         // 关联的ID，可能是数据集或任务ID
	Reason         string                    `json:"approvalOrderReason"`         // 审批原因
	ExtensionHours uint                      `json:"approvalOrderExtensionHours"` // 延长小时数
	ReviewerID     uint                      `json:"reviewerID"`                  // 审批人ID
	ReviewNotes    string                    `json:"reviewNotes"`                 // 审批备注
}

type ReviewApprovalOrderReq struct {
	Status      model.ApprovalOrderStatus `json:"status" binding:"required"`
	ReviewNotes string                    `json:"reviewNotes"`
}

func (req ReviewApprovalOrderReq) validate() error {
	switch req.Status {
	case model.ApprovalOrderStatusApproved, model.ApprovalOrderStatusCancelled:
		return nil
	case model.ApprovalOrderStatusRejected:
		if req.ReviewNotes == "" {
			return bizerr.BadRequest.MissingParameter.New("review notes are required when rejecting an order")
		}
		return nil
	default:
		return bizerr.BadRequest.ParameterError.New("invalid review status")
	}
}

type ApprovalOrderIDReq struct {
	ID uint `uri:"id" binding:"required"` // 工单ID
}

// swagger
//
//	@Summary		更新审批工单
//	@Description	更新现有的审批工单
//	@Tags			approvalorder
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			body body UpdateApprovalOrder true "审批工单信息"
//	@Success		200 {object} resputil.Response[string] "成功返回值描述"
//	@Failure		400 {object} resputil.Response[any] "请求参数错误"
//	@Failure		500 {object} resputil.Response[any] "服务器错误"
//	@Router			/v1/approvalorder/{id} [put]
func (mgr *ApprovalOrderMgr) UpdateApprovalOrder(c *gin.Context) {
	var req UpdateApprovalOrder
	var orderID ApprovalOrderIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.Errorf("failed to bind request parameters: %v", err)
		resputil.Error(c, "invalid request parameters", resputil.NotSpecified)
		return
	}
	if err := c.ShouldBindUri(&orderID); err != nil {
		klog.Errorf("failed to bind request parameters: %v", err)
		resputil.Error(c, "invalid request parameters", resputil.NotSpecified)
		return
	}
	// 1. 获取当前用户信息
	token := util.GetToken(c)
	if token.UserID == 0 {
		resputil.Error(c, "cannot get user id", resputil.NotSpecified)
		return
	}

	ao := query.ApprovalOrder
	existingOrder, err := ao.WithContext(c).
		Where(ao.ID.Eq(orderID.ID)).
		First()
	if err != nil {
		klog.Errorf("approval order not found, userID: %d, orderID: %d, err: %v", token.UserID, orderID.ID, err)
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.Wrap(err, "approval order not found"))
		return
	}

	if existingOrder.CreatorID != token.UserID {
		klog.Warningf("permission denied: user %d attempted to update order %d owned by %d",
			token.UserID, orderID.ID, existingOrder.CreatorID)
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("permission denied to update this approval order"))
		return
	}

	if existingOrder.Status != model.ApprovalOrderStatusPending {
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("only pending approval orders can be updated"))
		return
	}

	if req.Status != "" && req.Status != model.ApprovalOrderStatusPending {
		klog.Warningf("permission denied: user %d attempted to change order %d status to %s",
			token.UserID, orderID.ID, req.Status)
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("permission denied to review approval orders"))
		return
	}

	// 2. 更新审批工单
	order := model.ApprovalOrder{
		Name:   req.Name,
		Type:   req.Type,
		Status: model.ApprovalOrderStatusPending,
		Content: datatypes.NewJSONType(model.ApprovalOrderContent{
			ApprovalOrderTypeID:         req.TypeID,
			ApprovalOrderExtensionHours: req.ExtensionHours,
			ApprovalOrderReason:         req.Reason,
		}),
	}

	info, err := ao.WithContext(c).
		Where(ao.ID.Eq(orderID.ID), ao.Status.Eq(string(model.ApprovalOrderStatusPending))).
		Updates(&order)
	if err != nil {
		klog.Errorf("failed to update approval order, userID: %d, err: %v", token.UserID, err)
		resputil.Error(c, "failed to update approval order", resputil.NotSpecified)
		return
	}
	if info.RowsAffected == 0 {
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("only pending approval orders can be updated"))
		return
	}
	klog.Infof("updated approval order successfully, affected rows: %d", info.RowsAffected)
	resputil.Success(c, "update approvalorder successfully")
}

// swagger
//
//	@Summary		管理员审核审批工单
//	@Description	审批数据集工单或拒绝、取消工单。批准作业工单使用 approve-and-lock
//	@Tags			approvalorder
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id		path	int						true	"工单ID"
//	@Param			body	body	ReviewApprovalOrderReq	true	"审核信息"
//	@Success		200		{object}	resputil.Response[string]	"成功返回值描述"
//	@Failure		400		{object}	resputil.Response[any]		"请求参数错误"
//	@Failure		500		{object}	resputil.Response[any]		"服务器错误"
//	@Router			/v1/admin/approvalorder/{id}/review [put]
func (mgr *ApprovalOrderMgr) ReviewApprovalOrderAdmin(c *gin.Context) {
	var req ReviewApprovalOrderReq
	var orderID ApprovalOrderIDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		klog.Errorf("failed to bind review request: %v", err)
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid request parameters"))
		return
	}
	if err := c.ShouldBindUri(&orderID); err != nil {
		klog.Errorf("failed to bind request parameters: %v", err)
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid request parameters"))
		return
	}

	token := util.GetToken(c)
	if token.UserID == 0 {
		resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("cannot get user id"))
		return
	}
	if token.RolePlatform != model.RoleAdmin {
		klog.Warningf("permission denied: user %d (role: %s) attempted to review order %d",
			token.UserID, token.RolePlatform, orderID.ID)
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("permission denied: only admins can review orders"))
		return
	}
	if err := req.validate(); err != nil {
		resputil.HandleError(c, err)
		return
	}

	ao := query.ApprovalOrder
	existingOrder, err := ao.WithContext(c).
		Where(ao.ID.Eq(orderID.ID)).
		First()
	if err != nil {
		klog.Errorf("approval order not found, reviewerID: %d, orderID: %d, err: %v", token.UserID, orderID.ID, err)
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.Wrap(err, "approval order not found"))
		return
	}
	if existingOrder.Status != model.ApprovalOrderStatusPending {
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("only pending approval orders can be reviewed"))
		return
	}

	if existingOrder.Type == model.ApprovalOrderTypeJob && req.Status == model.ApprovalOrderStatusApproved {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("job approvals require approve-and-lock"))
		return
	}

	info, err := ao.WithContext(c).
		Where(ao.ID.Eq(orderID.ID),
			ao.Status.Eq(string(model.ApprovalOrderStatusPending)), ao.Type.Eq(string(existingOrder.Type))).
		Updates(map[string]any{
			"status":       req.Status,
			"reviewer_id":  token.UserID,
			"review_notes": req.ReviewNotes,
		})
	if err != nil {
		klog.Errorf("failed to review approval order, reviewerID: %d, orderID: %d, err: %v",
			token.UserID, orderID.ID, err)
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to review approval order"))
		return
	}

	if info.RowsAffected == 0 {
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("approval order changed before it could be reviewed"))
		return
	}
	klog.Infof("reviewed approval order successfully, reviewerID: %d, orderID: %d, affected rows: %d",
		token.UserID, orderID.ID, info.RowsAffected)
	resputil.Success(c, "review approvalorder successfully")
}

// swagger
//
//	@Summary		删除审批工单
//	@Description	删除指定的审批工单
//	@Tags			approvalorder
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id path int true "工单ID"
//	@Success		200 {object} resputil.Response[string] "成功返回值描述"
//	@Failure		400 {object} resputil.Response[any] "请求参数错误"
//	@Failure		500 {object} resputil.Response[any] "服务器错误"
//	@Router			/v1/approvalorder/{id} [delete]
//
// DeleteApprovalOrder 用户删除指定的审批工单
func (mgr *ApprovalOrderMgr) DeleteApprovalOrder(c *gin.Context) {
	// 1. 获取当前用户信息
	token := util.GetToken(c)

	if token.UserID == 0 {
		resputil.Error(c, "cannot get user id", resputil.NotSpecified)
		return
	}

	// 2. 获取工单ID
	var orderID ApprovalOrderIDReq
	if err := c.ShouldBindUri(&orderID); err != nil {
		klog.Errorf("failed to bind request parameters: %v", err)
		resputil.Error(c, "failed to bind request parameters", resputil.NotSpecified)
		return
	}

	// 3. 首先检查工单是否存在且属于当前用户
	ao := query.ApprovalOrder
	existingOrder, err := ao.WithContext(c).
		Where(ao.ID.Eq(orderID.ID)).
		First()
	if err != nil {
		klog.Errorf("approval order not found, userID: %d, orderID: %d, err: %v", token.UserID, orderID, err)
		resputil.Error(c, "approval order not found", resputil.NotSpecified)
		return
	}

	// 4. 权限检查：只有创建者才能删除自己的工单
	if existingOrder.CreatorID != token.UserID {
		klog.Warningf("user attempted to delete an order not created by themselves, userID: %d, orderID: %d, creatorID: %d",
			token.UserID, orderID, existingOrder.CreatorID)
		resputil.Error(c, "permission denied to delete this approval order", resputil.NotSpecified)
		return
	}
	if existingOrder.Status != model.ApprovalOrderStatusPending {
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("only pending approval orders can be deleted"))
		return
	}

	// 6. 执行删除操作
	result, err := ao.WithContext(c).
		Where(ao.ID.Eq(orderID.ID), ao.Status.Eq(string(model.ApprovalOrderStatusPending))).
		Delete()
	if err != nil {
		klog.Errorf("failed to delete approval order, userID: %d, orderID: %d, err: %v", token.UserID, orderID, err)
		resputil.Error(c, "failed to delete approval order", resputil.NotSpecified)
		return
	}

	// 7. 检查是否真的删除了记录
	if result.RowsAffected == 0 {
		klog.Warningf("delete operation affected no records, userID: %d, orderID: %d", token.UserID, orderID)
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("only pending approval orders can be deleted"))
		return
	}

	klog.Infof("deleted approval order successfully, userID: %d, orderID: %d", token.UserID, orderID)
	resputil.Success(c, "delete approvalorder successfully")
}

// swagger
//
//	@Summary		获取审批工单详情
//	@Description	通过ID获取审批工单详情（仅限创建者）
//	@Tags			approvalorder
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id path int true "工单ID"
//	@Success		200 {object} resputil.Response[ApprovalOrderResp] "成功返回工单详情"
//	@Failure		400 {object} resputil.Response[any] "请求参数错误"
//	@Failure		403 {object} resputil.Response[any] "权限不足"
//	@Failure		404 {object} resputil.Response[any] "工单不存在"
//	@Failure		500 {object} resputil.Response[any] "服务器错误"
//	@Router			/v1/approvalorder/{id} [get]
func (mgr *ApprovalOrderMgr) GetApprovalOrder(c *gin.Context) {
	// 1. 获取当前用户信息
	token := util.GetToken(c)
	if token.UserID == 0 {
		resputil.Error(c, "can not get user ID", resputil.NotSpecified)
		return
	}

	// 2. 获取工单ID
	var orderID ApprovalOrderIDReq
	if err := c.ShouldBindUri(&orderID); err != nil {
		klog.Errorf("failed to bind request parameters: %v", err)
		resputil.Error(c, "invalid request parameters", resputil.NotSpecified)
		return
	}

	// 3. 查询工单详情
	ao := query.ApprovalOrder
	order, err := ao.WithContext(c).
		Preload(ao.Creator).
		Preload(ao.Reviewer).
		Where(ao.ID.Eq(orderID.ID)).
		First()
	if err != nil {
		klog.Errorf("failed to query approval order, userID: %d, orderID: %d, err: %v", token.UserID, orderID, err)
		resputil.Error(c, "approval order not found", resputil.NotSpecified)
		return
	}

	// 4. 权限检查：只有创建者才能查看自己的工单
	if order.CreatorID != token.UserID {
		klog.Warningf("user attempted to view an order not created by themselves, userID: %d, orderID: %d, creatorID: %d",
			token.UserID, orderID.ID, order.CreatorID)
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("permission denied to view this approval order"))
		return
	}

	// 5. 转换为响应格式
	result := convertToApprovalOrderResp(order)
	resputil.Success(c, result)
}

// swagger
//
//	@Summary		管理员获取审批工单详情
//	@Description	管理员通过ID获取任意审批工单详情
//	@Tags			approvalorder
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id path int true "工单ID"
//	@Success		200 {object} resputil.Response[ApprovalOrderResp] "成功返回工单详情"
//	@Failure		400 {object} resputil.Response[any] "请求参数错误"
//	@Failure		404 {object} resputil.Response[any] "工单不存在"
//	@Failure		500 {object} resputil.Response[any] "服务器错误"
//	@Router			/v1/admin/approvalorder/{id} [get]
func (mgr *ApprovalOrderMgr) GetApprovalOrderAdmin(c *gin.Context) {
	// 1. 获取当前用户信息
	token := util.GetToken(c)
	if token.UserID == 0 {
		resputil.Error(c, "can not get user ID", resputil.NotSpecified)
		return
	}

	// 2. 权限检查：只有管理员才能查看
	if token.RolePlatform != model.RoleAdmin {
		klog.Warningf("permission denied: user %d (role: %s) attempted to access admin route",
			token.UserID, token.RolePlatform)
		resputil.Error(c, "permission denied", resputil.NotSpecified)
		return
	}

	// 3. 获取工单ID
	var orderID ApprovalOrderIDReq
	if err := c.ShouldBindUri(&orderID); err != nil {
		klog.Errorf("bind uri error: %v", err)
		resputil.Error(c, "uri error", resputil.NotSpecified)
		return
	}

	// 4. 查询工单详情
	ao := query.ApprovalOrder
	order, err := ao.WithContext(c).
		Preload(ao.Creator).
		Preload(ao.Reviewer).
		Where(ao.ID.Eq(orderID.ID)).
		First()
	if err != nil {
		klog.Errorf("admin failed to query approval order, orderID: %d, err: %v", orderID.ID, err)
		resputil.Error(c, "approval order not found", resputil.NotSpecified)
		return
	}

	// 5. 转换为响应格式
	result := convertToApprovalOrderResp(order)
	resputil.Success(c, result)
}

// 新增辅助函数：将单个 ApprovalOrder 转换为 ApprovalOrderResp
func convertToApprovalOrderResp(order *model.ApprovalOrder) ApprovalOrderResp {
	resp := ApprovalOrderResp{
		ID:          order.ID,
		Name:        order.Name,
		Type:        order.Type,
		Status:      order.Status,
		Content:     unmarshalApprovalOrderContent(order.Content),
		ReviewNotes: order.ReviewNotes,
		CreatedAt:   order.CreatedAt,
		CreatorID:   order.CreatorID,
		Creator: model.UserInfo{
			Username: order.Creator.Name,
			Nickname: order.Creator.Nickname,
		},
		ReviewerID: ptr.Deref(order.ReviewerID, 0),
	}

	if (order.Status == model.ApprovalOrderStatusApproved || order.Status == model.ApprovalOrderStatusRejected) &&
		!order.UpdatedAt.IsZero() && !order.UpdatedAt.Before(order.CreatedAt) {
		resp.ReviewedAt = &order.UpdatedAt
	}

	// 处理审批人信息
	if order.ReviewerID != nil {
		resp.Reviewer = model.UserInfo{
			Username: order.Reviewer.Name,
			Nickname: order.Reviewer.Nickname,
		}
	}

	return resp
}

// checkAutoApprovalEligibility 检查是否满足自动审批条件
func (mgr *ApprovalOrderMgr) checkAutoApprovalEligibility(c *gin.Context, userID uint, req *ApprovalOrderreq) (bool, error) {
	// 只有作业类型且延长时间小于12小时才可能自动审批
	if req.Type != model.ApprovalOrderTypeJob || req.ExtensionHours == 0 || req.ExtensionHours >= 12 {
		return false, nil
	}

	// 查询该用户48小时内的所有工单
	ao := query.ApprovalOrder
	fortyEightHoursAgo := time.Now().Add(-48 * time.Hour)

	recentOrders, err := ao.WithContext(c).
		Where(ao.CreatorID.Eq(userID)).
		Where(ao.CreatedAt.Gt(fortyEightHoursAgo)).
		Find()

	if err != nil {
		return false, err
	}

	for _, order := range recentOrders {
		if order.Type == model.ApprovalOrderTypeJob &&
			order.Status == model.ApprovalOrderStatusApproved &&
			ptr.Deref(order.ReviewerID, 0) == 0 {
			return false, nil
		}
	}

	return true, nil
}
