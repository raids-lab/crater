package payload

import "k8s.io/ai-task-controller/pkg/models"

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
