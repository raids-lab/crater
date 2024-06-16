package aijob

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/handler/vcjob"
	"github.com/raids-lab/crater/internal/resputil"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/crclient"
	tasksvc "github.com/raids-lab/crater/pkg/db/task"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	payload "github.com/raids-lab/crater/pkg/server/payload"
	"github.com/raids-lab/crater/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AIJobMgr struct {
	taskService    tasksvc.DBService
	pvcClient      *crclient.PVCClient
	logClient      *crclient.LogClient
	taskController *aitaskctl.TaskController
}

func NewAITaskMgr(taskController *aitaskctl.TaskController, pvcClient *crclient.PVCClient, logClient *crclient.LogClient) handler.Manager {
	return &AIJobMgr{
		taskService:    tasksvc.NewDBService(),
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

	g.POST("training", mgr.Create)
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
	var vcReq vcjob.CreateCustomReq
	if err := c.ShouldBindJSON(&vcReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var req payload.CreateTaskReq

	token := interutil.GetToken(c)
	req.TaskName = vcReq.Name
	req.UserName = token.QueueName
	req.SLO = 0
	req.TaskType = "training"
	req.Image = vcReq.Image
	req.ResourceRequest = vcReq.Resource
	req.Command = vcReq.Command
	req.WorkingDir = vcReq.WorkingDir

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
	token := interutil.GetToken(c)
	taskModels, err := mgr.taskService.ListByQueue(token.QueueName)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list task failed, err %v", err), resputil.NotSpecified)
		return
	}

	var jobs []vcjob.JobResp
	for i := range taskModels {
		taskModel := &taskModels[i]

		var runningTimestamp metav1.Time
		if taskModel.StartedAt != nil {
			runningTimestamp = metav1.NewTime(*taskModel.StartedAt)
		}

		var completedTimestamp metav1.Time
		if taskModel.FinishAt != nil {
			completedTimestamp = metav1.NewTime(*taskModel.FinishAt)
		}

		resources, _ := models.JSONToResourceList(taskModel.ResourceRequest)

		job := vcjob.JobResp{
			Name:               taskModel.TaskName,
			JobName:            taskModel.JobName,
			Owner:              token.Username,
			JobType:            taskModel.TaskType,
			Queue:              taskModel.UserName,
			Status:             taskModel.Status,
			CreationTimestamp:  metav1.NewTime(taskModel.CreatedAt),
			RunningTimestamp:   runningTimestamp,
			CompletedTimestamp: completedTimestamp,
			Nodes:              []string{},
			Resources:          resources,
		}
		jobs = append(jobs, job)
	}

	resputil.Success(c, jobs)
}

func (mgr *AIJobMgr) Get(c *gin.Context) {
	logutils.Log.Infof("Task Get, url: %s", c.Request.URL)
	var req payload.GetTaskReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate get parameters failed, err %v", err), resputil.NotSpecified)
		return
	}
	token := interutil.GetToken(c)
	taskModel, err := mgr.taskService.GetByUserAndID(token.Username, req.TaskID)
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
	token := interutil.GetToken(c)
	taskModel, err := mgr.taskService.GetByUserAndID(token.Username, req.TaskID)
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
	token := interutil.GetToken(c)
	// check if user is authorized to delete the task
	_, err = mgr.taskService.GetByUserAndID(token.Username, req.TaskID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}
	mgr.NotifyTaskUpdate(req.TaskID, token.Username, util.DeleteTask)
	if req.ForceDelete {
		err = mgr.taskService.ForceDeleteByUserAndID(token.Username, req.TaskID)
	} else {
		err = mgr.taskService.DeleteByUserAndID(token.Username, req.TaskID)
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
	token := interutil.GetToken(c)
	task, err := mgr.taskService.GetByUserAndID(token.Username, req.TaskID)
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
	mgr.NotifyTaskUpdate(req.TaskID, token.Username, util.UpdateTask)
	logutils.Log.Infof("update task success, taskID: %d", req.TaskID)
	resputil.Success(c, "")
}

func (mgr *AIJobMgr) GetQuota(c *gin.Context) {
	token := interutil.GetToken(c)
	quotaInfo := mgr.taskController.GetQuotaInfoSnapshotByUsername(token.Username)
	if quotaInfo == nil {
		resputil.Error(c, fmt.Sprintf("get user:%v quota failed", token.Username), resputil.NotSpecified)
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
	token := interutil.GetToken(c)
	taskCountList, err := mgr.taskService.GetUserTaskStatusCount(token.Username)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task count statistic failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.AITaskStatistic{
		TaskCount: taskCountList,
	}
	resputil.Success(c, resp)
}
