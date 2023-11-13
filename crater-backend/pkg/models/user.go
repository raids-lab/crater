package models

import "time"

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

type Model struct {
	ID        uint      `gorm:"primarykey;"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime" json:"updatedAt"`
}
type User struct {
	Model
	UserName  string `gorm:"column:username;uniqueIndex:username;type:varchar(128);not null" json:"username"`
	Role      string `gorm:"column:role;type:varchar(128);not null" json:"role"`
	Password  string `json:"password"`
	NameSpace string `gorm:"column:namespace;type:varchar(128);not null" json:"namespace"`
}

type UserQuota struct {
	User  User  `gorm:"embedded"`
	Quota Quota `gorm:"embedded"`
}

func (User) TableName() string {
	return "users"
}
