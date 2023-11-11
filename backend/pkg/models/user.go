package models

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

type Model struct {
	ID uint `gorm:"primarykey;"`
}
type User struct {
	Model
	UserName  string `json:"username"`
	Role      string `gorm:"column:role;type:varchar(128);not null" json:"role"`
	Password  string `json:"password"`
	NameSpace string
}
