package service

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gen/field"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/payload"
)

// Filter 描述统计查询的时间范围、粒度和作用域。
type Filter struct {
	Start, End    time.Time
	Scope, Step   string
	TargetID      uint
	allocatedOnly bool
}

func scopedRows(db *gorm.DB, filter *Filter) query.IResourceUsageDailyDo {
	daily := query.Use(db).ResourceUsageDaily
	q := daily.WithContext(db.Statement.Context).Where(daily.StatDate.Gte(filter.Start), daily.StatDate.Lt(filter.End),
		daily.ResourceName.Neq(""), daily.IsComplete.Is(true))
	switch filter.Scope {
	case ScopeUser:
		q = q.Where(daily.UserID.Eq(filter.TargetID))
	case ScopeAccount:
		q = q.Where(daily.AccountID.Eq(filter.TargetID))
	}
	return q
}

// overview 在一致性快照中汇总日用量、时间序列和进度。
func (s *StatisticsService) overview(ctx context.Context, filter *Filter) (*payload.StatisticsResp, error) {
	response := &payload.StatisticsResp{TotalUsage: make(map[string]payload.ResourceDetail), Series: []payload.TimePointData{}}
	err := s.q.ResourceUsageDaily.WithContext(ctx).UnderlyingDB().
		Session(&gorm.Session{NewDB: true}).Transaction(func(tx *gorm.DB) error {
		metadata, err := loadResources(tx)
		if err != nil {
			return err
		}
		markers, err := dayMarkers(tx)
		if err != nil {
			return err
		}

		var rows []model.ResourceUsageDaily
		daily := query.Use(tx).ResourceUsageDaily
		fields := []field.Expr{
			daily.StatDate, daily.ResourceName, daily.UsageSeconds.Sum().As("usage_seconds"),
		}
		if !filter.allocatedOnly {
			fields = append(fields, daily.EffectiveUsageSeconds.Sum().As("effective_usage_seconds"),
				daily.ProfiledUsageSeconds.Sum().As("profiled_usage_seconds"))
		}
		if err := scopedRows(tx, filter).Select(fields...).
			Group(daily.StatDate, daily.ResourceName).Order(daily.StatDate).Scan(&rows); err != nil {
			return err
		}

		days := 1
		if filter.Step == "week" {
			days = 7
		}
		buildStatisticsSeries(response, filter, days, markers, rows)
		totals := make(map[string]*model.ResourceUsageDaily)
		for i := range rows {
			row := &rows[i]
			total := totals[row.ResourceName]
			if total == nil {
				total = &model.ResourceUsageDaily{}
				totals[row.ResourceName] = total
			}
			total.UsageSeconds += row.UsageSeconds
			total.EffectiveUsageSeconds += row.EffectiveUsageSeconds
			total.ProfiledUsageSeconds += row.ProfiledUsageSeconds
		}
		for name, row := range totals {
			response.TotalUsage[name] = resourceDetail(row, name, metadata[name], filter.allocatedOnly)
		}
		response.Progress = progressFromMarkers(markers, time.Now())
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return response, err
}

func calendarDayDiff(start, end time.Time) int {
	return int((StatisticsDay(end).Unix() - StatisticsDay(start).Unix()) / SecondsPerDay)
}

const (
	bucketComplete = "complete"
	bucketPartial  = "partial"
	bucketPending  = "pending"
)

// buildStatisticsSeries 按日或周聚合用量，并根据日标记确定区间状态。
func buildStatisticsSeries(response *payload.StatisticsResp, filter *Filter, days int,
	markers []*model.ResourceUsageDaily, rows []model.ResourceUsageDaily,
) {
	byDay := make(map[time.Time]*model.ResourceUsageDaily, len(markers))
	for _, marker := range markers {
		byDay[StatisticsDay(marker.StatDate)] = marker
	}
	for day := filter.Start; day.Before(filter.End); day = day.AddDate(0, 0, days) {
		bucket := payload.TimePointData{Timestamp: day, Usage: make(map[string]float64),
			Status: bucketStatus(day, minDate(day.AddDate(0, 0, days), filter.End), byDay)}
		if !filter.allocatedOnly {
			bucket.TimePointProfile = &payload.TimePointProfile{EffectiveUsage: make(map[string]float64)}
		}
		response.Series = append(response.Series, bucket)
	}
	for i := range rows {
		row := &rows[i]
		bucket := &response.Series[calendarDayDiff(filter.Start, row.StatDate)/days]
		bucket.Usage[row.ResourceName] += row.UsageSeconds / SecondsPerHour
		if !filter.allocatedOnly && row.ProfiledUsageSeconds > 0 {
			bucket.EffectiveUsage[row.ResourceName] += row.EffectiveUsageSeconds / SecondsPerHour
		}
	}
}

func minDate(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func bucketStatus(start, end time.Time, byDay map[time.Time]*model.ResourceUsageDaily) string {
	status := bucketComplete
	hasMarker := false
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		marker := byDay[day]
		hasMarker = hasMarker || marker != nil
		if marker == nil || !marker.IsComplete || marker.CoveredUntil.Before(day.AddDate(0, 0, 1)) {
			status = bucketPartial
		}
	}
	if !hasMarker {
		return bucketPending
	}
	return status
}

// resourceDetail 将累计秒数转换为小时，并计算利用率和覆盖率。
func resourceDetail(row *model.ResourceUsageDaily, name string, resource *model.Resource, allocatedOnly bool) payload.ResourceDetail {
	detail := payload.ResourceDetail{Usage: row.UsageSeconds / SecondsPerHour, Label: name, Type: "common"}
	if resource != nil && resource.Label != "" {
		detail.Label = resource.Label
	}
	if resource != nil && resource.Type != nil {
		detail.Type = string(*resource.Type)
	}
	if allocatedOnly {
		return detail
	}
	detail.ResourceProfileDetail = &payload.ResourceProfileDetail{
		EffectiveUsage: row.EffectiveUsageSeconds / SecondsPerHour,
		ProfiledUsage:  row.ProfiledUsageSeconds / SecondsPerHour,
	}
	if row.ProfiledUsageSeconds > 0 {
		ratio := row.EffectiveUsageSeconds / row.ProfiledUsageSeconds
		detail.Utilization = &ratio
	}
	if row.UsageSeconds > 0 {
		ratio := row.ProfiledUsageSeconds / row.UsageSeconds
		detail.Coverage = &ratio
	}
	return detail
}
