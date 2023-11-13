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
	Status string `form:"status"`
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

type UpdateQuotaReq struct {
	UserName  string          `json:"userName" binding:"required"`
	HardQuota v1.ResourceList `json:"hardQuota" binding:"required"`
}

type UpdateRoleReq struct {
	UserName string `json:"userName" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

type DeleteUserReq struct {
	UserName       string `json:"userName" binding:"required"`
	DeleteResource bool   `json:"deleteResource"`
}

type GetUserReq struct {
	UserName string `json:"userName"`
	UserID   uint   `json:"userID"`
}

type CreateUserRequest struct {
	//Id       int    `json:"_id"`
	Name     string `json:"userName" binding:"required"`
	Role     string `json:"role" binding:"required"`
	Password string `json:"password" binding:"required"`
}
