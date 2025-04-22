package aitaskctl

import (
	"context"
	"time"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

type DBService interface {
	Create(task *model.AITask) error
	Update(task *model.AITask) error
	UpdateStatus(taskID uint, status string, reason string) error
	UpdateJobName(taskID uint, jobname string) error
	DeleteByID(taskID uint) error
	DeleteByQueueAndID(userName string, taskID uint) error
	ForceDeleteByUserAndID(userName string, taskID uint) error
	ListByTaskType(taskType string, page, pageSize int) ([]*model.AITask, int64, error)
	ListByUserAndStatuses(userName string, status []string) ([]*model.AITask, error)
	ListByQueue(queue string) ([]*model.AITask, error)
	ListAll() ([]*model.AITask, error)
	GetByID(taskID uint) (*model.AITask, error)
	GetByQueueAndID(userName string, taskID uint) (*model.AITask, error)
	GetByJobName(jobName string) (*model.AITask, error)
	UpdateProfilingStat(taskID uint, profileStatus uint, stat string, status string) error
	UpdateToken(taskID uint, token string) error
	UpdateNodePort(taskID uint, nodePort int32) error
	GetTaskStatusCount() ([]model.TaskStatusCount, error)
	GetUserTaskStatusCount(userName string) ([]model.TaskStatusCount, error)
}

type TaskStatusCount = model.TaskStatusCount

type service struct{}

func NewDBService() DBService {
	return &service{}
}

func (s *service) Create(task *model.AITask) error {
	return query.AITask.WithContext(context.Background()).Create(task)
}

func (s *service) Update(task *model.AITask) error {
	return query.AITask.WithContext(context.Background()).Save(task)
}

func (s *service) UpdateStatus(taskID uint, status, reason string) error {
	ctx := context.Background()
	task, err := s.GetByID(taskID)
	if err != nil {
		return err
	}

	if task.Status == status {
		return nil
	}

	q := query.AITask
	updateMap := make(map[string]any)
	updateMap["status"] = status
	updateMap["status_reason"] = reason

	t := time.Now()
	switch status {
	case "Created":
		updateMap["admitted_at"] = &t
	case "Running":
		updateMap["started_at"] = &t
	case "Succeeded", "Failed":
		updateMap["finish_at"] = &t
		if task.StartedAt != nil {
			updateMap["duration"] = uint(t.Sub(*task.StartedAt).Seconds())
		}
		updateMap["jct"] = uint(t.Sub(task.CreatedAt).Seconds())
	}

	_, err = q.WithContext(ctx).Where(q.ID.Eq(taskID)).Updates(updateMap)
	return err
}

func (s *service) UpdateJobName(taskID uint, jobname string) error {
	q := query.AITask
	_, err := q.WithContext(context.Background()).Where(q.ID.Eq(taskID)).Update(q.JobName, jobname)
	return err
}

func (s *service) DeleteByQueueAndID(queue string, taskID uint) error {
	q := query.AITask
	_, err := q.WithContext(context.Background()).Where(q.UserName.Eq(queue), q.ID.Eq(taskID)).Update(q.IsDeleted, true)
	return err
}

func (s *service) ForceDeleteByUserAndID(userName string, taskID uint) error {
	q := query.AITask
	_, err := q.WithContext(context.Background()).Where(q.UserName.Eq(userName), q.ID.Eq(taskID)).Delete()
	return err
}

func (s *service) DeleteByID(taskID uint) error {
	q := query.AITask
	_, err := q.WithContext(context.Background()).Where(q.ID.Eq(taskID)).Delete()
	return err
}

func (s *service) ListByUserAndStatuses(userName string, statuses []string) ([]*model.AITask, error) {
	q := query.AITask
	ctx := context.Background()

	if len(statuses) == 0 {
		return q.WithContext(ctx).Where(q.UserName.Eq(userName)).Find()
	}

	// 修复: 使用正确的 In 方法接收字符串切片
	return q.WithContext(ctx).Where(q.UserName.Eq(userName), q.Status.In(statuses...), q.IsDeleted.Is(false)).Find()
}

func (s *service) ListByTaskType(taskType string, page, pageSize int) ([]*model.AITask, int64, error) {
	q := query.AITask
	ctx := context.Background()

	// 计算总数
	count, err := q.WithContext(ctx).Where(q.TaskType.Eq(taskType)).Count()
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	qu := q.WithContext(ctx).Where(q.TaskType.Eq(taskType)).Order(q.CreatedAt.Desc())
	if pageSize > 0 {
		qu = qu.Offset(page * pageSize).Limit(pageSize)
	}

	tasks, err := qu.Find()
	return tasks, count, err
}

func (s *service) ListByQueue(queue string) ([]*model.AITask, error) {
	q := query.AITask
	return q.WithContext(context.Background()).Where(q.UserName.Eq(queue)).Order(q.CreatedAt.Desc()).Find()
}

func (s *service) ListAll() ([]*model.AITask, error) {
	q := query.AITask
	return q.WithContext(context.Background()).Order(q.CreatedAt.Desc()).Find()
}

func (s *service) GetByID(taskID uint) (*model.AITask, error) {
	q := query.AITask
	return q.WithContext(context.Background()).Where(q.ID.Eq(taskID)).First()
}

func (s *service) GetByJobName(jobName string) (*model.AITask, error) {
	q := query.AITask
	return q.WithContext(context.Background()).Where(q.JobName.Eq(jobName)).First()
}

func (s *service) GetByQueueAndID(userName string, taskID uint) (*model.AITask, error) {
	q := query.AITask
	return q.WithContext(context.Background()).Where(q.UserName.Eq(userName), q.ID.Eq(taskID)).First()
}

func (s *service) UpdateProfilingStat(taskID, profileStatus uint, stat, status string) error {
	q := query.AITask
	ctx := context.Background()

	updateMap := make(map[string]any)
	updateMap["profile_status"] = profileStatus

	if profileStatus == 3 { // ProfileFinish
		updateMap["profile_stat"] = stat
		if status != "" {
			updateMap["status"] = status
		}
	}

	_, err := q.WithContext(ctx).Where(q.ID.Eq(taskID)).Updates(updateMap)
	return err
}

func (s *service) UpdateToken(taskID uint, token string) error {
	q := query.AITask
	_, err := q.WithContext(context.Background()).Where(q.ID.Eq(taskID)).Update(q.Token, token)
	return err
}

func (s *service) UpdateNodePort(taskID uint, nodePort int32) error {
	q := query.AITask
	_, err := q.WithContext(context.Background()).Where(q.ID.Eq(taskID)).Update(q.NodePort, nodePort)
	return err
}

type TaskStatusCountResult struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

func (s *service) GetTaskStatusCount() ([]model.TaskStatusCount, error) {
	q := query.AITask
	ctx := context.Background()

	var results []TaskStatusCountResult
	err := q.WithContext(ctx).Where(q.IsDeleted.Is(false)).
		Select(q.Status.As("status"), q.Status.Count().As("count")).
		Group(q.Status).
		Scan(&results)

	if err != nil {
		return nil, err
	}

	// 转换结果格式
	stats := make([]model.TaskStatusCount, len(results))
	for i, r := range results {
		stats[i] = model.TaskStatusCount{
			Status: r.Status,
			Count:  int(r.Count),
		}
	}

	return stats, nil
}

func (s *service) GetUserTaskStatusCount(userName string) ([]model.TaskStatusCount, error) {
	q := query.AITask
	ctx := context.Background()

	var results []TaskStatusCountResult
	err := q.WithContext(ctx).Where(q.UserName.Eq(userName), q.IsDeleted.Is(false)).
		Select(q.Status.As("status"), q.Status.Count().As("count")).
		Group(q.Status).
		Scan(&results)

	if err != nil {
		return nil, err
	}

	// 转换结果格式
	stats := make([]model.TaskStatusCount, len(results))
	for i, r := range results {
		stats[i] = model.TaskStatusCount{
			Status: r.Status,
			Count:  int(r.Count),
		}
	}

	return stats, nil
}
