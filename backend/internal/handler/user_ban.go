package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

type UpdateUserBanReq struct {
	Banned          *bool                     `json:"banned" binding:"required"`
	IsPermanent     bool                      `json:"isPermanent"`
	Days            int                       `json:"days"`
	Hours           int                       `json:"hours"`
	Minutes         int                       `json:"minutes"`
	BanRestrictions model.UserBanRestrictions `json:"banRestrictions"`
	Reason          string                    `json:"reason" binding:"max=500"`
}

type UserBanRecordResp struct {
	ID               uint                      `json:"id"`
	CreatedAt        time.Time                 `json:"createdAt"`
	OperatorID       uint                      `json:"operatorId"`
	OperatorName     string                    `json:"operatorName"`
	OperatorNickname string                    `json:"operatorNickname"`
	Action           model.UserBanAction       `json:"action"`
	PermanentBanned  bool                      `json:"permanentBanned"`
	BannedTimestamp  *time.Time                `json:"bannedTimestamp,omitempty"`
	BanRestrictions  model.UserBanRestrictions `json:"banRestrictions"`
	Reason           string                    `json:"reason"`
}

type UserBanStatusResp struct {
	Banned          bool                      `json:"banned"`
	PermanentBanned bool                      `json:"permanentBanned"`
	BannedTimestamp *time.Time                `json:"bannedTimestamp,omitempty"`
	BanRestrictions model.UserBanRestrictions `json:"banRestrictions"`
	Reason          string                    `json:"reason"`
	Records         []UserBanRecordResp       `json:"records"`
}

type VisibleUserBanRecordResp struct {
	ID              uint                      `json:"id"`
	CreatedAt       time.Time                 `json:"createdAt"`
	Action          model.UserBanAction       `json:"action"`
	PermanentBanned bool                      `json:"permanentBanned"`
	BannedTimestamp *time.Time                `json:"bannedTimestamp,omitempty"`
	BanRestrictions model.UserBanRestrictions `json:"banRestrictions"`
	Reason          string                    `json:"reason"`
}

type VisibleUserBanStatusResp struct {
	Banned          bool                       `json:"banned"`
	PermanentBanned bool                       `json:"permanentBanned"`
	BannedTimestamp *time.Time                 `json:"bannedTimestamp,omitempty"`
	BanRestrictions model.UserBanRestrictions  `json:"banRestrictions"`
	Reason          string                     `json:"reason"`
	Records         []VisibleUserBanRecordResp `json:"records"`
}

type CurrentUserBanStatusResp struct {
	Banned          bool                      `json:"banned"`
	PermanentBanned bool                      `json:"permanentBanned"`
	BannedTimestamp *time.Time                `json:"bannedTimestamp,omitempty"`
	BanRestrictions model.UserBanRestrictions `json:"banRestrictions"`
	Reason          string                    `json:"reason"`
}

// GetUserBanStatus godoc
// @Summary      获取用户封禁状态和记录
// @Description 仅平台管理员可查看指定用户的当前封禁状态和完整操作记录
// @Tags         User
// @Produce      json
// @Security     Bearer
// @Param        name path string true "username"
// @Success      200 {object} resputil.Response[UserBanStatusResp]
// @Router       /v1/admin/users/{name}/ban [get]
func (mgr *UserMgr) GetUserBanStatus(c *gin.Context) {
	user, records, err := mgr.userBanService.ListRecords(c.Request.Context(), c.Param("name"))
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	banned := service.IsUserBanned(user.BannedTimestamp)
	reason := ""
	if banned && len(records) > 0 {
		reason = records[0].Reason
	}
	resp := UserBanStatusResp{
		Banned:          banned,
		PermanentBanned: service.IsUserPermanentlyBanned(user.BannedTimestamp),
		BannedTimestamp: user.BannedTimestamp,
		BanRestrictions: service.EffectiveUserBanRestrictions(
			banned,
			user.BanRestrictions.Data(),
		),
		Reason:  reason,
		Records: make([]UserBanRecordResp, 0, len(records)),
	}
	operatorIDs := make([]uint, 0, len(records))
	seenOperatorIDs := make(map[uint]struct{}, len(records))
	for _, record := range records {
		if _, ok := seenOperatorIDs[record.OperatorID]; ok {
			continue
		}
		seenOperatorIDs[record.OperatorID] = struct{}{}
		operatorIDs = append(operatorIDs, record.OperatorID)
	}
	operatorNicknames := make(map[uint]string, len(operatorIDs))
	if len(operatorIDs) > 0 {
		u := query.User
		operators, queryErr := u.WithContext(c).Unscoped().
			Select(u.ID, u.Nickname).
			Where(u.ID.In(operatorIDs...)).
			Find()
		if queryErr != nil {
			resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(queryErr, "load ban operators failed"))
			return
		}
		for _, operator := range operators {
			operatorNicknames[operator.ID] = operator.Nickname
		}
	}
	for _, record := range records {
		resp.Records = append(resp.Records, UserBanRecordResp{
			ID:               record.ID,
			CreatedAt:        record.CreatedAt,
			OperatorID:       record.OperatorID,
			OperatorName:     record.OperatorName,
			OperatorNickname: operatorNicknames[record.OperatorID],
			Action:           record.Action,
			PermanentBanned:  service.IsUserPermanentlyBanned(record.BannedTimestamp),
			BannedTimestamp:  record.BannedTimestamp,
			BanRestrictions:  record.BanRestrictions.Data(),
			Reason:           record.Reason,
		})
	}
	resputil.Success(c, resp)
}

// GetVisibleUserBanStatus godoc
// @Summary      获取用户封禁状态和记录
// @Description 已登录用户可查看指定用户的封禁状态和记录，但不返回执行管理员信息
// @Tags         User
// @Produce      json
// @Security     Bearer
// @Param        name path string true "username"
// @Success      200 {object} resputil.Response[VisibleUserBanStatusResp]
// @Router       /v1/users/{name}/ban [get]
func (mgr *UserMgr) GetVisibleUserBanStatus(c *gin.Context) {
	user, records, err := mgr.userBanService.ListRecords(c.Request.Context(), c.Param("name"))
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	banned := service.IsUserBanned(user.BannedTimestamp)
	reason := ""
	if banned && len(records) > 0 {
		reason = records[0].Reason
	}
	resp := VisibleUserBanStatusResp{
		Banned:          banned,
		PermanentBanned: service.IsUserPermanentlyBanned(user.BannedTimestamp),
		BannedTimestamp: user.BannedTimestamp,
		BanRestrictions: service.EffectiveUserBanRestrictions(
			banned,
			user.BanRestrictions.Data(),
		),
		Reason:  reason,
		Records: make([]VisibleUserBanRecordResp, 0, len(records)),
	}
	for _, record := range records {
		resp.Records = append(resp.Records, VisibleUserBanRecordResp{
			ID:              record.ID,
			CreatedAt:       record.CreatedAt,
			Action:          record.Action,
			PermanentBanned: service.IsUserPermanentlyBanned(record.BannedTimestamp),
			BannedTimestamp: record.BannedTimestamp,
			BanRestrictions: record.BanRestrictions.Data(),
			Reason:          record.Reason,
		})
	}
	resputil.Success(c, resp)
}

// UpdateUserBanStatus godoc
// @Summary      设置、延长或解除用户封禁
// @Description 按作业锁定规则增加封禁时长，并为该用户设置本次封禁的限制内容
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        name path string true "username"
// @Param        data body UpdateUserBanReq true "封禁状态、增加时长与原因；解除封禁时原因可为空"
// @Success      200 {object} resputil.Response[UserBanStatusResp]
// @Router       /v1/admin/users/{name}/ban [put]
func (mgr *UserMgr) UpdateUserBanStatus(c *gin.Context) {
	var req UpdateUserBanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid request body"))
		return
	}
	if req.Days < 0 || req.Hours < 0 || req.Minutes < 0 {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("ban duration values must be non-negative"))
		return
	}
	duration := time.Duration(req.Days)*24*time.Hour +
		time.Duration(req.Hours)*time.Hour +
		time.Duration(req.Minutes)*time.Minute

	token := util.GetToken(c)
	user, err := mgr.userBanService.SetBan(
		c.Request.Context(),
		c.Param("name"),
		token.UserID,
		token.Username,
		*req.Banned,
		req.IsPermanent,
		duration,
		req.BanRestrictions,
		req.Reason,
	)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	banned := service.IsUserBanned(user.BannedTimestamp)
	reason := ""
	if banned {
		reason = strings.TrimSpace(req.Reason)
	}
	resputil.Success(c, UserBanStatusResp{
		Banned:          banned,
		PermanentBanned: service.IsUserPermanentlyBanned(user.BannedTimestamp),
		BannedTimestamp: user.BannedTimestamp,
		BanRestrictions: service.EffectiveUserBanRestrictions(
			banned,
			user.BanRestrictions.Data(),
		),
		Reason:  reason,
		Records: []UserBanRecordResp{},
	})
}

// GetCurrentUserBanStatus godoc
// @Summary      获取当前用户封禁状态
// @Description 返回当前用户仍在生效的封禁时间、原因和限制内容，不包含管理员操作历史
// @Tags         User
// @Produce      json
// @Security     Bearer
// @Success      200 {object} resputil.Response[CurrentUserBanStatusResp]
// @Router       /v1/users/ban [get]
func (mgr *UserMgr) GetCurrentUserBanStatus(c *gin.Context) {
	token := util.GetToken(c)
	user, record, err := mgr.userBanService.GetCurrentState(c.Request.Context(), token.UserID)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}

	banned := service.IsUserBanned(user.BannedTimestamp)
	reason := ""
	if banned && record != nil {
		reason = record.Reason
	}
	resputil.Success(c, CurrentUserBanStatusResp{
		Banned:          banned,
		PermanentBanned: service.IsUserPermanentlyBanned(user.BannedTimestamp),
		BannedTimestamp: user.BannedTimestamp,
		BanRestrictions: service.EffectiveUserBanRestrictions(
			banned,
			user.BanRestrictions.Data(),
		),
		Reason: reason,
	})
}
