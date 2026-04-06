package agent

import (
	"context"
	"encoding/json"
	"fmt"

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
	if pendingConfirmation := mgr.loadPendingConfirmationContinuation(ctx, latestTurn); len(pendingConfirmation) > 0 {
		continuation["pending_confirmation"] = pendingConfirmation
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

	resume := map[string]any{
		"confirm_id":     fmt.Sprintf("%d", toolCall.ID),
		"tool_name":      toolCall.ToolName,
		"action_intent":  agentActionIntentFromToolName(toolCall.ToolName),
		"tool_args":      parseToolArgsMap(json.RawMessage(toolCall.ToolArgs)),
		"result_status":  toolCall.ResultStatus,
		"confirmed":      toolCall.UserConfirmed != nil && *toolCall.UserConfirmed,
		"source_turn_id": turn.TurnID,
	}
	if result, _ := parseToolCallResult(toolCall); result != nil {
		resume["result"] = result
	}
	if workflow := mgr.loadWorkflowState(ctx, turn); len(workflow) > 0 {
		continuation["workflow"] = workflow
		resume["workflow"] = workflow
	}
	continuation["resume_after_confirmation"] = resume
	return continuation
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

func (mgr *AgentMgr) loadPendingConfirmationContinuation(ctx context.Context, turn *model.AgentTurn) map[string]any {
	if mgr == nil || mgr.agentService == nil || turn == nil || turn.Status != "awaiting_confirmation" || turn.TurnID == "" {
		return nil
	}

	toolCalls, err := mgr.agentService.ListToolCallsByTurn(ctx, turn.TurnID)
	if err != nil {
		return nil
	}
	for i := len(toolCalls) - 1; i >= 0; i-- {
		toolCall := toolCalls[i]
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
		if workflow := mgr.loadWorkflowState(ctx, turn); len(workflow) > 0 {
			result["workflow"] = workflow
		}
		return result
	}
	return nil
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
	case agentToolCreateTrain:
		return "create_training_job"
	default:
		return ""
	}
}
