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

	Revision        string              `gorm:"type:varchar(128);uniqueIndex:idx_download_unique,priority:4;comment:版本/分支/commit"`
	Path            string              `gorm:"type:varchar(512);not null;comment:实际下载路径"`
	SizeBytes       int64               `gorm:"default:0;comment:文件总大小(字节)"`
	DownloadedBytes int64               `gorm:"default:0;comment:已下载大小(字节)"`
	DownloadSpeed   string              `gorm:"type:varchar(32);comment:下载速度(如: 10MB/s)"`
	Status          ModelDownloadStatus `gorm:"type:varchar(32);not null;default:Pending;comment:下载状态"`
	Message         string              `gorm:"type:text;comment:状态消息(错误信息等)"`
	JobName         string              `gorm:"type:varchar(256);comment:K8s Job名称"`
	CreatorID       uint                `gorm:"not null;comment:首个发起下载的用户ID"`
	Creator         User                `gorm:"foreignKey:CreatorID"`
	ReferenceCount  int                 `gorm:"default:0;comment:引用计数"`
}

// UserModelDownload 用户与下载的关联表
type UserModelDownload struct {
	ID              uint          `gorm:"primaryKey"`
	UserID          uint          `gorm:"not null;comment:用户ID;uniqueIndex:idx_user_download,priority:1"`
	ModelDownloadID uint          `gorm:"not null;comment:下载任务ID;uniqueIndex:idx_user_download,priority:2"`
	CreatedAt       time.Time     `gorm:"comment:用户添加此下载的时间"`
	User            User          `gorm:"foreignKey:UserID"`
	ModelDownload   ModelDownload `gorm:"foreignKey:ModelDownloadID"`
}
