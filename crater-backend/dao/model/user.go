package model

import (
	"gorm.io/gorm"
)

// User is the basic entity of the system
type User struct {
	gorm.Model
	Name         string  `gorm:"uniqueIndex;type:varchar(32);not null;comment:用户名"`
	Nickname     *string `gorm:"type:varchar(32);comment:昵称"`
	Password     *string `gorm:"type:varchar(128);comment:密码"`
	Role         string  `gorm:"type:varchar(32);not null;comment:集群角色 (admin, user, guest)"`
	Status       string  `gorm:"type:varchar(32);not null;comment:用户状态 (active, inactive)"`
	UserProjects []UserProject
}

type UserProject struct {
	gorm.Model
	UserID    uint   `gorm:"primaryKey"`
	ProjectID uint   `gorm:"primaryKey"`
	Role      string `gorm:"type:varchar(32);comment:用户在项目中的角色 (admin, user)"`
	Quota     string `gorm:"type:text;comment:配额限制"`
}
