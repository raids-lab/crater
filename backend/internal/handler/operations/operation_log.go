package operations

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
)

//nolint:gochecknoinits
func init() {
	handler.Registers = append(handler.Registers, NewOperationLogMgr)
}

type OperationLogMgr struct {
	name string
}

func NewOperationLogMgr(_ *handler.RegisterConfig) handler.Manager {
	return &OperationLogMgr{
		name: "operation-logs",
	}
}

func (mgr *OperationLogMgr) GetName() string { return mgr.name }

func (mgr *OperationLogMgr) RegisterPublic(_ *gin.RouterGroup)    {}
func (mgr *OperationLogMgr) RegisterProtected(_ *gin.RouterGroup) {}

func (mgr *OperationLogMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.ListOperationLogs)
	g.DELETE("", mgr.ClearOperationLogs)
}

type ListOperationLogsReq struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"limit,default=10"`
	Type     string `form:"operation_type"`
	Operator string `form:"operator"`
	Target   string `form:"target"`
	Search   string `form:"search"`
}

type OperationLogResp struct {
	ID            uint           `json:"id"`
	Operator      string         `json:"operator"`
	OperatorRole  string         `json:"operator_role"`
	OperationType string         `json:"operation_type"`
	Target        string         `json:"target"`
	Details       datatypes.JSON `json:"details"`
	Status        string         `json:"status"`
	ErrorMessage  string         `json:"error_message,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (mgr *OperationLogMgr) buildOperatorDisplayMap(c *gin.Context, logs []*model.OperationLog) map[string]string {
	result := make(map[string]string)
	if len(logs) == 0 {
		return result
	}

	usernameSet := make(map[string]struct{})
	for _, log := range logs {
		if log.Operator == "" {
			continue
		}
		usernameSet[log.Operator] = struct{}{}
	}

	if len(usernameSet) == 0 {
		return result
	}

	usernames := make([]string, 0, len(usernameSet))
	for name := range usernameSet {
		usernames = append(usernames, name)
	}

	ctx := c.Request.Context()
	u := query.User
	users, err := u.WithContext(ctx).
		Select(u.Name, u.Nickname).
		Where(u.Name.In(usernames...)).
		Find()
	if err != nil {
		klog.Errorf("failed to fetch operator nicknames: %v", err)
		return result
	}

	for _, user := range users {
		display := user.Nickname
		if display == "" {
			display = user.Name
		}
		result[user.Name] = display
	}

	return result
}

// ListOperationLogs godoc
//
//	@Summary		获取操作日志列表
//	@Description	获取操作日志列表，支持分页和筛选
//	@Tags			OperationLog
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param		page			query		int		false	"页码"
//	@Param		limit		query		int		false	"每页数量"
//	@Param		operation_type	query		string	false	"操作类型"
//	@Param		operator		query		string	false	"操作人"
//	@Param		target		query		string	false	"操作对象"
//	@Param		search		query		string	false	"模糊搜索关键词"
//	@Success		200			{object}	resputil.Response[resputil.List[OperationLogResp]]
//	@Failure		400			{object}	resputil.Response[any]
//	@Failure		500			{object}	resputil.Response[any]
//	@Router			/v1/admin/operation-logs [get]
func (mgr *OperationLogMgr) ListOperationLogs(c *gin.Context) {
	var req ListOperationLogsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		klog.Errorf("Bind Query failed, err: %v", err)
		resputil.Error(c, "Invalid request parameter", resputil.NotSpecified)
		return
	}

	logs, total, err := service.OpLog.List(c, req.Page, req.PageSize, req.Operator, req.Type, req.Target, req.Search)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list operation logs failed, err %v", err), resputil.NotSpecified)
		return
	}

	operatorDisplay := mgr.buildOperatorDisplayMap(c, logs)
	logResps := make([]OperationLogResp, 0, len(logs))
	for _, log := range logs {
		displayOperator := log.Operator
		if name, ok := operatorDisplay[log.Operator]; ok && name != "" {
			displayOperator = name
		}

		logResps = append(logResps, OperationLogResp{
			ID:            log.ID,
			Operator:      displayOperator,
			OperatorRole:  log.OperatorRole,
			OperationType: log.OperationType,
			Target:        log.Target,
			Details:       log.Details,
			Status:        log.Status,
			ErrorMessage:  log.Message,
			CreatedAt:     log.CreatedAt,
			UpdatedAt:     log.UpdatedAt,
		})
	}

	resputil.Success(c, resputil.List[OperationLogResp]{
		Total: total,
		Items: logResps,
	})
}

// ClearOperationLogs godoc
//
//	@Summary		清空操作日志
//	@Description	删除所有操作日志记录，仅用于测试/调试场景
//	@Tags		OperationLog
//	@Accept		json
//	@Produce	json
//	@Security	Bearer
//	@Success	200	{object}	resputil.Response[map[string]string]
//	@Failure	500	{object}	resputil.Response[any]
//	@Router		/v1/admin/operation-logs [delete]
func (mgr *OperationLogMgr) ClearOperationLogs(c *gin.Context) {
	if err := service.OpLog.Clear(c); err != nil {
		klog.Errorf("clear operation logs failed, err: %v", err)
		resputil.Error(c, fmt.Sprintf("clear operation logs failed, err %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, gin.H{"message": "cleared"})
}

// Helper function to create operation log
func CreateLog(c *gin.Context, opType string, target string, details datatypes.JSON, status string, message string) {
	// 尝试从 Context 中获取用户信息
	// 假设 middleware.AuthProtected() 设置了 "username" 和 "role"
	// 具体实现可能需要参考 internal/util/token.go 或相关逻辑

	// 这里假设有一个工具函数或者从 context 取
	// 暂时先留空或者简单的取值逻辑，实际项目中需要根据 auth middleware 来调整
	operator := ""
	role := ""

	// 检查是否有 user claims
	if claims, exists := c.Get("claims"); exists {
		// 根据实际 Claims 结构体断言
		// operator = claims.Username
		// role = claims.Role
		_ = claims
	}

	// 或者从 Header/Token 解析
	// ...

	// 调用 Service
	err := service.OpLog.Create(c, operator, role, opType, target, details, status, message)
	if err != nil {
		klog.Errorf("Create operation log failed: %v", err)
	}
}
