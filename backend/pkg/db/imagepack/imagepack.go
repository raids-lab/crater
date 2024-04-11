package imagepack

import (
	db "github.com/raids-lab/crater/pkg/db/orm"
	"github.com/raids-lab/crater/pkg/models"
)

type DBService interface {
	Create(imagepack *models.ImagePack) error
	GetImagePackByID(imagepackID uint, createrName string) (*models.ImagePack, error)
	GetImagePackByName(imagepackname string) (*models.ImagePack, error)
	ListAll(createrName string) ([]models.ImagePack, error)
	ListAvailable(createrName string) ([]models.ImagePack, error)
	ListAdminPersonal() ([]models.ImagePack, error)
	ListAdminPublic() ([]models.ImagePack, error)
	UpdateStatusByUserName(createrName string, status string) error
	UpdateStatusByEntity(imagepack *models.ImagePack, status string) error
	UpdateParamsByImagePackName(imagepackname string, params string) error
	DeleteByName(imagepackname string) error
	DeleteByID(imagepackID uint) error
}

const (
	ImageFinished = "Finished"
	ImageDeleted  = "Deleted"
)

type service struct{}

func NewDBService() DBService {
	return &service{}
}

func (s *service) Create(imagepack *models.ImagePack) error {
	return db.Orm.Model(&models.ImagePack{}).Create(imagepack).Error // db.Orm.Create(user).Error
}

func (s *service) ListAll(createrName string) ([]models.ImagePack, error) {
	var imagepacks []models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("creatername = ? AND status != ?", createrName, ImageDeleted).Find(&imagepacks).Error
	return imagepacks, err
}

func (s *service) ListAdminPersonal() ([]models.ImagePack, error) {
	var imagepacks []models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("creatername != ? AND status != ?", "admin", ImageDeleted).Find(&imagepacks).Error
	return imagepacks, err
}

func (s *service) ListAdminPublic() ([]models.ImagePack, error) {
	var imagepacks []models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("creatername = ? AND status != ?", "admin", ImageDeleted).Find(&imagepacks).Error
	return imagepacks, err
}

func (s *service) ListAvailable(createrName string) ([]models.ImagePack, error) {
	var imagepacks []models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("creatername = ? AND status = ?", createrName, ImageFinished).Find(&imagepacks).Error
	return imagepacks, err
}

func (s *service) UpdateStatusByUserName(createrName, status string) error {
	return db.Orm.Model(&models.ImagePack{}).Where("creatername = ? ", createrName).Update("Status", status).Error
}

func (s *service) UpdateStatusByEntity(imagepack *models.ImagePack, status string) error {
	imagepack.Status = status
	return db.Orm.Save(imagepack).Error
}

func (s *service) UpdateParamsByImagePackName(imagepackname, params string) error {
	return db.Orm.Model(&models.ImagePack{}).Where("imagepackname = ? ", imagepackname).Update("Params", params).Error
}

func (s *service) DeleteByName(imagepackname string) error {
	return db.Orm.Model(&models.ImagePack{}).Where("imagepackname = ? ", imagepackname).Update("Status", ImageDeleted).Error
}

func (s *service) DeleteByID(imagepackID uint) error {
	return db.Orm.Delete(&models.ImagePack{}, imagepackID).Error
}

func (s *service) GetImagePackByID(imagepackID uint, createrName string) (*models.ImagePack, error) {
	var imagepack *models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("id = ? AND creatername = ?", imagepackID, createrName).First(&imagepack).Error
	return imagepack, err
}

func (s *service) GetImagePackByName(imagepackname string) (*models.ImagePack, error) {
	var imagepack *models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("imagepackname = ?", imagepackname).First(&imagepack).Error
	return imagepack, err
}
