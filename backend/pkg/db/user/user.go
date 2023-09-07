package user

import (
	db "github.com/aisystem/ai-protal/pkg/db/orm"
	"github.com/aisystem/ai-protal/pkg/models"
)

type DBService interface {
	Create(user *models.User) error
	Update(user *models.User) error
	ListAllUsers() ([]models.User, error)
	DeleteByUserName(username string) error
	GetByUserName(username string) (*models.User, error)
}

type service struct{}

func NewDBService() DBService {
	return &service{}
}

func (s *service) Create(user *models.User) error {
	return db.Orm.Create(user).Error
}

func (s *service) Update(user *models.User) error {
	return db.Orm.Save(user).Error
}

func (s *service) ListAllUsers() ([]models.User, error) {
	var users []models.User
	err := db.Orm.Find(&users).Error
	return users, err
}

func (s *service) DeleteByUserName(username string) error {
	return db.Orm.Delete(&models.User{}, username).Error
}

func (s *service) GetByUserName(username string) (*models.User, error) {
	var user models.User
	err := db.Orm.First(&user, username).Error
	return &user, err
}
