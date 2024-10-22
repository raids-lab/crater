package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
)

const (
	DefaultQueueID = 1
)

type QueueQuota struct {
	Guaranteed v1.ResourceList `json:"guaranteed,omitempty"`
	Deserved   v1.ResourceList `json:"deserved,omitempty"`
	Capability v1.ResourceList `json:"capability,omitempty"`
}

type Queue struct {
	gorm.Model
	Name      string                         `gorm:"uniqueIndex;type:varchar(32);not null;comment:队列名称 (对应 Volcano Queue CRD)"`
	Nickname  string                         `gorm:"type:varchar(128);not null;comment:队列别名 (用于显示)"`
	Space     string                         `gorm:"uniqueIndex;type:varchar(512);not null;comment:队列空间绝对路径"`
	ExpiredAt time.Time                      `gorm:"comment:账户过期时间"`
	Quota     datatypes.JSONType[QueueQuota] `gorm:"comment:资源配额"`

	UserQueues    []UserQueue
	QueueDatasets []QueueDataset
}

type UserQueue struct {
	gorm.Model
	UserID     uint       `gorm:"primaryKey"`
	QueueID    uint       `gorm:"primaryKey"`
	Role       Role       `gorm:"not null;comment:用户在队列中的角色 (user, admin)"`
	AccessMode AccessMode `gorm:"not null;comment:用户在队列空间的访问模式 (na, ro, rw)"`

	Quota datatypes.JSONType[QueueQuota] `gorm:"comment:用户在队列中的资源配额"`
}
