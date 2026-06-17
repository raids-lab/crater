package cronjob

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/patrol"
)

// ExecutePatrolJobNow manually triggers a patrol-type cron job by name and returns the result.
func (cm *CronJobManager) ExecutePatrolJobNow(ctx context.Context, jobName string) (any, error) {
	// Load job config from DB
	conf := &model.CronJobConfig{}
	if err := query.GetDB().WithContext(ctx).
		Where(query.CronJobConfig.Name.Eq(jobName)).
		First(conf).Error; err != nil {
		return nil, fmt.Errorf("ExecutePatrolJobNow: job %q not found: %w", jobName, err)
	}

	if conf.Type != model.CronJobTypePatrolFunc {
		return nil, fmt.Errorf("ExecutePatrolJobNow: job %q is not a patrol_function (type: %s)", jobName, conf.Type)
	}

	f, err := patrol.GetPatrolFunc(jobName, cm.patrolClients, conf.Config)
	if err != nil {
		return nil, fmt.Errorf("ExecutePatrolJobNow: failed to get patrol func for %q: %w", jobName, err)
	}

	klog.Infof("ExecutePatrolJobNow: manually triggering patrol job %q", jobName)
	result, err := f(ctx)
	if err != nil {
		return nil, fmt.Errorf("ExecutePatrolJobNow: patrol job %q failed: %w", jobName, err)
	}
	return result, nil
}
