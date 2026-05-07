package prequeuewatcher

import (
	"context"
	"errors"
	"time"

	"github.com/raids-lab/crater/dao/model"
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

func (w *PrequeueWatcher) applyRuntimeConfig(cfg *model.PrequeueRuntimeConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	next := *cfg
	current := w.currentRuntimeConfig()
	w.runtimeConfig.Store(&next)
	if w.activateTicker != nil && current.ActivateTickerIntervalSeconds != next.ActivateTickerIntervalSeconds {
		w.activateTicker.Stop()
		w.activateTicker = time.NewTicker(time.Duration(next.ActivateTickerIntervalSeconds) * time.Second)
	}
	return nil
}

func (w *PrequeueWatcher) currentRuntimeConfig() *model.PrequeueRuntimeConfig {
	cfg, ok := w.runtimeConfig.Load().(*model.PrequeueRuntimeConfig)
	if !ok || cfg == nil {
		return model.NewPrequeueRuntimeConfig()
	}
	return cfg
}
