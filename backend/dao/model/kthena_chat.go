package model

import (
	"time"

	"gorm.io/datatypes"
)

// KthenaChatSession stores one user's conversation with one routed Kthena
// deployment. ClientSessionID is deliberately an application-level UUID: the
// web client can keep using its existing local UUID while the database keeps a
// compact numeric primary key for message joins.
//
// A session is scoped by user, account, namespace, deployment and served
// model. The scope is derived from the authorized ModelBooster object rather
// than from a client supplied model name.
//
//nolint:lll // Composite GORM index declarations must stay in a single struct tag.
type KthenaChatSession struct {
	ID uint `gorm:"primarykey"`

	UserID    uint   `gorm:"not null;uniqueIndex:idx_kthena_chat_session_scope,priority:1;index:idx_kthena_chat_session_list,priority:1;comment:会话所属用户ID"`
	AccountID uint   `gorm:"not null;uniqueIndex:idx_kthena_chat_session_scope,priority:2;index:idx_kthena_chat_session_list,priority:2;comment:会话所属账户ID"`
	Username  string `gorm:"type:varchar(64);not null;comment:创建会话时的用户名快照"`

	Namespace   string `gorm:"type:varchar(63);not null;uniqueIndex:idx_kthena_chat_session_scope,priority:3;index:idx_kthena_chat_session_list,priority:3;comment:模型部署命名空间"`
	ServiceName string `gorm:"type:varchar(63);not null;uniqueIndex:idx_kthena_chat_session_scope,priority:4;index:idx_kthena_chat_session_list,priority:4;comment:ModelBooster名称"`
	ModelName   string `gorm:"type:varchar(256);not null;uniqueIndex:idx_kthena_chat_session_scope,priority:5;index:idx_kthena_chat_session_list,priority:5;comment:路由模型名快照"`
	BackendType string `gorm:"type:varchar(64);not null;comment:推理后端快照"`

	ClientSessionID string     `gorm:"type:varchar(128);not null;uniqueIndex:idx_kthena_chat_session_scope,priority:6;comment:客户端会话UUID"`
	Title           string     `gorm:"type:varchar(256);not null;default:'';comment:会话标题"`
	MessageCount    int        `gorm:"not null;default:0;comment:消息数"`
	LastMessageAt   *time.Time `gorm:"index:idx_kthena_chat_session_list,priority:6;comment:最后一条消息时间"`
	CreatedAt       time.Time  `gorm:"not null;index:idx_kthena_chat_session_list,priority:7"`
	UpdatedAt       time.Time  `gorm:"not null;index:idx_kthena_chat_session_list,priority:8"`
}

func (KthenaChatSession) TableName() string {
	return "kthena_chat_sessions"
}

// KthenaChatMessage is an ordered message in a KthenaChatSession. ResponseJSON
// stores the original non-streaming OpenAI-compatible completion for safe
// idempotent retries of a client turn.
//
//nolint:lll // Composite GORM index declarations must stay in a single struct tag.
type KthenaChatMessage struct {
	ID        uint   `gorm:"primarykey"`
	SessionID uint   `gorm:"not null;uniqueIndex:idx_kthena_chat_message_sequence,priority:1;uniqueIndex:idx_kthena_chat_message_turn,priority:1;index:idx_kthena_chat_message_session,priority:1;comment:会话ID"`
	Sequence  int    `gorm:"not null;uniqueIndex:idx_kthena_chat_message_sequence,priority:2;comment:会话内消息顺序"`
	Role      string `gorm:"type:varchar(32);not null;comment:OpenAI消息角色"`
	Content   string `gorm:"type:text;not null;comment:消息正文"`
	// ClientTurnID is populated only for the user message of an atomic turn.
	// Nullable values keep normal CRUD message replacement unconstrained.
	ClientTurnID *string        `gorm:"type:varchar(128);uniqueIndex:idx_kthena_chat_message_turn,priority:2;comment:客户端幂等请求ID"`
	ResponseJSON datatypes.JSON `gorm:"type:jsonb;comment:助手消息对应的原始模型响应"`
	CreatedAt    time.Time      `gorm:"not null;index:idx_kthena_chat_message_session,priority:2"`
}

func (KthenaChatMessage) TableName() string {
	return "kthena_chat_messages"
}
