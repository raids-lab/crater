package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/crclient"
	quotasvc "github.com/raids-lab/crater/pkg/db/quota"
	tasksvc "github.com/raids-lab/crater/pkg/db/task"
	usersvc "github.com/raids-lab/crater/pkg/db/user"
	"github.com/raids-lab/crater/pkg/models"
	payload "github.com/raids-lab/crater/pkg/server/payload"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	log "github.com/sirupsen/logrus"
)

type AdminMgr struct {
	quotaService   quotasvc.DBService
	userService    usersvc.DBService
	taskServcie    tasksvc.DBService
	taskController *aitaskctl.TaskController
	nodeClient     *crclient.NodeClient
}

func (mgr *AdminMgr) RegisterRoute(g *gin.RouterGroup) {
	users := g.Group("/users")
	users.GET("", mgr.ListUser)
	users.GET("/:name", mgr.GetUser)
	users.DELETE("/:name", mgr.DeleteUser)
	users.PUT("/:name/role", mgr.UpdateRole)

	quotas := g.Group("/quotas")
	quotas.GET("", mgr.ListQuota)
	quotas.PUT("/:name", mgr.UpdateQuota)

	tasks := g.Group("/tasks")
	tasks.GET("", mgr.ListTaskByTaskType)
	tasks.GET("/stats", mgr.GetTaskStats)

	nodes := g.Group("/nodes")
	nodes.GET("", mgr.ListNode)
}

func NewAdminMgr(taskController *aitaskctl.TaskController, nodeClient *crclient.NodeClient) *AdminMgr {
	return &AdminMgr{
		quotaService:   quotasvc.NewDBService(),
		userService:    usersvc.NewDBService(),
		taskServcie:    tasksvc.NewDBService(),
		taskController: taskController,
		nodeClient:     nodeClient,
	}
}

func (mgr *AdminMgr) DeleteUser(c *gin.Context) {
	log.Infof("User Delete, url: %s", c.Request.URL)
	name := c.Param("name")
	err := mgr.userService.DeleteByUserName(name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("delete user failed, err %v", err), resputil.NotSpecified)
		return
	}
	// TODO: delete resource
	log.Infof("delete user success, username: %s", name)
	resputil.Success(c, "")
}

func (mgr *AdminMgr) ListUser(c *gin.Context) {
	log.Infof("User List, url: %s", c.Request.URL)
	userQuotas, err := mgr.userService.ListAllUserQuotas()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list user failed, err %v", err), resputil.NotSpecified)
		return
	}

	resp := payload.ListUserResp{
		Users: make([]payload.GetUserResp, 0),
	}
	for i := range userQuotas {
		userQuota := &userQuotas[i]
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
	resputil.Success(c, resp)
}

func (mgr *AdminMgr) ListQuota(c *gin.Context) {
	log.Infof("Quota List, url: %s", c.Request.URL)
	userQuotas := mgr.taskController.ListQuotaInfoSnapshot()

	resp := payload.ListUserQuotaResp{
		Quotas: make([]payload.GetQuotaResp, 0),
	}
	for i := 0; i < len(userQuotas); i++ {
		quotaInfo := &userQuotas[i]
		r := payload.GetQuotaResp{
			User:     quotaInfo.Name,
			Hard:     quotaInfo.Hard,
			HardUsed: quotaInfo.HardUsed,
			SoftUsed: quotaInfo.SoftUsed,
		}
		resp.Quotas = append(resp.Quotas, r)
	}

	log.Infof("list users success, taskNum: %d", len(resp.Quotas))
	resputil.Success(c, resp)
}

func (mgr *AdminMgr) GetUser(c *gin.Context) {
	name := c.Param("name")
	var user *models.User
	var err error
	user, err = mgr.userService.GetByUserName(name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get user failed, err %v", err), resputil.NotSpecified)
		return
	}

	resp := payload.GetUserResp{
		UserID:    user.ID,
		UserName:  user.UserName,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	log.Infof("get user success, user: %s", resp.UserName)
	resputil.Success(c, resp)
}

func (mgr *AdminMgr) UpdateQuota(c *gin.Context) {
	log.Infof("Quota Update, url: %s", c.Request.URL)
	var req payload.UpdateQuotaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, err %v", err), resputil.NotSpecified)
		return
	}
	name := c.Param("name")
	user, err := mgr.userService.GetByUserName(name)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get user failed, err %v", err), resputil.NotSpecified)
		return
	}
	quota, err := mgr.quotaService.GetByUserName(name)
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
		resputil.Error(c, fmt.Sprintf("update quota failed, err %v", err), resputil.NotSpecified)
		return
	}
	// notify taskController to update quota
	mgr.taskController.AddOrUpdateQuotaInfo(quota.UserName, quota)

	log.Infof("update quota success, user: %s, quota:%v", name, req.HardQuota)
	resputil.Success(c, "")
}

func (mgr *AdminMgr) UpdateRole(c *gin.Context) {
	log.Infof("Role Update, url: %s", c.Request.URL)
	var req payload.UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, err %v", err), resputil.NotSpecified)
		return
	}
	name := c.Param("name")
	if req.Role == models.RoleAdmin || req.Role == models.RoleUser {
		// do nothing
	} else {
		resputil.Error(c, fmt.Sprintf("role %s is not valid", req.Role), resputil.NotSpecified)
		return
	}
	err := mgr.userService.UpdateRole(name, req.Role)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("update user role failed, err %v", err), resputil.NotSpecified)
		return
	}
	log.Infof("update user role success, user: %s, role: %s", name, req.Role)
	resputil.Success(c, "")
}

func (mgr *AdminMgr) ListTaskByTaskType(c *gin.Context) {
	log.Infof("Task List, url: %s", c.Request.URL)
	var req payload.ListTaskByTypeReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate list parameters failed, err %v", err), resputil.NotSpecified)
		return
	}

	taskModels, totalRows, err := mgr.taskServcie.ListByTaskType(req.TaskType, req.PageIndex, req.PageSize)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list task failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.ListTaskResp{
		Rows:     taskModels,
		RowCount: totalRows,
	}
	resputil.Success(c, resp)
}

func (mgr *AdminMgr) GetTaskStats(c *gin.Context) {
	log.Infof("Task Count Statistic, url: %s", c.Request.URL)
	taskCountList, err := mgr.taskServcie.GetTaskStatusCount()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task count statistic failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.AITaskStatistic{
		TaskCount: taskCountList,
	}
	resputil.Success(c, resp)
}

func (mgr *AdminMgr) ListNode(c *gin.Context) {
	log.Infof("Node List, url: %s", c.Request.URL)
	// get all k8s nodes by k8s client
	nodes, err := mgr.nodeClient.ListNodes()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list nodes failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.ListNodeResp{
		Rows: nodes,
	}
	resputil.Success(c, resp)
}
