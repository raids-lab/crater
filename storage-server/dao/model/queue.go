package model

import "gorm.io/gorm"

const (
	DefaultQueueID = 1
)

type Queue struct {
	gorm.Model
	Name     string `gorm:"uniqueIndex;type:varchar(32);not null;comment:队列名称 (对应 Volcano Queue CRD)"`
	Nickname string `gorm:"type:varchar(128);not null;comment:队列别名 (用于显示)"`
	Space    string `gorm:"uniqueIndex;type:varchar(512);not null;comment:队列空间绝对路径"`

	UserQueues    []UserQueue
	QueueDatasets []QueueDataset
}

type UserQueue struct {
	gorm.Model
	UserID     uint       `gorm:"primaryKey"`
	QueueID    uint       `gorm:"primaryKey"`
	Role       Role       `gorm:"not null;comment:用户在队列中的角色 (user, admin)"`
	AccessMode AccessMode `gorm:"not null;comment:用户在队列空间的访问模式 (ro, rw)"`
}
