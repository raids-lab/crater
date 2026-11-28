package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestAgentToolResultReplayAndIsolation(t *testing.T) {
	s := NewAgentConversationService(newAgentConversationTestDB(t))
	ctx := context.Background()
	id := uuid.NewString()
	run, err := s.PrepareRun(ctx, 41, AgentIdentityUser, id, nil, AgentRunModeAppend,
		agentTestUserMessagePointer("user", "Open the job"))
	mustAgentTestNoError(t, err)
	parts := json.RawMessage(`[{"type":"tool-navigate_to_job_detail","toolCallId":"call","state":"input-available","input":{"jobName":"job-a"}}]`)
	mustAgentTestNoError(t, s.CompleteRun(ctx, 41, AgentIdentityUser, run.RunID, AgentRunCompletion{
		Status:  AgentRunStatusCompleted,
		Message: &AgentUIMessage{ID: "assistant", Role: "assistant", Parts: parts},
	}))
	output := &AgentToolOutput{MessageID: "assistant", ToolCallID: "call", State: "output-available", Output: json.RawMessage(`{"status":"rejected"}`)}
	mustAgentTestError(t, s.SaveToolOutput(ctx, 42, AgentIdentityUser, id, output))
	mustAgentTestError(t, s.SaveToolOutput(ctx, 41, AgentIdentityAdmin, id, output))
	mustAgentTestNoError(t, s.SaveToolOutput(ctx, 41, AgentIdentityUser, id, output))
	mustAgentTestNoError(t, s.SaveToolOutput(ctx, 41, AgentIdentityUser, id, output))
	output.Output = json.RawMessage(`{"status":"accepted"}`)
	mustAgentTestError(t, s.SaveToolOutput(ctx, 41, AgentIdentityUser, id, output))
	parts = json.RawMessage(`[{"type":"tool-list_jobs","toolCallId":"call","state":"input-available"}]`)
	_, err = mergeAgentToolOutput(parts, output)
	mustAgentTestError(t, err)
}
