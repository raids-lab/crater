package model

import (
	"time"

	"gorm.io/gorm"
)

// alert model
//
// 记录已经发送的邮件
type Alert struct {
	gorm.Model

	JobName        string    `gorm:"type:varchar(255);not null;comment:作业名" json:"jobName"`
	AlertType      string    `gorm:"type:varchar(255);not null;comment:邮件类型" json:"alertType"`
	AlertTimestamp time.Time `gorm:"comment:邮件发送时间"`
	AllowRepeat    bool      `gorm:"type:boolean;default:false;comment:是否允许重复发送"`
	SendCount      int       `gorm:"not null;comment:邮件发送次数"`
}
