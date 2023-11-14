package handlers

import (
	"fmt"
	"net/http"

	"github.com/aisystem/ai-protal/pkg/aitaskctl"
	tasksvc "github.com/aisystem/ai-protal/pkg/db/task"
	usersvc "github.com/aisystem/ai-protal/pkg/db/user"
	"github.com/aisystem/ai-protal/pkg/models"
	payload "github.com/aisystem/ai-protal/pkg/server/payload"
	resputil "github.com/aisystem/ai-protal/pkg/server/response"
	"github.com/aisystem/ai-protal/pkg/util"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (mgr *AITaskMgr) RegisterRoute(g *gin.RouterGroup) {
	g.POST("create", mgr.Create)
	g.POST("delete", mgr.Delete)
	g.POST("updateSLO", mgr.UpdateSLO)
	g.GET("list", mgr.List)
	g.GET("get", mgr.Get)
	g.GET("getQuota", mgr.GetQuota)
}

type AITaskMgr struct {
	taskService tasksvc.DBService
	userService usersvc.DBService
	// taskUpdateChan chan<- util.TaskUpdateChan
	taskController *aitaskctl.TaskController
}

func NewAITaskMgr(taskController *aitaskctl.TaskController) *AITaskMgr {
	return &AITaskMgr{
		taskService: tasksvc.NewDBService(),
		userService: usersvc.NewDBService(),
		// taskUpdateChan: taskUpdateChan,
		taskController: taskController,
	}
}

func (mgr *AITaskMgr) NotifyTaskUpdate(taskID uint, userName string, op util.TaskOperation) {
	mgr.taskController.TaskUpdated(util.TaskUpdateChan{
		TaskID:    taskID,
		UserName:  userName,
		Operation: op,
	})
}

func (mgr *AITaskMgr) Create(c *gin.Context) {
	log.Infof("Task Create, url: %s", c.Request.URL)
	var req payload.CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		log.Error(msg)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: msg,
			Code:    40001,
		})
		return
	}
	username, _ := c.Get("username")
	req.UserName = username.(string)
	req.Namespace = fmt.Sprintf("user-%s", username.(string))
	taskModel := models.FormatTaskAttrToModel(&req.TaskAttr)
	err := mgr.taskService.Create(taskModel)
	if err != nil {
		msg := fmt.Sprintf("create task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50001)
		return
	}
	mgr.NotifyTaskUpdate(taskModel.ID, taskModel.UserName, util.CreateTask)

	log.Infof("create task success, taskID: %d", req.ID)
	resputil.WrapSuccessResponse(c, "")
}

func (mgr *AITaskMgr) List(c *gin.Context) {
	// log.Infof("Task List, url: %s", c.Request.URL)
	var req payload.ListTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate list parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50002)
		return
	}
	username, _ := c.Get("username")
	taskModels, err := mgr.taskService.ListByUserAndStatus(username.(string), "")
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

func (mgr *AITaskMgr) Get(c *gin.Context) {
	log.Infof("Task Get, url: %s", c.Request.URL)
	var req payload.GetTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate get parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50004)
		return
	}
	username, _ := c.Get("username")
	taskModel, err := mgr.taskService.GetByUserAndID(username.(string), req.TaskID)
	if err != nil {
		msg := fmt.Sprintf("get task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50005)
		return
	}
	resp := payload.GetTaskResp{
		*taskModel,
	}
	log.Infof("get task success, taskID: %d", req.TaskID)
	resputil.WrapSuccessResponse(c, resp)
}

func (mgr *AITaskMgr) Delete(c *gin.Context) {
	log.Infof("Task Delete, url: %s", c.Request.URL)
	var req payload.DeleteTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50006)
		return
	}
	username, _ := c.Get("username")
	err := mgr.taskService.DeleteByUserAndID(username.(string), req.TaskID)
	if err != nil {
		msg := fmt.Sprintf("delete task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50007)
		return
	}
	mgr.NotifyTaskUpdate(req.TaskID, username.(string), util.DeleteTask)
	log.Infof("delete task success, taskID: %d", req.TaskID)
	resputil.WrapSuccessResponse(c, "")
}

func (mgr *AITaskMgr) UpdateSLO(c *gin.Context) {
	log.Infof("Task Update, url: %s", c.Request.URL)
	var req payload.UpdateTaskSLOReq
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate update parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50008)
		return
	}
	username, _ := c.Get("username")
	task, err := mgr.taskService.GetByUserAndID(username.(string), req.TaskID)
	if err != nil {
		msg := fmt.Sprintf("get task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50009)
		return
	}
	task.SLO = req.SLO
	err = mgr.taskService.Update(task)
	mgr.NotifyTaskUpdate(req.TaskID, username.(string), util.UpdateTask)
	log.Infof("update task success, taskID: %d", req.TaskID)
	resputil.WrapSuccessResponse(c, "")
}

func (mgr *AITaskMgr) GetQuota(c *gin.Context) {
	username, _ := c.Get("username")
	quotaInfo := mgr.taskController.GetQuotaInfoSnapshotByUsername(username.(string))
	if quotaInfo == nil {
		msg := fmt.Sprintf("get user:%v quota failed", username.(string))
		log.Errorf(msg)
		resputil.WrapFailedResponse(c, msg, 50009)
		return
	}
	resp := payload.GetQuotaResp{
		Hard:     quotaInfo.Hard,
		HardUsed: quotaInfo.HardUsed,
		SoftUsed: quotaInfo.SoftUsed,
	}
	log.Infof("get quota success, user: %v", username.(string))
	resputil.WrapSuccessResponse(c, resp)
}
