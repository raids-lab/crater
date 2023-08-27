package task

import (
	db "github.com/aisys/ai-task-controller/pkg/db/internal"
	"github.com/aisys/ai-task-controller/pkg/models"
)

type DBService interface {
	Create(task *models.TaskModel) error
	Update(task *models.TaskModel) error
	DeleteByID(taskID uint) error
	ListByUserAndStatus(userName string, status string) ([]models.TaskModel, error)
	GetByID(taskID uint) (*models.TaskModel, error)
}

type service struct{}

func NewDBService() DBService {
	return &service{}
}

func (s *service) Create(task *models.TaskModel) error {
	return db.Orm.Create(task).Error
}

func (s *service) Update(task *models.TaskModel) error {
	return db.Orm.Save(task).Error
}

func (s *service) DeleteByID(taskID uint) error {
	return db.Orm.Delete(&models.TaskModel{}, taskID).Error
}

func (s *service) ListByUserAndStatus(userName string, status string) ([]models.TaskModel, error) {
	var tasks []models.TaskModel
	var err error
	if status == "" {
		err = db.Orm.Where("user_name = ?", userName, status).Find(&tasks).Error
	} else {
		err = db.Orm.Where("user_name = ? and status = ?", userName, status).Find(&tasks).Error

	}
	return tasks, err
}

func (s *service) GetByID(taskID uint) (*models.TaskModel, error) {
	var task models.TaskModel
	err := db.Orm.First(&task, taskID).Error
	return &task, err
}
