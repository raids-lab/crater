package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type QueueQuotaLimit struct {
	gorm.Model
	Name  string                                `gorm:"uniqueIndex;type:varchar(256);not null;comment:队列名字"`
	Quota datatypes.JSONType[map[string]string] `gorm:"type:jsonb;comment:队列内资源限制"`
}

func (QueueQuotaLimit) TableName() string {
	return "queue_quotas"
}
