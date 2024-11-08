package model

import (
	imagepackv1 "github.com/raids-lab/crater/pkg/apis/imagepack/v1"
	"gorm.io/gorm"
)

type ImagePack struct {
	gorm.Model
	UserID        uint
	User          User
	AccountID     uint
	Account       Account
	ImagePackName string             `gorm:"column:imagepackname;uniqueIndex:imagepackname;type:varchar(128);not null" json:"imagepackname"`
	ImageLink     string             `gorm:"column:imagelink;type:varchar(128);not null" json:"imagelink"`
	NameSpace     string             `gorm:"column:namespace;type:varchar(128);not null" json:"namespace"`
	Status        string             `gorm:"column:status;type:varchar(128);not null" json:"status"`
	NameTag       string             `gorm:"column:nametag;type:varchar(128);not null" json:"nametag"`
	Params        ImageProfileParams `gorm:"column:params;type:varchar(512);serializer:json;" json:"params"`
	NeedProfile   bool               `gorm:"column:needprofile;type:boolean;default:false" json:"needprofile"`
	TaskType      ImageTaskType      `gorm:"column:tasktype;not null;comment:作业状态" json:"tasktype"`
	Alias         string             `gorm:"column:alias;type:varchar(128);not null" json:"alias"`
	Description   string             `gorm:"column:description;type:varchar(512);not null" json:"description"`
	CreatorName   string             `gorm:"column:creatorname;type:varchar(128);not null" json:"creatorname"`
	IsPublic      bool               `gorm:"column:ispublic;type:boolean;default:false" json:"ispublic"`
	Size          int64              `gorm:"column:size;type:bigint;default:0" json:"size"`
}

type Kaniko struct {
	gorm.Model
	UserID        uint
	User          User
	ImagePackName string                 `gorm:"uniqueIndex:imagepackname;type:varchar(128);not null;comment:ImagePack CRD名称"`
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
	ImagePackName string          `gorm:"uniqueIndex:imagepackname;type:varchar(128);not null;comment:ImagePack CRD名称"`
	Description   *string         `gorm:"type:varchar(512);comment:描述"`
	IsPublic      bool            `gorm:"type:boolean;default:false;comment:是否公共"`
	TaskType      JobType         `gorm:"not null;comment:镜像任务类型"`
	ImageSource   ImageSourceType `gorm:"not null;comment:镜像来源类型"`
	Size          int64           `gorm:"type:bigint;default:0;comment:镜像大小"`
}

type ImageUpload struct {
	gorm.Model
	UserID      uint
	User        User
	AccountID   uint
	Account     Account
	ImageLink   string  `gorm:"column:imagelink;type:varchar(128);not null" json:"imagelink"`
	Status      string  `gorm:"column:status;type:varchar(128);not null" json:"status"`
	NameTag     string  `gorm:"column:nametag;type:varchar(128);not null" json:"nametag"`
	TaskType    JobType `gorm:"column:tasktype;not null;comment:作业状态" json:"tasktype"`
	Alias       string  `gorm:"column:alias;type:varchar(128);not null" json:"alias"`
	Description string  `gorm:"column:description;type:varchar(512);not null" json:"description"`
	CreatorName string  `gorm:"column:creatorname;type:varchar(128);not null" json:"creatorname"`
	IsPublic    bool    `gorm:"column:ispublic;type:boolean;default:false" json:"ispublic"`
}

type ImageProfileParams struct {
	Convs       uint    `json:"Convs"`
	Activations uint    `json:"Activations"`
	Denses      uint    `json:"Denses"`
	Others      uint    `json:"Others"`
	GFLOPs      float64 `json:"GFLOPs"`
	BatchSize   uint    `json:"BatchSize"`
	Params      uint    `json:"Params"`
	ModelSize   float64 `json:"ModelSize"`
	GPUMemUsage float64 `json:"GPUMemUsage"`
	GPUUtil     float64 `json:"GPUUtil"`
}
