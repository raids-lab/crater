package cronjob

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/robfig/cron/v3"
	"golang.org/x/sync/errgroup"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/util"
	"github.com/raids-lab/crater/pkg/utils"
)

const (
	MAX_RETRY_COUNT = 3
)

func WrapFunc(jobName string, anyFunc util.AnyFunc) cron.FuncJob {
	return func() {
		db := query.GetDB()
		ctx := context.Background()

		// idle -> running
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			cur := &model.CronJobConfig{}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where(query.CronJobConfig.Name.Eq(jobName)).
				First(cur).Error; err != nil {
				return err
			}
			if cur.Status != model.CronJobConfigStatusIdle {
				klog.Errorf("WrapFunc job %s is not in idle status (current: %s)", jobName, cur.Status)
			}
			return tx.Model(cur).Update(
				query.CronJobConfig.Status.ColumnName().String(),
				model.CronJobConfigStatusRunning,
			).Error
		})
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return
			}
			klog.Errorf("WrapFunc failed to update job status to running: %v", err)
			return
		}

		executeTime := utils.GetLocalTime()
		jobResult, err := anyFunc(ctx)
		status := model.CronJobRecordStatusSuccess
		if err != nil {
			status = model.CronJobRecordStatusFailed
			klog.Errorf("AnyFunc %s failed: %v", jobName, err)
		}

		rec := &model.CronJobRecord{
			Name:        jobName,
			ExecuteTime: executeTime,
			Message:     "",
			Status:      status,
		}

		if jobResult != nil {
			if data, err := json.Marshal(jobResult); err != nil {
				klog.Errorf("WrapFunc failed to marshal job result: %v", err)
			} else {
				rec.JobData = datatypes.JSON(data)
			}
		}

		var g errgroup.Group
		g.Go(func() error {
			if err := db.WithContext(ctx).Model(rec).Create(rec).Error; err != nil {
				klog.Errorf("WrapFunc failed to create record: %v", err)
				return err
			}
			return nil
		})

		// running -> idle
		g.Go(func() error {
			var lastErr error
			for range MAX_RETRY_COUNT {
				result := db.WithContext(ctx).
					Model(&model.CronJobConfig{}).
					Where(query.CronJobConfig.Name.Eq(jobName)).
					Where(query.CronJobConfig.Status.Eq(string(model.CronJobConfigStatusRunning))).
					Update(query.CronJobConfig.Status.ColumnName().String(), model.CronJobConfigStatusIdle)
				if result.Error != nil {
					klog.Errorf("WrapFunc failed to update job status to idle: %v", result.Error)
					lastErr = result.Error
					continue
				}
				if result.RowsAffected == 0 {
					klog.Warningf("WrapFunc job %s status changed externally", jobName)
					lastErr = errors.New("job status changed externally")
				}
				break
			}
			return lastErr
		})

		if err := g.Wait(); err != nil {
			klog.Errorf("WrapFunc concurrent operations failed: %v", err)
		}
	}
}
