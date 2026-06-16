package storagegovernance

import (
	"strings"
	"testing"
	"time"

	"github.com/raids-lab/crater/pkg/llm"
)

func TestApplySafetyConstraintsClampsExpansion(t *testing.T) {
	cfg := DefaultConstraintConfig()
	cfg.MinPlatformReservedBytes = 10
	cfg.MinPlatformReservedRatio = 0
	snapshot := DecisionSnapshot{
		Username:               "alice",
		CurrentUsageBytes:      95,
		TheoreticalQuotaBytes:  100,
		UsageRatio:             0.95,
		PlatformTotalBytes:     1000,
		PlatformAvailableBytes: 400,
	}
	decision := llm.LLMDecisionResponse{
		AllowExpand: true,
		ExpandBytes: 80,
		Reason:      "raw decision",
	}

	finalDecision, evaluation := ApplySafetyConstraints(snapshot, decision, cfg, time.Now())

	if !finalDecision.AllowExpand {
		t.Fatalf("expected expansion to remain enabled after clamping")
	}
	if finalDecision.ExpandBytes != 30 {
		t.Fatalf("expected expansion to be clamped to 30, got %d", finalDecision.ExpandBytes)
	}
	if !evaluation.Adjusted {
		t.Fatalf("expected evaluation to mark the decision as adjusted")
	}
}

func TestApplySafetyConstraintsForcesFreezeWhenOverQuota(t *testing.T) {
	cfg := DefaultConstraintConfig()
	snapshot := DecisionSnapshot{
		Username:               "alice",
		CurrentUsageBytes:      120,
		TheoreticalQuotaBytes:  100,
		UsageRatio:             1.20,
		PlatformTotalBytes:     1000,
		PlatformAvailableBytes: 500,
	}
	decision := llm.LLMDecisionResponse{
		AllowExpand:   false,
		ExpandBytes:   0,
		FreezeNewJobs: false,
		Reason:        "raw decision",
	}

	finalDecision, evaluation := ApplySafetyConstraints(snapshot, decision, cfg, time.Now())

	if !finalDecision.FreezeNewJobs {
		t.Fatalf("expected freeze_new_jobs to be forced on when user is over quota")
	}
	if finalDecision.AllowExpand {
		t.Fatalf("expected allow_expand to remain disabled when user is over quota")
	}
	if !evaluation.Adjusted {
		t.Fatalf("expected evaluation to mark the decision as adjusted")
	}
	if strings.Contains(finalDecision.Reason, "建议扩容") || strings.Contains(finalDecision.Reason, "优先扩容") {
		t.Fatalf("expected rewritten final reason to stop supporting expansion, got %q", finalDecision.Reason)
	}
	if !strings.Contains(finalDecision.Reason, "冻结新作业") {
		t.Fatalf("expected rewritten final reason to mention freezing new jobs, got %q", finalDecision.Reason)
	}
}

func TestApplySafetyConstraintsBlocksExpansionWhenAlreadyOverQuota(t *testing.T) {
	cfg := DefaultConstraintConfig()
	snapshot := DecisionSnapshot{
		Username:               "alice",
		CurrentUsageBytes:      120,
		TheoreticalQuotaBytes:  100,
		UsageRatio:             1.20,
		PlatformTotalBytes:     1000,
		PlatformAvailableBytes: 500,
	}
	decision := llm.LLMDecisionResponse{
		AllowExpand:   true,
		ExpandBytes:   20,
		FreezeNewJobs: false,
		Reason:        "建议扩容保护作业",
	}

	finalDecision, evaluation := ApplySafetyConstraints(snapshot, decision, cfg, time.Now())

	if finalDecision.AllowExpand {
		t.Fatalf("expected expansion to be disabled when user is already over quota")
	}
	if finalDecision.ExpandBytes != 0 {
		t.Fatalf("expected expand_bytes to be reset to 0, got %d", finalDecision.ExpandBytes)
	}
	if !finalDecision.FreezeNewJobs {
		t.Fatalf("expected freeze_new_jobs to be forced on when user is already over quota")
	}
	if !evaluation.Adjusted {
		t.Fatalf("expected evaluation to mark the decision as adjusted")
	}
	if strings.Contains(finalDecision.Reason, "建议扩容") {
		t.Fatalf("expected final reason to be rewritten without expansion wording, got %q", finalDecision.Reason)
	}
}
