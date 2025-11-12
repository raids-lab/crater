package cronjob

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"

	"github.com/samber/lo"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

func (cm *CronJobManager) GetCronjobConfigs(
	ctx context.Context,
	names []string,
	types []model.CronJobType,
	spec []string,
	status []model.CronJobConfigStatus,
) ([]*model.CronJobConfig, error) {
	tx := query.GetDB().WithContext(ctx).Model(&model.CronJobConfig{})
	if len(names) > 0 {
		tx = tx.Where(query.CronJobConfig.Name.In(names...))
	}
	if len(types) > 0 {
		cronType := lo.Map(types, func(item model.CronJobType, _ int) string {
			return string(item)
		})
		tx = tx.Where(query.CronJobConfig.Type.In(cronType...))
	}
	if len(spec) > 0 {
		tx = tx.Where(query.CronJobConfig.Spec.In(spec...))
	}
	if len(status) > 0 {
		cronStatus := lo.Map(status, func(item model.CronJobConfigStatus, _ int) string {
			return string(item)
		})
		tx = tx.Where(query.CronJobConfig.Status.In(cronStatus...))
	}
	ret := make([]*model.CronJobConfig, 0)
	err := tx.Find(&ret).Error
	if err != nil {
		err := fmt.Errorf("CronJobManager.GetCronjobConfigs: %w", err)
		klog.Error(err)
		return nil, err
	}
	return ret, nil
}
