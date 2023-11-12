package models

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	DefaultQuota = v1.ResourceList{
		v1.ResourceCPU:                    resource.MustParse("40"),
		v1.ResourceMemory:                 resource.MustParse("80Gi"),
		v1.ResourceName("nvidia.com/gpu"): resource.MustParse("4"),
	}
)

// Quota model
type Quota struct {
	UserName  string `gorm:"primaryKey" json:"username"`
	NameSpace string `gorm:"column:namespace;type:varchar(128);not null" json:"namespace"`
	HardQuota string `gorm:"column:hardQuota;type:text;" json:"hardQuota"`
}
