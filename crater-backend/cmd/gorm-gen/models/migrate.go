// Migration script for gorm-gen
package main

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/models"
	"github.com/raids-lab/crater/pkg/monitor"
)

func main() {
	db := query.GetDB()

	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// your migrations here
		// See https://pkg.go.dev/github.com/go-gormigrate/gormigrate/v2#Migration for details.
		//
		// {
		// 	ID: "202411182330",
		// 	Migrate: func(tx *gorm.DB) error {
		// 		type Job struct {
		// 			Template string `gorm:"type:text;comment:作业的模板配置"`
		// 		}
		// 		return tx.Migrator().AddColumn(&Job{}, "Template")
		// 	},
		// 	Rollback: func(tx *gorm.DB) error {
		// 		type Job struct {
		// 			Template string `gorm:"type:text;comment:作业的模板配置"`
		// 		}
		// 		return tx.Migrator().DropColumn(&Job{}, "Template")
		// 	},
		// },
		// {
		// 	ID: "202412131147",
		// 	Migrate: func(tx *gorm.DB) error {
		// 		type Kaniko struct {
		// 			BuildSource model.BuildSource `gorm:"type:varchar(32);not null;default:buildkit;comment:构建来源"`
		// 		}
		// 		return tx.Migrator().AddColumn(&Kaniko{}, "BuildSource")
		// 	},
		// 	Rollback: func(tx *gorm.DB) error {
		// 		type Kaniko struct {
		// 			BuildSource model.BuildSource `gorm:"type:varchar(32);not null;default:buildkit;comment:构建来源"`
		// 		}
		// 		return tx.Migrator().DropColumn(&Kaniko{}, "BuildSource")
		// 	},
		// },
		{
			ID: "202412162220", // 确保ID是唯一的
			Migrate: func(tx *gorm.DB) error {
				type Datasets struct {
					Type  model.DataType                         `gorm:"type:varchar(32);not null;default:dataset;comment:数据类型"`
					Extra datatypes.JSONType[model.Extracontent] `gorm:"comment:额外信息(tags、weburl等)"`
				}
				if err := tx.Migrator().AddColumn(&Datasets{}, "Type"); err != nil {
					return err
				}
				return tx.Migrator().AddColumn(&Datasets{}, "Extra")
			},
			Rollback: func(tx *gorm.DB) error {
				type Datasets struct {
					Type  model.DataType                         `gorm:"type:varchar(32);not null;default:dataset;comment:数据类型"`
					Extra datatypes.JSONType[model.Extracontent] `gorm:"comment:额外信息(tags、weburl等)"`
				}
				if err := tx.Migrator().DropColumn(&Datasets{}, "Extra"); err != nil {
					return err
				}
				return tx.Migrator().DropColumn(&Datasets{}, "Type")
			},
		},
		{
			ID: "202412241200", // 确保ID是唯一的
			Migrate: func(tx *gorm.DB) error {
				type Job struct {
					AlertEnabled bool `gorm:"type:boolean;default:true;comment:是否启用通知"`
				}
				return tx.Migrator().AddColumn(&Job{}, "AlertEnabled")
			},
			Rollback: func(tx *gorm.DB) error {
				type Job struct {
					AlertEnabled bool `gorm:"type:boolean;default:true;comment:是否启用通知"`
				}
				return tx.Migrator().DropColumn(&Job{}, "AlertEnabled")
			},
		},
		{
			ID: "202503061740",
			Migrate: func(tx *gorm.DB) error {
				type Job struct {
					ProfileData datatypes.JSONType[*monitor.ProfileData] `gorm:"comment:作业的性能数据"`
				}
				return tx.Migrator().AddColumn(&Job{}, "ProfileData")
			},
			Rollback: func(tx *gorm.DB) error {
				type Job struct {
					ProfileData datatypes.JSONType[*monitor.ProfileData] `gorm:"comment:作业的性能数据"`
				}
				return tx.Migrator().DropColumn(&Job{}, "ProfileData")
			},
		},
		{
			ID: "202503251830",
			Migrate: func(tx *gorm.DB) error {
				type JobTemplate struct {
					gorm.Model
					Name     string `gorm:"not null;type:varchar(256)"`
					Describe string `gorm:"type:varchar(512)"`
					Document string `gorm:"type:text"`
					Template string `gorm:"type:text"`
					UserID   uint   `gorm:"index"`
					User     model.User
				}

				// 明确指定表名
				if err := tx.Table("jobtemplates").Migrator().CreateTable(&JobTemplate{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("jobtemplates") // 删除 jobtemplates 表
			},
		},
	})

	m.InitSchema(func(tx *gorm.DB) error {
		err := tx.AutoMigrate(
			&model.User{},
			&model.Label{},
			&model.Account{},
			&model.UserAccount{},
			&model.Dataset{},
			&model.AccountDataset{},
			&model.UserDataset{},
			&model.Resource{},
			&model.Job{},
			&models.AITask{},
			&model.Kaniko{},
			&model.Image{},
			&model.Jobtemplate{},
		)
		if err != nil {
			return err
		}

		// create default account
		account := model.Account{
			Name:     "default",
			Nickname: "公共账户",
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
			Name:     name,
			Nickname: "管理员",
			Password: ptr.To(string(hashedPassword)),
			Role:     model.RoleAdmin, // todo: change to model.RoleUser
			Status:   model.StatusActive,
			Space:    "u-admin",
			Attributes: datatypes.NewJSONType(model.UserAttribute{
				ID:       1,
				Name:     name,
				Nickname: "管理员",
				Email:    ptr.To("***REMOVED***"),
				Teacher:  ptr.To("管理员"),
				Group:    ptr.To("管理员"),
				UID:      ptr.To("1001"),
				GID:      ptr.To("1001"),
			}),
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
