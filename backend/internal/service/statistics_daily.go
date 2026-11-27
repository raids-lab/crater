package service

import (
	"database/sql"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gen"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

// prepareDay 在事务中发布日标记和全部占位行，任一批次失败时整体回滚。
func prepareDay(db *gorm.DB, day, now time.Time) error {
	watermark, err := statisticsWatermark(db)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return initializeDay(tx, day, dayCutoff(day, now), watermark)
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
}

func initializeDay(db *gorm.DB, day, cutoff, watermark time.Time) error {
	keys, err := initialDaySources(db, day, cutoff)
	if err != nil {
		return err
	}
	rows := make([]*model.ResourceUsageDaily, 0, len(keys)+1)
	rows = append(rows, &model.ResourceUsageDaily{
		StatDate: day, CoveredUntil: cutoff, UpdatedAt: watermark,
	})
	for key := range keys {
		rows = append(rows, &model.ResourceUsageDaily{StatDate: day, UserID: key.user, AccountID: key.account,
			ResourceName: key.resource, CoveredUntil: cutoff, UpdatedAt: watermark})
	}
	daily := query.Use(db).ResourceUsageDaily
	if _, err := daily.WithContext(db.Statement.Context).Where(daily.StatDate.Eq(day)).Delete(); err != nil {
		return err
	}
	return daily.WithContext(db.Statement.Context).CreateInBatches(rows, dailyBatchSize)
}

type daySourceJob struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	AccountID uint
	Resources datatypes.JSONType[v1.ResourceList]
}

func initialDaySources(db *gorm.DB, day, cutoff time.Time) (keys map[usageKey]struct{}, err error) {
	q := query.Use(db)
	job := q.Job
	keys = make(map[usageKey]struct{})
	var jobs []daySourceJob
	err = jobsForDay(db.Statement.Context, q, day, cutoff).Select(job.ID, job.UserID, job.AccountID, job.Resources).
		UnderlyingDB().Model(&daySourceJob{}).Table(job.TableName()).
		FindInBatches(&jobs, jobBatchSize, func(_ *gorm.DB, _ int) error {
			for i := range jobs {
				row := &jobs[i]
				for name := range ResourceAmounts(row.Resources.Data()) {
					keys[usageKey{row.UserID, row.AccountID, name}] = struct{}{}
				}
			}
			return nil
		}).Error
	return keys, err
}

func completeDay(db *gorm.DB, day time.Time) error {
	daily := query.Use(db).ResourceUsageDaily
	ctx := db.Statement.Context
	pending, err := daily.WithContext(ctx).
		Where(daily.StatDate.Eq(day), daily.ResourceName.Neq(""), daily.IsComplete.Is(false)).
		Order(daily.UserID, daily.AccountID, daily.ResourceName).Find()
	if err != nil {
		return err
	}
	for start := 0; start < len(pending); {
		end := start + 1
		for end < len(pending) && pending[end].UserID == pending[start].UserID && pending[end].AccountID == pending[start].AccountID {
			end++
		}
		values, err := calculateOwnerDay(db, pending[start])
		if err != nil {
			return err
		}
		for i := start; i < end; i++ {
			row := pending[i]
			value := values[usageKey{row.UserID, row.AccountID, row.ResourceName}]
			if value == nil {
				value = &model.ResourceUsageDaily{}
			}
			// 用量和完成标记在同一次更新中写入。
			if _, err := daily.WithContext(ctx).
				Where(daily.StatDate.Eq(day), daily.UserID.Eq(row.UserID), daily.AccountID.Eq(row.AccountID),
					daily.ResourceName.Eq(row.ResourceName), daily.IsComplete.Is(false)).
				UpdateSimple(daily.UsageSeconds.Value(value.UsageSeconds), daily.EffectiveUsageSeconds.Value(value.EffectiveUsageSeconds),
					daily.ProfiledUsageSeconds.Value(value.ProfiledUsageSeconds), daily.IsComplete.Value(true)); err != nil {
				return err
			}
		}
		start = end
	}
	// 日标记也表示没有资源记录的已初始化日期。
	entry := daily.As("entry")
	unfinished := entry.WithContext(ctx).
		Where(entry.StatDate.Eq(day), entry.ResourceName.Neq(""), entry.IsComplete.Is(false))
	_, err = daily.WithContext(ctx).Where(daily.StatDate.Eq(day), daily.ResourceName.Eq("")).
		Not(gen.Exists(unfinished)).UpdateSimple(daily.IsComplete.Value(true))
	return err
}

// calculateOwnerDay 按账户和用户扫描作业，各资源共享扫描结果。
func calculateOwnerDay(db *gorm.DB, row *model.ResourceUsageDaily) (map[usageKey]*model.ResourceUsageDaily, error) {
	var values map[usageKey]*model.ResourceUsageDaily
	err := db.Transaction(func(tx *gorm.DB) error {
		resources, err := loadResources(tx)
		if err != nil {
			return err
		}
		a := dayAccumulator{day: row.StatDate, cutoff: row.CoveredUntil, resources: resources, rows: make(map[usageKey]*model.ResourceUsageDaily)}
		var jobs []*model.Job
		q := query.Use(tx)
		job := q.Job
		if err := jobsForDay(tx.Statement.Context, q, row.StatDate, row.CoveredUntil).
			Where(job.UserID.Eq(row.UserID), job.AccountID.Eq(row.AccountID)).
			Select(job.ID, job.UserID, job.AccountID, job.JobType, job.RunningTimestamp,
				job.CompletedTimestamp, job.DeletedAt, job.Resources, job.ProfileData).
			FindInBatches(&jobs, jobBatchSize, func(_ gen.Dao, _ int) error {
				for i := range jobs {
					a.add(jobs[i])
				}
				return nil
			}); err != nil {
			return err
		}
		values = a.rows
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return values, err
}

// replaceDay 在事务中原子替换已发布的日统计，完成前保留旧结果。
func replaceDay(db *gorm.DB, day, now time.Time) error {
	watermark, err := statisticsWatermark(db)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := initializeDay(tx, day, dayCutoff(day, now), watermark); err != nil {
			return err
		}
		return completeDay(tx, day)
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
}
