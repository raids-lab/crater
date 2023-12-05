package handlers

import (
	"fmt"

	"github.com/aisystem/ai-protal/pkg/aitaskctl"
	quotasvc "github.com/aisystem/ai-protal/pkg/db/quota"
	tasksvc "github.com/aisystem/ai-protal/pkg/db/task"
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
	g.GET("listQuota", mgr.ListQuota)
	g.GET("listByTaskType", mgr.ListTaskByTaskType)
	g.GET("getTaskCountStatistic", mgr.GetTaskCountStatistic)
	g.POST("updateQuota", mgr.UpdateQuota)
	g.POST("deleteUser", mgr.DeleteUser)
	g.POST("updateRole", mgr.UpdateRole)

}

type AdminMgr struct {
	quotaService   quotasvc.DBService
	userService    usersvc.DBService
	taskServcie    tasksvc.DBService
	taskController *aitaskctl.TaskController
}

func NewAdminMgr(taskController *aitaskctl.TaskController) *AdminMgr {
	return &AdminMgr{
		quotaService:   quotasvc.NewDBService(),
		userService:    usersvc.NewDBService(),
		taskServcie:    tasksvc.NewDBService(),
		taskController: taskController,
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

func (mgr *AdminMgr) ListQuota(c *gin.Context) {
	log.Infof("Quota List, url: %s", c.Request.URL)
	userQuotas := mgr.taskController.ListQuotaInfoSnapshot()

	resp := payload.ListUserQuotaResp{
		Quotas: make([]payload.GetQuotaResp, 0),
	}
	for _, quotaInfo := range userQuotas {
		r := payload.GetQuotaResp{
			User:     quotaInfo.Name,
			Hard:     quotaInfo.Hard,
			HardUsed: quotaInfo.HardUsed,
			SoftUsed: quotaInfo.SoftUsed,
		}
		resp.Quotas = append(resp.Quotas, r)
	}

	log.Infof("list users success, taskNum: %d", len(resp.Quotas))
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
	if err != nil {
		msg := fmt.Sprintf("update quota failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50010)
		return
	}
	// notify taskController to update quota
	mgr.taskController.AddOrUpdateQuotaInfo(quota.UserName, *quota)

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

func (mgr *AdminMgr) ListTaskByTaskType(c *gin.Context) {
	log.Infof("Task List, url: %s", c.Request.URL)
	var req payload.ListTaskByTypeReq
	if err := c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate list parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50002)
		return
	}

	taskModels, err := mgr.taskServcie.ListByTaskType(req.TaskType)
	if err != nil {
		msg := fmt.Sprintf("list task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50003)
		return
	}
	resp := payload.ListTaskResp{
		Tasks: taskModels,
	}
	// log.Infof("list task success, taskNum: %d", len(resp.Tasks))
	resputil.WrapSuccessResponse(c, resp)
}

func (mgr *AdminMgr) GetTaskCountStatistic(c *gin.Context) {
	log.Infof("Task Count Statistic, url: %s", c.Request.URL)
	taskCountList, err := mgr.taskServcie.GetTaskStatusCount()
	if err != nil {
		msg := fmt.Sprintf("get task count statistic failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50003)
		return
	}
	var resp payload.AITaskCountStatistic
	for _, taskCount := range taskCountList {
		switch taskCount.Status {
		case models.TaskQueueingStatus:
			resp.QueueingTaskNum += taskCount.Count
		case models.TaskRunningStatus:
			resp.RunningTaskNum += taskCount.Count
		case models.TaskCreatedStatus:
			resp.PendingTaskNum += taskCount.Count
		case models.TaskPendingStatus:
			resp.PendingTaskNum += taskCount.Count
		case models.TaskPreemptedStatus:
			resp.PendingTaskNum += taskCount.Count
		case models.TaskSucceededStatus:
			resp.FinishedTaskNum += taskCount.Count
		case models.TaskFailedStatus:
			resp.FinishedTaskNum += taskCount.Count
		}
	}
	// log.Infof("list task success, taskNum: %d", len(resp.Tasks))
	resputil.WrapSuccessResponse(c, resp)
}
