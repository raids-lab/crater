package service

import (
	"testing"

	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

func TestBuildAgentDiagnosticFindingsUsesRecordedEvidence(t *testing.T) {
	snapshot := &AgentJobDiagnosticSnapshot{
		Terminations: []AgentDiagnosticTermination{{Reason: "OOMKilled", ExitCode: 137}},
		Events: []AgentDiagnosticEvent{
			{Reason: "FailedScheduling", Message: "0/3 nodes are available: Insufficient nvidia.com/gpu"},
			{Reason: "ErrImagePull", Message: "failed to pull image"},
		},
	}

	findings := buildAgentDiagnosticFindings(snapshot)
	wantCodes := map[string]bool{
		"oom-killed":             false,
		"insufficient-resources": false,
		"image-pull-failed":      false,
	}
	for _, finding := range findings {
		if _, wanted := wantCodes[finding.Code]; wanted {
			wantCodes[finding.Code] = true
		}
	}
	for code, found := range wantCodes {
		if !found {
			t.Errorf("missing finding %q in %#v", code, findings)
		}
	}
}

func TestBuildAgentDiagnosticFindingsMarksSchedulingEstimateObserved(t *testing.T) {
	snapshot := &AgentJobDiagnosticSnapshot{
		Scheduling: &AgentSchedulingDiagnosis{
			Tasks: []AgentSchedulingTaskDiagnosis{{
				CompatibleNodes:      2,
				ResourceMatchedNodes: 0,
				ResourceShortages:    map[string]int{"nvidia.com/gpu": 2},
			}},
		},
	}

	findings := buildAgentDiagnosticFindings(snapshot)
	if len(findings) != 1 || findings[0].Code != "estimated-resource-shortage" {
		t.Fatalf("findings = %#v", findings)
	}
	if findings[0].Confidence != "observed" {
		t.Fatalf("confidence = %q, want observed", findings[0].Confidence)
	}
}

func TestBuildAgentDiagnosticFindingsDoesNotTreatExit137AsConfirmedOOM(t *testing.T) {
	findings := buildAgentDiagnosticFindings(&AgentJobDiagnosticSnapshot{
		Terminations: []AgentDiagnosticTermination{{ExitCode: 137}},
	})
	if len(findings) != 1 || findings[0].Code != "sigkill-exit" {
		t.Fatalf("findings = %#v, want one sigkill-exit finding", findings)
	}
	if findings[0].Confidence != "observed" {
		t.Fatalf("confidence = %q, want observed", findings[0].Confidence)
	}
}

func TestIsAgentSchedulingPhase(t *testing.T) {
	for _, phase := range []batch.JobPhase{batch.Pending, batch.JobPhase("Prequeue")} {
		if !isAgentSchedulingPhase(phase) {
			t.Errorf("phase %q should be a scheduling phase", phase)
		}
	}
	if isAgentSchedulingPhase(batch.Running) {
		t.Fatal("running job should not be a scheduling phase")
	}
}
