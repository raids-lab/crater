package model

import (
	"fmt"
	"regexp"
	"strings"
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

type ScheduleData struct {
	ImagePullTime string  `json:"imagePullTime"`
	ImageSize     *string `json:"imageSize"`
}

// 从事件中获取镜像拉取数据，重点关注 Pod Pulled 事件
func (s *ScheduleData) Init(msg string) error {
	if strings.Contains(msg, "already present on machine") {
		s.ImagePullTime = "0s"
		return nil
	}
	if !strings.Contains(msg, "Successfully pulled image") {
		return fmt.Errorf("not a image pull event")
	}
	// 使用正则表达式获取(2m46.503s including waiting) 括号内的内容，正则表达式匹配 (.* including waiting)
	regex := `\((.*?) including waiting\)`
	re := regexp.MustCompile(regex)
	matches := re.FindStringSubmatch(msg)
	if len(matches) > 1 {
		s.ImagePullTime = matches[1]
	}

	// 使用正则表达式获取 Image size: 8341280688 bytes 内的数字
	regex = `Image size: (\d+) bytes`
	re = regexp.MustCompile(regex)
	matches = re.FindStringSubmatch(msg)
	if len(matches) > 1 {
		imageSize := matches[1]
		s.ImageSize = &imageSize
	}
	return nil
}

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
	Nodes              datatypes.JSONType[[]string]        `gorm:"comment:作业运行的节点"`
	Resources          datatypes.JSONType[v1.ResourceList] `gorm:"comment:作业的资源需求"`
	Attributes         datatypes.JSONType[*batch.Job]      `gorm:"comment:作业的原始属性"`
	Template           string                              `gorm:"type:text;comment:作业的模板配置"`

	// 通知相关
	AlertEnabled bool `gorm:"type:boolean;default:false;comment:是否启用通知"`
	Reminded     bool `gorm:"comment:是否已经处于发送了提醒的状态"`

	// 定时策略相关
	KeepWhenLowResourceUsage bool      `gorm:"comment:当资源利用率低时是否保留"`
	LockedTimestamp          time.Time `gorm:"comment:作业锁定时间"`

	// 诊断数据收集
	ProfileData      *datatypes.JSONType[*monitor.ProfileData]          `gorm:"comment:作业的性能数据"`
	ScheduleData     *datatypes.JSONType[*ScheduleData]                 `gorm:"comment:作业的调度数据"`
	Events           *datatypes.JSONType[[]v1.Event]                    `gorm:"comment:作业的事件 (运行时、失败时采集)"`
	TerminatedStates *datatypes.JSONType[[]v1.ContainerStateTerminated] `gorm:"comment:作业的终止状态 (运行时、失败时采集)"`
}
