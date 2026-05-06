package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ApprovalOrderType 审批订单类型
type ApprovalOrderType string

const (
	ApprovalOrderTypeDataset ApprovalOrderType = "dataset" // 数据集类型
	ApprovalOrderTypeJob     ApprovalOrderType = "job"     // 任务类型
)

// ApprovalOrderStatus 审批订单状态
type ApprovalOrderStatus string

const (
	ApprovalOrderStatusPending   ApprovalOrderStatus = "Pending"  // 待审批
	ApprovalOrderStatusApproved  ApprovalOrderStatus = "Approved" // 已批准
	ApprovalOrderStatusRejected  ApprovalOrderStatus = "Rejected" // 已拒绝
	ApprovalOrderStatusCancelled ApprovalOrderStatus = "Canceled" // 已取消
)

// ApprovalOrderContent 审批订单内容
type ApprovalOrderContent struct {
	ApprovalOrderTypeID         uint   `json:"approvalorderTypeID"`
	ApprovalOrderExtensionHours uint   `json:"approvalorderExtensionHours"` // 延长小时数
	ApprovalOrderReason         string `json:"approvalorderReason"`         // 审批原因
}

// ReviewSource 审批来源
type ReviewSource string

const (
	ReviewSourceNone        ReviewSource = ""             // 历史记录或未处理
	ReviewSourceSystemAuto  ReviewSource = "system_auto"  // 简单规则自动审批
	ReviewSourceAgentAuto   ReviewSource = "agent_auto"   // Agent 评估通过
	ReviewSourceAdminManual ReviewSource = "admin_manual" // 管理员手动审批
)

// ApprovalOrder 审批订单模型
type ApprovalOrder struct {
	gorm.Model
	Name        string                                   `gorm:"type:varchar(256);not null;comment:审批订单名称"`
	Type        ApprovalOrderType                        `gorm:"type:varchar(32);not null;default:job;comment:审批订单类型"`
	Status      ApprovalOrderStatus                      `gorm:"type:varchar(32);not null;default:Pending;comment:审批订单状态"`
	Content     datatypes.JSONType[ApprovalOrderContent] `gorm:"comment:审批订单内容"`
	ReviewNotes string                                   `gorm:"type:varchar(512);comment:审批备注"`

	CreatorID  uint `gorm:"comment:创建者ID"`
	Creator    User `gorm:"foreignKey:CreatorID"`
	ReviewerID uint `gorm:"comment:审批者ID"`
	Reviewer   User `gorm:"foreignKey:ReviewerID"`

	ReviewSource ReviewSource `gorm:"type:varchar(32);default:'';comment:审批来源(system_auto/agent_auto/admin_manual)"`
	AgentReport  string       `gorm:"type:text;default:'';comment:Agent评估报告JSON"`
}
