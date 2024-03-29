package model

import (
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Project struct {
	gorm.Model
	Name         string  `gorm:"uniqueIndex;type:varchar(32);comment:项目名"`
	Description  *string `gorm:"type:varchar(128);comment:项目描述"`
	NameSpace    string  `gorm:"column:namespace;type:varchar(128);comment:命名空间"`
	Status       string  `gorm:"type:varchar(32);comment:项目状态 (active, inactive)"`
	Quota        string  `gorm:"type:text;comment:配额信息"`
	IsPersonal   bool    `gorm:"type:boolean;comment:是否为个人项目"`
	UserProjects []UserProject
}

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
	RoleGuest = "guest"
)

const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

var (
	DefaultQuota = v1.ResourceList{
		v1.ResourceCPU:                    resource.MustParse("2"),
		v1.ResourceMemory:                 resource.MustParse("4Gi"),
		v1.ResourceName("nvidia.com/gpu"): resource.MustParse("0"),
	}
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
