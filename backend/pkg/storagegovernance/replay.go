package storagegovernance

import (
	"context"
	"encoding/json"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/llm"
)

func ReplayStoredDecisions(
	ctx context.Context,
	cfg ConstraintConfig,
	limit int,
) (*ReplaySummary, error) {
	if limit <= 0 {
		limit = 100
	}

	var records []model.StorageDecisionRecord
	if err := query.GetDB().WithContext(ctx).
		Where("status = ?", model.StorageDecisionStatusDone).
		Order("created_at desc").
		Limit(limit).
		Find(&records).Error; err != nil {
		return nil, err
	}

	summary := &ReplaySummary{
		PolicyVersion: cfg.PolicyVersion,
		Records:       make([]ReplayRecord, 0, len(records)),
	}

	for i := range records {
		record := &records[i]
		if len(record.Snapshot) == 0 || len(record.RawDecision) == 0 {
			continue
		}

		var snapshot DecisionSnapshot
		if err := json.Unmarshal(record.Snapshot, &snapshot); err != nil {
			return nil, err
		}

		var rawDecision llm.LLMDecisionResponse
		if err := json.Unmarshal(record.RawDecision, &rawDecision); err != nil {
			return nil, err
		}

		replayedDecision, evaluation := ApplySafetyConstraints(snapshot, rawDecision, cfg, record.CreatedAt)
		replayRecord := ReplayRecord{
			JobID:             record.JobID,
			Username:          record.Username,
			StoredAdjusted:    record.ConstraintAdjusted,
			StoredBlocked:     record.ConstraintBlocked,
			ReplayAdjusted:    evaluation.Adjusted,
			ReplayBlocked:     evaluation.Blocked,
			StoredAllowExpand: record.FinalAllowExpand,
			ReplayAllowExpand: replayedDecision.AllowExpand,
			StoredExpandBytes: record.FinalExpandBytes,
			ReplayExpandBytes: replayedDecision.ExpandBytes,
			StoredFreeze:      record.FinalFreezeNewJobs,
			ReplayFreeze:      replayedDecision.FreezeNewJobs,
			Evaluation:        evaluation,
		}

		summary.TotalCases++
		if evaluation.Blocked {
			summary.BlockedCases++
		}
		if evaluation.Adjusted {
			summary.ClampedCases++
		}
		if !record.FinalFreezeNewJobs && replayedDecision.FreezeNewJobs {
			summary.FreezeEscalations++
		}
		if record.FinalAllowExpand != replayedDecision.AllowExpand ||
			record.FinalExpandBytes != replayedDecision.ExpandBytes ||
			record.FinalFreezeNewJobs != replayedDecision.FreezeNewJobs {
			summary.ChangedCases++
		}

		summary.Records = append(summary.Records, replayRecord)
	}

	return summary, nil
}
