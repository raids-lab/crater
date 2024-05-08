package model

// Default Quota
// TODO: Make it configurable
const (
	defaultCPU     = 2  // 2 cores
	defaultMemory  = 4  // 4 Gi
	defaultGPU     = 0  // 0 GPU
	defaultStorage = 50 // 50 Gi
	Unlimited      = -1 // -1 means unlimited
)

var (
	QuotaDefault = EmbeddedQuota{
		JobReq:    Unlimited,
		Job:       Unlimited,
		NodeReq:   Unlimited,
		Node:      Unlimited,
		CPUReq:    defaultCPU,
		CPU:       defaultCPU,
		GPUReq:    defaultGPU,
		GPU:       defaultGPU,
		MemReq:    defaultMemory,
		Mem:       defaultMemory,
		GPUMemReq: Unlimited,
		GPUMem:    Unlimited,
		Storage:   defaultStorage,
	}

	QuotaUnlimited = EmbeddedQuota{
		JobReq:    Unlimited,
		Job:       Unlimited,
		NodeReq:   Unlimited,
		Node:      Unlimited,
		CPUReq:    Unlimited,
		CPU:       Unlimited,
		GPUReq:    Unlimited,
		GPU:       Unlimited,
		MemReq:    Unlimited,
		Mem:       Unlimited,
		GPUMemReq: Unlimited,
		GPUMem:    Unlimited,
		Storage:   Unlimited,
	}
)

// Quota Definition for Project and User in Project
// quota (job, node, cpu, memory, gpu, gpuMem, storage) for the project
// -1 means unlimited
// -1 means unlimited
type EmbeddedQuota struct {
	JobReq int `gorm:"type:int;not null;comment:可以提交的 Job 数量" json:"jobReq"`
	Job    int `gorm:"type:int;not null;comment:可以同时运行的 Job 数量" json:"job"`

	NodeReq int `gorm:"type:int;not null;comment:可以提交的节点数量" json:"nodeReq"`
	Node    int `gorm:"type:int;not null;comment:可以同时使用的节点数量" json:"node"`

	CPUReq int `gorm:"type:int;not null;comment:可以提交的 CPU 核心数量" json:"cpuReq"`
	CPU    int `gorm:"type:int;not null;comment:可以同时使用的 CPU 核心数量" json:"cpu"`

	GPUReq int `gorm:"type:int;not null;comment:可以提交的 GPU 数量" json:"gpuReq"`
	GPU    int `gorm:"type:int;not null;comment:可以同时使用的 GPU 数量" json:"gpu"`

	MemReq int `gorm:"type:int;not null;comment:可以提交的内存配额 (Gi)" json:"memReq"`
	Mem    int `gorm:"type:int;not null;comment:可以同时使用的内存配额 (Gi)" json:"mem"`

	GPUMemReq int `gorm:"type:int;not null;comment:可以提交的GPU内存配额 (Gi)" json:"gpuMemReq"`
	GPUMem    int `gorm:"type:int;not null;comment:可以同时使用的GPU内存配额 (Gi)" json:"gpuMem"`

	Storage int `gorm:"type:int;not null;comment:存储配额 (Gi)" json:"storage"`

	Extra *string `gorm:"comment:可访问的资源限制 (V100,P100...)" json:"extra"`
}
