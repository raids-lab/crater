package models

type User struct {
	UserName string `gorm:"primaryKey" json:"username"`
}
