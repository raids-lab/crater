package vcjob

import (
	"testing"

	"github.com/raids-lab/crater/dao/model"
)

func TestParseJobScheduleMetadataDefaultsToNormal(t *testing.T) {
	scheduleType, waitingToleranceSeconds, err := ParseJobScheduleMetadata(map[string]string{})
	if err != nil {
		t.Fatalf("ParseJobScheduleMetadata returned error: %v", err)
	}
	if scheduleType != model.ScheduleTypeNormal {
		t.Fatalf("expected normal schedule type, got %d", scheduleType)
	}
	if waitingToleranceSeconds != nil {
		t.Fatalf("expected nil waiting tolerance seconds, got %v", *waitingToleranceSeconds)
	}
}

func TestParseJobScheduleMetadataParsesWaitingTolerance(t *testing.T) {
	scheduleType, waitingToleranceSeconds, err := ParseJobScheduleMetadata(map[string]string{
		AnnotationKeyScheduleType:            "1",
		AnnotationKeyWaitingToleranceSeconds: "120",
	})
	if err != nil {
		t.Fatalf("ParseJobScheduleMetadata returned error: %v", err)
	}
	if scheduleType != model.ScheduleTypeNormal {
		t.Fatalf("expected normal schedule type, got %d", scheduleType)
	}
	if waitingToleranceSeconds == nil || *waitingToleranceSeconds != 120 {
		t.Fatalf("expected waiting tolerance seconds to be 120, got %v", waitingToleranceSeconds)
	}
}

func TestParseJobScheduleMetadataRejectsBackfillWaitingTolerance(t *testing.T) {
	_, _, err := ParseJobScheduleMetadata(map[string]string{
		AnnotationKeyScheduleType:            "0",
		AnnotationKeyWaitingToleranceSeconds: "120",
	})
	if err == nil {
		t.Fatal("expected error for backfill waiting tolerance seconds")
	}
}

func TestParseJobScheduleMetadataKeepsLegacyStringScheduleType(t *testing.T) {
	scheduleType, waitingToleranceSeconds, err := ParseJobScheduleMetadata(map[string]string{
		AnnotationKeyScheduleType: model.ScheduleTypeBackfillName,
	})
	if err != nil {
		t.Fatalf("ParseJobScheduleMetadata returned error: %v", err)
	}
	if scheduleType != model.ScheduleTypeBackfill {
		t.Fatalf("expected backfill schedule type, got %d", scheduleType)
	}
	if waitingToleranceSeconds != nil {
		t.Fatalf("expected nil waiting tolerance seconds, got %v", *waitingToleranceSeconds)
	}
}

func TestApplyScheduleMetadataAnnotationsRoundTrip(t *testing.T) {
	annotations := map[string]string{}

	ApplyScheduleMetadataAnnotations(annotations, model.ScheduleTypeNormal, ptrToInt64(180))
	scheduleType, waitingToleranceSeconds, err := ParseJobScheduleMetadata(annotations)
	if err != nil {
		t.Fatalf("ParseJobScheduleMetadata returned error: %v", err)
	}
	if scheduleType != model.ScheduleTypeNormal {
		t.Fatalf("expected schedule type %d, got %d", model.ScheduleTypeNormal, scheduleType)
	}
	if waitingToleranceSeconds == nil || *waitingToleranceSeconds != 180 {
		t.Fatalf("expected waiting tolerance seconds 180, got %v", waitingToleranceSeconds)
	}
}

func ptrToInt64(value int64) *int64 {
	return &value
}
