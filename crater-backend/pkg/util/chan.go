package util

type JobStatusChan struct {
	TaskID    string
	NewStatus string
}

type TaskUpdateChan struct {
	TaskID    uint
	UserName  string
	Operation TaskOperation
}

type TaskOperation string

const (
	CreateTask TaskOperation = "create"
	UpdateTask TaskOperation = "update"
	DeleteTask TaskOperation = "delete"
)
