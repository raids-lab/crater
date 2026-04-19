package handler

import (
	"fmt"
	"time"

	"encoding/json"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"k8s.io/klog/v2"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/utils"
)

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
	g.GET("", mgr.ListAllApprovalOrders)                // 获取所有审批工单
	g.GET("/:id", mgr.GetApprovalOrderAdmin)            // 管理员通过ID获取审批工单详情
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
	_, err := ao.WithContext(c).
		Where(ao.ID.Eq(orderID)).
		Updates(map[string]any{
			"status":       string(model.ApprovalOrderStatusCancelled),
			"review_notes": reason,
		})

	if err != nil {
		return fmt.Errorf("failed to update approval order status: %w", err)
	}

	return nil
}

// isJobRunning 判断作业是否在运行状态
func (mgr *ApprovalOrderMgr) isJobRunning(jobStatus batch.JobPhase) bool {
	// 根据volcano的JobPhase枚举值判断，运行状态包括：
	// Running, Pending, Inqueue 等状态
	runningStatuses := []batch.JobPhase{
		"Running",
		// 可以根据实际需要添加其他被认为是"运行"状态的状态
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
	for i := range orders {
		resp := ApprovalOrderResp{
			ID:          orders[i].ID,
			Name:        orders[i].Name,
			Type:        orders[i].Type,
			Status:      orders[i].Status,
			Content:     unmarshalApprovalOrderContent(orders[i].Content),
			ReviewNotes: orders[i].ReviewNotes,
			CreatedAt:   orders[i].CreatedAt,
			CreatorID:   orders[i].CreatorID,
			Creator: model.UserInfo{
				Username: orders[i].Creator.Name,
				Nickname: orders[i].Creator.Nickname,
			},
			ReviewerID: orders[i].ReviewerID,
		}

		// 处理审批人信息
		if orders[i].ReviewerID != 0 {
			resp.Reviewer = model.UserInfo{
				Username: orders[i].Reviewer.Name,
				Nickname: orders[i].Reviewer.Nickname,
			}
		}

		result = append(result, resp)
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
	TypeID         uint                      `json:"approvalorderTypeID" `        // 关联的ID，可能是数据集或任务ID
	Reason         string                    `json:"approvalOrderReason" `        // 审批原因
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

	// 2. 检查是否满足自动审批条件
	autoApproved := false
	autoApprovalReason := "whitout review，approved due to system"

	canAutoApprove, err := mgr.checkAutoApprovalEligibility(c, token.UserID, &req)
	if err != nil {
		klog.Errorf("failed to check auto approval eligibility, userID: %d, err: %v", token.UserID, err)
		resputil.Error(c, "failed to check recent orders", resputil.NotSpecified)
		return
	}

	if canAutoApprove {
		// 尝试锁定作业
		if err := mgr.lockJobForApproval(c, req.Name, req.ExtensionHours); err != nil {
			klog.Errorf("failed to lock job for auto approval, jobName: %s, err: %v", req.Name, err)
			// 锁定失败时不进行自动审批，但继续创建工单
		} else {
			autoApproved = true
		}
	}

	// 3. 创建审批工单
	orderStatus := model.ApprovalOrderStatusPending
	orderReason := ""

	// 如果满足自动审批条件，则将工单状态设置为已批准,并添加备注
	if autoApproved {
		orderStatus = model.ApprovalOrderStatusApproved
		orderReason = autoApprovalReason
	}

	order := model.ApprovalOrder{
		Name:   req.Name,
		Type:   req.Type,
		Status: orderStatus,
		Content: datatypes.NewJSONType(model.ApprovalOrderContent{
			ApprovalOrderTypeID:         req.TypeID,
			ApprovalOrderExtensionHours: req.ExtensionHours,
			ApprovalOrderReason:         req.Reason,
		}),
		CreatorID:   token.UserID,
		ReviewNotes: orderReason,
	}

	if err := query.ApprovalOrder.WithContext(c).Create(&order); err != nil {
		klog.Errorf("failed to create approval order, userID: %d, err: %v", token.UserID, err)
		resputil.Error(c, "failed to create approval order", resputil.NotSpecified)
		return
	}

	message := "create approvalorder successfully"
	if autoApproved {
		message = "create approvalorder successfully and auto-approved with job locked"
	}

	resputil.Success(c, message)
}

type UpdateApprovalOrder struct {
	Name           string                    `json:"name" binding:"required"`
	Type           model.ApprovalOrderType   `json:"type" binding:"required"`
	Status         model.ApprovalOrderStatus `json:"status"`
	TypeID         uint                      `json:"approvalorderTypeID" `        // 关联的ID，可能是数据集或任务ID
	Reason         string                    `json:"approvalOrderReason" `        // 审批原因
	ExtensionHours uint                      `json:"approvalOrderExtensionHours"` // 延长小时数
	ReviewerID     uint                      `json:"reviewerID"`                  // 审批人ID
	ReviewNotes    string                    `json:"reviewNotes"`                 // 审批备注
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

	// 权限检查：只有管理员才能将工单状态更新为“已批准”
	if req.Status == model.ApprovalOrderStatusApproved && token.RolePlatform != model.RoleAdmin {
		klog.Warningf("permission denied: user %d (role: %s) attempted to approve order %d",
			token.UserID, token.RolePlatform, orderID.ID)
		resputil.Error(c, "permission denied: only admins can approve orders", resputil.NotSpecified)
		return
	}

	// 2. 更新审批工单
	order := model.ApprovalOrder{
		Name:   req.Name,
		Type:   req.Type,
		Status: req.Status,
		Content: datatypes.NewJSONType(model.ApprovalOrderContent{
			ApprovalOrderTypeID:         req.TypeID,
			ApprovalOrderExtensionHours: req.ExtensionHours,
			ApprovalOrderReason:         req.Reason,
		}),
		ReviewerID:  req.ReviewerID,
		ReviewNotes: req.ReviewNotes,
	}

	info, err := query.ApprovalOrder.WithContext(c).Where(query.ApprovalOrder.ID.Eq(orderID.ID)).Updates(&order)
	if err != nil {
		klog.Errorf("failed to update approval order, userID: %d, err: %v", token.UserID, err)
		resputil.Error(c, "failed to update approval order", resputil.NotSpecified)
		return
	}
	klog.Infof("updated approval order successfully, affected rows: %d", info.RowsAffected)
	resputil.Success(c, "update approvalorder successfully")
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

	// 6. 执行删除操作
	result, err := ao.WithContext(c).Where(ao.ID.Eq(orderID.ID)).Delete(&model.ApprovalOrder{})
	if err != nil {
		klog.Errorf("failed to delete approval order, userID: %d, orderID: %d, err: %v", token.UserID, orderID, err)
		resputil.Error(c, "failed to delete approval order", resputil.NotSpecified)
		return
	}

	// 7. 检查是否真的删除了记录
	if result.RowsAffected == 0 {
		klog.Warningf("delete operation affected no records, userID: %d, orderID: %d", token.UserID, orderID)
		resputil.Error(c, "delete failed, record not found", resputil.NotSpecified)
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
		resputil.Error(c, "permission denied to view this approval order", resputil.NotSpecified)
		return
	}

	// 5. 转换为响应格式
	result := convertToApprovalOrderResp(order)
	resputil.Success(c, result)
}

type ApprovalOrderNameReq struct {
	Name string `uri:"name" binding:"required"` // 工单名称
}

// swagger
//
//	@Summary		通过名称获取审批工单
//	@Description	通过名称获取所有同名审批工单（无需身份验证）
//	@Tags			approvalorder
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name path string true "工单名称"
//	@Success		200	{object}	resputil.Response[[]ApprovalOrderResp]	"成功返回工单列表"
//	@Failure		400	{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		404	{object}	resputil.Response[any]	"工单不存在"
//	@Failure		500	{object}	resputil.Response[any]	"服务器错误"
//	@Router			/v1/approvalorder/name/{name} [get]
func (mgr *ApprovalOrderMgr) GetApprovalOrderByName(c *gin.Context) {
	// 1. 获取工单名称
	var nameReq ApprovalOrderNameReq
	if err := c.ShouldBindUri(&nameReq); err != nil {
		klog.Errorf("failed to bind request parameters: %v", err)
		resputil.Error(c, "invalid request parameters", resputil.NotSpecified)
		return
	}

	// 2. 查询指定名称的所有工单（不验证身份）
	ao := query.ApprovalOrder
	orders, err := ao.WithContext(c).
		Preload(ao.Creator).  // 预加载创建人信息
		Preload(ao.Reviewer). // 预加载审批人信息（可能为空）
		Where(ao.Name.Eq(nameReq.Name)).
		Order(ao.CreatedAt.Desc()). // 按创建时间倒序排列，最新的在前
		Find()

	if err != nil {
		klog.Errorf("failed to query approval orders by name, name: %s, err: %v", nameReq.Name, err)
		resputil.Error(c, "failed to get approval orders", resputil.NotSpecified)
		return
	}

	// 3. 转换为响应格式（即使为空也返回空列表）
	result := convertToApprovalOrderResps(orders)
	if len(orders) == 0 {
		klog.Infof("no approval orders found for name: %s", nameReq.Name)
	} else {
		klog.Infof("found %d approval orders for name: %s", len(result), nameReq.Name)
	}
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
		ReviewerID: order.ReviewerID,
	}

	// 处理审批人信息
	if order.ReviewerID != 0 {
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
	if req.Type != model.ApprovalOrderTypeJob || req.ExtensionHours >= 12 {
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

	// 检查是否所有工单的ReviewNotes都不为自动审批原因
	autoApprovalReason := "whitout review，approved due to system"
	for _, order := range recentOrders {
		if order.ReviewNotes == autoApprovalReason {
			return false, nil
		}
	}

	return true, nil
}

// lockJobForApproval 为审批工单锁定作业
func (mgr *ApprovalOrderMgr) lockJobForApproval(c *gin.Context, jobName string, extensionHours uint) error {
	jobDB := query.Job

	// 查找作业
	j, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(jobName)).First()
	if err != nil {
		return err
	}

	// 检查是否已经永久锁定
	if j.LockedTimestamp.Equal(utils.GetPermanentTime()) {
		return fmt.Errorf("job %s is already permanently locked", jobName)
	}

	// 检查延长小时数是否在合理范围内，防止整数溢出
	const maxHours = 1440 // 两个月的小时数，作为合理的上限
	if extensionHours > maxHours {
		return fmt.Errorf("extension hours %d exceeds maximum allowed value %d", extensionHours, maxHours)
	}

	// 计算锁定时间：基于当前锁定时间或当前时间 + 延长小时数
	lockTime := utils.GetLocalTime()
	if j.LockedTimestamp.After(utils.GetLocalTime()) {
		lockTime = j.LockedTimestamp
	}

	// 安全地转换为 Duration，已经确保 extensionHours 在合理范围内
	extensionDuration := time.Duration(extensionHours) * time.Hour
	lockTime = lockTime.Add(extensionDuration)

	// 更新作业锁定时间
	if _, err := jobDB.WithContext(c).Where(jobDB.JobName.Eq(jobName)).Update(jobDB.LockedTimestamp, lockTime); err != nil {
		return err
	}

	klog.Infof("auto-locked job %s until %s", jobName, lockTime.Format("2006-01-02 15:04:05"))
	return nil
}
