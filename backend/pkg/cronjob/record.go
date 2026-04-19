package cronjob

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gen/field"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

const (
	MAX_GO_ROUTINE_NUM = 10
)

// GetCronjobRecordTimeRange retrieves the time range of all cronjob records
func (cm *CronJobManager) GetCronjobRecordTimeRange(ctx context.Context) (startTime, endTime time.Time, err error) {
	recordQuery := query.CronJobRecord
	var result struct {
		StartTime time.Time
		EndTime   time.Time
	}
	err = recordQuery.WithContext(ctx).
		Select(
			recordQuery.ExecuteTime.Min().As("start_time"),
			recordQuery.ExecuteTime.Max().As("end_time"),
		).
		Scan(&result)
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
) (records []*model.CronJobRecord, err error) {
	recordQuery := query.CronJobRecord
	tx := query.GetDB().WithContext(ctx)
	if len(names) > 0 {
		tx = tx.Where(recordQuery.Name.In(names...))
	}
	if startTime != nil {
		tx = tx.Where(recordQuery.ExecuteTime.Gte(*startTime))
	}
	if endTime != nil {
		tx = tx.Where(recordQuery.ExecuteTime.Lte(*endTime))
	}
	if status != nil {
		tx = tx.Where(recordQuery.Status.Eq(*status))
	}
	err = tx.
		Order(recordQuery.ExecuteTime.Desc()).
		Find(&records).
		Error
	if err != nil {
		err := fmt.Errorf("CronJobManager.GetCronjobRecords: %w", err)
		klog.Error(err)
		return nil, err
	}
	return records, nil
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
	res := tx.
		Unscoped().
		Delete(&model.CronJobRecord{})
	if err := res.Error; err != nil {
		err := fmt.Errorf("CronJobManager.DeleteCronjobRecords: %w", err)
		klog.Error(err)
		return 0, err
	}

	return res.RowsAffected, nil
}

func (cm *CronJobManager) GetLastCronjobRecord(
	ctx context.Context, names []string, status *string, startTime, endTime *time.Time,
) ([]*model.CronJobRecord, error) {
	recordQuery := query.CronJobRecord
	lastRecordQuery := query.CronJobRecord.As("last_record")
	lastExecuteTime := field.NewTime(lastRecordQuery.TableName(), "last_execute_time")
	subTx := recordQuery.WithContext(ctx).Select(
		recordQuery.Name,
		recordQuery.ExecuteTime.Max().As(lastExecuteTime.ColumnName().String()),
	)

	if len(names) > 0 {
		subTx = subTx.Where(recordQuery.Name.In(names...))
	}
	if status != nil {
		subTx = subTx.Where(recordQuery.Status.Eq(*status))
	}
	if startTime != nil {
		subTx = subTx.Where(recordQuery.ExecuteTime.Gte(*startTime))
	}
	if endTime != nil {
		subTx = subTx.Where(recordQuery.ExecuteTime.Lte(*endTime))
	}
	subTx = subTx.Group(recordQuery.Name)

	tx := recordQuery.WithContext(ctx)
	if len(names) > 0 {
		tx = tx.Where(recordQuery.Name.In(names...))
	}
	tx = tx.Join(
		subTx.As(lastRecordQuery.TableName()),
		recordQuery.Name.EqCol(lastRecordQuery.Name),
		recordQuery.ExecuteTime.EqCol(lastExecuteTime),
	)

	res := make([]*model.CronJobRecord, 0)
	err := tx.Scan(&res)
	if err != nil {
		err := fmt.Errorf("CronJobManager.GetLastCronjobRecord: %w", err)
		klog.Error(err)
		return nil, err
	}

	return res, nil
}
