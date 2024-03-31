package model

import (
	"gorm.io/gorm"
)

type AIJob struct {
	gorm.Model
	Name      string `gorm:"type:varchar(128);not null;comment:作业名称"`
	UserID    uint
	User      User
	ProjectID uint
	Project   Project
	TaskType  string `gorm:"type:varchar(128);not null"`
	// AdmittedAt      *time.Time  `gorm:"column:admitted_at" json:"admittedAt"`
	// StartedAt       *time.Time  `gorm:"column:started_at" json:"startedAt"`
	// FinishAt        *time.Time  `gorm:"column:finish_at" json:"finishAt"`
	// Duration        uint        `gorm:"column:duration;type:int;not null" json:"duration"`
	// JCT             uint        `gorm:"column:jct;type:int;not null" json:"jct"`
	// Image           string      `gorm:"column:image;type:text;not null" json:"image"`
	// ResourceRequest string      `gorm:"column:resource_request;type:text;not null" json:"resourceRequest"`
	// WorkingDir      string      `gorm:"column:working_dir;type:text" json:"workingDir"`
	// ShareDirs       string      `gorm:"column:share_dirs;type:text" json:"shareDirs"`
	// Command         string      `gorm:"column:command;type:text" json:"command"`
	// Args            string      `gorm:"column:args;type:text" json:"args"`
	// SLO             uint        `gorm:"column:slo;type:int;not null" json:"slo"`
	// Status          string      `gorm:"column:status;type:varchar(128)" json:"status"`
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
