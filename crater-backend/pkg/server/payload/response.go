package payload

import (
	"github.com/raids-lab/crater/pkg/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CreateTaskResp struct {
	TaskID uint
}

type AllocatedInfo struct {
	CPU string `json:"cpu"`
	Mem string `json:"memory"`
	GPU string `json:"nvidia.com/gpu"`
}

type ClusterNodeInfo struct {
	Type     string              `json:"type"`
	Name     string              `json:"name"`
	Role     string              `json:"role"`
	Labels   map[string]string   `json:"labels"`
	IsReady  bool                `json:"isReady"`
	Capacity corev1.ResourceList `json:"capacity"`
	// Alocated v1.ResourceList   `json:"allocated"`
	Allocated AllocatedInfo `json:"allocated"`
}

type Pod struct {
	Name           string                  `json:"name"`
	Namespace      string                  `json:"namespace"`
	OwnerReference []metav1.OwnerReference `json:"ownerReference"`
	IP             string                  `json:"ip"`
	CreateTime     metav1.Time             `json:"createTime"`
	Status         corev1.PodPhase         `json:"status"`
	Resources      corev1.ResourceList     `json:"resources"`
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

type ListNodeResp struct {
	Rows []ClusterNodeInfo `json:"rows"`
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
