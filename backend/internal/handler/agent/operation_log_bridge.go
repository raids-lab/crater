package agent

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/pkg/constants"
)

func recordAgentMutationOperationLog(
	c *gin.Context,
	toolName string,
	toolArgs json.RawMessage,
	result any,
	execErr error,
	executionBackend string,
	confirmID string,
) {
	opType, target, details, ok := buildAgentOperationLogPayload(
		toolName,
		toolArgs,
		result,
		executionBackend,
		confirmID,
	)
	if !ok {
		return
	}

	status := constants.OpStatusSuccess
	message := ""
	if execErr != nil {
		status = constants.OpStatusFailed
		message = execErr.Error()
	}
	handler.RecordOperationLog(c, opType, target, status, message, details)
}

func buildAgentOperationLogPayload(
	toolName string,
	toolArgs json.RawMessage,
	result any,
	executionBackend string,
	confirmID string,
) (string, string, map[string]any, bool) {
	args := parseToolArgsMap(toolArgs)
	resultMap, _ := result.(map[string]any)

	details := map[string]any{
		"source":            "agent_confirmation",
		"tool_name":         toolName,
		"execution_backend": strings.TrimSpace(executionBackend),
	}
	if confirmID = strings.TrimSpace(confirmID); confirmID != "" {
		details["confirm_id"] = confirmID
	}
	if reason := getToolArgString(args, "reason", ""); reason != "" {
		details["reason"] = reason
	}
	if resultStatus, _ := resultMap["status"].(string); strings.TrimSpace(resultStatus) != "" {
		details["result_status"] = strings.TrimSpace(resultStatus)
	}
	if resultMessage, _ := resultMap["message"].(string); strings.TrimSpace(resultMessage) != "" {
		details["result_message"] = strings.TrimSpace(resultMessage)
	}

	switch toolName {
	case agentToolCordonNode:
		target := getToolArgString(args, "node_name", "")
		if target == "" {
			target, _ = resultMap["node_name"].(string)
		}
		return constants.OpTypeSetUnschedulable, target, details, target != ""
	case agentToolUncordonNode:
		target := getToolArgString(args, "node_name", "")
		if target == "" {
			target, _ = resultMap["node_name"].(string)
		}
		return constants.OpTypeCancelUnschedulable, target, details, target != ""
	case agentToolDrainNode:
		target := getToolArgString(args, "node_name", "")
		if target == "" {
			target, _ = resultMap["node_name"].(string)
		}
		return constants.OpTypeDrainNode, target, details, target != ""
	default:
		return "", "", nil, false
	}
}
