package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AgentSession represents a conversation session between a user and the agent.
type AgentSession struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	SessionID    string         `gorm:"type:uuid;uniqueIndex;not null" json:"sessionId"`
	UserID       uint           `gorm:"not null" json:"userId"`
	AccountID    uint           `gorm:"not null" json:"accountId"`
	Title        string         `gorm:"type:varchar(255)" json:"title"`
	PageContext  datatypes.JSON `json:"pageContext"`
	MessageCount int            `gorm:"default:0" json:"messageCount"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deletedAt"`
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
	MessageID     *uint          `json:"messageId,omitempty"`
	ToolName      string         `gorm:"type:varchar(100);not null;index" json:"toolName"`
	ToolArgs      datatypes.JSON `gorm:"not null" json:"toolArgs"`
	ToolResult    datatypes.JSON `json:"toolResult,omitempty"`
	ResultStatus  string         `gorm:"type:varchar(20);not null;default:'success'" json:"resultStatus"`
	LatencyMs     int            `json:"latencyMs,omitempty"`
	TokenCount    int            `json:"tokenCount,omitempty"`
	UserConfirmed *bool          `json:"userConfirmed,omitempty"`
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
