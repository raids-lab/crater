package model

import (
	"time"
)

// UserSpaceSize 用户空间大小模型
type UserSpaceSize struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	Username  string    `gorm:"size:64;not null;uniqueIndex" json:"username"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TenantUsageHistory 租户存储使用历史
type TenantUsageHistory struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TenantID   uint      `gorm:"index" json:"tenant_id"`
	UsageBytes int64     `json:"usage_bytes"`
	RecordedAt time.Time `json:"recorded_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
