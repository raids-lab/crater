package service

import (
	"context"
	"time"

	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

type dayPlan struct {
	day     time.Time
	prepare bool
	replace bool
}

func jobsForDay(ctx context.Context, q *query.Query, day, cutoff time.Time) query.IJobDo {
	job := q.Job
	return job.WithContext(ctx).Unscoped().
		Where(job.JobType.Neq(string(model.JobTypeTensorboard)), jobOverlap(q, day, cutoff))
}

// jobOverlap 筛选与统计区间重叠的作业，考虑结束和删除时间。
func jobOverlap(q *query.Query, day, cutoff time.Time) field.Expr {
	job := q.Job
	day = day.UTC()
	open := field.Or(job.CompletedTimestamp.Eq(time.Time{}), job.CompletedTimestamp.IsNull())
	notDeletedByDay := field.Or(job.DeletedAt.IsNull(), job.DeletedAt.Gt(gorm.DeletedAt{Time: day, Valid: true}))
	return field.And(
		job.RunningTimestamp.Gt(time.Time{}),
		job.RunningTimestamp.Lt(cutoff.UTC()),
		field.Or(job.CompletedTimestamp.Gt(day), field.And(open, notDeletedByDay)),
	)
}

func dayMarkers(db *gorm.DB) ([]*model.ResourceUsageDaily, error) {
	daily := query.Use(db).ResourceUsageDaily
	return daily.WithContext(db.Statement.Context).
		Select(daily.StatDate, daily.CoveredUntil, daily.IsComplete, daily.UpdatedAt).
		Where(daily.ResourceName.Eq("")).Order(daily.StatDate).Find()
}

// planDays 根据日进度和源数据变化安排初始化、续算或替换。
func planDays(db *gorm.DB, now time.Time) ([]dayPlan, error) {
	var source struct{ Earliest *time.Time }
	job := query.Use(db).Job
	if err := job.WithContext(db.Statement.Context).Unscoped().Select(job.RunningTimestamp.Min().As("earliest")).
		Where(job.RunningTimestamp.Gt(time.Time{}), job.RunningTimestamp.Lt(now), job.JobType.Neq(string(model.JobTypeTensorboard))).
		Scan(&source); err != nil {
		return nil, err
	}
	first := StatisticsDay(now)
	if source.Earliest != nil {
		first = StatisticsDay(*source.Earliest)
	}
	markers, err := dayMarkers(db)
	if err != nil {
		return nil, err
	}
	byDate := make(map[time.Time]*model.ResourceUsageDaily, len(markers))
	for _, row := range markers {
		byDate[StatisticsDay(row.StatDate)] = row
		if row.StatDate.Before(first) {
			first = StatisticsDay(row.StatDate)
		}
	}
	plans := make([]dayPlan, 0)
	today := StatisticsDay(now)
	for day := first; !day.After(today); day = day.AddDate(0, 0, 1) {
		row := byDate[day]
		switch {
		case row == nil:
			plans = append(plans, dayPlan{day: day, prepare: true})
		case !row.IsComplete:
			// 未完成的检查点先按原截止时间完成，再处理源数据变化。
			plans = append(plans, dayPlan{day: day})
		default:
			stale := row.CoveredUntil.Before(dayCutoff(day, now))
			if !stale {
				stale, err = daySourcesChanged(db, day, row)
				if err != nil {
					return nil, err
				}
			}
			if stale {
				plans = append(plans, dayPlan{day: day, replace: true})
			}
		}
	}
	return plans, nil
}

// statisticsWatermark 在源数据快照前采样活动事务，避免漏掉并发提交的变更。
func statisticsWatermark(db *gorm.DB) (time.Time, error) {
	var watermark time.Time
	err := db.Raw(`
		SELECT LEAST(statement_timestamp() - INTERVAL '1 microsecond', MIN(xact_start) - INTERVAL '1 millisecond')
		FROM pg_stat_activity
		WHERE datname = current_database() AND pid <> pg_backend_pid()
	`).Scan(&watermark).Error
	return watermark.UTC(), err
}

// daySourcesChanged 根据更新时间检测已发布日期的源数据变化。
func daySourcesChanged(db *gorm.DB, day time.Time, marker *model.ResourceUsageDaily) (bool, error) {
	q := query.Use(db)
	ctx := db.Statement.Context
	job, resource := q.Job, q.Resource
	owner := q.ResourceUsageDaily.As("owner_usage")
	ownerUsage := owner.WithContext(ctx).Select(owner.UserID).
		Where(owner.StatDate.Eq(day), owner.ResourceName.Neq(""), owner.UserID.EqCol(job.UserID), owner.AccountID.EqCol(job.AccountID))
	// 已移出当天的旧作业仍可能在该用户的已发布用量中。
	priorOwnerJob := job.WithContext(ctx).Where(job.CreationTimestamp.Lt(marker.CoveredUntil.UTC()), gen.Exists(ownerUsage))
	updatedJobs := job.WithContext(ctx).Unscoped().Select(job.ID).
		Where(job.JobType.Neq(string(model.JobTypeTensorboard))).
		Where(field.Or(job.UpdatedAt.Gt(marker.UpdatedAt),
			job.DeletedAt.Gt(gorm.DeletedAt{Time: marker.UpdatedAt, Valid: true}))).
		Where(job.WithContext(ctx).Where(jobOverlap(q, day, marker.CoveredUntil)).Or(priorOwnerJob))
	updatedResources := resource.WithContext(ctx).Unscoped().Select(resource.ID).
		Where(resource.UpdatedAt.Gt(marker.UpdatedAt)).
		Or(resource.DeletedAt.Gt(gorm.DeletedAt{Time: marker.UpdatedAt, Valid: true}))

	published := q.ResourceUsageDaily.As("published_day")
	changed := published.WithContext(ctx).Where(gen.Exists(updatedJobs)).Or(gen.Exists(updatedResources))
	count, err := published.WithContext(ctx).Where(published.StatDate.Eq(day), published.ResourceName.Eq(""), changed).Count()
	return count != 0, err
}
