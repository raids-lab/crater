package model

import (
	"gorm.io/gorm"
)

type Jobtemplate struct {
	gorm.Model
	Name     string `gorm:"not null;type:varchar(256);comment:作业模板名称"`
	Describe string `gorm:"type:varchar(512);comment:作业模板的描述"`
	Document string `gorm:"type:text;comment:作业模板的文档"`
	Template string `gorm:"type:text;comment:作业的模板配置"`

	UserID uint
	User   User
}
