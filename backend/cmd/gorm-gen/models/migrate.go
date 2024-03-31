// Migration script for gorm-gen
package main

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/raids-lab/crater/dao/model"
	"gorm.io/driver/mysql"
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

func ConnectMySQL() *gorm.DB {
	dsn := "root:buaak8sportal@2023mysql@tcp(***REMOVED***:30306)/crater?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("connect to postgres: %w", err))
	}
	return db
}

func main() {
	db := ConnectPostgres()

	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// your migrations here
		// {
		// 	// create `users` table
		// 	ID: "201608301400",
		// 	Migrate: func(tx *gorm.DB) error {
		// 		// it's a good practice to copy the struct inside the function,
		// 		// so side effects are prevented if the original struct changes during the time
		// 		type AIJob struct {
		// 			gorm.Model
		// 			Name     string `gorm:"type:varchar(128);not null;comment:作业名称"`
		// 			TaskType string `gorm:"type:varchar(128);not null"`
		// 		}
		// 		return tx.Migrator().CreateTable(&AIJob{})
		// 	},
		// 	Rollback: func(tx *gorm.DB) error {
		// 		return tx.Migrator().DropTable("users")
		// 	},
		// },
	})

	m.InitSchema(func(tx *gorm.DB) error {
		err := tx.AutoMigrate(
			&model.Project{},
			&model.Space{},
			&model.User{},
			&model.UserProject{},
			&model.ProjectSpace{},
			&model.Quota{},
			&model.AIJob{},
		)
		if err != nil {
			return err
		}
		// all other constraints, indexes, etc...
		return nil
	})

	if err := m.Migrate(); err != nil {
		panic(fmt.Errorf("could not migrate: %w", err))
	}
}
