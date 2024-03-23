package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
	recommenddljobapi "github.com/raids-lab/crater/pkg/apis/recommenddljob/v1"
	"github.com/raids-lab/crater/pkg/crclient"
	usersvc "github.com/raids-lab/crater/pkg/db/user"
	"github.com/raids-lab/crater/pkg/server/payload"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	"github.com/raids-lab/crater/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RecommendDLJobMgr struct {
	userService usersvc.DBService
	jobclient   *crclient.RecommendDLJobController
}

func NewRecommendDLJobMgr(userSvc usersvc.DBService, crClient client.Client) *RecommendDLJobMgr {
	return &RecommendDLJobMgr{
		userService: userSvc,
		jobclient:   &crclient.RecommendDLJobController{Client: crClient},
	}
}

func (mgr *RecommendDLJobMgr) RegisterRoute(g *gin.RouterGroup) {
	g.POST("/create", mgr.Create)
	g.POST("/delete", mgr.Delete)
	g.GET("/list", mgr.List)
	g.GET("/info", mgr.GetByName)
	g.GET("/pods", mgr.GetPodsByName)
	g.POST("/analyze", mgr.AnalyzeResourceUsage)
}

func (mgr *RecommendDLJobMgr) Create(c *gin.Context) {
	userContext, _ := util.GetUserFromGinContext(c)
	req := &payload.CreateRecommendDLJobReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request body failed, err:%v", err), resputil.NotSpecified)
		return
	}
	job := &recommenddljobapi.RecommendDLJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      req.Name,
			Namespace: userContext.Namespace,
		},
		Spec: recommenddljobapi.RecommendDLJobSpec{
			Replicas:            req.Replicas,
			RunningType:         recommenddljobapi.RunningType(req.RunningType),
			DataSets:            make([]recommenddljobapi.DataSetRef, 0, len(req.DataSets)),
			RelationShips:       make([]recommenddljobapi.DataRelationShip, 0, len(req.RelationShips)),
			Template:            req.Template,
			Username:            userContext.UserName,
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
			JobNamespace: userContext.Namespace,
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
	resp := payload.GetRecommendDLJobResp{
		ObjectMeta: job.ObjectMeta,
		Spec:       &req.RecommendDLJobSpec,
		Status: &payload.RecommendDLJobStatus{
			Phase:    string(job.Status.Phase),
			PodNames: job.Status.PodNames,
		},
	}
	resputil.Success(c, resp)
}

func (mgr *RecommendDLJobMgr) List(c *gin.Context) {
	userContext, err := util.GetUserFromGinContext(c)
	if err != nil {
		resputil.Error(c, "get namespace failed", resputil.NotSpecified)
		return
	} else {
		resputil.Success(c, userContext.Namespace)
	}
	var jobList []*recommenddljobapi.RecommendDLJob
	if jobList, err = mgr.jobclient.ListRecommendDLJob(c, userContext.Namespace); err != nil {
		resputil.Error(c, fmt.Sprintf("list recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}
	ret := make(payload.ListRecommendDLJobResp, 0, len(jobList))
	for _, job := range jobList {
		//nolint:dupl // TODO: refactor
		retJob := payload.GetRecommendDLJobResp{
			ObjectMeta: job.ObjectMeta,
			Spec: &payload.RecommendDLJobSpec{
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
			Status: &payload.RecommendDLJobStatus{
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

func (mgr *RecommendDLJobMgr) GetByName(c *gin.Context) {
	userContext, _ := util.GetUserFromGinContext(c)
	req := &payload.GetRecommendDLJobReq{}
	if err := c.ShouldBindQuery(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request query failed, err:%v", err), resputil.NotSpecified)
		return
	}
	var job *recommenddljobapi.RecommendDLJob
	var err error
	if job, err = mgr.jobclient.GetRecommendDLJob(c, req.Name, userContext.Namespace); err != nil {
		resputil.Error(c, fmt.Sprintf("get recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}
	//nolint:dupl // TODO: refactor
	ret := payload.GetRecommendDLJobResp{
		ObjectMeta: job.ObjectMeta,
		Spec: &payload.RecommendDLJobSpec{
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
		Status: &payload.RecommendDLJobStatus{
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

func (mgr *RecommendDLJobMgr) GetPodsByName(c *gin.Context) {
	userContext, _ := util.GetUserFromGinContext(c)
	req := &payload.GetRecommendDLJobPodListReq{}
	if err := c.ShouldBindQuery(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request query failed, err:%v", err), resputil.NotSpecified)
		return
	}
	var podList []*corev1.Pod
	var err error
	if podList, err = mgr.jobclient.GetRecommendDLJobPodList(c, req.Name, userContext.Namespace); err != nil {
		resputil.Error(c, fmt.Sprintf("get recommenddljob pods failed, err:%v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, podList)
}

func (mgr *RecommendDLJobMgr) Delete(c *gin.Context) {
	userContext, _ := util.GetUserFromGinContext(c)
	req := &payload.DeleteRecommendDLJobReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		resputil.Error(c, fmt.Sprintf("bind request body failed, err:%v", err), resputil.NotSpecified)
		return
	}
	if err := mgr.jobclient.DeleteRecommendDLJob(c, req.Name, userContext.Namespace); err != nil {
		resputil.Error(c, fmt.Sprintf("delete recommenddljob failed, err:%v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, nil)
}

func (mgr *RecommendDLJobMgr) AnalyzeResourceUsage(c *gin.Context) {
	req := &payload.AnalyzeRecommendDLJobReq{}
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
	analyzeResp := &payload.ResourceAnalyzeWebhookResponse{}
	if err := util.PostJson(c, "http://***REMOVED***:30500", "/api/v1/task/analyze/end2end", map[string]any{
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
	//nolint:gomnd // TODO: refactor
	if p100Mem > 16 {
		p100Mem = 16.01
	}
	resputil.Success(c, &payload.ResourceAnalyzeResponse{
		"p100": payload.ResourceAnalyzeResult{
			GPUUtilAvg:   analyzeResp.Data["P100"].GPUUtilAvg,
			GPUMemoryMax: p100Mem,
		},
		"v100": payload.ResourceAnalyzeResult{
			GPUUtilAvg:     analyzeResp.Data["V100"].GPUUtilAvg,
			GPUMemoryMax:   analyzeResp.Data["V100"].GPUMemoryMax,
			SMActiveAvg:    analyzeResp.Data["V100"].SMActiveAvg,
			SMOccupancyAvg: analyzeResp.Data["V100"].SMOccupancyAvg,
			DramActiveAvg:  analyzeResp.Data["V100"].DramActiveAvg,
			FP32ActiveAvg:  analyzeResp.Data["V100"].FP32ActiveAvg,
		},
	})
}
