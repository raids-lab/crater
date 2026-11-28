package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
	if toolOutput.State != agentToolStateAvailable && toolOutput.State != "output-error" {
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
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where(
		"id = ? AND conversation_id = ? AND role = ? AND superseded_at IS NULL",
		toolOutput.MessageID, conversationID, agentRoleAssistant,
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
	var conversation model.AgentConversation
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND user_id = ? AND identity_scope = ?", conversationID, userID, identity).
		First(&conversation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return bizerr.NotFound.DataBaseNotFound.New("Agent conversation not found")
		}
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to validate Agent conversation")
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
		if !isAgentConfirmationPart(parts[index]["type"]) {
			return nil, bizerr.BadRequest.ParameterError.New("Agent tool call is invalid")
		}
		var currentState string
		if err := json.Unmarshal(parts[index]["state"], &currentState); err != nil ||
			currentState != "input-available" {
			if currentState == toolOutput.State && sameAgentToolOutput(parts[index], toolOutput) {
				return partsJSON, nil
			}
			return nil, bizerr.Conflict.ResourceStatusError.New("Agent tool call is already resolved")
		}
		parts[index]["state"] = mustMarshalJSON(toolOutput.State)
		if toolOutput.State == agentToolStateAvailable {
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
	if len(updated) > agentMaximumMessagePartsBytes {
		return nil, bizerr.BadRequest.ParameterError.New("Agent message is too large")
	}
	return updated, nil
}

func isAgentConfirmationPart(raw json.RawMessage) bool {
	var partType string
	if json.Unmarshal(raw, &partType) != nil {
		return false
	}
	switch partType {
	case "tool-navigate_to_page", "tool-navigate_to_job_detail", "tool-propose_form_patch":
		return true
	default:
		return false
	}
}

// A lost HTTP response may cause the browser to submit the same result again.
func sameAgentToolOutput(part map[string]json.RawMessage, output *AgentToolOutput) bool {
	if output.State == "output-error" {
		var text string
		return json.Unmarshal(part["errorText"], &text) == nil && text == output.ErrorText
	}
	var stored, incoming any
	return json.Unmarshal(part["output"], &stored) == nil &&
		json.Unmarshal(output.Output, &incoming) == nil && reflect.DeepEqual(stored, incoming)
}

func mustMarshalJSON(value string) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}
