package model

import (
	"time"

	"gorm.io/datatypes"
)

type UserBanAction string

const (
	UserBanActionBan    UserBanAction = "ban"
	UserBanActionExtend UserBanAction = "extend"
	UserBanActionUnban  UserBanAction = "unban"
)

// UserBanRecord stores the administrator audit trail for user ban state changes.
type UserBanRecord struct {
	ID              uint                                    `gorm:"primarykey"`
	CreatedAt       time.Time                               `gorm:"index"`
	UserID          uint                                    `gorm:"index;not null"`
	UserName        string                                  `gorm:"type:varchar(64);not null"`
	OperatorID      uint                                    `gorm:"index;not null"`
	OperatorName    string                                  `gorm:"type:varchar(64);not null"`
	Action          UserBanAction                           `gorm:"type:varchar(16);not null"`
	BannedTimestamp *time.Time                              `gorm:"comment:操作后的封禁截止时间，为空表示解除封禁"`
	BanRestrictions datatypes.JSONType[UserBanRestrictions] `gorm:"type:jsonb;not null;default:'{}';comment:操作后的封禁限制内容"`
	Reason          string                                  `gorm:"type:text;not null"`
}
