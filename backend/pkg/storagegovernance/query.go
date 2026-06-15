//nolint:gocritic // Query projection helpers favor value semantics for readability on bounded records.
package storagegovernance

import (
	"context"
	"encoding/json"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/llm"
)

type DecisionRecordSummary struct {
	JobID              string                      `json:"job_id"`
	Username           string                      `json:"username"`
	Source             model.StorageDecisionSource `json:"source"`
	Status             model.StorageDecisionStatus `json:"status"`
	TriggerReason      string                      `json:"trigger_reason"`
	RawAllowExpand     bool                        `json:"raw_allow_expand"`
	RawExpandBytes     int64                       `json:"raw_expand_bytes"`
	RawFreezeNewJobs   bool                        `json:"raw_freeze_new_jobs"`
	FinalAllowExpand   bool                        `json:"final_allow_expand"`
	FinalExpandBytes   int64                       `json:"final_expand_bytes"`
	FinalFreezeNewJobs bool                        `json:"final_freeze_new_jobs"`
	ConstraintAdjusted bool                        `json:"constraint_adjusted"`
	ConstraintBlocked  bool                        `json:"constraint_blocked"`
	AppliedAction      string                      `json:"applied_action"`
	ErrorMessage       string                      `json:"error_message"`
	ConstraintVersion  string                      `json:"constraint_version"`
	LatencyMs          int64                       `json:"latency_ms"`
	CreatedAt          string                      `json:"created_at"`
	UpdatedAt          string                      `json:"updated_at"`
}

type DecisionRecordDetail struct {
	DecisionRecordSummary
	CurrentShrinkStage string                   `json:"current_shrink_stage,omitempty"`
	Snapshot           *DecisionSnapshot        `json:"snapshot,omitempty"`
	RawDecision        *llm.LLMDecisionResponse `json:"raw_decision,omitempty"`
	FinalDecision      *llm.LLMDecisionResponse `json:"final_decision,omitempty"`
	ConstraintResult   *ConstraintEvaluation    `json:"constraint_result,omitempty"`
}

type DecisionRecordPage struct {
	Items      []DecisionRecordSummary `json:"items"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

func ListDecisionRecords(
	ctx context.Context,
	page int,
	pageSize int,
	username string,
	status string,
	source string,
) (*DecisionRecordPage, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	tx := query.GetDB().WithContext(ctx).Model(&model.StorageDecisionRecord{})
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if source != "" {
		tx = tx.Where("source = ?", source)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, err
	}

	var records []model.StorageDecisionRecord
	if err := tx.Order("created_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&records).Error; err != nil {
		return nil, err
	}

	items := make([]DecisionRecordSummary, 0, len(records))
	for _, record := range records {
		items = append(items, summarizeRecord(record))
	}

	return &DecisionRecordPage{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
	}, nil
}

func GetDecisionRecord(ctx context.Context, jobID string) (*DecisionRecordDetail, error) {
	var record model.StorageDecisionRecord
	if err := query.GetDB().WithContext(ctx).Where("job_id = ?", jobID).First(&record).Error; err != nil {
		return nil, err
	}

	detail := &DecisionRecordDetail{
		DecisionRecordSummary: summarizeRecord(record),
	}
	if len(record.Snapshot) > 0 {
		var snapshot DecisionSnapshot
		if err := json.Unmarshal(record.Snapshot, &snapshot); err == nil {
			detail.Snapshot = &snapshot
		}
	}
	if len(record.RawDecision) > 0 {
		var raw llm.LLMDecisionResponse
		if err := json.Unmarshal(record.RawDecision, &raw); err == nil {
			detail.RawDecision = &raw
		}
	}
	if len(record.FinalDecision) > 0 {
		var final llm.LLMDecisionResponse
		if err := json.Unmarshal(record.FinalDecision, &final); err == nil {
			detail.FinalDecision = &final
		}
	}
	if len(record.ConstraintResult) > 0 {
		var evaluation ConstraintEvaluation
		if err := json.Unmarshal(record.ConstraintResult, &evaluation); err == nil {
			detail.ConstraintResult = &evaluation
		}
	}

	var userState struct {
		ShrinkStage string `gorm:"column:shrink_stage"`
	}
	if err := query.GetDB().WithContext(ctx).
		Raw("SELECT shrink_stage FROM users WHERE name = ? AND deleted_at IS NULL", record.Username).
		Scan(&userState).Error; err == nil {
		detail.CurrentShrinkStage = userState.ShrinkStage
	}

	return detail, nil
}

func summarizeRecord(record model.StorageDecisionRecord) DecisionRecordSummary {
	return DecisionRecordSummary{
		JobID:              record.JobID,
		Username:           record.Username,
		Source:             record.Source,
		Status:             record.Status,
		TriggerReason:      record.TriggerReason,
		RawAllowExpand:     record.RawAllowExpand,
		RawExpandBytes:     record.RawExpandBytes,
		RawFreezeNewJobs:   record.RawFreezeNewJobs,
		FinalAllowExpand:   record.FinalAllowExpand,
		FinalExpandBytes:   record.FinalExpandBytes,
		FinalFreezeNewJobs: record.FinalFreezeNewJobs,
		ConstraintAdjusted: record.ConstraintAdjusted,
		ConstraintBlocked:  record.ConstraintBlocked,
		AppliedAction:      record.AppliedAction,
		ErrorMessage:       record.ErrorMessage,
		ConstraintVersion:  record.ConstraintVersion,
		LatencyMs:          record.LatencyMs,
		CreatedAt:          record.CreatedAt.Format(timeLayout),
		UpdatedAt:          record.UpdatedAt.Format(timeLayout),
	}
}

const timeLayout = "2006-01-02 15:04:05"
