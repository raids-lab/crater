package service

import (
	"strings"
	"testing"

	"github.com/raids-lab/crater/dao/model"
)

func TestParsePrequeueRuntimeConfig(t *testing.T) {
	t.Parallel()

	cfg, err := parsePrequeueRuntimeConfig(map[string]string{
		model.PrequeueBackfillEnabledKey:                  "true",
		model.PrequeueQueueQuotaEnabledKey:                "false",
		model.PrequeueNormalJobWaitingToleranceSecondsKey: "300",
		model.PrequeueActivateTickerIntervalSecondsKey:    "5",
		model.PrequeueMaxTotalActivationsPerRoundKey:      "500",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.BackfillEnabled {
		t.Fatalf("expected backfill to be enabled")
	}
	if cfg.QueueQuotaEnabled {
		t.Fatalf("expected queue quota to be disabled")
	}
	if cfg.NormalJobWaitingToleranceSeconds != 300 {
		t.Fatalf("expected waiting tolerance 300, got %d", cfg.NormalJobWaitingToleranceSeconds)
	}
	if cfg.ActivateTickerIntervalSeconds != 5 {
		t.Fatalf("expected activate ticker 5, got %d", cfg.ActivateTickerIntervalSeconds)
	}
}

func TestParsePrequeueRuntimeConfig_MissingKeyKeepsDefault(t *testing.T) {
	t.Parallel()

	cfg, err := parsePrequeueRuntimeConfig(map[string]string{
		model.PrequeueBackfillEnabledKey: "true",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.BackfillEnabled {
		t.Fatalf("expected backfill to be enabled")
	}
	if cfg.NormalJobWaitingToleranceSeconds != model.PrequeueDefaultNormalJobWaitingToleranceSeconds {
		t.Fatalf("expected default waiting tolerance, got %d", cfg.NormalJobWaitingToleranceSeconds)
	}
	if cfg.ActivateTickerIntervalSeconds != model.PrequeueDefaultActivateTickerIntervalSeconds {
		t.Fatalf("expected default activate ticker, got %d", cfg.ActivateTickerIntervalSeconds)
	}
}

func TestParsePrequeueRuntimeConfig_InvalidWaitingTolerance(t *testing.T) {
	t.Parallel()

	_, err := parsePrequeueRuntimeConfig(map[string]string{
		model.PrequeueNormalJobWaitingToleranceSecondsKey: "0",
	})
	if err == nil {
		t.Fatal("expected error for invalid waiting tolerance")
	}
	if !strings.Contains(err.Error(), "must be greater than 0") {
		t.Fatalf("expected positive value error, got %v", err)
	}
}

func TestParsePrequeueRuntimeConfig_InvalidQueueQuotaFlag(t *testing.T) {
	t.Parallel()

	_, err := parsePrequeueRuntimeConfig(map[string]string{
		model.PrequeueQueueQuotaEnabledKey: "oops",
	})
	if err == nil {
		t.Fatal("expected error for invalid queue quota flag")
	}
	if !strings.Contains(err.Error(), model.PrequeueQueueQuotaEnabledKey) {
		t.Fatalf("expected key name in error, got %v", err)
	}
}
