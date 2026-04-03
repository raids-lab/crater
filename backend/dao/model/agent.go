package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AgentSession represents a conversation session between a user and the agent.
type AgentSession struct {
	ID                    uint           `gorm:"primarykey" json:"id"`
	SessionID             string         `gorm:"type:uuid;uniqueIndex;not null" json:"sessionId"`
	UserID                uint           `gorm:"not null" json:"userId"`
	AccountID             uint           `gorm:"not null" json:"accountId"`
	Title                 string         `gorm:"type:varchar(255)" json:"title"`
	PageContext           datatypes.JSON `json:"pageContext"`
	MessageCount          int            `gorm:"default:0" json:"messageCount"`
	LastOrchestrationMode string         `gorm:"type:varchar(32);default:'single_agent'" json:"lastOrchestrationMode"`
	PinnedAt              *time.Time     `gorm:"index" json:"pinnedAt,omitempty"`
	CreatedAt             time.Time      `json:"createdAt"`
	UpdatedAt             time.Time      `json:"updatedAt"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

// AgentMessage represents an individual message in an agent session.
type AgentMessage struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	SessionID  string         `gorm:"type:uuid;index;not null" json:"sessionId"`
	Role       string         `gorm:"type:varchar(20);not null" json:"role"` // user|assistant|tool
	Content    string         `gorm:"type:text" json:"content"`
	ToolCalls  datatypes.JSON `json:"toolCalls,omitempty"`
	ToolCallID string         `gorm:"type:varchar(100)" json:"toolCallId,omitempty"`
	ToolName   string         `gorm:"type:varchar(100)" json:"toolName,omitempty"`
	Metadata   datatypes.JSON `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

// AgentToolCall is an audit log for tool executions, also used for benchmark data collection.
type AgentToolCall struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	SessionID     string         `gorm:"type:uuid;index;not null" json:"sessionId"`
	TurnID        string         `gorm:"type:uuid;index" json:"turnId,omitempty"`
	MessageID     *uint          `json:"messageId,omitempty"`
	ToolCallID    string         `gorm:"type:varchar(128);index" json:"toolCallId,omitempty"`
	AgentID       string         `gorm:"type:varchar(128);index" json:"agentId,omitempty"`
	ParentEventID *uint          `gorm:"index" json:"parentEventId,omitempty"`
	AgentRole     string         `gorm:"type:varchar(32);index" json:"agentRole,omitempty"`
	ToolName      string         `gorm:"type:varchar(100);not null;index" json:"toolName"`
	ToolArgs      datatypes.JSON `gorm:"not null" json:"toolArgs"`
	ToolResult    datatypes.JSON `json:"toolResult,omitempty"`
	ResultStatus  string         `gorm:"type:varchar(32);not null;default:'success'" json:"resultStatus"`
	LatencyMs     int            `json:"latencyMs,omitempty"`
	TokenCount    int            `json:"tokenCount,omitempty"`
	UserConfirmed *bool          `json:"userConfirmed,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
}

// AgentTurn represents one agent execution turn within a session.
type AgentTurn struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	TurnID            string         `gorm:"type:uuid;uniqueIndex;not null" json:"turnId"`
	SessionID         string         `gorm:"type:uuid;index;not null" json:"sessionId"`
	RequestID         string         `gorm:"type:varchar(128);index" json:"requestId,omitempty"`
	OrchestrationMode string         `gorm:"type:varchar(32);default:'single_agent';index" json:"orchestrationMode"`
	RootAgentID       string         `gorm:"type:varchar(128);index" json:"rootAgentId,omitempty"`
	Status            string         `gorm:"type:varchar(32);default:'running';index" json:"status"`
	FinalMessageID    *uint          `gorm:"index" json:"finalMessageId,omitempty"`
	Metadata          datatypes.JSON `json:"metadata,omitempty"`
	StartedAt         time.Time      `gorm:"index" json:"startedAt"`
	EndedAt           *time.Time     `json:"endedAt,omitempty"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

// AgentRunEvent stores semantic events emitted during an agent turn.
type AgentRunEvent struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	TurnID        string         `gorm:"type:uuid;index;not null" json:"turnId"`
	SessionID     string         `gorm:"type:uuid;index;not null" json:"sessionId"`
	AgentID       string         `gorm:"type:varchar(128);index" json:"agentId"`
	ParentAgentID string         `gorm:"type:varchar(128);index" json:"parentAgentId,omitempty"`
	AgentRole     string         `gorm:"type:varchar(32);index" json:"agentRole"`
	EventType     string         `gorm:"type:varchar(64);index" json:"eventType"`
	EventStatus   string         `gorm:"type:varchar(32);index" json:"eventStatus,omitempty"`
	Title         string         `gorm:"type:varchar(255)" json:"title,omitempty"`
	Content       string         `gorm:"type:text" json:"content,omitempty"`
	Metadata      datatypes.JSON `json:"metadata,omitempty"`
	Sequence      int            `gorm:"index" json:"sequence"`
	StartedAt     *time.Time     `json:"startedAt,omitempty"`
	EndedAt       *time.Time     `json:"endedAt,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
}

// JobLogSnapshot is a persisted log snippet captured when a job reaches terminal state.
type JobLogSnapshot struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	JobName       string    `gorm:"type:varchar(255);index;not null" json:"jobName"`
	PodName       string    `gorm:"type:varchar(255);not null" json:"podName"`
	ContainerName string    `gorm:"type:varchar(255);not null" json:"containerName"`
	LogTail       string    `gorm:"type:text" json:"logTail"`
	LogHead       string    `gorm:"type:text" json:"logHead"`
	CapturedAt    time.Time `gorm:"not null" json:"capturedAt"`
	JobStatus     string    `gorm:"type:varchar(50)" json:"jobStatus"`
	CreatedAt     time.Time `json:"createdAt"`
}
