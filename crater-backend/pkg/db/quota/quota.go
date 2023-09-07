package quota

import (
	db "github.com/aisystem/ai-protal/pkg/db/orm"
	"github.com/aisystem/ai-protal/pkg/models"
)

type DBService interface {
	Create(task *models.Quota) error
	Update(task *models.Quota) error
	ListAllQuotas() ([]models.Quota, error)
	DeleteByUserName(username string) error
	GetByUserName(username string) (*models.Quota, error)
}

type service struct{}

func NewDBService() DBService {
	return &service{}
}

func (s *service) Create(quota *models.Quota) error {
	return db.Orm.Create(quota).Error
}

func (s *service) Update(quota *models.Quota) error {
	return db.Orm.Save(quota).Error
}

func (s *service) ListAllQuotas() ([]models.Quota, error) {
	var quotas []models.Quota
	err := db.Orm.Find(&quotas).Error
	return quotas, err
}

func (s *service) DeleteByUserName(username string) error {
	return db.Orm.Delete(&models.Quota{}, username).Error
}

func (s *service) GetByUserName(username string) (*models.Quota, error) {
	var quota models.Quota
	err := db.Orm.First(&quota, username).Error
	return &quota, err
}
