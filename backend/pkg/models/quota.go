package models

// Quota model
type Quota struct {
	UserName  string `gorm:"primaryKey" json:"username"`
	Namespace string `gorm:"column:namespace;type:varchar(128);not null" json:"namespace"`
	HardQuota string `gorm:"column:hardQuota;type:text;" json:"hardQuota"`
}
