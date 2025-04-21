package model

import (
	"gorm.io/gorm"
)

// Resource model
//
// https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/
type Resource struct {
	gorm.Model

	// Resource name
	ResourceName string  `gorm:"uniqueIndex;type:varchar(255);not null;comment:资源名称" json:"name"`
	VendorDomain *string `gorm:"type:varchar(255);comment:供应商域名" json:"vendorDomain"`
	ResourceType string  `gorm:"type:varchar(255);not null;comment:资源类型" json:"resourceType"`

	// Resource quantity
	Amount          int64  `gorm:"not null;comment:资源总量" json:"amount"`
	AmountSingleMax int64  `gorm:"not null;comment:单机最大资源量" json:"amountSingleMax"`
	Format          string `gorm:"type:varchar(255);not null;comment:资源格式" json:"format"`
	Priority        int    `gorm:"not null;comment:优先级" json:"priority"`
	Label           string `gorm:"type:varchar(255);not null;comment:用于显示的别名" json:"label"`
}

const (
	NvidiaDomain = "nvidia.com"
)
