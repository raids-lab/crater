package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// UserLLMConfig stores one account-level model configuration. The API key is
// encrypted before persistence and is never serialized to clients.
type UserLLMConfig struct {
	UserID          uint      `gorm:"primaryKey;autoIncrement:false;comment:用户 ID"`
	BaseURL         string    `gorm:"type:varchar(1024);not null;comment:OpenAI 兼容接口基础地址"`
	ModelName       string    `gorm:"type:varchar(256);not null;comment:模型名称"`
	EncryptedAPIKey string    `gorm:"type:text;not null;comment:加密后的 API Key"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

// AgentConversation is the durable owner-scoped container for one chat.
type AgentConversation struct {
	ID            string         `gorm:"type:varchar(36);primaryKey"`
	UserID        uint           `gorm:"index:idx_agent_conversations_identity_updated,priority:1;not null"`
	IdentityScope string         `gorm:"type:varchar(16);index:idx_agent_conversations_identity_updated,priority:2;not null"`
	Title         string         `gorm:"type:varchar(80);not null"`
	CustomTitle   bool           `gorm:"not null;default:false"`
	CreatedAt     time.Time      `gorm:"not null"`
	UpdatedAt     time.Time      `gorm:"index:idx_agent_conversations_identity_updated,priority:3,sort:desc;not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// AgentRun tracks one user turn from submission through completion. Tool calls
// and approvals can be attached to this stable execution boundary later.
type AgentRun struct {
	ID               string     `gorm:"type:varchar(36);primaryKey"`
	ConversationID   string     `gorm:"type:varchar(36);index;not null"`
	UserID           uint       `gorm:"index;not null"`
	IdentityScope    string     `gorm:"type:varchar(16);index;not null"`
	TriggerMessageID string     `gorm:"type:varchar(128);index;not null"`
	Status           string     `gorm:"type:varchar(24);index;not null"`
	ErrorMessage     string     `gorm:"type:text"`
	StartedAt        time.Time  `gorm:"not null"`
	FinishedAt       *time.Time `gorm:"index"`
	CreatedAt        time.Time  `gorm:"not null"`
	UpdatedAt        time.Time  `gorm:"not null"`
}

// AgentMessage stores versioned structured UI parts rather than flattening a
// response to text. Future tool, attachment, form, and artifact parts can be
// added without changing the conversation schema.
type AgentMessage struct {
	ID              string         `gorm:"type:varchar(128);primaryKey"`
	ConversationID  string         `gorm:"type:varchar(36);index:idx_ag_msg_active,priority:1;uniqueIndex:idx_ag_msg_seq,priority:1;not null"`
	RunID           *string        `gorm:"type:varchar(36);index"`
	ParentMessageID *string        `gorm:"type:varchar(128);index"`
	Sequence        int64          `gorm:"index:idx_ag_msg_active,priority:2;uniqueIndex:idx_ag_msg_seq,priority:2;not null"`
	Role            string         `gorm:"type:varchar(16);not null"`
	Status          string         `gorm:"type:varchar(24);not null"`
	SchemaVersion   uint           `gorm:"not null;default:1"`
	Parts           datatypes.JSON `gorm:"type:jsonb;not null"`
	Metadata        datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt       time.Time      `gorm:"not null"`
	UpdatedAt       time.Time      `gorm:"not null"`
	SupersededAt    *time.Time     `gorm:"index:idx_ag_msg_active,priority:3"`
}
