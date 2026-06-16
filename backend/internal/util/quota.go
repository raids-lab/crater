//nolint:lll,mnd // Quota checks keep SQL and percentage thresholds inline for operational clarity.
package util

import (
	"fmt"

	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

// CheckStorageQuota 检查用户存储是否超过理论配额，或作业是否被管理员冻结。
// 任一条件成立时返回非 nil 错误，调用方应拒绝创建新作业。
func CheckStorageQuota(username string) error {
	db := query.GetDB()

	// 步骤 1：查用户 ID 和 space_quota
	var baseRow struct {
		ID         uint  `gorm:"column:id"`
		SpaceQuota int64 `gorm:"column:space_quota"`
	}
	if err := db.Raw(
		"SELECT id, space_quota FROM users WHERE name = ? AND deleted_at IS NULL",
		username,
	).Scan(&baseRow).Error; err != nil || baseRow.ID == 0 {
		klog.Warningf("CheckStorageQuota: user %q not found or query error, skip. err=%v id=%d", username, err, baseRow.ID)
		return nil
	}

	// 步骤 2：尝试获取 jobs_frozen
	var frozenRow struct {
		JobsFrozen bool `gorm:"column:jobs_frozen"`
	}
	if err := db.Raw("SELECT jobs_frozen FROM users WHERE id = ?", baseRow.ID).Scan(&frozenRow).Error; err == nil && frozenRow.JobsFrozen {
		klog.Infof("CheckStorageQuota: user=%q jobs_frozen=true, blocking job creation", username)
		return fmt.Errorf("管理员已暂停您的新作业创建权限，请联系管理员")
	}

	theoreticalQuota := baseRow.SpaceQuota

	// 步骤 3：尝试获取 original_space_quota（临时扩容时才有值）
	var origRow struct {
		OriginalSpaceQuota *int64 `gorm:"column:original_space_quota"`
	}
	if err := db.Raw("SELECT original_space_quota FROM users WHERE id = ?", baseRow.ID).Scan(&origRow).Error; err == nil && origRow.OriginalSpaceQuota != nil {
		theoreticalQuota = *origRow.OriginalSpaceQuota
	}

	klog.Infof("CheckStorageQuota: user=%q id=%d space_quota=%d original_space_quota=%v theoretical=%d",
		username, baseRow.ID, baseRow.SpaceQuota, origRow.OriginalSpaceQuota, theoreticalQuota)

	// -1 = 无限制，0 = 未设置，均跳过
	if theoreticalQuota <= 0 {
		klog.Infof("CheckStorageQuota: user=%q quota=%d (unlimited/unset), skip", username, theoreticalQuota)
		return nil
	}

	// 步骤 4：从 user_space_sizes 取最近一次记录的用量
	var usage model.UserSpaceSize
	if err := db.Where("user_id = ?", baseRow.ID).First(&usage).Error; err != nil {
		klog.Warningf("CheckStorageQuota: user=%q no user_space_sizes record (err=%v), skip", username, err)
		return nil
	}

	klog.Infof("CheckStorageQuota: user=%q size=%d theoretical=%d (%.1f%%)",
		username, usage.Size, theoreticalQuota, float64(usage.Size)/float64(theoreticalQuota)*100)

	if usage.Size >= theoreticalQuota {
		return fmt.Errorf("存储空间已超过理论配额（已用 %s / 配额 %s），禁止创建新作业",
			FormatStorageSize(usage.Size), FormatStorageSize(theoreticalQuota))
	}
	return nil
}

// FormatStorageSize 将字节数格式化为人类可读的字符串。
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
