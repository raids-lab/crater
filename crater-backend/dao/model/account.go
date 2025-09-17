package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	DefaultAccountID = 1
)

var (
	DefaultQuota = QueueQuota{
		Capability: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("80"),
			v1.ResourceMemory: resource.MustParse("160Gi"),
		},
	}
)

type QueueQuota struct {
	Guaranteed v1.ResourceList `json:"guaranteed,omitempty"`
	Deserved   v1.ResourceList `json:"deserved,omitempty"`
	Capability v1.ResourceList `json:"capability,omitempty"`
}

type Account struct {
	gorm.Model
	Name             string                         `gorm:"uniqueIndex;type:varchar(32);not null;comment:账户名称 (对应 Volcano Queue CRD)"`
	Nickname         string                         `gorm:"type:varchar(128);not null;comment:账户别名 (用于显示)"`
	Space            string                         `gorm:"uniqueIndex;type:varchar(512);not null;comment:账户空间绝对路径"`
	ExpiredAt        *time.Time                     `gorm:"comment:账户过期时间"`
	Quota            datatypes.JSONType[QueueQuota] `gorm:"comment:账户对应队列的资源配额"`
	UserDefaultQuota datatypes.JSONType[QueueQuota] `gorm:"comment:账户中用户默认的资源配额模版"`

	UserAccounts    []UserAccount
	AccountDatasets []AccountDataset
}

type UserAccount struct {
	gorm.Model
	UserID     uint       `gorm:"primaryKey"`
	AccountID  uint       `gorm:"primaryKey"`
	Role       Role       `gorm:"not null;comment:用户在账户中的角色 (user, admin)"`
	AccessMode AccessMode `gorm:"not null;comment:用户在账户空间的访问模式 (na, ro, rw)"`

	Quota datatypes.JSONType[QueueQuota] `gorm:"comment:用户在账户中的资源配额"`
}
