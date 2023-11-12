package payload

import "github.com/aisystem/ai-protal/pkg/models"

type CreateTaskResp struct {
}

type ListTaskResp struct {
	Tasks []models.TaskAttr
}

type GetTaskResp struct {
	models.TaskAttr
}

type DeleteTaskResp struct {
}

type UpdateTaskResp struct {
}


type ListQuotaResp struct {
	Quotas []models.Quota
}

type GetQuotaResp struct {
	Quota models.Quota
}
