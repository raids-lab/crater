package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BuildStatus string

const (
	BuildJobInitial  BuildStatus = "Initial"
	BuildJobPending  BuildStatus = "Pending"
	BuildJobRunning  BuildStatus = "Running"
	BuildJobFinished BuildStatus = "Finished"
	BuildJobFailed   BuildStatus = "Failed"
	BuildJobCanceled BuildStatus = "Canceled"
)

type BuildSource string

const (
	Dockerfile   BuildSource = "Dockerfile"
	PipApt       BuildSource = "PipApt"
	Snapshot     BuildSource = "Snapshot"
	EnvdAdvanced BuildSource = "EnvdAdvanced"
	EnvdRaw      BuildSource = "EnvdRaw"
)

type ImageShareType string

const (
	Private      ImageShareType = "Private"
	Public       ImageShareType = "Public"
	UserShare    ImageShareType = "UserShare"
	AccountShare ImageShareType = "AccountShare"
)

type Kaniko struct {
	gorm.Model
	UserID        uint
	User          User
	ImagePackName string                       `gorm:"uniqueIndex;type:varchar(128);not null;comment:ImagePack CRD 名称"`
	ImageLink     string                       `gorm:"type:varchar(512);not null;comment:镜像链接"`
	NameSpace     string                       `gorm:"type:varchar(128);not null;comment:命名空间"`
	Status        BuildStatus                  `gorm:"not null;comment:构建状态"`
	Description   *string                      `gorm:"type:varchar(512);comment:描述"`
	Size          int64                        `gorm:"type:bigint;default:0;comment:镜像大小"`
	Dockerfile    *string                      `gorm:"type:text;comment:Dockerfile内容"`
	BuildSource   BuildSource                  `gorm:"type:varchar(32);not null;default:buildkit;comment:构建来源"`
	Tags          datatypes.JSONType[[]string] `gorm:"null;comment:镜像标签"`
	Template      string                       `gorm:"type:text;comment:镜像的模板配置"`
	Archs         datatypes.JSONType[[]string] `gorm:"null;comment:镜像架构"`
}

type Image struct {
	gorm.Model
	UserID        uint
	User          User
	ImageLink     string                       `gorm:"type:varchar(128);not null;comment:镜像链接"`
	ImagePackName *string                      `gorm:"uniqueIndex;type:varchar(128);null;comment:ImagePack CRD 名称"`
	Description   *string                      `gorm:"type:varchar(512);comment:描述"`
	IsPublic      bool                         `gorm:"type:boolean;default:false;comment:是否公共"`
	TaskType      JobType                      `gorm:"not null;comment:镜像任务类型"`
	ImageSource   ImageSourceType              `gorm:"not null;comment:镜像来源类型"`
	Size          int64                        `gorm:"type:bigint;default:0;comment:镜像大小"`
	Tags          datatypes.JSONType[[]string] `gorm:"null;comment:镜像标签"`
	Archs         datatypes.JSONType[[]string] `gorm:"null;comment:镜像架构"`
}

type ImageUser struct {
	gorm.Model
	ImageID uint
	Image   Image
	UserID  uint
	User    User
}

type ImageAccount struct {
	gorm.Model
	ImageID   uint
	Image     Image
	AccountID uint
	Account   Account
}

type CudaBaseImage struct {
	gorm.Model
	Label      string `gorm:"type:varchar(128);not null;comment:image label showed in UI"`
	ImageLabel string `gorm:"uniqueIndex;type:varchar(128);null;comment:image label for imagelink generate"`
	Value      string `gorm:"type:varchar(512);comment:Full Cuda Image Link"`
}
