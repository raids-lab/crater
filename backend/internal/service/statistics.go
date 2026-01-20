package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
)

type StatisticsService struct{}

var Statistics = &StatisticsService{}

const (
	// 消除 Magic String
	ResNameCPU    = "cpu"
	ResNameMemory = "memory"

	// 内存单位配置
	// 1 << 30 = 1024^3 = GiB
	// 1 << 20 = 1024^2 = MiB
	// 1       = Byte
	ConfigMemDivisor = 1 << 30
	ConfigMemLabel   = "内存 (GiB)"

	// CPU 单位配置
	// 1000.0 = Core (1核 = 1000m)
	// 1.0    = MilliCore (毫核)
	ConfigCPUDivisor = 1000.0
	ConfigCPULabel   = "CPU (Core)"

	DaysInWeek      = 7
	PercentageScale = 100.0
)

// GetResourceStatistics 获取资源统计信息 (主入口，负责编排)
func (s *StatisticsService) GetResourceStatistics(ctx context.Context, req *payload.StatisticsReq) (*payload.StatisticsResp, error) {
	// 1. 准备工作：加载资源元数据
	resourceMap, err := s.loadResourceMetadata(ctx)
	if err != nil {
		klog.Errorf("failed to load resource metadata: %v", err)
		resourceMap = make(map[string]model.Resource)
	}

	// 2. 准备查询并获取作业数据
	jobs, err := s.fetchStatsJobs(ctx, req)
	if err != nil {
		return nil, err
	}

	// 3. 初始化时间桶
	buckets := s.initStatsBuckets(req)

	// 4. 核心计算：遍历作业填充 buckets，并返回总量原始数据
	rawTotalUsage := s.calculateJobUsage(jobs, buckets, req)

	// 5. 数据格式化：处理 Series 和 TotalUsage
	// 处理 Series 小数点
	for i := range buckets {
		s.roundUsageMap(buckets[i].Usage)
	}
	// 处理 TotalUsage 注入元数据
	finalTotalUsage := s.formatStatsUsage(rawTotalUsage, resourceMap)

	return &payload.StatisticsResp{
		TotalUsage: finalTotalUsage,
		Series:     buckets,
	}, nil
}

// -------------------------------------------------------------------------
// 下面是提取出的私有辅助方法，用于降低主函数复杂度
// -------------------------------------------------------------------------

// fetchStatsJobs 构建查询条件并获取作业列表
func (s *StatisticsService) fetchStatsJobs(ctx context.Context, req *payload.StatisticsReq) ([]*model.Job, error) {
	q := query.Job
	jobDo := q.WithContext(ctx).Unscoped()

	// 过滤范围 (Scope)
	switch req.Scope {
	case payload.ScopeUser:
		if req.TargetID == 0 {
			return nil, fmt.Errorf("targetID is required for user scope")
		}
		jobDo = jobDo.Where(q.UserID.Eq(req.TargetID))
	case payload.ScopeAccount:
		if req.TargetID == 0 {
			return nil, fmt.Errorf("targetID is required for account scope")
		}
		jobDo = jobDo.Where(q.AccountID.Eq(req.TargetID))
	case payload.ScopeCluster:
		// Cluster 级别查询所有
	}

	// 过滤时间
	jobDo = jobDo.Where(q.RunningTimestamp.Lt(req.EndTime)).
		Where(q.RunningTimestamp.Neq(time.Time{})). // 排除未开始运行的
		Where(
			q.WithContext(ctx).
				Where(q.CompletedTimestamp.Gt(req.StartTime)).
				Or(q.CompletedTimestamp.Eq(time.Time{})),
		)

	return jobDo.Find()
}

// initStatsBuckets 初始化时间分桶
func (s *StatisticsService) initStatsBuckets(req *payload.StatisticsReq) []payload.TimePointData {
	buckets := make([]payload.TimePointData, 0)
	currentTime := req.StartTime
	for currentTime.Before(req.EndTime) {
		buckets = append(buckets, payload.TimePointData{
			Timestamp: currentTime,
			Usage:     make(map[string]float64),
		})
		if req.Step == payload.StepWeek {
			currentTime = currentTime.AddDate(0, 0, DaysInWeek)
		} else {
			currentTime = currentTime.AddDate(0, 0, 1)
		}
	}
	return buckets
}

// calculateJobUsage 核心计算逻辑：计算作业在时间桶内的资源占用
func (s *StatisticsService) calculateJobUsage(
	jobs []*model.Job,
	buckets []payload.TimePointData,
	req *payload.StatisticsReq,
) map[string]float64 {
	rawTotalUsage := make(map[string]float64)

	for _, job := range jobs {
		// 解析作业的资源配额
		jobRes := s.parseJobResources(job.Resources.Data())
		if len(jobRes) == 0 {
			continue
		}

		// 确定作业起止时间
		jobStart := job.RunningTimestamp
		jobEnd := job.CompletedTimestamp
		if jobEnd.IsZero() {
			jobEnd = time.Now()
			if jobEnd.After(req.EndTime) {
				jobEnd = req.EndTime
			}
		}

		// 遍历每个时间桶
		for i := range buckets {
			bucketStart := buckets[i].Timestamp
			var bucketEnd time.Time
			if req.Step == payload.StepWeek {
				bucketEnd = bucketStart.AddDate(0, 0, DaysInWeek)
			} else {
				bucketEnd = bucketStart.AddDate(0, 0, 1)
			}

			// 计算交集并累加
			overlapDuration := s.calculateOverlapDuration(jobStart, jobEnd, bucketStart, bucketEnd)
			if overlapDuration > 0 {
				hours := overlapDuration.Hours()
				for resName, amount := range jobRes {
					usage := amount * hours

					// 累加到 Bucket
					if _, ok := buckets[i].Usage[resName]; !ok {
						buckets[i].Usage[resName] = 0
					}
					buckets[i].Usage[resName] += usage

					// 累加到 Total
					if _, ok := rawTotalUsage[resName]; !ok {
						rawTotalUsage[resName] = 0
					}
					rawTotalUsage[resName] += usage
				}
			}
		}
	}
	return rawTotalUsage
}

// formatStatsUsage 格式化统计结果，注入 Label 和 Type
func (s *StatisticsService) formatStatsUsage(
	rawTotalUsage map[string]float64,
	resourceMap map[string]model.Resource,
) map[string]payload.ResourceDetail {
	finalTotalUsage := make(map[string]payload.ResourceDetail)

	for resName, usage := range rawTotalUsage {
		usage = math.Round(usage*PercentageScale) / PercentageScale

		detail := payload.ResourceDetail{
			Usage: usage,
			Label: resName,
			Type:  "common",
		}

		// 匹配元数据
		if resName == ResNameCPU {
			detail.Label = ConfigCPULabel
			detail.Type = "common"
		} else if resName == ResNameMemory {
			detail.Label = ConfigMemLabel
			detail.Type = "common"
		} else if res, ok := resourceMap[resName]; ok {
			detail.Label = res.Label
			if res.Type != nil {
				detail.Type = string(*res.Type)
			} else {
				detail.Type = "custom"
			}
		} else if strings.Contains(resName, "gpu") {
			detail.Type = "gpu"
		}

		finalTotalUsage[resName] = detail
	}
	return finalTotalUsage
}

// loadResourceMetadata 加载所有资源定义到内存 Map 中
func (s *StatisticsService) loadResourceMetadata(ctx context.Context) (map[string]model.Resource, error) {
	// 查询 Resource 表
	resources, err := query.Resource.WithContext(ctx).Find()
	if err != nil {
		return nil, err
	}

	resMap := make(map[string]model.Resource)
	for _, r := range resources {
		// r 是 *model.Resource 指针，如果不为空，解引用存入 Map
		if r != nil {
			resMap[r.ResourceName] = *r
		}
	}
	return resMap, nil
}

// calculateOverlapDuration 计算两个时间段的交集时长
func (s *StatisticsService) calculateOverlapDuration(jobStart, jobEnd, bucketStart, bucketEnd time.Time) time.Duration {
	start := jobStart
	if bucketStart.After(start) {
		start = bucketStart
	}

	end := jobEnd
	if bucketEnd.Before(end) {
		end = bucketEnd
	}

	if end.After(start) {
		return end.Sub(start)
	}
	return 0
}

// parseJobResources 解析 ResourceList 为 float64 map
func (s *StatisticsService) parseJobResources(resList v1.ResourceList) map[string]float64 {
	result := make(map[string]float64)
	if resList == nil {
		return result
	}

	for name, quantity := range resList {
		resName := string(name)

		// 忽略 requests/limits 前缀
		if strings.HasPrefix(resName, "requests.") || strings.HasPrefix(resName, "limits.") {
			continue
		}

		var val float64

		switch resName {
		case ResNameCPU:
			// 获取毫核数 (m)，然后除以配置的除数
			// 如果 ConfigCPUDivisor 是 1000，则结果为 Core
			// 如果 ConfigCPUDivisor 是 1，则结果为 mCore
			val = float64(quantity.MilliValue()) / ConfigCPUDivisor

		case ResNameMemory:
			// 获取字节数 (Byte)，然后除以配置的除数
			// 如果 ConfigMemDivisor 是 1<<30，则结果为 GiB
			val = float64(quantity.Value()) / float64(ConfigMemDivisor)

		default:
			// 其他资源（GPU等）
			val = quantity.AsApproximateFloat64()
		}

		result[resName] = val
	}
	return result
}

func (s *StatisticsService) roundUsageMap(usage map[string]float64) {
	for k, v := range usage {
		usage[k] = math.Round(v*PercentageScale) / PercentageScale
	}
}
