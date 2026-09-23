package operations

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/pagination"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
)

//nolint:gochecknoinits // Gin managers are registered via package init hooks.
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
	g.GET("/page", mgr.ListOperationLogsPage)
	g.GET("", mgr.ListOperationLogs)
	g.DELETE("", mgr.ClearOperationLogs)
}

type operationLogPageQuery struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	PageSize int    `form:"page_size,default=10" binding:"min=1,max=200"`
	Search   string `form:"search"`
	Sort     string `form:"sort"`
}

var operationLogSortFields = map[string]string{
	"createdAt":     "created_at",
	"updatedAt":     "updated_at",
	"operator":      "operator",
	"operationType": "operation_type",
	"target":        "target",
	"status":        "status",
}

func operationLogSortClauses(sortValue string) []string {
	return pagination.SortClauses(sortValue, operationLogSortFields, "id DESC")
}

func firstOperationLogQueryValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func (mgr *OperationLogMgr) bindPageQuery(c *gin.Context) (operationLogPageQuery, error) {
	var request operationLogPageQuery
	if err := c.ShouldBindQuery(&request); err != nil {
		return operationLogPageQuery{}, bizerr.BadRequest.ParameterError.Wrap(
			err,
			"invalid operation log list query",
		)
	}
	var err error
	request.Search, err = pagination.NormalizePageQuery(request.Search, request.Page, request.PageSize)
	if err != nil {
		return operationLogPageQuery{}, err
	}
	request.Sort, err = pagination.DefaultAndValidateSort(request.Sort, "-createdAt", operationLogSortFields)
	if err != nil {
		return operationLogPageQuery{}, err
	}
	return request, nil
}

type ListOperationLogsReq struct {
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"limit,default=10"`
	Type      string `form:"operation_type"`
	Operator  string `form:"operator"`
	Target    string `form:"target"`
	Search    string `form:"search"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}

type OperationLogResp struct {
	ID            uint           `json:"id"`
	Operator      string         `json:"operator"`
	OperatorRole  string         `json:"operator_role"`
	OperationType string         `json:"operation_type"`
	Target        string         `json:"target"`
	Details       map[string]any `json:"details" swaggertype:"object"`
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

func (mgr *OperationLogMgr) buildOperationLogResponses(
	c *gin.Context,
	logs []*model.OperationLog,
) []OperationLogResp {
	operatorDisplay := mgr.buildOperatorDisplayMap(c, logs)
	logResps := make([]OperationLogResp, 0, len(logs))
	for _, log := range logs {
		details := make(map[string]any)
		if len(log.Details) > 0 {
			if err := json.Unmarshal(log.Details, &details); err != nil {
				klog.Warningf("failed to unmarshal operation log details for log %d: %v", log.ID, err)
			}
		}

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
			Details:       details,
			Status:        log.Status,
			ErrorMessage:  log.Message,
			CreatedAt:     log.CreatedAt,
			UpdatedAt:     log.UpdatedAt,
		})
	}
	return logResps
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
//	@Param		start_time		query		string	false	"开始时间，格式例如：2024-01-02T15:04:05Z"
//	@Param		end_time		query		string	false	"结束时间，格式例如：2024-01-02T15:04:05Z"
//	@Success		200			{object}	resputil.Response[resputil.List[OperationLogResp]]
//	@Failure		400			{object}	resputil.Response[any]
//	@Failure		500			{object}	resputil.Response[any]
//	@Router			/v1/admin/operation-logs [get]
func (mgr *OperationLogMgr) ListOperationLogs(c *gin.Context) {
	var req ListOperationLogsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		klog.Errorf("Bind Query failed, err: %v", err)
		resputil.BadRequestError(c, "invalid request parameter")
		return
	}

	startTime, endTime, err := parseOperationLogTimeRange(req.StartTime, req.EndTime)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	logs, total, err := service.OpLog.List(
		c,
		req.Page,
		req.PageSize,
		req.Operator,
		req.Type,
		req.Target,
		req.Search,
		startTime,
		endTime,
	)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list operation logs failed, err %v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, resputil.List[OperationLogResp]{
		Total: total,
		Items: mgr.buildOperationLogResponses(c, logs),
	})
}

// ListOperationLogsPage returns operation logs using the shared page protocol.
//
//	@Summary	获取操作日志分页
//	@Description	按统一分页协议获取操作日志，筛选和排序在数据库中执行
//	@Tags		OperationLog
//	@Produce	json
//	@Security	Bearer
//	@Param		page		query	int		false	"Page number"
//	@Param		page_size	query	int		false	"Page size, 1-200"
//	@Param		search		query	string	false	"Search operator, type, target, or message"
//	@Param		operation_type	query	string	false	"Filter by operation type"
//	@Param		operator	query	string	false	"Filter by operator"
//	@Param		target		query	string	false	"Filter by target"
//	@Param		start_time	query	string	false	"Start time, RFC3339"
//	@Param		end_time	query	string	false	"End time, RFC3339"
//	@Param		sort		query	string	false	"Sort fields"
//	@Success	200	{object}	resputil.Response[resputil.Page[OperationLogResp]]
//	@Router		/v1/admin/operation-logs/page [get]
func (mgr *OperationLogMgr) ListOperationLogsPage(c *gin.Context) {
	request, err := mgr.bindPageQuery(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	startTime, endTime, err := parseOperationLogTimeRange(
		c.Query("start_time"),
		c.Query("end_time"),
	)
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid operation log time range"))
		return
	}

	logs, total, err := service.OpLog.ListPage(
		c,
		request.Page,
		request.PageSize,
		firstOperationLogQueryValue(c.QueryArray("operator")),
		firstOperationLogQueryValue(c.QueryArray("operation_type")),
		firstOperationLogQueryValue(c.QueryArray("target")),
		request.Search,
		startTime,
		endTime,
		strings.Join(operationLogSortClauses(request.Sort), ", "),
	)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "list operation logs failed"))
		return
	}

	resputil.Success(c, resputil.NewPage(
		mgr.buildOperationLogResponses(c, logs),
		total,
		request.Page,
		request.PageSize,
	))
}

func parseOperationLogTimeRange(
	startTimeRaw string,
	endTimeRaw string,
) (startTime, endTime *time.Time, err error) {
	parseTime := func(value string) (*time.Time, error) {
		if value == "" {
			return nil, nil
		}

		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			parsed, err := time.Parse(layout, value)
			if err == nil {
				return &parsed, nil
			}
		}

		return nil, fmt.Errorf("invalid time format: %s", value)
	}

	startTime, err = parseTime(startTimeRaw)
	if err != nil {
		return nil, nil, err
	}

	endTime, err = parseTime(endTimeRaw)
	if err != nil {
		return nil, nil, err
	}

	return startTime, endTime, nil
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
