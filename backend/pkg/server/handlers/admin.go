package handlers

import (
	"fmt"

	quotasvc "github.com/aisystem/ai-protal/pkg/db/quota"
	usersvc "github.com/aisystem/ai-protal/pkg/db/user"
	"github.com/aisystem/ai-protal/pkg/models"
	payload "github.com/aisystem/ai-protal/pkg/server/payload"
	resputil "github.com/aisystem/ai-protal/pkg/server/response"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (mgr *AdminMgr) RegisterRoute(g *gin.RouterGroup) {

	// g.POST("createUser", mgr.CreateUser)
	g.GET("listUser", mgr.ListUser)
	g.GET("getUser", mgr.GetUser)
	g.POST("updateQuota", mgr.UpdateQuota)
	g.POST("deleteUser", mgr.DeleteUser)
	g.POST("updateRole", mgr.UpdateRole)

}

type AdminMgr struct {
	quotaService quotasvc.DBService
	userService  usersvc.DBService
	// taskUpdateChan <-chan util.TaskUpdateChan
}

func NewAdminMgr() *AdminMgr {
	return &AdminMgr{
		quotaService: quotasvc.NewDBService(),
		userService:  usersvc.NewDBService(),
	}
}

// func (mgr *AdminMgr) CreateUser(c *gin.Context) {
// 	log.Infof("User Create, url: %s", c.Request.URL)

// 	resputil.WrapSuccessResponse(c, "")
// }

func (mgr *AdminMgr) DeleteUser(c *gin.Context) {
	log.Infof("User Delete, url: %s", c.Request.URL)
	var req payload.DeleteUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50008)
		return
	}
	err := mgr.userService.DeleteByUserName(req.UserName)
	if err != nil {
		msg := fmt.Sprintf("delete user failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50009)
		return
	}
	// TODO: delete resource
	log.Infof("delete user success, username: %s", req.UserName)
	resputil.WrapSuccessResponse(c, "")
}

func (mgr *AdminMgr) ListUser(c *gin.Context) {
	log.Infof("User List, url: %s", c.Request.URL)
	userQuotas, err := mgr.userService.ListAllUserQuotas()
	if err != nil {
		msg := fmt.Sprintf("list user failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50001)
		return
	}

	resp := payload.ListUserResp{
		Users: make([]payload.GetUserResp, 0),
	}
	for _, userQuota := range userQuotas {
		userResp := payload.GetUserResp{
			UserID:    userQuota.User.ID,
			UserName:  userQuota.User.UserName,
			Role:      userQuota.User.Role,
			CreatedAt: userQuota.User.CreatedAt,
			UpdatedAt: userQuota.User.UpdatedAt,
		}
		resList, err := models.JSONToResourceList(userQuota.Quota.HardQuota)
		if err == nil {
			userResp.QuotaHard = resList
		}
		resp.Users = append(resp.Users, userResp)
	}

	log.Infof("list users success, taskNum: %d", len(resp.Users))
	resputil.WrapSuccessResponse(c, resp)
}

func (mgr *AdminMgr) GetUser(c *gin.Context) {
	log.Infof("User Get, url: %s", c.Request.URL)
	var req payload.GetUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate get parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50004)
		return
	}
	var user *models.User
	var err error
	if req.UserName != "" {
		user, err = mgr.userService.GetByUserName(req.UserName)
	} else {
		user, err = mgr.userService.GetUserByID(req.UserID)
	}
	if err != nil {
		msg := fmt.Sprintf("get user failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50001)
		return
	}
	resp := payload.GetUserResp{
		UserID:    user.ID,
		UserName:  user.UserName,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	log.Infof("get user success, user: %d", resp.UserName)
	resputil.WrapSuccessResponse(c, resp)
}

func (mgr *AdminMgr) UpdateQuota(c *gin.Context) {
	log.Infof("Quota Update, url: %s", c.Request.URL)
	var req payload.UpdateQuotaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate update parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50008)
		return
	}
	user, err := mgr.userService.GetByUserName(req.UserName)
	if err != nil {
		msg := fmt.Sprintf("get user failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50009)
		return
	}
	quota, err := mgr.quotaService.GetByUserName(req.UserName)
	if err != nil {
		quota = &models.Quota{
			// UserID:    user.ID,
			UserName:  user.UserName,
			NameSpace: fmt.Sprintf("user-%s", user.UserName),
		}
	}
	quota.HardQuota = models.ResourceListToJSON(req.HardQuota)
	err = mgr.quotaService.Update(quota)
	log.Infof("update quota success, user: %d, quota:%v", req.UserName, req.HardQuota)
	resputil.WrapSuccessResponse(c, "")
}

func (mgr *AdminMgr) UpdateRole(c *gin.Context) {
	log.Infof("Role Update, url: %s", c.Request.URL)
	var req payload.UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate update parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50008)
		return
	}

	if req.Role == models.RoleAdmin || req.Role == models.RoleUser {
		// do nothing
	} else {
		msg := fmt.Sprintf("role %s is not valid", req.Role)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50010)
		return
	}
	err := mgr.userService.UpdateRole(req.UserName, req.Role)
	if err != nil {
		msg := fmt.Sprintf("update user role failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50011)
		return
	}
	log.Infof("update user role success, user: %s, role: %s", req.UserName, req.Role)
	resputil.WrapSuccessResponse(c, "")
}
