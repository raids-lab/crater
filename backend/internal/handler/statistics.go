package handler

import (
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

//nolint:gochecknoinits // init 用于注册处理器路由
func init() {
	Registers = append(Registers, NewStatisticsMgr)
}

type StatisticsMgr struct {
	name    string
	service *service.StatisticsService
}

func NewStatisticsMgr(conf *RegisterConfig) Manager {
	return &StatisticsMgr{
		name:    "statistics",
		service: conf.StatisticsService,
	}
}

func (mgr *StatisticsMgr) GetName() string { return mgr.name }

func (mgr *StatisticsMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *StatisticsMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.UserStatistics)
	g.GET("/rankings", mgr.UserRankings)
}

func (mgr *StatisticsMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.AdminStatistics)
	g.GET("/rankings", mgr.AdminRankings)
}

// UserStatistics 查询用户或账户的资源统计。
// @Summary 获取资源统计信息
// @Description 按 UTC 自然日统计，最多 366 天。个人维度查询本人返回完整指标，查询他人仅返回占用量。账户管理员可查询完整指标。
// @Tags statistics
// @Accept json
// @Produce json
// @Security Bearer
// @Param startTime query string true "开始 UTC 自然日，必须为 UTC 00:00 (RFC3339)"
// @Param endTime query string true "结束 UTC 自然日，不包含，必须为 UTC 00:00 且晚于开始时间 (RFC3339)"
// @Param step query string false "聚合粒度 (day/week)，默认 day"
// @Param scope query string true "统计范围 (user/account)"
// @Param targetID query int true "目标ID (user_id 或 account_id)"
// @Success 200 {object} resputil.Response[payload.StatisticsResp]
// @Router /v1/statistics [get]
func (mgr *StatisticsMgr) UserStatistics(c *gin.Context) { mgr.getStatistics(c, false) }

func (mgr *StatisticsMgr) getStatistics(c *gin.Context, adminRoute bool) {
	var req payload.StatisticsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		klog.Errorf("bind query params failed: %v", err)
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid query parameters"))
		return
	}

	if !authorizeStatistics(c, &req, adminRoute) || !validateStatisticsRange(c, &req) {
		return
	}

	token := util.GetToken(c)
	allocatedOnly := !adminRoute && req.Scope == payload.ScopeUser && req.TargetID != token.UserID
	resp, err := mgr.service.GetResourceStatistics(c.Request.Context(), &req, allocatedOnly)
	if err != nil {
		klog.Errorf("get resource statistics failed: %v", err)
		resputil.HandleError(c, err)
		return
	}

	resputil.Success(c, resp)
}

func authorizeStatistics(c *gin.Context, req *payload.StatisticsReq, adminRoute bool) bool {
	// 1. 获取当前用户身份
	token := util.GetToken(c)
	if token.UserID == 0 {
		resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("unauthorized"))
		return false
	}
	if adminRoute && token.RolePlatform != model.RoleAdmin {
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("permission denied: admin route requires the admin role"))
		return false
	}
	ua := query.UserAccount

	// 2. 权限校验与参数修正
	switch req.Scope {
	case payload.ScopeCluster:
		// 只有管理员可以查询集群维度
		if !adminRoute {
			resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("permission denied: cluster scope requires the admin role"))
			return false
		}

	case payload.ScopeAccount:
		if req.TargetID == 0 {
			resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("targetID is required for account scope"))
			return false
		}

		// 如果是平台管理员，拥有全量权限，直接跳过校验
		if token.RolePlatform == model.RoleAdmin {
			break
		}

		// 非平台管理员须是目标账户的管理员。
		count, err := ua.WithContext(c.Request.Context()).
			Where(
				ua.UserID.Eq(token.UserID),
				ua.AccountID.Eq(req.TargetID),
				ua.Role.Eq(uint8(model.RoleAdmin)), // 确保 role 匹配 admin
			).Count()

		if err != nil {
			klog.Errorf("query user account permission failed: %v", err)
			resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "query user account permission failed"))
			return false
		}

		if count == 0 {
			resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("permission denied: you are not the admin of this account"))
			return false
		}

	case payload.ScopeUser:
		// 所有用户可互相查询
		if req.TargetID == 0 {
			resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("targetID is required for user scope"))
			return false
		}

	default:
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("invalid scope value"))
		return false
	}

	return true
}

const maxStatisticsDays = 366

func validateStatisticsRange(c *gin.Context, req *payload.StatisticsReq) bool {
	if !req.EndTime.After(req.StartTime) || !service.StatisticsDay(req.StartTime).Equal(req.StartTime) ||
		!service.StatisticsDay(req.EndTime).Equal(req.EndTime) {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("statistics requires complete UTC days and endTime > startTime"))
		return false
	}
	if req.EndTime.After(req.StartTime.UTC().AddDate(0, 0, maxStatisticsDays)) {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("statistics range cannot exceed 366 UTC days"))
		return false
	}
	return true
}

// UserRankings 查询所管理账户内的用户排名。
// @Summary 获取资源用量与利用率排名
// @Description 按 UTC 自然日统计，最多 366 天。利用率缺失的对象不进入利用率排名。
// @Tags statistics
// @Produce json
// @Security Bearer
// @Param startTime query string true "开始 UTC 日 (RFC3339)"
// @Param endTime query string true "结束 UTC 日，不包含 (RFC3339)"
// @Param scope query string true "account"
// @Param targetID query int true "账户 ID"
// @Param dimension query string true "user" Enums(user)
// @Param metric query string true "usage/effectiveUsage/utilization"
// @Param order query string true "asc/desc"
// @Param resourceName query string false "完整资源名，与 resourceType 二选一"
// @Param resourceType query string false "gpu：全部整卡 GPU 资源"
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页条数，默认 20，最多 100"
// @Success 200 {object} resputil.Response[payload.StatisticsRankingResp]
// @Router /v1/statistics/rankings [get]
func (mgr *StatisticsMgr) UserRankings(c *gin.Context) { mgr.getRankings(c, false) }

func (mgr *StatisticsMgr) getRankings(c *gin.Context, adminRoute bool) {
	var req payload.StatisticsRankingReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid ranking parameters"))
		return
	}
	if req.Scope == payload.ScopeUser || (req.Scope == payload.ScopeAccount && req.Dimension != "user") ||
		(req.ResourceName == "") == (req.ResourceType == "") {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("rankings requires cluster/account scope and one resource selector"))
		return
	}
	if !authorizeStatistics(c, &req.StatisticsReq, adminRoute) || !validateStatisticsRange(c, &req.StatisticsReq) {
		return
	}
	response, err := mgr.service.GetRankings(c.Request.Context(), &req)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, response)
}

// AdminStatistics 为平台管理员查询资源统计。
// @Summary 获取管理员资源统计
// @Description 按 UTC 自然日统计，最多 366 天。时间区间为左闭右开。
// @Tags statistics
// @Produce json
// @Security Bearer
// @Param startTime query string true "开始 UTC 日，UTC 00:00 (RFC3339)"
// @Param endTime query string true "结束 UTC 日，不包含，必须晚于开始时间 (RFC3339)"
// @Param step query string false "day/week，默认 day"
// @Param scope query string true "user/account/cluster"
// @Param targetID query int false "scope=user 或 account 时必填，scope=cluster 时忽略"
// @Success 200 {object} resputil.Response[payload.StatisticsResp]
// @Router /v1/admin/statistics [get]
func (mgr *StatisticsMgr) AdminStatistics(c *gin.Context) { mgr.getStatistics(c, true) }

// AdminRankings 为平台管理员查询用户或账户排名。
// @Summary 获取管理员资源排名
// @Description 按 UTC 自然日统计，最多 366 天。时间区间为左闭右开。
// @Tags statistics
// @Produce json
// @Security Bearer
// @Param startTime query string true "开始 UTC 日，UTC 00:00 (RFC3339)"
// @Param endTime query string true "结束 UTC 日，不包含，必须晚于开始时间 (RFC3339)"
// @Param scope query string true "cluster/account"
// @Param targetID query int false "scope=account 时必填，scope=cluster 时忽略"
// @Param dimension query string true "scope=account 时仅支持 user，scope=cluster 时支持 user/account" Enums(user, account)
// @Param metric query string true "usage/effectiveUsage/utilization"
// @Param order query string true "asc/desc"
// @Param resourceName query string false "完整资源名，与 resourceType 二选一"
// @Param resourceType query string false "gpu：全部整卡 GPU 资源"
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页条数，默认 20，最多 100"
// @Success 200 {object} resputil.Response[payload.StatisticsRankingResp]
// @Router /v1/admin/statistics/rankings [get]
func (mgr *StatisticsMgr) AdminRankings(c *gin.Context) { mgr.getRankings(c, true) }
