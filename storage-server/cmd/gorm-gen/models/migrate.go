// Migration script for gorm-gen
package main

import (
	"fmt"

	"webdav/dao/model"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectPostgres() *gorm.DB {
	// Connect to the database
	dsn := `host=192.168.5.60 user=postgres password=DcNuWzUh0kI2k7tZSl3Uf84LwDm0cRMSWqwcYtTbgK35g56rjenpXfUzOjy7N0zz 
		dbname=crater port=30432 sslmode=require TimeZone=Asia/Shanghai`
	db, err := gorm.Open(postgres.Open(dsn))
	if err != nil {
		panic(fmt.Errorf("connect to postgres: %w", err))
	}
	return db
}

func main() {
	db := ConnectPostgres()

	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// your migrations here
		{
			// create `datasets,userdatasets,queuedatasets` table
			ID: "2024052221486",
			Migrate: func(tx *gorm.DB) error {
				// it's a good practice to copy the struct inside the function,
				// so side effects are prevented if the original struct changes during the time
				type Dataset struct {
					gorm.Model
					Name     string `gorm:"uniqueIndex;type:varchar(256);not null;comment:数据集名"`
					URL      string `gorm:"type:varchar(512);not null;comment:数据集空间路径"`
					Describe string `gorm:"type:text;comment:数据集描述"`
					UserID   uint
				}
				type UserDataset struct {
					gorm.Model
					UserID    uint `gorm:"primaryKey"`
					DatasetID uint `gorm:"primaryKey"`
				}

				type QueueDataset struct {
					gorm.Model
					QueueID   uint `gorm:"primaryKey"`
					DatasetID uint `gorm:"primaryKey"`
				}

				return tx.Migrator().CreateTable(&Dataset{}, &UserDataset{}, &QueueDataset{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("dataset", "userdataset", "queuedataset")
			},
		},
	})

	m.InitSchema(func(tx *gorm.DB) error {
		err := tx.AutoMigrate(
			&model.User{},
			&model.Account{},
			&model.UserAccount{},
			&model.Dataset{},
			&model.AccountDataset{},
			&model.UserDataset{},
		)
		if err != nil {
			return err
		}

		queue := model.Account{
			Name:     "default",
			Nickname: "公共队列",
			Space:    "/public",
		}

		res := tx.Create(&queue)
		if res.Error != nil {
			return res.Error
		}

		return nil
	})

	if err := m.Migrate(); err != nil {
		panic(fmt.Errorf("could not migrate: %w", err))
	}
}
