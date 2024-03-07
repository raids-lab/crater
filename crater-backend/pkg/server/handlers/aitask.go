package handlers

import (
	"fmt"
	"net/http"

	"github.com/aisystem/ai-protal/pkg/aitaskctl"
	"github.com/aisystem/ai-protal/pkg/crclient"
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
	g.GET("getLogs", mgr.GetLogs)
	g.GET("getQuota", mgr.GetQuota)
	g.GET("taskStats", mgr.GetTaskStats)

}

type AITaskMgr struct {
	taskService tasksvc.DBService
	userService usersvc.DBService
	pvcClient   *crclient.PVCClient
	logClient   *crclient.LogClient
	// taskUpdateChan chan<- util.TaskUpdateChan
	taskController *aitaskctl.TaskController
}

func NewAITaskMgr(taskController *aitaskctl.TaskController, pvcClient *crclient.PVCClient, logClient *crclient.LogClient) *AITaskMgr {
	return &AITaskMgr{
		taskService:    tasksvc.NewDBService(),
		userService:    usersvc.NewDBService(),
		pvcClient:      pvcClient,
		logClient:      logClient,
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

	if req.ShareDirs != nil && len(req.ShareDirs) > 0 {
		for pvcName := range req.ShareDirs {
			err := mgr.pvcClient.CheckOrCreateUserPvc(req.Namespace, pvcName)
			if err != nil {
				resputil.Error(c, fmt.Sprintf("get user pvc failed, err %v", err), 50001)
				return
			}
		}
	}

	taskModel := models.FormatTaskAttrToModel(&req.TaskAttr)
	err := mgr.taskService.Create(taskModel)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("create task failed, err %v", err), 50001)
		return
	}
	mgr.NotifyTaskUpdate(taskModel.ID, taskModel.UserName, util.CreateTask)

	log.Infof("create task success, taskID: %d", taskModel.ID)
	resp := payload.CreateTaskResp{
		TaskID: taskModel.ID,
	}
	resputil.Success(c, resp)
}

func (mgr *AITaskMgr) List(c *gin.Context) {
	// log.Infof("Task List, url: %s", c.Request.URL)
	var req payload.ListTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate list parameters failed, err %v", err), 50002)
		return
	}
	username, _ := c.Get("username")
	// taskModels, err := mgr.taskService.ListByUserAndStatuses(username.(string), nil)
	taskModels, err := mgr.taskService.ListByUserAndTaskType(username.(string), models.TrainingTask)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list task failed, err %v", err), 50003)
		return
	}
	resp := payload.ListTaskResp{
		Tasks: taskModels,
	}
	// log.Infof("list task success, taskNum: %d", len(resp.Tasks))
	resputil.Success(c, resp)
}

func (mgr *AITaskMgr) Get(c *gin.Context) {
	log.Infof("Task Get, url: %s", c.Request.URL)
	var req payload.GetTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate get parameters failed, err %v", err), 50004)
		return
	}
	username, _ := c.Get("username")
	taskModel, err := mgr.taskService.GetByUserAndID(username.(string), req.TaskID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), 50005)
		return
	}
	resp := payload.GetTaskResp{
		AITask: *taskModel,
	}
	log.Infof("get task success, taskID: %d", req.TaskID)
	resputil.Success(c, resp)
}

func (mgr *AITaskMgr) GetLogs(c *gin.Context) {
	log.Infof("Task Get, url: %s", c.Request.URL)
	var req payload.GetTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate get parameters failed, err %v", err), 50004)
		return
	}
	username, _ := c.Get("username")
	taskModel, err := mgr.taskService.GetByUserAndID(username.(string), req.TaskID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), 50005)
		return
	}
	// get log
	pods, err := mgr.logClient.GetPodsWithLabel(taskModel.Namespace, taskModel.JobName)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task log failed, err %v", err), 50005)
		return
	}
	var logs []string
	for _, pod := range pods {
		podLog, err := mgr.logClient.GetPodLogs(pod)
		if err != nil {
			resputil.Error(c, fmt.Sprintf("get task log failed, err %v", err), 50005)
			return
		}
		logs = append(logs, podLog)
	}
	resp := payload.GetTaskLogResp{
		Logs: logs,
	}
	log.Infof("get task success, taskID: %d", req.TaskID)
	resputil.Success(c, resp)
}

func (mgr *AITaskMgr) Delete(c *gin.Context) {
	log.Infof("Task Delete, url: %s", c.Request.URL)
	var req payload.DeleteTaskReq
	var err error
	if err = c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate delete parameters failed, err %v", err), 50006)
		return
	}
	username, _ := c.Get("username")
	// check if user is authorized to delete the task
	_, err = mgr.taskService.GetByUserAndID(username.(string), req.TaskID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), 50006)
		return
	}
	mgr.NotifyTaskUpdate(req.TaskID, username.(string), util.DeleteTask)
	if req.ForceDelete {
		err = mgr.taskService.ForceDeleteByUserAndID(username.(string), req.TaskID)
	} else {
		err = mgr.taskService.DeleteByUserAndID(username.(string), req.TaskID)
	}
	if err != nil {
		resputil.Error(c, fmt.Sprintf("delete task failed, err %v", err), 50007)
		return
	}

	log.Infof("delete task success, taskID: %d", req.TaskID)
	resputil.Success(c, "")
}

func (mgr *AITaskMgr) UpdateSLO(c *gin.Context) {
	log.Infof("Task Update, url: %s", c.Request.URL)
	var req payload.UpdateTaskSLOReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, err %v", err), 50008)
		return
	}
	username, _ := c.Get("username")
	task, err := mgr.taskService.GetByUserAndID(username.(string), req.TaskID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), 50009)
		return
	}
	task.SLO = req.SLO
	err = mgr.taskService.Update(task)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("update task failed, err %v", err), 50009)
		return
	}
	mgr.NotifyTaskUpdate(req.TaskID, username.(string), util.UpdateTask)
	log.Infof("update task success, taskID: %d", req.TaskID)
	resputil.Success(c, "")
}

func (mgr *AITaskMgr) GetQuota(c *gin.Context) {
	username, _ := c.Get("username")
	quotaInfo := mgr.taskController.GetQuotaInfoSnapshotByUsername(username.(string))
	if quotaInfo == nil {
		resputil.Error(c, fmt.Sprintf("get user:%v quota failed", username.(string)), 50009)
		return
	}
	resp := payload.GetQuotaResp{
		Hard:     quotaInfo.Hard,
		HardUsed: quotaInfo.HardUsed,
		SoftUsed: quotaInfo.SoftUsed,
	}
	// log.Infof("get quota success, user: %v", username.(string))
	resputil.Success(c, resp)
}

func (mgr *AITaskMgr) GetTaskStats(c *gin.Context) {
	// log.Infof("Task Count Statistic, url: %s", c.Request.URL)
	username, _ := c.Get("username")
	taskCountList, err := mgr.taskService.GetUserTaskStatusCount(username.(string))
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task count statistic failed, err %v", err), 50003)
		return
	}
	// var respCnt payload.AITaskCountStatistic
	// for _, taskCount := range taskCountList {
	// 	switch taskCount.Status {
	// 	case models.TaskQueueingStatus:
	// 		respCnt.Queueing += taskCount.Count
	// 	case models.TaskRunningStatus:
	// 		respCnt.Running += taskCount.Count
	// 	case models.TaskCreatedStatus:
	// 		respCnt.Pending += taskCount.Count
	// 	case models.TaskPendingStatus:
	// 		respCnt.Pending += taskCount.Count
	// 	case models.TaskPreemptedStatus:
	// 		respCnt.Pending += taskCount.Count
	// 	case models.TaskSucceededStatus:
	// 		respCnt.Finished += taskCount.Count
	// 	case models.TaskFailedStatus:
	// 		respCnt.Finished += taskCount.Count
	// 	}
	// }
	resp := payload.AITaskStatistic{
		TaskCount: taskCountList,
	}
	// log.Infof("list task success, taskNum: %d", len(resp.Tasks))
	resputil.Success(c, resp)
}
