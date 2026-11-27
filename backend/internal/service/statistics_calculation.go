package service

import (
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
)

const (
	ScopeUser            = "user"
	ScopeAccount         = "account"
	MetricUsage          = "usage"
	MetricEffectiveUsage = "effectiveUsage"
	MetricUtilization    = "utilization"
	resourceCPU          = "cpu"
	resourceMemory       = "memory"
	SecondsPerDay        = 24 * 60 * 60
	SecondsPerHour       = 60 * 60
	bytesPerGiB          = 1 << 30
	mibPerGiB            = 1024
)

// StatisticsDay 返回时间所在 UTC 自然日的零点。
func StatisticsDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// ResourceAmounts 将资源数量转换为界面使用的核、卡和 GiB 单位。
func ResourceAmounts(resources v1.ResourceList) map[string]float64 {
	amounts := make(map[string]float64, len(resources))
	for name, quantity := range resources {
		key := string(name)
		if strings.HasPrefix(key, "requests.") || strings.HasPrefix(key, "limits.") {
			continue
		}
		amount := quantity.AsApproximateFloat64()
		if name == v1.ResourceMemory {
			amount /= bytesPerGiB
		}
		if amount > 0 {
			amounts[key] = amount
		}
	}
	return amounts
}

type usageKey struct {
	user, account uint
	resource      string
}

type dayAccumulator struct {
	day, cutoff time.Time
	resources   map[string]*model.Resource
	rows        map[usageKey]*model.ResourceUsageDaily
}

// add 按作业与当日区间的重叠时长累加分配用量和可评估用量。
func (a *dayAccumulator) add(job *model.Job) {
	if job.JobType == model.JobTypeTensorboard || job.RunningTimestamp.IsZero() {
		return
	}
	start, end := job.RunningTimestamp, job.CompletedTimestamp
	if start.Before(a.day) {
		start = a.day
	}
	if end.IsZero() && job.DeletedAt.Valid {
		end = job.DeletedAt.Time
	}
	if end.IsZero() || end.After(a.cutoff) {
		end = a.cutoff
	}
	seconds := end.Sub(start).Seconds()
	if seconds <= 0 {
		return
	}
	amounts := ResourceAmounts(job.Resources.Data())
	for resource, amount := range amounts {
		key := usageKey{job.UserID, job.AccountID, resource}
		row := a.rows[key]
		if row == nil {
			row = &model.ResourceUsageDaily{StatDate: a.day, UserID: job.UserID, AccountID: job.AccountID,
				ResourceName: resource, CoveredUntil: a.cutoff}
			a.rows[key] = row
		}
		allocated := amount * seconds
		row.UsageSeconds += allocated
		if used, ok := evaluatedAmount(job, resource, amounts, a.resources); ok {
			row.ProfiledUsageSeconds += allocated
			row.EffectiveUsageSeconds += used * seconds
		}
	}
}

// evaluatedAmount 使用已完成作业保存的监控均值，按重叠时长计算有效用量。
func evaluatedAmount(job *model.Job, resource string, amounts map[string]float64,
	metadata map[string]*model.Resource,
) (float64, bool) {
	if job.CompletedTimestamp.IsZero() || job.ProfileData == nil || job.ProfileData.Data() == nil {
		return 0, false
	}
	profile := job.ProfileData.Data()
	var value *float32
	scale := 1.0
	switch resource {
	case resourceCPU:
		value = profile.CPUUsageAvg
	case resourceMemory:
		value, scale = profile.CPUMemAvg, 1.0/mibPerGiB
	default:
		if !isGPU(metadata[resource]) {
			return 0, false
		}
		value, scale = profile.GPUUtilAvg, amounts[resource]
	}
	if value == nil {
		return 0, false
	}
	return float64(*value) * scale, true
}

func isGPU(resource *model.Resource) bool {
	return resource != nil && resource.Type != nil && *resource.Type == model.ResourceTypeGPU
}
