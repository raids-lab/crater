package service

import (
	"time"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/payload"
)

func progress(db *gorm.DB, now time.Time) (payload.StatisticsProgress, error) {
	markers, err := dayMarkers(db)
	if err != nil {
		return payload.StatisticsProgress{}, err
	}
	return progressFromMarkers(markers, now), nil
}

// progressFromMarkers 根据日标记汇总覆盖时间、待处理日期和历史回填进度。
func progressFromMarkers(markers []*model.ResourceUsageDaily, now time.Time) payload.StatisticsProgress {
	if len(markers) == 0 {
		return payload.StatisticsProgress{}
	}
	first, today := StatisticsDay(markers[0].StatDate), StatisticsDay(now)
	result := payload.StatisticsProgress{FirstDate: first.Format(time.DateOnly), BackfillComplete: true}
	byDate := make(map[time.Time]*model.ResourceUsageDaily, len(markers))
	for _, row := range markers {
		byDate[StatisticsDay(row.StatDate)] = row
		if row.IsComplete {
			result.Ready = true
			if result.CoveredUntil == nil || row.CoveredUntil.After(*result.CoveredUntil) {
				cutoff := row.CoveredUntil
				result.CoveredUntil = &cutoff
			}
		}
		if result.UpdatedAt == nil || row.UpdatedAt.After(*result.UpdatedAt) {
			updated := row.UpdatedAt
			result.UpdatedAt = &updated
		}
	}
	next := today
	for day := first; !day.After(today); day = day.AddDate(0, 0, 1) {
		row := byDate[day]
		if row == nil || !row.IsComplete || row.CoveredUntil.Before(dayCutoff(day, today)) {
			result.PendingDays++
			if day.Before(today) {
				result.BackfillComplete = false
				if day.Before(next) {
					next = day
				}
			}
		}
	}
	result.CompleteBefore = next.Format(time.DateOnly)
	return result
}
