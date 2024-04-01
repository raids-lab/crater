package model

import "gorm.io/gorm"

type Project struct {
	gorm.Model
	Name         string  `gorm:"uniqueIndex;type:varchar(32);not null;comment:项目名"`
	Description  *string `gorm:"type:varchar(128);comment:项目描述"`
	Namespace    string  `gorm:"type:varchar(128);not null;comment:命名空间"`
	Status       Status  `gorm:"index:status;not null;comment:项目状态 (pending, active, inactive)"`
	IsPersonal   bool    `gorm:"type:boolean;not null;comment:是否为个人项目"`
	UserProjects []UserProject
	// Project can also associate with multiple spaces in RW or RO mode
	ProjectSpaces []ProjectSpace
}

type ProjectSpace struct {
	gorm.Model
	ProjectID uint       `gorm:"primaryKey"`
	SpaceID   uint       `gorm:"primaryKey"`
	Mode      AccessMode `gorm:"not null;comment:项目空间的访问模式 (rw, ro, ao)"`
}
