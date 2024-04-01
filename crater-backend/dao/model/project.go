package model

import "gorm.io/gorm"

type Project struct {
	gorm.Model
	Name         string  `gorm:"uniqueIndex;type:varchar(32);comment:项目名"`
	Description  *string `gorm:"type:varchar(128);comment:项目描述"`
	Namespace    string  `gorm:"column:namespace;type:varchar(128);comment:命名空间"`
	Status       string  `gorm:"type:varchar(32);comment:项目状态 (active, inactive)"`
	IsPersonal   bool    `gorm:"type:boolean;comment:是否为个人项目"`
	UserProjects []UserProject
	// Project can also associate with multiple spaces in RW or RO mode
	ProjectSpaces []ProjectSpace
}

type ProjectSpace struct {
	gorm.Model
	ProjectID uint   `gorm:"primaryKey"`
	SpaceID   uint   `gorm:"primaryKey"`
	Mode      string `gorm:"type:varchar(32);comment:项目空间的访问模式 (rw, ro)"`
}
