package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
)

type UserBanPolicyReq struct {
	AllowPlatformAccess  bool `json:"allowPlatformAccess"`
	AllowJobSubmission   bool `json:"allowJobSubmission"`
	AllowImageBuild      bool `json:"allowImageBuild"`
	AllowModelDownload   bool `json:"allowModelDownload"`
	AllowDatasetDownload bool `json:"allowDatasetDownload"`
}

// GetUserBanPolicy godoc
// @Summary      获取用户封禁能力策略
// @Description 获取封禁用户仍被允许使用的平台能力
// @Tags         SystemConfig
// @Produce      json
// @Security     Bearer
// @Success      200 {object} resputil.Response[UserBanPolicyReq]
// @Router       /v1/admin/system-config/user-ban [get]
func (mgr *SystemConfigMgr) GetUserBanPolicy(c *gin.Context) {
	policy, err := mgr.userBanService.GetPolicy(c.Request.Context())
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, UserBanPolicyReq(policy))
}

// UpdateUserBanPolicy godoc
// @Summary      更新用户封禁能力策略
// @Description 设置封禁用户是否仍可访问平台、提交作业、制作镜像以及下载模型或数据集
// @Tags         SystemConfig
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        data body UserBanPolicyReq true "封禁能力策略"
// @Success      200 {object} resputil.Response[UserBanPolicyReq]
// @Router       /v1/admin/system-config/user-ban [put]
func (mgr *SystemConfigMgr) UpdateUserBanPolicy(c *gin.Context) {
	var req UserBanPolicyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid request body"))
		return
	}
	policy := service.UserBanPolicy(req)
	if err := mgr.userBanService.UpdatePolicy(c.Request.Context(), policy); err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, req)
}
