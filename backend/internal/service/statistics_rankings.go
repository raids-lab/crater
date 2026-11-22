package service

import (
	"cmp"
	"context"
	"database/sql"
	"slices"
	"time"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/payload"
)

const statisticsOrderDesc = "desc"

// RankingFilter 描述排名维度、指标、资源和分页条件。
type RankingFilter struct {
	Filter
	Dimension, Metric, Order   string
	ResourceName, ResourceType string
	Page, PageSize             int
}

func (s *StatisticsService) rankings(ctx context.Context, filter *RankingFilter) (*payload.StatisticsRankingResp, error) {
	if filter.Dimension != ScopeUser && filter.Dimension != ScopeAccount {
		return nil, bizerr.BadRequest.ParameterError.New("dimension must be user or account")
	}
	if filter.Metric != MetricUsage && filter.Metric != MetricEffectiveUsage && filter.Metric != MetricUtilization {
		return nil, bizerr.BadRequest.ParameterError.New("metric must be usage, effectiveUsage or utilization")
	}
	if filter.Order != "asc" && filter.Order != statisticsOrderDesc {
		return nil, bizerr.BadRequest.ParameterError.New("order must be asc or desc")
	}
	response := &payload.StatisticsRankingResp{Rows: []payload.StatisticsRankingRow{}}
	err := s.q.ResourceUsageDaily.WithContext(ctx).UnderlyingDB().
		Session(&gorm.Session{NewDB: true}).Transaction(func(tx *gorm.DB) error {
		rows, err := aggregateRankings(tx, filter)
		if err != nil {
			return err
		}
		response.Total = int64(len(rows))
		response.Rows = rankStatistics(rows, filter)
		if err := loadRankingNames(tx, filter.Dimension, response.Rows); err != nil {
			return err
		}
		response.Progress, err = progress(tx, time.Now())
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return response, err
}

// aggregateRankings 在数据库按用户或账户聚合后再排序。
func aggregateRankings(db *gorm.DB, filter *RankingFilter) ([]payload.StatisticsRankingRow, error) {
	q, ctx := query.Use(db), db.Statement.Context
	daily := q.ResourceUsageDaily
	entity := daily.UserID
	if filter.Dimension == ScopeAccount {
		entity = daily.AccountID
	}
	aggregated := scopedRows(db, &filter.Filter)
	if filter.ResourceName != "" {
		aggregated = aggregated.Where(daily.ResourceName.Eq(filter.ResourceName))
	} else {
		resource := q.Resource
		aggregated = aggregated.Where(daily.Columns(daily.ResourceName).In(
			resource.WithContext(ctx).Unscoped().Select(resource.ResourceName).Where(resource.Type.Eq(filter.ResourceType))))
	}
	if filter.Metric != MetricUsage {
		aggregated = aggregated.Having(daily.ProfiledUsageSeconds.Sum().Gt(0))
	}
	rows := []payload.StatisticsRankingRow{}
	err := aggregated.Select(entity.As("id"), daily.UsageSeconds.Sum().As("usage"),
		daily.EffectiveUsageSeconds.Sum().As("effective_usage"),
		daily.ProfiledUsageSeconds.Sum().As("profiled_usage")).Group(entity).Scan(&rows)
	for i := range rows {
		row := &rows[i]
		row.Usage /= SecondsPerHour
		row.EffectiveUsage /= SecondsPerHour
		row.ProfiledUsage /= SecondsPerHour
		if row.ProfiledUsage > 0 {
			ratio := row.EffectiveUsage / row.ProfiledUsage
			row.Utilization = &ratio
		}
		if row.Usage > 0 {
			ratio := row.ProfiledUsage / row.Usage
			row.Coverage = &ratio
		}
	}
	return rows, err
}

func rankingScore(row *payload.StatisticsRankingRow, metric string) float64 {
	switch metric {
	case MetricEffectiveUsage:
		return row.EffectiveUsage
	case MetricUtilization:
		return *row.Utilization
	default:
		return row.Usage
	}
}

// rankStatistics 按指标排序、计算并列名次并分页。
func rankStatistics(rows []payload.StatisticsRankingRow, filter *RankingFilter) []payload.StatisticsRankingRow {
	slices.SortFunc(rows, func(a, b payload.StatisticsRankingRow) int {
		order := cmp.Compare(rankingScore(&a, filter.Metric), rankingScore(&b, filter.Metric))
		if filter.Order == statisticsOrderDesc {
			order = -order
		}
		if order == 0 {
			return cmp.Compare(a.ID, b.ID)
		}
		return order
	})
	var rank int64
	var previous float64
	for i := range rows {
		score := rankingScore(&rows[i], filter.Metric)
		if i == 0 || score != previous {
			rank++
		}
		rows[i].Rank, previous = rank, score
	}
	if filter.Page-1 > len(rows)/filter.PageSize {
		return []payload.StatisticsRankingRow{}
	}
	start := (filter.Page - 1) * filter.PageSize
	end := min(start+filter.PageSize, len(rows))
	return rows[start:end]
}

func loadRankingNames(db *gorm.DB, dimension string, rows []payload.StatisticsRankingRow) error {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]uint, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	var names []struct {
		ID             uint
		Name, Nickname string
	}
	q, ctx := query.Use(db), db.Statement.Context
	if dimension == ScopeUser {
		user := q.User
		if err := user.WithContext(ctx).Unscoped().Select(user.ID, user.Name, user.Nickname).Where(user.ID.In(ids...)).Scan(&names); err != nil {
			return err
		}
	} else {
		account := q.Account
		if err := account.WithContext(ctx).Unscoped().Select(account.ID, account.Name, account.Nickname).
			Where(account.ID.In(ids...)).Scan(&names); err != nil {
			return err
		}
	}
	byID := make(map[uint]int, len(rows))
	for i := range rows {
		byID[rows[i].ID] = i
	}
	for _, name := range names {
		rows[byID[name.ID]].Name, rows[byID[name.ID]].Nickname = name.Name, name.Nickname
	}
	return nil
}
