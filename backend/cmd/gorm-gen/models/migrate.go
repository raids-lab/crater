// Migration script for gorm-gen
package main

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/google/uuid"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/samber/lo"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/pkg/models"
)

func main() {
	db := query.GetDB()

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
					UserID    uint
					User      model.User
					AccountID uint
					Queue     model.Account
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
					AccountID uint `gorm:"primaryKey"`
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
		{
			// create `labels` table
			ID: "202405240951",
			Migrate: func(tx *gorm.DB) error {
				// it's a good practice to copy the struct inside the function,
				// so side effects are prevented if the original struct changes during the time
				return tx.Migrator().CreateTable(&model.Resource{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("resources")
			},
		},
		{
			// add `quota` column to `users` table
			ID: "202406181403",
			Migrate: func(tx *gorm.DB) error {
				// when table already exists, define only columns that are about to change
				type Queue struct {
					Quota datatypes.JSONType[model.QueueQuota] `gorm:"comment:资源配额"`
				}
				return tx.Migrator().AddColumn(&Queue{}, "Quota")
			},
			Rollback: func(tx *gorm.DB) error {
				type Queue struct {
					Quota datatypes.JSONType[model.QueueQuota] `gorm:"comment:资源配额"`
				}
				return tx.Migrator().DropColumn(&Queue{}, "Quota")
			},
		},
		{
			// add `quota` column to `users` table
			ID: "202406182314",
			Migrate: func(tx *gorm.DB) error {
				// when table already exists, define only columns that are about to change
				type AITask struct {
					Owner string `json:"owner"`
				}
				return tx.Migrator().AddColumn(&AITask{}, "Owner")
			},
			Rollback: func(tx *gorm.DB) error {
				type AITask struct {
					Owner string `json:"owner"`
				}
				return tx.Migrator().DropColumn(&AITask{}, "Owner")
			},
		},
		{
			// create `jobs` table
			ID: "202408310951",
			Migrate: func(tx *gorm.DB) error {
				// it's a good practice to copy the struct inside the function,
				// so side effects are prevented if the original struct changes during the time
				return tx.Migrator().CreateTable(&model.Job{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("jobs")
			},
		},
		{
			ID: "202410012314",
			Migrate: func(tx *gorm.DB) error {
				type Job struct {
					KeepWhenLowResourceUsage bool `gorm:"comment:当资源利用率低时是否保留"`
				}
				return tx.Migrator().AddColumn(&Job{}, "KeepWhenLowResourceUsage")
			},
			Rollback: func(tx *gorm.DB) error {
				type Job struct {
					KeepWhenLowResourceUsage bool `gorm:"comment:当资源利用率低时是否保留"`
				}
				return tx.Migrator().DropColumn(&Job{}, "KeepWhenLowResourceUsage")
			},
		},
		{
			ID: "202410022314",
			Migrate: func(tx *gorm.DB) error {
				// when table already exists, define only columns that are about to change
				type AITask struct {
					KeepWhenLowResourceUsage bool `gorm:"comment:当资源利用率低时是否保留"`
				}
				return tx.Migrator().AddColumn(&AITask{}, "KeepWhenLowResourceUsage")
			},
			Rollback: func(tx *gorm.DB) error {
				type AITask struct {
					KeepWhenLowResourceUsage bool `gorm:"comment:当资源利用率低时是否保留"`
				}
				return tx.Migrator().DropColumn(&AITask{}, "KeepWhenLowResourceUsage")
			},
		},
		{
			// add `ispublic` column to `Image` table
			ID: "202410101955",
			Migrate: func(tx *gorm.DB) error {
				// when table already exists, define only columns that are about to change
				type ImagePack struct {
					IsPublic bool `gorm:"column:ispublic;type:boolean;default:false" json:"ispublic"`
				}
				return tx.Migrator().AddColumn(&ImagePack{}, "IsPublic")
			},
			Rollback: func(tx *gorm.DB) error {
				type ImagePack struct {
					IsPublic bool `gorm:"column:ispublic;type:boolean;default:false" json:"ispublic"`
				}
				return tx.Migrator().DropColumn(&ImagePack{}, "IsPublic")
			},
		},
		{
			// add `ispublic` column to `Image` table
			ID: "202410102007",
			Migrate: func(tx *gorm.DB) error {
				// when table already exists, define only columns that are about to change
				type ImageUpload struct {
					IsPublic bool `gorm:"column:ispublic;type:boolean;default:false" json:"ispublic"`
				}
				return tx.Migrator().AddColumn(&ImageUpload{}, "IsPublic")
			},
			Rollback: func(tx *gorm.DB) error {
				type ImageUpload struct {
					IsPublic bool `gorm:"column:ispublic;type:boolean;default:false" json:"ispublic"`
				}
				return tx.Migrator().DropColumn(&ImageUpload{}, "IsPublic")
			},
		},
		{
			// add user quota json column to userQueue table
			ID: "202410131759",
			Migrate: func(tx *gorm.DB) error {
				type UserQueue struct {
					Quota datatypes.JSONType[model.QueueQuota] `gorm:"comment:用户在队列中的资源配额"`
				}
				return tx.Migrator().AddColumn(&UserQueue{}, "Quota")
			},
			Rollback: func(tx *gorm.DB) error {
				type UserQueue struct {
					Quota datatypes.JSONType[model.QueueQuota] `gorm:"comment:用户在队列中的资源配额"`
				}
				return tx.Migrator().DropColumn(&UserQueue{}, "Quota")
			},
		},
		{
			// add some columns to job table
			ID: "202410132006",
			Migrate: func(tx *gorm.DB) error {
				type Job struct {
					IsPublic bool `gorm:"comment:是否公开"`
				}
				return tx.Migrator().AddColumn(&Job{}, "IsPublic")
			},
			Rollback: func(tx *gorm.DB) error {
				type Job struct {
					IsPublic bool `gorm:"comment:是否公开"`
				}
				return tx.Migrator().DropColumn(&Job{}, "IsPublic")
			},
		},
		{
			// add one column to ImagePack table
			ID: "202410151052",
			Migrate: func(tx *gorm.DB) error {
				// when table already exists, define only columns that are about to change
				type ImagePack struct {
					Size int64 `gorm:"column:size;type:bigint;default:0" json:"size"`
				}
				return tx.Migrator().AddColumn(&ImagePack{}, "Size")
			},
			Rollback: func(tx *gorm.DB) error {
				type ImagePack struct {
					Size int64 `gorm:"column:size;type:bigint;default:0" json:"size"`
				}
				return tx.Migrator().DropColumn(&ImagePack{}, "Size")
			},
		},
		{
			// add one column to User table
			ID: "202410151153",
			Migrate: func(tx *gorm.DB) error {
				// when table already exists, define only columns that are about to change
				type User struct {
					ImageQuota int64 `gorm:"type:bigint;default:-1;comment:用户在镜像仓库的配额"`
				}
				return tx.Migrator().AddColumn(&User{}, "ImageQuota")
			},
			Rollback: func(tx *gorm.DB) error {
				type User struct {
					ImageQuota int64 `gorm:"type:bigint;default:-1;comment:用户在镜像仓库的配额"`
				}
				return tx.Migrator().DropColumn(&User{}, "ImageQuota")
			},
		},
		{
			// create `jobs` table
			ID: "202410311449",
			Migrate: func(tx *gorm.DB) error {
				// it's a good practice to copy the struct inside the function,
				// so side effects are prevented if the original struct changes during the time
				return tx.Migrator().CreateTable(&model.Kaniko{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("kanikos")
			},
		},
		{
			// create `jobs` table
			ID: "202410311450",
			Migrate: func(tx *gorm.DB) error {
				// it's a good practice to copy the struct inside the function,
				// so side effects are prevented if the original struct changes during the time
				return tx.Migrator().CreateTable(&model.Image{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("images")
			},
		},
		{
			ID: "202411071606",
			Migrate: func(tx *gorm.DB) error {
				type Image struct {
					ImagePackName string `gorm:"uniqueIndex:imagepackname;type:varchar(128);not null;comment:ImagePack CRD名称"`
				}
				return tx.Migrator().AddColumn(&Image{}, "ImagePackName")
			},
			Rollback: func(tx *gorm.DB) error {
				type Image struct {
					ImagePackName string `gorm:"uniqueIndex:imagepackname;type:varchar(128);not null;comment:ImagePack CRD名称"`
				}
				return tx.Migrator().DropColumn(&Image{}, "ImagePackName")
			},
		},
	})

	m.InitSchema(func(tx *gorm.DB) error {
		err := tx.AutoMigrate(
			&model.User{},
			&model.ImagePack{},
			&model.ImageUpload{},
			&model.Label{},
			&model.Account{},
			&model.UserAccount{},
			&model.Dataset{},
			&model.AccountDataset{},
			&model.UserDataset{},
			&model.Resource{},
			&model.Job{},
			&models.AITask{},
			&model.Whitelist{},
			&model.Kaniko{},
			&model.Image{},
		)
		if err != nil {
			return err
		}

		// create default account
		account := model.Account{
			Name:     "default",
			Nickname: "公共队列",
			Space:    "/public",
			Quota:    datatypes.NewJSONType(model.QueueQuota{}),
		}

		res := tx.Create(&account)
		if res.Error != nil {
			return res.Error
		}

		// create default admin user, add to default queue
		// 1. generate a random name and password
		name := fmt.Sprintf("admin%s", uuid.New().String()[:5])
		password := uuid.New().String()[:8]
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		// 2. create a user with the name and password
		user := model.User{
			Name:       name,
			Nickname:   "管理员",
			Password:   lo.ToPtr(string(hashedPassword)),
			Role:       model.RoleAdmin, // todo: change to model.RoleUser
			Status:     model.StatusActive,
			Space:      fmt.Sprintf("u-%s", name),
			Attributes: datatypes.NewJSONType(model.UserAttribute{}),
		}

		res = tx.Create(&user)
		if res.Error != nil {
			return res.Error
		}

		// 3. add the user to the default queue
		userQueue := model.UserAccount{
			UserID:     user.ID,
			AccountID:  account.ID,
			Role:       model.RoleAdmin,
			AccessMode: model.AccessModeRW,
		}

		res = tx.Create(&userQueue)
		if res.Error != nil {
			return res.Error
		}

		// 4. print the name and password
		fmt.Printf(`Default admin user created:
	Name: %s
	Password: %s
		`, name, password)

		return nil
	})

	if err := m.Migrate(); err != nil {
		panic(fmt.Errorf("could not migrate: %w", err))
	}
}
