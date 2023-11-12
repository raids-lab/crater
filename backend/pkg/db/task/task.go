package task

import (
	db "github.com/aisystem/ai-protal/pkg/db/orm"
	"github.com/aisystem/ai-protal/pkg/models"
)

type DBService interface {
	Create(task *models.AITask) error
	Update(task *models.AITask) error
	UpdateStatus(taskID uint, status string) error
	DeleteByID(taskID uint) error
	DeleteByUserAndID(userName string, taskID uint) error
	ListByUserAndStatus(userName string, status string) ([]models.AITask, error)
	GetByID(taskID uint) (*models.AITask, error)
	GetByUserAndID(userName string, taskID uint) (*models.AITask, error)
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

func (s *service) UpdateStatus(taskID uint, status string) error {
	err := db.Orm.Model(&models.AITask{}).Where("id = ?", taskID).Update("status", status).Error
	return err
}

func (s *service) DeleteByUserAndID(userName string, taskID uint) error {
	err := db.Orm.Model(&models.AITask{}).Where("user_name = ? and id = ?", userName, taskID).Update("is_deleted", true).Error
	return err
	// return db.Orm.Delete(&models.AITask{}, taskID).Error
}

func (s *service) DeleteByID(taskID uint) error {
	return db.Orm.Delete(&models.AITask{}, taskID).Error
}

func (s *service) ListByUserAndStatus(userName string, status string) ([]models.AITask, error) {
	var tasks []models.AITask
	var err error
	if status == "" {
		err = db.Orm.Where("user_name = ? and is_deleted = ?", userName, false).Find(&tasks).Error
	} else {
		err = db.Orm.Where("user_name = ? and status = ? and is_deleted = ?", userName, status, false).Find(&tasks).Error

	}
	return tasks, err
}

func (s *service) GetByID(taskID uint) (*models.AITask, error) {
	var task models.AITask
	err := db.Orm.First(&task, taskID).Error
	return &task, err
}

func (s *service) GetByUserAndID(userName string, taskID uint) (*models.AITask, error) {
	var task models.AITask
	err := db.Orm.Where("user_name = ? and id = ?", userName, taskID).First(&task).Error
	return &task, err
}
