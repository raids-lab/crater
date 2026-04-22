package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const DefaultPrequeueCandidateSize = 10

type QueueQuotaLimit struct {
	gorm.Model
	Name                  string                                `gorm:"uniqueIndex;type:varchar(256);not null;comment:队列名字"`
	Enabled               bool                                  `gorm:"not null;default:false;comment:是否启用队列资源限制"`
	PrequeueCandidateSize int                                   `gorm:"not null;default:10;comment:Prequeue 候选作业集大小"`
	Quota                 datatypes.JSONType[map[string]string] `gorm:"type:jsonb;comment:队列内资源限制"`
}

func (QueueQuotaLimit) TableName() string {
	return "queue_quotas"
}
