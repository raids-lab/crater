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
			// create `imageupload` table
			ID: "2024051216174",
			Migrate: func(tx *gorm.DB) error {
				// it's a good practice to copy the struct inside the function,
				// so side effects are prevented if the original struct changes during the time
				type ImagePack struct {
					gorm.Model
					UserID  uint
					User    model.User
					QueueID uint
					Queue   model.Queue
					//nolint:lll // need these description
					ImagePackName string                   `gorm:"column:imagepackname;uniqueIndex:imagepackname;type:varchar(128);not null" json:"imagepackname"`
					ImageLink     string                   `gorm:"column:imagelink;type:varchar(128);not null" json:"imagelink"`
					NameSpace     string                   `gorm:"column:namespace;type:varchar(128);not null" json:"namespace"`
					Status        string                   `gorm:"column:status;type:varchar(128);not null" json:"status"`
					NameTag       string                   `gorm:"column:nametag;type:varchar(128);not null" json:"nametag"`
					Params        model.ImageProfileParams `gorm:"column:params;type:varchar(512);serializer:json;" json:"params"`
					NeedProfile   bool                     `gorm:"column:needprofile;type:boolean;default:false" json:"needprofile"`
					TaskType      model.ImageTaskType      `gorm:"column:tasktype;not null;comment:作业状态" json:"tasktype"`
					Alias         string                   `gorm:"column:alias;type:varchar(128);not null" json:"alias"`
					Description   string                   `gorm:"column:description;type:varchar(512);not null" json:"description"`
					CreatorName   string                   `gorm:"column:creatorname;type:varchar(128);not null" json:"creatorname"`
				}
				return tx.Migrator().CreateTable(&ImagePack{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("imagepack")
			},
		},
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
		{
			// create `labels` table
			ID: "202405222223",
			Migrate: func(tx *gorm.DB) error {
				// it's a good practice to copy the struct inside the function,
				// so side effects are prevented if the original struct changes during the time
				type Label struct {
					gorm.Model
					Label    string           `gorm:"uniqueIndex;type:varchar(255);not null;comment:标签名"`
					Name     string           `gorm:"type:varchar(255);not null;comment:别名"`
					Resource string           `gorm:"type:varchar(255);not null;comment:资源"`
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
			&model.ImagePack{},
			&model.ImageUpload{},
			&model.Label{},
			&model.Queue{},
			&model.UserQueue{},
			&model.Dataset{},
			&model.QueueDataset{},
			&model.UserDataset{},
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
