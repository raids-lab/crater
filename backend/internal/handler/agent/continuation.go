package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/raids-lab/crater/dao/model"
)

func (mgr *AgentMgr) buildAgentContinuation(ctx context.Context, sessionID string) map[string]any {
	if mgr == nil || mgr.agentService == nil || sessionID == "" {
		return nil
	}

	turns, err := mgr.agentService.ListTurns(ctx, sessionID)
	if err != nil || len(turns) == 0 {
		return nil
	}

	latestTurn := turns[0]
	continuation := map[string]any{
		"source_turn_id":     latestTurn.TurnID,
		"source_turn_status": latestTurn.Status,
	}
	meaningful := false

	if clarification := mgr.loadClarificationContinuation(ctx, latestTurn); len(clarification) > 0 {
		continuation["clarification"] = clarification
		meaningful = true
	}
	if pendingConfirmations := mgr.loadPendingConfirmationContinuation(ctx, latestTurn); len(pendingConfirmations) > 0 {
		continuation["pending_confirmations"] = pendingConfirmations
		meaningful = true
	}

	if !meaningful {
		return nil
	}
	return continuation
}

func (mgr *AgentMgr) buildResumeContinuation(
	ctx context.Context,
	turn *model.AgentTurn,
	toolCall *model.AgentToolCall,
) map[string]any {
	if mgr == nil || mgr.agentService == nil || turn == nil || toolCall == nil {
		return nil
	}

	continuation := map[string]any{
		"source_turn_id":     turn.TurnID,
		"source_turn_status": turn.Status,
	}

	resume := mgr.buildToolCallResumeResult(turn, toolCall)
	if confirmationResults := mgr.loadConfirmationResultContinuations(ctx, turn); len(confirmationResults) > 0 {
		resume["confirmation_results"] = confirmationResults
	}
	if workflow := mgr.loadWorkflowState(ctx, turn); len(workflow) > 0 {
		continuation["workflow"] = workflow
		resume["workflow"] = workflow
	}
	if sourceTurnContext := mgr.loadSourceTurnContext(ctx, turn); len(sourceTurnContext) > 0 {
		continuation["source_turn_context"] = sourceTurnContext
		resume["source_turn_context"] = sourceTurnContext
		if originalUserMessage := historyStringValue(sourceTurnContext["original_user_message"]); strings.TrimSpace(originalUserMessage) != "" {
			continuation["original_user_message"] = originalUserMessage
			resume["original_user_message"] = originalUserMessage
		}
	}
	continuation["resume_after_confirmation"] = resume
	return continuation
}

func (mgr *AgentMgr) buildToolCallResumeResult(turn *model.AgentTurn, toolCall *model.AgentToolCall) map[string]any {
	if mgr == nil || turn == nil || toolCall == nil {
		return nil
	}
	result := map[string]any{
		"confirm_id":     fmt.Sprintf("%d", toolCall.ID),
		"tool_name":      toolCall.ToolName,
		"action_title":   mgr.buildConfirmationDescription(toolCall.ToolName, json.RawMessage(toolCall.ToolArgs)),
		"action_intent":  agentActionIntentFromToolName(toolCall.ToolName),
		"tool_args":      parseToolArgsMap(json.RawMessage(toolCall.ToolArgs)),
		"result_status":  toolCall.ResultStatus,
		"confirmed":      toolCall.UserConfirmed != nil && *toolCall.UserConfirmed,
		"source_turn_id": turn.TurnID,
	}
	if parsedResult, _ := parseToolCallResult(toolCall); parsedResult != nil {
		result["result"] = parsedResult
	}
	return result
}

func (mgr *AgentMgr) loadConfirmationResultContinuations(ctx context.Context, turn *model.AgentTurn) []map[string]any {
	if mgr == nil || mgr.agentService == nil || turn == nil || turn.TurnID == "" {
		return nil
	}
	toolCalls, err := mgr.agentService.ListToolCallsByTurn(ctx, turn.TurnID)
	if err != nil {
		return nil
	}
	results := make([]map[string]any, 0)
	orderedToolCalls := mgr.orderToolCallsForWorkflow(toolCalls, mgr.loadWorkflowState(ctx, turn))
	for _, toolCall := range orderedToolCalls {
		if toolCall == nil || toolCall.ResultStatus == agentToolStatusAwaitConfirm {
			continue
		}
		if toolCall.UserConfirmed == nil {
			continue
		}
		if resumeResult := mgr.buildToolCallResumeResult(turn, toolCall); len(resumeResult) > 0 {
			results = append(results, resumeResult)
		}
	}
	return results
}

func (mgr *AgentMgr) loadClarificationContinuation(ctx context.Context, turn *model.AgentTurn) map[string]any {
	if mgr == nil || mgr.agentService == nil || turn == nil || turn.TurnID == "" {
		return nil
	}

	events, err := mgr.agentService.ListRunEvents(ctx, turn.TurnID)
	if err != nil {
		return nil
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event == nil || event.EventType != "final_answer" || len(event.Metadata) == 0 {
			continue
		}
		metadata := parseAgentJSONMap(json.RawMessage(event.Metadata))
		if continuation, ok := metadata["continuation"].(map[string]any); ok && len(continuation) > 0 {
			return continuation
		}
	}
	return nil
}

func (mgr *AgentMgr) loadPendingConfirmationContinuation(ctx context.Context, turn *model.AgentTurn) []map[string]any {
	if mgr == nil || mgr.agentService == nil || turn == nil || turn.Status != "awaiting_confirmation" || turn.TurnID == "" {
		return nil
	}

	toolCalls, err := mgr.agentService.ListToolCallsByTurn(ctx, turn.TurnID)
	if err != nil {
		return nil
	}
	results := make([]map[string]any, 0)
	workflow := mgr.loadWorkflowState(ctx, turn)
	for _, toolCall := range mgr.orderToolCallsForWorkflow(toolCalls, workflow) {
		if toolCall == nil || toolCall.ResultStatus != agentToolStatusAwaitConfirm {
			continue
		}
		result := map[string]any{
			"confirm_id":     fmt.Sprintf("%d", toolCall.ID),
			"tool_name":      toolCall.ToolName,
			"action_intent":  agentActionIntentFromToolName(toolCall.ToolName),
			"tool_args":      parseToolArgsMap(json.RawMessage(toolCall.ToolArgs)),
			"result_status":  toolCall.ResultStatus,
			"agent_role":     toolCall.AgentRole,
			"source_turn_id": turn.TurnID,
		}
		if len(workflow) > 0 {
			result["workflow"] = workflow
		}
		results = append(results, result)
	}
	return results
}

func (mgr *AgentMgr) hasOtherPendingConfirmations(
	ctx context.Context,
	turnID string,
	excludeToolCallID uint,
) bool {
	if mgr == nil || mgr.agentService == nil || turnID == "" {
		return false
	}
	toolCalls, err := mgr.agentService.ListToolCallsByTurn(ctx, turnID)
	if err != nil {
		return false
	}
	for _, toolCall := range toolCalls {
		if toolCall == nil || toolCall.ID == excludeToolCallID {
			continue
		}
		if toolCall.ResultStatus == agentToolStatusAwaitConfirm {
			return true
		}
	}
	return false
}

func (mgr *AgentMgr) orderToolCallsForWorkflow(
	toolCalls []*model.AgentToolCall,
	workflow map[string]any,
) []*model.AgentToolCall {
	if len(toolCalls) <= 1 {
		return toolCalls
	}
	ordered := append([]*model.AgentToolCall(nil), toolCalls...)
	rankByConfirmID := map[string]int{}
	rankByToolCallID := map[string]int{}
	rankBySignature := map[string]int{}

	addRank := func(index int, item any) {
		switch typed := item.(type) {
		case string:
			if typed != "" {
				rankByConfirmID[typed] = index
			}
		case map[string]any:
			confirmID := historyStringValue(typed["confirm_id"])
			if confirmID != "" {
				rankByConfirmID[confirmID] = index
			}
			toolCallID := historyStringValue(typed["tool_call_id"])
			if toolCallID == "" {
				toolCallID = historyStringValue(typed["toolCallId"])
			}
			if toolCallID != "" {
				rankByToolCallID[toolCallID] = index
			}
			toolName := historyStringValue(typed["tool_name"])
			if toolName == "" {
				toolName = historyStringValue(typed["toolName"])
			}
			args := typed["tool_args"]
			if args == nil {
				args = typed["toolArgs"]
			}
			if sig := toolCallSignatureFromValues(toolName, args); sig != "" {
				rankBySignature[sig] = index
			}
		}
	}

	if values, ok := workflow["pending_confirmations"].([]any); ok {
		for index, item := range values {
			if itemMap, ok := item.(map[string]any); ok {
				if confirmation, ok := itemMap["confirmation"].(map[string]any); ok {
					addRank(index, confirmation)
					toolName := historyStringValue(confirmation["tool_name"])
					if toolName == "" {
						toolName = historyStringValue(confirmation["toolName"])
					}
					args := itemMap["tool_args"]
					if args == nil {
						args = itemMap["toolArgs"]
					}
					if sig := toolCallSignatureFromValues(toolName, args); sig != "" {
						rankBySignature[sig] = index
					}
					continue
				}
			}
			addRank(index, item)
		}
	}
	if values, ok := workflow["pending_confirmation_ids"].([]any); ok {
		for index, item := range values {
			addRank(index, item)
		}
	}
	if values, ok := workflow["actions"].([]any); ok {
		for index, item := range values {
			addRank(index, item)
		}
	}

	rankFor := func(toolCall *model.AgentToolCall) int {
		if toolCall == nil {
			return 1 << 30
		}
		if rank, ok := rankByConfirmID[fmt.Sprintf("%d", toolCall.ID)]; ok {
			return rank
		}
		if rank, ok := rankByToolCallID[toolCall.ToolCallID]; ok {
			return rank
		}
		if sig := toolCallSignatureFromValues(toolCall.ToolName, parseToolArgsMap(json.RawMessage(toolCall.ToolArgs))); sig != "" {
			if rank, ok := rankBySignature[sig]; ok {
				return rank
			}
		}
		return 1 << 20
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		leftRank := rankFor(ordered[i])
		rightRank := rankFor(ordered[j])
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if !ordered[i].CreatedAt.Equal(ordered[j].CreatedAt) {
			return ordered[i].CreatedAt.Before(ordered[j].CreatedAt)
		}
		return ordered[i].ID < ordered[j].ID
	})
	return ordered
}

func toolCallSignatureFromValues(toolName string, args any) string {
	toolName = historyStringValue(toolName)
	if toolName == "" {
		return ""
	}
	normalizedArgs := "{}"
	if args != nil {
		if marshaled, err := json.Marshal(args); err == nil {
			normalizedArgs = string(marshaled)
		}
	}
	return toolName + ":" + normalizedArgs
}

func (mgr *AgentMgr) loadSourceTurnContext(ctx context.Context, turn *model.AgentTurn) map[string]any {
	if mgr == nil || mgr.agentService == nil || turn == nil || turn.TurnID == "" {
		return nil
	}
	originalUserMessage := mgr.loadTurnOriginalUserMessage(ctx, turn)
	toolCallSnapshots := mgr.loadSourceTurnToolCalls(ctx, turn)
	events, err := mgr.agentService.ListRunEvents(ctx, turn.TurnID)
	if err != nil || len(events) == 0 {
		if originalUserMessage == "" && len(toolCallSnapshots) == 0 {
			return nil
		}
		result := map[string]any{
			"turn_id":               turn.TurnID,
			"source_turn_status":    turn.Status,
			"original_user_message": originalUserMessage,
		}
		if len(toolCallSnapshots) > 0 {
			result["tool_calls"] = toolCallSnapshots
		}
		return result
	}
	const maxEvents = 80
	start := 0
	if len(events) > maxEvents {
		start = len(events) - maxEvents
	}
	snapshots := make([]map[string]any, 0, len(events)-start)
	for _, event := range events[start:] {
		if event == nil {
			continue
		}
		item := map[string]any{
			"sequence":   event.Sequence,
			"event_type": event.EventType,
			"status":     event.EventStatus,
			"agent_id":   event.AgentID,
			"agent_role": event.AgentRole,
			"title":      event.Title,
			"content":    truncateAgentHistoryContent(event.Content, 1200),
			"created_at": event.CreatedAt,
		}
		metadata := parseAgentJSONMap(json.RawMessage(event.Metadata))
		if len(metadata) > 0 {
			item["metadata"] = metadata
		}
		snapshots = append(snapshots, item)
	}
	if len(snapshots) == 0 {
		if originalUserMessage == "" && len(toolCallSnapshots) == 0 {
			return nil
		}
		result := map[string]any{
			"turn_id":               turn.TurnID,
			"source_turn_status":    turn.Status,
			"original_user_message": originalUserMessage,
		}
		if len(toolCallSnapshots) > 0 {
			result["tool_calls"] = toolCallSnapshots
		}
		return result
	}
	result := map[string]any{
		"turn_id":               turn.TurnID,
		"source_turn_status":    turn.Status,
		"original_user_message": originalUserMessage,
		"events":                snapshots,
	}
	if len(toolCallSnapshots) > 0 {
		result["tool_calls"] = toolCallSnapshots
	}
	return result
}

func (mgr *AgentMgr) loadSourceTurnToolCalls(ctx context.Context, turn *model.AgentTurn) []map[string]any {
	if mgr == nil || mgr.agentService == nil || turn == nil || turn.TurnID == "" {
		return nil
	}
	toolCalls, err := mgr.agentService.ListToolCallsByTurn(ctx, turn.TurnID)
	if err != nil || len(toolCalls) == 0 {
		return nil
	}
	workflow := mgr.loadWorkflowState(ctx, turn)
	orderedToolCalls := mgr.orderToolCallsForWorkflow(toolCalls, workflow)
	results := make([]map[string]any, 0, len(orderedToolCalls))
	for _, toolCall := range orderedToolCalls {
		if toolCall == nil {
			continue
		}
		item := map[string]any{
			"id":            fmt.Sprintf("%d", toolCall.ID),
			"tool_call_id":  toolCall.ToolCallID,
			"agent_id":      toolCall.AgentID,
			"agent_role":    toolCall.AgentRole,
			"tool_name":     toolCall.ToolName,
			"tool_args":     parseToolArgsMap(json.RawMessage(toolCall.ToolArgs)),
			"result_status": toolCall.ResultStatus,
			"confirmed":     toolCall.UserConfirmed != nil && *toolCall.UserConfirmed,
			"created_at":    toolCall.CreatedAt,
		}
		if parsedResult, _ := parseToolCallResult(toolCall); parsedResult != nil {
			item["result"] = parsedResult
		}
		results = append(results, item)
	}
	return results
}

func (mgr *AgentMgr) loadTurnOriginalUserMessage(ctx context.Context, turn *model.AgentTurn) string {
	if mgr == nil || mgr.agentService == nil || turn == nil || turn.SessionID == "" {
		return ""
	}
	messages, err := mgr.agentService.ListMessages(ctx, turn.SessionID)
	if err != nil {
		return ""
	}
	if turn.RequestID != "" {
		for _, msg := range messages {
			if msg == nil || msg.Role != "user" {
				continue
			}
			if agentMessageRequestID(msg) == turn.RequestID {
				return strings.TrimSpace(msg.Content)
			}
		}
	}
	var latest string
	for _, msg := range messages {
		if msg == nil || msg.Role != "user" {
			continue
		}
		if !msg.CreatedAt.After(turn.StartedAt) || msg.CreatedAt.Equal(turn.StartedAt) {
			latest = strings.TrimSpace(msg.Content)
		}
	}
	return latest
}

func (mgr *AgentMgr) loadWorkflowState(ctx context.Context, turn *model.AgentTurn) map[string]any {
	if mgr == nil || mgr.agentService == nil || turn == nil || turn.TurnID == "" {
		return nil
	}

	events, err := mgr.agentService.ListRunEvents(ctx, turn.TurnID)
	if err != nil {
		return nil
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event == nil || len(event.Metadata) == 0 {
			continue
		}
		metadata := parseAgentJSONMap(json.RawMessage(event.Metadata))
		if workflow, ok := metadata["workflow"].(map[string]any); ok && len(workflow) > 0 {
			return workflow
		}
		if continuation, ok := metadata["continuation"].(map[string]any); ok {
			if workflow, ok := continuation["workflow"].(map[string]any); ok && len(workflow) > 0 {
				return workflow
			}
		}
	}
	return nil
}

func parseAgentJSONMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	result := map[string]any{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result
}

func agentActionIntentFromToolName(toolName string) string {
	switch toolName {
	case agentToolResubmitJob:
		return "resubmit"
	case agentToolStopJob:
		return "stop"
	case agentToolDeleteJob:
		return "delete"
	case agentToolCreateJupyter:
		return "create_jupyter_job"
	case agentToolCreateWebIDE:
		return "create_webide_job"
	case agentToolCreateTrain:
		return "create_training_job"
	case agentToolCreateCustom:
		return "create_custom_job"
	case agentToolCreatePytorch:
		return "create_pytorch_job"
	case agentToolCreateTensorflow:
		return "create_tensorflow_job"
	case agentToolCreateImage:
		return "create_image_build"
	case agentToolManageBuild:
		return "manage_image_build"
	case agentToolRegisterImage:
		return "register_external_image"
	case agentToolManageAccess:
		return "manage_image_access"
	case agentToolCordonNode:
		return "cordon_node"
	case agentToolUncordonNode:
		return "uncordon_node"
	case agentToolDrainNode:
		return "drain_node"
	case agentToolDeletePod:
		return "delete_pod"
	case agentToolRestartWL:
		return "restart_workload"
	case agentToolK8sScaleWL:
		return "k8s_scale_workload"
	case agentToolK8sLabelNode:
		return "k8s_label_node"
	case agentToolK8sTaintNode:
		return "k8s_taint_node"
	case agentToolRunKubectl:
		return "run_kubectl"
	case agentToolAdminCommand:
		return "execute_admin_command"
	default:
		return ""
	}
}
