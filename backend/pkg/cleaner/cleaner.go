package cleaner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/monitor"
)

const (
	VCJOBAPIVERSION = "batch.volcano.sh/v1alpha1"
	VCJOBKIND       = "Job"
	AIJOBAPIVERSION = "aisystem.github.com/v1alpha1"
	AIJOBKIND       = "AIJob"
)

const (
	CLEAN_LONG_TIME_RUNNING_JOB = "clean-long-time-job"
	CLEAN_LOW_GPU_USAGE_JOB     = "clean-low-gpu-util-job"
	CLEAN_WAITING_JUPYTER_JOB   = "clean-waiting-jupyter-job"
)

// Clients 包含清理任务所需的所有客户端
type Clients struct {
	Client     client.Client
	KubeClient kubernetes.Interface
	PromClient monitor.PrometheusInterface
}

// CleanerFunc 定义清理函数的类型
type CleanerFunc func(ctx context.Context) (any, error)

// GetCleanerFunc 根据作业名称返回对应的清理函数
func GetCleanerFunc(jobName string, clients *Clients, jobConfig datatypes.JSON) (CleanerFunc, error) {
	switch jobName {
	case CLEAN_LONG_TIME_RUNNING_JOB:
		req := &CleanLongTimeRunningJobsRequest{}
		if err := json.Unmarshal(jobConfig, req); err != nil {
			return nil, err
		}
		return func(ctx context.Context) (any, error) {
			return CleanLongTimeRunningJobs(ctx, clients, req)
		}, nil

	case CLEAN_LOW_GPU_USAGE_JOB:
		req := &CleanLowGPUUsageRequest{}
		if err := json.Unmarshal(jobConfig, req); err != nil {
			return nil, err
		}
		return func(ctx context.Context) (any, error) {
			return CleanLowGPUUsageJobs(ctx, clients, req)
		}, nil

	case CLEAN_WAITING_JUPYTER_JOB:
		req := &CancelWaitingJupyterJobsRequest{}
		if err := json.Unmarshal(jobConfig, req); err != nil {
			return nil, err
		}
		return func(ctx context.Context) (any, error) {
			return CleanWaitingJupyterJobs(ctx, clients, req)
		}, nil

	default:
		return nil, fmt.Errorf("unsupported cleaner job name: %s", jobName)
	}
}

// GetWrapCleanerFunc 获取并封装清理函数（GetCleanerFunc + WrapCleanerFunc 的组合）
func GetWrapCleanerFunc(jobName string, clients *Clients, jobConfig datatypes.JSON) (func(), error) {
	cleanerFunc, err := GetCleanerFunc(jobName, clients, jobConfig)
	if err != nil {
		return nil, err
	}
	return WrapCleanerFunc(jobName, cleanerFunc), nil
}

// WrapCleanerFunc 封装清理函数，添加通用的错误处理和记录逻辑
func WrapCleanerFunc(jobName string, cleanerFunc CleanerFunc) func() {
	return func() {
		ctx := context.Background()
		// 执行清理函数
		jobResult, err := cleanerFunc(ctx)
		status := model.CronJobRecordStatusSuccess
		if err != nil {
			status = model.CronJobRecordStatusFailed
			klog.Errorf("CleanerFunc %s failed: %v", jobName, err)
		}

		// 创建作业记录
		rec := &model.CronJobRecord{
			Name:        jobName,
			ExecuteTime: time.Now(),
			Message:     "",
			Status:      status,
		}

		// 将结果序列化为JSON
		if jobResult != nil {
			if data, err := json.Marshal(jobResult); err != nil {
				klog.Errorf("WrapCleanerFunc failed to marshal job result: %v", err)
			} else {
				rec.JobData = datatypes.JSON(data)
			}
		}

		// 保存记录到数据库
		db := query.GetDB()
		if err := db.Model(rec).Create(rec).Error; err != nil {
			klog.Errorf("WrapCleanerFunc failed to create record: %v", err)
		}
	}
}
