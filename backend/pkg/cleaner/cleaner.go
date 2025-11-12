package cleaner

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/util"
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
	CLEAN_WAITING_JUPYTER_JOB   = "clean-waiting-jupyter"
)

// Clients 包含清理任务所需的所有客户端
type Clients struct {
	Client     client.Client
	KubeClient kubernetes.Interface
	PromClient monitor.PrometheusInterface
}

// GetCleanerFunc 根据作业名称返回对应的清理函数
func GetCleanerFunc(jobName string, clients *Clients, jobConfig datatypes.JSON) (util.AnyFunc, error) {
	var f util.AnyFunc
	switch jobName {
	case CLEAN_LONG_TIME_RUNNING_JOB:
		req := &CleanLongTimeRunningJobsRequest{}
		if err := json.Unmarshal(jobConfig, req); err != nil {
			return nil, err
		}
		f = func(ctx context.Context) (any, error) {
			return CleanLongTimeRunningJobs(ctx, clients, req)
		}

	case CLEAN_LOW_GPU_USAGE_JOB:
		req := &CleanLowGPUUsageRequest{}
		if err := json.Unmarshal(jobConfig, req); err != nil {
			return nil, err
		}
		f = func(ctx context.Context) (any, error) {
			return CleanLowGPUUsageJobs(ctx, clients, req)
		}

	case CLEAN_WAITING_JUPYTER_JOB:
		req := &CancelWaitingJupyterJobsRequest{}
		if err := json.Unmarshal(jobConfig, req); err != nil {
			return nil, err
		}
		f = func(ctx context.Context) (any, error) {
			return CleanWaitingJupyterJobs(ctx, clients, req)
		}

	default:
		return nil, fmt.Errorf("unsupported cleaner job name: %s", jobName)
	}
	return f, nil
}
