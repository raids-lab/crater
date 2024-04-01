package models

type GPU struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	GPUName     string `gorm:"column:gpu_name;type:varchar(128);not null" json:"gpuName"`
	GPUPriority int    `gorm:"column:gpu_priority" json:"gpuPriority"`
}
