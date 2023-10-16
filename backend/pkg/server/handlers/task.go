package handlers

import (
	"fmt"

	"github.com/aisystem/ai-protal/pkg/constants"
	tasksvc "github.com/aisystem/ai-protal/pkg/db/task"
	"github.com/aisystem/ai-protal/pkg/models"
	payload "github.com/aisystem/ai-protal/pkg/server/payload"
	resputil "github.com/aisystem/ai-protal/pkg/server/response"
	"github.com/aisystem/ai-protal/pkg/util"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (mgr *TaskMgr) RegisterRoute(r *gin.Engine) {
	g := r.Group(constants.APIPrefix + "/task")
	g.POST("create", mgr.Create)
	g.POST("delete", mgr.Delete)
	g.POST("updateSLO", mgr.UpdateSLO)
	g.GET("list", mgr.List)
	g.GET("get", mgr.Get)

}

type TaskMgr struct {
	taskService    tasksvc.DBService
	taskUpdateChan <-chan util.TaskUpdateChan
}

func NewTaskMgr(taskUpdateChan <-chan util.TaskUpdateChan) *TaskMgr {
	return &TaskMgr{
		taskService:    tasksvc.NewDBService(),
		taskUpdateChan: taskUpdateChan,
	}
}

func (mgr *TaskMgr) Create(c *gin.Context) {
	log.Infof("Quota Create, url: %s", c.Request.URL)
	var req payload.CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate create parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg)
		return
	}
	taskModel := models.FormatTaskAttrToModel(&req.TaskAttr)
	err := mgr.taskService.Create(taskModel)
	if err != nil {
		msg := fmt.Sprintf("create task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg)
		return
	}

	log.Infof("create task success, taskID: %d", req.ID)
	resputil.WrapSuccessResponse(c, "")
}

func (mgr *TaskMgr) List(c *gin.Context) {
	log.Infof("Quota List, url: %s", c.Request.URL)
	var req payload.ListTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate list parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg)
		return
	}
	taskModels, err := mgr.taskService.ListByUserAndStatus(req.UserName, req.Status)
	if err != nil {
		msg := fmt.Sprintf("list task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg)
		return
	}
	resp := payload.ListTaskResp{
		Tasks: make([]models.TaskAttr, 0),
	}
	for _, taskModel := range taskModels {
		resp.Tasks = append(resp.Tasks, *models.FormatTaskModelToAttr(&taskModel))
	}
	log.Infof("list task success, taskNum: %d", len(resp.Tasks))
	resputil.WrapSuccessResponse(c, resp)
}

func (mgr *TaskMgr) Get(c *gin.Context) {
	log.Infof("Quota Get, url: %s", c.Request.URL)
	var req payload.GetTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate get parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg)
		return
	}
	taskModel, err := mgr.taskService.GetByID(req.TaskID)
	if err != nil {
		msg := fmt.Sprintf("get task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg)
		return
	}
	resp := payload.GetTaskResp{
		*models.FormatTaskModelToAttr(taskModel),
	}
	log.Infof("get task success, taskID: %d", req.TaskID)
	resputil.WrapSuccessResponse(c, resp)
}

func (mgr *TaskMgr) Delete(c *gin.Context) {
	log.Infof("Quota Delete, url: %s", c.Request.URL)
	var req payload.DeleteTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		msg := fmt.Sprintf("validate delete parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg)
		return
	}
	err := mgr.taskService.DeleteByID(req.TaskID)
	if err != nil {
		msg := fmt.Sprintf("delete task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg)
		return
	}
	log.Infof("delete task success, taskID: %d", req.TaskID)
	resputil.WrapSuccessResponse(c, "")
}

func (mgr *TaskMgr) UpdateSLO(c *gin.Context) {
	log.Infof("Quota Update, url: %s", c.Request.URL)
	var req payload.UpdateTaskSLOReq
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := fmt.Sprintf("validate update parameters failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg)
		return
	}
	task, err := mgr.taskService.GetByID(req.TaskID)
	if err != nil {
		msg := fmt.Sprintf("get task failed, err %v", err)
		log.Error(msg)
		resputil.WrapFailedResponse(c, msg)
		return
	}
	task.SLO = req.SLO
	err = mgr.taskService.Update(task)
	log.Infof("update task success, taskID: %d", req.TaskID)
	resputil.WrapSuccessResponse(c, "")
}
