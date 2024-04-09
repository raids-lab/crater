package model

import "time"

type Label struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string     `gorm:"type:varchar(128);not null;comment:标签名"`
	Type      WorkerType `gorm:"not null;comment:节点类型"`
	Priority  int        `gorm:"not null;comment:优先级"`
}
