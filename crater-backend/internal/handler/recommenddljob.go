package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/crclient"
	utils "github.com/raids-lab/crater/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewRecommendDLJobMgr)
}

type RecommendDLJobMgr struct {
	name      string
	jobclient *crclient.RecommendDLJobController
}

func NewRecommendDLJobMgr(conf RegisterConfig) Manager {
	return &RecommendDLJobMgr{
		name:      "recommenddljob",
		jobclient: &crclient.RecommendDLJobController{Client: conf.Client},
	}
}

func (mgr *RecommendDLJobMgr) GetName() string { return mgr.name }

func (mgr *RecommendDLJobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *RecommendDLJobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.POST("/analyze", mgr.AnalyzeResourceUsage)
}

func (mgr *RecommendDLJobMgr) RegisterAdmin(_ *gin.RouterGroup) {
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

type GetRecommendDLJobPodListReq struct {
	Name string `form:"name" binding:"required"`
}

type DeleteRecommendDLJobReq struct {
	Name string `form:"name" binding:"required"`
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
