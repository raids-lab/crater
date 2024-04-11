package models

type ImagePack struct {
	Model
	ImagePackName string          `gorm:"column:imagepackname;uniqueIndex:imagepackname;type:varchar(128);not null" json:"imagepackname"`
	Project       string          `gorm:"column:project;type:varchar(128);not null" json:"project"`
	CreaterName   string          `gorm:"column:username;type:varchar(128);not null" json:"creatername"`
	ImageLink     string          `gorm:"column:imagelink;type:varchar(128);not null" json:"imagelink"`
	NameSpace     string          `gorm:"column:namespace;type:varchar(128);not null" json:"namespace"`
	Status        string          `gorm:"column:status;type:varchar(128);not null" json:"status"`
	NameTag       string          `gorm:"column:nametag;type:varchar(128);not null" json:"nametag"`
	Params        ImagePackParams `gorm:"column:params;type:varchar(128);serializer:json;" json:"params"`
	NeedProfile   bool            `gorm:"column:needprofile;type:tinyint;default:false" json:"needprofile"`
}

type ImagePackParams struct {
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
