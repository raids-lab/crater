package model

import "gorm.io/gorm"

// Default Quota
const (
	DefaultCPU    = 2
	DefaultMemory = 4
	DefaultGPU    = 0
)

// Quota belongs to project, defines the resource quota of a project
type Quota struct {
	gorm.Model
	ProjectID uint
	Project   Project
	// quota (cpu, memory, gpu, storage) for the project
	CPU     int    `gorm:"type:int;not null;default:0;comment:CPU配额(核)"`
	Memory  int    `gorm:"type:int;not null;default:0;comment:内存配额(Gi)"`
	GPU     int    `gorm:"type:int;not null;default:0;comment:GPU配额(个)"`
	GPUMem  int    `gorm:"type:int;not null;default:0;comment:GPU内存配额(Gi)"`
	Storage int    `gorm:"type:int;not null;default:0;comment:存储配额(Gi)"`
	Access  string `gorm:"type:varchar(512);comment:可访问的资源种类(V100,P100...)"`
}
