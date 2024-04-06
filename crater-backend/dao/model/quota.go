package model

import "gorm.io/gorm"

// Default Quota
// TODO: Make it configurable
const (
	defaultCPU     = 2  // 2 cores
	defaultMemory  = 4  // 4 Gi
	defaultGPU     = 0  // 0 GPU
	defaultStorage = 50 // 50 Gi
)

// Since quota is not constant, we need to define a function to return the copied default quota
func DefaultQuota() Quota {
	return Quota{
		JobReq:    -1,
		Job:       -1,
		NodeReq:   -1,
		Node:      -1,
		CPUReq:    defaultCPU,
		CPU:       defaultCPU,
		GPUReq:    defaultGPU,
		GPU:       defaultGPU,
		MemReq:    defaultMemory,
		Mem:       defaultMemory,
		GPUMemReq: -1,
		GPUMem:    -1,
		Storage:   defaultStorage,
	}
}

// Quota belongs to project, defines the resource quota of a project
type Quota struct {
	gorm.Model
	ProjectID uint
	Project   Project

	// quota (job, node, cpu, memory, gpu, gpuMem, storage) for the project
	// -1 means unlimited
	JobReq int `gorm:"type:int;not null;default:-1;comment:可以提交的 Job 数量"`
	Job    int `gorm:"type:int;not null;default:-1;comment:可以同时运行的 Job 数量"`

	NodeReq int `gorm:"type:int;not null;default:-1;comment:可以提交的节点数量"`
	Node    int `gorm:"type:int;not null;default:-1;comment:可以同时使用的节点数量"`

	CPUReq int `gorm:"type:int;not null;default:0;comment:可以提交的 CPU 核心数量"`
	CPU    int `gorm:"type:int;not null;default:0;comment:可以同时使用的 CPU 核心数量"`

	GPUReq int `gorm:"type:int;not null;default:0;comment:可以提交的 GPU 数量"`
	GPU    int `gorm:"type:int;not null;default:0;comment:可以同时使用的 GPU 数量"`

	MemReq int `gorm:"type:int;not null;default:0;comment:可以提交的内存配额 (Gi)"`
	Mem    int `gorm:"type:int;not null;default:0;comment:可以同时使用的内存配额 (Gi)"`

	GPUMemReq int `gorm:"type:int;not null;default:-1;comment:可以提交的GPU内存配额 (Gi)"`
	GPUMem    int `gorm:"type:int;not null;default:-1;comment:可以同时使用的GPU内存配额 (Gi)"`

	Storage int `gorm:"type:int;not null;default:50;comment:存储配额 (Gi)"`

	Extra *string `gorm:"comment:可访问的资源限制 (V100,P100...)"`
}
