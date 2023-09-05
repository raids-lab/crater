package payload

import "github.com/aisystem/ai-protal/pkg/models"

type CreateTaskReq struct {
	models.TaskAttr
}

// ListTaskReq is the request payload for listing tasks. Get Method
type ListTaskReq struct {
	UserName string `form:"userName" binding:"required"`
	Status   string `form:"status"`
}

type GetTaskReq struct {
	TaskID uint `form:"taskID" binding:"required"`
}

type DeleteTaskReq struct {
	TaskID uint `form:"taskID" binding:"required"`
}

type UpdateTaskSLOReq struct {
	TaskID uint `json:"taskID" binding:"required"`
	SLO    uint `json:"slo" binding:"required"` // change the slo of the task
}

// TODO: update task sequence
