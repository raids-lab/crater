package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type JobCheckpointStatus string

const (
	JobCheckpointStatusReady   JobCheckpointStatus = "ready"
	JobCheckpointStatusMissing JobCheckpointStatus = "missing"
	JobCheckpointStatusDeleted JobCheckpointStatus = "deleted"
)

type JobCheckpoint struct {
	gorm.Model
	JobID       uint                `json:"jobID" gorm:"not null;index:idx_job_checkpoints_job;uniqueIndex:idx_job_checkpoint_job_path;comment:作业ID"`
	JobName     string              `json:"jobName" gorm:"type:varchar(256);not null;index;comment:作业集群名称"`
	UserID      uint                `json:"userID" gorm:"not null;index;comment:用户ID"`
	AccountID   uint                `json:"accountID" gorm:"not null;index;comment:账户ID"`
	Framework   string              `json:"framework" gorm:"type:varchar(32);index;comment:训练框架"`
	Name        string              `json:"name" gorm:"type:varchar(256);not null;comment:checkpoint名称"`
	Path        string              `json:"path" gorm:"type:varchar(1024);not null;uniqueIndex:idx_job_checkpoint_job_path;comment:容器内checkpoint路径"`
	StoragePath string              `json:"storagePath" gorm:"type:varchar(1024);not null;comment:存储根目录下的相对路径"`
	Step        int64               `json:"step" gorm:"index;comment:checkpoint步数，无法识别时为-1"`
	SizeBytes   int64               `json:"sizeBytes" gorm:"not null;default:0;comment:checkpoint大小"`
	ModTime     time.Time           `json:"modTime" gorm:"index;comment:checkpoint最后修改时间"`
	Status      JobCheckpointStatus `json:"status" gorm:"type:varchar(32);not null;default:ready;index;comment:checkpoint状态"`
	Latest      bool                `json:"latest" gorm:"not null;default:false;index;comment:是否为最新checkpoint"`
	Source      string              `json:"source" gorm:"type:varchar(32);comment:来源(scan/manual)"`
	Metadata    datatypes.JSONMap   `json:"metadata" gorm:"comment:checkpoint元数据"`
}
