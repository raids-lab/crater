package patrol

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
	// 占卡检测任务
	TRIGGER_GPU_ANALYSIS_JOB = "trigger-gpu-analysis-job"
	// 未来可以扩展其他巡检任务，例如：
	// CHECK_NODE_HEALTH = "check-node-health"
)

type GpuAnalysisServiceInterface interface {
	TriggerAllJobsAnalysis(ctx context.Context) (int, error)
}

// Clients 包含巡检任务所需的客户端
type Clients struct {
	Client             client.Client
	KubeClient         kubernetes.Interface
	PromClient         monitor.PrometheusInterface
	GpuAnalysisService GpuAnalysisServiceInterface
}

func NewPatrolClients(
	cli client.Client,
	kubeClient kubernetes.Interface,
	promClient monitor.PrometheusInterface,
	gpuAnalysisService GpuAnalysisServiceInterface,
) *Clients {
	return &Clients{
		Client:             cli,
		KubeClient:         kubeClient,
		PromClient:         promClient,
		GpuAnalysisService: gpuAnalysisService,
	}
}

// GetPatrolFunc 根据作业名称返回对应的巡检函数
func GetPatrolFunc(jobName string, clients *Clients, jobConfig datatypes.JSON) (util.AnyFunc, error) {
	var f util.AnyFunc
	switch jobName {
	case TRIGGER_GPU_ANALYSIS_JOB:
		// TRIGGER_GPU_ANALYSIS_JOB 不需要 req 参数，但为了保持一致性，仍然定义了结构体
		req := &TriggerGpuAnalysisRequest{}
		if len(jobConfig) > 0 {
			if err := json.Unmarshal(jobConfig, req); err != nil {
				return nil, err
			}
		}
		f = func(ctx context.Context) (any, error) {
			return RunTriggerGpuAnalysis(ctx, clients)
		}

	default:
		return nil, fmt.Errorf("unsupported patrol job name: %s", jobName)
	}
	return f, nil
}
