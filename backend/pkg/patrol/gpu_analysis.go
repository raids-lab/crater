package patrol

import (
	"context"
	"fmt"
)

// TriggerGpuAnalysisRequest 用于接收 CronJob 的配置参数
type TriggerGpuAnalysisRequest struct {
	// 预留参数字段
}

// RunTriggerGpuAnalysis 是实际被 CronJob 调用的函数
func RunTriggerGpuAnalysis(ctx context.Context, clients *Clients) (any, error) {
	// 防御性检查
	if clients.GpuAnalysisService == nil {
		return nil, fmt.Errorf("analysis service is not initialized in patrol clients")
	}

	// 调用接口方法执行占卡/异常检测逻辑
	count, err := clients.GpuAnalysisService.TriggerAllJobsAnalysis(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger gpu analysis patrol: %w", err)
	}

	return map[string]any{
		"status":      "success",
		"type":        "gpu_occupancy_check",
		"queued_jobs": count,
	}, nil
}
