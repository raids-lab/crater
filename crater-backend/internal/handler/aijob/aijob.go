package aijob

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"gopkg.in/yaml.v2"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/handler/vcjob"
	interpayload "github.com/raids-lab/crater/internal/payload"
	"github.com/raids-lab/crater/internal/resputil"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	aijobapi "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	tasksvc "github.com/raids-lab/crater/pkg/db/task"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	payload "github.com/raids-lab/crater/pkg/server/payload"
	"github.com/raids-lab/crater/pkg/util"
)

type AIJobMgr struct {
	name           string
	client         client.Client
	kubeClient     kubernetes.Interface
	taskService    tasksvc.DBService
	logClient      *crclient.LogClient
	taskController aitaskctl.TaskControllerInterface
}

func NewAITaskMgr(taskController aitaskctl.TaskControllerInterface, cl client.Client,
	kc kubernetes.Interface, logClient *crclient.LogClient) handler.Manager {
	return &AIJobMgr{
		name:           "aijobs",
		client:         cl,
		kubeClient:     kc,
		taskService:    tasksvc.NewDBService(),
		logClient:      logClient,
		taskController: taskController,
	}
}

func (mgr *AIJobMgr) GetName() string { return mgr.name }

func (mgr *AIJobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *AIJobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET(":id/token", mgr.GetJupyterToken)
	g.GET("", mgr.List)
	g.GET("all", mgr.ListAll)
	g.GET("quota", mgr.GetQuota)
	g.DELETE(":id", mgr.Delete)

	g.GET(":id/log", mgr.GetLogs)
	g.GET(":id/yaml", mgr.GetJobYaml)
	g.GET(":id/detail", mgr.Get)

	g.POST("training", mgr.Create)
	g.POST("jupyter", mgr.CreateJupyterJob)
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

// GetQuota godoc
// @Summary Get the quota of the queue
// @Description Get the quota of the queue by client-go
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "Quota Information"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/aijobs/quota [get]
//
//nolint:gocyclo // TODO: refactor
func (mgr *AIJobMgr) GetQuota(c *gin.Context) {
	token := interutil.GetToken(c)

	q := query.Queue
	queue, err := q.WithContext(c).Where(q.Name.Eq(token.QueueName)).First()
	if err != nil {
		resputil.Error(c, "Queue not found", resputil.TokenInvalid)
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
		if name == v1.ResourceCPU || name == v1.ResourceMemory || strings.Contains(string(name), "/") {
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
		if strings.Contains(string(name), "/") {
			// convert nvidia.com/v100 to v100
			split := strings.Split(string(name), "/")
			if len(split) == 2 {
				resourceType := split[1]
				label := resourceType
				resource.Label = label
			}
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
	CreateJupyterReq struct {
		vcjob.CreateJobCommon `json:",inline"`
		Name                  string          `json:"name" binding:"required"`
		Resource              v1.ResourceList `json:"resource"`
		Image                 string          `json:"image" binding:"required"`
	}
)

func (mgr *AIJobMgr) CreateJupyterJob(c *gin.Context) {
	token := interutil.GetToken(c)
	var vcReq CreateJupyterReq
	if err := c.ShouldBindJSON(&vcReq); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var req payload.CreateTaskReq

	req.TaskName = vcReq.Name
	req.Namespace = config.GetConfig().Workspace.Namespace
	req.UserName = token.QueueName
	req.SLO = 1
	req.TaskType = "jupyter"
	req.Image = vcReq.Image
	req.ResourceRequest = vcReq.Resource

	taskModel := models.FormatTaskAttrToModel(&req.TaskAttr)

	// Command to start Jupyter
	commandSchema := "start.sh jupyter lab --allow-root --notebook-dir=/home/%s"
	command := fmt.Sprintf(commandSchema, token.Username)

	// 1. Volume Mounts
	volumes, volumeMounts, err := vcjob.GenerateVolumeMounts(c, token.UserID, vcReq.VolumeMounts)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 1.1 Support NGC images
	if !strings.Contains(req.Image, "jupyter") {
		volumes = append(volumes, v1.Volume{
			Name: "bash-script-volume",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: "jupyter-start-configmap",
					},
					//nolint:mnd // 0755 is the default mode
					DefaultMode: lo.ToPtr(int32(0755)),
				},
			},
		})
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      "bash-script-volume",
			MountPath: "/usr/bin/start.sh",
			ReadOnly:  false,
			SubPath:   "start.sh",
		})

		commandSchema := "/usr/bin/start.sh jupyter lab --allow-root --notebook-dir=/home/%s"
		command = fmt.Sprintf(commandSchema, token.Username)
	}

	// 2. Env Vars
	//nolint:mnd // 4 is the number of default envs
	envs := make([]v1.EnvVar, len(vcReq.Envs)+4)
	envs[0] = v1.EnvVar{Name: "GRANT_SUDO", Value: "1"}
	envs[1] = v1.EnvVar{Name: "CHOWN_HOME", Value: "1"}
	envs[2] = v1.EnvVar{Name: "NB_UID", Value: "1001"}
	envs[3] = v1.EnvVar{Name: "NB_USER", Value: token.Username}
	for i, env := range vcReq.Envs {
		envs[i+4] = env
	}

	// 3. TODO: Node Affinity for ARM64 Nodes
	affinity := vcjob.GenerateNodeAffinity(vcReq.Selectors)

	// 5. Create the pod spec
	podSpec := v1.PodSpec{
		Affinity: affinity,
		Volumes:  volumes,
		Containers: []v1.Container{
			{
				Name:    "jupyter-notebook",
				Image:   req.Image,
				Command: []string{"bash", "-c", command},
				Resources: v1.ResourceRequirements{
					Limits:   vcReq.Resource,
					Requests: vcReq.Resource,
				},
				WorkingDir: fmt.Sprintf("/home/%s", token.Username),

				Env: envs,
				Ports: []v1.ContainerPort{
					{ContainerPort: vcjob.JupyterPort, Name: "notebook-port", Protocol: v1.ProtocolTCP},
				},
				SecurityContext: &v1.SecurityContext{
					AllowPrivilegeEscalation: lo.ToPtr(true),
					RunAsUser:                lo.ToPtr(int64(0)),
					RunAsGroup:               lo.ToPtr(int64(0)),
				},
				TerminationMessagePath:   "/dev/termination-log",
				TerminationMessagePolicy: v1.TerminationMessageReadFile,
				VolumeMounts:             volumeMounts,
			},
		},
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

type (
	CreateAIJobReq struct {
		vcjob.CreateCustomReq `json:",inline"`
		SLO                   uint `json:"slo"`
	}
)

// Create godoc
// @Summary Create a new AI job
// @Description Create a new AI job by client-go
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param job body any true "Create AI Job Request"
// @Success 200 {object} resputil.Response[any] "Create AI Job Response"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/aijobs/training [post]
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
	req.SLO = vcReq.SLO
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

// List godoc
// @Summary List AI jobs
// @Description List AI jobs by client-go
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "AI Job List"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/aijobs [get]
func (mgr *AIJobMgr) List(c *gin.Context) {
	token := interutil.GetToken(c)
	taskModels, err := mgr.taskService.ListByQueue(token.QueueName)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list task failed, err %v", err), resputil.NotSpecified)
		return
	}

	jobs := make([]AIJobResp, len(taskModels))
	for i := range taskModels {
		jobs[i] = convertToAIJobResp(&taskModels[i])
	}

	resputil.Success(c, jobs)
}

func (mgr *AIJobMgr) ListAll(c *gin.Context) {
	taskModels, err := mgr.taskService.ListAll()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list task failed, err %v", err), resputil.NotSpecified)
		return
	}

	jobs := make([]AIJobResp, len(taskModels))
	for i := range taskModels {
		jobs[i] = convertToAIJobResp(&taskModels[i])
	}

	resputil.Success(c, jobs)
}

func convertToAIJobResp(aiTask *models.AITask) AIJobResp {
	var runningTimestamp metav1.Time
	if aiTask.StartedAt != nil {
		runningTimestamp = metav1.NewTime(*aiTask.StartedAt)
	}

	var completedTimestamp metav1.Time
	if aiTask.FinishAt != nil {
		completedTimestamp = metav1.NewTime(*aiTask.FinishAt)
	}

	var priority string
	if aiTask.SLO > 0 {
		priority = "high"
	} else {
		priority = "low"
	}

	resources, _ := models.JSONToResourceList(aiTask.ResourceRequest)

	return AIJobResp{
		ID:            aiTask.ID,
		Priority:      priority,
		ProfileStatus: strconv.FormatUint(uint64(aiTask.ProfileStatus), 10),
		JobResp: vcjob.JobResp{
			Name:               aiTask.TaskName,
			JobName:            aiTask.JobName,
			Owner:              aiTask.Owner,
			JobType:            aiTask.TaskType,
			Queue:              aiTask.UserName,
			Status:             aiTask.Status,
			CreationTimestamp:  metav1.NewTime(aiTask.CreatedAt),
			RunningTimestamp:   runningTimestamp,
			CompletedTimestamp: completedTimestamp,
			Nodes:              []string{aiTask.Node},
			Resources:          resources,
		},
	}
}

type AIJobDetailReq struct {
	JobID uint `uri:"id" binding:"required"`
}

type AIJobDetailResp struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Username          string            `json:"username"`
	JobName           string            `json:"jobName"`
	Retry             string            `json:"retry"`
	Queue             string            `json:"queue"`
	Status            batch.JobPhase    `json:"status"`
	CreationTimestamp metav1.Time       `json:"createdAt"`
	RunningTimestamp  metav1.Time       `json:"startedAt"`
	Duration          string            `json:"runtime"`
	PodDetails        []vcjob.PodDetail `json:"podDetails"`
	UseTensorBoard    bool              `json:"useTensorBoard"`
	ID                uint              `json:"id"`
	Priority          string            `json:"priority"`
	ProfileStat       string            `json:"profileStat"`
	ProfileStatus     string            `json:"profileStatus"`
}

// Get godoc
// @Summary Get AI job details
// @Description Get AI job details by client-go
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path uint true "Job ID"
// @Success 200 {object} resputil.Response[any] "AI Job Details"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/aijobs/{id}/detail [get]
func (mgr *AIJobMgr) Get(c *gin.Context) {
	var req AIJobDetailReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	token := interutil.GetToken(c)
	taskModel, err := mgr.taskService.GetByQueueAndID(token.QueueName, req.JobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}

	var runningTimestamp metav1.Time
	var duration string
	if taskModel.StartedAt != nil {
		runningTimestamp = metav1.NewTime(*taskModel.StartedAt)
		duration = time.Since(runningTimestamp.Time).Truncate(time.Second).String()
	} else {
		runningTimestamp = metav1.Time{}
		duration = "0s"
	}

	var priority string
	if taskModel.SLO > 0 {
		priority = "high"
	} else {
		priority = "low"
	}

	var podDetail []vcjob.PodDetail
	podName := fmt.Sprintf("%s-0", taskModel.JobName)
	pod, err := mgr.kubeClient.CoreV1().Pods(taskModel.Namespace).Get(c, podName, metav1.GetOptions{})
	if err == nil {
		podDetail = []vcjob.PodDetail{{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			NodeName:  lo.ToPtr(pod.Spec.NodeName),
			IP:        pod.Status.PodIP,
			Resource:  pod.Spec.Containers[0].Resources.Requests,
			Phase:     pod.Status.Phase,
		}}
	}

	resp := AIJobDetailResp{
		Name:              taskModel.TaskName,
		Namespace:         taskModel.Namespace,
		Username:          taskModel.Owner,
		JobName:           taskModel.JobName,
		Retry:             fmt.Sprintf("%d", 0),
		Queue:             taskModel.UserName,
		Status:            convertJobPhase(taskModel.Status),
		CreationTimestamp: metav1.NewTime(taskModel.CreatedAt),
		RunningTimestamp:  runningTimestamp,
		Duration:          duration,
		PodDetails:        podDetail,
		UseTensorBoard:    false,
		ID:                taskModel.ID,
		Priority:          priority,
		ProfileStat:       taskModel.ProfileStat,
		ProfileStatus:     strconv.FormatUint(uint64(taskModel.ProfileStatus), 10),
	}
	logutils.Log.Infof("get task success, taskID: %d", req.JobID)
	resputil.Success(c, resp)
}

type AIJobLogReq struct {
	JobID uint `uri:"id" binding:"required"`
}

type AIJobLogResp struct {
	Logs map[string]string `json:"logs"`
}

// GetLogs godoc
// @Summary Get AI job logs
// @Description Get AI job logs by client-go
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path uint true "Job ID"
// @Success 200 {object} resputil.Response[any] "AI Job Logs"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/aijobs/{id}/log [get]
func (mgr *AIJobMgr) GetLogs(c *gin.Context) {
	var req AIJobLogReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	token := interutil.GetToken(c)
	taskModel, err := mgr.taskService.GetByQueueAndID(token.QueueName, req.JobID)
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
	logs := make(map[string]string, len(pods))
	for i := range pods {
		pod := &pods[i]
		podLog, err := mgr.logClient.GetPodLogs(pod)
		if err != nil {
			resputil.Error(c, fmt.Sprintf("get task log failed, err %v", err), resputil.NotSpecified)
			return
		}
		logs[pod.Name] = podLog
	}
	resp := AIJobLogResp{
		Logs: logs,
	}
	logutils.Log.Infof("get task success, taskID: %d", req.JobID)
	resputil.Success(c, resp)
}

// Delete godoc
// @Summary Delete an AI job
// @Description Delete an AI job by client-go
// @Tags AIJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path uint true "Job ID"
// @Success 200 {object} resputil.Response[any] "Delete AI Job Response"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/aijobs/{id} [delete]
func (mgr *AIJobMgr) Delete(c *gin.Context) {
	var req AIJobDetailReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := interutil.GetToken(c)

	// check if user is authorized to delete the task
	_, err := mgr.taskService.GetByQueueAndID(token.QueueName, req.JobID)
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
	task, err := mgr.taskService.GetByQueueAndID(token.QueueName, req.TaskID)
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

func convertJobPhase(aijobStatus string) batch.JobPhase {
	switch aijobStatus {
	case "Pending":
		return batch.Pending
	case "Running":
		return batch.Running
	case "Succeeded":
		return batch.Completed
	case "Failed":
		return batch.Failed
	case "Preempted":
		return batch.Aborted
	default:
		return batch.Pending
	}
}

// GetJobYaml godoc
// @Summary 获取vcjob Yaml详情
// @Description 调用k8s get crd
// @Tags vcjob-jupyter
// @Accept json
// @Produce json
// @Security Bearer
// @Param jobname query string true "vcjob-name"
// @Success 200 {object} resputil.Response[any] "任务yaml"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/aijobs/{id}/yaml [get]
func (mgr *AIJobMgr) GetJobYaml(c *gin.Context) {
	var req AIJobDetailReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := interutil.GetToken(c)
	taskModel, err := mgr.taskService.GetByQueueAndID(token.QueueName, req.JobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}

	if taskModel.UserName != token.QueueName {
		resputil.Error(c, "Job not found", resputil.NotSpecified)
		return
	}

	job := &aijobapi.AIJob{}
	namespace := config.GetConfig().Workspace.Namespace
	if err = mgr.client.Get(c, client.ObjectKey{Name: taskModel.JobName,
		Namespace: namespace}, job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// prune useless field
	job.ObjectMeta.ManagedFields = nil

	// utilize json omitempty tag to further prune
	jsonData, err := json.Marshal(job)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	var prunedJob map[string]any
	err = json.Unmarshal(jsonData, &prunedJob)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	JobYaml, err := yaml.Marshal(prunedJob)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, string(JobYaml))
}

func (mgr *AIJobMgr) GetJupyterToken(c *gin.Context) {
	var req AIJobDetailReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := interutil.GetToken(c)
	taskModel, err := mgr.taskService.GetByQueueAndID(token.QueueName, req.JobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}

	if taskModel.UserName != token.QueueName {
		resputil.Error(c, "Job not found", resputil.NotSpecified)
		return
	}

	svc := &v1.Service{}
	namespace := config.GetConfig().Workspace.Namespace
	if err = mgr.client.Get(c, client.ObjectKey{Name: "svc-" + taskModel.JobName, Namespace: namespace}, svc); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if svc.Spec.Type != v1.ServiceTypeNodePort {
		resputil.Error(c, "Service type is not NodePort", resputil.NotSpecified)
		return
	}

	if len(svc.Spec.Ports) == 0 {
		resputil.Error(c, "Service port not found", resputil.NotSpecified)
		return
	}

	baseURL := fmt.Sprintf("http://8.141.83.224:%d", int(svc.Spec.Ports[0].NodePort))

	// Get the logs of the job pod
	var jupyterToken string

	podName := fmt.Sprintf("%s-0", taskModel.JobName)
	buf, err := mgr.getPodLog(c, namespace, podName)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	re := regexp.MustCompile(`\?token=([a-zA-Z0-9]+)`)
	matches := re.FindStringSubmatch(buf.String())
	if len(matches) >= 2 {
		jupyterToken = matches[1]
	} else {
		resputil.Error(c, "Jupyter token not found", resputil.NotSpecified)
		return
	}

	if jupyterToken == "" {
		resputil.Error(c, "Jupyter token not found", resputil.NotSpecified)
		return
	}

	resputil.Success(c, vcjob.JobTokenResp{BaseURL: baseURL, Token: jupyterToken})
}

func (mgr *AIJobMgr) getPodLog(c *gin.Context, namespace, podName string) (*bytes.Buffer, error) {
	logOptions := &v1.PodLogOptions{}
	logReq := mgr.kubeClient.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	logs, err := logReq.Stream(c)
	if err != nil {
		return nil, err
	}
	defer logs.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(logs)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
