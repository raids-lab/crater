package model

import "time"

type UserBanAction string

const (
	UserBanActionBan   UserBanAction = "ban"
	UserBanActionUnban UserBanAction = "unban"
)

// UserBanRecord stores the administrator audit trail for user ban state changes.
type UserBanRecord struct {
	ID           uint          `gorm:"primarykey"`
	CreatedAt    time.Time     `gorm:"index"`
	UserID       uint          `gorm:"index;not null"`
	UserName     string        `gorm:"type:varchar(64);not null"`
	OperatorID   uint          `gorm:"index;not null"`
	OperatorName string        `gorm:"type:varchar(64);not null"`
	Action       UserBanAction `gorm:"type:varchar(16);not null"`
	Reason       string        `gorm:"type:text;not null"`
}
