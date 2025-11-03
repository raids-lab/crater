package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CronJobRecordStatus string

const (
	CronJobRecordStatusUnknown CronJobRecordStatus = "unknown"
	CronJobRecordStatusSuccess CronJobRecordStatus = "success"
	CronJobRecordStatusFailed  CronJobRecordStatus = "failed"
)

type CronJobRecord struct {
	gorm.Model
	Name        string              `gorm:"type:varchar(128);not null;index;comment:Cronjob名称" json:"name"`
	ExecuteTime time.Time           `gorm:"not null;index;comment:执行时间" json:"executeTime"`
	Status      CronJobRecordStatus `gorm:"type:varchar(128);not null;index;default:unknown;comment:执行状态" json:"status"`
	Message     string              `gorm:"type:text;comment:执行消息或错误信息" json:"message"`
	JobData     datatypes.JSON      `gorm:"type:jsonb;comment:任务数据(包含提醒和删除的任务列表)" json:"jobData"`
}

// TableName 指定表名
func (CronJobRecord) TableName() string {
	return "cron_job_records"
}

type CronJobType string

func (c CronJobType) String() string {
	return string(c)
}

const (
	CronJobTypeCleanerFunc CronJobType = "cleaner_function"
)

func GetAllCronJobTypes() []CronJobType {
	return []CronJobType{
		CronJobTypeCleanerFunc,
	}
}

type CronJobConfig struct {
	gorm.Model
	Name    string         `gorm:"type:varchar(128);not null;index;unique;comment:Cronjob配置名称" json:"name"`
	Type    CronJobType    `gorm:"type:varchar(128);not null;index;comment:Cronjob类型" json:"type"`
	Spec    string         `gorm:"type:varchar(128);not null;index;comment:Cron调度规范" json:"spec"`
	Suspend *bool          `gorm:"not null;default:false;comment:是否暂停执行" json:"suspend"`
	Config  datatypes.JSON `gorm:"type:jsonb;comment:Cronjob配置数据" json:"config"`
	EntryID int            `gorm:"type:int;comment:Cronjob标识ID" json:"entry_id"`
}

func (c *CronJobConfig) GetSuspend() bool {
	var v bool
	if c.Suspend != nil {
		v = *c.Suspend
	}
	return v
}

func (CronJobConfig) TableName() string {
	return "cron_job_configs"
}
