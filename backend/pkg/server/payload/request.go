package payload

import (
	"fmt"

	"github.com/raids-lab/crater/pkg/models"
	v1 "k8s.io/api/core/v1"
)

type CreateTaskReq struct {
	models.TaskAttr
}

type CreateJupyterReq struct {
	models.JupyterTaskAttr
}

// ListTaskReq is the request payload for listing tasks. Get Method
type ListTaskReq struct {
	Status string `form:"status"`
}

type ListTaskByTypeReq struct {
	PageSize  int    `form:"pageSize"`
	PageIndex int    `form:"pageIndex"`
	TaskType  string `form:"taskType"`
}

type GetTaskReq struct {
	TaskID uint `form:"taskID" binding:"required"`
}

type DeleteTaskReq struct {
	TaskID      uint `json:"taskID" binding:"required"`
	ForceDelete bool `json:"forceDelete"`
}

type UpdateTaskSLOReq struct {
	TaskID uint `json:"taskID" binding:"required"`
	SLO    uint `json:"slo"` // change the slo of the task
}

type GetImagesResp struct {
	Images []string `json:"images"`
}

// TODO: update task sequence

type UpdateQuotaReq struct {
	HardQuota v1.ResourceList `json:"hardQuota" binding:"required"`
}

type UpdateRoleReq struct {
	Role string `json:"role" binding:"required"`
}

type DeleteUserReq struct {
	DeleteResource bool `json:"deleteResource"`
}

type GetUserReq struct {
	UserName string `json:"userName"`
	UserID   uint   `json:"userID"`
}

type CreateUserRequest struct {
	// Id       int    `json:"_id"`
	Name     string `json:"userName" binding:"required"`
	Role     string `json:"role" binding:"required"`
	Password string `json:"passWord" binding:"required"`
}

// RecommendDLJob Request

type CreateRecommendDLJobReq struct {
	Name string `json:"name" binding:"required"`
	RecommendDLJobSpec
}

type AnalyzeRecommendDLJobReq struct {
	RecommendDLJobSpec
}

type DataRelationShipReq struct {
	Type    string `json:"type"` // input, output, bothyway
	JobName string `json:"jobName"`
}

func (r *CreateRecommendDLJobReq) PreCheck() error {
	if len(r.Template.Spec.Containers) == 0 {
		return fmt.Errorf("empty containers")
	}
	for i := 0; i < len(r.Template.Spec.Containers); i++ {
		c := r.Template.Spec.Containers[i]
		if c.Name == "" {
			return fmt.Errorf("empty container name")
		}
		if c.Image == "" {
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
