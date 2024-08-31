package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

const Deleted batch.JobPhase = "deleted"

// User is the basic entity of the system
type Job struct {
	gorm.Model
	Name               string                              `gorm:"not null;comment:作业名称"`
	JobName            string                              `gorm:"index:job_name;not null;comment:作业名称"`
	UserID             uint                                `gorm:"primaryKey"`
	User               User                                `gorm:"foreignKey:UserID"`
	QueueID            uint                                `gorm:"primaryKey"`
	Queue              Queue                               `gorm:"foreignKey:QueueID"`
	JobType            string                              `gorm:"not null;comment:作业类型"`
	Status             batch.JobPhase                      `gorm:"index:status;not null;comment:用户状态 (pending, active, inactive)"`
	CreationTimestamp  time.Time                           `gorm:"not null;comment:作业创建时间"`
	RunningTimestamp   time.Time                           `gorm:"comment:作业开始运行时间"`
	CompletedTimestamp time.Time                           `gorm:"comment:作业完成时间"`
	Nodes              datatypes.JSONType[[]string]        `gorm:"comment:作业运行的节点"`
	Resources          datatypes.JSONType[v1.ResourceList] `gorm:"comment:作业的资源需求"`

	Attributes datatypes.JSONType[*batch.Job] `gorm:"comment:作业的原始属性"`
}
