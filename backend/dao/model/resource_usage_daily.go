package model

import "time"

// ResourceUsageDaily 保存一个 UTC 自然日内可累加的资源用量。CPU 以核、内存以 GiB 计，其他资源使用各自的单位。
// ResourceName 为空的行记录当日初始化和统计进度。
type ResourceUsageDaily struct {
	StatDate              time.Time `gorm:"type:date;primaryKey;comment:统计日期(UTC)"`
	UserID                uint      `gorm:"primaryKey;autoIncrement:false;index:idx_daily_usage_user;comment:用户ID"`
	AccountID             uint      `gorm:"primaryKey;autoIncrement:false;index:idx_daily_usage_account;comment:账户ID"`
	ResourceName          string    `gorm:"type:varchar(255);primaryKey;comment:资源名称(空串表示日进度标记)"`
	UsageSeconds          float64   `gorm:"type:numeric(30,9);not null;comment:分配用量(资源单位秒)"`
	EffectiveUsageSeconds float64   `gorm:"type:numeric(30,9);not null;comment:估算有效用量(资源单位秒)"`
	ProfiledUsageSeconds  float64   `gorm:"type:numeric(30,9);not null;comment:已评估分配用量(资源单位秒)"`
	CoveredUntil          time.Time `gorm:"not null;comment:统计覆盖截止时间"`
	IsComplete            bool      `gorm:"not null;default:false;comment:当日统计是否完成"`
	UpdatedAt             time.Time `gorm:"not null;autoUpdateTime:false;comment:统计水位更新时间"`
}

func (ResourceUsageDaily) TableName() string { return "resource_usage_daily" }
