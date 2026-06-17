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
	// 管理员智能运维报告任务
	TRIGGER_ADMIN_OPS_REPORT_JOB = "trigger-admin-ops-report-job"
	// 存储专项巡检任务
	TRIGGER_STORAGE_DAILY_AUDIT_JOB = "trigger-storage-daily-audit-job"
	// Billing 基础循环
	TRIGGER_BILLING_BASE_LOOP_JOB = "biling-base-loop"
	// 未来可以扩展其他巡检任务，例如：
	// CHECK_NODE_HEALTH = "check-node-health"
)

type GpuAnalysisServiceInterface interface {
	TriggerAllJobsAnalysis(ctx context.Context) (int, error)
}

type AdminOpsReportServiceInterface interface {
	TriggerAdminOpsReport(ctx context.Context, req TriggerAdminOpsReportRequest) (map[string]any, error)
	TriggerStorageAudit(ctx context.Context, req TriggerStorageAuditRequest) (map[string]any, error)
}

type TriggerAdminOpsReportRequest struct {
	Days          int                          `json:"days,omitempty"`
	LookbackHours int                          `json:"lookback_hours,omitempty"`
	GPUThreshold  int                          `json:"gpu_threshold,omitempty"`
	IdleHours     int                          `json:"idle_hours,omitempty"`
	RunningLimit  int                          `json:"running_limit,omitempty"`
	NodeLimit     int                          `json:"node_limit,omitempty"`
	DryRun        bool                         `json:"dry_run,omitempty"`
	Notification  *OpsReportNotificationPolicy `json:"notification,omitempty"`
}

type OpsReportNotificationPolicy struct {
	Enabled                     bool `json:"enabled,omitempty"`
	NotifyAdmins                bool `json:"notify_admins,omitempty"`
	NotifyJobOwners             bool `json:"notify_job_owners,omitempty"`
	FailureJobThreshold         int  `json:"failure_job_threshold,omitempty"`
	FailureRateThresholdPercent int  `json:"failure_rate_threshold_percent,omitempty"`
	UnhealthyNodeThreshold      int  `json:"unhealthy_node_threshold,omitempty"`
	NetworkAlertThreshold       int  `json:"network_alert_threshold,omitempty"`
	HighRiskNetworkJobThreshold int  `json:"high_risk_network_job_threshold,omitempty"`
	MaxJobOwnerEmails           int  `json:"max_job_owner_emails,omitempty"`
	CooldownHours               int  `json:"cooldown_hours,omitempty"`
}

func normalizeOpsReportNotificationPolicy(policy *OpsReportNotificationPolicy) {
	if policy == nil {
		return
	}
	if policy.FailureJobThreshold <= 0 {
		policy.FailureJobThreshold = 10
	}
	if policy.FailureRateThresholdPercent <= 0 {
		policy.FailureRateThresholdPercent = 15
	}
	if policy.UnhealthyNodeThreshold <= 0 {
		policy.UnhealthyNodeThreshold = 1
	}
	if policy.NetworkAlertThreshold <= 0 {
		policy.NetworkAlertThreshold = 3
	}
	if policy.HighRiskNetworkJobThreshold <= 0 {
		policy.HighRiskNetworkJobThreshold = 1
	}
	if policy.MaxJobOwnerEmails <= 0 {
		policy.MaxJobOwnerEmails = 10
	}
	if policy.CooldownHours <= 0 {
		policy.CooldownHours = 12
	}
}

type TriggerStorageAuditRequest struct {
	Days     int  `json:"days,omitempty"`
	PVCLimit int  `json:"pvc_limit,omitempty"`
	DryRun   bool `json:"dry_run,omitempty"`
}

type BillingServiceInterface interface {
	RunBaseLoopOnce(ctx context.Context) (any, error)
}

// Clients 包含巡检任务所需的客户端
type Clients struct {
	Client             client.Client
	KubeClient         kubernetes.Interface
	PromClient         monitor.PrometheusInterface
	GpuAnalysisService GpuAnalysisServiceInterface
	AdminOpsService    AdminOpsReportServiceInterface
	BillingService     BillingServiceInterface
}

func NewPatrolClients(
	cli client.Client,
	kubeClient kubernetes.Interface,
	promClient monitor.PrometheusInterface,
	gpuAnalysisService GpuAnalysisServiceInterface,
	adminOpsReportService AdminOpsReportServiceInterface,
	billingService BillingServiceInterface,
) *Clients {
	return &Clients{
		Client:             cli,
		KubeClient:         kubeClient,
		PromClient:         promClient,
		GpuAnalysisService: gpuAnalysisService,
		AdminOpsService:    adminOpsReportService,
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
		normalizeOpsReportNotificationPolicy(req.Notification)
		f = func(ctx context.Context) (any, error) {
			return RunTriggerAdminOpsReport(ctx, clients, req)
		}
	case TRIGGER_STORAGE_DAILY_AUDIT_JOB:
		req := TriggerStorageAuditRequest{
			Days:     1,
			PVCLimit: 200,
		}
		if len(jobConfig) > 0 {
			if err := json.Unmarshal(jobConfig, &req); err != nil {
				return nil, err
			}
		}
		if req.Days <= 0 {
			req.Days = 1
		}
		if req.PVCLimit <= 0 {
			req.PVCLimit = 200
		}
		f = func(ctx context.Context) (any, error) {
			return RunTriggerStorageAudit(ctx, clients, req)
		}
	case TRIGGER_BILLING_BASE_LOOP_JOB:
		f = func(ctx context.Context) (any, error) {
			return RunTriggerBillingBaseLoop(ctx, clients)
		}

	default:
		return nil, fmt.Errorf("unsupported patrol job name: %s", jobName)
	}
	return f, nil
}
