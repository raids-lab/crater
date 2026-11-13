package util

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/pkg/ceph"
	"github.com/raids-lab/crater/pkg/config"
)

// CheckStorageQuota checks the enforced CephFS quota for jobs with storage mounts.
// Jobs without storage mounts must not depend on the storage quota provider.
// Mounted jobs fail closed on provider errors rather than use stale cache entries.
func CheckStorageQuota(
	username string,
	volumeMounts []VolumeMount,
	kubeClient kubernetes.Interface,
	kubeConfig *rest.Config,
) error {
	if !ceph.StorageQuotaEnabled() || len(volumeMounts) == 0 {
		return nil
	}

	var user struct {
		ID    uint   `gorm:"column:id"`
		Space string `gorm:"column:space"`
	}
	if err := query.GetDB().Raw(
		"SELECT id, space FROM users WHERE name = ? AND deleted_at IS NULL",
		username,
	).Scan(&user).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to load user storage path")
	}
	if user.ID == 0 || user.Space == "" {
		return bizerr.Internal.DatabaseError.New("user storage path was not found")
	}

	cfg := config.GetConfig()
	prefixes := ceph.StoragePrefixConfig{
		User: cfg.Storage.Prefix.User, Account: cfg.Storage.Prefix.Account, Public: cfg.Storage.Prefix.Public,
	}
	logicalPath := "/user/" + user.Space
	quota, err := ceph.GetCephDirectoryQuota(
		kubeClient, kubeConfig, ceph.StorageQuotaRookNamespace(), logicalPath, prefixes,
	)
	if err != nil {
		return bizerr.Internal.FileSystemError.Wrap(err,
			"Failed to read the storage quota. "+storageQuotaRetryAdvice)
	}
	if quota <= 0 {
		return nil
	}

	usage, err := ceph.GetCephDirectorySize(
		kubeClient, kubeConfig, ceph.StorageQuotaRookNamespace(), logicalPath, prefixes,
	)
	if err != nil {
		return bizerr.Internal.FileSystemError.Wrap(err,
			"Failed to read current storage usage. "+storageQuotaRetryAdvice)
	}
	klog.Infof(
		"CheckStorageQuota: user=%q size=%d quota=%d (%.1f%%)",
		username, usage, quota, float64(usage)/float64(quota)*percentageMultiplier,
	)
	if usage >= quota {
		return bizerr.Conflict.ResourceStatusError.New(fmt.Sprintf(
			"storage usage has reached the quota (%s used / %s quota); new jobs cannot be created",
			FormatStorageSize(usage), FormatStorageSize(quota),
		))
	}
	return nil
}

const percentageMultiplier = 100

const storageQuotaRetryAdvice = "Retry later, or remove all storage mounts and resubmit if your job does not require shared storage."

// FormatStorageSize formats a byte count for user-facing quota errors.
func FormatStorageSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
