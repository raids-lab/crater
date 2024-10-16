package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const ImageQuotaInfinity = -1

// Optional fields for user
type UserAttribute struct {
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Avatar string `json:"avatar"`
}

// User is the basic entity of the system
type User struct {
	gorm.Model
	Name       string     `gorm:"uniqueIndex;type:varchar(64);not null;comment:用户名"`
	Nickname   string     `gorm:"type:varchar(64);comment:昵称"`
	Password   *string    `gorm:"type:varchar(256);comment:密码"`
	Role       Role       `gorm:"index:role;not null;comment:用户在平台的角色 (guest, user, admin)"`
	Status     Status     `gorm:"index:status;not null;comment:用户状态 (pending, active, inactive)"`
	Space      string     `gorm:"uniqueIndex;type:varchar(256);not null;comment:用户空间绝对路径"`
	AccessMode AccessMode `gorm:"not null;comment:用户在公共空间的访问模式 (na, ro, rw)"`
	ImageQuota int64      `gorm:"type:bigint;default:-1;comment:用户在镜像仓库的配额"`

	Attributes   datatypes.JSONType[UserAttribute] `gorm:"comment:用户的额外属性 (昵称、邮箱、电话、头像等)"`
	UserQueues   []UserQueue
	UserDatasets []UserDataset
}
