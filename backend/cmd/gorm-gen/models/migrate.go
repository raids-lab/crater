// Migration script for gorm-gen
package main

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Project is a group of users, with project-level quota
type Project struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex;type:varchar(32);comment:项目名"`
	Description string `gorm:"type:varchar(128);comment:项目描述"`
	NameSpace   string `gorm:"column:namespace;type:varchar(128);comment:命名空间"`
	Status      string `gorm:"type:varchar(32);comment:项目状态 (active, inactive)"`
	Quota       string `gorm:"type:text;comment:配额信息"`
}

// User is the basic entity of the system
type User struct {
	gorm.Model
	Name      string    `gorm:"uniqueIndex;type:varchar(32);not null;comment:用户名"`
	Nickname  *string   `gorm:"type:varchar(32);comment:昵称"`
	Password  *string   `gorm:"type:varchar(128);comment:密码"`
	Role      string    `gorm:"type:varchar(32);not null;comment:集群角色 (admin, user, guest)"`
	NameSpace string    `gorm:"column:namespace;type:varchar(128);not null;comment:命名空间"`
	Status    string    `gorm:"type:varchar(32);not null;comment:用户状态 (active, inactive)"`
	Projects  []Project `gorm:"many2many:user_projects;"`
}

// UserProject is the many-to-many relationship between User and Project
type UserProject struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    uint   `gorm:"primaryKey"`
	ProjectID uint   `gorm:"primaryKey"`
	Role      string `gorm:"type:varchar(32);comment:用户在项目中的角色 (admin, user)"`
	Quota     string `gorm:"type:text;comment:配额限制"`
}

const MySQLDSN = "root:buaak8sportal@2023mysql@tcp(***REMOVED***:30306)/crater?charset=utf8mb4&parseTime=True"

func ConnectDB(dsn string) *gorm.DB {
	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		panic(fmt.Errorf("connect db fail: %w", err))
	}
	return db
}

func main() {
	db := ConnectDB(MySQLDSN)
	err := db.SetupJoinTable(&User{}, "Projects", &UserProject{})
	if err != nil {
		panic(fmt.Errorf("setup join table user_project fail: %w", err))
	}
	if err := db.AutoMigrate(&Project{}, &User{}); err != nil {
		panic(fmt.Errorf("auto migrate user fail: %w", err))
	}
}
