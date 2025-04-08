package model

import (
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
	BuildKit BuildSource = "buildkit"
	Snapshot BuildSource = "snapshot"
	Envd     BuildSource = "envd"
)

type Kaniko struct {
	gorm.Model
	UserID        uint
	User          User
	ImagePackName string      `gorm:"uniqueIndex;type:varchar(128);not null;comment:ImagePack CRD 名称"`
	ImageLink     string      `gorm:"type:varchar(128);not null;comment:镜像链接"`
	NameSpace     string      `gorm:"type:varchar(128);not null;comment:命名空间"`
	Status        BuildStatus `gorm:"not null;comment:构建状态"`
	Description   *string     `gorm:"type:varchar(512);comment:描述"`
	Size          int64       `gorm:"type:bigint;default:0;comment:镜像大小"`
	Dockerfile    *string     `gorm:"type:text;comment:Dockerfile内容"`
	BuildSource   BuildSource `gorm:"type:varchar(32);not null;default:buildkit;comment:构建来源"`
}

type Image struct {
	gorm.Model
	UserID        uint
	User          User
	ImageLink     string          `gorm:"type:varchar(128);not null;comment:镜像链接"`
	ImagePackName *string         `gorm:"uniqueIndex;type:varchar(128);null;comment:ImagePack CRD 名称"`
	Description   *string         `gorm:"type:varchar(512);comment:描述"`
	IsPublic      bool            `gorm:"type:boolean;default:false;comment:是否公共"`
	TaskType      JobType         `gorm:"not null;comment:镜像任务类型"`
	ImageSource   ImageSourceType `gorm:"not null;comment:镜像来源类型"`
	Size          int64           `gorm:"type:bigint;default:0;comment:镜像大小"`
}
