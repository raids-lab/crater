package model

import "gorm.io/gorm"

type Image struct {
	gorm.Model
	UserID        uint
	User          User
	QueueID       uint
	Queue         Queue
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
