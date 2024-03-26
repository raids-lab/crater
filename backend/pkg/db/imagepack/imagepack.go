package imagepack

import (
	db "github.com/raids-lab/crater/pkg/db/orm"
	"github.com/raids-lab/crater/pkg/models"
)

type DBService interface {
	Create(imagepack *models.ImagePack) error
	ListAll(userName string) ([]models.ImagePack, error)
	ListAvailable(userName string) ([]models.ImagePack, error)
	UpdateStatusByUserName(userName string, status string) error
	UpdateStatusByEntity(imagepack *models.ImagePack, status string) error
	DeleteByName(imagepackname string) error
}

type service struct{}

func NewDBService() DBService {
	return &service{}
}

func (s *service) Create(imagepack *models.ImagePack) error {
	return db.Orm.Model(&models.ImagePack{}).Create(imagepack).Error // db.Orm.Create(user).Error
}

func (s *service) ListAll(userName string) ([]models.ImagePack, error) {
	var imagepacks []models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("username = ? AND status != ?", userName, "Deleted").Find(&imagepacks).Error
	return imagepacks, err
}

func (s *service) ListAvailable(userName string) ([]models.ImagePack, error) {
	var imagepacks []models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("username = ? AND status = ?", userName, "Finished").Find(&imagepacks).Error
	return imagepacks, err
}

func (s *service) UpdateStatusByUserName(userName, status string) error {
	return db.Orm.Model(&models.ImagePack{}).Where("username = ? ", userName).Update("Status", status).Error
}

func (s *service) UpdateStatusByEntity(imagepack *models.ImagePack, status string) error {
	imagepack.Status = status
	return db.Orm.Save(imagepack).Error
}

func (s *service) DeleteByName(imagepackname string) error {
	return db.Orm.Model(&models.ImagePack{}).Where("imagepackname = ? ", imagepackname).Update("Status", "Deleted").Error
}
