package model

import (
	"gorm.io/gorm"
)

type Dataset struct {
	gorm.Model
	Name     string `gorm:"type:varchar(256);not null;comment:数据集名"`
	URL      string `gorm:"type:varchar(512);not null;comment:数据集空间路径"`
	Describe string `gorm:"type:text;comment:数据集描述"`
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
