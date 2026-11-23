package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/config"
)

// GetUserFields godoc
//
// @Summary Get authentication-independent user field presentation
// @Tags SystemConfig
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[map[string]config.UserFieldPresentation]
// @Router /v1/admin/system-config/user-fields [get]
func (mgr *SystemConfigMgr) GetUserFields(c *gin.Context) {
	resputil.Success(c, config.GetConfig().UserFieldPresentation())
}
