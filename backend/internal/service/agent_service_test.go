package service

import (
	"errors"
	"testing"
)

func TestIsMissingAgentToolAuditColumnError(t *testing.T) {
	t.Parallel()

	err := errors.New(`ERROR: column "execution_backend" of relation "agent_tool_calls" does not exist (SQLSTATE 42703)`)
	if !isMissingAgentToolAuditColumnError(err) {
		t.Fatalf("expected missing audit column error to be detected")
	}

	otherErr := errors.New(`ERROR: column "unknown_field" of relation "agent_tool_calls" does not exist (SQLSTATE 42703)`)
	if isMissingAgentToolAuditColumnError(otherErr) {
		t.Fatalf("expected unrelated column error to be ignored")
	}
}

func TestStripUnsupportedAgentToolAuditUpdates(t *testing.T) {
	t.Parallel()

	updates := map[string]any{
		"result_status":       "success",
		"execution_backend":   "k8s_sandbox_job",
		"sandbox_job_name":    "sandbox-1",
		"script_name":         "inspect_pvc",
		"result_artifact_ref": "s3://artifact",
		"egress_domains":      []string{"example.com"},
	}

	compat := stripUnsupportedAgentToolAuditUpdates(updates)
	if _, ok := compat["execution_backend"]; ok {
		t.Fatalf("execution_backend should be stripped")
	}
	if _, ok := compat["sandbox_job_name"]; ok {
		t.Fatalf("sandbox_job_name should be stripped")
	}
	if _, ok := compat["script_name"]; ok {
		t.Fatalf("script_name should be stripped")
	}
	if _, ok := compat["result_artifact_ref"]; ok {
		t.Fatalf("result_artifact_ref should be stripped")
	}
	if _, ok := compat["egress_domains"]; ok {
		t.Fatalf("egress_domains should be stripped")
	}
	if compat["result_status"] != "success" {
		t.Fatalf("expected non-audit fields to be preserved, got %+v", compat)
	}
}
