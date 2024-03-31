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
	// limit (cpu, memory, gpu, storage) for the user in the project
	// if the limit is 0, it means no limit
	CPU     int    `gorm:"type:int;not null;default:0;comment:CPU限制(核)"`
	Memory  int    `gorm:"type:int;not null;default:0;comment:内存限制(Gi)"`
	GPU     int    `gorm:"type:int;not null;default:0;comment:GPU限制(个)"`
	GPUMem  int    `gorm:"type:int;not null;default:0;comment:GPU内存配额(Gi)"`
	Storage int    `gorm:"type:int;not null;default:0;comment:存储限制(Gi)"`
	Access  string `gorm:"type:varchar(512);comment:可访问的资源种类(V100,P100...)"`
}
