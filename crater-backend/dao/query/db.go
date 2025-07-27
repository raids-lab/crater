package query

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/pkg/config"
)

var (
	once     sync.Once
	instance *gorm.DB
)

// GetDB returns the singleton instance of the database connection.
func GetDB() *gorm.DB {
	once.Do(func() {
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
		instance, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			panic(err)
		}
		maxIdleConns := 5
		maxOpenConns := 10
		sqlDB, err := instance.DB()
		if err != nil {
			panic(err)
		}
		sqlDB.SetMaxIdleConns(maxIdleConns)
		sqlDB.SetMaxOpenConns(maxOpenConns)
		sqlDB.SetConnMaxLifetime(time.Hour)

		klog.Info("Postgres init success!")
	})
	return instance
}
