package aijob

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/crclient"
	tasksvc "github.com/raids-lab/crater/pkg/db/task"
	usersvc "github.com/raids-lab/crater/pkg/db/user"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	payload "github.com/raids-lab/crater/pkg/server/payload"
	"github.com/raids-lab/crater/pkg/util"
)

type AIJobMgr struct {
	taskService    tasksvc.DBService
	userService    usersvc.DBService
	pvcClient      *crclient.PVCClient
	logClient      *crclient.LogClient
	taskController *aitaskctl.TaskController
}

func NewAITaskMgr(taskController *aitaskctl.TaskController, pvcClient *crclient.PVCClient, logClient *crclient.LogClient) handler.Manager {
	return &AIJobMgr{
		taskService:    tasksvc.NewDBService(),
		userService:    usersvc.NewDBService(),
		pvcClient:      pvcClient,
		logClient:      logClient,
		taskController: taskController,
	}
}

func (mgr *AIJobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *AIJobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.List)
	g.GET("all", mgr.List)
	g.DELETE(":name", mgr.Delete)

	g.GET(":name/log", mgr.GetLogs)
	g.GET(":name/detail", mgr.Get)

	g.POST("custom", mgr.Create)
}

func (mgr *AIJobMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.List)
	g.GET(":name/detail", mgr.Get)
}

func (mgr *AIJobMgr) NotifyTaskUpdate(taskID uint, userName string, op util.TaskOperation) {
	mgr.taskController.TaskUpdated(util.TaskUpdateChan{
		TaskID:    taskID,
		UserName:  userName,
		Operation: op,
	})
}

func (mgr *AIJobMgr) Create(c *gin.Context) {
	logutils.Log.Infof("Task Create, url: %s", c.Request.URL)
	var req payload.CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		logutils.Log.Error(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}

	userContext, _ := util.GetUserFromGinContext(c)
	req.UserName = userContext.UserName
	req.Namespace = userContext.Namespace

	if req.ShareDirs != nil && len(req.ShareDirs) > 0 {
		for pvcName := range req.ShareDirs {
			err := mgr.pvcClient.CheckOrCreateUserPvc(req.Namespace, pvcName)
			if err != nil {
				resputil.Error(c, fmt.Sprintf("get user pvc failed, err %v", err), resputil.NotSpecified)
				return
			}
		}
	}

	taskModel := models.FormatTaskAttrToModel(&req.TaskAttr)
	err := mgr.taskService.Create(taskModel)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("create task failed, err %v", err), resputil.NotSpecified)
		return
	}
	mgr.NotifyTaskUpdate(taskModel.ID, taskModel.UserName, util.CreateTask)

	logutils.Log.Infof("create task success, taskID: %d", taskModel.ID)
	resp := payload.CreateTaskResp{
		TaskID: taskModel.ID,
	}
	resputil.Success(c, resp)
}

func (mgr *AIJobMgr) List(c *gin.Context) {
	var req payload.ListTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate list parameters failed, err %v", err), resputil.NotSpecified)
		return
	}

	userContext, _ := util.GetUserFromGinContext(c)
	taskModels, err := mgr.taskService.ListByUserAndTaskType(userContext.UserName, models.TrainingTask)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list task failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.ListTaskResp{
		Rows: taskModels,
	}
	resputil.Success(c, resp)
}

func (mgr *AIJobMgr) Get(c *gin.Context) {
	logutils.Log.Infof("Task Get, url: %s", c.Request.URL)
	var req payload.GetTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate get parameters failed, err %v", err), resputil.NotSpecified)
		return
	}
	userContext, _ := util.GetUserFromGinContext(c)
	taskModel, err := mgr.taskService.GetByUserAndID(userContext.UserName, req.TaskID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.GetTaskResp{
		AITask: *taskModel,
	}
	logutils.Log.Infof("get task success, taskID: %d", req.TaskID)
	resputil.Success(c, resp)
}

func (mgr *AIJobMgr) GetLogs(c *gin.Context) {
	logutils.Log.Infof("Task Get, url: %s", c.Request.URL)
	var req payload.GetTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate get parameters failed, err %v", err), resputil.NotSpecified)
		return
	}
	userContext, _ := util.GetUserFromGinContext(c)
	taskModel, err := mgr.taskService.GetByUserAndID(userContext.UserName, req.TaskID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}
	// get log
	pods, err := mgr.logClient.GetPodsWithLabel(taskModel.Namespace, taskModel.JobName)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task log failed, err %v", err), resputil.NotSpecified)
		return
	}
	var logs []string
	for i := range pods {
		pod := &pods[i]
		podLog, err := mgr.logClient.GetPodLogs(pod)
		if err != nil {
			resputil.Error(c, fmt.Sprintf("get task log failed, err %v", err), resputil.NotSpecified)
			return
		}
		logs = append(logs, podLog)
	}
	resp := payload.GetTaskLogResp{
		Logs: logs,
	}
	logutils.Log.Infof("get task success, taskID: %d", req.TaskID)
	resputil.Success(c, resp)
}

func (mgr *AIJobMgr) Delete(c *gin.Context) {
	logutils.Log.Infof("Task Delete, url: %s", c.Request.URL)
	var req payload.DeleteTaskReq
	var err error
	if err = c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate delete parameters failed, err %v", err), resputil.NotSpecified)
		return
	}
	userContext, _ := util.GetUserFromGinContext(c)
	// check if user is authorized to delete the task
	_, err = mgr.taskService.GetByUserAndID(userContext.UserName, req.TaskID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}
	mgr.NotifyTaskUpdate(req.TaskID, userContext.UserName, util.DeleteTask)
	if req.ForceDelete {
		err = mgr.taskService.ForceDeleteByUserAndID(userContext.UserName, req.TaskID)
	} else {
		err = mgr.taskService.DeleteByUserAndID(userContext.UserName, req.TaskID)
	}
	if err != nil {
		resputil.Error(c, fmt.Sprintf("delete task failed, err %v", err), resputil.NotSpecified)
		return
	}

	logutils.Log.Infof("delete task success, taskID: %d", req.TaskID)
	resputil.Success(c, "")
}

func (mgr *AIJobMgr) UpdateSLO(c *gin.Context) {
	logutils.Log.Infof("Task Update, url: %s", c.Request.URL)
	var req payload.UpdateTaskSLOReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, err %v", err), resputil.NotSpecified)
		return
	}
	userContext, _ := util.GetUserFromGinContext(c)
	task, err := mgr.taskService.GetByUserAndID(userContext.UserName, req.TaskID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}
	task.SLO = req.SLO
	err = mgr.taskService.Update(task)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("update task failed, err %v", err), resputil.NotSpecified)
		return
	}
	mgr.NotifyTaskUpdate(req.TaskID, userContext.UserName, util.UpdateTask)
	logutils.Log.Infof("update task success, taskID: %d", req.TaskID)
	resputil.Success(c, "")
}

func (mgr *AIJobMgr) GetQuota(c *gin.Context) {
	userContext, _ := util.GetUserFromGinContext(c)
	quotaInfo := mgr.taskController.GetQuotaInfoSnapshotByUsername(userContext.UserName)
	if quotaInfo == nil {
		resputil.Error(c, fmt.Sprintf("get user:%v quota failed", userContext.UserName), resputil.NotSpecified)
		return
	}
	resp := payload.GetQuotaResp{
		Hard:     quotaInfo.Hard,
		HardUsed: quotaInfo.HardUsed,
		SoftUsed: quotaInfo.SoftUsed,
	}
	resputil.Success(c, resp)
}

func (mgr *AIJobMgr) GetTaskStats(c *gin.Context) {
	userContext, _ := util.GetUserFromGinContext(c)
	taskCountList, err := mgr.taskService.GetUserTaskStatusCount(userContext.UserName)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task count statistic failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.AITaskStatistic{
		TaskCount: taskCountList,
	}
	resputil.Success(c, resp)
}
