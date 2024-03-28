// Migration script for gorm-gen
package main

import (
	"fmt"

	"github.com/raids-lab/crater/pkg/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

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
	if err := db.AutoMigrate(&model.Project{}, &model.User{}, &model.UserProject{}); err != nil {
		panic(fmt.Errorf("auto migrate user fail: %w", err))
	}
}
