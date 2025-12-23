package cronjob

import (
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/pkg/cleaner"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/patrol"
)

type CronJobManager struct {
	Client         client.Client
	KubeClient     kubernetes.Interface
	PromClient     monitor.PrometheusInterface
	cleanerClients *cleaner.Clients
	patrolClients  *patrol.Clients
	cron           *cron.Cron
	cronMutex      sync.RWMutex
}

func NewCronJobManager(
	cli client.Client,
	kubeClient kubernetes.Interface,
	promClient monitor.PrometheusInterface,
	gpuAnalysisService patrol.GpuAnalysisServiceInterface,
) *CronJobManager {
	return &CronJobManager{
		Client:     cli,
		KubeClient: kubeClient,
		PromClient: promClient,
		cleanerClients: &cleaner.Clients{
			Client:     cli,
			KubeClient: kubeClient,
			PromClient: promClient,
		},
		patrolClients: &patrol.Clients{
			Client:             cli,
			KubeClient:         kubeClient,
			PromClient:         promClient,
			GpuAnalysisService: gpuAnalysisService,
		},
		cron: cron.New(cron.WithLocation(time.Local)),
	}
}
