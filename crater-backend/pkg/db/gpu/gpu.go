package gpu

import (
	db "github.com/raids-lab/crater/pkg/db/orm"
	"github.com/raids-lab/crater/pkg/models"
)

type DBService interface {
	Create(gpu *models.GPU) error
	Update(gpu *models.GPU) error
	DeleteByID(gpuID uint) error
	GetByID(gpuID uint) (*models.GPU, error)
	List() ([]models.GPU, error)
}

type service struct{}

func NewDBService() DBService {
	return &service{}
}

func (s *service) Create(gpu *models.GPU) error {
	return db.Orm.Create(gpu).Error
}

func (s *service) Update(gpu *models.GPU) error {
	return db.Orm.Save(gpu).Error
}

func (s *service) DeleteByID(gpuID uint) error {
	return db.Orm.Delete(&models.GPU{}, gpuID).Error
}

func (s *service) GetByID(gpuID uint) (*models.GPU, error) {
	var gpu models.GPU
	err := db.Orm.First(&gpu, gpuID).Error
	return &gpu, err
}

func (s *service) List() ([]models.GPU, error) {
	var gpus []models.GPU
	err := db.Orm.Order("gpu_priority DESC").Find(&gpus).Error
	return gpus, err
}
