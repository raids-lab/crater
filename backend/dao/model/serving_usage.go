package model

import (
	"time"

	"gorm.io/datatypes"
	corev1 "k8s.io/api/core/v1"
)

// ServingUsage is a durable billing checkpoint for one native inference Pod.
// PodUID separates replacements; DeploymentID separates reused service names.
type ServingUsage struct {
	PodUID            string                                  `gorm:"primaryKey;type:varchar(128)"`
	DeploymentID      string                                  `gorm:"type:varchar(128);not null;index"`
	Namespace         string                                  `gorm:"type:varchar(63);not null"`
	ServiceName       string                                  `gorm:"type:varchar(63);not null"`
	UserID            uint                                    `gorm:"not null;index"`
	AccountID         uint                                    `gorm:"not null;index"`
	Resources         datatypes.JSONType[corev1.ResourceList] `gorm:"not null"`
	StartedAt         time.Time
	ObservedUntil     time.Time
	EndedAt           *time.Time
	LastSettledAt     *time.Time
	BilledPointsTotal int64 `gorm:"not null;default:0"`
	UpdatedAt         time.Time
}
