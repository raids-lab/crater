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
						"node_limit": 10,
						"notification": {
							"enabled": false,
							"notify_admins": true,
							"notify_job_owners": true,
							"failure_job_threshold": 10,
							"failure_rate_threshold_percent": 15,
							"unhealthy_node_threshold": 1,
							"network_alert_threshold": 3,
							"high_risk_network_job_threshold": 1,
							"max_job_owner_emails": 10,
							"cooldown_hours": 12
						}
					}`),
			},
			{
				Name:    patrol.TRIGGER_STORAGE_DAILY_AUDIT_JOB,
				Type:    model.CronJobTypePatrolFunc,
				Spec:    "0 3 * * *",
				Status:  model.CronJobConfigStatusSuspended,
				EntryID: -1,
				Config: datatypes.JSON(`{
					"days": 1,
					"pvc_limit": 200
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
