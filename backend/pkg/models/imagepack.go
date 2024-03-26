package models

type ImagePack struct {
	Model
	ImagePackName string `gorm:"column:imagepackname;uniqueIndex:imagepackname;type:varchar(128);not null" json:"imagepackname"`
	UserName      string `gorm:"column:username;type:varchar(128);not null" json:"username"`
	ImageLink     string `gorm:"column:imagelink;type:varchar(128);not null" json:"imagelink"`
	NameSpace     string `gorm:"column:namespace;type:varchar(128);not null" json:"namespace"`
	Status        string `gorm:"column:status;type:varchar(128);not null" json:"status"`
	NameTag       string `gorm:"column:nametag;type:varchar(128);not null" json:"nametag"`
}
