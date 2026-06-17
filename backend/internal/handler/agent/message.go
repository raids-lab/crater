package agent

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/raids-lab/crater/dao/model"
)

const (
	agentHistoryMessageLimit      = 24
	agentHistoryContentLimit      = 1600
	agentHistoryToolContentLimit  = 480
	agentHistoryAssistantMaxChars = 1200
)

type agentHistoryEntry struct {
	createdAt time.Time
	payload   map[string]any
}

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

func historyStringValue(value any) string {
	if value == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", value))
	if s == "<nil>" {
		return ""
	}
	return s
}

func compactHistoryJSON(raw json.RawMessage, maxChars int) string {
	if len(raw) == 0 {
		return ""
	}

	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return truncateAgentHistoryContent(string(raw), maxChars)
	}

	normalized, err := json.Marshal(payload)
	if err != nil {
		return truncateAgentHistoryContent(string(raw), maxChars)
	}
	return truncateAgentHistoryContent(string(normalized), maxChars)
}

func decorateAssistantHistoryContent(msg *model.AgentMessage, content string) string {
	if msg == nil || msg.Role != "assistant" || len(msg.Metadata) == 0 {
		return content
	}

	metadata := parseAgentJSONMap(json.RawMessage(msg.Metadata))
	if len(metadata) == 0 {
		return content
	}

	source := historyStringValue(metadata["source"])
	if source != "tool_confirmation" && source != "confirmation_resume" {
		return content
	}

	toolName := historyStringValue(metadata["toolName"])
	status := historyStringValue(metadata["status"])
	prefixParts := make([]string, 0, 2)
	if toolName != "" {
		prefixParts = append(prefixParts, "tool="+toolName)
	}
	if status != "" {
		prefixParts = append(prefixParts, "status="+status)
	}
	if len(prefixParts) == 0 {
		return "【上轮工具结果】" + content
	}
	return "【上轮工具结果 " + strings.Join(prefixParts, " ") + "】" + content
}

func buildAgentHistoryMessageEntry(msg *model.AgentMessage) *agentHistoryEntry {
	if msg == nil || strings.TrimSpace(msg.Content) == "" {
		return nil
	}

	maxChars := agentHistoryContentLimit
	switch msg.Role {
	case "tool":
		maxChars = agentHistoryToolContentLimit
	case "assistant":
		maxChars = agentHistoryAssistantMaxChars
	}

	content := truncateAgentHistoryContent(msg.Content, maxChars)
	if msg.Role == "assistant" {
		content = decorateAssistantHistoryContent(msg, content)
		content = truncateAgentHistoryContent(content, maxChars)
	}

	payload := map[string]any{
		"role":    msg.Role,
		"content": content,
	}
	if msg.Role == "tool" && strings.TrimSpace(msg.ToolCallID) != "" {
		payload["tool_call_id"] = msg.ToolCallID
	}

	return &agentHistoryEntry{
		createdAt: msg.CreatedAt,
		payload:   payload,
	}
}

func buildAgentToolCallHistoryContent(toolCall *model.AgentToolCall) string {
	if toolCall == nil || strings.TrimSpace(toolCall.ToolName) == "" {
		return ""
	}

	status := strings.TrimSpace(toolCall.ResultStatus)
	parts := []string{
		fmt.Sprintf("tool=%s", toolCall.ToolName),
	}
	if status != "" {
		parts = append(parts, "status="+status)
	}
	if toolCall.UserConfirmed != nil {
		parts = append(parts, fmt.Sprintf("user_confirmed=%t", *toolCall.UserConfirmed))
	}

	if args := compactHistoryJSON(json.RawMessage(toolCall.ToolArgs), 180); args != "" {
		parts = append(parts, "args="+args)
	}

	switch status {
	case "rejected":
		parts = append(parts, "result=operation rejected by user")
	case agentToolStatusAwaitConfirm, agentToolStatusConfirmationRequired:
		parts = append(parts, "result=awaiting user confirmation")
	default:
		if result := compactHistoryJSON(json.RawMessage(toolCall.ToolResult), 220); result != "" {
			parts = append(parts, "result="+result)
		}
	}

	return strings.Join(parts, " ; ")
}

func buildAgentHistoryToolEntry(toolCall *model.AgentToolCall) *agentHistoryEntry {
	if toolCall == nil {
		return nil
	}

	content := buildAgentToolCallHistoryContent(toolCall)
	if strings.TrimSpace(content) == "" {
		return nil
	}

	toolCallID := strings.TrimSpace(toolCall.ToolCallID)
	if toolCallID == "" {
		toolCallID = fmt.Sprintf("tool-call-%d", toolCall.ID)
	}

	return &agentHistoryEntry{
		createdAt: toolCall.CreatedAt,
		payload: map[string]any{
			"role":         "tool",
			"content":      truncateAgentHistoryContent(content, agentHistoryToolContentLimit),
			"tool_call_id": toolCallID,
		},
	}
}

func buildAgentHistory(messages []*model.AgentMessage, toolCalls []*model.AgentToolCall) []map[string]any {
	if len(messages) == 0 && len(toolCalls) == 0 {
		return nil
	}

	entries := make([]agentHistoryEntry, 0, len(messages)+len(toolCalls))
	for _, msg := range messages {
		if entry := buildAgentHistoryMessageEntry(msg); entry != nil {
			entries = append(entries, *entry)
		}
	}
	for _, toolCall := range toolCalls {
		if entry := buildAgentHistoryToolEntry(toolCall); entry != nil {
			entries = append(entries, *entry)
		}
	}
	if len(entries) == 0 {
		return nil
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].createdAt.Before(entries[j].createdAt)
	})

	start := len(entries) - agentHistoryMessageLimit
	if start < 0 {
		start = 0
	}
	history := make([]map[string]any, 0, len(entries)-start)
	for _, entry := range entries[start:] {
		if len(entry.payload) == 0 {
			continue
		}
		history = append(history, entry.payload)
	}
	return history
}
