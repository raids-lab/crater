package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/patrol"
)

// EnsureBuiltinCronJobs creates default cron configs required by built-in platform features.
func (s *ConfigService) EnsureBuiltinCronJobs(ctx context.Context) error {
	return s.q.Transaction(func(tx *query.Query) error {
		cjc := tx.CronJobConfig
		defaultConfigs := []*model.CronJobConfig{
			{
				Name:    patrol.TRIGGER_ADMIN_OPS_REPORT_JOB,
				Type:    model.CronJobTypePatrolFunc,
				Spec:    "0 * * * *",
				Status:  model.CronJobConfigStatusSuspended,
				EntryID: -1,
				Config: datatypes.JSON(`{
						"days": 1,
						"lookback_hours": 1,
					"gpu_threshold": 5,
					"idle_hours": 1,
					"running_limit": 20,
					"node_limit": 10
				}`),
			},
		}

		for _, job := range defaultConfigs {
			_, err := cjc.WithContext(ctx).Where(cjc.Name.Eq(job.Name)).First()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if createErr := cjc.WithContext(ctx).Create(job); createErr != nil {
					return fmt.Errorf("failed to create builtin cron job %s: %w", job.Name, createErr)
				}
				continue
			}
			if err != nil {
				return fmt.Errorf("failed to query builtin cron job %s: %w", job.Name, err)
			}
		}
		return nil
	})
}
