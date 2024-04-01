package db

import (
	"fmt"
	"time"

	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
	"github.com/raids-lab/crater/pkg/models"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Orm *gorm.DB

// todo: mysql configuration
// InitDB init mysql connection
func InitDB(dbConfig *config.Config) error {
	user := dbConfig.DBUser
	password := dbConfig.DBPassword
	dbName := dbConfig.DBName
	host := dbConfig.DBHost
	port := dbConfig.DBPort
	charset := dbConfig.DBCharset

	timeout := dbConfig.DBConnectionTimeout
	if timeout == 0 {
		timeout = 10
	}
	if charset == "" {
		charset = "utf8mb4"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local&timeout=%ds",
		user, password, host, port, dbName, charset, timeout)
	var err error
	Orm, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
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

	logutils.Log.Info("mysql init success!")
	return nil
}

// InitMigration init mysql migration
func InitMigration() error {
	if err := Orm.AutoMigrate(&models.AITask{}); err != nil {
		return fmt.Errorf("init migration AITask: %w", err)
	}
	if err := Orm.AutoMigrate(&models.Quota{}, &models.User{}, &models.ImagePack{}); err != nil {
		return fmt.Errorf("init migration User and Quota: %w", err)
	}
	if err := Orm.AutoMigrate(&models.GPU{}); err != nil {
		return fmt.Errorf("init migration GPU: %w", err)
	}
	return nil
}

//nolint:gochecknoinits // todo: init db conf
func init() {
	viper.AddConfigPath("./conf")
}
