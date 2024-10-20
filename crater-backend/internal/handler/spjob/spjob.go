package spjob

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/handler/vcjob"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	recommenddljobapi "github.com/raids-lab/crater/pkg/apis/recommenddljob/v1"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	utils "github.com/raids-lab/crater/pkg/util"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	handler.Registers = append(handler.Registers, NewSparseJobMgr)
}

const AnnotationKeyTaskName = "crater.raids.io/task-name"

var dlNamespace = config.GetConfig().Workspace.Namespace
var jobStatusMap = map[corev1.PodPhase]batch.JobPhase{
	corev1.PodFailed:    batch.Failed,
	corev1.PodSucceeded: batch.Completed,
	corev1.PodRunning:   batch.Running,
	corev1.PodPending:   batch.Pending,
	corev1.PodUnknown:   batch.Pending,
}

type SparseJobMgr struct {
	name      string
	jobclient *crclient.RecommendDLJobController
}

func NewSparseJobMgr(conf handler.RegisterConfig) handler.Manager {
	return &SparseJobMgr{
		name:      "spjobs",
		jobclient: &crclient.RecommendDLJobController{Client: conf.Client},
	}
}

func (mgr *SparseJobMgr) GetName() string { return mgr.name }

func (mgr *SparseJobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *SparseJobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("", mgr.List)
	g.GET("all", mgr.List)
	g.DELETE(":name", mgr.Delete)

	g.GET(":name/detail", mgr.GetByName)
	g.GET(":name/yaml", mgr.GetYaml)

	g.POST("training", mgr.Create)

	g.POST("/analyze", mgr.AnalyzeResourceUsage)
}

func (mgr *SparseJobMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("", mgr.List)
	g.GET(":name/detail", mgr.GetByName)
}

func (mgr *SparseJobMgr) rolePermit(token *util.JWTMessage, reqName string) bool {
	// TODO: 适配新的 Queue 机制，这先改成不报错的形式了
	if token.RolePlatform == model.RoleAdmin {
		return true
	}
	if token.Username != reqName {
		return false
	}
	return true
}

type (
	CreateRecommendDLJobReq struct {
		vcjob.CreateCustomReq
		RunningType         recommenddljobapi.RunningType `json:"runningType"`
		Macs                int64                         `json:"macs"`
		Params              int64                         `json:"params"`
		BatchSize           int                           `json:"batchSize"`
		EmbeddingSizeTotal  int64                         `json:"embeddingSizeTotal"`
		EmbeddingDimTotal   int                           `json:"embeddingDimTotal"`
		EmbeddingTableCount int                           `json:"embeddingTableCount"`
		VocabularySize      []int                         `json:"vocabularySize"`
		EmbeddingDim        []int                         `json:"embeddingDim"`
		Replicas            int32                         `json:"replicas"`
		InputTensor         []int                         `json:"inputTensor"`
	}
)

func (mgr *SparseJobMgr) Create(c *gin.Context) {
	req := &CreateRecommendDLJobReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request body failed, err:%v", err), resputil.NotSpecified)
		return
	}

	token := util.GetToken(c)

	volumes, volumeMounts, err := vcjob.GenerateVolumeMounts(c, token.UserID, req.VolumeMounts)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	baseURL := fmt.Sprintf("%s-%s", token.Username, uuid.New().String()[:5])
	jobName := fmt.Sprintf("sparse-%s", baseURL)

	annotations := map[string]string{
		AnnotationKeyTaskName: req.Name,
	}

	job := &recommenddljobapi.RecommendDLJob{
		ObjectMeta: v1.ObjectMeta{
			Name:        jobName,
			Namespace:   dlNamespace,
			Annotations: annotations,
		},
		Spec: recommenddljobapi.RecommendDLJobSpec{
			Replicas:    req.Replicas,
			RunningType: req.RunningType,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: volumes,
					Containers: []corev1.Container{
						{
							Name:    "sparse-recdl",
							Image:   req.Image,
							Command: []string{"sh", "-c", req.Command},
							Resources: corev1.ResourceRequirements{
								Limits: req.Resource,
							},
							WorkingDir: req.WorkingDir,

							Env:          req.Envs,
							VolumeMounts: volumeMounts,
						},
					},
				},
			},
			Username:            token.Username,
			Macs:                req.Macs,
			Params:              req.Params,
			BatchSize:           req.BatchSize,
			VocabularySize:      req.VocabularySize,
			EmbeddingDim:        req.EmbeddingDim,
			EmbeddingSizeTotal:  req.EmbeddingSizeTotal,
			EmbeddingDimTotal:   req.EmbeddingDimTotal,
			EmbeddingTableCount: req.EmbeddingTableCount,
			InputTensor:         req.InputTensor,
		},
	}

	if err := mgr.jobclient.CreateRecommendDLJob(c, job); err != nil {
		resputil.Error(c, fmt.Sprintf("create recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}

	resputil.Success(c, job.Name)
}

func (mgr *SparseJobMgr) List(c *gin.Context) {
	token := util.GetToken(c)

	jobList, err := mgr.jobclient.ListRecommendDLJob(c, dlNamespace)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}
	var jobs []vcjob.JobResp
	for _, spjob := range jobList {
		if !mgr.rolePermit(&token, spjob.Spec.Username) {
			continue
		}
		pod := mgr.GetPodsByName(c, spjob.Name)[0]
		conditions := pod.Status.Conditions
		var runningTimestamp v1.Time
		var completedTimestamp v1.Time
		for _, condition := range conditions {
			if condition.Type == corev1.PodReady {
				runningTimestamp = condition.LastTransitionTime
			}
		}
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			completedTimestamp = conditions[len(conditions)-1].LastTransitionTime
		}

		job := vcjob.JobResp{
			Name:               spjob.Annotations[AnnotationKeyTaskName],
			JobName:            spjob.Name,
			Owner:              spjob.Spec.Username,
			JobType:            "training",
			Queue:              token.QueueName,
			Status:             string(jobStatusMap[pod.Status.Phase]),
			CreationTimestamp:  spjob.CreationTimestamp,
			RunningTimestamp:   runningTimestamp,
			CompletedTimestamp: completedTimestamp,
			Nodes:              []string{pod.Spec.NodeName},
			Resources:          pod.Spec.Containers[0].Resources.Limits,
		}
		jobs = append(jobs, job)
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreationTimestamp.After(jobs[j].CreationTimestamp.Time)
	})

	resputil.Success(c, jobs)
}

type (
	GetRecommendDLJobReq struct {
		Name string `uri:"name" binding:"required"`
	}
	JobDetailResp struct {
		Name              string      `json:"name"`
		Namespace         string      `json:"namespace"`
		Username          string      `json:"username"`
		JobName           string      `json:"jobName"`
		Retry             string      `json:"retry"`
		Queue             string      `json:"queue"`
		Status            string      `json:"status"`
		CreationTimestamp v1.Time     `json:"createdAt"`
		RunningTimestamp  v1.Time     `json:"startedAt"`
		Duration          string      `json:"runtime"`
		PodDetails        []PodDetail `json:"podDetails"`
		UseTensorBoard    bool        `json:"useTensorBoard"`
	}

	PodDetail struct {
		Name     string `json:"name"`
		NodeName string `json:"nodename"`
		IP       string `json:"ip"`
		Port     string `json:"port"`
		Resource string `json:"resource"`
		Status   string `json:"status"`
	}
)

func (mgr *SparseJobMgr) GetByName(c *gin.Context) {
	token := util.GetToken(c)
	req := &GetRecommendDLJobReq{}
	if err := c.ShouldBindUri(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request query failed, err:%v", err), resputil.NotSpecified)
		return
	}
	var job *recommenddljobapi.RecommendDLJob
	var err error
	if job, err = mgr.jobclient.GetRecommendDLJob(c, req.Name, dlNamespace); err != nil {
		resputil.Error(c, fmt.Sprintf("get recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}

	var pods []*corev1.Pod
	PodDetails := []PodDetail{}

	if pods = mgr.GetPodsByName(c, job.Name); pods == nil {
		resputil.Error(c, "get recommenddljob failed, err: nil pods", resputil.NotSpecified)
		return
	}
	pod := pods[0]
	conditions := pod.Status.Conditions

	var runningTimestamp v1.Time
	var duration string
	for _, condition := range conditions {
		if condition.Type == corev1.PodReady {
			runningTimestamp = condition.LastTransitionTime
			duration = time.Since(runningTimestamp.Time).Truncate(time.Second).String()
		} else {
			runningTimestamp = v1.Time{}
			duration = "0s"
		}
	}

	retryAmount := -1
	for _, condition := range conditions {
		if condition.Type == corev1.ContainersReady {
			retryAmount += 1
		}
	}

	for _, pod := range pods {
		// assume one pod running one container
		if pod.Status.Phase == corev1.PodRunning {
			portStr := ""
			for _, port := range pod.Spec.Containers[0].Ports {
				portStr += fmt.Sprintf("%s:%d,", port.Name, port.ContainerPort)
			}
			if portStr != "" {
				portStr = portStr[:len(portStr)-1]
			}
			podDetail := PodDetail{
				Name:     pod.Name,
				NodeName: pod.Spec.NodeName,
				IP:       pod.Status.PodIP,
				Port:     portStr,
				Resource: model.ResourceListToJSON(pod.Spec.Containers[0].Resources.Requests),
				Status:   string(pod.Status.Phase),
			}
			PodDetails = append(PodDetails, podDetail)
		} else {
			podDetail := PodDetail{
				Name:   pod.Name,
				Status: string(pod.Status.Phase),
			}
			PodDetails = append(PodDetails, podDetail)
		}
	}

	ret := JobDetailResp{
		Name:              job.Name,
		JobName:           job.Name,
		Username:          token.Username,
		Namespace:         dlNamespace,
		Queue:             token.QueueName,
		Status:            string(jobStatusMap[pod.Status.Phase]),
		CreationTimestamp: job.CreationTimestamp,
		RunningTimestamp:  runningTimestamp,
		Duration:          duration,
		Retry:             fmt.Sprintf("%d", retryAmount),
		PodDetails:        PodDetails,
		UseTensorBoard:    false,
	}

	resputil.Success(c, ret)
}

type GetRecommendDLJobPodListReq struct {
	Name string `form:"name" binding:"required"`
}

func (mgr *SparseJobMgr) GetPodsByName(c *gin.Context, name string) []*corev1.Pod {
	var err error
	var podList []*corev1.Pod
	if podList, err = mgr.jobclient.GetRecommendDLJobPodList(c, name, dlNamespace); err != nil {
		resputil.Error(c, fmt.Sprintf("get recommenddljob pods failed, err:%v", err), resputil.NotSpecified)
		return nil
	}
	return podList
}

func (mgr *SparseJobMgr) GetYaml(c *gin.Context) {
	var req GetRecommendDLJobReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	var job *recommenddljobapi.RecommendDLJob
	var err error
	if job, err = mgr.jobclient.GetRecommendDLJob(c, req.Name, dlNamespace); err != nil {
		resputil.Error(c, fmt.Sprintf("get recommenddljob failed, err:%v", err), resputil.NotSpecified)
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

type DeleteRecommendDLJobReq struct {
	Name string `uri:"name" binding:"required"`
}

func (mgr *SparseJobMgr) Delete(c *gin.Context) {
	req := &DeleteRecommendDLJobReq{}
	if err := c.ShouldBindUri(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request body failed, err:%v", err), resputil.NotSpecified)
		return
	}
	var err error
	if _, err = mgr.jobclient.GetRecommendDLJob(c, req.Name, dlNamespace); err != nil {
		resputil.Error(c, fmt.Sprintf("delete recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}

	if err := mgr.jobclient.DeleteRecommendDLJob(c, req.Name, dlNamespace); err != nil {
		resputil.Error(c, fmt.Sprintf("delete recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nil)
}

type (
	AnalyzeRecommendDLJobReq struct {
		Replicas            int32                  `json:"replicas"`
		RunningType         string                 `json:"runningType"`
		DataSets            []string               `json:"datasets"`
		RelationShips       []string               `json:"relationShips"`
		Template            corev1.PodTemplateSpec `json:"template"`
		Username            string                 `json:"username"`
		Macs                int64                  `json:"macs"`
		Params              int64                  `json:"params"`
		BatchSize           int                    `json:"batchSize"`
		EmbeddingSizeTotal  int64                  `json:"embeddingSizeTotal"`
		EmbeddingDimTotal   int                    `json:"embeddingDimTotal"`
		EmbeddingTableCount int                    `json:"embeddingTableCount"`
		VocabularySize      []int                  `json:"vocabularySize"`
		EmbeddingDim        []int                  `json:"embeddingDim"`
		InputTensor         []int                  `json:"inputTensor"`
	}
	ResourceAnalyzeResult struct {
		GPUUtilAvg     float32 `json:"gpuUtilAvg"`
		GPUMemoryMax   float32 `json:"gpuMemoryMaxGB"`
		SMActiveAvg    float32 `json:"smActiveAvg"`
		SMOccupancyAvg float32 `json:"smOccupancyAvg"`
		FP32ActiveAvg  float32 `json:"fp32ActiveAvg"`
		DramActiveAvg  float32 `json:"dramActiveAvg"`
	}
	ResourceAnalyzeResponse    map[string]ResourceAnalyzeResult
	ResourceAnalyzeWebhookData struct {
		GPUUtilAvg     float32 `json:"gpu_util_avg"`
		GPUMemoryMax   float32 `json:"mem_usage"`
		SMActiveAvg    float32 `json:"sm_active_avg,omitempty"`
		SMOccupancyAvg float32 `json:"sm_occupied_avg,omitempty"`
		FP32ActiveAvg  float32 `json:"fp32_active_avg,omitempty"`
		DramActiveAvg  float32 `json:"dram_active_avg,omitempty"`
	}
	ResourceAnalyzeWebhookResponse struct {
		Code int                                   `json:"code"`
		Data map[string]ResourceAnalyzeWebhookData `json:"data"`
		Msg  string                                `json:"msg"`
	}
)

func (mgr *SparseJobMgr) AnalyzeResourceUsage(c *gin.Context) {
	req := &AnalyzeRecommendDLJobReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request body failed, err:%v", err), resputil.NotSpecified)
		return
	}
	if len(req.VocabularySize) != 0 {
		req.EmbeddingSizeTotal = 0
		for _, size := range req.VocabularySize {
			req.EmbeddingSizeTotal += int64(size)
		}
		req.EmbeddingTableCount = len(req.VocabularySize)
	}
	if len(req.EmbeddingDim) != 0 {
		req.EmbeddingDimTotal = 0
		for _, dim := range req.EmbeddingDim {
			req.EmbeddingDimTotal += dim
		}
	}
	if len(req.RelationShips) != 0 {
		req.EmbeddingSizeTotal = 0
		req.EmbeddingDimTotal = 0
		req.EmbeddingTableCount = 0
	}
	analyzeResp := &ResourceAnalyzeWebhookResponse{}
	if err := utils.PostJSON(c, "http://***REMOVED***:30500", "/api/v1/task/analyze/end2end", map[string]any{
		"embedding_table_count": req.EmbeddingTableCount,
		"embedding_dim_total":   req.EmbeddingDimTotal,
		"embedding_size_total":  req.EmbeddingSizeTotal / 1e4,
		"batch_size":            req.BatchSize,
		"params":                req.Params / 1e3,
		"macs":                  req.Macs / 1e6,
	}, nil, analyzeResp); err != nil {
		resputil.Error(c, fmt.Sprintf("request resource analyze failed, err:%v", err), resputil.NotSpecified)
		return
	}
	p100Mem := analyzeResp.Data["V100"].GPUMemoryMax
	//nolint:mnd // TODO: refactor
	if p100Mem > 16 {
		p100Mem = 16.01
	}
	resputil.Success(c, ResourceAnalyzeResponse{
		"p100": ResourceAnalyzeResult{
			GPUUtilAvg:   analyzeResp.Data["P100"].GPUUtilAvg,
			GPUMemoryMax: p100Mem,
		},
		"v100": ResourceAnalyzeResult{
			GPUUtilAvg:     analyzeResp.Data["V100"].GPUUtilAvg,
			GPUMemoryMax:   analyzeResp.Data["V100"].GPUMemoryMax,
			SMActiveAvg:    analyzeResp.Data["V100"].SMActiveAvg,
			SMOccupancyAvg: analyzeResp.Data["V100"].SMOccupancyAvg,
			DramActiveAvg:  analyzeResp.Data["V100"].DramActiveAvg,
			FP32ActiveAvg:  analyzeResp.Data["V100"].FP32ActiveAvg,
		},
	})
}
