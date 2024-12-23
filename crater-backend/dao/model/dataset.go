package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DataType string

const (
	DataTypeDataset DataType = "dataset"
	DataTypeModel   DataType = "model"
)

type Extracontent struct {
	Tags   []string `json:"tag,omitempty"`
	WebURL *string  `json:"weburl,omitempty"`
}
type Dataset struct {
	gorm.Model
	Name     string                           `gorm:"type:varchar(256);not null;comment:数据集名"`
	URL      string                           `gorm:"type:varchar(512);not null;comment:数据集空间路径"`
	Describe string                           `gorm:"type:text;comment:数据集描述"`
	Type     DataType                         `gorm:"type:varchar(32);not null;default:dataset;comment:数据类型"`
	Extra    datatypes.JSONType[Extracontent] `gorm:"comment:额外信息(tags、weburl等)"`
	UserID   uint

	UserDatasets    []UserDataset
	AccountDatasets []AccountDataset
}

type UserDataset struct {
	gorm.Model
	UserID    uint `gorm:"primaryKey"`
	DatasetID uint `gorm:"primaryKey"`
}

type AccountDataset struct {
	gorm.Model
	AccountID uint `gorm:"primaryKey"`
	DatasetID uint `gorm:"primaryKey"`
}
