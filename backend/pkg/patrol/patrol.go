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
	// Billing 基础循环
	TRIGGER_BILLING_BASE_LOOP_JOB = "biling-base-loop"
	// 管理员智能运维报告任务
	TRIGGER_ADMIN_OPS_REPORT_JOB = "trigger-admin-ops-report-job"
	// 未来可以扩展其他巡检任务，例如：
	// CHECK_NODE_HEALTH = "check-node-health"
)

type GpuAnalysisServiceInterface interface {
	TriggerAllJobsAnalysis(ctx context.Context) (int, error)
}

type BillingServiceInterface interface {
	RunBaseLoopOnce(ctx context.Context) (any, error)
}

type AdminOpsReportServiceInterface interface {
	TriggerAdminOpsReport(ctx context.Context, req TriggerAdminOpsReportRequest) (map[string]any, error)
}

type TriggerAdminOpsReportRequest struct {
	Days          int  `json:"days,omitempty"`
	LookbackHours int  `json:"lookback_hours,omitempty"`
	GPUThreshold  int  `json:"gpu_threshold,omitempty"`
	IdleHours     int  `json:"idle_hours,omitempty"`
	RunningLimit  int  `json:"running_limit,omitempty"`
	NodeLimit     int  `json:"node_limit,omitempty"`
	DryRun        bool `json:"dry_run,omitempty"`
}

// Clients 包含巡检任务所需的客户端
type Clients struct {
	Client             client.Client
	KubeClient         kubernetes.Interface
	PromClient         monitor.PrometheusInterface
	GpuAnalysisService GpuAnalysisServiceInterface
	BillingService     BillingServiceInterface
	AdminOpsService    AdminOpsReportServiceInterface
}

func NewPatrolClients(
	cli client.Client,
	kubeClient kubernetes.Interface,
	promClient monitor.PrometheusInterface,
	gpuAnalysisService GpuAnalysisServiceInterface,
	adminOpsService AdminOpsReportServiceInterface,
	billingService BillingServiceInterface,
) *Clients {
	return &Clients{
		Client:             cli,
		KubeClient:         kubeClient,
		PromClient:         promClient,
		GpuAnalysisService: gpuAnalysisService,
		AdminOpsService:    adminOpsService,
		BillingService:     billingService,
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
	case TRIGGER_BILLING_BASE_LOOP_JOB:
		f = func(ctx context.Context) (any, error) {
			return RunTriggerBillingBaseLoop(ctx, clients)
		}
	case TRIGGER_ADMIN_OPS_REPORT_JOB:
		req := TriggerAdminOpsReportRequest{
			Days:          1,
			LookbackHours: 1,
			GPUThreshold:  5,
			IdleHours:     1,
			RunningLimit:  20,
			NodeLimit:     10,
		}
		if len(jobConfig) > 0 {
			if err := json.Unmarshal(jobConfig, &req); err != nil {
				return nil, err
			}
		}
		if req.Days <= 0 {
			req.Days = 1
		}
		if req.LookbackHours <= 0 {
			req.LookbackHours = 1
		}
		if req.GPUThreshold <= 0 {
			req.GPUThreshold = 5
		}
		if req.IdleHours <= 0 {
			req.IdleHours = req.LookbackHours
		}
		if req.RunningLimit <= 0 {
			req.RunningLimit = 20
		}
		if req.NodeLimit <= 0 {
			req.NodeLimit = 10
		}
		f = func(ctx context.Context) (any, error) {
			return RunTriggerAdminOpsReport(ctx, clients, req)
		}
	default:
		return nil, fmt.Errorf("unsupported patrol job name: %s", jobName)
	}
	return f, nil
}
