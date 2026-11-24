package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"

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
// @Router /v1/system-config/user-fields [get]
// @Router /v1/admin/system-config/user-fields [get]
func (mgr *SystemConfigMgr) GetUserFields(c *gin.Context) {
	resputil.Success(c, config.GetConfig().UserFieldPresentation())
}

// visibleUserAttributes is shared by detail, membership and dataset responses.
// Never return private profile fields through a secondary user listing.
func visibleUserAttributes(c *gin.Context, id uint, nickname string, original *model.UserAttribute) model.UserAttribute {
	attributes := *original
	token := util.GetToken(c)
	privileged := token.RolePlatform == model.RoleAdmin || token.UserID == id
	cfg := config.GetConfig()
	attributes.Nickname = nickname
	if !cfg.UserFieldVisible("email", privileged) {
		attributes.Email = nil
	}
	if !cfg.UserFieldVisible("identity", privileged) {
		attributes.Identity = nil
	}
	if !cfg.UserFieldVisible("teacher", privileged) {
		attributes.Teacher = nil
	}
	if !cfg.UserFieldVisible("group", privileged) {
		attributes.Group = nil
	}
	if !cfg.UserFieldVisible("phone", privileged) {
		attributes.Phone = nil
	}
	if !cfg.UserFieldVisible("expiredAt", privileged) {
		attributes.ExpiredAt = nil
	}
	if !privileged {
		attributes.UID = nil
		attributes.GID = nil
	}
	return attributes
}
