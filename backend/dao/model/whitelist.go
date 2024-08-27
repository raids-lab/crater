package model

import (
	"gorm.io/gorm"
)

type Whitelist struct {
	gorm.Model
	Name string `gorm:"uniqueIndex;type:varchar(64);not null;comment:白名单名称"`
}
