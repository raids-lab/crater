package db

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"k8s.io/ai-task-controller/pkg/models"
)

var Orm *gorm.DB

func InitDB() {
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
	log.Infof("dsn: %s", dsn)
	var err error
	Orm, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Panic(err)
	}
	maxIdleConns := 5
	maxOpenConns := 10
	sqlDB, err := Orm.DB()
	if err != nil {
		log.Panic(err)
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Info("mysql init success!")
}

func InitMigration() {
	if err := Orm.AutoMigrate(&models.TaskModel{}); err != nil {
		log.Fatal(err)
	}
}
