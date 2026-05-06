package main

import (
	"fmt"
	"os"
	"strconv"
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
	"github.com/raids-lab/crater/pkg/patrol"
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
		{
			ID: "202510272300",
			Migrate: func(tx *gorm.DB) error {
				type CronJobRecord struct {
					gorm.Model
					Name        string                    `gorm:"type:varchar(128);not null;index;comment:Cronjob名称" json:"name"`
					ExecuteTime time.Time                 `gorm:"not null;index;comment:执行时间" json:"executeTime"`
					Status      model.CronJobRecordStatus `gorm:"type:varchar(128);not null;index;default:unknown;comment:执行状态" json:"status"`
					Message     string                    `gorm:"type:text;comment:执行消息或错误信息" json:"message"`
					JobData     datatypes.JSON            `gorm:"type:jsonb;comment:任务数据(包含提醒和删除的任务列表)" json:"jobData"`
				}
				return tx.Table("cron_job_records").Migrator().CreateTable(&CronJobRecord{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("cron_job_records")
			},
		},
		{
			ID: "202510202499",
			Migrate: func(tx *gorm.DB) error {
				type CronJobConfig struct {
					gorm.Model
					Name    string            `gorm:"type:varchar(128);not null;index;unique;comment:Cronjob配置名称" json:"name"`
					Type    model.CronJobType `gorm:"type:varchar(128);not null;index;comment:Cronjob类型" json:"type"`
					Spec    string            `gorm:"type:varchar(128);not null;index;comment:Cron调度规范" json:"spec"`
					Suspend bool              `gorm:"not null;default:false;comment:是否暂停执行" json:"suspend"`
					Config  datatypes.JSON    `gorm:"type:jsonb;comment:Cronjob配置数据" json:"config"`
					EntryID int               `gorm:"type:int;comment:Cronjob标识ID" json:"entry_id"`
				}
				if err := tx.Table("cron_job_configs").Migrator().CreateTable(&CronJobConfig{}); err != nil {
					return err
				}

				initialConfigs := []*CronJobConfig{
					{
						Name:    "clean-long-time-job",
						Type:    model.CronJobTypeCleanerFunc,
						Spec:    "*/5 * * * *",
						Suspend: true,
						Config:  datatypes.JSON(`{"batchDays": "4", "interactiveDays": 4}`),
						EntryID: -1,
					},
					{
						Name:    "clean-low-gpu-util-job",
						Type:    model.CronJobTypeCleanerFunc,
						Spec:    "*/5 * * * *",
						Suspend: true,
						Config:  datatypes.JSON(`{"util": 0, "waitTime": 30, "timeRange": 90}`),
						EntryID: -1,
					},
					{
						Name:    "clean-waiting-jupyter",
						Type:    model.CronJobTypeCleanerFunc,
						Spec:    "*/5 * * * *",
						Suspend: true,
						Config:  datatypes.JSON(`{"waitMinitues": 5}`),
						EntryID: -1,
					},
				}

				for _, config := range initialConfigs {
					if err := tx.Where("name = ?", config.Name).FirstOrCreate(&config).Error; err != nil {
						return err
					}
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("cron_job_configs")
			},
		},
		{
			ID: "202511061031",
			Migrate: func(tx *gorm.DB) error {
				type CronJobConfig struct {
					Suspend *bool                     `gorm:"not null;default:false;comment:是否暂停执行" json:"suspend"`
					Status  model.CronJobConfigStatus `gorm:"type:varchar(128);index;default:unknown;comment:执行状态" json:"status"`
				}
				if err := tx.Migrator().AddColumn(&CronJobConfig{}, "Status"); err != nil {
					return err
				}
				// suspend == true -> suspended
				if err := tx.Model(&CronJobConfig{}).
					Where("suspend = ?", true).
					Update("status", model.CronJobConfigStatusSuspended).Error; err != nil {
					return err
				}
				// suspend == false -> idle
				if err := tx.Model(&CronJobConfig{}).
					Where("suspend = ?", false).
					Update("status", model.CronJobConfigStatusIdle).Error; err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type CronJobConfig struct {
					Status model.CronJobConfigStatus `gorm:"type:varchar(128);index;default:unknown;comment:执行状态" json:"status"`
				}
				return tx.Migrator().DropColumn(&CronJobConfig{}, "Status")
			},
		},
		{
			ID: "202511101503",
			Migrate: func(tx *gorm.DB) error {
				type CronJobConfig struct {
					Suspend *bool `gorm:"not null;default:true;comment:是否暂停执行" json:"suspend"`
				}
				if err := tx.Migrator().DropColumn(&CronJobConfig{}, "Suspend"); err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type CronJobConfig struct {
					Suspend *bool `gorm:"not null;default:true;comment:是否暂停执行" json:"suspend"`
				}
				if err := tx.Migrator().AddColumn(&CronJobConfig{}, "Suspend"); err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "202511131200",
			Migrate: func(tx *gorm.DB) error {
				type ModelDownload struct {
					gorm.Model
					Name            string `gorm:"type:varchar(512);not null;uniqueIndex:idx_download_unique,priority:1;comment:模型/数据集名称"`
					Source          string `gorm:"type:varchar(32);not null;default:modelscope;uniqueIndex:idx_download_unique,priority:2;comment:下载来源"`
					Category        string `gorm:"type:varchar(32);not null;default:model;uniqueIndex:idx_download_unique,priority:3;comment:类别(模型/数据集)"`
					Revision        string `gorm:"type:varchar(128);uniqueIndex:idx_download_unique,priority:4;comment:版本/分支/commit"`
					Path            string `gorm:"type:varchar(512);not null;comment:实际下载路径"`
					SizeBytes       int64  `gorm:"default:0;comment:文件总大小(字节)"`
					DownloadedBytes int64  `gorm:"default:0;comment:已下载大小(字节)"`
					DownloadSpeed   string `gorm:"type:varchar(32);comment:下载速度(如: 10MB/s)"`
					Status          string `gorm:"type:varchar(32);not null;default:Pending;comment:下载状态"`
					Message         string `gorm:"type:text;comment:状态消息(错误信息等)"`
					JobName         string `gorm:"type:varchar(256);comment:K8s Job名称"`
					CreatorID       uint   `gorm:"not null;comment:首个发起下载的用户ID"`
					ReferenceCount  int    `gorm:"default:0;comment:引用计数"`
				}
				return tx.Migrator().CreateTable(&ModelDownload{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("model_downloads")
			},
		},
		{
			ID: "202511131210",
			Migrate: func(tx *gorm.DB) error {
				type UserModelDownload struct {
					ID              uint      `gorm:"primaryKey"`
					UserID          uint      `gorm:"not null;uniqueIndex:idx_user_download,priority:1"`
					ModelDownloadID uint      `gorm:"not null;uniqueIndex:idx_user_download,priority:2"`
					CreatedAt       time.Time `gorm:"comment:用户添加此下载的时间"`
				}
				return tx.Migrator().CreateTable(&UserModelDownload{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("user_model_downloads")
			},
		},
		{
			ID: "202512061528",
			Migrate: func(tx *gorm.DB) error {
				type SystemConfig struct {
					Key   string `gorm:"primarykey;size:100;comment:配置项的键"`
					Value string `gorm:"type:text;comment:配置项的值"`
				}
				type GpuAnalysis struct {
					ID                uint `gorm:"primarykey"`
					CreatedAt         time.Time
					DeletedAt         gorm.DeletedAt     `gorm:"index;comment:软删除时间戳"` // 必须显式定义为 gorm.DeletedAt
					JobID             uint               `gorm:"index;comment:关联的作业ID，用于跳转，但不设外键约束"`
					JobName           string             `gorm:"index;comment:关联的作业名称 (冗余字段，用于显示)"`
					UserID            uint               `gorm:"index;comment:关联的用户ID，用于统计"`
					UserName          string             `gorm:"comment:提交作业的用户名 (冗余字段，用于显示)"`
					PodName           string             `gorm:"index;comment:被分析的Pod名称"`
					Namespace         string             `gorm:"comment:Pod所在的命名空间"`
					Phase1Score       int                `gorm:"comment:基于监控数据的初步评分"`
					Phase2Score       int                `gorm:"comment:结合脚本内容的二次评分"`
					Phase1LLMReason   string             `gorm:"type:text;comment:LLM给出的初步分析理由"`
					Phase2LLMReason   string             `gorm:"type:text;comment:LLM给出的二次分析理由"`
					LLMVersion        string             `gorm:"size:100;comment:使用的LLM模型版本"`
					Command           string             `gorm:"type:text;comment:GPU进程的启动命令"`
					ScriptContent     string             `gorm:"type:text;comment:获取到的脚本内容" json:"-"`
					HistoricalMetrics string             `gorm:"type:text;comment:用于分析的历史指标摘要(JSON格式)"`
					ReviewStatus      model.ReviewStatus `gorm:"default:1;comment:审核状态 (1: Pending, 2: Confirmed, 3: Ignored)"`
				}
				return tx.Migrator().CreateTable(&GpuAnalysis{}, &SystemConfig{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&model.GpuAnalysis{}, &model.SystemConfig{})
			},
		},
		{
			ID: "202603290001",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(
					&model.AgentSession{},
					&model.AgentMessage{},
					&model.AgentToolCall{},
					&model.JobLogSnapshot{},
				)
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(
					"agent_sessions",
					"agent_messages",
					"agent_tool_calls",
					"job_log_snapshots",
				)
			},
		},
		{
			ID: "202604030001",
			Migrate: func(tx *gorm.DB) error {
				type AgentSession struct {
					PinnedAt *time.Time `gorm:"index"`
				}
				return tx.Migrator().AddColumn(&AgentSession{}, "PinnedAt")
			},
			Rollback: func(tx *gorm.DB) error {
				type AgentSession struct {
					PinnedAt *time.Time `gorm:"index"`
				}
				return tx.Migrator().DropColumn(&AgentSession{}, "PinnedAt")
			},
		},
		{
			ID: "202604030002",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(
					&model.AgentSession{},
					&model.AgentToolCall{},
					&model.AgentTurn{},
					&model.AgentRunEvent{},
				); err != nil {
					return err
				}
				return tx.Exec(`
					UPDATE agent_sessions
					SET last_orchestration_mode = 'single_agent'
					WHERE last_orchestration_mode IS NULL
					   OR BTRIM(last_orchestration_mode) = ''
				`).Error
			},
			Rollback: func(tx *gorm.DB) error {
				type AgentSession struct {
					LastOrchestrationMode string `gorm:"type:varchar(32);default:'single_agent'"`
				}
				type AgentToolCall struct {
					TurnID        string `gorm:"type:uuid;index"`
					ToolCallID    string `gorm:"type:varchar(128);index"`
					AgentID       string `gorm:"type:varchar(128);index"`
					ParentEventID *uint  `gorm:"index"`
					AgentRole     string `gorm:"type:varchar(32);index"`
				}
				if err := tx.Migrator().DropTable("agent_run_events", "agent_turns"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&AgentToolCall{}, "AgentRole"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&AgentToolCall{}, "ParentEventID"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&AgentToolCall{}, "AgentID"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&AgentToolCall{}, "ToolCallID"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&AgentToolCall{}, "TurnID"); err != nil {
					return err
				}
				return tx.Migrator().DropColumn(&AgentSession{}, "LastOrchestrationMode")
			},
		},
		{
			ID: "202604040001",
			Migrate: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`CREATE TABLE IF NOT EXISTS ops_audit_reports (
						id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
						report_type VARCHAR(32) NOT NULL,
						status VARCHAR(16) NOT NULL DEFAULT 'running',
						trigger_source VARCHAR(32),
						summary JSONB,
						created_at TIMESTAMPTZ DEFAULT NOW(),
						completed_at TIMESTAMPTZ
					)`,
					`CREATE TABLE IF NOT EXISTS ops_audit_items (
						id BIGSERIAL PRIMARY KEY,
						report_id UUID NOT NULL REFERENCES ops_audit_reports(id),
						job_name VARCHAR(255) NOT NULL,
						user_id VARCHAR(128),
						account_id VARCHAR(128),
						username VARCHAR(128),
						action_type VARCHAR(32) NOT NULL,
						severity VARCHAR(16) NOT NULL,
						gpu_utilization FLOAT,
						gpu_requested INT,
						gpu_actual_used INT,
						analysis_detail JSONB,
						handled BOOLEAN DEFAULT FALSE,
						handled_at TIMESTAMPTZ,
						handled_by VARCHAR(128),
						created_at TIMESTAMPTZ DEFAULT NOW()
					)`,
					`CREATE INDEX IF NOT EXISTS idx_audit_items_report ON ops_audit_items(report_id)`,
					`CREATE INDEX IF NOT EXISTS idx_audit_items_action ON ops_audit_items(action_type, severity)`,
					`CREATE INDEX IF NOT EXISTS idx_audit_items_handled ON ops_audit_items(handled) WHERE NOT handled`,
					`CREATE INDEX IF NOT EXISTS idx_audit_reports_type_status ON ops_audit_reports(report_type, status)`,
					`CREATE INDEX IF NOT EXISTS idx_audit_reports_created ON ops_audit_reports(created_at DESC)`,
				})
			},
			Rollback: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`DROP TABLE IF EXISTS ops_audit_items`,
					`DROP TABLE IF EXISTS ops_audit_reports`,
				})
			},
		},
		{
			ID: "202604050001",
			Migrate: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`ALTER TABLE ops_audit_reports
						ADD COLUMN IF NOT EXISTS report_json JSONB,
						ADD COLUMN IF NOT EXISTS period_start TIMESTAMPTZ,
						ADD COLUMN IF NOT EXISTS period_end TIMESTAMPTZ,
						ADD COLUMN IF NOT EXISTS comparison_report_id UUID,
						ADD COLUMN IF NOT EXISTS job_total INT DEFAULT 0,
						ADD COLUMN IF NOT EXISTS job_success INT DEFAULT 0,
						ADD COLUMN IF NOT EXISTS job_failed INT DEFAULT 0,
						ADD COLUMN IF NOT EXISTS job_pending INT DEFAULT 0`,
					`ALTER TABLE ops_audit_items
						ADD COLUMN IF NOT EXISTS category VARCHAR(32),
						ADD COLUMN IF NOT EXISTS job_type VARCHAR(32),
						ADD COLUMN IF NOT EXISTS owner VARCHAR(128),
						ADD COLUMN IF NOT EXISTS namespace VARCHAR(128),
						ADD COLUMN IF NOT EXISTS duration_seconds INT,
						ADD COLUMN IF NOT EXISTS resource_requested JSONB,
						ADD COLUMN IF NOT EXISTS resource_actual JSONB,
						ADD COLUMN IF NOT EXISTS exit_code INT,
						ADD COLUMN IF NOT EXISTS failure_reason VARCHAR(64)`,
					`CREATE INDEX IF NOT EXISTS idx_audit_reports_type_created
						ON ops_audit_reports (report_type, created_at DESC)`,
					`CREATE INDEX IF NOT EXISTS idx_audit_items_category
						ON ops_audit_items (category) WHERE category IS NOT NULL`,
					`CREATE INDEX IF NOT EXISTS idx_audit_items_failure
						ON ops_audit_items (failure_reason) WHERE failure_reason IS NOT NULL`,
				})
			},
			Rollback: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`DROP INDEX IF EXISTS idx_audit_items_failure`,
					`DROP INDEX IF EXISTS idx_audit_items_category`,
					`DROP INDEX IF EXISTS idx_audit_reports_type_created`,
					`ALTER TABLE ops_audit_items
						DROP COLUMN IF EXISTS failure_reason,
						DROP COLUMN IF EXISTS exit_code,
						DROP COLUMN IF EXISTS resource_actual,
						DROP COLUMN IF EXISTS resource_requested,
						DROP COLUMN IF EXISTS duration_seconds,
						DROP COLUMN IF EXISTS namespace,
						DROP COLUMN IF EXISTS owner,
						DROP COLUMN IF EXISTS job_type,
						DROP COLUMN IF EXISTS category`,
					`ALTER TABLE ops_audit_reports
						DROP COLUMN IF EXISTS job_pending,
						DROP COLUMN IF EXISTS job_failed,
						DROP COLUMN IF EXISTS job_success,
						DROP COLUMN IF EXISTS job_total,
						DROP COLUMN IF EXISTS comparison_report_id,
						DROP COLUMN IF EXISTS period_end,
						DROP COLUMN IF EXISTS period_start,
						DROP COLUMN IF EXISTS report_json`,
				})
			},
		},
		{
			ID: "202604110001",
			Migrate: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`ALTER TABLE agent_tool_calls
						ADD COLUMN IF NOT EXISTS execution_backend VARCHAR(64),
						ADD COLUMN IF NOT EXISTS sandbox_job_name VARCHAR(255),
						ADD COLUMN IF NOT EXISTS script_name VARCHAR(128),
						ADD COLUMN IF NOT EXISTS result_artifact_ref TEXT,
						ADD COLUMN IF NOT EXISTS egress_domains JSONB`,
					`CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_sandbox_job_name
						ON agent_tool_calls (sandbox_job_name)`,
					`CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_script_name
						ON agent_tool_calls (script_name)`,
				})
			},
			Rollback: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`DROP INDEX IF EXISTS idx_agent_tool_calls_script_name`,
					`DROP INDEX IF EXISTS idx_agent_tool_calls_sandbox_job_name`,
					`ALTER TABLE agent_tool_calls
						DROP COLUMN IF EXISTS egress_domains,
						DROP COLUMN IF EXISTS result_artifact_ref,
						DROP COLUMN IF EXISTS script_name,
						DROP COLUMN IF EXISTS sandbox_job_name,
						DROP COLUMN IF EXISTS execution_backend`,
				})
			},
		},
		{
			ID: "202604180001",
			Migrate: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`ALTER TABLE approval_orders
						ADD COLUMN IF NOT EXISTS review_source VARCHAR(32) DEFAULT '',
						ADD COLUMN IF NOT EXISTS agent_report TEXT DEFAULT ''`,
					`UPDATE approval_orders
						SET review_source = 'system_auto'
						WHERE review_notes LIKE '%approved due to system%'
						  AND review_source = ''`,
					`UPDATE approval_orders
						SET review_source = 'admin_manual'
						WHERE reviewer_id > 0
						  AND review_source = ''
						  AND status IN ('Approved', 'Rejected')`,
				})
			},
			Rollback: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`ALTER TABLE approval_orders
						DROP COLUMN IF EXISTS agent_report,
						DROP COLUMN IF EXISTS review_source`,
				})
			},
		},
		{
			ID: "202512261300",
			Migrate: func(tx *gorm.DB) error {
				config := &model.CronJobConfig{
					Name:    "clean-waiting-custom",
					Type:    model.CronJobTypeCleanerFunc,
					Spec:    "*/5 * * * *",
					Status:  model.CronJobConfigStatusSuspended,
					Config:  datatypes.JSON(`{"waitMinitues": 5, "jobTypes": ["custom"]}`),
					EntryID: -1,
				}
				if err := tx.Where("name = ?", config.Name).FirstOrCreate(&config).Error; err != nil {
					return err
				}

				// 2. Update clean-waiting-jupyter
				if err := tx.Model(&model.CronJobConfig{}).
					Where("name = ?", "clean-waiting-jupyter").
					Update("config", datatypes.JSON(`{"waitMinitues": 5, "jobTypes": ["jupyter"]}`)).Error; err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Unscoped().Where("name = ?", "clean-waiting-custom").Delete(&model.CronJobConfig{}).Error; err != nil {
					return err
				}
				if err := tx.Model(&model.CronJobConfig{}).
					Where("name = ?", "clean-waiting-jupyter").
					Update("config", datatypes.JSON(`{"waitMinitues": 5}`)).Error; err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "202603170000",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.OperationLog{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&model.OperationLog{})
			},
		},
		{
			ID: "202604220001",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.AgentFeedback{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("agent_feedbacks")
			},
		},
		{
			ID: "202604220002",
			Migrate: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`ALTER TABLE agent_sessions
						ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'chat'`,
					`UPDATE agent_sessions
						SET source = 'chat'
						WHERE source IS NULL OR BTRIM(source) = ''`,
					`UPDATE agent_sessions
						SET source = 'ops_audit'
						WHERE title LIKE '[audit] 审批%'
						  AND source = 'chat'`,
					`CREATE INDEX IF NOT EXISTS idx_agent_sessions_source
						ON agent_sessions (source)`,
					`ALTER TABLE agent_tool_calls
						ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'backend'`,
					`UPDATE agent_tool_calls
						SET source = 'backend'
						WHERE source IS NULL OR BTRIM(source) = ''`,
					`CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_source
						ON agent_tool_calls (source)`,
				})
			},
			Rollback: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`DROP INDEX IF EXISTS idx_agent_tool_calls_source`,
					`ALTER TABLE agent_tool_calls
						DROP COLUMN IF EXISTS source`,
					`DROP INDEX IF EXISTS idx_agent_sessions_source`,
					`ALTER TABLE agent_sessions
						DROP COLUMN IF EXISTS source`,
				})
			},
		},
		{
			ID: "202604230001",
			Migrate: func(tx *gorm.DB) error {
				type AgentFeedback struct {
					EnrichedAt *time.Time `json:"enrichedAt,omitempty"`
				}
				return tx.Migrator().AddColumn(&AgentFeedback{}, "EnrichedAt")
			},
			Rollback: func(tx *gorm.DB) error {
				type AgentFeedback struct {
					EnrichedAt *time.Time `json:"enrichedAt,omitempty"`
				}
				return tx.Migrator().DropColumn(&AgentFeedback{}, "EnrichedAt")
			},
		},
		{
			ID: "202604230002",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&model.AgentQualityEval{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("agent_quality_evals")
			},
		},
		{
			ID: "202604230003",
			Migrate: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`ALTER TABLE agent_quality_evals
						ADD COLUMN IF NOT EXISTS eval_scope VARCHAR(16) NOT NULL DEFAULT 'session',
						ADD COLUMN IF NOT EXISTS eval_type VARCHAR(32) NOT NULL DEFAULT 'full',
						ADD COLUMN IF NOT EXISTS target_id VARCHAR(128),
						ADD COLUMN IF NOT EXISTS metadata JSONB`,
					`UPDATE agent_quality_evals
						SET eval_scope = CASE WHEN turn_id IS NULL THEN 'session' ELSE 'turn' END
						WHERE eval_scope IS NULL OR BTRIM(eval_scope) = ''`,
					`UPDATE agent_quality_evals
						SET eval_type = 'full'
						WHERE eval_type IS NULL OR BTRIM(eval_type) = ''`,
					`UPDATE agent_quality_evals
						SET target_id = CASE
							WHEN eval_scope = 'turn' AND turn_id IS NOT NULL THEN turn_id::text
							ELSE session_id::text
						END
						WHERE target_id IS NULL OR BTRIM(target_id) = ''`,
					`CREATE INDEX IF NOT EXISTS idx_agent_quality_evals_session_created
						ON agent_quality_evals (session_id, created_at DESC)`,
					`CREATE INDEX IF NOT EXISTS idx_agent_quality_evals_scope_type_status
						ON agent_quality_evals (eval_scope, eval_type, eval_status)`,
					`CREATE INDEX IF NOT EXISTS idx_agent_quality_evals_target_created
						ON agent_quality_evals (eval_scope, target_id, created_at DESC)`,
					`CREATE INDEX IF NOT EXISTS idx_agent_quality_evals_active
						ON agent_quality_evals (eval_status, created_at DESC)
						WHERE eval_status IN ('pending', 'running')`,
				})
			},
			Rollback: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`DROP INDEX IF EXISTS idx_agent_quality_evals_active`,
					`DROP INDEX IF EXISTS idx_agent_quality_evals_target_created`,
					`DROP INDEX IF EXISTS idx_agent_quality_evals_scope_type_status`,
					`DROP INDEX IF EXISTS idx_agent_quality_evals_session_created`,
					`ALTER TABLE agent_quality_evals
						DROP COLUMN IF EXISTS metadata,
						DROP COLUMN IF EXISTS target_id,
						DROP COLUMN IF EXISTS eval_type,
						DROP COLUMN IF EXISTS eval_scope`,
				})
			},
		},
		{
			ID: "202603311930",
			Migrate: func(tx *gorm.DB) error {
				type Account struct {
					BillingIssueAmount        *int64     `gorm:"comment:账户周期发放点数额度(内部微点, 为空表示未配置)"`
					BillingIssuePeriodMinutes *int       `gorm:"comment:账户周期发放间隔分钟(<=0表示关闭, 为空表示未配置)"`
					BillingLastIssuedAt       *time.Time `gorm:"comment:账户上次发放时间"`
				}
				type UserAccount struct {
					BillingIssueAmountOverride *int64 `gorm:"comment:用户在账户内的周期发放额度覆盖(内部微点, 为空表示沿用账户配置)"`
					PeriodFreeBalance          int64  `gorm:"not null;default:0;comment:用户在当前周期的免费额度剩余(内部微点)"`
				}
				type User struct {
					ExtraBalance int64 `gorm:"type:bigint;not null;default:0;comment:用户额外点数余额(内部微点, 充值/奖励)"`
				}
				type Job struct {
					LastSettledAt     *time.Time `gorm:"comment:作业上次结算时间"`
					BilledPointsTotal int64      `gorm:"not null;default:0;comment:作业累计已结算点数(内部微点)"`
				}
				type Resource struct {
					UnitPrice int64 `gorm:"not null;default:0;comment:资源单位价格(内部微点, 展示为点数/单位/小时)"`
				}

				if err := tx.Migrator().AddColumn(&Account{}, "BillingIssueAmount"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&Account{}, "BillingIssuePeriodMinutes"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&Account{}, "BillingLastIssuedAt"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&UserAccount{}, "BillingIssueAmountOverride"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&UserAccount{}, "PeriodFreeBalance"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&User{}, "ExtraBalance"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&Job{}, "LastSettledAt"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&Job{}, "BilledPointsTotal"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&Resource{}, "UnitPrice"); err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				type Account struct {
					BillingIssueAmount        *int64     `gorm:"comment:账户周期发放点数额度(内部微点, 为空表示未配置)"`
					BillingIssuePeriodMinutes *int       `gorm:"comment:账户周期发放间隔分钟(<=0表示关闭, 为空表示未配置)"`
					BillingLastIssuedAt       *time.Time `gorm:"comment:账户上次发放时间"`
				}
				type UserAccount struct {
					BillingIssueAmountOverride *int64 `gorm:"comment:用户在账户内的周期发放额度覆盖(内部微点, 为空表示沿用账户配置)"`
					PeriodFreeBalance          int64  `gorm:"not null;default:0;comment:用户在当前周期的免费额度剩余(内部微点)"`
				}
				type User struct {
					ExtraBalance int64 `gorm:"type:bigint;not null;default:0;comment:用户额外点数余额(内部微点, 充值/奖励)"`
				}
				type Job struct {
					LastSettledAt     *time.Time `gorm:"comment:作业上次结算时间"`
					BilledPointsTotal int64      `gorm:"not null;default:0;comment:作业累计已结算点数(内部微点)"`
				}
				type Resource struct {
					UnitPrice int64 `gorm:"not null;default:0;comment:资源单位价格(内部微点, 展示为点数/单位/小时)"`
				}

				if err := tx.Migrator().DropColumn(&Account{}, "BillingIssueAmount"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&Account{}, "BillingIssuePeriodMinutes"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&Account{}, "BillingLastIssuedAt"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&UserAccount{}, "BillingIssueAmountOverride"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&UserAccount{}, "PeriodFreeBalance"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&User{}, "ExtraBalance"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&Job{}, "LastSettledAt"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&Job{}, "BilledPointsTotal"); err != nil {
					return err
				}
				return tx.Migrator().DropColumn(&Resource{}, "UnitPrice")
			},
		},
		{
			ID: "202603111300",
			Migrate: func(tx *gorm.DB) error {
				type Job struct {
					ID    uint   `gorm:"primaryKey"`
					Queue string `gorm:"type:varchar(256);index:idx_jobs_queue;comment:作业提交的volcano队列"`
				}

				if err := tx.Migrator().AddColumn(&Job{}, "Queue"); err != nil {
					return err
				}
				return tx.Migrator().CreateIndex(&Job{}, "Queue")
			},
			Rollback: func(tx *gorm.DB) error {
				type Job struct {
					Queue string `gorm:"type:varchar(256);index:idx_jobs_queue;comment:作业提交的volcano队列"`
				}

				if err := tx.Migrator().DropIndex(&Job{}, "Queue"); err != nil {
					return err
				}
				return tx.Migrator().DropColumn(&Job{}, "Queue")
			},
		},
		{
			ID: "202603140930",
			Migrate: func(tx *gorm.DB) error {
				type QueueQuotaLimit struct {
					gorm.Model
					Name                  string                                `gorm:"uniqueIndex;type:varchar(256);not null;comment:队列名字"`
					Enabled               bool                                  `gorm:"not null;default:false;comment:是否启用队列资源限制"`
					PrequeueCandidateSize int                                   `gorm:"not null;default:10;comment:Prequeue 候选作业集大小"`
					Quota                 datatypes.JSONType[map[string]string] `gorm:"type:jsonb;comment:队列内资源限制"`
				}
				return tx.Table("queue_quotas").Migrator().CreateTable(&QueueQuotaLimit{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("queue_quotas")
			},
		},
		{
			ID: "202604011747",
			Migrate: func(tx *gorm.DB) error {
				type Job struct {
					gorm.Model
					ScheduleType *int `gorm:"index:idx_jobs_schedule_type;default:1;not null;comment:调度类型"`
				}
				if err := tx.Migrator().AddColumn(&Job{}, "ScheduleType"); err != nil {
					return err
				}
				return tx.Migrator().CreateIndex(&Job{}, "ScheduleType")
			},
			Rollback: func(tx *gorm.DB) error {
				type Job struct {
					ScheduleType *int `gorm:"index:idx_jobs_schedule_type;default:1;not null;comment:调度类型"`
				}
				if err := tx.Migrator().DropIndex(&Job{}, "ScheduleType"); err != nil {
					return err
				}
				return tx.Migrator().DropColumn(&Job{}, "ScheduleType")
			},
		},
		{
			ID: "202604041030",
			Migrate: func(tx *gorm.DB) error {
				type Job struct {
					gorm.Model
					WaitingToleranceSeconds *int64 `gorm:"comment:作业等待忍耐时间(秒)"`
				}
				return tx.Migrator().AddColumn(&Job{}, "WaitingToleranceSeconds")
			},
			Rollback: func(tx *gorm.DB) error {
				type Job struct {
					WaitingToleranceSeconds *int64 `gorm:"comment:作业等待忍耐时间(秒)"`
				}
				return tx.Migrator().DropColumn(&Job{}, "WaitingToleranceSeconds")
			},
		},
		{
			ID: "202604091400",
			Migrate: func(tx *gorm.DB) error {
				type PrequeueConfig struct {
					gorm.Model
					Key      string     `gorm:"uniqueIndex:idx_prequeue_configs_key;size:100;not null;comment:配置项的键"`
					Value    string     `gorm:"type:text;not null;comment:配置项的值"`
					ExpireAt *time.Time `gorm:"index:idx_prequeue_configs_expire_at;comment:配置项过期时间"`
				}
				migrator := tx.Table("prequeue_configs").Migrator()
				if err := migrator.CreateTable(&PrequeueConfig{}); err != nil {
					return err
				}
				if !migrator.HasIndex(&PrequeueConfig{}, "idx_prequeue_configs_key") {
					if err := migrator.CreateIndex(&PrequeueConfig{}, "idx_prequeue_configs_key"); err != nil {
						return err
					}
				}
				if !migrator.HasIndex(&PrequeueConfig{}, "idx_prequeue_configs_expire_at") {
					if err := migrator.CreateIndex(&PrequeueConfig{}, "idx_prequeue_configs_expire_at"); err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("prequeue_configs")
			},
		},
		{
			ID: "202604161300",
			Migrate: func(tx *gorm.DB) error {
				defaults := map[string]string{
					model.PrequeueBackfillEnabledKey:   strconv.FormatBool(model.PrequeueDefaultBackfillEnabled),
					model.PrequeueQueueQuotaEnabledKey: strconv.FormatBool(model.PrequeueDefaultQueueQuotaEnabled),
				}

				for key, value := range defaults {
					var count int64
					if err := tx.Table("prequeue_configs").Where("key = ?", key).Count(&count).Error; err != nil {
						return err
					}
					if count > 0 {
						continue
					}
					if err := tx.Table("prequeue_configs").Create(map[string]any{
						"created_at": time.Now(),
						"updated_at": time.Now(),
						"key":        key,
						"value":      value,
					}).Error; err != nil {
						return err
					}
				}

				return tx.Table("prequeue_configs").Where("key = ?", "enabled").Delete(nil).Error
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Table("prequeue_configs").Where("key IN ?", []string{
					model.PrequeueBackfillEnabledKey,
					model.PrequeueQueueQuotaEnabledKey,
				}).Delete(nil).Error; err != nil {
					return err
				}

				var count int64
				if err := tx.Table("prequeue_configs").Where("key = ?", "enabled").Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return nil
				}

				return tx.Table("prequeue_configs").Create(map[string]any{
					"created_at": time.Now(),
					"updated_at": time.Now(),
					"key":        "enabled",
					"value":      strconv.FormatBool(model.PrequeueDefaultBackfillEnabled),
				}).Error
			},
		},
		{
			ID: "202604171200",
			Migrate: func(tx *gorm.DB) error {
				defaults := map[string]string{
					model.PrequeueActivateTickerIntervalSecondsKey: strconv.FormatInt(model.PrequeueDefaultActivateTickerIntervalSeconds, 10),
					model.PrequeueMaxTotalActivationsPerRoundKey:   strconv.FormatInt(model.PrequeueDefaultMaxTotalActivationsPerRound, 10),
				}
				for key, value := range defaults {
					var count int64
					if err := tx.Table("prequeue_configs").Where("key = ?", key).Count(&count).Error; err != nil {
						return err
					}
					if count > 0 {
						continue
					}
					if err := tx.Table("prequeue_configs").Create(map[string]any{
						"created_at": time.Now(),
						"updated_at": time.Now(),
						"key":        key,
						"value":      value,
					}).Error; err != nil {
						return err
					}
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Table("prequeue_configs").Where("key IN ?", []string{
					model.PrequeueActivateTickerIntervalSecondsKey,
					model.PrequeueMaxTotalActivationsPerRoundKey,
				}).Delete(nil).Error
			},
		},
		{
			ID: "202604270001",
			Migrate: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`DROP INDEX IF EXISTS idx_agent_tool_calls_script_name`,
					`DROP INDEX IF EXISTS idx_agent_tool_calls_sandbox_job_name`,
					`ALTER TABLE agent_tool_calls
						DROP COLUMN IF EXISTS egress_domains,
						DROP COLUMN IF EXISTS result_artifact_ref,
						DROP COLUMN IF EXISTS script_name,
						DROP COLUMN IF EXISTS sandbox_job_name`,
				})
			},
			Rollback: func(tx *gorm.DB) error {
				return runStatements(tx, []string{
					`ALTER TABLE agent_tool_calls
						ADD COLUMN IF NOT EXISTS sandbox_job_name VARCHAR(255),
						ADD COLUMN IF NOT EXISTS script_name VARCHAR(128),
						ADD COLUMN IF NOT EXISTS result_artifact_ref TEXT,
						ADD COLUMN IF NOT EXISTS egress_domains JSONB`,
					`CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_sandbox_job_name
						ON agent_tool_calls (sandbox_job_name)`,
					`CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_script_name
						ON agent_tool_calls (script_name)`,
				})
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
			&model.ModelDownload{},
			&model.UserModelDownload{},
			&model.CronJobConfig{},
			&model.CronJobRecord{},
			&model.GpuAnalysis{},
			&model.SystemConfig{},
			&model.AgentSession{},
			&model.AgentMessage{},
			&model.AgentToolCall{},
			&model.AgentTurn{},
			&model.AgentRunEvent{},
			&model.JobLogSnapshot{},
			&model.AgentFeedback{},
			&model.AgentQualityEval{},
			&model.OperationLog{},
			&model.PrequeueConfig{},
			&model.QueueQuotaLimit{},
		)
		if err != nil {
			return err
		}
		if err := ensureOpsAuditSchema(tx); err != nil {
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

		// 5. create initial cronjob configs
		initialCronJobConfigs := []*model.CronJobConfig{
			{
				Name:    "clean-long-time-job",
				Type:    model.CronJobTypeCleanerFunc,
				Spec:    "*/5 * * * *",
				Status:  model.CronJobConfigStatusSuspended,
				Config:  datatypes.JSON(`{"batchDays": 4, "interactiveDays": 4}`),
				EntryID: -1,
			},
			{
				Name:    "clean-low-gpu-util-job",
				Type:    model.CronJobTypeCleanerFunc,
				Spec:    "*/5 * * * *",
				Status:  model.CronJobConfigStatusSuspended,
				Config:  datatypes.JSON(`{"util": 0, "waitTime": 30, "timeRange": 90}`),
				EntryID: -1,
			},
			{
				Name:    "clean-waiting-jupyter",
				Type:    model.CronJobTypeCleanerFunc,
				Spec:    "*/5 * * * *",
				Status:  model.CronJobConfigStatusSuspended,
				Config:  datatypes.JSON(`{"waitMinitues": 5, "jobTypes": ["jupyter"]}`),
				EntryID: -1,
			},
			{
				Name:    "clean-waiting-custom",
				Type:    model.CronJobTypeCleanerFunc,
				Spec:    "*/5 * * * *",
				Status:  model.CronJobConfigStatusSuspended,
				Config:  datatypes.JSON(`{"waitMinitues": 5, "jobTypes": ["custom"]}`),
				EntryID: -1,
			},
			{
				Name:   patrol.TRIGGER_ADMIN_OPS_REPORT_JOB,
				Type:   model.CronJobTypePatrolFunc,
				Spec:   "0 * * * *",
				Status: model.CronJobConfigStatusSuspended,
				Config: datatypes.JSON(`{
					"days": 1,
					"lookback_hours": 1,
					"gpu_threshold": 5,
					"idle_hours": 1,
					"running_limit": 20,
					"node_limit": 10
				}`),
				EntryID: -1,
			},
			{
				Name:   patrol.TRIGGER_STORAGE_DAILY_AUDIT_JOB,
				Type:   model.CronJobTypePatrolFunc,
				Spec:   "0 3 * * *",
				Status: model.CronJobConfigStatusSuspended,
				Config: datatypes.JSON(`{
					"days": 1,
					"pvc_limit": 200
				}`),
				EntryID: -1,
			},
		}

		for _, config := range initialCronJobConfigs {
			if err := tx.Where("name = ?", config.Name).FirstOrCreate(&config).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err := m.Migrate(); err != nil {
		panic(fmt.Errorf("could not migrate: %w", err))
	}
}

func runStatements(tx *gorm.DB, statements []string) error {
	for _, stmt := range statements {
		if err := tx.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureOpsAuditSchema(tx *gorm.DB) error {
	return runStatements(tx, []string{
		`CREATE TABLE IF NOT EXISTS ops_audit_reports (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			report_type VARCHAR(32) NOT NULL,
			status VARCHAR(16) NOT NULL DEFAULT 'running',
			trigger_source VARCHAR(32),
			summary JSONB,
			report_json JSONB,
			period_start TIMESTAMPTZ,
			period_end TIMESTAMPTZ,
			comparison_report_id UUID,
			job_total INT DEFAULT 0,
			job_success INT DEFAULT 0,
			job_failed INT DEFAULT 0,
			job_pending INT DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			completed_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS ops_audit_items (
			id BIGSERIAL PRIMARY KEY,
			report_id UUID NOT NULL REFERENCES ops_audit_reports(id),
			job_name VARCHAR(255) NOT NULL,
			user_id VARCHAR(128),
			account_id VARCHAR(128),
			username VARCHAR(128),
			action_type VARCHAR(32) NOT NULL,
			severity VARCHAR(16) NOT NULL,
			gpu_utilization FLOAT,
			gpu_requested INT,
			gpu_actual_used INT,
			analysis_detail JSONB,
			handled BOOLEAN DEFAULT FALSE,
			handled_at TIMESTAMPTZ,
			handled_by VARCHAR(128),
			category VARCHAR(32),
			job_type VARCHAR(32),
			owner VARCHAR(128),
			namespace VARCHAR(128),
			duration_seconds INT,
			resource_requested JSONB,
			resource_actual JSONB,
			exit_code INT,
			failure_reason VARCHAR(64),
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_items_report ON ops_audit_items(report_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_items_action ON ops_audit_items(action_type, severity)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_items_handled ON ops_audit_items(handled) WHERE NOT handled`,
		`CREATE INDEX IF NOT EXISTS idx_audit_reports_type_status ON ops_audit_reports(report_type, status)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_reports_created ON ops_audit_reports(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_reports_type_created
			ON ops_audit_reports (report_type, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_items_category
			ON ops_audit_items (category) WHERE category IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_audit_items_failure
			ON ops_audit_items (failure_reason) WHERE failure_reason IS NOT NULL`,
	})
}
