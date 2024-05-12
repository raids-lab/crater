// Migration script for gorm-gen
package main

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/raids-lab/crater/dao/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectPostgres() *gorm.DB {
	// Connect to the database
	dsn := `host=***REMOVED*** user=postgres password=***REMOVED*** 
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
			// create `labels` table
			ID: "202405121603",
			Migrate: func(tx *gorm.DB) error {
				// it's a good practice to copy the struct inside the function,
				// so side effects are prevented if the original struct changes during the time
				type Label struct {
					gorm.Model
					Label    string           `gorm:"uniqueIndex;type:varchar(256);not null;comment:标签名"`
					Name     string           `gorm:"type:varchar(256);not null;comment:别名"`
					Type     model.WorkerType `gorm:"not null;comment:类型"`
					Count    int              `gorm:"not null;comment:节点数量"`
					Priority int              `gorm:"not null;comment:优先级"`
				}
				return tx.Migrator().CreateTable(&Label{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("labels")
			},
		},
	})

	m.InitSchema(func(tx *gorm.DB) error {
		err := tx.AutoMigrate(
			&model.Project{},
			&model.Space{},
			&model.User{},
			&model.UserProject{},
			&model.ProjectSpace{},
			&model.AIJob{},
			&model.Image{},
			&model.Label{},
			&model.Queue{},
			&model.UserQueue{},
		)
		if err != nil {
			return err
		}

		queue := model.Queue{
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
