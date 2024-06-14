package orm

import (
	"fmt"
	"time"

	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var Orm *gorm.DB

// todo: mysql configuration
// InitDB init mysql connection
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
	Orm, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	maxIdleConns := 5
	maxOpenConns := 10
	sqlDB, err := Orm.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logutils.Log.Info("Postgres init success!")
	return nil
}

// InitMigration init mysql migration
func InitMigration() error {
	if err := Orm.AutoMigrate(&models.AITask{}); err != nil {
		return fmt.Errorf("init migration AITask: %w", err)
	}
	if err := Orm.AutoMigrate(&models.Quota{}, &models.User{}); err != nil {
		return fmt.Errorf("init migration User and Quota: %w", err)
	}
	return nil
}
