package models

import (
	"strings"
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

	RunningStatus   = "Running"
	PendingStatus   = "Pending"
	FailedStatus    = "Failed"
	SucceededStatus = "Succeeded"
	SuspendedStatus = "Suspended"
)

// TaskModel is task presented in db
type TaskModel struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TaskName        string    `gorm:"column:task_name;type:varchar(128);not null" json:"taskName"`
	UserName        string    `gorm:"column:user_name;type:varchar(128);not null" json:"userName"`
	Namespace       string    `gorm:"column:namespace;type:varchar(128);not null" json:"namespace"`
	TaskType        string    `gorm:"column:task_type;type:varchar(128);not null" json:"taskType"`
	CreatedAt       time.Time `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null" json:"updatedAt"`
	Image           string    `gorm:"column:image;type:text;not null" json:"image"`
	ResourceRequest string    `gorm:"column:resource_request;type:text;not null" json:"resourceRequest"`
	WorkingDir      string    `gorm:"column:working_dir;type:text" json:"workingDir"`
	Command         string    `gorm:"column:command;type:text" json:"command"`
	Args            string    `gorm:"column:args;type:text" json:"args"`
	SLO             uint      `gorm:"column:slo;type:int;not null" json:"slo"`
	Status          string    `gorm:"column:status;type:varchar(128)" json:"status"`
	IsDeleted       bool      `gorm:"column:is_deleted;type:bool" json:"isDeleted"`

	// PreTaskID       uint
	// PostTaskID      uint
}

// TaskAttr request
type TaskAttr struct {
	TaskName        string            `json:"taskName" binding:"required"`
	UserName        string            `json:"userName" binding:"required"`
	SLO             uint              `json:"slo" binding:"required"`
	TaskType        string            `json:"taskType" binding:"required"`
	Image           string            `json:"image" binding:"required"`
	ResourceRequest v1.ResourceList   `json:"resourceRequest" binding:"required"`
	Command         string            `json:"command" binding:"required"`
	Args            map[string]string `json:"args"`
	WorkingDir      string            `json:"workingDir"`
	// not for request
	ID        uint `json:"id"`
	Namespace string
	Status    string
}

func FormatTaskAttrToModel(task *TaskAttr) *TaskModel {
	return &TaskModel{
		TaskName:        task.TaskName,
		UserName:        task.UserName,
		TaskType:        task.TaskType,
		Image:           task.Image,
		ResourceRequest: ResourceListToJSON(task.ResourceRequest),
		WorkingDir:      task.WorkingDir,
		Command:         task.Command,
		Args:            argsToString(task.Args),
		SLO:             task.SLO,
	}
}

func FormatTaskModelToAttr(model *TaskModel) *TaskAttr {
	resourceJson, _ := JSONToResourceList(model.ResourceRequest)
	return &TaskAttr{
		ID:              model.ID,
		TaskName:        model.TaskName,
		UserName:        model.UserName,
		Namespace:       model.Namespace,
		TaskType:        model.TaskType,
		Image:           model.Image,
		ResourceRequest: resourceJson,
		WorkingDir:      model.WorkingDir,
		Command:         model.Command,
		Args:            dbstringToArgs(model.Args),
		SLO:             model.SLO,
		Status:          model.Status,
	}
}

func argsToString(args map[string]string) string {
	str := ""
	for key, value := range args {
		str += key + "=" + value + " "
	}
	return str
}

func dbstringToArgs(str string) map[string]string {
	args := map[string]string{}
	if len(str) == 0 {
		return args
	}
	ls := strings.Split(str, " ")
	for item := range ls {
		kv := strings.Split(ls[item], "=")
		args[kv[0]] = kv[1]
	}
	return args
}
