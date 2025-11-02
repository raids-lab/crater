package cronjob

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

const (
	MAX_GO_ROUTINE_NUM = 10
)

// GetCronjobNames retrieves all cron job names from database
func (cm *CronJobManager) GetCronjobNames(ctx context.Context) ([]string, error) {
	names := make([]string, 0)
	if err := query.GetDB().WithContext(ctx).Model(&model.CronJobConfig{}).Select("name").Find(&names).Error; err != nil {
		err := fmt.Errorf("CronJobManager.GetCronjobNames: %w", err)
		klog.Error(err)
		return nil, err
	}
	return names, nil
}

// GetCronjobRecordTimeRange retrieves the time range of all cronjob records
func (cm *CronJobManager) GetCronjobRecordTimeRange(ctx context.Context) (startTime, endTime time.Time, err error) {
	var result struct {
		StartTime time.Time
		EndTime   time.Time
	}
	err = query.
		GetDB().
		WithContext(ctx).
		Model(&model.CronJobRecord{}).
		Select("min(execute_time) as start_time", "max(execute_time) as end_time").
		Scan(&result).
		Error
	if err != nil {
		err = fmt.Errorf("CronJobManager.GetCronjobRecordTimeRange: %w", err)
		klog.Error(err)
		return time.Time{}, time.Time{}, err
	}
	// 最小时间向下取整到当天的 00:00:00
	startTime = result.StartTime.AddDate(0, 0, -1)
	endTime = result.EndTime.AddDate(0, 0, 1)

	return startTime, endTime, nil
}

// GetCronjobRecords retrieves cronjob records with pagination and filtering
func (cm *CronJobManager) GetCronjobRecords(
	ctx context.Context,
	names []string,
	startTime *time.Time,
	endTime *time.Time,
	status *string,
) (records []*model.CronJobRecord, total int64, err error) {
	g, groupCtx := errgroup.WithContext(ctx)
	g.SetLimit(MAX_GO_ROUTINE_NUM)

	g.Go(func() error {
		tx := query.GetDB().WithContext(groupCtx)
		if len(names) > 0 {
			tx = tx.Where(query.CronJobRecord.Name.In(names...))
		}
		if startTime != nil {
			tx = tx.Where(query.CronJobRecord.ExecuteTime.Gte(*startTime))
		}
		if endTime != nil {
			tx = tx.Where(query.CronJobRecord.ExecuteTime.Lte(*endTime))
		}
		if status != nil {
			tx = tx.Where(query.CronJobRecord.Status.Eq(*status))
		}
		err := tx.
			Find(&records).Error
		if err != nil {
			err := fmt.Errorf("CronJobManager.GetCronjobRecords: %w", err)
			klog.Error(err)
			return err
		}
		return nil
	})

	g.Go(func() error {
		tx := query.GetDB().WithContext(groupCtx)
		if len(names) > 0 {
			tx = tx.Where(query.CronJobRecord.Name.In(names...))
		}
		if startTime != nil {
			tx = tx.Where(query.CronJobRecord.ExecuteTime.Gte(*startTime))
		}
		if endTime != nil {
			tx = tx.Where(query.CronJobRecord.ExecuteTime.Lte(*endTime))
		}
		if status != nil {
			tx = tx.Where(query.CronJobRecord.Status.Eq(*status))
		}

		err := tx.
			Model(&model.CronJobRecord{}).
			Count(&total).
			Error
		if err != nil {
			err := fmt.Errorf("CronJobManager.GetCronjobRecords: %w", err)
			klog.Error(err)
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		err := fmt.Errorf("CronJobManager.GetCronjobRecords: %w", err)
		klog.Error(err)
		return nil, 0, err
	}

	return records, total, nil
}

// DeleteCronjobRecords deletes cronjob records based on the given criteria
func (cm *CronJobManager) DeleteCronjobRecords(
	ctx context.Context,
	ids []uint,
	startTime *time.Time,
	endTime *time.Time,
) (int64, error) {
	tx := query.GetDB().WithContext(ctx)

	if len(ids) > 0 {
		tx = tx.Where(query.CronJobRecord.ID.In(ids...))
	}

	if startTime != nil {
		tx = tx.Where(query.CronJobRecord.ExecuteTime.Gte(*startTime))
	}
	if endTime != nil {
		tx = tx.Where(query.CronJobRecord.ExecuteTime.Lte(*endTime))
	}

	res := tx.Delete(&model.CronJobRecord{})
	if err := res.Error; err != nil {
		err := fmt.Errorf("CronJobManager.DeleteCronjobRecords: %w", err)
		klog.Error(err)
		return 0, err
	}

	return res.RowsAffected, nil
}
