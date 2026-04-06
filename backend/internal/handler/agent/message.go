package agent

import (
	"encoding/json"
	"strings"

	"github.com/raids-lab/crater/dao/model"
)

const (
	agentHistoryMessageLimit      = 24
	agentHistoryContentLimit      = 1600
	agentHistoryToolContentLimit  = 480
	agentHistoryAssistantMaxChars = 1200
)

func truncateAgentHistoryContent(content string, maxChars int) string {
	content = strings.TrimSpace(content)
	if content == "" || maxChars <= 0 {
		return content
	}
	runes := []rune(content)
	if len(runes) <= maxChars {
		return content
	}
	return string(runes[:maxChars]) + "..."
}

func agentMessageRequestID(msg *model.AgentMessage) string {
	if msg == nil || len(msg.Metadata) == 0 {
		return ""
	}
	var metadata map[string]any
	if err := json.Unmarshal(msg.Metadata, &metadata); err != nil {
		return ""
	}
	requestID, _ := metadata["requestId"].(string)
	return requestID
}

func historyContainsRequestID(messages []*model.AgentMessage, requestID string) bool {
	if requestID == "" {
		return false
	}
	for _, msg := range messages {
		if agentMessageRequestID(msg) == requestID {
			return true
		}
	}
	return false
}

func filterHistoryMessagesForRequest(messages []*model.AgentMessage, requestID string) []*model.AgentMessage {
	if requestID == "" {
		return messages
	}
	filtered := make([]*model.AgentMessage, 0, len(messages))
	for _, msg := range messages {
		if agentMessageRequestID(msg) == requestID {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func buildAgentHistory(messages []*model.AgentMessage) []map[string]any {
	if len(messages) == 0 {
		return nil
	}
	start := len(messages) - agentHistoryMessageLimit
	if start < 0 {
		start = 0
	}
	history := make([]map[string]any, 0, len(messages)-start)
	for _, msg := range messages[start:] {
		if msg == nil || msg.Content == "" {
			continue
		}
		maxChars := agentHistoryContentLimit
		switch msg.Role {
		case "tool":
			maxChars = agentHistoryToolContentLimit
		case "assistant":
			maxChars = agentHistoryAssistantMaxChars
		}
		history = append(history, map[string]any{
			"role":    msg.Role,
			"content": truncateAgentHistoryContent(msg.Content, maxChars),
		})
	}
	return history
}
