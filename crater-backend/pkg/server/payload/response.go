package payload

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AllocatedInfo struct {
	CPU string `json:"cpu"`
	Mem string `json:"memory"`
	GPU string `json:"nvidia.com/gpu"`
}

type ClusterNodeInfo struct {
	Type      string              `json:"type"`
	Name      string              `json:"name"`
	Role      string              `json:"role"`
	Labels    map[string]string   `json:"labels"`
	IsReady   string              `json:"isReady"`
	Taint     string              `json:"taint"`
	Capacity  corev1.ResourceList `json:"capacity"`
	Allocated AllocatedInfo       `json:"allocated"`
	PodCount  int                 `json:"podCount"`
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
	IsReady                 string `json:"isReady"`
	Taint                   string `json:"taint"`
	Time                    string `json:"time"`
	Address                 string `json:"address"`
	Os                      string `json:"os"`
	OsVersion               string `json:"osVersion"`
	Arch                    string `json:"arch"`
	KubeletVersion          string `json:"kubeletVersion"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion"`
	GPUMemory               string `json:"gpuMemory"`
	GPUCount                int    `json:"gpuCount"`
	GPUArch                 string `json:"gpuArch"`
}

type ListNodeResp struct {
	Rows []ClusterNodeInfo `json:"rows"`
}

type GPUInfo struct {
	Name        string              `json:"name"`
	HaveGPU     bool                `json:"haveGPU"`
	GPUCount    int                 `json:"gpuCount"`
	GPUUtil     map[string]float32  `json:"gpuUtil"`
	RelateJobs  map[string][]string `json:"relateJobs"`
	GPUMemory   string              `json:"gpuMemory"`
	GPUArch     string              `json:"gpuArch"`
	GPUDriver   string              `json:"gpuDriver"`
	CudaVersion string              `json:"cudaVersion"`
	GPUProduct  string              `json:"gpuProduct"`
}
