package payload

import (
	"fmt"

	recommenddljobapi "github.com/aisystem/ai-protal/pkg/apis/recommenddljob/v1"
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
	SLO    uint `json:"slo"` // change the slo of the task
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

// RecommendDLJob Request

type CreateRecommendDLJobReq struct {
	Name string `json:"name" binding:"required"`
	recommenddljobapi.RecommendDLJobSpec
}

func (r *CreateRecommendDLJobReq) PreCheck() error {
	if len(r.Template.Spec.Containers) == 0 {
		return fmt.Errorf("empty containers")
	}
	for _, c := range r.Template.Spec.Containers {
		if len(c.Name) == 0 {
			return fmt.Errorf("empty container name")
		}
		if len(c.Image) == 0 {
			return fmt.Errorf("empty image")
		}
	}
	return nil
}

type GetRecommendDLJobReq struct {
	Name string `form:"name" binding:"required"`
}

type GetRecommendDLJobPodListReq struct {
	Name string `form:"name" binding:"required"`
}

type DeleteRecommendDLJobReq struct {
	Name string `form:"name" binding:"required"`
}

// DataSet Request

type GetDataSetReq struct {
	Name string `form:"name" binding:"required"`
}
