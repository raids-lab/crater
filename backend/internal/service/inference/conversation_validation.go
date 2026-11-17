package inference

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
)

func ValidateKthenaConversationTurn(req *ConversationTurnReq) error {
	if req == nil {
		return bizerr.BadRequest.ParameterError.New("conversation turn is required")
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ClientTurnID = strings.TrimSpace(req.ClientTurnID)
	req.Content = strings.TrimSpace(req.Content)
	if err := ValidateKthenaConversationSessionID(req.SessionID); err != nil {
		return err
	}
	if err := ValidateKthenaConversationSessionID(req.ClientTurnID); err != nil {
		return bizerr.BadRequest.ParameterError.Wrap(err, "invalid clientTurnId")
	}
	if req.Content == "" {
		return bizerr.BadRequest.ParameterError.New("content is required")
	}
	if utf8.RuneCountInString(req.Content) > MaxKthenaConversationContentRunes {
		return bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("content must not exceed %d characters", MaxKthenaConversationContentRunes),
		)
	}
	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
		return bizerr.BadRequest.ParameterError.New("temperature must be between 0 and 2")
	}
	if req.MaxTokens != nil && *req.MaxTokens <= 0 {
		return bizerr.BadRequest.ParameterError.New("maxTokens must be greater than 0")
	}
	return nil
}

func ValidateKthenaConversationSessionID(value string) error {
	if utf8.RuneCountInString(value) > MaxKthenaConversationSessionIDRunes {
		return bizerr.BadRequest.ParameterError.New("sessionId must not exceed 128 characters")
	}
	return nil
}

func ValidateKthenaConversationTitle(value string) error {
	if utf8.RuneCountInString(value) > MaxKthenaConversationTitleRunes {
		return bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("title must not exceed %d characters", MaxKthenaConversationTitleRunes),
		)
	}
	return nil
}

func NormalizeKthenaConversationMessages(
	messages []ConversationMessageReq,
) ([]ConversationMessageReq, error) {
	if len(messages) > MaxKthenaConversationMessages {
		return nil, bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("messages must not exceed %d entries", MaxKthenaConversationMessages),
		)
	}
	normalized := make([]ConversationMessageReq, len(messages))
	for index, message := range messages {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		switch role {
		case KthenaConversationRoleSystem, KthenaConversationRoleUser, KthenaConversationRoleAssistant:
		default:
			return nil, bizerr.BadRequest.ParameterError.New(
				fmt.Sprintf("messages[%d].role must be system, user, or assistant", index),
			)
		}
		content := strings.TrimSpace(message.Content)
		if role != KthenaConversationRoleAssistant && content == "" {
			return nil, bizerr.BadRequest.ParameterError.New(
				fmt.Sprintf("messages[%d].content is required", index),
			)
		}
		if utf8.RuneCountInString(content) > MaxKthenaConversationContentRunes {
			return nil, bizerr.BadRequest.ParameterError.New(
				fmt.Sprintf(
					"messages[%d].content must not exceed %d characters", index, MaxKthenaConversationContentRunes,
				),
			)
		}
		normalized[index] = ConversationMessageReq{Role: role, Content: content}
	}
	return normalized, nil
}

func NormalizeKthenaConversationLimit(value, fallback, maximum int) int {
	if value <= 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func KthenaConversationTitle(messages []ConversationMessageReq) string {
	for _, message := range messages {
		if message.Role != KthenaConversationRoleUser || message.Content == "" {
			continue
		}
		title := strings.Join(strings.Fields(message.Content), " ")
		runes := []rune(title)
		if len(runes) > KthenaConversationTitlePreviewRunes {
			return string(runes[:KthenaConversationTitlePreviewRunes]) + "…"
		}
		return title
	}
	return ""
}

func KthenaConversationToResp(
	conversation *model.KthenaChatSession, messages []model.KthenaChatMessage,
) ConversationResp {
	response := ConversationResp{
		SessionID:    conversation.ClientSessionID,
		Title:        conversation.Title,
		Namespace:    conversation.Namespace,
		ServiceName:  conversation.ServiceName,
		ModelName:    conversation.ModelName,
		BackendType:  conversation.BackendType,
		MessageCount: conversation.MessageCount,
		CreatedAt:    conversation.CreatedAt,
		UpdatedAt:    conversation.UpdatedAt,
	}
	if messages != nil {
		response.Messages = make([]ConversationMessageResp, 0, len(messages))
		for index := range messages {
			response.Messages = append(response.Messages, KthenaConversationMessageToResp(&messages[index]))
		}
	}
	return response
}

func KthenaConversationMessageToResp(message *model.KthenaChatMessage) ConversationMessageResp {
	return ConversationMessageResp{
		Sequence:  message.Sequence,
		Role:      message.Role,
		Content:   message.Content,
		CreatedAt: message.CreatedAt,
	}
}

func KthenaConversationMessages(
	sessionID uint, messages []ConversationMessageReq, createdAt time.Time,
) []model.KthenaChatMessage {
	rows := make([]model.KthenaChatMessage, 0, len(messages))
	for index, message := range messages {
		rows = append(rows, model.KthenaChatMessage{
			SessionID: sessionID,
			Sequence:  index + 1,
			Role:      message.Role,
			Content:   message.Content,
			CreatedAt: createdAt,
		})
	}
	return rows
}

func BuildKthenaConversationTurnBody(
	modelName string, history []model.KthenaChatMessage, req ConversationTurnReq,
) ([]byte, error) {
	messages := make([]ChatMessage, 0, len(history)+1)
	for i := range history {
		message := &history[i]
		messages = append(messages, ChatMessage{Role: message.Role, Content: message.Content})
	}
	messages = append(messages, ChatMessage{Role: KthenaConversationRoleUser, Content: req.Content})
	return json.Marshal(ProxyReq{
		Model:       modelName,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
}

func KthenaAssistantMessageFromCompletion(rawCompletion []byte) (ChatMessage, error) {
	var completion struct {
		Choices []struct {
			Message *ChatMessage `json:"message"`
			Text    string       `json:"text"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rawCompletion, &completion); err != nil {
		return ChatMessage{}, err
	}
	if len(completion.Choices) == 0 {
		return ChatMessage{}, bizerr.Internal.K8sServiceError.New("completion has no choices")
	}
	choice := completion.Choices[0]
	if choice.Message != nil {
		return ChatMessage{
			Role:    nonEmpty(choice.Message.Role, KthenaConversationRoleAssistant),
			Content: choice.Message.Content,
		}, nil
	}
	return ChatMessage{Role: KthenaConversationRoleAssistant, Content: choice.Text}, nil
}

func nonEmpty(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}
