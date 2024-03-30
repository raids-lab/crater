package query

import (
	"fmt"
	"time"

	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB init postgres connection
func InitDB(dbConfig *config.Config) error {
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
