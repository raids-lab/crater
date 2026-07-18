package service

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

func TestModelDownloadLimitConfigDefaultsAndUpdate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:model_download_limit_config?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.PrequeueConfig{}); err != nil {
		t.Fatal(err)
	}
	service := NewConfigService(query.Use(db))

	cfg, err := service.GetModelDownloadLimitConfig(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defaultConfig := ModelDownloadLimitConfig{
		Enabled: true, MaxConcurrent: 5, WindowHours: 2, MaxSuccessfulDownloads: 5,
		WhitelistUserIDs: []uint{},
	}
	if !reflect.DeepEqual(*cfg, defaultConfig) {
		t.Fatalf("unexpected default model download limits: %+v", cfg)
	}

	want := ModelDownloadLimitConfig{
		Enabled: false, MaxConcurrent: 3, WindowHours: 6, MaxSuccessfulDownloads: 11,
		WhitelistUserIDs: []uint{9, 9, 11},
	}
	if err := service.UpdateModelDownloadLimitConfig(t.Context(), want); err != nil {
		t.Fatal(err)
	}
	got, err := service.GetModelDownloadLimitConfig(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want.WhitelistUserIDs = []uint{9, 11}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("updated config = %+v, want scalar values from %+v and deduplicated whitelist", *got, want)
	}
	if err := service.UpdateModelDownloadLimitConfig(t.Context(), ModelDownloadLimitConfig{}); err == nil {
		t.Fatal("zero limits should be rejected")
	}
}

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
