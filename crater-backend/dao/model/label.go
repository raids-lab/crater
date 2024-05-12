package model

import (
	"gorm.io/gorm"
)

type Label struct {
	gorm.Model
	Label    string     `gorm:"uniqueIndex;type:varchar(256);not null;comment:标签名"`
	Name     string     `gorm:"type:varchar(256);not null;comment:别名"`
	Type     WorkerType `gorm:"not null;comment:类型"`
	Count    int        `gorm:"not null;comment:节点数量"`
	Priority int        `gorm:"not null;comment:优先级"`
}
