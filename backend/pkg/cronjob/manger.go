package cronjob

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/cleaner"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/patrol"
	"github.com/raids-lab/crater/pkg/storagegovernance"
	"github.com/raids-lab/crater/pkg/storageindex"
)

type CronJobManager struct {
	Client         client.Client
	KubeClient     kubernetes.Interface
	KubeConfig     *rest.Config
	PromClient     monitor.PrometheusInterface
	cleanerClients *cleaner.Clients
	patrolClients  *patrol.Clients
	cron           *cron.Cron
	cronMutex      sync.RWMutex
}

func NewCronJobManager(
	cli client.Client,
	kubeClient kubernetes.Interface,
	kubeConfig *rest.Config,
	promClient monitor.PrometheusInterface,
	gpuAnalysisService patrol.GpuAnalysisServiceInterface,
	billingService patrol.BillingServiceInterface,
) *CronJobManager {
	decisionEngine := storagegovernance.NewEngine(
		kubeClient,
		kubeConfig,
		promClient,
		storagegovernance.DefaultConstraintConfig(),
	)
	indexService := storageindex.NewService(kubeClient, kubeConfig)

	return &CronJobManager{
		Client:     cli,
		KubeClient: kubeClient,
		KubeConfig: kubeConfig,
		PromClient: promClient,
		cleanerClients: &cleaner.Clients{
			Client:     cli,
			KubeClient: kubeClient,
			PromClient: promClient,
		},
		patrolClients: &patrol.Clients{
			Client:             cli,
			KubeClient:         kubeClient,
			KubeConfig:         kubeConfig,
			PromClient:         promClient,
			GpuAnalysisService: gpuAnalysisService,
			BillingService:     billingService,
			RecordDecision: func(ctx context.Context, jobID string, action string, runErr error) {
				_ = storagegovernance.MarkDecisionExecution(ctx, jobID, action, runErr)
			},
			StorageAgent: func(tenantID string) (*patrol.AgentDecision, error) {
				resp, jobID, err := decisionEngine.DecideAndRecord(context.Background(), storagegovernance.DecisionRequest{
					Username:      tenantID,
					Source:        model.StorageDecisionSourcePatrol,
					TriggerReason: "cronjob analyze-storage-alerts",
				})
				if err != nil {
					return nil, err
				}
				return &patrol.AgentDecision{
					AllowExpand:   resp.AllowExpand,
					ExpandBytes:   resp.ExpandBytes,
					FreezeNewJobs: resp.FreezeNewJobs,
					Reason:        resp.Reason,
					DecisionJobID: jobID,
				}, nil
			},
			StorageAgentStart: func(ctx context.Context, tenantID string) (string, error) {
				return decisionEngine.StartAsyncDecision(ctx, storagegovernance.DecisionRequest{
					Username:      tenantID,
					Source:        model.StorageDecisionSourcePatrol,
					TriggerReason: "cronjob analyze-storage-alerts",
				})
			},
			StorageAgentAwait: awaitStorageDecisionResult,
			StorageIndex:      indexService,
		},
		cron: cron.New(cron.WithLocation(time.Local)),
	}
}

func awaitStorageDecisionResult(ctx context.Context, tenantID, jobID string) (*patrol.AgentDecision, error) {
	const (
		defaultTimeout = 5 * time.Minute
		pollInterval   = 2 * time.Second
	)

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		status, err := storagegovernance.GetDecisionStatus(ctx, jobID)
		if err != nil {
			return nil, fmt.Errorf("query storage decision status failed: %w", err)
		}

		switch status.Status {
		case string(model.StorageDecisionStatusDone):
			if status.Result == nil {
				return nil, fmt.Errorf("storage decision %s finished without result", jobID)
			}
			return &patrol.AgentDecision{
				AllowExpand:   status.Result.AllowExpand,
				ExpandBytes:   status.Result.ExpandBytes,
				FreezeNewJobs: status.Result.FreezeNewJobs,
				Reason:        status.Result.Reason,
				DecisionJobID: jobID,
			}, nil
		case string(model.StorageDecisionStatusError):
			if status.ErrorMsg == "" {
				return nil, fmt.Errorf("storage decision %s failed", jobID)
			}
			return nil, fmt.Errorf("storage decision %s failed: %s", jobID, status.ErrorMsg)
		case string(model.StorageDecisionStatusPending), string(model.StorageDecisionStatusRunning):
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("wait storage decision timeout for user %s job %s: %w", tenantID, jobID, ctx.Err())
			case <-ticker.C:
			}
		default:
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("wait storage decision timeout for user %s job %s: %w", tenantID, jobID, ctx.Err())
			case <-ticker.C:
			}
		}
	}
}

// GetPatrolClients returns the patrol clients
func (cm *CronJobManager) GetPatrolClients() *patrol.Clients {
	return cm.patrolClients
}
