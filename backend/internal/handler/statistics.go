package handler

import (
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

//nolint:gochecknoinits // init is used to register handler routes
func init() {
	Registers = append(Registers, NewStatisticsMgr)
}

type StatisticsMgr struct {
	name string
}

func NewStatisticsMgr(_ *RegisterConfig) Manager {
	return &StatisticsMgr{
		name: "statistics",
	}
}

func (mgr *StatisticsMgr) GetName() string { return mgr.name }

func (mgr *StatisticsMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *StatisticsMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.GetStatistics)
}

func (mgr *StatisticsMgr) RegisterAdmin(_ *gin.RouterGroup) {
}

// GetStatistics 获取资源统计
// @Summary 获取资源统计信息
// @Description 获取指定时间范围、指定维度的资源使用统计（核时/卡时）
// @Tags statistics
// @Accept json
// @Produce json
// @Security Bearer
// @Param startTime query string true "开始时间 (RFC3339)"
// @Param endTime query string true "结束时间 (RFC3339)"
// @Param step query string true "聚合粒度 (day/week)"
// @Param scope query string true "统计范围 (user/account/cluster)"
// @Param targetID query int false "目标ID (user_id 或 account_id)"
// @Success 200 {object} resputil.Response[payload.StatisticsResp]
// @Router /v1/statistics [get]
func (mgr *StatisticsMgr) GetStatistics(c *gin.Context) {
	var req payload.StatisticsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		klog.Errorf("bind query params failed: %v", err)
		resputil.Error(c, "invalid query parameters", resputil.NotSpecified)
		return
	}

	// 1. 获取当前用户身份
	token := util.GetToken(c)
	if token.UserID == 0 {
		resputil.Error(c, "unauthorized", resputil.NotSpecified)
		return
	}
	ua := query.UserAccount

	// 2. 权限校验与参数修正
	switch req.Scope {
	case payload.ScopeCluster:
		// 只有管理员可以查询集群维度
		if token.RolePlatform != model.RoleAdmin {
			resputil.Error(c, "permission denied: cluster scope requires admin role", resputil.NotSpecified)
			return
		}

	case payload.ScopeAccount:
		if req.TargetID == 0 {
			resputil.Error(c, "targetID is required for account scope", resputil.NotSpecified)
			return
		}

		// 如果是平台管理员，拥有全量权限，直接跳过校验
		if token.RolePlatform == model.RoleAdmin {
			break
		}

		// 【核心修改】：非平台管理员，检查该用户是否是目标账户的管理员 (RoleAdmin)
		count, err := ua.WithContext(c.Request.Context()).
			Where(
				ua.UserID.Eq(token.UserID),
				ua.AccountID.Eq(req.TargetID),
				ua.Role.Eq(uint8(model.RoleAdmin)), // 确保 role 匹配 admin
			).Count()

		if err != nil {
			klog.Errorf("query user account permission failed: %v", err)
			resputil.Error(c, "internal server error", resputil.NotSpecified)
			return
		}

		if count == 0 {
			resputil.Error(c, "permission denied: you are not the admin of this account", resputil.NotSpecified)
			return
		}

	case payload.ScopeUser:
		// 如果是普通用户，只能查自己
		if token.RolePlatform != model.RoleAdmin {
			req.TargetID = token.UserID
		}
		// 如果是 Admin 且没传 targetID，则报错；或者 Admin 可以指定查某个 User
		if req.TargetID == 0 {
			req.TargetID = token.UserID
		}
	}

	// 3. 校验时间范围
	if req.EndTime.Before(req.StartTime) {
		resputil.Error(c, "endTime must be after startTime", resputil.NotSpecified)
		return
	}
	// 限制查询跨度，防止数据库压力过大（例如最大一年）
	// if req.EndTime.Sub(req.StartTime) > 366*24*time.Hour {
	// 	resputil.Error(c, "time range too large (max 1 year)", resputil.NotSpecified)
	// 	return
	// }

	// 4. 调用 Service
	resp, err := service.Statistics.GetResourceStatistics(c.Request.Context(), &req)
	if err != nil {
		klog.Errorf("get resource statistics failed: %v", err)
		resputil.Error(c, "internal server error", resputil.NotSpecified)
		return
	}

	resputil.Success(c, resp)
}
