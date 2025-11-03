package cronjob

import (
	"bytes"
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"k8s.io/klog/v2"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/cleaner"
)

// AddCronJob adds a cron job to the scheduler based on job type
func (cm *CronJobManager) AddCronJob(
	_ *gin.Context,
	jobName string,
	jobSpec string,
	jobType model.CronJobType,
	jobConfig datatypes.JSON,
) (cron.EntryID, error) {
	f, err := cm.newCronJobFunc(jobName, jobType, jobConfig)
	if err != nil {
		klog.Error(err)
		return -1, err
	}

	entryID, err := cm.cron.AddFunc(jobSpec, f)
	if err != nil {
		klog.Error(err)
		return -1, err
	}
	return entryID, nil
}

// newCronJobFunc creates the appropriate cron job function based on job name
func (cm *CronJobManager) newCronJobFunc(jobName string, jobType model.CronJobType, jobConfig datatypes.JSON) (cron.FuncJob, error) {
	switch jobType {
	case model.CronJobTypeCleanerFunc:
		return cleaner.GetWrapCleanerFunc(jobName, cm.cleanerClients, jobConfig)
	default:
		return nil, fmt.Errorf("unsupported cron job type: %s", jobType)
	}
}

// UpdateJobConfig updates the configuration of an existing cron job
func (cm *CronJobManager) UpdateJobConfig(
	ctx *gin.Context,
	name string,
	jobType *model.CronJobType,
	spec *string,
	suspend *bool,
	config *string,
) error {
	cm.cronMutex.Lock()
	defer cm.cronMutex.Unlock()

	var (
		cur    *model.CronJobConfig
		update *model.CronJobConfig
		err    error
	)

	err = query.GetDB().Transaction(func(tx *gorm.DB) error {
		cur, err = cm.getCurrentJobConfigFromDB(tx, name)
		if err != nil {
			return err
		}

		update = cm.prepareUpdateConfig(cur, jobType, spec, suspend, config)

		// Handle suspend state transition
		if suspend != nil && cm.shouldSuspendJob(cur.GetSuspend(), *suspend) {
			return cm.updateSuspendedJobConfig(tx, name, cur, update)
		}

		// Handle active job (not suspended)
		if suspend != nil && !(*suspend) {
			return cm.updateActiveJobConfig(ctx, tx, name, cur, update)
		}

		return tx.Model(cur).Where(query.CronJobConfig.Name.Eq(name)).Updates(update).Error
	})
	return err
}

// getCurrentJobConfigFromDB retrieves current job configuration from database with row-level lock
func (cm *CronJobManager) getCurrentJobConfigFromDB(tx *gorm.DB, name string) (*model.CronJobConfig, error) {
	cur := &model.CronJobConfig{}
	if txErr := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(cur).
		Where(query.CronJobConfig.Name.Eq(name)).
		First(cur).Error; txErr != nil {
		err := fmt.Errorf("CronJobManager.getCurrentJobConfigFromDB failed: %w", txErr)
		klog.Error(err)
		return nil, err
	}
	return cur, nil
}

// prepareUpdateConfig creates update configuration
func (cm *CronJobManager) prepareUpdateConfig(
	cur *model.CronJobConfig,
	jobType *model.CronJobType,
	spec *string,
	suspend *bool,
	config *string,
) *model.CronJobConfig {
	update := &model.CronJobConfig{
		Name:    cur.Name,
		Type:    cur.Type,
		Spec:    cur.Spec,
		Suspend: cur.Suspend,
		Config:  cur.Config,
	}
	if jobType != nil {
		update.Type = *jobType
	}
	if spec != nil && *spec != "" {
		update.Spec = *spec
	}
	if suspend != nil {
		update.Suspend = suspend
	}
	if config != nil && *config != "" {
		update.Config = datatypes.JSON(*config)
	}
	return update
}

// shouldSuspendJob checks if job should be suspended
func (cm *CronJobManager) shouldSuspendJob(wasSuspended, shouldSuspend bool) bool {
	return !wasSuspended && shouldSuspend
}

// updateSuspendedJobConfig handles suspending an active job
func (cm *CronJobManager) updateSuspendedJobConfig(
	tx *gorm.DB,
	name string,
	cur *model.CronJobConfig,
	update *model.CronJobConfig,
) error {
	curEntryID := cur.EntryID
	update.EntryID = -1
	if err := tx.Model(cur).Where(query.CronJobConfig.Name.Eq(name)).Updates(update).Error; err != nil {
		err := fmt.Errorf("CronJobManager.updateSuspendedJobConfig failed to update cron job config for job %s: %w", name, err)
		klog.Error(err)
		return err
	}
	cm.cron.Remove(cron.EntryID(curEntryID))
	return nil
}

// updateActiveJobConfig handles job need to active (not suspended)
func (cm *CronJobManager) updateActiveJobConfig(
	ctx *gin.Context,
	tx *gorm.DB,
	name string,
	cur *model.CronJobConfig,
	update *model.CronJobConfig,
) error {
	if cur.GetSuspend() {
		if cm.jobNeedsUpdate(cur, update) {
			cm.cron.Remove(cron.EntryID(cur.EntryID))
		}
	}
	entryID, err := cm.AddCronJob(ctx, name, update.Spec, update.Type, update.Config)
	if err != nil {
		err := fmt.Errorf("addCronJob failed: %w", err)
		klog.Error(err)
		return err
	}
	update.EntryID = int(entryID)
	if err := tx.Model(cur).Where(query.CronJobConfig.Name.Eq(name)).Updates(update).Error; err != nil {
		err := fmt.Errorf("DB failed to update cron job config for job %s: %w", name, err)
		cm.cron.Remove(entryID)
		klog.Error(err)
		return err
	}
	return nil
}

// jobNeedsUpdate checks if job configuration has changed
func (cm *CronJobManager) jobNeedsUpdate(
	cur *model.CronJobConfig,
	update *model.CronJobConfig,
) bool {
	if cur.Type != update.Type {
		return true
	}
	if cur.Spec != update.Spec {
		return true
	}
	if update.Config != nil && !bytes.Equal(cur.Config, update.Config) {
		return true
	}
	return false
}

// SyncCronJob synchronizes cron jobs from database and starts the scheduler
func (cm *CronJobManager) SyncCronJob() {
	db := query.GetDB()
	cm.cronMutex.Lock()
	defer cm.cronMutex.Unlock()
	cm.cron.Start()
	err := db.Transaction(func(tx *gorm.DB) error {
		var configs []*model.CronJobConfig
		if err := db.Where(query.CronJobConfig.Suspend.Is(false)).Find(&configs).Error; err != nil {
			err := fmt.Errorf("CronJobManager.SyncCronJob: failed to load cron job configs: %w", err)
			klog.Error(err)
			return nil
		}
		klog.Infof("CronJobManager.SyncCronJob: loaded %d non-suspended cron jobs from database", len(configs))

		for _, conf := range configs {
			entryID, err := cm.AddCronJob(nil, conf.Name, conf.Spec, conf.Type, conf.Config)
			if err != nil {
				err := fmt.Errorf("CronJobManager.AddCronJob: failed to add cron job %s with spec %s: %w", conf.Name, conf.Spec, err)
				klog.Error(err)
				continue
			}
			if int(entryID) != conf.EntryID {
				err := tx.
					Model(&model.CronJobConfig{}).
					Where(query.CronJobConfig.Name.Eq(conf.Name)).
					Update("entry_id", int(entryID)).
					Error
				if err != nil {
					err := fmt.Errorf("DB failed to update entry_id for job %s: %w", conf.Name, err)
					klog.Error(err)
				}
			}
		}
		return nil
	})

	if err != nil {
		klog.Error(err)
	}
	klog.Info("CronJobManager.SyncCronJob: cron scheduler started")
}

// GetAllCronJobs retrieves all cron job configurations from database
func (cm *CronJobManager) GetAllCronJobs(ctx context.Context) ([]*model.CronJobConfig, error) {
	var configs []*model.CronJobConfig
	if err := query.GetDB().WithContext(ctx).Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// StopCron stops the cron scheduler
func (cm *CronJobManager) StopCron() {
	cm.cronMutex.Lock()
	defer cm.cronMutex.Unlock()
	cm.cron.Stop()
}
