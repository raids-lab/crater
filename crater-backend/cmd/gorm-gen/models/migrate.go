// Migration script for gorm-gen
package main

import (
	"fmt"
	"os"

	"github.com/raids-lab/crater/dao/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectPostgres() *gorm.DB {
	// Connect to the database
	password := os.Getenv("PGPASSWORD")
	port := os.Getenv("PGPORT")
	if password == "" || port == "" {
		panic("Please read the README.md file to set the environment variable.")
	}
	dsnPattern := "host=***REMOVED*** user=postgres password=%s dbname=crater port=%s sslmode=require TimeZone=Asia/Shanghai"
	dsn := fmt.Sprintf(dsnPattern, password, port)
	db, err := gorm.Open(postgres.Open(dsn))
	if err != nil {
		panic(fmt.Errorf("connect to postgres: %w", err))
	}
	return db
}

func ConnectMySQL() *gorm.DB {
	dsn := "root:buaak8sportal@2023mysql@tcp(***REMOVED***:30306)/crater?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("connect to postgres: %w", err))
	}
	return db
}

func main() {
	db := ConnectMySQL()
	if err := db.AutoMigrate(
		&model.Space{},
		&model.User{},
		&model.UserProject{},
		&model.ProjectSpace{},
		&model.Project{},
	); err != nil {
		panic(fmt.Errorf("auto migrate user fail: %w", err))
	}
}
