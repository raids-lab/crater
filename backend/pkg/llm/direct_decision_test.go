package llm

import "testing"

func TestValidateDirectDecisionConsistencyFieldConflicts(t *testing.T) {
	tests := []struct {
		name     string
		decision LLMDecisionResponse
		wantCode string
	}{
		{
			name: "expand enabled but bytes missing",
			decision: LLMDecisionResponse{
				Reason:      "建议扩容保护作业",
				AllowExpand: true,
				ExpandBytes: 0,
			},
			wantCode: "expand_bytes_missing",
		},
		{
			name: "expand disabled but bytes kept",
			decision: LLMDecisionResponse{
				Reason:      "无需扩容，保持观察",
				AllowExpand: false,
				ExpandBytes: 1024,
			},
			wantCode: "expand_bytes_should_be_zero",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issues := validateDirectDecisionConsistency(tc.decision)
			if !hasValidationIssue(issues, tc.wantCode) {
				t.Fatalf("expected issue %q, got %+v", tc.wantCode, issues)
			}
		})
	}
}

func TestValidateDirectDecisionConsistencyReasonConflicts(t *testing.T) {
	tests := []struct {
		name     string
		decision LLMDecisionResponse
		wantCode string
	}{
		{
			name: "allow expand but reason says no expansion",
			decision: LLMDecisionResponse{
				Reason:      "当前无需扩容，继续观察即可",
				AllowExpand: true,
				ExpandBytes: 1024,
			},
			wantCode: "reason_blocks_expansion",
		},
		{
			name: "disallow expand but reason still supports expansion",
			decision: LLMDecisionResponse{
				Reason:      "建议扩容保护落盘阶段",
				AllowExpand: false,
				ExpandBytes: 0,
			},
			wantCode: "reason_supports_expansion",
		},
		{
			name: "freeze enabled but reason says no freeze",
			decision: LLMDecisionResponse{
				Reason:        "当前不需要冻结新作业，只需观察",
				FreezeNewJobs: true,
				AllowExpand:   false,
				ExpandBytes:   0,
			},
			wantCode: "reason_blocks_freeze",
		},
		{
			name: "freeze disabled but reason still recommends freeze",
			decision: LLMDecisionResponse{
				Reason:        "建议冻结新作业，避免继续上涨",
				FreezeNewJobs: false,
				AllowExpand:   false,
				ExpandBytes:   0,
			},
			wantCode: "reason_supports_freeze",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issues := validateDirectDecisionConsistency(tc.decision)
			if !hasValidationIssue(issues, tc.wantCode) {
				t.Fatalf("expected issue %q, got %+v", tc.wantCode, issues)
			}
		})
	}
}

func TestValidateDirectDecisionConsistencyAlignedDecision(t *testing.T) {
	decision := LLMDecisionResponse{
		Reason:        "usage_ratio 接近阈值且平台仍有余量，建议扩容 21474836480 字节保护作业，本轮不冻结新作业",
		AllowExpand:   true,
		ExpandBytes:   21474836480,
		FreezeNewJobs: false,
	}

	issues := validateDirectDecisionConsistency(decision)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %+v", issues)
	}
}

func hasValidationIssue(issues []directDecisionValidationIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
