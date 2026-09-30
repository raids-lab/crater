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
	if err := db.AutoMigrate(&model.SystemConfig{}); err != nil {
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

func TestParseSchedulerExtenderConfig(t *testing.T) {
	t.Parallel()

	cfg, err := parseSchedulerExtenderConfig(map[string]string{
		model.ConfigKeySchedulerExtenderEnabled:   "true",
		model.ConfigKeyQueueQuotaEnabled:          "false",
		model.ConfigKeyJobWaitingToleranceSeconds: "300",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Distinct values so a swapped key-to-field mapping cannot pass.
	if !cfg.SchedulerExtenderEnabled || cfg.QueueQuotaEnabled {
		t.Fatalf("expected extender on and queue quota off, got %+v", *cfg)
	}
	if cfg.JobWaitingToleranceSeconds != 300 {
		t.Fatalf("expected waiting tolerance 300, got %d", cfg.JobWaitingToleranceSeconds)
	}
}

func TestParseSchedulerExtenderConfig_MissingKeyKeepsDefault(t *testing.T) {
	t.Parallel()

	cfg, err := parseSchedulerExtenderConfig(map[string]string{
		model.ConfigKeyQueueQuotaEnabled: "true",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.QueueQuotaEnabled {
		t.Fatalf("expected queue quota to be enabled")
	}
	if cfg.SchedulerExtenderEnabled {
		t.Fatalf("expected the extender switch to keep its disabled default")
	}
	if cfg.JobWaitingToleranceSeconds != model.DefaultJobWaitingToleranceSeconds {
		t.Fatalf("expected default waiting tolerance, got %d", cfg.JobWaitingToleranceSeconds)
	}
}

func TestParseSchedulerExtenderConfig_InvalidWaitingTolerance(t *testing.T) {
	t.Parallel()

	_, err := parseSchedulerExtenderConfig(map[string]string{
		model.ConfigKeyJobWaitingToleranceSeconds: "0",
	})
	if err == nil {
		t.Fatal("expected error for invalid waiting tolerance")
	}
	if !strings.Contains(err.Error(), "must be greater than 0") {
		t.Fatalf("expected positive value error, got %v", err)
	}
}

func TestParseSchedulerExtenderConfig_InvalidQueueQuotaFlag(t *testing.T) {
	t.Parallel()

	_, err := parseSchedulerExtenderConfig(map[string]string{
		model.ConfigKeyQueueQuotaEnabled: "oops",
	})
	if err == nil {
		t.Fatal("expected error for invalid queue quota flag")
	}
	if !strings.Contains(err.Error(), model.ConfigKeyQueueQuotaEnabled) {
		t.Fatalf("expected key name in error, got %v", err)
	}
}
