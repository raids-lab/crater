package agent

import "encoding/json"

type AgentChatRequest struct {
	Message           string          `json:"message" binding:"required"`
	SessionID         string          `json:"sessionId,omitempty"`
	RequestID         string          `json:"requestId,omitempty"`
	PageContext       json.RawMessage `json:"pageContext,omitempty"`
	ClientContext     json.RawMessage `json:"clientContext,omitempty"`
	OrchestrationMode string          `json:"orchestrationMode,omitempty"`
}

type ConfirmToolRequest struct {
	ConfirmID string          `json:"confirmId" binding:"required"`
	Confirmed bool            `json:"confirmed"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type AgentResumeRequest struct {
	ConfirmID string `json:"confirmId" binding:"required"`
}

type AgentSessionPinRequest struct {
	Pinned bool `json:"pinned"`
}

type ExecuteToolRequest struct {
	ToolName        string                `json:"tool_name" binding:"required"`
	ToolArgs        json.RawMessage       `json:"tool_args" binding:"required"`
	SessionID       string                `json:"session_id" binding:"required"`
	TurnID          string                `json:"turn_id,omitempty"`
	ToolCallID      string                `json:"tool_call_id,omitempty"`
	AgentID         string                `json:"agent_id,omitempty"`
	AgentRole       string                `json:"agent_role,omitempty"`
	InternalContext *AgentInternalContext `json:"internal_context,omitempty"`
}

type AgentInternalContext struct {
	Role        string `json:"role,omitempty"`
	Username    string `json:"username,omitempty"`
	AccountName string `json:"account_name,omitempty"`
}

type AgentConfigSummary struct {
	DefaultOrchestrationMode string   `json:"defaultOrchestrationMode"`
	AvailableModes           []string `json:"availableModes,omitempty"`
}

type AgentTurnRequest struct {
	SessionID string         `json:"session_id"`
	TurnID    string         `json:"turn_id"`
	Message   string         `json:"message"`
	Context   map[string]any `json:"context"`
}

type AgentToolConfirmation struct {
	ConfirmID   string         `json:"confirm_id"`
	ToolName    string         `json:"tool_name"`
	Description string         `json:"description"`
	RiskLevel   string         `json:"risk_level"`
	Interaction string         `json:"interaction,omitempty"`
	Form        *AgentToolForm `json:"form,omitempty"`
}

type AgentToolForm struct {
	Title       string           `json:"title,omitempty"`
	Description string           `json:"description,omitempty"`
	SubmitLabel string           `json:"submitLabel,omitempty"`
	Fields      []AgentToolField `json:"fields,omitempty"`
}

type AgentToolField struct {
	Key          string                 `json:"key"`
	Label        string                 `json:"label"`
	Type         string                 `json:"type"`
	Required     bool                   `json:"required,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Placeholder  string                 `json:"placeholder,omitempty"`
	DefaultValue any                    `json:"defaultValue,omitempty"`
	Options      []AgentToolFieldOption `json:"options,omitempty"`
}

type AgentToolFieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type AgentToolResponse struct {
	ToolCallID   string                 `json:"tool_call_id,omitempty"`
	Status       string                 `json:"status"`
	Result       json.RawMessage        `json:"result,omitempty"`
	Message      string                 `json:"message,omitempty"`
	Confirmation *AgentToolConfirmation `json:"confirmation,omitempty"`
	LatencyMs    int                    `json:"latency_ms,omitempty"`
}
