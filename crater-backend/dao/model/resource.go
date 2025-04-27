package model

import (
	"gorm.io/gorm"
)

type CraterResourceType string

const (
	ResourceTypeGPU  CraterResourceType = "gpu"
	ResourceTypeRDMA CraterResourceType = "rdma"
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

	// Resource relationship
	Type     *CraterResourceType `gorm:"type:varchar(32);comment:资源类型" json:"type"`
	Networks []*Resource         `gorm:"many2many:resource_networks;" json:"networks"`
}

// ResourceNetwork is the join table between Resource and Resource self
// The first foreign key is the gpu type resource,
// The second foreign key is the rdma type resource
// It is used to indicate that the two resources are connected
type ResourceNetwork struct {
	gorm.Model
	ResourceID uint `gorm:"primaryKey;comment:资源ID" json:"resourceId"`
	NetworkID  uint `gorm:"primaryKey;comment:网络ID" json:"networkId"`

	Resource Resource `gorm:"foreignKey:ResourceID;constraint:OnDelete:CASCADE;" json:"resource"`
	Network  Resource `gorm:"foreignKey:NetworkID;constraint:OnDelete:CASCADE;" json:"network"`
}

const (
	NvidiaDomain      = "nvidia.com"
	RDMASharedDomain  = "rdma"
	KoordinatorDomain = "koordinator.sh"
)
