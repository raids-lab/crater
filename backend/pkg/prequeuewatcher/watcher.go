package prequeuewatcher

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
)

const (
	activationRetryDelay = time.Second
	signalBufferSize     = 1024
)

// PrequeueWatcher coordinates full-scan activation and preemption for prequeue jobs.
type PrequeueWatcher struct {
	q              *query.Query
	queueQuotaSvc  *service.PrequeueService
	configService  *service.ConfigService
	k8sClient      client.Client
	serviceMgr     crclient.ServiceManagerInterface
	logger         logr.Logger
	signalCh       chan struct{}
	activateTicker *time.Ticker
	wakeCh         chan struct{}
	runtimeConfig  atomic.Value

	needScan bool
}

func New(
	q *query.Query,
	queueQuotaSvc *service.PrequeueService,
	configService *service.ConfigService,
	k8sClient client.Client,
	serviceMgr crclient.ServiceManagerInterface,
) *PrequeueWatcher {
	return &PrequeueWatcher{
		q:             q,
		queueQuotaSvc: queueQuotaSvc,
		configService: configService,
		k8sClient:     k8sClient,
		serviceMgr:    serviceMgr,
		logger:        ctrl.Log.WithName("prequeue-watcher"),
		signalCh:      make(chan struct{}, signalBufferSize),
		wakeCh:        make(chan struct{}, 1),
	}
}

func (w *PrequeueWatcher) NeedLeaderElection() bool {
	return true
}

// RequestFullScan schedules a full scan without blocking the caller.
func (w *PrequeueWatcher) RequestFullScan() {
	w.notify()
}

func (w *PrequeueWatcher) notify() {
	select {
	case w.signalCh <- struct{}{}:
	default:
		return
	}
}

func (w *PrequeueWatcher) finalize() {
	if w.activateTicker != nil {
		w.activateTicker.Stop()
	}
}

// Start runs the signal-driven activation loop under the manager lifecycle.
func (w *PrequeueWatcher) Start(ctx context.Context) error {
	if err := w.refreshRuntimeConfig(ctx); err != nil {
		return err
	}
	defer w.finalize()
	cfg := w.currentRuntimeConfig()
	w.needScan = true
	w.activateTicker = time.NewTicker(seconds(cfg.ActivateTickerIntervalSeconds))

	w.logger.Info("prequeue watcher started",
		"activateTickerIntervalSeconds", cfg.ActivateTickerIntervalSeconds,
	)

	if !config.GetConfig().EnableLeaderElection {
		w.logger.Info("prequeue watcher is running without leader election")
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-w.wakeCh:
		case <-w.signalCh:
			w.needScan = true
			w.drainSignals()
		case <-w.activateTicker.C:
			w.needScan = true
		}

		if err := w.refreshRuntimeConfig(ctx); err != nil {
			w.logger.Error(err, "failed to refresh prequeue config")
			w.needScan = true
			w.retryLater()
			continue
		}

		if err := w.runScanIfRequested(ctx); err != nil {
			w.logger.Error(err, "activation round failed")
			w.needScan = true
			w.retryLater()
			continue
		}
		if w.needScan {
			w.kick()
		}
	}
}

func (w *PrequeueWatcher) drainSignals() {
	for {
		select {
		case <-w.signalCh:
			w.needScan = true
		default:
			return
		}
	}
}

func (w *PrequeueWatcher) kick() {
	select {
	case w.wakeCh <- struct{}{}:
	default:
	}
}

func (w *PrequeueWatcher) retryLater() {
	time.AfterFunc(activationRetryDelay, w.kick)
}
