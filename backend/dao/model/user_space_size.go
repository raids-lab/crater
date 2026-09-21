package model

import (
	"time"
)

// UserSpaceSize stores the latest successfully observed usage for a user directory.
type UserSpaceSize struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
