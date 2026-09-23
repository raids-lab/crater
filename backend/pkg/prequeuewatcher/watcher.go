package prequeuewatcher

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/pkg/crclient"
)

const (
	drainInterval      = time.Minute
	maxSubmitsPerRound = 50
)

// PrequeueWatcher drains the jobs left behind by the retired prequeue stage. Submitting them is safe
// because it no longer implies admission: the pod group stops at Pending and the extender judges it
// like any other job on the next scheduling session.
type PrequeueWatcher struct {
	q             *query.Query
	configService *service.ConfigService
	k8sClient     client.Client
	kubeClient    kubernetes.Interface
	serviceMgr    crclient.ServiceManagerInterface
	logger        logr.Logger
}

func New(
	q *query.Query,
	configService *service.ConfigService,
	k8sClient client.Client,
	kubeClient kubernetes.Interface,
	serviceMgr crclient.ServiceManagerInterface,
) *PrequeueWatcher {
	return &PrequeueWatcher{
		q:             q,
		configService: configService,
		k8sClient:     k8sClient,
		kubeClient:    kubeClient,
		serviceMgr:    serviceMgr,
		logger:        ctrl.Log.WithName("prequeue-watcher"),
	}
}

func (w *PrequeueWatcher) NeedLeaderElection() bool {
	return true
}

func (w *PrequeueWatcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(drainInterval)
	defer ticker.Stop()

	for {
		// Returning an error would take the whole manager down, so a failed round only waits.
		if err := w.drainRound(ctx); err != nil {
			w.logger.Error(err, "prequeue drain round failed")
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// drainRound submits one bounded batch so a large backlog spreads over several rounds instead of
// bursting against the apiserver at startup.
func (w *PrequeueWatcher) drainRound(ctx context.Context) error {
	candidates, err := w.listPrequeueJobs(ctx, maxSubmitsPerRound)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return nil
	}

	submitted := 0
	for _, candidate := range candidates {
		activated, activateErr := w.claimAndActivatePrequeueJob(ctx, candidate)
		if activateErr != nil {
			// A broken template or a deleted queue needs a human; the record stays where it is.
			w.logger.Error(activateErr, "unable to submit prequeue job", "job", candidate.JobName)
			continue
		}
		if activated {
			submitted++
		}
	}
	w.logger.Info("submitted jobs left in the retired prequeue state",
		"submitted", submitted, "candidates", len(candidates))
	return nil
}
