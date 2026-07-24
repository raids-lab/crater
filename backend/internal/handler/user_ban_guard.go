package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

// RequireUserBanCapability enforces an active user's configured ban restrictions.
func RequireUserBanCapability(
	c *gin.Context,
	userBanService *service.UserBanService,
	capability service.UserBanCapability,
) bool {
	if userBanService == nil {
		return true
	}
	token := util.GetToken(c)
	if err := userBanService.RequireCapability(c.Request.Context(), token.UserID, capability); err != nil {
		resputil.HandleError(c, err)
		return false
	}
	return true
}
