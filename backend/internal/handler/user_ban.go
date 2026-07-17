package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

type UpdateUserBanReq struct {
	Banned *bool  `json:"banned" binding:"required"`
	Reason string `json:"reason" binding:"required,max=500"`
}

type UserBanRecordResp struct {
	ID           uint                `json:"id"`
	CreatedAt    time.Time           `json:"createdAt"`
	OperatorID   uint                `json:"operatorId"`
	OperatorName string              `json:"operatorName"`
	Action       model.UserBanAction `json:"action"`
	Reason       string              `json:"reason"`
}

type UserBanStatusResp struct {
	Banned   bool                `json:"banned"`
	BannedAt *time.Time          `json:"bannedAt"`
	Records  []UserBanRecordResp `json:"records"`
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

	resp := UserBanStatusResp{
		Banned:   user.BannedAt != nil,
		BannedAt: user.BannedAt,
		Records:  make([]UserBanRecordResp, 0, len(records)),
	}
	for _, record := range records {
		resp.Records = append(resp.Records, UserBanRecordResp{
			ID:           record.ID,
			CreatedAt:    record.CreatedAt,
			OperatorID:   record.OperatorID,
			OperatorName: record.OperatorName,
			Action:       record.Action,
			Reason:       record.Reason,
		})
	}
	resputil.Success(c, resp)
}

// UpdateUserBanStatus godoc
// @Summary      封禁或解除封禁用户
// @Description 仅平台管理员可修改用户封禁状态，操作原因会写入封禁记录
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        name path string true "username"
// @Param        data body UpdateUserBanReq true "封禁状态与原因"
// @Success      200 {object} resputil.Response[UserBanStatusResp]
// @Router       /v1/admin/users/{name}/ban [put]
func (mgr *UserMgr) UpdateUserBanStatus(c *gin.Context) {
	var req UpdateUserBanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid request body"))
		return
	}

	token := util.GetToken(c)
	user, err := mgr.userBanService.SetBan(
		c.Request.Context(),
		c.Param("name"),
		token.UserID,
		token.Username,
		*req.Banned,
		req.Reason,
	)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, UserBanStatusResp{
		Banned:   user.BannedAt != nil,
		BannedAt: user.BannedAt,
		Records:  []UserBanRecordResp{},
	})
}
