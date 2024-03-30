package imagepack

import (
	db "github.com/raids-lab/crater/pkg/db/orm"
	"github.com/raids-lab/crater/pkg/models"
)

type DBService interface {
	Create(imagepack *models.ImagePack) error
	ListAll(userName string) ([]models.ImagePack, error)
	ListAvailable(userName string) ([]models.ImagePack, error)
	ListAdminPersonal() ([]models.ImagePack, error)
	ListAdminPublic() ([]models.ImagePack, error)
	UpdateStatusByUserName(userName string, status string) error
	UpdateStatusByEntity(imagepack *models.ImagePack, status string) error
	DeleteByName(imagepackname string) error
	DeleteByID(imagepackID uint) error
	GetImagePack(imagepackID uint, username string) (*models.ImagePack, error)
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

func (s *service) ListAll(userName string) ([]models.ImagePack, error) {
	var imagepacks []models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("username = ? AND status != ?", userName, ImageDeleted).Find(&imagepacks).Error
	return imagepacks, err
}

func (s *service) ListAdminPersonal() ([]models.ImagePack, error) {
	var imagepacks []models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("username != ? AND status != ?", "admin", ImageDeleted).Find(&imagepacks).Error
	return imagepacks, err
}

func (s *service) ListAdminPublic() ([]models.ImagePack, error) {
	var imagepacks []models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("username = ? AND status != ?", "admin", ImageDeleted).Find(&imagepacks).Error
	return imagepacks, err
}

func (s *service) ListAvailable(userName string) ([]models.ImagePack, error) {
	var imagepacks []models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("username = ? AND status = ?", userName, ImageFinished).Find(&imagepacks).Error
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
	return db.Orm.Model(&models.ImagePack{}).Where("imagepackname = ? ", imagepackname).Update("Status", ImageDeleted).Error
}

func (s *service) DeleteByID(imagepackID uint) error {
	return db.Orm.Delete(&models.ImagePack{}, imagepackID).Error
}

func (s *service) GetImagePack(imagepackID uint, username string) (*models.ImagePack, error) {
	var imagepack *models.ImagePack
	err := db.Orm.Model(&models.ImagePack{}).Where("id = ? AND username = ?", imagepackID, username).First(&imagepack).Error
	return imagepack, err
}
