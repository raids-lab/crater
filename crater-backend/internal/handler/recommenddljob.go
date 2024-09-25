package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	recommenddljobapi "github.com/raids-lab/crater/pkg/apis/recommenddljob/v1"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	utils "github.com/raids-lab/crater/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var dlNamespace = config.GetConfig().Workspace.Namespace

type RecommendDLJobMgr struct {
	jobclient *crclient.RecommendDLJobController
}

func NewRecommendDLJobMgr(crClient client.Client) *RecommendDLJobMgr {
	return &RecommendDLJobMgr{
		jobclient: &crclient.RecommendDLJobController{Client: crClient},
	}
}

func (mgr *RecommendDLJobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *RecommendDLJobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.POST("/create", mgr.Create)
	g.POST("/delete", mgr.Delete)
	g.GET("/list", mgr.List)
	g.GET("/info", mgr.GetByName)
	g.GET("/pods", mgr.GetPodsByName)
	g.POST("/analyze", mgr.AnalyzeResourceUsage)
}

func (mgr *RecommendDLJobMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.POST("/create", mgr.Create)
	g.POST("/delete", mgr.Delete)
	g.GET("/list", mgr.List)
	g.GET("/info", mgr.GetByName)
	g.GET("/pods", mgr.GetPodsByName)
	g.POST("/analyze", mgr.AnalyzeResourceUsage)
}

func (mgr *RecommendDLJobMgr) rolePermit(token *util.JWTMessage, reqName string) bool {
	// TODO: 适配新的 Queue 机制，这先改成不报错的形式了
	var uid, pid uint
	if num, err := fmt.Sscanf("%d-%d", reqName, uid, pid); err != nil || num != 2 {
		return false
	}
	ok := false
	if token.RolePlatform == model.RoleAdmin {
		ok = true
	}
	return ok
}

type (
	RecommendDLJobSpec struct {
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
	CreateRecommendDLJobReq struct {
		Name string `json:"name" binding:"required"`
		RecommendDLJobSpec
	}
	RecommendDLJobStatus struct {
		Phase    string   `json:"phase"`
		PodNames []string `json:"podNames"`
	}
	GetRecommendDLJobResp struct {
		v1.ObjectMeta
		Spec   *RecommendDLJobSpec   `json:"spec"`
		Status *RecommendDLJobStatus `json:"status"`
	}
)

func (mgr *RecommendDLJobMgr) Create(c *gin.Context) {
	token := util.GetToken(c)
	req := &CreateRecommendDLJobReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request body failed, err:%v", err), resputil.NotSpecified)
		return
	}
	job := &recommenddljobapi.RecommendDLJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      req.Name,
			Namespace: dlNamespace,
		},
		Spec: recommenddljobapi.RecommendDLJobSpec{
			Replicas:            req.Replicas,
			RunningType:         recommenddljobapi.RunningType(req.RunningType),
			DataSets:            make([]recommenddljobapi.DataSetRef, 0, len(req.DataSets)),
			RelationShips:       make([]recommenddljobapi.DataRelationShip, 0, len(req.RelationShips)),
			Template:            req.Template,
			Username:            token.Username,
			Macs:                req.Macs,
			Params:              req.Params,
			BatchSize:           req.BatchSize,
			EmbeddingSizeTotal:  req.EmbeddingSizeTotal,
			EmbeddingDimTotal:   req.EmbeddingDimTotal,
			EmbeddingTableCount: req.EmbeddingTableCount,
			VocabularySize:      req.VocabularySize,
			EmbeddingDim:        req.EmbeddingDim,
			InputTensor:         req.InputTensor,
		},
	}
	for _, releationShip := range req.RelationShips {
		job.Spec.RelationShips = append(job.Spec.RelationShips, recommenddljobapi.DataRelationShip{
			Type:         recommenddljobapi.DataRelationShipType("input"),
			JobName:      releationShip,
			JobNamespace: dlNamespace,
		})
	}
	for _, datasetName := range req.DataSets {
		job.Spec.DataSets = append(job.Spec.DataSets, recommenddljobapi.DataSetRef{
			Name: datasetName,
		})
	}

	if err := mgr.jobclient.CreateRecommendDLJob(c, job); err != nil {
		resputil.Error(c, fmt.Sprintf("create recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}
	resp := GetRecommendDLJobResp{
		ObjectMeta: job.ObjectMeta,
		Spec:       &req.RecommendDLJobSpec,
		Status: &RecommendDLJobStatus{
			Phase:    string(job.Status.Phase),
			PodNames: job.Status.PodNames,
		},
	}
	resputil.Success(c, resp)
}

type (
	ListRecommendDLJobResp []GetRecommendDLJobResp
)

func (mgr *RecommendDLJobMgr) List(c *gin.Context) {
	token := util.GetToken(c)

	jobList, err := mgr.jobclient.ListRecommendDLJob(c, dlNamespace)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("list recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}
	ret := make(ListRecommendDLJobResp, 0, len(jobList))
	for _, job := range jobList {
		if !mgr.rolePermit(&token, job.Spec.Username) {
			continue
		}
		//nolint:dupl // TODO: refactor
		retJob := GetRecommendDLJobResp{
			ObjectMeta: job.ObjectMeta,
			Spec: &RecommendDLJobSpec{
				Replicas:            job.Spec.Replicas,
				RunningType:         string(job.Spec.RunningType),
				DataSets:            make([]string, 0, len(job.Spec.DataSets)),
				RelationShips:       make([]string, 0, len(job.Spec.RelationShips)),
				Template:            job.Spec.Template,
				Username:            job.Spec.Username,
				Macs:                job.Spec.Macs,
				Params:              job.Spec.Params,
				BatchSize:           job.Spec.BatchSize,
				EmbeddingSizeTotal:  job.Spec.EmbeddingSizeTotal,
				EmbeddingDimTotal:   job.Spec.EmbeddingDimTotal,
				EmbeddingTableCount: job.Spec.EmbeddingTableCount,
				VocabularySize:      job.Spec.VocabularySize,
				EmbeddingDim:        job.Spec.EmbeddingDim,
				InputTensor:         job.Spec.InputTensor,
			},
			Status: &RecommendDLJobStatus{
				Phase:    string(job.Status.Phase),
				PodNames: job.Status.PodNames,
			},
		}
		for _, dataset := range job.Spec.DataSets {
			retJob.Spec.DataSets = append(retJob.Spec.DataSets, dataset.Name)
		}
		for _, releationship := range job.Spec.RelationShips {
			retJob.Spec.RelationShips = append(retJob.Spec.RelationShips, releationship.JobName)
		}
		ret = append(ret, retJob)
	}
	resputil.Success(c, ret)
}

type GetRecommendDLJobReq struct {
	Name string `form:"name" binding:"required"`
}

func (mgr *RecommendDLJobMgr) GetByName(c *gin.Context) {
	token := util.GetToken(c)
	req := &GetRecommendDLJobReq{}
	if err := c.ShouldBindQuery(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request query failed, err:%v", err), resputil.NotSpecified)
		return
	}
	var job *recommenddljobapi.RecommendDLJob
	var err error
	if job, err = mgr.jobclient.GetRecommendDLJob(c, req.Name, dlNamespace); err != nil {
		resputil.Error(c, fmt.Sprintf("get recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}
	if !mgr.rolePermit(&token, job.Spec.Username) {
		resputil.Error(c, "get recommenddljob failed, err: access deny", resputil.NotSpecified)
		return
	}
	//nolint:dupl // TODO: refactor
	ret := GetRecommendDLJobResp{
		ObjectMeta: job.ObjectMeta,
		Spec: &RecommendDLJobSpec{
			Replicas:            job.Spec.Replicas,
			RunningType:         string(job.Spec.RunningType),
			DataSets:            make([]string, 0, len(job.Spec.DataSets)),
			RelationShips:       make([]string, 0, len(job.Spec.RelationShips)),
			Template:            job.Spec.Template,
			Username:            job.Spec.Username,
			Macs:                job.Spec.Macs,
			Params:              job.Spec.Params,
			BatchSize:           job.Spec.BatchSize,
			EmbeddingSizeTotal:  job.Spec.EmbeddingSizeTotal,
			EmbeddingDimTotal:   job.Spec.EmbeddingDimTotal,
			EmbeddingTableCount: job.Spec.EmbeddingTableCount,
			VocabularySize:      job.Spec.VocabularySize,
			EmbeddingDim:        job.Spec.EmbeddingDim,
			InputTensor:         job.Spec.InputTensor,
		},
		Status: &RecommendDLJobStatus{
			Phase:    string(job.Status.Phase),
			PodNames: job.Status.PodNames,
		},
	}
	for _, dataset := range job.Spec.DataSets {
		ret.Spec.DataSets = append(ret.Spec.DataSets, dataset.Name)
	}
	for _, releationship := range job.Spec.RelationShips {
		ret.Spec.RelationShips = append(ret.Spec.RelationShips, releationship.JobName)
	}
	resputil.Success(c, ret)
}

type GetRecommendDLJobPodListReq struct {
	Name string `form:"name" binding:"required"`
}

func (mgr *RecommendDLJobMgr) GetPodsByName(c *gin.Context) {
	token := util.GetToken(c)
	req := &GetRecommendDLJobPodListReq{}
	if err := c.ShouldBindQuery(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request query failed, err:%v", err), resputil.NotSpecified)
		return
	}
	var job *recommenddljobapi.RecommendDLJob
	var err error
	if job, err = mgr.jobclient.GetRecommendDLJob(c, req.Name, dlNamespace); err != nil {
		resputil.Error(c, fmt.Sprintf("get recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}
	if !mgr.rolePermit(&token, job.Spec.Username) {
		resputil.Error(c, "get recommenddljob pods failed, err: access deny", resputil.NotSpecified)
		return
	}
	var podList []*corev1.Pod
	if podList, err = mgr.jobclient.GetRecommendDLJobPodList(c, req.Name, dlNamespace); err != nil {
		resputil.Error(c, fmt.Sprintf("get recommenddljob pods failed, err:%v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, podList)
}

type DeleteRecommendDLJobReq struct {
	Name string `form:"name" binding:"required"`
}

func (mgr *RecommendDLJobMgr) Delete(c *gin.Context) {
	token := util.GetToken(c)
	req := &DeleteRecommendDLJobReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request body failed, err:%v", err), resputil.NotSpecified)
		return
	}
	var job *recommenddljobapi.RecommendDLJob
	var err error
	if job, err = mgr.jobclient.GetRecommendDLJob(c, req.Name, dlNamespace); err != nil {
		resputil.Error(c, fmt.Sprintf("delete recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}
	if !mgr.rolePermit(&token, job.Spec.Username) {
		resputil.Error(c, "get recommenddljob pods failed, err: access deny", resputil.NotSpecified)
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
		RecommendDLJobSpec
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

func (mgr *RecommendDLJobMgr) AnalyzeResourceUsage(c *gin.Context) {
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
