package util

import (
	"strings"
	"testing"
	"time"
)

func TestMapToStruct_BasicAndTags(t *testing.T) {
	t.Parallel()

	type sample struct {
		BackfillEnabled bool          `json:"backfill_enabled"`
		QueueQuota      bool          `yaml:"queue_quota_enabled"`
		Tolerance       int           `json:"normal_job_waiting_tolerance_seconds"`
		ActivateTick    time.Duration `json:"activateTickerInterval"`
		Retries         uint          `mapstructure:"retries"`
		Ratio           float64       `json:"ratio"`
		Name            string
	}

	dst := sample{}
	err := MapToStruct(map[string]string{
		"backfill_enabled":                     "true",
		"queue_quota_enabled":                  "false",
		"normal_job_waiting_tolerance_seconds": "300",
		"activateTickerInterval":               "2m",
		"retries":                              "4",
		"ratio":                                "0.75",
		"name":                                 "watcher",
	}, &dst)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !dst.BackfillEnabled {
		t.Fatalf("expected BackfillEnabled true")
	}
	if dst.QueueQuota {
		t.Fatalf("expected QueueQuota false")
	}
	if dst.Tolerance != 300 {
		t.Fatalf("expected Tolerance 300, got %d", dst.Tolerance)
	}
	if dst.ActivateTick != 2*time.Minute {
		t.Fatalf("expected ActivateTick 2m, got %s", dst.ActivateTick)
	}
	if dst.Retries != 4 {
		t.Fatalf("expected Retries 4, got %d", dst.Retries)
	}
	if dst.Ratio != 0.75 {
		t.Fatalf("expected Ratio 0.75, got %v", dst.Ratio)
	}
	if dst.Name != "watcher" {
		t.Fatalf("expected Name watcher, got %q", dst.Name)
	}
}

func TestMapToStruct_DurationSecondsFallback(t *testing.T) {
	t.Parallel()

	type sample struct {
		RepairTickerInterval time.Duration `json:"repairTickerInterval"`
	}

	dst := sample{}
	err := MapToStruct(map[string]string{
		"repairTickerInterval": "120",
	}, &dst)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if dst.RepairTickerInterval != 120*time.Second {
		t.Fatalf("expected 120s, got %s", dst.RepairTickerInterval)
	}
}

func TestMapToStruct_PointerField(t *testing.T) {
	t.Parallel()

	type sample struct {
		Threshold *int `json:"threshold"`
	}

	dst := sample{}
	err := MapToStruct(map[string]string{"threshold": "9"}, &dst)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if dst.Threshold == nil || *dst.Threshold != 9 {
		t.Fatalf("expected Threshold pointer value 9")
	}
}

func TestMapToStruct_MissingFieldKeepsDefault(t *testing.T) {
	t.Parallel()

	type sample struct {
		BackfillEnabled bool `json:"backfill_enabled"`
		Tolerance       int  `json:"normal_job_waiting_tolerance_seconds"`
	}

	dst := sample{BackfillEnabled: true, Tolerance: 300}
	err := MapToStruct(map[string]string{
		"backfill_enabled": "false",
	}, &dst)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if dst.BackfillEnabled {
		t.Fatalf("expected BackfillEnabled false")
	}
	if dst.Tolerance != 300 {
		t.Fatalf("expected default Tolerance 300, got %d", dst.Tolerance)
	}
}

func TestMapToStruct_InvalidValue(t *testing.T) {
	t.Parallel()

	type sample struct {
		BackfillEnabled bool `json:"backfill_enabled"`
	}

	dst := sample{}
	err := MapToStruct(map[string]string{"backfill_enabled": "not-bool"}, &dst)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "backfill_enabled") {
		t.Fatalf("expected key name in error, got %v", err)
	}
}

func TestMapToStruct_InvalidInput(t *testing.T) {
	t.Parallel()

	type sample struct{ Value int }

	if err := MapToStruct(nil, sample{}); err == nil {
		t.Fatal("expected error for non-pointer")
	}

	var nilPtr *sample
	if err := MapToStruct(nil, nilPtr); err == nil {
		t.Fatal("expected error for nil pointer")
	}
}
