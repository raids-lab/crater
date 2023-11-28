package models

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
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

	PriorityClassGauranteed = "gpu-gauranteed"
	PriorityClassBestEffort = "gpu-besteffort"

	// TaskStatus
	TaskQueueingStatus  = "Queueing" // 用户队列里的状态
	TaskCreatedStatus   = "Created"  // AIJob Created
	TaskPendingStatus   = "Pending"  // AIJob排队的状态
	TaskRunningStatus   = "Running"
	TaskFailedStatus    = "Failed"
	TaskSucceededStatus = "Succeeded"
	TaskPreemptedStatus = "Preempted"

	// ProfilingStatus
	UnProfiled    = 0
	ProfileQueued = 1
	Profiling     = 2
	ProfileFinish = 3
	ProfileFailed = 4
)

var (
	TaskOcupiedQuotaStatuses = []string{TaskCreatedStatus, TaskPendingStatus, TaskRunningStatus, TaskPreemptedStatus}
	TaskQueueingStatuses     = []string{TaskQueueingStatus}
)

// TaskModel is task presented in db
type AITask struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	TaskName        string     `gorm:"column:task_name;type:varchar(128);not null" json:"taskName"`
	UserName        string     `gorm:"column:username;type:varchar(128);not null" json:"userName"`
	Namespace       string     `gorm:"column:namespace;type:varchar(128);not null" json:"nameSpace"`
	TaskType        string     `gorm:"column:task_type;type:varchar(128);not null" json:"taskType"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;not null" json:"updatedAt"`
	AdmittedAt      *time.Time `gorm:"column:admitted_at" json:"admittedAt"`
	StartedAt       *time.Time `gorm:"column:started_at" json:"startedAt"`
	FinishAt        *time.Time `gorm:"column:finish_at" json:"finishAt"`
	Duration        uint       `gorm:"column:duration;type:int;not null" json:"duration"`
	JCT             uint       `gorm:"column:jct;type:int;not null" json:"jct"`
	Image           string     `gorm:"column:image;type:text;not null" json:"image"`
	ResourceRequest string     `gorm:"column:resource_request;type:text;not null" json:"resourceRequest"`
	WorkingDir      string     `gorm:"column:working_dir;type:text" json:"workingDir"`
	ShareDirs       string     `gorm:"column:share_dirs;type:text" json:"shareDirs"`
	Command         string     `gorm:"column:command;type:text" json:"command"`
	Args            string     `gorm:"column:args;type:text" json:"args"`
	SLO             uint       `gorm:"column:slo;type:int;not null" json:"slo"`
	Status          string     `gorm:"column:status;type:varchar(128)" json:"status"`
	StatusReason    string     `gorm:"column:status_reason;type:text" json:"statusReason"`
	JobName         string     `gorm:"column:jobname;type:varchar(128)" json:"jobName"`
	IsDeleted       bool       `gorm:"column:is_deleted;type:bool" json:"isDeleted"`
	ProfileStatus   uint       `gorm:"column:profile_status;type:int" json:"profileStatus"`
	ProfileStat     string     `gorm:"column:profile_stat;type:text" json:"profileStat"`
	EsitmatedTime   uint       `gorm:"column:estimated_time;type:int" json:"estimatedTime"`
	ScheduleInfo    string     `gorm:"column:schedule_info;type:text" json:"scheduleInfo"`
}

// TaskAttr request
type TaskAttr struct {
	TaskName        string                `json:"taskName" binding:"required"`
	UserName        string                //`json:"userName" binding:"required"`
	SLO             uint                  `json:"slo"`
	TaskType        string                `json:"taskType" binding:"required"`
	Image           string                `json:"image" binding:"required"`
	ResourceRequest v1.ResourceList       `json:"resourceRequest" binding:"required"`
	Command         string                `json:"command" binding:"required"`
	Args            map[string]string     `json:"args"`
	WorkingDir      string                `json:"workingDir"`
	ShareDirs       map[string][]DirMount `json:"shareDirs"`
	EsitmatedTime   uint                  `json:"estimatedTime"`
	// not for request
	ID        uint      `json:"id"`
	Namespace string    `json:"namspace"`
	Status    string    `json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null" json:"updatedAt"`
	StartedAt time.Time `gorm:"column:started_at" json:"startedAt"`
}

// type Volume struct {
// 	Name string `json:"name"`
// 	Mounts []DirMount `json:"mounts"`
// }
type DirMount struct {
	Volume    string `json:"volume"`
	MountPath string `json:"mountPath"`
	SubPath   string `json:"subPath"`
}

func JSONStringToVolumes(str string) map[string][]DirMount {
	var volumes map[string][]DirMount
	err := json.Unmarshal([]byte(str), &volumes)
	if err != nil {
		logrus.Errorf("JSONStringToVolumes error: %v", err)
		return nil
	}
	return volumes
}

func VolumesToJSONString(volumes map[string][]DirMount) string {
	bytes, err := json.Marshal(volumes)
	if err != nil {
		return ""
	}
	return string(bytes)
}
