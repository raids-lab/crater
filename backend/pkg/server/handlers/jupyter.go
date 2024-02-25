package handlers

import (
	"fmt"
	"net/http"
	"regexp"

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

func (mgr *JupyterMgr) RegisterRoute(g *gin.RouterGroup) {
	g.POST("create", mgr.Create)
	g.POST("delete", mgr.Delete)
	g.GET("list", mgr.List)
	g.GET("getToken", mgr.GetToken)
	g.GET("getImages", mgr.GetImages)
}

type JupyterMgr struct {
	taskService    tasksvc.DBService
	userService    usersvc.DBService
	pvcClient      *crclient.PVCClient
	logClient      *crclient.LogClient
	taskController *aitaskctl.TaskController
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
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: msg,
			Code:    40001,
		})
		return
	}
	username, _ := c.Get("username")

	var taskAttr models.TaskAttr
	taskAttr.TaskName = req.TaskName
	taskAttr.UserName = username.(string)
	taskAttr.Namespace = fmt.Sprintf("user-%s", username.(string))
	taskAttr.SLO = 1
	taskAttr.TaskType = models.JupyterTask
	taskAttr.Image = req.Image
	taskAttr.ResourceRequest = req.ResourceRequest
	taskAttr.Command = "start.sh jupyter lab --allow-root"
	taskAttr.WorkingDir = fmt.Sprintf("/home/%s", username.(string))
	taskAttr.ShareDirs = req.ShareDirs

	if len(taskAttr.ShareDirs) > 0 {
		for pvcName := range taskAttr.ShareDirs {
			err := mgr.pvcClient.CheckOrCreateUserPvc(taskAttr.Namespace, pvcName)
			if err != nil {
				msg := fmt.Sprintf("get user pvc failed, err %v", err)
				log.Error(msg)
				resputil.WrapFailedResponse(c, msg, 50001)
				return
			}
		}
	}

	taskModel := models.FormatTaskAttrToModel(&taskAttr)
	err := mgr.taskService.Create(taskModel)
	if err != nil {
		msg := fmt.Sprintf("create task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50001)
		return
	}
	mgr.NotifyTaskUpdate(taskModel.ID, taskModel.UserName, util.CreateTask)

	log.Infof("create task success, taskID: %d", taskModel.ID)
	resp := payload.CreateTaskResp{
		TaskID: taskModel.ID,
	}
	resputil.WrapSuccessResponse(c, resp)
}

func (mgr *JupyterMgr) List(c *gin.Context) {
	// log.Infof("Task List, url: %s", c.Request.URL)
	var req payload.ListTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate list parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50002)
		return
	}
	username, _ := c.Get("username")
	taskModels, err := mgr.taskService.ListByUserAndTaskType(username.(string), models.JupyterTask)
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

func (mgr *JupyterMgr) GetToken(c *gin.Context) {
	log.Infof("Task Token Get, url: %s", c.Request.URL)
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
	if taskModel.Status != "Running" {
		resp := payload.GetJupyterResp{
			Port:  0,
			Token: "",
		}
		log.Infof("task token not ready, taskID: %d", req.TaskID)
		resputil.WrapSuccessResponse(c, resp)
		return
	}
	if taskModel.Token != "" {
		resp := payload.GetJupyterResp{
			Port:  taskModel.NodePort,
			Token: taskModel.Token,
		}
		log.Infof("get task token success, taskID: %d", req.TaskID)
		resputil.WrapSuccessResponse(c, resp)
		return
	}

	// get log
	pods, err := mgr.logClient.GetPodsWithLabel(taskModel.Namespace, taskModel.JobName)
	if err != nil {
		msg := fmt.Sprintf("get task log failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50005)
		return
	}
	var token string
	re := regexp.MustCompile(`\?token=([a-zA-Z0-9]+)`)
	for _, pod := range pods {
		podLog, err := mgr.logClient.GetPodLogs(pod)
		if err != nil {
			msg := fmt.Sprintf("get task log failed, err %v", err)
			log.Error(msg)
			resputil.WrapFailedResponse(c, msg, 50005)
			return
		}
		matches := re.FindStringSubmatch(podLog)
		if len(matches) >= 2 {
			token = matches[1]
			break
		}
	}

	// Get service port
	port, err := mgr.logClient.GetSvcPort(taskModel.Namespace, taskModel.JobName)
	if err != nil {
		msg := fmt.Sprintf("get task svc failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50005)
		return
	}

	// Save token to db
	err = mgr.taskService.UpdateToken(taskModel.ID, token)
	if err != nil {
		msg := fmt.Sprintf("update task token failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50005)
		return
	}
	err = mgr.taskService.UpdateNodePort(taskModel.ID, port)
	if err != nil {
		msg := fmt.Sprintf("update task node port failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50005)
		return
	}

	resp := payload.GetJupyterResp{
		Port:  port,
		Token: token,
	}
	log.Infof("get task token success, taskID: %d", req.TaskID)
	resputil.WrapSuccessResponse(c, resp)
}

func (mgr *JupyterMgr) Delete(c *gin.Context) {
	log.Infof("Task Delete, url: %s", c.Request.URL)
	var req payload.DeleteTaskReq
	var err error
	if err = c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50006)
		return
	}
	username, _ := c.Get("username")
	// check if task.username is same as username
	_, err = mgr.taskService.GetByUserAndID(username.(string), req.TaskID)
	if err != nil {
		msg := fmt.Sprintf("get task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50007)
		return
	}
	mgr.NotifyTaskUpdate(req.TaskID, username.(string), util.DeleteTask)
	if req.ForceDelete {
		err = mgr.taskService.ForceDeleteByUserAndID(username.(string), req.TaskID)
	} else {
		err = mgr.taskService.DeleteByUserAndID(username.(string), req.TaskID)
	}
	if err != nil {
		msg := fmt.Sprintf("delete task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg, 50007)
		return
	}

	log.Infof("delete task success, taskID: %d", req.TaskID)
	resputil.WrapSuccessResponse(c, "")
}

func (mgr *JupyterMgr) GetImages(c *gin.Context) {
	// 现阶段先返回一个固定的镜像列表
	images := []string{
		"jupyter/base-notebook:ubuntu-22.04",
	}
	resp := payload.GetImagesResp{
		Images: images,
	}
	resputil.WrapSuccessResponse(c, resp)
}
