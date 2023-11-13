package user

import (
	db "github.com/aisystem/ai-protal/pkg/db/orm"
	"github.com/aisystem/ai-protal/pkg/models"
	"github.com/aisystem/ai-protal/pkg/util"
)

type DBService interface {
	Create(user *models.User) error
	UpdateRole(userName string, role1 string) error
	ListAllUsers() ([]models.User, error)
	ListAllUserQuotas() ([]models.UserQuota, error)
	DeleteByUserName(username string) error
	GetByUserName(username string) (*models.User, error)
	CreateAccessToken(user *models.User, secret string, expiry int) (accessToken string, err error)
	CreateRefreshToken(user *models.User, secret string, expiry int) (refreshToken string, err error)
	GetUserByID(id uint) (*models.User, error)
}

type service struct{}

func NewDBService() DBService {
	return &service{}
}

func (s *service) Create(user *models.User) error {
	return db.Orm.Create(user).Error //db.Orm.Create(user).Error
}

func (s *service) UpdateRole(userName string, role1 string) error {
	var user models.User
	return db.Orm.Where("username=?", userName).First(&user).Update("role", role1).Error //db.Orm.Save(user).Error
}

func (s *service) ListAllUsers() ([]models.User, error) {
	var users []models.User
	err := db.Orm.Find(&users).Error
	return users, err
}

func (s *service) ListAllUserQuotas() ([]models.UserQuota, error) {
	var userQuotas []models.UserQuota
	err := db.Orm.Table("users").Select("users.*, quotas.*").Joins("left join quotas on users.username = quotas.username").Scan(&userQuotas).Error
	if err != nil {
		return nil, err
	}
	return userQuotas, nil
}

func (s *service) DeleteByUserName(username string) error {
	return db.Orm.Where("username=?", username).Delete(&models.User{}).Error
}
func (s *service) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	err := db.Orm.Where("id=?", id).First(&user).Error
	return &user, err
}

func (s *service) GetByUserName(username string) (*models.User, error) {
	var user models.User
	err := db.Orm.Where("username=?", username).First(&user).Error
	return &user, err
}
func (s *service) CreateAccessToken(user *models.User, secret string, expiry int) (accessToken string, err error) {
	return util.CreateAccessToken(user, secret, expiry)
}

func (s *service) CreateRefreshToken(user *models.User, secret string, expiry int) (refreshToken string, err error) {
	return util.CreateRefreshToken(user, secret, expiry)
}
