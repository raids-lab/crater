package models

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

type User struct {
	UserName string `gorm:"primaryKey" json:"username"`
	Role     string `gorm:"column:role;type:varchar(128);not null" json:"role"`
}
