package orm

import (
	"sync"

	"webdav/logutils"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var instance *gorm.DB = nil
var once sync.Once

type FilePermission int

const (
	NotAllowed FilePermission = 0
	ReadOnly
	ReadWrite
)

type GormDBWrapper struct {
	*gorm.DB
}

func opendb() *gorm.DB {
	once.Do(func() {
		if instance == nil {
			dsn := `host=192.168.5.60 user=postgres password=DcNuWzUh0kI2k7tZSl3Uf84LwDm0cRMSWqwcYtTbgK35g56rjenpXfUzOjy7N0zz 
				dbname=crater port=30432 sslmode=require TimeZone=Asia/Shanghai`
			var err error
			instance, err = gorm.Open(postgres.Open(dsn))
			if err != nil {
				logutils.Log.Fatalf("connect to postgres")
				instance = nil
			}
		}
	})
	return instance
}

func DB() *gorm.DB {
	ans := opendb()
	if ans == nil {
		logutils.Log.Fatalf("connect to postgres")
	}
	return ans
}
