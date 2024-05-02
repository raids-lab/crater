package model

import "gorm.io/gorm"

type Queue struct {
	gorm.Model
	Name  string `gorm:"uniqueIndex;type:varchar(48);not null;comment:队列名称 (对应 Volcano Queue CRD)"`
	Space string `gorm:"uniqueIndex;type:varchar(512);not null;comment:队列空间标识"`
}

type UserQueue struct {
	gorm.Model
	UserID     uint       `gorm:"primaryKey"`
	QueueID    uint       `gorm:"primaryKey"`
	Role       Role       `gorm:"not null;comment:用户在队列中的角色 (user, admin)"`
	AccessMode AccessMode `gorm:"not null;comment:用户在队列空间的访问模式 (ro, rw)"`
}
