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

	AccessMode AccessMode `gorm:"not null;default:0;comment:项目空间的访问模式 (ro, ao, rw)"`

	// quota (job, node, cpu, memory, gpu, gpuMem, storage) for the project
	// same as Quota
	JobReq int `gorm:"type:int;not null;default:-1;comment:可以提交的 Job 数量"`
	Job    int `gorm:"type:int;not null;default:-1;comment:可以同时运行的 Job 数量"`

	NodeReq int `gorm:"type:int;not null;default:-1;comment:可以提交的节点数量"`
	Node    int `gorm:"type:int;not null;default:-1;comment:可以同时使用的节点数量"`

	CPUReq int `gorm:"type:int;not null;default:0;comment:可以提交的 CPU 核心数量"`
	CPU    int `gorm:"type:int;not null;default:0;comment:可以同时使用的 CPU 核心数量"`

	GPUReq int `gorm:"type:int;not null;default:0;comment:可以提交的 GPU 数量"`
	GPU    int `gorm:"type:int;not null;default:0;comment:可以同时使用的 GPU 数量"`

	MemReq int `gorm:"type:int;not null;default:0;comment:可以提交的内存配额 (Gi)"`
	Mem    int `gorm:"type:int;not null;default:0;comment:可以同时使用的内存配额 (Gi)"`

	GPUMemReq int `gorm:"type:int;not null;default:-1;comment:可以提交的 GPU 内存配额 (Gi)"`
	GPUMem    int `gorm:"type:int;not null;default:-1;comment:可以同时使用的 GPU 内存配额 (Gi)"`

	Storage int `gorm:"type:int;not null;default:50;comment:存储配额 (Gi)"`

	Extra *string `gorm:"comment:可访问的资源限制 (V100,P100...)"`
}
