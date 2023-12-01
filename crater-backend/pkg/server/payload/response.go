package payload

import (
	"time"

	"github.com/aisystem/ai-protal/pkg/models"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CreateTaskResp struct {
	TaskID uint
}

type ListTaskResp struct {
	Tasks []models.AITask
}

type GetTaskResp struct {
	models.AITask
}

type DeleteTaskResp struct {
}

type UpdateTaskResp struct {
}

type ListUserResp struct {
	Users []GetUserResp
}

type GetQuotaResp struct {
	Hard     v1.ResourceList `json:"hard"`
	HardUsed v1.ResourceList `json:"hardUsed"`
	SoftUsed v1.ResourceList `json:"softUsed"`
}

type GetUserResp struct {
	UserID    uint            `json:"userID"`
	UserName  string          `json:"userName"`
	Role      string          `json:"role"`
	QuotaHard v1.ResourceList `json:"quotaHard"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type RecommendDLJobSpec struct {
	Replicas            int32              `json:"replicas"`
	RunningType         string             `json:"runningType"`
	DataSets            []string           `json:"datasets"`
	RelationShips       []string           `json:"relationShips"`
	Template            v1.PodTemplateSpec `json:"template"`
	Username            string             `json:"username"`
	Macs                int64              `json:"macs"`
	Params              int64              `json:"params"`
	BatchSize           int                `json:"batchSize"`
	EmbeddingSizeTotal  int64              `json:"embeddingSizeTotal"`
	EmbeddingDimTotal   int                `json:"embeddingDimTotal"`
	EmbeddingTableCount int                `json:"embeddingTableCount"`
	VocabularySize      []int              `json:"vocabularySize"`
	EmbeddingDim        []int              `json:"embeddingDim"`
	InputTensor         []int              `json:"inputTensor"`
}

type RecommendDLJobStatus struct {
	Phase    string   `json:"phase"`
	PodNames []string `json:"podNames"`
}

type GetRecommendDLJobResp struct {
	metav1.ObjectMeta
	Spec   *RecommendDLJobSpec   `json:"spec"`
	Status *RecommendDLJobStatus `json:"status"`
}

type ListRecommendDLJobResp []GetRecommendDLJobResp

type DatasetSpec struct {
	PVC         string `json:"pvc"`
	DownloadURL string `json:"downloadUrl"`
	Size        int    `json:"size"`
}

type DatasetStaus struct {
	Phase string `json:"phase"`
}

type GetDatasetResp struct {
	metav1.ObjectMeta
	Spec   *DatasetSpec  `json:"spec"`
	Status *DatasetStaus `json:"status"`
}

type ListDatasetResp []GetDatasetResp

type ResourceAnalyzeResponse map[string]ResourceAnalyzeResult

type ResourceAnalyzeResult struct {
	GPUUtilAvg     float32 `json:"gpuUtilAvg"`
	GPUMemoryMax   float32 `json:"gpuMemoryMaxGB"`
	SMActiveAvg    float32 `json:"smActiveAvg"`
	SMOccupancyAvg float32 `json:"smOccupancyAvg"`
	FP32ActiveAvg  float32 `json:"fp32ActiveAvg"`
	DramActiveAvg  float32 `json:"dramActiveAvg"`
}

type ResourceAnalyzeWebhookData struct {
	GPUUtilAvg     float32 `json:"gpu_util_avg"`
	GPUMemoryMax   float32 `json:"mem_usage"`
	SMActiveAvg    float32 `json:"sm_active_avg,omitempty"`
	SMOccupancyAvg float32 `json:"sm_occupied_avg,omitempty"`
	FP32ActiveAvg  float32 `json:"fp32_active_avg,omitempty"`
	DramActiveAvg  float32 `json:"dram_active_avg,omitempty"`
}

type ResourceAnalyzeWebhookResponse struct {
	Code int                                   `json:"code"`
	Data map[string]ResourceAnalyzeWebhookData `json:"data"`
	Msg  string                                `json:"msg"`
}
