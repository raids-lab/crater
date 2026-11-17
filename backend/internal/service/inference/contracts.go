package inference

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

const (
	DefaultKthenaConversationLimit        = 30
	MaxKthenaConversationLimit            = 100
	DefaultKthenaConversationMessageLimit = 100
	MaxKthenaConversationMessageLimit     = 500
	MaxKthenaConversationMessages         = 500
	MaxKthenaConversationTitleRunes       = 256
	MaxKthenaConversationContentRunes     = 32768
	MaxKthenaConversationTurnHistory      = 100
	MaxKthenaConversationSessionIDRunes   = 128
	KthenaConversationTitlePreviewRunes   = 48
	KthenaConversationLogVerbosity        = 4
)

const (
	KthenaConversationRoleSystem    = "system"
	KthenaConversationRoleUser      = "user"
	KthenaConversationRoleAssistant = "assistant"
)

type ConversationStore struct {
	DB *gorm.DB
}

// ConversationMessageReq is one OpenAI-compatible message persisted in
// a conversation. Update requests replace the complete ordered message list.
type ConversationMessageReq struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ConversationCreateReq creates a conversation. SessionID is optional:
// existing clients can provide their own UUID, while an empty value gets a
// server-generated UUID in the response.
type ConversationCreateReq struct {
	SessionID string                   `json:"sessionId"`
	Title     string                   `json:"title"`
	Messages  []ConversationMessageReq `json:"messages"`
}

// ConversationUpdateReq changes the title and/or replaces all messages.
// A nil Messages field means leave the current messages untouched; an empty
// array clears them.
type ConversationUpdateReq struct {
	Title    *string                   `json:"title"`
	Messages *[]ConversationMessageReq `json:"messages"`
}

// ConversationListReq controls the bounded conversation history list.
type ConversationListReq struct {
	IncludeMessages bool `form:"includeMessages"`
	Limit           int  `form:"limit"`
	MessageLimit    int  `form:"messageLimit"`
}

// ConversationTurnReq sends one user turn atomically. The backend reads
// the persisted context, calls Kthena, and stores the user/assistant pair only
// after a successful non-streaming completion. ClientTurnID is optional but
// makes retries idempotent for an existing conversation.
type ConversationTurnReq struct {
	SessionID    string   `json:"sessionId"`
	Content      string   `json:"content"`
	Temperature  *float64 `json:"temperature"`
	MaxTokens    *int64   `json:"maxTokens"`
	ClientTurnID string   `json:"clientTurnId"`
}

// ConversationMessageResp is a stored message returned to the client.
type ConversationMessageResp struct {
	Sequence  int       `json:"sequence"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// ConversationResp is the persisted, deployment-scoped conversation.
// Messages are populated for conversation detail and when requested from list.
type ConversationResp struct {
	SessionID    string                    `json:"sessionId"`
	Title        string                    `json:"title"`
	Namespace    string                    `json:"namespace"`
	ServiceName  string                    `json:"serviceName"`
	ModelName    string                    `json:"modelName"`
	BackendType  string                    `json:"backendType"`
	MessageCount int                       `json:"messageCount"`
	CreatedAt    time.Time                 `json:"createdAt"`
	UpdatedAt    time.Time                 `json:"updatedAt"`
	Messages     []ConversationMessageResp `json:"messages,omitempty"`
}

// ConversationTurnResp returns the canonical persisted assistant turn
// alongside the original OpenAI-compatible completion object.
type ConversationTurnResp struct {
	Conversation ConversationResp        `json:"conversation"`
	Assistant    ConversationMessageResp `json:"assistant"`
	Completion   json.RawMessage         `json:"completion" swaggertype:"object"`
}

type ConversationScope struct {
	UserID      uint
	AccountID   uint
	Username    string
	Namespace   string
	ServiceName string
	// ModelName is the served model snapshot persisted with the conversation.
	// It is intentionally distinct from RouteModelName: Kthena routes requests
	// by the ModelRoute name, while a vLLM served-model-name can be
	// an entirely different user-facing identifier.
	ModelName      string
	RouteModelName string
	BackendType    string
}

type ProxyReq struct {
	Model       string         `json:"model,omitempty"`
	Messages    []ChatMessage  `json:"messages,omitempty"`
	Prompt      any            `json:"prompt,omitempty"`
	Temperature *float64       `json:"temperature,omitempty"`
	MaxTokens   *int64         `json:"max_tokens,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
	Extra       map[string]any `json:"-"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
