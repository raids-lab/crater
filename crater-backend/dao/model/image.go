package model

import (
	imagepackv1 "github.com/raids-lab/crater/pkg/apis/imagepack/v1"
	"gorm.io/gorm"
)

type Kaniko struct {
	gorm.Model
	UserID        uint
	User          User
	ImagePackName string                 `gorm:"uniqueIndex;type:varchar(128);not null;comment:ImagePack CRD 名称"`
	ImageLink     string                 `gorm:"type:varchar(128);not null;comment:镜像链接"`
	NameSpace     string                 `gorm:"type:varchar(128);not null;comment:命名空间"`
	Status        imagepackv1.PackStatus `gorm:"not null;comment:构建状态"`
	Description   *string                `gorm:"type:varchar(512);comment:描述"`
	Size          int64                  `gorm:"type:bigint;default:0;comment:镜像大小"`
}

type Image struct {
	gorm.Model
	UserID        uint
	User          User
	ImageLink     string          `gorm:"type:varchar(128);not null;comment:镜像链接"`
	ImagePackName string          `gorm:"uniqueIndex;type:varchar(128);not null;comment:ImagePack CRD 名称"`
	Description   *string         `gorm:"type:varchar(512);comment:描述"`
	IsPublic      bool            `gorm:"type:boolean;default:false;comment:是否公共"`
	TaskType      JobType         `gorm:"not null;comment:镜像任务类型"`
	ImageSource   ImageSourceType `gorm:"not null;comment:镜像来源类型"`
	Size          int64           `gorm:"type:bigint;default:0;comment:镜像大小"`
}
