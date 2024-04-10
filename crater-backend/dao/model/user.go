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
	Role         Role    `gorm:"index:role;not null;comment:用户在平台的角色 (guest, user, admin)"`
	Status       Status  `gorm:"index:status;not null;comment:用户状态 (pending, active, inactive)"`
	UserProjects []UserProject
}

type UserProject struct {
	gorm.Model
	UserID    uint `gorm:"primaryKey"`
	ProjectID uint `gorm:"primaryKey"`
	Role      Role `gorm:"not null;comment:用户在项目中的角色 (guest, user, admin)"`

	AccessMode AccessMode `gorm:"not null;comment:项目空间的访问模式 (ro, ao, rw)"`

	// quota limit (job, node, cpu, memory, gpu, gpuMem, storage) for the project
	// if value is -1, than use the quota value in project
	EmbeddedQuota `gorm:"embedded"`
}
