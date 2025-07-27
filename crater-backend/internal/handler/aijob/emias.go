package aijob

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
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
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/util"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	if config.GetConfig().SchedulerPlugins.Aijob.AijobEn {
		handler.Registers = append(handler.Registers, NewAITaskMgr)
	}
}

type AIJobMgr struct {
	name           string
	client         client.Client
	kubeClient     kubernetes.Interface
	taskService    aitaskctl.DBService
	taskController aitaskctl.TaskControllerInterface
}

func NewAITaskMgr(conf *handler.RegisterConfig) handler.Manager {
	return &AIJobMgr{
		name:           "aijobs",
		client:         conf.Client,
		kubeClient:     conf.KubeClient,
		taskService:    aitaskctl.NewDBService(),
		taskController: conf.AITaskCtrl,
	}
}

func (mgr *AIJobMgr) GetName() string { return mgr.name }

func (mgr *AIJobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *AIJobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET(":id/token", mgr.GetJupyterToken)
	g.GET("", mgr.ListUserJob)
	g.GET("all", mgr.ListAllJob)
	g.GET("quota", mgr.GetQuota)
	g.DELETE(":id", mgr.Delete)

	g.GET(":id/detail", mgr.GetDetail)
	g.GET(":id/yaml", mgr.GetJobYaml)
	g.GET(":id/pods", mgr.GetJobPods)
	g.GET(":id/event", mgr.GetJobEvents) // 添加获取事件的路由

	g.POST("training", mgr.CreateCustom)
	g.POST("jupyter", mgr.CreateJupyterJob)
}

func (mgr *AIJobMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.ListUserJob)
	g.GET(":id/detail", mgr.GetDetail)
}

func (mgr *AIJobMgr) NotifyTaskUpdate(taskID uint, userName string, op util.TaskOperation) {
	mgr.taskController.TaskUpdated(util.TaskUpdateChan{
		TaskID:    taskID,
		UserName:  userName,
		Operation: op,
	})
}

// GetQuota godoc
//
//	@Summary		Get the quota of the queue
//	@Description	Get the quota of the queue by client-go
//	@Tags			AIJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"Quota Information"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/aijobs/quota [get]
//
//nolint:gocyclo // TODO: refactor
func (mgr *AIJobMgr) GetQuota(c *gin.Context) {
	token := interutil.GetToken(c)

	q := query.Account
	queue, err := q.WithContext(c).Where(q.Name.Eq(token.AccountName)).First()
	if err != nil {
		resputil.Error(c, "Queue not found", resputil.NotSpecified)
		return
	}

	quota := queue.Quota.Data()

	guarantee := quota.Guaranteed
	deserved := quota.Deserved
	capability := quota.Capability

	// resources is a map, key is the resource name, value is the resource amount
	resources := make(map[v1.ResourceName]interpayload.ResourceResp)

	usedQuota := mgr.taskController.GetQuotaInfoSnapshotByUsername(token.AccountName)

	for name, quantity := range deserved {
		if name == v1.ResourceCPU || name == v1.ResourceMemory || strings.Contains(string(name), "/") {
			resources[name] = interpayload.ResourceResp{
				Label: string(name),
				Deserved: ptr.To(interpayload.ResourceBase{
					Amount: quantity.Value(),
					Format: string(quantity.Format),
				}),
			}
		}
	}

	for name, quantity := range usedQuota.HardUsed {
		if v, ok := resources[name]; ok {
			v.Allocated = ptr.To(interpayload.ResourceBase{
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
			v.Allocated = ptr.To(interpayload.ResourceBase{
				Amount: amount,
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}
	for name, quantity := range guarantee {
		if v, ok := resources[name]; ok {
			v.Guarantee = ptr.To(interpayload.ResourceBase{
				Amount: quantity.Value(),
				Format: string(quantity.Format),
			})
			resources[name] = v
		}
	}

	for name, quantity := range capability {
		if v, ok := resources[name]; ok {
			v.Capability = ptr.To(interpayload.ResourceBase{
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
		Resource              v1.ResourceList `json:"resource"`
		Image                 string          `json:"image" binding:"required"`
	}

	CreateTaskReq struct {
		model.TaskAttr
	}
)

// GetJobPods godoc
//
//	@Summary		获取任务的Pod列表
//	@Description	获取任务的Pod列表
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string					true	"Job Name"
//	@Success		200		{object}	resputil.Response[any]	"Pod列表"
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/{name}/pods [get]
func (mgr *AIJobMgr) GetJobPods(c *gin.Context) {
	var req AIJobDetailReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	token := interutil.GetToken(c)
	taskModel, err := mgr.taskService.GetByQueueAndID(token.AccountName, req.JobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}

	var podDetails []vcjob.PodDetail
	podName := fmt.Sprintf("%s-0", taskModel.JobName)
	pod, err := mgr.kubeClient.CoreV1().Pods(taskModel.Namespace).Get(c, podName, metav1.GetOptions{})
	if err == nil {
		podDetails = []vcjob.PodDetail{{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			NodeName:  ptr.To(pod.Spec.NodeName),
			IP:        pod.Status.PodIP,
			Resource:  pod.Spec.Containers[0].Resources.Requests,
			Phase:     pod.Status.Phase,
		}}
	}

	resputil.Success(c, podDetails)
}

type (
	CreateAIJobReq struct {
		vcjob.CreateCustomReq `json:",inline"`
		SLO                   uint `json:"slo"`
	}
)

type AIJobResp struct {
	vcjob.JobResp `json:",inline"`
	ID            uint   `json:"id"`
	Priority      string `json:"priority"`
	ProfileStatus string `json:"profileStatus"`
}

// ListUserJob godoc
//
//	@Summary		ListUserJob AI jobs
//	@Description	ListUserJob AI jobs by client-go
//	@Tags			AIJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"AI Job ListUserJob"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/aijobs [get]
func (mgr *AIJobMgr) ListUserJob(c *gin.Context) {
	token := interutil.GetToken(c)
	taskModels, err := mgr.taskService.ListByQueue(token.AccountName)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list task failed, err %v", err), resputil.NotSpecified)
		return
	}

	jobs := make([]AIJobResp, len(taskModels))
	for i := range taskModels {
		jobs[i], _ = convertToAIJobResp(c, taskModels[i])
	}

	resputil.Success(c, jobs)
}

func (mgr *AIJobMgr) ListAllJob(c *gin.Context) {
	taskModels, err := mgr.taskService.ListAll()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list task failed, err %v", err), resputil.NotSpecified)
		return
	}

	jobs := make([]AIJobResp, len(taskModels))
	for i := range taskModels {
		jobs[i], _ = convertToAIJobResp(c, taskModels[i])
	}

	resputil.Success(c, jobs)
}

func convertToAIJobResp(c context.Context, aiTask *model.AITask) (AIJobResp, error) {
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

	resources, _ := model.JSONToResourceList(aiTask.ResourceRequest)

	u := query.User
	user, err := u.WithContext(c).Where(u.Name.Eq(aiTask.Owner)).First()
	if err != nil {
		return AIJobResp{}, err
	}

	profileStatus := aiTask.ProfileStatus
	if aiTask.IsDeleted && profileStatus != model.EmiasProfileFinish {
		profileStatus = model.EmiasProfileSkipped
	}

	return AIJobResp{
		ID:            aiTask.ID,
		Priority:      priority,
		ProfileStatus: strconv.FormatUint(uint64(profileStatus), 10),
		JobResp: vcjob.JobResp{
			Name:    aiTask.TaskName,
			JobName: fmt.Sprintf("%d", aiTask.ID),
			Owner:   aiTask.Owner,
			UserInfo: model.UserInfo{
				Nickname: user.Nickname,
				Username: user.Name,
			},
			JobType:            aiTask.TaskType,
			Queue:              aiTask.UserName,
			Status:             string(convertJobPhase(aiTask)),
			CreationTimestamp:  metav1.NewTime(aiTask.CreatedAt),
			RunningTimestamp:   runningTimestamp,
			CompletedTimestamp: completedTimestamp,
			Nodes:              []string{aiTask.Node},
			Resources:          resources,
		},
	}, nil
}

type AIJobDetailReq struct {
	JobID uint `uri:"id" binding:"required"`
}

type AIJobDetailResp struct {
	vcjob.JobDetailResp
	Retry         string `json:"retry"`
	Duration      string `json:"runtime"`
	Priority      string `json:"priority"`
	ProfileStat   string `json:"profileStat"`
	ProfileStatus string `json:"profileStatus"`
}

// GetDetail godoc
//
//	@Summary		GetDetail AI job details
//	@Description	GetDetail AI job details by client-go
//	@Tags			AIJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		uint					true	"Job ID"
//	@Success		200	{object}	resputil.Response[any]	"AI Job Details"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/aijobs/{id}/detail [get]
func (mgr *AIJobMgr) GetDetail(c *gin.Context) {
	var req AIJobDetailReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	token := interutil.GetToken(c)
	taskModel, err := mgr.taskService.GetByQueueAndID(token.AccountName, req.JobID)
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

	u := query.User
	user, err := u.WithContext(c).Where(u.Name.Eq(taskModel.Owner)).First()
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get user failed, err %v", err), resputil.NotSpecified)
		return
	}

	var profileDataPtr *monitor.ProfileData
	// unmarshal profile data from taskModel.ProfileStat
	if taskModel.ProfileStat != "" {
		profileData := monitor.ProfileData{}
		_ = json.Unmarshal([]byte(taskModel.ProfileStat), &profileData)
		profileDataPtr = &profileData
	} else {
		profileDataPtr = nil
	}

	resp := AIJobDetailResp{
		JobDetailResp: vcjob.JobDetailResp{
			Name:      taskModel.TaskName,
			Namespace: taskModel.Namespace,
			Username:  taskModel.Owner,
			UserInfo: model.UserInfo{
				Nickname: user.Nickname,
				Username: user.Name,
			},
			JobName:           taskModel.JobName,
			Queue:             taskModel.UserName,
			Status:            convertJobPhase(taskModel),
			CreationTimestamp: metav1.NewTime(taskModel.CreatedAt),
			RunningTimestamp:  runningTimestamp,
			ProfileData:       profileDataPtr,
			ScheduleData: &model.ScheduleData{
				ImagePullTime: "0",
			},
		},
		Retry:         fmt.Sprintf("%d", 0),
		Duration:      duration,
		Priority:      priority,
		ProfileStat:   taskModel.ProfileStat,
		ProfileStatus: strconv.FormatUint(uint64(taskModel.ProfileStatus), 10),
	}
	if taskModel.IsDeleted {
		resp.CompletedTimestamp = metav1.NewTime(taskModel.UpdatedAt)
	}

	klog.Infof("get task success, taskID: %d", req.JobID)
	resputil.Success(c, resp)
}

type AIJobLogReq struct {
	JobID uint `uri:"id" binding:"required"`
}

type AIJobLogResp struct {
	Logs map[string]string `json:"logs"`
}

// Delete godoc
//
//	@Summary		Delete an AI job
//	@Description	Delete an AI job by client-go
//	@Tags			AIJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		uint					true	"Job ID"
//	@Success		200	{object}	resputil.Response[any]	"Delete AI Job Response"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/aijobs/{id} [delete]
func (mgr *AIJobMgr) Delete(c *gin.Context) {
	var req AIJobDetailReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := interutil.GetToken(c)

	// check if user is authorized to delete the task
	_, err := mgr.taskService.GetByQueueAndID(token.AccountName, req.JobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}
	mgr.NotifyTaskUpdate(req.JobID, token.AccountName, util.DeleteTask)

	err = mgr.taskService.DeleteByQueueAndID(token.AccountName, req.JobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("delete task failed, err %v", err), resputil.NotSpecified)
		return
	}

	klog.Infof("delete task success, taskID: %d", req.JobID)
	resputil.Success(c, "")
}

type UpdateTaskSLOReq struct {
	TaskID uint `json:"taskID" binding:"required"`
	SLO    uint `json:"slo"` // change the slo of the task
}

func (mgr *AIJobMgr) UpdateSLO(c *gin.Context) {
	klog.Infof("Task Update, url: %s", c.Request.URL)
	var req UpdateTaskSLOReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, fmt.Sprintf("validate update parameters failed, err %v", err), resputil.NotSpecified)
		return
	}
	token := interutil.GetToken(c)
	task, err := mgr.taskService.GetByQueueAndID(token.AccountName, req.TaskID)
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
	klog.Infof("update task success, taskID: %d", req.TaskID)
	resputil.Success(c, "")
}

// GetJobYaml godoc
//
//	@Summary		获取vcjob Yaml详情
//	@Description	调用k8s get crd
//	@Tags			vcjob-jupyter
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			jobname	query		string					true	"vcjob-name"
//	@Success		200		{object}	resputil.Response[any]	"任务yaml"
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/aijobs/{id}/yaml [get]
func (mgr *AIJobMgr) GetJobYaml(c *gin.Context) {
	var req AIJobDetailReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := interutil.GetToken(c)
	taskModel, err := mgr.taskService.GetByQueueAndID(token.AccountName, req.JobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}

	if taskModel.UserName != token.AccountName {
		resputil.Error(c, "Job not found", resputil.NotSpecified)
		return
	}

	job := &aijobapi.AIJob{}
	namespace := config.GetConfig().Workspace.Namespace
	if err = mgr.client.Get(c, client.ObjectKey{Name: taskModel.JobName,
		Namespace: namespace}, job); err != nil {
		resputil.Success(c, nil)
		return
	}

	// prune useless field
	job.ManagedFields = nil

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
	taskModel, err := mgr.taskService.GetByQueueAndID(token.AccountName, req.JobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}

	if taskModel.UserName != token.AccountName {
		resputil.Error(c, "Job not found", resputil.NotSpecified)
		return
	}

	// Get AIJob to check annotations
	job := &aijobapi.AIJob{}
	namespace := config.GetConfig().Workspace.Namespace
	if err = mgr.client.Get(c, client.ObjectKey{Name: taskModel.JobName, Namespace: namespace}, job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Construct the full URL directly
	host := config.GetConfig().Host
	baseURL := job.Labels[crclient.LabelKeyBaseURL]
	fullURL := fmt.Sprintf("https://%s/ingress/%s", host, baseURL)

	// Check if jupyter token has been cached in the job annotations
	const AnnotationKeyJupyter = "jupyter.token"
	jupyterToken, ok := job.Annotations[AnnotationKeyJupyter]
	if ok {
		resputil.Success(c, vcjob.JobTokenResp{
			BaseURL:   baseURL,
			Token:     jupyterToken,
			FullURL:   fullURL,
			PodName:   fmt.Sprintf("%s-0", taskModel.JobName),
			Namespace: namespace,
		})
		return
	}

	// Get the logs of the job pod to find token
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

	// Cache the jupyter token in the job annotations
	if job.Annotations == nil {
		job.Annotations = make(map[string]string)
	}
	job.Annotations[AnnotationKeyJupyter] = jupyterToken
	if err := mgr.client.Update(c, job); err != nil {
		// Log the error but continue, as this is not critical
		klog.Errorf("Failed to update job annotations: %v", err)
	}

	resputil.Success(c, vcjob.JobTokenResp{
		BaseURL:   baseURL,
		Token:     jupyterToken,
		FullURL:   fullURL,
		PodName:   podName,
		Namespace: namespace,
	})
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

// GetJobEvents godoc
//
//	@Summary		获取AI任务的事件
//	@Description	获取AI任务的事件
//	@Tags			AIJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		uint					true	"Job ID"
//	@Success		200	{object}	resputil.Response[any]	"事件列表"
//	@Failure		400	{object}	resputil.Response[any]	"请求参数错误"
//	@Failure		500	{object}	resputil.Response[any]	"其他错误"
//	@Router			/v1/aijobs/{id}/event [get]
func (mgr *AIJobMgr) GetJobEvents(c *gin.Context) {
	var req AIJobDetailReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := interutil.GetToken(c)
	taskModel, err := mgr.taskService.GetByQueueAndID(token.AccountName, req.JobID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("get task failed, err %v", err), resputil.NotSpecified)
		return
	}

	namespace := taskModel.Namespace
	jobName := taskModel.JobName

	if jobName == "" {
		// 作业还在排队中，模拟一个事件，说明作业正在排队
		jobName = taskModel.TaskName
		event := &v1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: namespace,
			},
			Reason:        "Queueing",
			Type:          "Warning",
			LastTimestamp: metav1.Time{Time: taskModel.CreatedAt},
			Message:       fmt.Sprintf("Job %s is queueing", jobName),
		}
		resputil.Success(c, []v1.Event{*event})
		return
	}

	// 获取任务相关事件
	jobEvents, err := mgr.kubeClient.CoreV1().Events(namespace).List(c, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", jobName),
		TypeMeta:      metav1.TypeMeta{Kind: "AIJob", APIVersion: "aisystem.github.com/v1alpha1"},
	})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	events := jobEvents.Items

	// 获取Pod相关事件
	podName := fmt.Sprintf("%s-0", taskModel.JobName)
	podEvents, err := mgr.kubeClient.CoreV1().Events(namespace).List(c, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", podName),
		TypeMeta:      metav1.TypeMeta{Kind: "Pod"},
	})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 如果存在Pod事件，则不返回Job事件
	if len(podEvents.Items) > 0 {
		events = []v1.Event{}
	}
	events = append(events, podEvents.Items...)

	// 按时间排序
	sort.Slice(events, func(i, j int) bool {
		return events[i].LastTimestamp.After(events[j].LastTimestamp.Time)
	})

	resputil.Success(c, events)
}
