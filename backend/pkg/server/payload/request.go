package payload

import (
	"github.com/aisystem/ai-protal/pkg/models"
	v1 "k8s.io/api/core/v1"
)

type CreateTaskReq struct {
	models.TaskAttr
}

// ListTaskReq is the request payload for listing tasks. Get Method
type ListTaskReq struct {
	Status   string `form:"status"`
}

type GetTaskReq struct {
	TaskID uint `form:"taskID" binding:"required"`
}

type DeleteTaskReq struct {
	TaskID uint `json:"taskID" binding:"required"`
}

type UpdateTaskSLOReq struct {
	TaskID uint `json:"taskID" binding:"required"`
	SLO    uint `json:"slo" binding:"required"` // change the slo of the task
}

// TODO: update task sequence

type CreateOrUpdateQuotaReq struct {
	UserName  string          `json:"userName" binding:"required"`
	HardQuota v1.ResourceList `json:"hardQuota" binding:"required"`
}

type ListQuotaReq struct {
}

type GetQuotaReq struct {
	UserName string `form:"userName" binding:"required"`
}
