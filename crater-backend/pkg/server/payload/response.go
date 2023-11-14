package payload

import (
	"time"

	"github.com/aisystem/ai-protal/pkg/models"
	v1 "k8s.io/api/core/v1"
)

type CreateTaskResp struct {
}

type ListTaskResp struct {
	Tasks []models.AITask
}

type GetTaskResp struct {
	models.AITask
}

type DeleteTaskResp struct {
}

type UpdateTaskResp struct {
}

type ListUserResp struct {
	Users []GetUserResp
}

type GetQuotaResp struct {
	Hard     v1.ResourceList `json:"hard"`
	HardUsed v1.ResourceList `json:"hardUsed"`
	SoftUsed v1.ResourceList `json:"softUsed"`
}

type GetUserResp struct {
	UserID    uint            `json:"userID"`
	UserName  string          `json:"userName"`
	Role      string          `json:"role"`
	QuotaHard v1.ResourceList `json:"quotaHard"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}
