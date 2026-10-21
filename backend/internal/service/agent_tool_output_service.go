package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
)

const agentMaximumToolOutputBytes = 32 << 10

type AgentToolOutput struct {
	MessageID  string          `json:"messageId"`
	ToolCallID string          `json:"toolCallId"`
	State      string          `json:"state"`
	Output     json.RawMessage `json:"output,omitempty"`
	ErrorText  string          `json:"errorText,omitempty"`
}

func (s *AgentConversationService) SaveToolOutput(
	ctx context.Context,
	userID uint,
	identity AgentIdentity,
	conversationID string,
	toolOutput *AgentToolOutput,
) error {
	if err := validateConversationID(conversationID); err != nil {
		return err
	}
	if err := validateAgentToolOutput(toolOutput); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return saveAgentToolOutputTransaction(ctx, tx, userID, identity, conversationID, toolOutput)
	})
}

func validateAgentToolOutput(toolOutput *AgentToolOutput) error {
	if toolOutput == nil {
		return bizerr.BadRequest.MissingParameter.New("Agent tool output is required")
	}
	toolOutput.MessageID = strings.TrimSpace(toolOutput.MessageID)
	toolOutput.ToolCallID = strings.TrimSpace(toolOutput.ToolCallID)
	toolOutput.ErrorText = strings.TrimSpace(toolOutput.ErrorText)
	if toolOutput.MessageID == "" || len(toolOutput.MessageID) > agentMaximumMessageIDLength ||
		toolOutput.ToolCallID == "" || len(toolOutput.ToolCallID) > agentMaximumMessageIDLength {
		return bizerr.BadRequest.ParameterError.New("Agent tool output identifier is invalid")
	}
	if toolOutput.State != "output-available" && toolOutput.State != "output-error" {
		return bizerr.BadRequest.ParameterError.New("Agent tool output state is invalid")
	}
	if len(toolOutput.Output) > agentMaximumToolOutputBytes || len(toolOutput.ErrorText) > agentMaximumErrorLength ||
		(len(toolOutput.Output) > 0 && !json.Valid(toolOutput.Output)) {
		return bizerr.BadRequest.ParameterError.New("Agent tool output is invalid")
	}
	return nil
}

func saveAgentToolOutputTransaction(
	ctx context.Context,
	tx *gorm.DB,
	userID uint,
	identity AgentIdentity,
	conversationID string,
	toolOutput *AgentToolOutput,
) error {
	if err := ensureOwnedAgentConversation(ctx, tx, userID, identity, conversationID); err != nil {
		return err
	}

	var message model.AgentMessage
	if err := tx.WithContext(ctx).Where(
		"id = ? AND conversation_id = ? AND role = ? AND superseded_at IS NULL",
		toolOutput.MessageID, conversationID, "assistant",
	).First(&message).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return bizerr.NotFound.DataBaseNotFound.New("Agent message not found")
		}
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to load Agent message")
	}
	updatedParts, err := mergeAgentToolOutput(message.Parts, toolOutput)
	if err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Model(&message).Update("parts", datatypes.JSON(updatedParts)).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to save Agent tool output")
	}
	return nil
}

func ensureOwnedAgentConversation(
	ctx context.Context,
	tx *gorm.DB,
	userID uint,
	identity AgentIdentity,
	conversationID string,
) error {
	var conversationCount int64
	if err := tx.WithContext(ctx).Model(&model.AgentConversation{}).
		Where("id = ? AND user_id = ? AND identity_scope = ?", conversationID, userID, identity).
		Count(&conversationCount).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to validate Agent conversation")
	}
	if conversationCount == 0 {
		return bizerr.NotFound.DataBaseNotFound.New("Agent conversation not found")
	}
	return nil
}

func mergeAgentToolOutput(partsJSON []byte, toolOutput *AgentToolOutput) ([]byte, error) {
	var parts []map[string]json.RawMessage
	if err := json.Unmarshal(partsJSON, &parts); err != nil {
		return nil, bizerr.Internal.ServiceError.Wrap(err, "failed to decode Agent message parts")
	}
	found := false
	for index := range parts {
		var toolCallID string
		if err := json.Unmarshal(parts[index]["toolCallId"], &toolCallID); err != nil ||
			toolCallID != toolOutput.ToolCallID {
			continue
		}
		var partType string
		if err := json.Unmarshal(parts[index]["type"], &partType); err != nil ||
			(partType != "tool-navigate_to_page" &&
				partType != "tool-navigate_to_job_detail" &&
				partType != "tool-propose_form_patch") {
			return nil, bizerr.BadRequest.ParameterError.New("Agent tool call is invalid")
		}
		var currentState string
		if err := json.Unmarshal(parts[index]["state"], &currentState); err != nil ||
			currentState != "input-available" {
			return nil, bizerr.Conflict.ResourceStatusError.New("Agent tool call is already resolved")
		}
		parts[index]["state"] = mustMarshalJSON(toolOutput.State)
		if toolOutput.State == "output-available" {
			parts[index]["output"] = toolOutput.Output
			delete(parts[index], "errorText")
		} else {
			parts[index]["errorText"] = mustMarshalJSON(toolOutput.ErrorText)
			delete(parts[index], "output")
		}
		found = true
		break
	}
	if !found {
		return nil, bizerr.NotFound.DataBaseNotFound.New("Agent tool call not found")
	}
	updated, err := json.Marshal(parts)
	if err != nil {
		return nil, bizerr.Internal.ServiceError.Wrap(err, "failed to encode Agent message parts")
	}
	return updated, nil
}

func mustMarshalJSON(value string) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}
