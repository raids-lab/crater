package agent

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
)

func TestBuildAgentCapabilitiesExposeUserToolsWithoutMessageHeuristics(t *testing.T) {
	token := util.JWTMessage{}
	capabilities := buildAgentCapabilities(token, map[string]any{"route": "/jobs"})
	enabledTools, ok := capabilities["enabled_tools"].([]string)
	if !ok {
		t.Fatalf("enabled_tools has unexpected type: %T", capabilities["enabled_tools"])
	}
	for _, expected := range []string{
		agentToolResubmitJob,
		agentToolCreateJupyter,
		agentToolGetJobDetail,
		agentToolListUserJobs,
		agentToolRecommendImages,
	} {
		if !containsTool(enabledTools, expected) {
			t.Fatalf("expected %q to be enabled, got %v", expected, enabledTools)
		}
	}
}

func TestBuildAgentCapabilitiesExposeAdminToolsForAdminRole(t *testing.T) {
	token := util.JWTMessage{RolePlatform: model.RoleAdmin}
	capabilities := buildAgentCapabilities(token, map[string]any{"route": "/admin/ops"})
	enabledTools, ok := capabilities["enabled_tools"].([]string)
	if !ok {
		t.Fatalf("enabled_tools has unexpected type: %T", capabilities["enabled_tools"])
	}
	for _, expected := range []string{
		toolGetLatestAuditReport,
		toolBatchStopJobs,
		agentToolGetClusterHealth,
		agentToolListClusterNodes,
	} {
		if !containsTool(enabledTools, expected) {
			t.Fatalf("expected %q to be enabled for admin, got %v", expected, enabledTools)
		}
	}
}

func TestBuildToolConfirmationUsesFormsForMutableDrafts(t *testing.T) {
	mgr := &AgentMgr{}
	jupyterArgs, err := json.Marshal(map[string]any{
		"name":       "agent-demo",
		"image_link": "example/jupyter:latest",
		"cpu":        "2",
		"memory":     "8Gi",
	})
	if err != nil {
		t.Fatalf("marshal jupyter args: %v", err)
	}
	jupyterConfirmation := mgr.buildToolConfirmation(util.JWTMessage{}, agentToolCreateJupyter, jupyterArgs)
	if jupyterConfirmation.Interaction != "form" || jupyterConfirmation.Form == nil {
		t.Fatalf("expected create_jupyter_job confirmation to use form interaction, got %+v", jupyterConfirmation)
	}

	resubmitArgs, err := json.Marshal(map[string]any{
		"job_name": "sg-foo",
		"name":     "test",
		"cpu":      "4",
	})
	if err != nil {
		t.Fatalf("marshal resubmit args: %v", err)
	}
	resubmitConfirmation := mgr.buildToolConfirmation(util.JWTMessage{}, agentToolResubmitJob, resubmitArgs)
	if resubmitConfirmation.Interaction != "form" || resubmitConfirmation.Form == nil {
		t.Fatalf("expected resubmit_job confirmation to use form interaction, got %+v", resubmitConfirmation)
	}
	if !containsField(resubmitConfirmation.Form.Fields, "name") {
		t.Fatalf("expected resubmit form to expose display-name field, got %+v", resubmitConfirmation.Form.Fields)
	}
}

func TestNormalizeJobTypes(t *testing.T) {
	got := normalizeJobTypes([]string{" custom ", "Jupyter", "自定义", "unknown"})
	want := []string{string(model.JobTypeCustom), string(model.JobTypeJupyter)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeJobTypes() = %v, want %v", got, want)
	}
}

func TestRunOpsScriptIsConfirmationTool(t *testing.T) {
	if !isAgentConfirmTool(agentToolRunOpsScript) {
		t.Fatalf("expected %q to be a confirmation tool", agentToolRunOpsScript)
	}
}

func TestNewOpsToolsAreReadOnlyOrConfirm(t *testing.T) {
	readOnlyTools := []string{
		agentToolListStoragePVCs,
		agentToolGetPVCDetail,
		agentToolGetPVCEvents,
		agentToolInspectJobStorage,
		agentToolStorageCapacity,
		agentToolNodeNetwork,
		agentToolDiagnoseJobNet,
		agentToolWebSearch,
		agentToolSandboxGrep,
	}
	for _, toolName := range readOnlyTools {
		if !isAgentReadOnlyTool(toolName) {
			t.Fatalf("expected %q to be read-only", toolName)
		}
		if isAgentConfirmTool(toolName) {
			t.Fatalf("expected %q not to require confirmation", toolName)
		}
	}
	if !isAgentConfirmTool(agentToolRunOpsScript) {
		t.Fatalf("expected %q to be confirmation tool", agentToolRunOpsScript)
	}
}

func TestAdminOnlyOpsToolAccessCheck(t *testing.T) {
	adminToken := util.JWTMessage{RolePlatform: model.RoleAdmin}
	userToken := util.JWTMessage{RolePlatform: model.RoleUser}

	if err := ensureOpsAdmin(adminToken, agentToolWebSearch); err != nil {
		t.Fatalf("admin should pass ensureOpsAdmin: %v", err)
	}
	if err := ensureOpsAdmin(userToken, agentToolWebSearch); err == nil {
		t.Fatalf("non-admin should be rejected by ensureOpsAdmin")
	}
	if !isAgentAdminOnlyTool(agentToolWebSearch) {
		t.Fatalf("expected %q to be admin-only", agentToolWebSearch)
	}
}

func TestIsChatSessionSource(t *testing.T) {
	cases := []struct {
		source string
		want   bool
	}{
		{source: "", want: true},
		{source: "chat", want: true},
		{source: " Chat ", want: true},
		{source: "ops_audit", want: false},
		{source: "system", want: false},
		{source: "benchmark", want: false},
	}

	for _, tc := range cases {
		if got := isChatSessionSource(tc.source); got != tc.want {
			t.Fatalf("isChatSessionSource(%q) = %v, want %v", tc.source, got, tc.want)
		}
	}
}

func containsTool(tools []string, target string) bool {
	for _, tool := range tools {
		if tool == target {
			return true
		}
	}
	return false
}

func containsField(fields []AgentToolField, key string) bool {
	for _, field := range fields {
		if field.Key == key {
			return true
		}
	}
	return false
}
