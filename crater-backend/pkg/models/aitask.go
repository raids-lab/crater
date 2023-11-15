package models

import (
	"time"

	v1 "k8s.io/api/core/v1"
)

// TaskType
const (
	TrainingTask   = "training"
	DebuggingTask  = "debugging"
	InferenceTask  = "inference"
	ExperimentTask = "experiment"

	HighSLO = 1
	LowSLO  = 0

	// TaskStatus
	QueueingStatus  = "Queueing" // 用户队列里的状态
	PendingStatus   = "Pending"  // AIJob排队的状态
	RunningStatus   = "Running"
	FailedStatus    = "Failed"
	SucceededStatus = "Succeeded"
	SuspendedStatus = "Suspended"

	// ProfilingStatus
	UnProfiled    = 0
	ProfileQueued = 1
	Profiling     = 2
	ProfileFinish = 3
)

// TaskModel is task presented in db
type AITask struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	TaskName        string     `gorm:"column:task_name;type:varchar(128);not null" json:"taskName"`
	UserName        string     `gorm:"column:username;type:varchar(128);not null" json:"userName"`
	Namespace       string     `gorm:"column:namespace;type:varchar(128);not null" json:"namespace"`
	TaskType        string     `gorm:"column:task_type;type:varchar(128);not null" json:"taskType"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;not null" json:"updatedAt"`
	AdmittedAt      *time.Time `gorm:"column:admitted_at" json:"admittedAt"`
	StartedAt       *time.Time `gorm:"column:started_at" json:"startedAt"`
	Image           string     `gorm:"column:image;type:text;not null" json:"image"`
	ResourceRequest string     `gorm:"column:resource_request;type:text;not null" json:"resourceRequest"`
	WorkingDir      string     `gorm:"column:working_dir;type:text" json:"workingDir"`
	ShareDirs       string     `gorm:"column:share_dirs;type:text" json:"shareDirs"`
	Command         string     `gorm:"column:command;type:text" json:"command"`
	Args            string     `gorm:"column:args;type:text" json:"args"`
	SLO             uint       `gorm:"column:slo;type:int;not null" json:"slo"`
	Status          string     `gorm:"column:status;type:varchar(128)" json:"status"`
	IsDeleted       bool       `gorm:"column:is_deleted;type:bool" json:"isDeleted"`
	ProfileStatus   uint       `gorm:"column:profile_status;type:int" json:"profileStatus"`
	ProfileStat     string     `gorm:"column:profile_stat;type:text" json:"profileStat"`
	EsitmatedTime   uint       `gorm:"column:estimated_time;type:int" json:"estimatedTime"`
	ScheduleInfo    string     `gorm:"column:schedule_info;type:text" json:"scheduleInfo"`
}

// TaskAttr request
type TaskAttr struct {
	TaskName        string            `json:"taskName" binding:"required"`
	UserName        string            //`json:"userName" binding:"required"`
	SLO             uint              `json:"slo"`
	TaskType        string            `json:"taskType" binding:"required"`
	Image           string            `json:"image" binding:"required"`
	ResourceRequest v1.ResourceList   `json:"resourceRequest" binding:"required"`
	Command         string            `json:"command" binding:"required"`
	Args            map[string]string `json:"args"`
	WorkingDir      string            `json:"workingDir"`
	ShareDirs       map[string]string `json:"shareDirs"`
	// not for request
	ID        uint `json:"id"`
	Namespace string
	Status    string
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null" json:"updatedAt"`
	StartedAt time.Time `gorm:"column:started_at" json:"startedAt"`
}
