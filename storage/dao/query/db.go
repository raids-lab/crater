package query

import (
	"fmt"
	"time"

	"webdav/config"
	"webdav/logutils"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init postgres connection
func InitDB() error {
	dbConfig := config.GetConfig()

	host := dbConfig.Postgres.Host
	port := dbConfig.Postgres.Port
	dbName := dbConfig.Postgres.DBName
	user := dbConfig.Postgres.User
	password := dbConfig.Postgres.Password
	sslMode := dbConfig.Postgres.SSLMode
	timeZone := dbConfig.Postgres.TimeZone

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		host, user, password, dbName, port, sslMode, timeZone)
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	maxIdleConns := 5
	maxOpenConns := 10
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logutils.Log.Info("Postgres init success!")
	return nil
}
