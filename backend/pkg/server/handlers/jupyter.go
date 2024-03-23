package handlers

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/crclient"
	tasksvc "github.com/raids-lab/crater/pkg/db/task"
	usersvc "github.com/raids-lab/crater/pkg/db/user"
	"github.com/raids-lab/crater/pkg/models"
	payload "github.com/raids-lab/crater/pkg/server/payload"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	"github.com/raids-lab/crater/pkg/util"
	log "github.com/sirupsen/logrus"
)

type JupyterMgr struct {
	taskService    tasksvc.DBService
	userService    usersvc.DBService
	pvcClient      *crclient.PVCClient
	logClient      *crclient.LogClient
	taskController *aitaskctl.TaskController
}

func (mgr *JupyterMgr) RegisterRoute(g *gin.RouterGroup) {
	g.POST("create", mgr.Create)
	g.POST("delete", mgr.Delete)
	g.GET("list", mgr.List)
	g.GET("getToken", mgr.GetToken)
	g.GET("getImages", mgr.GetImages)
}

func NewJupyterMgr(taskController *aitaskctl.TaskController, pvcClient *crclient.PVCClient, logClient *crclient.LogClient) *JupyterMgr {
	return &JupyterMgr{
		taskService:    tasksvc.NewDBService(),
		userService:    usersvc.NewDBService(),
		pvcClient:      pvcClient,
		logClient:      logClient,
		taskController: taskController,
	}
}

func (mgr *JupyterMgr) NotifyTaskUpdate(taskID uint, userName string, op util.TaskOperation) {
	mgr.taskController.TaskUpdated(util.TaskUpdateChan{
		TaskID:    taskID,
		UserName:  userName,
		Operation: op,
	})
}

func (mgr *JupyterMgr) Create(c *gin.Context) {
	log.Infof("Task Create, url: %s", c.Request.URL)
	var req payload.CreateJupyterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		log.Error(msg)
		resputil.HTTPError(c, http.StatusBadRequest, msg, resputil.NotSpecified)
		return
	}

	userContext, _ := util.GetUserFromGinContext(c)

	var taskAttr models.TaskAttr
	taskAttr.TaskName = req.TaskName
	taskAttr.UserName = userContext.UserName
	taskAttr.Namespace = userContext.Namespace
	taskAttr.SLO = 1
	taskAttr.TaskType = models.JupyterTask
	taskAttr.Image = req.Image
	taskAttr.ResourceRequest = req.ResourceRequest
	taskAttr.Command = "start.sh jupyter lab --allow-root --NotebookApp.base_url=/jupyter/%s/"
	taskAttr.WorkingDir = fmt.Sprintf("/home/%s", userContext.UserName)
	taskAttr.ShareDirs = req.ShareDirs
	taskAttr.SchedulerName = req.SchedulerName
	taskAttr.GPUModel = req.GPUModel

	if len(taskAttr.ShareDirs) > 0 {
		for pvcName := range taskAttr.ShareDirs {
			err := mgr.pvcClient.CheckOrCreateUserPvc(taskAttr.Namespace, pvcName)
			if err != nil {
				resputil.Error(c, fmt.Sprintf("get user pvc failed, err %v", err), resputil.NotSpecified)
				return
			}
		}
	}

	taskModel := models.FormatTaskAttrToModel(&taskAttr)
	err := mgr.taskService.Create(taskModel)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("create task failed, err %v", err), resputil.NotSpecified)
		return
	}
	mgr.NotifyTaskUpdate(taskModel.ID, taskModel.UserName, util.CreateTask)

	log.Infof("create task success, taskID: %d", taskModel.ID)
	resp := payload.CreateTaskResp{
		TaskID: taskModel.ID,
	}
	resputil.Success(c, resp)
}

//nolint:dupl // TODO: refactor
func (mgr *JupyterMgr) List(c *gin.Context) {
	var req payload.ListTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate list parameters failed, err %v", err), resputil.NotSpecified)
		return
	}
	userContext, _ := util.GetUserFromGinContext(c)
	taskModels, err := mgr.taskService.ListByUserAndTaskType(userContext.UserName, models.JupyterTask)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list task failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.ListTaskResp{
		Rows: taskModels,
	}
	resputil.Success(c, resp)
}

func (mgr *JupyterMgr) GetToken(c *gin.Context) {
	log.Infof("Task Token Get, url: %s", c.Request.URL)
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
	if taskModel.Status != "Running" {
		resp := payload.GetJupyterResp{
			Port:  0,
			Token: "",
		}
		log.Infof("task token not ready, taskID: %d", req.TaskID)
		resputil.Success(c, resp)
		return
	}
	if taskModel.Token != "" {
		resp := payload.GetJupyterResp{
			Port:  taskModel.NodePort,
			Token: taskModel.Token,
		}
		log.Infof("get task token success, taskID: %d", req.TaskID)
		resputil.Success(c, resp)
		return
	}

	// get log
	pods, err := mgr.logClient.GetPodsWithLabel(taskModel.Namespace, taskModel.JobName)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task log failed, err %v", err), resputil.NotSpecified)
		return
	}
	var token string
	re := regexp.MustCompile(`\?token=([a-zA-Z0-9]+)`)
	for i := range pods {
		pod := &pods[i]
		podLog, getPodLogsErr := mgr.logClient.GetPodLogs(pod)
		if getPodLogsErr != nil {
			resputil.Error(c, fmt.Sprintf("get task log failed, err %v", getPodLogsErr), resputil.NotSpecified)
			return
		}
		matches := re.FindStringSubmatch(podLog)
		if len(matches) >= 2 {
			token = matches[1]
			break
		}
	}

	// Get service port
	port, getServicePortErr := mgr.logClient.GetSvcPort(taskModel.Namespace, taskModel.JobName)
	if getServicePortErr != nil {
		resputil.Error(c, fmt.Sprintf("get service port failed, err %v", getServicePortErr), resputil.NotSpecified)
		return
	}

	// Save token to db
	err = mgr.taskService.UpdateToken(taskModel.ID, token)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("update task token failed, err %v", err), resputil.NotSpecified)
		return
	}
	err = mgr.taskService.UpdateNodePort(taskModel.ID, port)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("update task node port failed, err %v", err), resputil.NotSpecified)
		return
	}

	resp := payload.GetJupyterResp{
		Port:  port,
		Token: token,
	}
	log.Infof("get task token success, taskID: %d", req.TaskID)
	resputil.Success(c, resp)
}

//nolint:dupl // TODO: refactor
func (mgr *JupyterMgr) Delete(c *gin.Context) {
	log.Infof("Task Delete, url: %s", c.Request.URL)
	var req payload.DeleteTaskReq
	var err error
	if err = c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate delete parameters failed, err %v", err), resputil.NotSpecified)
		return
	}
	userContext, _ := util.GetUserFromGinContext(c)
	// check if task.username is same as username
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

	log.Infof("delete task success, taskID: %d", req.TaskID)
	resputil.Success(c, "")
}

func (mgr *JupyterMgr) GetImages(c *gin.Context) {
	// 现阶段先返回一个固定的镜像列表
	images := []string{
		"jupyter/base-notebook:ubuntu-22.04",
	}
	resp := payload.GetImagesResp{
		Images: images,
	}
	resputil.Success(c, resp)
}
