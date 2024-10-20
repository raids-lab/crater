package payload

import (
	"github.com/raids-lab/crater/pkg/models"
	v1 "k8s.io/api/core/v1"
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
	Type     string            `json:"type"`
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
	Namespace  string  `json:"namespace"`
	CPU        float32 `json:"cpu"`
	Mem        string  `json:"memory"`
	IP         string  `json:"ip"`
	CreateTime string  `json:"createTime"`
	Status     string  `json:"status"`
	OwnerKind  string  `json:"ownerKind"`
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
