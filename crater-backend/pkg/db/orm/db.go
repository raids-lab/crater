package db

import (
	"fmt"
	"time"

	"github.com/aisystem/ai-protal/pkg/models"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Orm *gorm.DB

// todo: mysql configuration
// InitDB init mysql connection
func InitDB(configFile string) error {
	if configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			// 配置文件出错
			return err
		}
	}

	user := viper.GetString("DB_USER")
	password := viper.GetString("DB_PASSWORD")
	dbName := viper.GetString("DB_NAME")
	host := viper.GetString("DB_HOST")
	port := viper.GetString("DB_PORT")
	charset := viper.GetString("DB_CHARSET")

	timeout := viper.GetUint32("DB_CONN_TIMEOUT")
	if timeout == 0 {
		timeout = 10
	}
	if charset == "" {
		charset = "utf8mb4"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local&timeout=%ds", user, password, host, port, dbName, charset, timeout)
	// log.Infof("dsn: %s", dsn)
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

	log.Info("mysql init success!")
	return nil
}

// InitMigration init mysql migration
func InitMigration() error {
	if err := Orm.AutoMigrate(&models.AITask{}); err != nil {
		return fmt.Errorf("init migration AITask err: %v", err)
	}
	if err := Orm.AutoMigrate(&models.User{}); err != nil {
		return fmt.Errorf("init migration User err: %v", err)
	}
	if err := Orm.AutoMigrate(&models.Quota{}); err != nil {
		return fmt.Errorf("init migration Quota err: %v", err)
	}
	return nil
}

// todo: init db conf
func init() {
	viper.AddConfigPath("./conf")
}
