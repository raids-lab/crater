package vcjob

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/utils"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	handler.Registers = append(handler.Registers, NewVolcanojobMgr)
}

type VolcanojobMgr struct {
	name       string
	client     client.Client
	kubeClient kubernetes.Interface
}

func NewVolcanojobMgr(conf handler.RegisterConfig) handler.Manager {
	return &VolcanojobMgr{
		name:       "vcjobs",
		client:     conf.Client,
		kubeClient: conf.KubeClient,
	}
}

func (mgr *VolcanojobMgr) GetName() string { return mgr.name }

func (mgr *VolcanojobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *VolcanojobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.GetUserJobs)
	g.GET("all", mgr.GetAllJobs)
	g.DELETE(":name", mgr.DeleteJob)

	g.GET(":name/detail", mgr.GetJobDetail)
	g.GET(":name/yaml", mgr.GetJobYaml)
	g.GET(":name/pods", mgr.GetJobPods)

	// jupyter
	g.POST("jupyter", mgr.CreateJupyterJob)
	g.GET(":name/token", mgr.GetJobToken)

	// training
	g.POST("training", mgr.CreateTrainingJob)

	// tensorflow
	g.POST("tensorflow", mgr.CreateTensorflowJob)

	// pytorch
	g.POST("pytorch", mgr.CreatePytorchJob)
}

func (mgr *VolcanojobMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.GetAllJobs)
}

const (
	VolcanoSchedulerName = "volcano"

	LabelKeyTaskType = "crater.raids.io/task-type"
	LabelKeyTaskUser = "crater.raids.io/task-user"
	LabelKeyBaseURL  = "crater.raids.io/base-url"

	AnnotationKeyTaskName       = "crater.raids.io/task-name"
	AnnotationKeyJupyter        = "crater.raids.io/jupyter-token"
	AnnotationKeyUseTensorBoard = "crater.raids.io/use-tensorboard"

	VolumeData  = "crater-workspace"
	VolumeCache = "crater-cache"

	JupyterPort     = 8888
	TensorBoardPort = 6006
)

type (
	JobActionReq struct {
		JobName string `uri:"name" binding:"required"`
	}

	JobTokenResp struct {
		BaseURL   string `json:"baseURL"`
		Token     string `json:"token"`
		PodName   string `json:"podName"`
		Namespace string `json:"namespace"`
	}
)

type (
	VolumeMount struct {
		Type      uint   `json:"type"`
		DatasetID uint   `json:"datasetID"`
		SubPath   string `json:"subPath"`
		MountPath string `json:"mountPath"`
	}

	DatasetMount struct {
		DatasetID uint   `json:"datasetID"`
		MountPath string `json:"mountPath"`
	}

	CreateJobCommon struct {
		VolumeMounts   []VolumeMount                `json:"volumeMounts,omitempty"`
		DatasetMounts  []DatasetMount               `json:"datasetMounts,omitempty"`
		Envs           []v1.EnvVar                  `json:"envs,omitempty"`
		Selectors      []v1.NodeSelectorRequirement `json:"selectors,omitempty"`
		UseTensorBoard bool                         `json:"useTensorBoard"`
	}
)

// DeleteJob godoc
// @Summary Delete the job
// @Description Delete the job by client-go
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param name path string true "Job Name"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name} [delete]

// 简化 DeleteJob 方法
// 无需显式删除 Ingress 和 Service
// OwnerReference 会自动处理
func (mgr *VolcanojobMgr) DeleteJob(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	j := query.Job
	job := &batch.Job{}
	namespace := config.GetConfig().Workspace.Namespace
	if err := mgr.client.Get(c, client.ObjectKey{Name: req.JobName, Namespace: namespace}, job); err != nil {
		if errors.IsNotFound(err) {
			if _, err = j.WithContext(c).Where(j.JobName.Eq(req.JobName)).Delete(); err != nil {
				resputil.Error(c, err.Error(), resputil.NotSpecified)
				return
			}
			resputil.Success(c, nil)
			return
		}
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	token := util.GetToken(c)
	if job.Labels[LabelKeyTaskUser] != token.Username {
		resputil.Error(c, "Job not found", resputil.NotSpecified)
		return
	}

	// update job status as deleted
	if _, err := j.WithContext(c).Where(j.JobName.Eq(req.JobName)).Updates(model.Job{
		Status:             model.Deleted,
		CompletedTimestamp: time.Now(),
		Nodes:              datatypes.NewJSONType([]string{}),
	}); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 直接删除 Job，OwnerReference 会自动删除 Ingress 和 Service
	if err := mgr.client.Delete(c, job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, nil)
}

func (mgr *VolcanojobMgr) getPodLog(c *gin.Context, namespace, podName string) (*bytes.Buffer, error) {
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

type GetJobLogResp struct {
	Logs map[string]string `json:"logs"`
}

type (
	JobResp struct {
		Name               string          `json:"name"`
		JobName            string          `json:"jobName"`
		Owner              string          `json:"owner"`
		JobType            string          `json:"jobType"`
		Queue              string          `json:"queue"`
		Status             string          `json:"status"`
		CreationTimestamp  metav1.Time     `json:"createdAt"`
		RunningTimestamp   metav1.Time     `json:"startedAt"`
		CompletedTimestamp metav1.Time     `json:"completedAt"`
		Nodes              []string        `json:"nodes"`
		Resources          v1.ResourceList `json:"resources"`
		KeepWhenLowUsage   bool            `json:"keepWhenLowUsage"`
	}
)

// GetUserJobs godoc
// @Summary Get the jobs of the user
// @Description Get the jobs of the user by client-go
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "Volcano Job List"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs [get]
func (mgr *VolcanojobMgr) GetUserJobs(c *gin.Context) {
	token := util.GetToken(c)

	// TODO: add indexer to list jobs by user
	j := query.Job
	jobs, err := j.WithContext(c).Preload(j.Account).Preload(j.User).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.QueueID)).Find()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	jobList := convertJobResp(jobs)

	resputil.Success(c, jobList)
}

// GetAllJobs godoc
// @Summary Get all of the jobs
// @Description Get all jobs  by client-go
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} resputil.Response[any] "admin get Volcano Job List"
// @Failure 400 {object} resputil.Response[any] "admin Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/all [get]
func (mgr *VolcanojobMgr) GetAllJobs(c *gin.Context) {
	j := query.Job
	jobs, err := j.WithContext(c).Preload(j.Account).Preload(query.Job.User).Find()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	jobList := convertJobResp(jobs)

	resputil.Success(c, jobList)
}

func convertJobResp(jobs []*model.Job) []JobResp {
	jobList := make([]JobResp, len(jobs))
	for i := range jobs {
		job := jobs[i]
		jobList[i] = JobResp{
			Name:               job.Name,
			JobName:            job.JobName,
			Owner:              job.User.Nickname,
			JobType:            string(job.JobType),
			Queue:              job.Account.Nickname,
			Status:             string(job.Status),
			CreationTimestamp:  metav1.NewTime(job.CreationTimestamp),
			RunningTimestamp:   metav1.NewTime(job.RunningTimestamp),
			CompletedTimestamp: metav1.NewTime(job.CompletedTimestamp),
			Nodes:              job.Nodes.Data(),
			Resources:          job.Resources.Data(),
			KeepWhenLowUsage:   job.KeepWhenLowResourceUsage,
		}
	}
	sort.Slice(jobList, func(i, j int) bool {
		return jobList[i].CreationTimestamp.After(jobList[j].CreationTimestamp.Time)
	})
	return jobList
}

type (
	JobDetailResp struct {
		Name               string         `json:"name"`
		Namespace          string         `json:"namespace"`
		Username           string         `json:"username"`
		JobName            string         `json:"jobName"`
		JobType            model.JobType  `json:"jobType"`
		Queue              string         `json:"queue"`
		Status             batch.JobPhase `json:"status"`
		CreationTimestamp  metav1.Time    `json:"createdAt"`
		RunningTimestamp   metav1.Time    `json:"startedAt"`
		CompletedTimestamp metav1.Time    `json:"completedAt"`
	}

	PodDetail struct {
		Name      string          `json:"name"`
		Namespace string          `json:"namespace"`
		NodeName  *string         `json:"nodename"`
		IP        string          `json:"ip"`
		Port      string          `json:"port"`
		Resource  v1.ResourceList `json:"resource"`
		Phase     v1.PodPhase     `json:"phase"`
	}
)

// GetJobDetail godoc
// @Summary 获取jupyter详情
// @Description 调用k8s get crd
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param jobname query string true "vcjob-name"
// @Success 200 {object} resputil.Response[any] "任务描述"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name}/detail [get]
func (mgr *VolcanojobMgr) GetJobDetail(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)

	// find from db
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	jobDetail := JobDetailResp{
		Name:               job.Name,
		Namespace:          job.Attributes.Data().Namespace,
		Username:           job.User.Nickname,
		JobName:            job.JobName,
		JobType:            job.JobType,
		Queue:              job.Account.Nickname,
		Status:             job.Status,
		CreationTimestamp:  metav1.NewTime(job.CreationTimestamp),
		RunningTimestamp:   metav1.NewTime(job.RunningTimestamp),
		CompletedTimestamp: metav1.NewTime(job.CompletedTimestamp),
	}
	resputil.Success(c, jobDetail)
}

func getJob(c context.Context, name string, token *util.JWTMessage) (*model.Job, error) {
	j := query.Job
	if token.RolePlatform == model.RoleAdmin {
		return j.WithContext(c).
			Preload(j.Account).
			Preload(j.User).
			Where(j.JobName.Eq(name)).
			First()
	} else {
		return j.WithContext(c).
			Preload(j.Account).
			Preload(j.User).
			Where(j.JobName.Eq(name)).
			Where(j.AccountID.Eq(token.QueueID)).
			Where(j.UserID.Eq(token.UserID)).
			First()
	}
}

// GetJobPods godoc
// @Summary 获取任务的Pod列表
// @Description 获取任务的Pod列表
// @Tags VolcanoJob
// @Accept json
// @Produce json
// @Security Bearer
// @Param jobname query string true "vcjob-name"
// @Success 200 {object} resputil.Response[any] "Pod列表"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/vcjobs/{name}/pods [get]
func (mgr *VolcanojobMgr) GetJobPods(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)

	// find from db
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// every pod has label crater.raids.io/base-url: tf-liyilong-1314c
	// get pods with label selector
	vcjob := job.Attributes.Data()
	var podList = &v1.PodList{}
	if value, ok := vcjob.Labels[LabelKeyBaseURL]; !ok {
		resputil.Error(c, "label not found", resputil.NotSpecified)
		return
	} else {
		labels := client.MatchingLabels{LabelKeyBaseURL: value}
		err = mgr.client.List(c, podList, client.InNamespace(vcjob.Namespace), labels)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
	}

	PodDetails := []PodDetail{}
	for i := range podList.Items {
		pod := &podList.Items[i]

		// resource
		resources := utils.CalculateRequsetsByContainers(pod.Spec.Containers)

		portStr := ""
		for _, port := range pod.Spec.Containers[0].Ports {
			portStr += fmt.Sprintf("%s:%d,", port.Name, port.ContainerPort)
		}
		if portStr != "" {
			portStr = portStr[:len(portStr)-1]
		}
		podDetail := PodDetail{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			NodeName:  lo.ToPtr(pod.Spec.NodeName),
			IP:        pod.Status.PodIP,
			Port:      portStr,
			Resource:  resources,
			Phase:     pod.Status.Phase,
		}
		PodDetails = append(PodDetails, podDetail)
	}

	resputil.Success(c, PodDetails)
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
// @Router /v1/vcjobs/{name}/yaml [get]
func (mgr *VolcanojobMgr) GetJobYaml(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	// find from db
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	vcjob := job.Attributes.Data()

	// prune useless field
	vcjob.ObjectMeta.ManagedFields = nil

	// utilize json omitempty tag to further prune
	jsonData, err := json.Marshal(vcjob)
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

	// remove status field
	delete(prunedJob, "status")

	JobYaml, err := marshalYAMLWithIndent(prunedJob, 2)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, string(JobYaml))
}

func marshalYAMLWithIndent(v any, indent int) ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(indent)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
