package orm

import (
	"sync"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var instance *gorm.DB = nil
var lock sync.Mutex

type FilePermission int

const (
	NotAllowed FilePermission = 0
	ReadOnly
	ReadWrite
)

type GormDBWrapper struct {
	*gorm.DB
}

func opendb() *GormDBWrapper {
	if instance == nil {
		lock.Lock()
		if instance == nil {
			// db_host := "192.168.5.60"
			// db_user := "postgres"
			// db_pass := "DcNuWzUh0kI2k7tZSl3Uf84LwDm0cRMSWqwcYtTbgK35g56rjenpXfUzOjy7N0zz"
			// var dia gorm.Dialector
			// dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
			// 	db_user, db_pass, db_host, "30432", "crater")
			// if db_host == "sqlite_memory" {
			// 	dia = sqlite.Open("file::memory:?cache=shared")
			// } else {
			// 	dia = postgres.Open(dsn)
			// }
			dsn := `host=192.168.5.60 user=postgres password=DcNuWzUh0kI2k7tZSl3Uf84LwDm0cRMSWqwcYtTbgK35g56rjenpXfUzOjy7N0zz 
				dbname=crater port=30432 sslmode=require TimeZone=Asia/Shanghai`
			var err error
			instance, err = gorm.Open(postgres.Open(dsn))
			if err != nil {
				logrus.WithError(err).Error("connect to postgres:")
				instance = nil
			}
		}
		lock.Unlock()
	}
	return &GormDBWrapper{DB: instance}
}

func checkdb(cnt int) *GormDBWrapper {
	ans := opendb()
	if ans == nil {
		return nil
	}
	db, err := ans.DB.DB()
	if err != nil {
		logrus.WithError(err).Error("Cannot get *sql.DB.")
		db = nil
	}
	if db != nil {
		err = db.Ping()
		if err != nil {
			logrus.WithError(err).Errorf("Ping error %d.", cnt)
			err = db.Close()
			if err != nil {
				logrus.WithError(err).Errorf("Close error %d.", cnt)
			}
			instance = nil
			ans = nil
		}
	}
	return ans
}

func DB() *GormDBWrapper {
	ans := checkdb(1)
	if ans != nil {
		return ans
	}
	instance = nil
	return checkdb(2)
}
