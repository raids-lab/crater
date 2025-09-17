package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/monitor"
)

//nolint:gocyclo // ignore cyclomatic complexity
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
					Extra datatypes.JSONType[model.ExtraContent] `gorm:"comment:额外信息(tags、weburl等)"`
				}
				if err := tx.Migrator().AddColumn(&Datasets{}, "Type"); err != nil {
					return err
				}
				return tx.Migrator().AddColumn(&Datasets{}, "Extra")
			},
			Rollback: func(tx *gorm.DB) error {
				type Datasets struct {
					Type  model.DataType                         `gorm:"type:varchar(32);not null;default:dataset;comment:数据类型"`
					Extra datatypes.JSONType[model.ExtraContent] `gorm:"comment:额外信息(tags、weburl等)"`
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
		{
			ID: "202504050201", // 确保ID是唯一的
			Migrate: func(tx *gorm.DB) error {
				type Job struct {
					LockedTimestamp time.Time `gorm:"comment:作业锁定时间"`
				}
				return tx.Migrator().AddColumn(&Job{}, "LockedTimestamp")
			},
			Rollback: func(tx *gorm.DB) error {
				type Job struct {
					LockedTimestamp time.Time `gorm:"comment:作业锁定时间"`
				}
				return tx.Migrator().DropColumn(&Job{}, "LockedTimestamp")
			},
		},
		{
			ID: "202504061413", // 确保ID是唯一的
			Migrate: func(tx *gorm.DB) error {
				type User struct {
					LastEmailVerifiedAt time.Time `gorm:"comment:最后一次邮箱验证时间"`
				}
				return tx.Migrator().AddColumn(&User{}, "LastEmailVerifiedAt")
			},
			Rollback: func(tx *gorm.DB) error {
				type User struct {
					LastEmailVerifiedAt time.Time `gorm:"comment:最后一次邮箱验证时间"`
				}
				return tx.Migrator().DropColumn(&User{}, "LastEmailVerifiedAt")
			},
		},
		{
			ID: "202504112350", // 确保ID是唯一的
			//nolint:dupl // ignore duplicate code
			Migrate: func(tx *gorm.DB) error {
				type Job struct {
					ScheduleData     *datatypes.JSONType[*model.ScheduleData]           `gorm:"comment:作业的调度数据"`
					Events           *datatypes.JSONType[[]v1.Event]                    `gorm:"comment:作业的事件 (运行时、失败时采集)"`
					TerminatedStates *datatypes.JSONType[[]v1.ContainerStateTerminated] `gorm:"comment:作业的终止状态 (运行时、失败时采集)"`
				}
				if err := tx.Migrator().AddColumn(&Job{}, "ScheduleData"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&Job{}, "Events"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&Job{}, "TerminatedStates"); err != nil {
					return err
				}
				return nil
			},
			//nolint:dupl // ignore duplicate code
			Rollback: func(tx *gorm.DB) error {
				type Job struct {
					ScheduleData     *datatypes.JSONType[*model.ScheduleData]           `gorm:"comment:作业的调度数据"`
					Events           *datatypes.JSONType[[]v1.Event]                    `gorm:"comment:作业的事件 (运行时、失败时采集)"`
					TerminatedStates *datatypes.JSONType[[]v1.ContainerStateTerminated] `gorm:"comment:作业的终止状态 (运行时、失败时采集)"`
				}
				if err := tx.Migrator().DropColumn(&Job{}, "ScheduleData"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&Job{}, "Events"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&Job{}, "TerminatedStates"); err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "202504181353", // Ensure the ID is unique
			Migrate: func(tx *gorm.DB) error {
				type Alert struct {
					gorm.Model
					JobName        string    `gorm:"type:varchar(255);not null;comment:作业名" json:"jobName"`
					AlertType      string    `gorm:"type:varchar(255);not null;comment:邮件类型" json:"alertType"`
					AlertTimestamp time.Time `gorm:"comment:邮件发送时间"`
					AllowRepeat    bool      `gorm:"type:boolean;default:false;comment:是否允许重复发送"`
					SendCount      int       `gorm:"not null;comment:邮件发送次数"`
				}

				// Create the table for alerts
				if err := tx.Migrator().CreateTable(&Alert{}); err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// Drop the alerts table if rolling back
				return tx.Migrator().DropTable("alerts")
			},
		},
		{
			ID: "202504221200", // 确保ID是唯一的
			Migrate: func(tx *gorm.DB) error {
				type AITask struct {
					DeletedAt gorm.DeletedAt `gorm:"index"`
				}

				// Add the DeletedAt column to the AITask table
				if err := tx.Migrator().AddColumn(&AITask{}, "DeletedAt"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type AITask struct {
					DeletedAt gorm.DeletedAt `gorm:"index"`
				}

				// Drop the DeletedAt column from the AITask table
				if err := tx.Migrator().DropColumn(&AITask{}, "DeletedAt"); err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "202504272234",
			Migrate: func(tx *gorm.DB) error {
				type Resource struct {
					// Resource relationship
					Type *model.CraterResourceType `gorm:"type:varchar(32);comment:资源类型" json:"type"`
				}

				// Add the Type and Networks columns to the Resource tableturn err

				if err := tx.Migrator().AddColumn(&Resource{}, "Type"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type Resource struct {
					// Resource relationship
					Type *model.CraterResourceType `gorm:"type:varchar(32);comment:资源类型" json:"type"`
				}

				// Drop the Type and Networks columns from the Resource table
				if err := tx.Migrator().DropColumn(&Resource{}, "Type"); err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "202504272311", // 确保ID是唯一的
			Migrate: func(tx *gorm.DB) error {
				type ResourceNetwork struct {
					gorm.Model
					ResourceID uint `gorm:"primaryKey;comment:资源ID" json:"resourceId"`
					NetworkID  uint `gorm:"primaryKey;comment:网络ID" json:"networkId"`

					Resource model.Resource `gorm:"foreignKey:ResourceID;constraint:OnDelete:CASCADE;" json:"resource"`
					Network  model.Resource `gorm:"foreignKey:NetworkID;constraint:OnDelete:CASCADE;" json:"network"`
				}
				// Create the table for resource networks
				if err := tx.Migrator().CreateTable(&ResourceNetwork{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// Drop the resource_networks table if rolling back
				return tx.Migrator().DropTable("resource_networks")
			},
		},
		//nolint:dupl// 相似的migrate代码
		{
			ID: "202504281510",
			Migrate: func(tx *gorm.DB) error {
				type Kaniko struct {
					// Resource relationship
					Tags datatypes.JSONType[[]string] `gorm:"null;comment:镜像标签"`
				}

				// Add the Type and Networks columns to the Resource tableturn err

				if err := tx.Migrator().AddColumn(&Kaniko{}, "Tags"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type Kaniko struct {
					// Resource relationship
					Tags datatypes.JSONType[[]string] `gorm:"null;comment:镜像标签"`
				}

				// Drop the Type and Networks columns from the Resource table
				if err := tx.Migrator().DropColumn(&Kaniko{}, "Tags"); err != nil {
					return err
				}
				return nil
			},
		},
		//nolint:dupl// 相似的migrate代码
		{
			ID: "202504281511",
			Migrate: func(tx *gorm.DB) error {
				type Image struct {
					// Resource relationship
					Tags datatypes.JSONType[[]string] `gorm:"null;comment:镜像标签"`
				}

				// Add the Type and Networks columns to the Resource tableturn err

				if err := tx.Migrator().AddColumn(&Image{}, "Tags"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type Image struct {
					// Resource relationship
					Tags datatypes.JSONType[[]string] `gorm:"null;comment:镜像标签"`
				}

				// Drop the Type and Networks columns from the Resource table
				if err := tx.Migrator().DropColumn(&Image{}, "Tags"); err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "202505061457",
			Migrate: func(tx *gorm.DB) error {
				type Kaniko struct {
					Template string `gorm:"type:text;comment:镜像的模板配置"`
				}
				// Add the Type and Networks columns to the Resource tableturn err
				if err := tx.Migrator().AddColumn(&Kaniko{}, "Template"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type Kaniko struct {
					Template string `gorm:"type:text;comment:镜像的模板配置"`
				}
				// Drop the Type and Networks columns from the Resource table
				if err := tx.Migrator().DropColumn(&Kaniko{}, "Template"); err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "202505192046",
			Migrate: func(tx *gorm.DB) error {
				type ImageUser struct {
					gorm.Model
					ImageID uint
					Image   model.Image
					UserID  uint
					User    model.User
				}

				// 明确指定表名
				if err := tx.Table("image_users").Migrator().CreateTable(&ImageUser{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("image_users") // 删除 imageuser 表
			},
		},
		{
			ID: "202505192047",
			Migrate: func(tx *gorm.DB) error {
				type ImageAccount struct {
					gorm.Model
					ImageID   uint
					Image     model.Image
					AccountID uint
					Account   model.Account
				}

				// 明确指定表名
				if err := tx.Table("image_accounts").Migrator().CreateTable(&ImageAccount{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("image_accounts") // 删除 image_accounts 表
			},
		},
		{
			ID: "202507141714",
			Migrate: func(tx *gorm.DB) error {
				type CudaBaseImage struct {
					gorm.Model
					Label      string `gorm:"type:varchar(128);not null;comment:image label showed in UI"`
					ImageLabel string `gorm:"uniqueIndex;type:varchar(128);null;comment:image label for imagelink generate"`
					Value      string `gorm:"type:varchar(512);comment:Full Cuda Image Link"`
				}

				// 明确指定表名
				if err := tx.Table("cuda_base_images").Migrator().CreateTable(&CudaBaseImage{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("cuda_base_images")
			},
		},
		//nolint:dupl// 相似的migrate代码
		{
			ID: "202507291446",
			Migrate: func(tx *gorm.DB) error {
				type Kaniko struct {
					Archs datatypes.JSONType[[]string] `gorm:"null;comment:镜像架构"`
				}
				if err := tx.Migrator().AddColumn(&Kaniko{}, "Archs"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type Kaniko struct {
					Archs datatypes.JSONType[[]string] `gorm:"null;comment:镜像架构"`
				}
				if err := tx.Migrator().DropColumn(&Kaniko{}, "Archs"); err != nil {
					return err
				}
				return nil
			},
		},
		//nolint:dupl// 相似的migrate代码
		{
			ID: "202507291447",
			Migrate: func(tx *gorm.DB) error {
				type Image struct {
					Archs datatypes.JSONType[[]string] `gorm:"null;comment:镜像架构"`
				}
				if err := tx.Migrator().AddColumn(&Image{}, "Archs"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type Image struct {
					Archs datatypes.JSONType[[]string] `gorm:"null;comment:镜像架构"`
				}
				if err := tx.Migrator().DropColumn(&Image{}, "Archs"); err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "202508041548",
			Migrate: func(tx *gorm.DB) error {
				type ApprovalOrder struct {
					gorm.Model
					Name        string                                         `gorm:"type:varchar(256);not null;comment:审批订单名称"`
					Type        model.ApprovalOrderType                        `gorm:"type:varchar(32);not null;default:job;comment:审批订单类型"`
					Status      model.ApprovalOrderStatus                      `gorm:"type:varchar(32);not null;default:Pending;comment:审批订单状态"`
					Content     datatypes.JSONType[model.ApprovalOrderContent] `gorm:"comment:审批订单内容"`
					ReviewNotes string                                         `gorm:"type:varchar(512);comment:审批备注"`
					CreatorID   uint                                           `gorm:"comment:创建者ID"`
					ReviewerID  uint                                           `gorm:"comment:审批者ID"`
				}
				// 明确指定表名
				if err := tx.Table("approval_orders").Migrator().CreateTable(&ApprovalOrder{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("approval_orders")
			},
		},
		{
			ID: "202508241756",
			Migrate: func(tx *gorm.DB) error {
				// ResourceVGPU is the table for GPU and VGPU resource relationships
				// It stores the one-to-one association between GPU resources and VGPU resources
				type ResourceVGPU struct {
					gorm.Model
					// GPU resource ID (nvidia.com/gpu)
					GPUResourceID uint `gorm:"not null;comment:GPU资源ID" json:"gpuResourceId"`
					// VGPU resource ID (nvidia.com/gpucores or nvidia.com/gpumem)
					VGPUResourceID uint `gorm:"not null;comment:VGPU资源ID" json:"vgpuResourceId"`

					// Configuration range
					Min         *int    `gorm:"comment:最小值" json:"min"`
					Max         *int    `gorm:"comment:最大值" json:"max"`
					Description *string `gorm:"type:text;comment:备注说明(用于区分是Cores还是Mem)" json:"description"`

					// Foreign key relationships
					GPUResource  model.Resource `gorm:"foreignKey:GPUResourceID;constraint:OnDelete:CASCADE;" json:"gpuResource"`
					VGPUResource model.Resource `gorm:"foreignKey:VGPUResourceID;constraint:OnDelete:CASCADE;" json:"vgpuResource"`
				}

				if err := tx.Migrator().CreateTable(&ResourceVGPU{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&model.ResourceVGPU{})
			},
		},
		{
			ID: "202509171000",
			Migrate: func(tx *gorm.DB) error {
				type Account struct {
					UserDefaultQuota datatypes.JSONType[model.QueueQuota] `gorm:"comment:账户中用户默认的资源配额模版"`
				}
				return tx.Migrator().AddColumn(&Account{}, "UserDefaultQuota")
			},
			Rollback: func(tx *gorm.DB) error {
				type Account struct {
					UserDefaultQuota datatypes.JSONType[model.QueueQuota] `gorm:"comment:账户中用户默认的资源配额模版"`
				}
				return tx.Migrator().DropColumn(&Account{}, "UserDefaultQuota")
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
			&model.Resource{},
			&model.Job{},
			&model.AITask{},
			&model.Kaniko{},
			&model.Image{},
			&model.Jobtemplate{},
			&model.Alert{},
			&model.ImageAccount{},
			&model.ImageUser{},
			&model.CudaBaseImage{},
			&model.ApprovalOrder{},
			&model.ResourceNetwork{},
			&model.ResourceVGPU{},
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
		var name, password string
		var ok bool
		if name, ok = os.LookupEnv("CRATER_ADMIN_USERNAME"); !ok {
			return fmt.Errorf("ADMIN_NAME is required for initial admin user")
		}
		if password, ok = os.LookupEnv("CRATER_ADMIN_PASSWORD"); !ok {
			return fmt.Errorf("ADMIN_PASSWORD is required for initial admin user")
		}
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
				Email:    ptr.To("admin@crater.io"),
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
