// Migration script for gorm-gen
package main

import (
	"fmt"
	"os"

	"github.com/raids-lab/crater/dao/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectDB() *gorm.DB {
	// Connect to the database
	password := os.Getenv("PGPASSWORD")
	port := os.Getenv("PGPORT")
	if password == "" || port == "" {
		panic("Please read the README.md file to set the environment variable.")
	}
	dsnPattern := "host=localhost user=postgres password=%s dbname=crater port=%s sslmode=require TimeZone=Asia/Shanghai"
	dsn := fmt.Sprintf(dsnPattern, password, port)
	db, err := gorm.Open(postgres.Open(dsn))
	if err != nil {
		panic(fmt.Errorf("connect to postgres: %w", err))
	}
	return db
}

func main() {
	db := ConnectDB()
	if err := db.AutoMigrate(&model.Project{}, &model.User{}, &model.UserProject{}); err != nil {
		panic(fmt.Errorf("auto migrate user fail: %w", err))
	}
}
