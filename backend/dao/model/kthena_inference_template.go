package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// KthenaInferenceTemplate is a private reusable deployment preset. It is
// intentionally scoped by both user and account, so a template never becomes
// visible when a browser switches identity or account context.
//
// Config keeps the complete form payload as JSON. The deployment API remains
// authoritative when a template is eventually submitted, while templates can
// evolve without a schema migration for every new form field.
//
//nolint:lll // Composite GORM index declarations must stay in a single struct tag.
type KthenaInferenceTemplate struct {
	gorm.Model

	UserID      uint           `gorm:"not null;uniqueIndex:idx_kthena_inference_template_scope,priority:1;index:idx_kthena_inference_template_list,priority:1;comment:模板所属用户ID"`
	AccountID   uint           `gorm:"not null;uniqueIndex:idx_kthena_inference_template_scope,priority:2;index:idx_kthena_inference_template_list,priority:2;comment:模板所属账户ID"`
	Name        string         `gorm:"type:varchar(64);not null;uniqueIndex:idx_kthena_inference_template_scope,priority:3;comment:模板名称"`
	Description string         `gorm:"type:varchar(512);not null;default:'';comment:模板说明"`
	Config      datatypes.JSON `gorm:"type:jsonb;not null;comment:模型部署表单配置"`
}

func (KthenaInferenceTemplate) TableName() string {
	return "kthena_inference_templates"
}
