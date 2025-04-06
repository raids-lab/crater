package model

import (
	"time"

	"github.com/raids-lab/crater/pkg/monitor"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

const (
	Deleted batch.JobPhase = "Deleted"
	Freed   batch.JobPhase = "Freed"
)

type JobType string

const (
	JobTypeAll        JobType = "all"
	JobTypeJupyter    JobType = "jupyter"
	JobTypeWebIDE     JobType = "webide"
	JobTypePytorch    JobType = "pytorch"
	JobTypeTensorflow JobType = "tensorflow"
	JobTypeKubeRay    JobType = "kuberay"
	JobTypeDeepSpeed  JobType = "deepspeed"
	JobTypeOpenMPI    JobType = "openmpi"
	JobTypeCustom     JobType = "custom"
)

type Job struct {
	gorm.Model
	Name               string                              `gorm:"not null;type:varchar(256);comment:作业名称"`
	JobName            string                              `gorm:"uniqueIndex;type:varchar(256);not null;comment:作业名称"`
	UserID             uint                                `gorm:"primaryKey"`
	User               User                                `gorm:"foreignKey:UserID"`
	AccountID          uint                                `gorm:"primaryKey"`
	Account            Account                             `gorm:"foreignKey:AccountID"`
	JobType            JobType                             `gorm:"not null;comment:作业类型"`
	Status             batch.JobPhase                      `gorm:"index:status;not null;comment:作业状态"`
	CreationTimestamp  time.Time                           `gorm:"not null;comment:作业创建时间"`
	RunningTimestamp   time.Time                           `gorm:"comment:作业开始运行时间"`
	CompletedTimestamp time.Time                           `gorm:"comment:作业完成时间"`
	LockedTimestamp    time.Time                           `gorm:"comment:作业锁定时间"`
	Nodes              datatypes.JSONType[[]string]        `gorm:"comment:作业运行的节点"`
	Resources          datatypes.JSONType[v1.ResourceList] `gorm:"comment:作业的资源需求"`

	KeepWhenLowResourceUsage bool `gorm:"comment:当资源利用率低时是否保留"`
	Reminded                 bool `gorm:"comment:是否已经处于发送了提醒的状态"`

	Attributes   datatypes.JSONType[*batch.Job]            `gorm:"comment:作业的原始属性"`
	ProfileData  *datatypes.JSONType[*monitor.ProfileData] `gorm:"comment:作业的性能数据"`
	Template     string                                    `gorm:"type:text;comment:作业的模板配置"`
	AlertEnabled bool                                      `gorm:"type:boolean;default:false;comment:是否启用通知"`
}
