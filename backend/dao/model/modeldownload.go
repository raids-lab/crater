package model

import (
	"time"

	"gorm.io/gorm"
)

type ModelSource string

const (
	ModelSourceModelScope  ModelSource = "modelscope"
	ModelSourceHuggingFace ModelSource = "huggingface"
)

type DownloadCategory string

const (
	DownloadCategoryModel   DownloadCategory = "model"
	DownloadCategoryDataset DownloadCategory = "dataset"
)

type ModelDownloadStatus string

const (
	ModelDownloadStatusPending     ModelDownloadStatus = "Pending"
	ModelDownloadStatusDownloading ModelDownloadStatus = "Downloading"
	ModelDownloadStatusPaused      ModelDownloadStatus = "Paused"
	ModelDownloadStatusReady       ModelDownloadStatus = "Ready"
	ModelDownloadStatusFailed      ModelDownloadStatus = "Failed"
)

// ModelDownload 全局唯一的下载任务表
type ModelDownload struct {
	gorm.Model
	Name     string           `gorm:"type:varchar(512);not null;uniqueIndex:idx_download_unique,priority:1"`
	Source   ModelSource      `gorm:"type:varchar(32);not null;default:modelscope;uniqueIndex:idx_download_unique,priority:2"`
	Category DownloadCategory `gorm:"type:varchar(32);not null;default:model;uniqueIndex:idx_download_unique,priority:3"`

	Revision             string              `gorm:"type:varchar(128);uniqueIndex:idx_download_unique,priority:4;comment:版本/分支/commit"`
	Path                 string              `gorm:"type:varchar(512);not null;comment:实际下载路径"`
	SizeBytes            int64               `gorm:"default:0;comment:文件总大小(字节)"`
	DownloadedBytes      int64               `gorm:"default:0;comment:已下载大小(字节)"`
	DownloadSpeed        string              `gorm:"type:varchar(32);comment:下载速度(如: 10MB/s)"`
	Status               ModelDownloadStatus `gorm:"type:varchar(32);not null;default:Pending;comment:下载状态"`
	Message              string              `gorm:"type:text;comment:状态消息(错误信息等)"`
	Logs                 string              `gorm:"type:text;comment:终态时保存的Pod日志(K8s Job被GC后仍可查看)"`
	LogsSavedAt          *time.Time          `gorm:"comment:日志保存时间"`
	Organization         string              `gorm:"type:varchar(128);comment:源站组织或作者"`
	LogoURL              string              `gorm:"type:varchar(512);comment:源站组织头像地址"`
	SourceURL            string              `gorm:"type:varchar(512);comment:源站仓库详情地址"`
	DisplayName          string              `gorm:"type:varchar(256);comment:源站展示名称"`
	SourceDescription    string              `gorm:"type:text;comment:源站简介摘要"`
	SourceReadme         string              `gorm:"type:text;comment:源站README内容(截断保存)"`
	License              string              `gorm:"type:varchar(128);comment:源站许可证"`
	Task                 string              `gorm:"type:varchar(128);comment:源站任务分类"`
	Library              string              `gorm:"type:varchar(128);comment:源站框架或库"`
	ModelType            string              `gorm:"type:varchar(128);comment:源站模型类型"`
	ParameterCount       int64               `gorm:"default:0;comment:模型参数量"`
	SourcePrivate        bool                `gorm:"default:false;comment:源站是否私有"`
	SourceGated          bool                `gorm:"default:false;comment:源站是否需要申请访问"`
	SourceLoginRequired  bool                `gorm:"default:false;comment:源站是否要求登录下载"`
	SourceDownloads      int64               `gorm:"default:0;comment:源站下载次数"`
	SourceLikes          int64               `gorm:"default:0;comment:源站点赞次数"`
	SourceCreatedAt      *time.Time          `gorm:"comment:源站创建时间"`
	SourceUpdatedAt      *time.Time          `gorm:"comment:源站更新时间"`
	MetadataRefreshedAt  *time.Time          `gorm:"comment:源站元数据刷新时间"`
	ModelDatasetSourceID *uint               `gorm:"index;comment:模型或数据集外部来源ID"`
	ModelDatasetSource   *ModelDatasetSource `gorm:"foreignKey:ModelDatasetSourceID"`
	JobName              string              `gorm:"type:varchar(256);comment:K8s Job名称"`
	CreatorID            uint                `gorm:"not null;comment:首个发起下载的用户ID"`
	Creator              User                `gorm:"foreignKey:CreatorID"`
	ReferenceCount       int                 `gorm:"default:0;comment:提交下载需求的用户计数"`
}

// UserModelDownload records users who explicitly submitted a need for this download.
type UserModelDownload struct {
	ID              uint          `gorm:"primaryKey"`
	UserID          uint          `gorm:"not null;comment:用户ID;uniqueIndex:idx_user_download,priority:1"`
	ModelDownloadID uint          `gorm:"not null;comment:下载任务ID;uniqueIndex:idx_user_download,priority:2"`
	CreatedAt       time.Time     `gorm:"comment:用户添加此下载的时间"`
	User            User          `gorm:"foreignKey:UserID"`
	ModelDownload   ModelDownload `gorm:"foreignKey:ModelDownloadID"`
}

type ModelDownloadSubmissionAction string

const (
	ModelDownloadSubmissionCreate ModelDownloadSubmissionAction = "create"
	ModelDownloadSubmissionRetry  ModelDownloadSubmissionAction = "retry"
	ModelDownloadSubmissionResume ModelDownloadSubmissionAction = "resume"
)

type ModelDownloadSubmissionStatus string

const (
	// ModelDownloadSubmissionReserved temporarily occupies one rolling-window
	// slot while the Kubernetes download Job is active.
	ModelDownloadSubmissionReserved ModelDownloadSubmissionStatus = "Reserved"
	// ModelDownloadSubmissionSucceeded starts consuming the rolling window from
	// the time the model or dataset finishes downloading.
	ModelDownloadSubmissionSucceeded ModelDownloadSubmissionStatus = "Succeeded"
	// ModelDownloadSubmissionReleased records an attempt that failed, was paused,
	// or was otherwise canceled and therefore does not consume quota.
	ModelDownloadSubmissionReleased ModelDownloadSubmissionStatus = "Released"
)

// ModelDownloadSubmission records quota reservations for Jobs that may produce
// a completed model or dataset. Reusing an existing public download does not
// create a quota submission.
type ModelDownloadSubmission struct {
	ID              uint                          `gorm:"primaryKey"`
	UserID          uint                          `gorm:"not null;index:idx_mds_uc,priority:1;index:idx_mds_q,priority:1"`
	ModelDownloadID uint                          `gorm:"not null;index"`
	Action          ModelDownloadSubmissionAction `gorm:"type:varchar(16);not null"`
	Status          ModelDownloadSubmissionStatus `gorm:"type:varchar(16);not null;default:Reserved;index:idx_mds_q,priority:2"`
	CreatedAt       time.Time                     `gorm:"not null;index:idx_mds_uc,priority:2"`
	CompletedAt     *time.Time                    `gorm:"index:idx_mds_q,priority:3"`
}
