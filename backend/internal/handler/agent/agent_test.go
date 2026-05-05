package agent

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

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
		agentToolCreateWebIDE,
		agentToolCreateCustom,
		agentToolCreatePytorch,
		agentToolCreateTensorflow,
		agentToolCreateImage,
		agentToolGetJobDetail,
		agentToolListImageBuilds,
		agentToolListUserJobs,
		agentToolRecommendImages,
		agentToolK8sListPods,
		agentToolK8sGetService,
		agentToolK8sGetEndpoints,
		agentToolK8sGetIngress,
		agentToolWebSearch,
		agentToolFetchURL,
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

func TestBuildAgentCapabilitiesMergesDynamicLocalCatalog(t *testing.T) {
	token := util.JWTMessage{RolePlatform: model.RoleAdmin}
	capabilities := buildAgentCapabilitiesWithCatalog(
		token,
		map[string]any{"route": "/admin/ops"},
		[]agentLocalToolCatalogEntry{
			{
				Name:             agentToolK8sScaleWL,
				Mode:             "confirm",
				AdminOnly:        true,
				Description:      "调整 Deployment/StatefulSet 副本数。需要管理员确认。",
				ExecutionBackend: "python_local",
			},
			{
				Name:             "get_node_gpu_info",
				Mode:             "read_only",
				AdminOnly:        true,
				Description:      "读取节点 GPU 设备信息。",
				ExecutionBackend: "python_local",
			},
		},
	)

	enabledTools, ok := capabilities["enabled_tools"].([]string)
	if !ok {
		t.Fatalf("enabled_tools has unexpected type: %T", capabilities["enabled_tools"])
	}
	if !containsTool(enabledTools, agentToolK8sScaleWL) {
		t.Fatalf("expected %q to be dynamically enabled, got %v", agentToolK8sScaleWL, enabledTools)
	}
	if !containsTool(enabledTools, "get_node_gpu_info") {
		t.Fatalf("expected dynamic local read tool to be enabled, got %v", enabledTools)
	}

	confirmTools, ok := capabilities["confirm_tools"].([]string)
	if !ok {
		t.Fatalf("confirm_tools has unexpected type: %T", capabilities["confirm_tools"])
	}
	if !containsTool(confirmTools, agentToolK8sScaleWL) {
		t.Fatalf("expected %q to require confirmation, got %v", agentToolK8sScaleWL, confirmTools)
	}

	catalog, ok := capabilities["tool_catalog"].([]map[string]any)
	if !ok {
		t.Fatalf("tool_catalog has unexpected type: %T", capabilities["tool_catalog"])
	}
	foundScale := false
	foundGPU := false
	for _, entry := range catalog {
		if entry["name"] == agentToolK8sScaleWL {
			foundScale = true
			if entry["mode"] != "confirm" {
				t.Fatalf("expected dynamic confirm mode for %q, got %#v", agentToolK8sScaleWL, entry)
			}
		}
		if entry["name"] == "get_node_gpu_info" {
			foundGPU = true
			if entry["description"] != "读取节点 GPU 设备信息。" {
				t.Fatalf("expected dynamic description override, got %#v", entry)
			}
		}
	}
	if !foundScale || !foundGPU {
		t.Fatalf("expected dynamic local catalog entries in tool_catalog, got %#v", catalog)
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
	if !containsField(jupyterConfirmation.Form.Fields, "forwards_text") {
		t.Fatalf("expected jupyter form to expose forwards_text, got %+v", jupyterConfirmation.Form.Fields)
	}

	webideConfirmation := mgr.buildToolConfirmation(util.JWTMessage{}, agentToolCreateWebIDE, jupyterArgs)
	if webideConfirmation.Interaction != "form" || webideConfirmation.Form == nil {
		t.Fatalf("expected create_webide_job confirmation to use form interaction, got %+v", webideConfirmation)
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

	customArgs, err := json.Marshal(map[string]any{
		"name":          "svc-milvus",
		"image_link":    "milvusdb/milvus:v2.4.0",
		"command":       "milvus run standalone",
		"working_dir":   "/workspace",
		"forwards_text": "grpc:19530:ingress",
	})
	if err != nil {
		t.Fatalf("marshal custom args: %v", err)
	}
	customConfirmation := mgr.buildToolConfirmation(util.JWTMessage{}, agentToolCreateCustom, customArgs)
	if customConfirmation.Interaction != "form" || customConfirmation.Form == nil {
		t.Fatalf("expected create_custom_job confirmation to use form interaction, got %+v", customConfirmation)
	}
	if !containsField(customConfirmation.Form.Fields, "forwards_text") {
		t.Fatalf("expected custom form to expose forwards_text, got %+v", customConfirmation.Form.Fields)
	}

	distributedArgs, err := json.Marshal(map[string]any{
		"name":       "dist-train",
		"tasks_json": `[{"name":"master","replicas":1,"image_link":"pytorch:latest","cpu":"8","memory":"32Gi","gpu_count":1},{"name":"worker","replicas":1,"image_link":"pytorch:latest","cpu":"8","memory":"32Gi","gpu_count":1}]`,
	})
	if err != nil {
		t.Fatalf("marshal distributed args: %v", err)
	}
	pytorchConfirmation := mgr.buildToolConfirmation(util.JWTMessage{}, agentToolCreatePytorch, distributedArgs)
	if pytorchConfirmation.Interaction != "form" || pytorchConfirmation.Form == nil {
		t.Fatalf("expected create_pytorch_job confirmation to use form interaction, got %+v", pytorchConfirmation)
	}
	if !containsField(pytorchConfirmation.Form.Fields, "tasks_json") {
		t.Fatalf("expected pytorch form to expose tasks_json, got %+v", pytorchConfirmation.Form.Fields)
	}

	tensorflowConfirmation := mgr.buildToolConfirmation(util.JWTMessage{}, agentToolCreateTensorflow, distributedArgs)
	if tensorflowConfirmation.Interaction != "form" || tensorflowConfirmation.Form == nil {
		t.Fatalf("expected create_tensorflow_job confirmation to use form interaction, got %+v", tensorflowConfirmation)
	}
	if !containsField(tensorflowConfirmation.Form.Fields, "tasks_json") {
		t.Fatalf("expected tensorflow form to expose tasks_json, got %+v", tensorflowConfirmation.Form.Fields)
	}

	imageBuildArgs, err := json.Marshal(map[string]any{
		"mode":        "pip_apt",
		"description": "test image build",
		"base_image":  "crater-harbor.example.com/user/base:latest",
	})
	if err != nil {
		t.Fatalf("marshal image build args: %v", err)
	}
	imageBuildConfirmation := mgr.buildToolConfirmation(util.JWTMessage{}, agentToolCreateImage, imageBuildArgs)
	if imageBuildConfirmation.Interaction != "form" || imageBuildConfirmation.Form == nil {
		t.Fatalf("expected create_image_build confirmation to use form interaction, got %+v", imageBuildConfirmation)
	}
	if !containsField(imageBuildConfirmation.Form.Fields, "mode") {
		t.Fatalf("expected image build form to expose mode field, got %+v", imageBuildConfirmation.Form.Fields)
	}

	registerImageArgs, err := json.Marshal(map[string]any{
		"image_link": "registry.example.com/team/demo:latest",
		"task_type":  "custom",
	})
	if err != nil {
		t.Fatalf("marshal register image args: %v", err)
	}
	registerImageConfirmation := mgr.buildToolConfirmation(util.JWTMessage{}, agentToolRegisterImage, registerImageArgs)
	if registerImageConfirmation.Interaction != "form" || registerImageConfirmation.Form == nil {
		t.Fatalf("expected register_external_image confirmation to use form interaction, got %+v", registerImageConfirmation)
	}
}

func TestNormalizeJobTypes(t *testing.T) {
	got := normalizeJobTypes([]string{" custom ", "Jupyter", "自定义", "unknown"})
	want := []string{string(model.JobTypeCustom), string(model.JobTypeJupyter)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeJobTypes() = %v, want %v", got, want)
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
		agentToolListImageBuilds,
		agentToolGetImageBuild,
		agentToolGetImageAccess,
	}
	for _, toolName := range readOnlyTools {
		if !isAgentReadOnlyTool(toolName) {
			t.Fatalf("expected %q to be read-only", toolName)
		}
		if isAgentConfirmTool(toolName) {
			t.Fatalf("expected %q not to require confirmation", toolName)
		}
	}
	for _, toolName := range []string{
		agentToolCreateImage,
		agentToolManageBuild,
		agentToolRegisterImage,
		agentToolManageAccess,
		agentToolCreatePytorch,
		agentToolCreateTensorflow,
		agentToolK8sScaleWL,
		agentToolK8sLabelNode,
		agentToolK8sTaintNode,
		agentToolRunKubectl,
		agentToolAdminCommand,
	} {
		if !isAgentConfirmTool(toolName) {
			t.Fatalf("expected %q to be confirmation tool", toolName)
		}
	}
}

func TestAdminOnlyOpsToolAccessCheck(t *testing.T) {
	adminToken := util.JWTMessage{RolePlatform: model.RoleAdmin}
	userToken := util.JWTMessage{RolePlatform: model.RoleUser}

	if err := ensureOpsAdmin(adminToken, agentToolSandboxGrep); err != nil {
		t.Fatalf("admin should pass ensureOpsAdmin: %v", err)
	}
	if err := ensureOpsAdmin(userToken, agentToolSandboxGrep); err == nil {
		t.Fatalf("non-admin should be rejected by ensureOpsAdmin")
	}
	if isAgentAdminOnlyTool(agentToolWebSearch) {
		t.Fatalf("expected %q not to be admin-only", agentToolWebSearch)
	}
	if !isAgentAdminOnlyTool(agentToolSandboxGrep) {
		t.Fatalf("expected %q to be admin-only", agentToolSandboxGrep)
	}
}

func TestNotifyJobOwnerRequiresConfirmation(t *testing.T) {
	if !isAgentConfirmTool(toolNotifyJobOwner) {
		t.Fatalf("expected %q to require confirmation", toolNotifyJobOwner)
	}
	if isAgentReadOnlyTool(toolNotifyJobOwner) {
		t.Fatalf("expected %q not to be read-only", toolNotifyJobOwner)
	}
	if isAgentAutoActionTool(toolNotifyJobOwner) {
		t.Fatalf("expected %q not to be an auto-action tool", toolNotifyJobOwner)
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

func TestBuildAgentHistoryIncludesToolCallsAndToolConfirmationContext(t *testing.T) {
	metadata, err := json.Marshal(map[string]any{
		"source":   "tool_confirmation",
		"toolName": agentToolCreateCustom,
		"status":   "rejected",
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	baseTime := time.Now()
	history := buildAgentHistory(
		[]*model.AgentMessage{
			{
				SessionID: "session-1",
				Role:      "assistant",
				Content:   "已取消创建自定义作业。",
				Metadata:  metadata,
				CreatedAt: baseTime.Add(2 * time.Second),
			},
			{
				SessionID: "session-1",
				Role:      "user",
				Content:   "帮我起一个 milvus 服务",
				CreatedAt: baseTime,
			},
		},
		[]*model.AgentToolCall{
			{
				ID:           42,
				SessionID:    "session-1",
				ToolCallID:   "tool-call-42",
				ToolName:     agentToolCreateCustom,
				ToolArgs:     []byte(`{"name":"svc-milvus","image_link":""}`),
				ResultStatus: "rejected",
				CreatedAt:    baseTime.Add(time.Second),
			},
		},
	)

	if len(history) != 3 {
		t.Fatalf("expected 3 history entries, got %d: %#v", len(history), history)
	}

	if history[1]["role"] != "tool" {
		t.Fatalf("expected middle entry to be tool history, got %#v", history[1])
	}
	if got, _ := history[1]["content"].(string); got == "" || !containsSubstring(got, "tool=create_custom_job") || !containsSubstring(got, "status=rejected") {
		t.Fatalf("expected tool history content to include tool/status, got %q", got)
	}

	if got, _ := history[2]["content"].(string); got == "" || !containsSubstring(got, "【上轮工具结果 tool=create_custom_job status=rejected】") {
		t.Fatalf("expected assistant history to be decorated with tool confirmation context, got %q", got)
	}
}

func TestOrderToolCallsForWorkflowUsesPendingConfirmationOrder(t *testing.T) {
	mgr := &AgentMgr{}
	baseTime := time.Now()
	labelArgs := datatypesJSONForTest(t, map[string]any{
		"node_name": "dell-gpu-21",
		"key":       "rdma-status",
		"value":     "isolated",
	})
	taintArgs := datatypesJSONForTest(t, map[string]any{
		"node_name": "dell-gpu-21",
		"key":       "rdma",
		"value":     "degraded",
		"effect":    "NoSchedule",
	})
	toolCalls := []*model.AgentToolCall{
		{
			ID:           20,
			ToolCallID:   "executor-1-action-2",
			ToolName:     agentToolK8sTaintNode,
			ToolArgs:     taintArgs,
			ResultStatus: agentToolStatusSuccess,
			CreatedAt:    baseTime,
		},
		{
			ID:           10,
			ToolCallID:   "executor-1-action-1",
			ToolName:     agentToolK8sLabelNode,
			ToolArgs:     labelArgs,
			ResultStatus: agentToolStatusSuccess,
			CreatedAt:    baseTime,
		},
	}
	workflow := map[string]any{
		"pending_confirmation_ids": []any{"10", "20"},
		"actions": []any{
			map[string]any{
				"action_id":    "action-1",
				"confirm_id":   "10",
				"tool_name":    agentToolK8sLabelNode,
				"tool_call_id": "executor-1-action-1",
				"tool_args":    map[string]any{"node_name": "dell-gpu-21", "key": "rdma-status", "value": "isolated"},
			},
			map[string]any{
				"action_id":    "action-2",
				"confirm_id":   "20",
				"tool_name":    agentToolK8sTaintNode,
				"tool_call_id": "executor-1-action-2",
				"tool_args":    map[string]any{"node_name": "dell-gpu-21", "key": "rdma", "value": "degraded", "effect": "NoSchedule"},
			},
		},
	}

	ordered := mgr.orderToolCallsForWorkflow(toolCalls, workflow)
	if len(ordered) != 2 {
		t.Fatalf("expected 2 ordered tool calls, got %d", len(ordered))
	}
	if ordered[0].ID != 10 || ordered[1].ID != 20 {
		t.Fatalf("expected workflow order [10,20], got [%d,%d]", ordered[0].ID, ordered[1].ID)
	}
}

func datatypesJSONForTest(t *testing.T, payload map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal test payload: %v", err)
	}
	return raw
}

func TestBuildToolOutcomeMessageRejectedUsesSpecificCopy(t *testing.T) {
	mgr := &AgentMgr{}

	if got := mgr.buildToolOutcomeMessage(agentToolCreateCustom, "rejected", nil, ""); got != "已取消创建自定义作业。" {
		t.Fatalf("unexpected create_custom_job rejected copy: %q", got)
	}
	if got := mgr.buildToolOutcomeMessage(agentToolCreateImage, "rejected", nil, ""); got != "已取消创建镜像构建任务。" {
		t.Fatalf("unexpected create_image_build rejected copy: %q", got)
	}
	if got := mgr.buildToolOutcomeMessage("unknown_tool", "rejected", nil, ""); got != "已取消该操作。" {
		t.Fatalf("unexpected fallback rejected copy: %q", got)
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

func containsSubstring(text, substring string) bool {
	return strings.Contains(text, substring)
}

func containsField(fields []AgentToolField, key string) bool {
	for _, field := range fields {
		if field.Key == key {
			return true
		}
	}
	return false
}
