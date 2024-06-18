package aijob

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/handler/vcjob"
	interpayload "github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/resputil"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	tasksvc "github.com/raids-lab/crater/pkg/db/task"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	payload "github.com/raids-lab/crater/pkg/server/payload"
	"github.com/raids-lab/crater/pkg/util"
	"github.com/samber/lo"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
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
	g.GET("quota", mgr.GetQuota)
	g.DELETE(":id", mgr.Delete)

	g.GET(":id/log", mgr.GetLogs)
	g.GET(":id/detail", mgr.Get)

	g.POST("training", mgr.Create)
}

func (mgr *AIJobMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.List)
	g.GET(":id/detail", mgr.Get)
}

func (mgr *AIJobMgr) NotifyTaskUpdate(taskID uint, userName string, op util.TaskOperation) {
	mgr.taskController.TaskUpdated(util.TaskUpdateChan{
		TaskID:    taskID,
		UserName:  userName,
		Operation: op,
	})
}

//nolint:gocyclo // TODO: refactor
func (mgr *AIJobMgr) GetQuota(c *gin.Context) {
	token := interutil.GetToken(c)
	if token.QueueName == interutil.QueueNameNull {
		resputil.Error(c, "Queue not specified", resputil.QueueNotFound)
		return
	}

	q := query.Queue
	queue, err := q.WithContext(c).Where(q.Name.Eq(token.QueueName)).First()
	if err != nil {
		resputil.Error(c, "Queue not found", resputil.QueueNotFound)
		return
	}

	quota := queue.Quota.Data()

	guarantee := quota.Guaranteed
	deserved := quota.Deserved
	capability := quota.Capability

	// resources is a map, key is the resource name, value is the resource amount
	resources := make(map[v1.ResourceName]interpayload.ResourceResp)

	usedQuota := mgr.taskController.GetQuotaInfoSnapshotByUsername(token.QueueName)

	for name, quantity := range deserved {
		if name == v1.ResourceCPU || name == v1.ResourceMemory || strings.HasPrefix(string(name), "nvidia.com/") {
			resources[name] = interpayload.ResourceResp{
				Label: string(name),
				Deserved: lo.ToPtr(interpayload.ResourceBase{
					Amount: quantity.Value(),
					Format: string(quantity.Format),
				}),
			}
		}
	}

	for name, quantity := range usedQuota.HardUsed {
		if v, ok := resources[name]; ok {
			v.Allocated = lo.ToPtr(interpayload.ResourceBase{
				Amount: quantity.Value(),
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}
	for name, quantity := range usedQuota.SoftUsed {
		if v, ok := resources[name]; ok {
			var amount int64
			if hard := v.Allocated; hard != nil {
				amount = hard.Amount + quantity.Value()
			} else {
				amount = quantity.Value()
			}
			v.Allocated = lo.ToPtr(interpayload.ResourceBase{
				Amount: amount,
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}
	for name, quantity := range guarantee {
		if v, ok := resources[name]; ok {
			v.Guarantee = lo.ToPtr(interpayload.ResourceBase{
				Amount: quantity.Value(),
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}

	for name, quantity := range capability {
		if v, ok := resources[name]; ok {
			v.Capability = lo.ToPtr(interpayload.ResourceBase{
				Amount: quantity.Value(),
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}

	// if capability is not set, read max from db
	r := query.Resource
	for name, resource := range resources {
		if resource.Capability == nil {
			resouece, err := r.WithContext(c).Where(r.ResourceName.Eq(string(name))).First()
			if err != nil {
				continue
			}
			resource.Capability = &interpayload.ResourceBase{
				Amount: resouece.Amount,
				Format: resouece.Format,
			}
			resources[name] = resource
		}
	}

	// map contains cpu, memory, gpus, get them from the map
	cpu := resources[v1.ResourceCPU]
	cpu.Label = "cpu"
	memory := resources[v1.ResourceMemory]
	memory.Label = "mem"
	var gpus []interpayload.ResourceResp
	for name, resource := range resources {
		if strings.HasPrefix(string(name), "nvidia.com/") {
			// convert nvidia.com/v100 to v100
			resource.Label = strings.TrimPrefix(string(name), "nvidia.com/")
			gpus = append(gpus, resource)
		}
	}
	sort.Slice(gpus, func(i, j int) bool {
		return gpus[i].Label < gpus[j].Label
	})

	resputil.Success(c, interpayload.QuotaResp{
		CPU:    cpu,
		Memory: memory,
		GPUs:   gpus,
	})
}

type (
	CreateAIJobReq struct {
		vcjob.CreateCustomReq `json:",inline"`
		SLO                   uint `json:"slo"`
	}
)

func (mgr *AIJobMgr) Create(c *gin.Context) {
	var vcReq CreateAIJobReq
	if err := c.ShouldBindJSON(&vcReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var req payload.CreateTaskReq

	token := interutil.GetToken(c)
	req.TaskName = vcReq.Name
	req.Namespace = config.GetConfig().Workspace.Namespace
	req.UserName = token.QueueName
	// vcReq.SLO
	req.SLO = 0
	req.TaskType = "training"
	req.Image = vcReq.Image
	req.ResourceRequest = vcReq.Resource
	req.Command = vcReq.Command
	req.WorkingDir = vcReq.WorkingDir

	taskModel := models.FormatTaskAttrToModel(&req.TaskAttr)
	podSpec, err := vcjob.GenerateCustomPodSpec(c, token.UserID, &vcReq.CreateCustomReq)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("generate pod spec failed, err %v", err), resputil.NotSpecified)
		return
	}

	taskModel.PodTemplate = datatypes.NewJSONType(podSpec)
	taskModel.Owner = token.Username
	err = mgr.taskService.Create(taskModel)
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

type AIJobResp struct {
	vcjob.JobResp `json:",inline"`
	ID            uint   `json:"id"`
	Priority      string `json:"priority"`
	ProfileStatus string `json:"profileStatus"`
}

func (mgr *AIJobMgr) List(c *gin.Context) {
	token := interutil.GetToken(c)
	taskModels, err := mgr.taskService.ListByQueue(token.QueueName)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list task failed, err %v", err), resputil.NotSpecified)
		return
	}

	jobs := make([]AIJobResp, len(taskModels))
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

		var priority string
		if taskModel.SLO > 0 {
			priority = "high"
		} else {
			priority = "low"
		}

		resources, _ := models.JSONToResourceList(taskModel.ResourceRequest)

		job := AIJobResp{
			ID:            taskModel.ID,
			Priority:      priority,
			ProfileStatus: strconv.FormatUint(uint64(taskModel.ProfileStatus), 10),
			JobResp: vcjob.JobResp{ // 显式初始化嵌入的结构体
				Name:               taskModel.TaskName,
				JobName:            taskModel.TaskName,
				Owner:              taskModel.Owner,
				JobType:            taskModel.TaskType,
				Queue:              taskModel.UserName,
				Status:             taskModel.Status,
				CreationTimestamp:  metav1.NewTime(taskModel.CreatedAt),
				RunningTimestamp:   runningTimestamp,
				CompletedTimestamp: completedTimestamp,
				Nodes:              []string{taskModel.Node},
				Resources:          resources,
			},
		}
		jobs[i] = job
	}

	resputil.Success(c, jobs)
}

type AIJobDetailReq struct {
	JobID uint `uri:"id" binding:"required"`
}

func (mgr *AIJobMgr) Get(c *gin.Context) {
	var req AIJobDetailReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	token := interutil.GetToken(c)
	taskModel, err := mgr.taskService.GetByUserAndID(token.Username, req.JobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}
	resp := payload.GetTaskResp{
		AITask: *taskModel,
	}
	logutils.Log.Infof("get task success, taskID: %d", req.JobID)
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
	var req AIJobDetailReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := interutil.GetToken(c)

	// check if user is authorized to delete the task
	_, err := mgr.taskService.GetByUserAndID(token.QueueName, req.JobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}
	mgr.NotifyTaskUpdate(req.JobID, token.QueueName, util.DeleteTask)

	err = mgr.taskService.DeleteByUserAndID(token.QueueName, req.JobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("delete task failed, err %v", err), resputil.NotSpecified)
		return
	}

	logutils.Log.Infof("delete task success, taskID: %d", req.JobID)
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
