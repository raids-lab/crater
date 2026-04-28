package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

const (
	agentSessionSourceChat      = "chat"
	agentSessionSourceOpsAudit  = "ops_audit"
	agentSessionSourceSystem    = "system"
	agentSessionSourceBenchmark = "benchmark"

	agentToolCallSourceBackend   = "backend"
	agentToolCallSourceLocal     = "local"
	agentToolCallSourceBenchmark = "benchmark"
)

func normalizeAgentSessionSource(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "", agentSessionSourceChat:
		return agentSessionSourceChat
	case agentSessionSourceOpsAudit:
		return agentSessionSourceOpsAudit
	case agentSessionSourceSystem:
		return agentSessionSourceSystem
	case agentSessionSourceBenchmark:
		return agentSessionSourceBenchmark
	default:
		return agentSessionSourceChat
	}
}

func normalizeAgentToolCallSource(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case agentToolCallSourceLocal:
		return agentToolCallSourceLocal
	case agentToolCallSourceBenchmark:
		return agentToolCallSourceBenchmark
	default:
		return agentToolCallSourceBackend
	}
}

type AgentAuditSessionListOptions struct {
	Source  string
	Keyword string
	From    *time.Time
	To      *time.Time
	HasEval string // "" (any) | "yes" | "no"
	Limit   int
	Offset  int
}

type AgentAuditSessionSummary struct {
	Chat      int64 `json:"chat"`
	OpsAudit  int64 `json:"opsAudit"`
	System    int64 `json:"system"`
	Benchmark int64 `json:"benchmark"`
	Total     int64 `json:"total"`
}

type AgentAuditSessionListItem struct {
	SessionID             string     `json:"sessionId"`
	Title                 string     `json:"title"`
	Source                string     `json:"source"`
	UserID                uint       `json:"userId"`
	Username              string     `json:"username,omitempty"`
	Nickname              string     `json:"nickname,omitempty"`
	AccountID             uint       `json:"accountId"`
	AccountName           string     `json:"accountName,omitempty"`
	AccountNickname       string     `json:"accountNickname,omitempty"`
	MessageCount          int        `json:"messageCount"`
	ToolCallCount         int        `json:"toolCallCount"`
	TurnCount             int        `json:"turnCount"`
	LastOrchestrationMode string     `json:"lastOrchestrationMode,omitempty"`
	OrchestrationModes    []string   `json:"orchestrationModes,omitempty"`
	PinnedAt              *time.Time `json:"pinnedAt,omitempty"`
	LatestEvalID          *uint      `json:"latestEvalId,omitempty"`
	LatestEvalScope       string     `json:"latestEvalScope,omitempty"`
	LatestEvalType        string     `json:"latestEvalType,omitempty"`
	LatestEvalStatus      string     `json:"latestEvalStatus,omitempty"`
	LatestEvalCompletedAt *time.Time `json:"latestEvalCompletedAt,omitempty"`
	FeedbackRating        *int16     `json:"feedbackRating,omitempty"`
	HasFeedback           bool       `json:"hasFeedback"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

// agentAuditSessionListRow is the raw scan target; orchestration_modes_raw is a
// comma-separated string aggregated from agent_turns which we split into the
// public []string field after scanning.
type agentAuditSessionListRow struct {
	SessionID             string
	Title                 string
	Source                string
	UserID                uint
	Username              string
	Nickname              string
	AccountID             uint
	AccountName           string
	AccountNickname       string
	MessageCount          int
	ToolCallCount         int
	TurnCount             int
	LastOrchestrationMode string
	OrchestrationModesRaw string
	PinnedAt              *time.Time
	LatestEvalID          *uint
	LatestEvalScope       string
	LatestEvalType        string
	LatestEvalStatus      string
	LatestEvalCompletedAt *time.Time
	FeedbackRating        *int16
	HasFeedback           bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (r agentAuditSessionListRow) toItem() AgentAuditSessionListItem {
	modes := splitOrchestrationModes(r.OrchestrationModesRaw)
	return AgentAuditSessionListItem{
		SessionID:             r.SessionID,
		Title:                 r.Title,
		Source:                r.Source,
		UserID:                r.UserID,
		Username:              r.Username,
		Nickname:              r.Nickname,
		AccountID:             r.AccountID,
		AccountName:           r.AccountName,
		AccountNickname:       r.AccountNickname,
		MessageCount:          r.MessageCount,
		ToolCallCount:         r.ToolCallCount,
		TurnCount:             r.TurnCount,
		LastOrchestrationMode: r.LastOrchestrationMode,
		OrchestrationModes:    modes,
		PinnedAt:              r.PinnedAt,
		LatestEvalID:          r.LatestEvalID,
		LatestEvalScope:       r.LatestEvalScope,
		LatestEvalType:        r.LatestEvalType,
		LatestEvalStatus:      r.LatestEvalStatus,
		LatestEvalCompletedAt: r.LatestEvalCompletedAt,
		FeedbackRating:        r.FeedbackRating,
		HasFeedback:           r.HasFeedback,
		CreatedAt:             r.CreatedAt,
		UpdatedAt:             r.UpdatedAt,
	}
}

func splitOrchestrationModes(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

type AgentAuditSessionListResult struct {
	Total   int64                       `json:"total"`
	Items   []AgentAuditSessionListItem `json:"items"`
	Summary AgentAuditSessionSummary    `json:"summary"`
}

type agentAuditSourceCountRow struct {
	Source string
	Count  int64
}

func (s *AgentService) GetOrCreateSessionWithSource(
	ctx context.Context,
	sessionID string,
	userID, accountID uint,
	title string,
	pageContext json.RawMessage,
	source string,
) (*model.AgentSession, bool, error) {
	var session model.AgentSession
	err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error
	if err == nil {
		return &session, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	created, createErr := s.CreateSession(ctx, sessionID, userID, accountID, title, pageContext, source)
	if createErr != nil {
		return nil, false, createErr
	}
	return created, true, nil
}

func (s *AgentService) buildAgentAuditSessionBaseQuery(ctx context.Context, keyword string) *gorm.DB {
	db := s.db.WithContext(ctx).
		Table("agent_sessions AS s").
		Joins("LEFT JOIN users u ON u.id = s.user_id").
		Joins("LEFT JOIN accounts a ON a.id = s.account_id").
		Where("s.deleted_at IS NULL")

	trimmedKeyword := strings.TrimSpace(keyword)
	if trimmedKeyword == "" {
		return db
	}

	pattern := "%" + trimmedKeyword + "%"
	return db.Where(
		`(
			s.session_id::text ILIKE ?
			OR s.title ILIKE ?
			OR COALESCE(u.name, '') ILIKE ?
			OR COALESCE(u.nickname, '') ILIKE ?
			OR COALESCE(a.name, '') ILIKE ?
			OR COALESCE(a.nickname, '') ILIKE ?
		)`,
		pattern,
		pattern,
		pattern,
		pattern,
		pattern,
		pattern,
	)
}

// buildAgentAuditSessionEnrichedQuery builds the SELECT that feeds both the
// admin list and single-session detail view. The caller controls keyword /
// filters / pagination. Scan target: []agentAuditSessionListRow.
func (s *AgentService) buildAgentAuditSessionEnrichedQuery(ctx context.Context, keyword string) *gorm.DB {
	messageStats := s.db.WithContext(ctx).
		Table("agent_messages").
		Select("session_id, COUNT(*) AS message_count").
		Group("session_id")
	toolCallStats := s.db.WithContext(ctx).
		Table("agent_tool_calls").
		Select("session_id, COUNT(*) AS tool_call_count").
		Group("session_id")
	turnStats := s.db.WithContext(ctx).
		Table("agent_turns").
		Select("session_id, COUNT(*) AS turn_count").
		Group("session_id")

	// Distinct orchestration modes seen across all turns of a session,
	// comma-separated (split in Go). Fallback to session.last_orchestration_mode
	// is handled by the outer SELECT.
	turnModes := s.db.WithContext(ctx).
		Table("agent_turns").
		Select(`session_id,
			STRING_AGG(DISTINCT COALESCE(NULLIF(orchestration_mode, ''), 'single_agent'), ',') AS modes`).
		Group("session_id")

	// Latest quality eval per session (DISTINCT ON picks most recent)
	latestEval := s.db.WithContext(ctx).
		Table("agent_quality_evals").
		Select(`DISTINCT ON (session_id)
			session_id,
			id AS eval_id,
			COALESCE(NULLIF(eval_scope, ''), CASE WHEN turn_id IS NULL THEN 'session' ELSE 'turn' END) AS eval_scope,
			COALESCE(NULLIF(eval_type, ''), 'full') AS eval_type,
			eval_status,
			completed_at`).
		Order("session_id, created_at DESC")

	// Latest submitted feedback rating per session
	latestFeedback := s.db.WithContext(ctx).
		Table("agent_feedbacks").
		Select(`DISTINCT ON (session_id)
			session_id,
			rating AS feedback_rating,
			status AS feedback_status`).
		Where("status = 'submitted'").
		Order("session_id, submitted_at DESC NULLS LAST, updated_at DESC")

	return s.buildAgentAuditSessionBaseQuery(ctx, keyword).
		Joins("LEFT JOIN (?) AS msg_stats ON msg_stats.session_id = s.session_id", messageStats).
		Joins("LEFT JOIN (?) AS tool_stats ON tool_stats.session_id = s.session_id", toolCallStats).
		Joins("LEFT JOIN (?) AS turn_stats ON turn_stats.session_id = s.session_id", turnStats).
		Joins("LEFT JOIN (?) AS turn_modes ON turn_modes.session_id = s.session_id", turnModes).
		Joins("LEFT JOIN (?) AS latest_eval ON latest_eval.session_id = s.session_id", latestEval).
		Joins("LEFT JOIN (?) AS latest_fb ON latest_fb.session_id = s.session_id", latestFeedback).
		Select(`
			s.session_id,
			s.title,
			COALESCE(NULLIF(s.source, ''), 'chat') AS source,
			s.user_id,
			COALESCE(u.name, '') AS username,
			COALESCE(NULLIF(u.nickname, ''), u.name, '') AS nickname,
			s.account_id,
			COALESCE(a.name, '') AS account_name,
			COALESCE(NULLIF(a.nickname, ''), a.name, '') AS account_nickname,
			COALESCE(msg_stats.message_count, s.message_count, 0) AS message_count,
			COALESCE(tool_stats.tool_call_count, 0) AS tool_call_count,
			COALESCE(turn_stats.turn_count, 0) AS turn_count,
			COALESCE(NULLIF(s.last_orchestration_mode, ''), 'single_agent') AS last_orchestration_mode,
			COALESCE(turn_modes.modes, COALESCE(NULLIF(s.last_orchestration_mode, ''), 'single_agent')) AS orchestration_modes_raw,
			s.pinned_at,
			latest_eval.eval_id AS latest_eval_id,
			COALESCE(latest_eval.eval_scope, '') AS latest_eval_scope,
			COALESCE(latest_eval.eval_type, '') AS latest_eval_type,
			COALESCE(latest_eval.eval_status, '') AS latest_eval_status,
			latest_eval.completed_at AS latest_eval_completed_at,
			latest_fb.feedback_rating AS feedback_rating,
			(latest_fb.session_id IS NOT NULL) AS has_feedback,
			s.created_at,
			s.updated_at
		`)
}

func (s *AgentService) ListAdminSessions(
	ctx context.Context,
	opts AgentAuditSessionListOptions,
) (*AgentAuditSessionListResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 40
	}
	if limit > 100 {
		limit = 100
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	source := normalizeAgentSessionSource(opts.Source)
	if strings.TrimSpace(opts.Source) == "" || strings.EqualFold(strings.TrimSpace(opts.Source), "all") {
		source = ""
	}

	summary := AgentAuditSessionSummary{}
	summaryQuery := s.buildAgentAuditSessionBaseQuery(ctx, opts.Keyword).
		Select("COALESCE(NULLIF(s.source, ''), 'chat') AS source, COUNT(*) AS count").
		Group("COALESCE(NULLIF(s.source, ''), 'chat')")

	var summaryRows []agentAuditSourceCountRow
	if err := summaryQuery.Scan(&summaryRows).Error; err != nil {
		return nil, err
	}
	for _, row := range summaryRows {
		switch normalizeAgentSessionSource(row.Source) {
		case agentSessionSourceChat:
			summary.Chat = row.Count
		case agentSessionSourceOpsAudit:
			summary.OpsAudit = row.Count
		case agentSessionSourceSystem:
			summary.System = row.Count
		case agentSessionSourceBenchmark:
			summary.Benchmark = row.Count
		}
	}
	summary.Total = summary.Chat + summary.OpsAudit + summary.System + summary.Benchmark

	countQuery := s.buildAgentAuditSessionBaseQuery(ctx, opts.Keyword)
	if source != "" {
		countQuery = countQuery.Where("COALESCE(NULLIF(s.source, ''), 'chat') = ?", source)
	}

	listQuery := s.buildAgentAuditSessionEnrichedQuery(ctx, opts.Keyword)
	if source != "" {
		listQuery = listQuery.Where("COALESCE(NULLIF(s.source, ''), 'chat') = ?", source)
	}
	if opts.From != nil {
		listQuery = listQuery.Where("s.created_at >= ?", *opts.From)
		countQuery = countQuery.Where("s.created_at >= ?", *opts.From)
	}
	if opts.To != nil {
		listQuery = listQuery.Where("s.created_at <= ?", *opts.To)
		countQuery = countQuery.Where("s.created_at <= ?", *opts.To)
	}
	switch opts.HasEval {
	case "yes":
		listQuery = listQuery.Where("EXISTS (SELECT 1 FROM agent_quality_evals qe WHERE qe.session_id = s.session_id)")
		countQuery = countQuery.Where("EXISTS (SELECT 1 FROM agent_quality_evals qe WHERE qe.session_id = s.session_id)")
	case "no":
		listQuery = listQuery.Where("NOT EXISTS (SELECT 1 FROM agent_quality_evals qe WHERE qe.session_id = s.session_id)")
		countQuery = countQuery.Where("NOT EXISTS (SELECT 1 FROM agent_quality_evals qe WHERE qe.session_id = s.session_id)")
	}

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	var rows []agentAuditSessionListRow
	if err := listQuery.
		Order("s.updated_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]AgentAuditSessionListItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, r.toItem())
	}

	return &AgentAuditSessionListResult{
		Total:   total,
		Items:   items,
		Summary: summary,
	}, nil
}

// GetSessionDetail returns the enriched list-item for a single session. Used by
// the detail page so it does not have to keyword-search for the session.
func (s *AgentService) GetSessionDetail(
	ctx context.Context,
	sessionID string,
) (*AgentAuditSessionListItem, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, ErrSessionNotFound
	}
	var rows []agentAuditSessionListRow
	if err := s.buildAgentAuditSessionEnrichedQuery(ctx, "").
		Where("s.session_id = ?", sessionID).
		Limit(1).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrSessionNotFound
	}
	item := rows[0].toItem()
	return &item, nil
}
