package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/payload"
)

const (
	advisoryLockID int64 = 731949621
	unlockTimeout        = 5 * time.Second
	jobBatchSize         = 500
	dailyBatchSize       = 200
)

// RefreshResult 描述本次刷新处理的日期数和统计进度。
type RefreshResult struct {
	Days     int                        `json:"days"`
	Skipped  bool                       `json:"skipped"`
	Progress payload.StatisticsProgress `json:"progress"`
}

// Refresh 使用 PostgreSQL advisory lock 串行刷新日统计。
func (s *StatisticsService) Refresh(ctx context.Context, now time.Time) (RefreshResult, error) {
	result := RefreshResult{}
	err := s.q.ResourceUsageDaily.WithContext(ctx).UnderlyingDB().
		Session(&gorm.Session{NewDB: true}).Connection(func(db *gorm.DB) (runErr error) {
		db = db.Session(&gorm.Session{})
		var locked bool
		if err := db.Raw("SELECT pg_try_advisory_lock(?)", advisoryLockID).Scan(&locked).Error; err != nil {
			return errors.Join(err, discardStatisticsConnection(db))
		}
		if !locked {
			result.Skipped = true
			return nil
		}
		defer func() {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), unlockTimeout)
			defer cancel()
			runErr = errors.Join(runErr, unlockStatisticsConnection(db.WithContext(cleanup)))
		}()
		plans, err := planDays(db, now)
		if err != nil {
			return err
		}
		// 先初始化每天的完整记录集合，再计算用量。
		for _, plan := range plans {
			if plan.prepare {
				if err := prepareDay(db, plan.day, now); err != nil {
					return err
				}
			}
		}
		// 优先发布当日数据，再按日期恢复历史统计。
		if len(plans) > 0 && plans[len(plans)-1].day.Equal(StatisticsDay(now)) {
			if err := finishPlannedDay(db, plans[len(plans)-1], now); err != nil {
				return err
			}
			result.Days++
			plans = plans[:len(plans)-1]
		}
		for _, plan := range plans {
			if err := finishPlannedDay(db, plan, now); err != nil {
				return err
			}
			result.Days++
		}
		result.Progress, err = progress(db, now)
		return err
	})
	if err != nil && ctx.Err() != nil {
		err = errors.Join(ctx.Err(), err)
	}
	return result, err
}

func finishPlannedDay(db *gorm.DB, plan dayPlan, now time.Time) error {
	if plan.replace {
		return replaceDay(db, plan.day, now)
	}
	return completeDay(db, plan.day)
}

// unlockStatisticsConnection 解锁失败时丢弃连接，避免锁随连接留在池中。
func unlockStatisticsConnection(db *gorm.DB) error {
	var unlocked bool
	err := db.Raw("SELECT pg_advisory_unlock(?)", advisoryLockID).Scan(&unlocked).Error
	if err == nil && unlocked {
		return nil
	}
	if err == nil {
		err = bizerr.Internal.DatabaseError.New(fmt.Sprintf("PostgreSQL did not release statistics advisory lock %d", advisoryLockID))
	}
	return errors.Join(err, discardStatisticsConnection(db))
}

func discardStatisticsConnection(db *gorm.DB) error {
	// GORM Connection 在回调期间固定使用同一个数据库连接。
	conn := db.Statement.ConnPool.(*sql.Conn)
	err := conn.Raw(func(any) error { return driver.ErrBadConn })
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, sql.ErrConnDone) {
		return nil
	}
	return err
}

func loadResources(db *gorm.DB) (map[string]*model.Resource, error) {
	rows, err := query.Use(db).Resource.WithContext(db.Statement.Context).Unscoped().Find()
	if err != nil {
		return nil, err
	}
	resources := make(map[string]*model.Resource, len(rows))
	for i := range rows {
		resources[rows[i].ResourceName] = rows[i]
	}
	return resources, nil
}

func dayCutoff(day, now time.Time) time.Time {
	cutoff := day.AddDate(0, 0, 1)
	if now.Before(cutoff) {
		return now
	}
	return cutoff
}
