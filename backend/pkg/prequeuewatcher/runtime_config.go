package prequeuewatcher

import (
	"context"
	"errors"
	"time"

	"github.com/raids-lab/crater/internal/service"
)

func (w *PrequeueWatcher) refreshRuntimeConfig(ctx context.Context) error {
	if w.configService == nil {
		return errors.New("config service is not initialized")
	}
	cfg, err := w.configService.GetPrequeueConfig(ctx)
	if err != nil {
		return err
	}
	return w.applyRuntimeConfig(cfg)
}

func (w *PrequeueWatcher) applyRuntimeConfig(cfg *service.PrequeueRuntimeConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	next := *cfg
	current := w.currentRuntimeConfig()
	w.runtimeConfig.Store(&next)
	if w.activateTicker != nil && current.ActivateTickerIntervalSeconds != next.ActivateTickerIntervalSeconds {
		w.activateTicker.Stop()
		w.activateTicker = time.NewTicker(seconds(next.ActivateTickerIntervalSeconds))
	}
	return nil
}

func (w *PrequeueWatcher) currentRuntimeConfig() *service.PrequeueRuntimeConfig {
	cfg, ok := w.runtimeConfig.Load().(*service.PrequeueRuntimeConfig)
	if !ok || cfg == nil {
		return service.NewPrequeueRuntimeConfig()
	}
	return cfg
}

func seconds(value int64) time.Duration {
	return time.Duration(value) * time.Second
}
