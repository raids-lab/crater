package task

import (
	"time"

	db "github.com/raids-lab/crater/pkg/db/orm"
	"github.com/raids-lab/crater/pkg/models"
)

type DBService interface {
	Create(task *models.AITask) error
	Update(task *models.AITask) error
	UpdateStatus(taskID uint, status string, reason string) error
	UpdateJobName(taskID uint, jobname string) error
	DeleteByID(taskID uint) error
	DeleteByUserAndID(userName string, taskID uint) error
	ForceDeleteByUserAndID(userName string, taskID uint) error
	ListByTaskType(taskType string, page, pageSize int) ([]models.AITask, int64, error)
	ListByUserAndStatuses(userName string, status []string) ([]models.AITask, error)
	ListByQueue(queue string) ([]models.AITask, error)
	ListAll() ([]models.AITask, error)
	GetByID(taskID uint) (*models.AITask, error)
	GetByUserAndID(userName string, taskID uint) (*models.AITask, error)
	UpdateProfilingStat(taskID uint, profileStatus uint, stat string, status string) error
	UpdateToken(taskID uint, token string) error
	UpdateNodePort(taskID uint, nodePort int32) error
	GetTaskStatusCount() ([]models.TaskStatusCount, error)
	GetUserTaskStatusCount(userName string) ([]models.TaskStatusCount, error)
}

type service struct{}

func NewDBService() DBService {
	return &service{}
}

func (s *service) Create(task *models.AITask) error {
	return db.Orm.Create(task).Error
}

func (s *service) Update(task *models.AITask) error {
	return db.Orm.Save(task).Error
}

func (s *service) UpdateStatus(taskID uint, status, reason string) error {
	task, _ := s.GetByID(taskID)
	if task.Status == status {
		return nil
	}
	updateMap := make(map[string]any)
	updateMap["status"] = status
	// 取前100
	updateMap["status_reason"] = reason
	t := time.Now()
	switch status {
	case models.TaskCreatedStatus:
		updateMap["admitted_at"] = &t
	case models.TaskRunningStatus:
		updateMap["started_at"] = &t
	case models.TaskSucceededStatus, models.TaskFailedStatus:
		updateMap["finish_at"] = &t
		if task.StartedAt != nil {
			updateMap["duration"] = t.Sub(*task.StartedAt).Seconds()
		}
		updateMap["jct"] = t.Sub(task.CreatedAt).Seconds()
	}
	err := db.Orm.Model(&models.AITask{}).Where("id = ?", taskID).Updates(updateMap).Error
	return err
}

func (s *service) UpdateJobName(taskID uint, jobname string) error {
	err := db.Orm.Model(&models.AITask{}).Where("id = ?", taskID).Update("jobname", jobname).Error
	return err
}

func (s *service) DeleteByUserAndID(userName string, taskID uint) error {
	err := db.Orm.Model(&models.AITask{}).Where("username = ? and id = ?", userName, taskID).Update("is_deleted", true).Error
	return err
}

func (s *service) ForceDeleteByUserAndID(userName string, taskID uint) error {
	err := db.Orm.Where("username = ? and id = ?", userName, taskID).Delete(&models.AITask{}).Error
	return err
}

func (s *service) DeleteByID(taskID uint) error {
	return db.Orm.Delete(&models.AITask{}, taskID).Error
}

func (s *service) ListByUserAndStatuses(userName string, statuses []string) ([]models.AITask, error) {
	var tasks []models.AITask
	var err error
	if len(statuses) == 0 {
		err = db.Orm.Where("username = ? ", userName).Find(&tasks).Error
	} else {
		err = db.Orm.Where("username = ? and status IN ? and is_deleted = ?", userName, statuses, false).Find(&tasks).Error
	}
	return tasks, err
}

func (s *service) ListByTaskType(taskType string, page, pageSize int) ([]models.AITask, int64, error) {
	var totalRows int64
	var tasks []models.AITask

	query := db.Orm.Model(&models.AITask{}).Where("task_type = ?", taskType)
	query.Count(&totalRows)

	query.Order("created_at DESC")
	if pageSize > 0 {
		query = query.Limit(pageSize).Offset(page * pageSize)
	}
	err := query.Find(&tasks).Error
	return tasks, totalRows, err
}

func (s *service) ListByQueue(queue string) ([]models.AITask, error) {
	var tasks []models.AITask
	err := db.Orm.Where("username = ? and is_deleted = ?", queue, false).Find(&tasks).Error
	return tasks, err
}

func (s *service) ListAll() ([]models.AITask, error) {
	var tasks []models.AITask
	err := db.Orm.Find(&tasks).Error
	return tasks, err
}

func (s *service) GetByID(taskID uint) (*models.AITask, error) {
	var task models.AITask
	err := db.Orm.First(&task, taskID).Error
	return &task, err
}

func (s *service) GetByUserAndID(userName string, taskID uint) (*models.AITask, error) {
	var task models.AITask
	err := db.Orm.Where("username = ? and id = ?", userName, taskID).First(&task).Error
	return &task, err
}

func (s *service) UpdateProfilingStat(taskID, profileStatus uint, stat, status string) error {
	updateMap := make(map[string]any)
	updateMap["profile_status"] = profileStatus
	if profileStatus == models.ProfileFinish {
		updateMap["profile_stat"] = stat
		if status != "" {
			updateMap["status"] = status
		}
	}
	err := db.Orm.Model(&models.AITask{}).Where("id = ?", taskID).Updates(updateMap).Error
	return err
}

func (s *service) UpdateToken(taskID uint, token string) error {
	err := db.Orm.Model(&models.AITask{}).Where("id = ?", taskID).Update("token", token).Error
	return err
}

func (s *service) UpdateNodePort(taskID uint, nodePort int32) error {
	err := db.Orm.Model(&models.AITask{}).Where("id = ?", taskID).Update("node_port", nodePort).Error
	return err
}

func (s *service) GetTaskStatusCount() ([]models.TaskStatusCount, error) {
	var stats []models.TaskStatusCount
	err := db.Orm.Model(&models.AITask{}).Where("is_deleted = ?", false).
		Select("status, count(*) as count").
		Group("status").
		Scan(&stats).Error
	return stats, err
}

func (s *service) GetUserTaskStatusCount(userName string) ([]models.TaskStatusCount, error) {
	var stats []models.TaskStatusCount
	err := db.Orm.Model(&models.AITask{}).Where("username = ? and is_deleted = ?", userName, false).
		Select("status, count(*) as count").
		Group("status").
		Scan(&stats).Error
	return stats, err
}
