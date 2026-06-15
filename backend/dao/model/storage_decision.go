package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type StorageDecisionStatus string

const (
	StorageDecisionStatusPending StorageDecisionStatus = "pending"
	StorageDecisionStatusRunning StorageDecisionStatus = "running"
	StorageDecisionStatusDone    StorageDecisionStatus = "done"
	StorageDecisionStatusError   StorageDecisionStatus = "error"
)

type StorageDecisionSource string

const (
	StorageDecisionSourceManual StorageDecisionSource = "manual"
	StorageDecisionSourcePatrol StorageDecisionSource = "patrol"
	StorageDecisionSourceReplay StorageDecisionSource = "replay"
)

// StorageDecisionRecord stores the full decision trail of a storage governance run.
// It keeps the input snapshot, the raw LLM proposal, and the final decision after
// applying platform safety constraints so the result can be audited and replayed.
type StorageDecisionRecord struct {
	gorm.Model
	JobID         string                `gorm:"type:varchar(64);not null;uniqueIndex;comment:决策任务ID" json:"jobId"`
	UserID        uint                  `gorm:"index;comment:用户ID" json:"userId"`
	Username      string                `gorm:"type:varchar(64);not null;index;comment:用户名" json:"username"`
	Source        StorageDecisionSource `gorm:"type:varchar(32);not null;index;comment:触发来源" json:"source"`
	Status        StorageDecisionStatus `gorm:"type:varchar(32);not null;index;default:pending;comment:决策状态" json:"status"`
	TriggerReason string                `gorm:"type:text;comment:触发原因" json:"triggerReason"`

	Snapshot         datatypes.JSON `gorm:"type:jsonb;comment:决策时的输入快照" json:"snapshot"`
	RawDecision      datatypes.JSON `gorm:"type:jsonb;comment:LLM原始决策" json:"rawDecision"`
	FinalDecision    datatypes.JSON `gorm:"type:jsonb;comment:约束校验后的最终决策" json:"finalDecision"`
	ConstraintResult datatypes.JSON `gorm:"type:jsonb;comment:安全约束评估结果" json:"constraintResult"`

	RawAllowExpand     bool       `gorm:"not null;default:false;comment:LLM原始是否允许扩容" json:"rawAllowExpand"`
	RawExpandBytes     int64      `gorm:"not null;default:0;comment:LLM原始建议扩容量" json:"rawExpandBytes"`
	RawFreezeNewJobs   bool       `gorm:"not null;default:false;comment:LLM原始是否冻结新作业" json:"rawFreezeNewJobs"`
	FinalAllowExpand   bool       `gorm:"not null;default:false;comment:最终是否允许扩容" json:"finalAllowExpand"`
	FinalExpandBytes   int64      `gorm:"not null;default:0;comment:最终扩容量" json:"finalExpandBytes"`
	FinalFreezeNewJobs bool       `gorm:"not null;default:false;comment:最终是否冻结新作业" json:"finalFreezeNewJobs"`
	ConstraintAdjusted bool       `gorm:"not null;default:false;comment:安全约束是否调整了决策" json:"constraintAdjusted"`
	ConstraintBlocked  bool       `gorm:"not null;default:false;comment:安全约束是否阻断了扩容" json:"constraintBlocked"`
	AppliedAction      string     `gorm:"type:varchar(64);comment:最终动作摘要" json:"appliedAction"`
	ErrorMessage       string     `gorm:"type:text;comment:错误信息" json:"errorMessage"`
	StartedAt          *time.Time `gorm:"comment:开始时间" json:"startedAt"`
	FinishedAt         *time.Time `gorm:"comment:结束时间" json:"finishedAt"`
	LatencyMs          int64      `gorm:"not null;default:0;comment:决策耗时毫秒" json:"latencyMs"`
	ConstraintVersion  string     `gorm:"type:varchar(64);comment:约束策略版本" json:"constraintVersion"`
}

func (StorageDecisionRecord) TableName() string {
	return "storage_decision_records"
}
