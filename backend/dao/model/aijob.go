package model

import (
	"time"

	"gorm.io/gorm"
)

type AIJob struct {
	gorm.Model
	Name            string `gorm:"type:varchar(128);not null;comment:作业名称" json:"name"`
	UserID          uint
	User            User
	ProjectID       uint
	Project         Project
	TaskType        string     `gorm:"type:varchar(128);not null;comment:任务类型 (e.g. training, jupyter, recommend)" json:"taskType"`
	AdmittedAt      *time.Time `gorm:"comment:作业提交到集群中的时间 (通过配额检查后的时间)" json:"admittedAt"`
	StartedAt       *time.Time `gorm:"comment:作业开始运行的时间" json:"startedAt"`
	FinishAt        *time.Time `gorm:"comment:作业结束 (成功或失败) 的时间" json:"finishAt"`
	Status          JobStatus  `gorm:"not null;comment:作业状态" json:"status"`
	ResourceRequest string     `gorm:"not null;comment:资源请求 (分布式任务则从 Extra 中解析)" json:"resourceRequest"`
	Extra           *string    `gorm:"comment:额外信息 (JSON格式)" json:"extra"`
	// Duration        uint        `gorm:"column:duration;type:int;not null" json:"duration"`
	// JCT             uint        `gorm:"column:jct;type:int;not null" json:"jct"`
	// Image           string      `gorm:"column:image;type:text;not null" json:"image"`
	// ResourceRequest string      `gorm:"column:resource_request;type:text;not null" json:"resourceRequest"`
	// WorkingDir      string      `gorm:"column:working_dir;type:text" json:"workingDir"`
	// ShareDirs       string      `gorm:"column:share_dirs;type:text" json:"shareDirs"`
	// Command         string      `gorm:"column:command;type:text" json:"command"`
	// Args            string      `gorm:"column:args;type:text" json:"args"`
	// SLO             uint        `gorm:"column:slo;type:int;not null" json:"slo"`
	// StatusReason    string      `gorm:"column:status_reason;type:text" json:"statusReason"`
	// JobName         string      `gorm:"column:jobname;type:varchar(128)" json:"jobName"`
	// IsDeleted       bool        `gorm:"column:is_deleted;type:bool" json:"isDeleted"`
	// ProfileStatus   uint        `gorm:"column:profile_status;type:int" json:"profileStatus"`
	// ProfileStat     string      `gorm:"column:profile_stat;type:text" json:"profileStat"`
	// EsitmatedTime   uint        `gorm:"column:estimated_time;type:int" json:"estimatedTime"`
	// ScheduleInfo    string      `gorm:"column:schedule_info;type:text" json:"scheduleInfo"`
	// Token           string      `gorm:"column:token;type:varchar(128)" json:"token"`
	// NodePort        int32       `gorm:"column:node_port;type:int" json:"nodePort"`
	// SchedulerName   string      `gorm:"column:scheduler_name;type:varchar(128)" json:"schedulerName"`
	// GPUModel        string      `gorm:"column:gpu_model;type:varchar(128)" json:"gpuModel"`
}
