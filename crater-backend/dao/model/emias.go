package model

import (
	"encoding/json"
	"time"

	"github.com/raids-lab/crater/pkg/logutils"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
)

// TaskType
const (
	EmiasTrainingTask   = "training"
	EmiasJupyterTask    = "jupyter"
	EmiasDebuggingTask  = "debugging"
	EmiasInferenceTask  = "inference"
	EmiasExperimentTask = "experiment"

	EmiasHighSLO = 1
	EmiasLowSLO  = 0

	EmiasPriorityClassGauranteed = "gpu-guaranteed"
	EmiasPriorityClassBestEffort = "gpu-besteffort"

	// TaskStatus
	EmiasTaskQueueingStatus  = "Queueing" // 用户队列里的状态
	EmiasTaskCreatedStatus   = "Created"  // AIJob Created
	EmiasTaskPendingStatus   = "Pending"  // AIJob排队的状态
	EmiasTaskRunningStatus   = "Running"
	EmiasTaskFailedStatus    = "Failed"
	EmiasTaskSucceededStatus = "Succeeded"
	EmiasTaskPreemptedStatus = "Preempted"
	EmiasTaskFreedStatus     = "Freed"

	// ProfilingStatus
	EmiasUnProfiled     = 0
	EmiasProfileQueued  = 1
	EmiasProfiling      = 2
	EmiasProfileFinish  = 3
	EmiasProfileFailed  = 4
	EmiasProfileSkipped = 5
)

var (
	EmiasTaskOcupiedQuotaStatuses = []string{EmiasTaskCreatedStatus, EmiasTaskPendingStatus, EmiasTaskRunningStatus, EmiasTaskPreemptedStatus}
	EmiasTaskQueueingStatuses     = []string{EmiasTaskQueueingStatus}
)

type AITask struct {
	gorm.Model

	TaskName  string `gorm:"column:task_name;type:varchar(128);not null" json:"taskName"`
	UserName  string `gorm:"column:username;type:varchar(128);not null" json:"userName"`
	Namespace string `gorm:"column:namespace;type:varchar(128);not null" json:"nameSpace"`
	TaskType  string `gorm:"column:task_type;type:varchar(128);not null" json:"taskType"`

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
	Token           string     `gorm:"column:token;type:varchar(128)" json:"token"`
	NodePort        int32      `gorm:"column:node_port;type:int" json:"nodePort"`

	PodTemplate datatypes.JSONType[v1.PodSpec] `json:"podTemplate"`
	Node        string                         `json:"node"`
	Owner       string                         `json:"owner"`
}

type TaskAttr struct {
	TaskName        string                `json:"taskName" binding:"required"`
	UserName        string                // `json:"userName" binding:"required"`
	SLO             uint                  `json:"slo"`
	TaskType        string                `json:"taskType" binding:"required"`
	GPUModel        string                `json:"gpuModel"`
	SchedulerName   string                `json:"schedulerName"`
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

func FormatTaskAttrToModel(task *TaskAttr) *AITask {
	aiTask := &AITask{
		TaskName:        task.TaskName,
		UserName:        task.UserName,
		Namespace:       task.Namespace,
		TaskType:        task.TaskType,
		Image:           task.Image,
		ResourceRequest: ResourceListToJSON(task.ResourceRequest),
		WorkingDir:      task.WorkingDir,
		ShareDirs:       VolumesToJSONString(task.ShareDirs),
		Command:         task.Command,
		Args:            MapToJSONString(task.Args),
		SLO:             task.SLO,
		Status:          EmiasTaskQueueingStatus,
		EsitmatedTime:   task.EsitmatedTime,
	}

	// 单独设置 gorm.Model 嵌入字段
	aiTask.ID = task.ID

	return aiTask
}

// TODO: directly return AITask
func FormatAITaskToAttr(model *AITask) *TaskAttr {
	resourceJSON, _ := JSONToResourceList(model.ResourceRequest)
	return &TaskAttr{
		ID:              model.ID,
		TaskName:        model.TaskName,
		UserName:        model.UserName,
		Namespace:       model.Namespace,
		TaskType:        model.TaskType,
		Image:           model.Image,
		ResourceRequest: resourceJSON,
		WorkingDir:      model.WorkingDir,
		ShareDirs:       JSONStringToVolumes(model.ShareDirs),
		Command:         model.Command,
		Args:            JSONStringToMap(model.Args),
		SLO:             model.SLO,
		Status:          model.Status,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

type DirMount struct {
	Volume    string `json:"volume"`
	MountPath string `json:"mountPath"`
	SubPath   string `json:"subPath"`
}

func JSONStringToVolumes(str string) map[string][]DirMount {
	var volumes map[string][]DirMount
	err := json.Unmarshal([]byte(str), &volumes)
	if err != nil {
		logutils.Log.Errorf("JSONStringToVolumes error: %v", err)
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
