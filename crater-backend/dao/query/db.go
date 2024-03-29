package query

import (
	"fmt"
	"os"
	"time"

	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() error {
	// Connect to the database
	password := os.Getenv("PGPASSWORD")
	port := os.Getenv("PGPORT")
	if password == "" || port == "" {
		return fmt.Errorf("please read the README.md file to set the environment variable")
	}
	dsnPattern := "host=localhost user=postgres password=%s dbname=crater port=%s sslmode=require TimeZone=Asia/Shanghai"
	dsn := fmt.Sprintf(dsnPattern, password, port)
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
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
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

	logutils.Log.Info("MySQL init success!")
	return nil
}
