package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// OperationLog 记录关键操作日志
type OperationLog struct {
	gorm.Model

	// 操作人信息
	Operator     string `gorm:"type:varchar(64);index;not null;comment:操作人用户名"`
	OperatorRole string `gorm:"type:varchar(32);comment:操作人角色(admin/user)"`

	// 操作类型
	// 建议使用 constants 包中定义的类型
	OperationType string `gorm:"type:varchar(64);index;not null;comment:操作类型"`

	// 操作对象
	Target string `gorm:"type:varchar(255);comment:操作目标对象(如节点名, Pod名)"`

	// 操作详情 (记录变更前后的值或关键参数)
	Details datatypes.JSON `gorm:"comment:操作详情JSON"`

	// 执行状态
	Status  string `gorm:"type:varchar(32);comment:执行状态(Success/Failed)"`
	Message string `gorm:"type:text;comment:错误信息或备注"`
}

// TableName 指定表名
func (OperationLog) TableName() string {
	return "operation_logs"
}
