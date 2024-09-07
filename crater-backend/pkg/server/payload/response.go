package payload

import (
	"time"

	"github.com/raids-lab/crater/pkg/models"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CreateTaskResp struct {
	TaskID uint
}

type ListTaskResp struct {
	RowCount int64           `json:"rowCount"`
	Rows     []models.AITask `json:"rows"`
}

type AllocatedInfo struct {
	CPU string `json:"cpu"`
	Mem string `json:"memory"`
	GPU string `json:"nvidia.com/gpu"`
}

type ClusterNodeInfo struct {
	Name     string            `json:"name"`
	Role     string            `json:"role"`
	Labels   map[string]string `json:"labels"`
	IsReady  bool              `json:"isReady"`
	Capacity v1.ResourceList   `json:"capacity"`
	// Alocated v1.ResourceList   `json:"allocated"`
	Allocated AllocatedInfo `json:"allocated"`
}

type Pod struct {
	Name       string  `json:"name"`
	CPU        float32 `json:"CPU"`
	Mem        string  `json:"Mem"`
	IP         string  `json:"IP"`
	CreateTime string  `json:"createTime"`
	Status     string  `json:"status"`
	IsVcjob    string  `json:"isVcjob"`
}

type ClusterNodeDetail struct {
	Name                    string `json:"name"`
	Role                    string `json:"role"`
	IsReady                 bool   `json:"isReady"`
	Time                    string `json:"time"`
	Address                 string `json:"address"`
	Os                      string `json:"os"`
	OsVersion               string `json:"osVersion"`
	Arch                    string `json:"arch"`
	KubeletVersion          string `json:"kubeletVersion"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion"`
}

type ClusterNodePodInfo struct {
	Pods []Pod `json:"pods"`
}

type ListNodeResp struct {
	Rows []ClusterNodeInfo `json:"rows"`
}

type ListNodePodResp struct {
	Rows []ClusterNodePodInfo `json:"rows"`
}

type GPUInfo struct {
	Name       string              `json:"name"`
	HaveGPU    bool                `json:"haveGPU"`
	GPUCount   int                 `json:"gpuCount"`
	GPUUtil    map[string]float32  `json:"gpuUtil"`
	RelateJobs map[string][]string `json:"relateJobs"`
}

type GetTaskResp struct {
	models.AITask
}

type GetTaskLogResp struct {
	Logs []string `json:"logs"`
}

type GetJupyterResp struct {
	Port  int32  `json:"port"`
	Token string `json:"token"`
}

type DeleteTaskResp struct {
}

type UpdateTaskResp struct {
}

type ListUserResp struct {
	Users []GetUserResp
}

type ListUserQuotaResp struct {
	Quotas []GetQuotaResp
}

type GetQuotaResp struct {
	User     string          `json:"user"`
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

type AITaskStatistic struct {
	TaskCount []models.TaskStatusCount `json:"taskCount"`
}

type AITaskCountStatistic struct {
	Queueing int `json:"queueing"`
	Pending  int `json:"pending"`
	Running  int `json:"running"`
	Finished int `json:"finished"`
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

// type ImagePackGetByNameResponse struct {
// 	ImagePack models.ImagePack `json:"imagepack"`
// }
