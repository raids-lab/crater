package model

import (
	"strconv"
	"time"

	"gorm.io/gorm"
)

const (
	PrequeueBackfillEnabledKey                            = "backfill_enabled"
	PrequeueQueueQuotaEnabledKey                          = "queue_quota_enabled"
	PrequeueNormalJobWaitingToleranceSecondsKey           = "normal_job_waiting_tolerance_seconds"
	PrequeueActivateTickerIntervalSecondsKey              = "activate_ticker_interval_seconds"
	PrequeueMaxTotalActivationsPerRoundKey                = "max_total_activations_per_round"
	PrequeueDefaultBackfillEnabled                        = false
	PrequeueDefaultQueueQuotaEnabled                      = false
	PrequeueDefaultNormalJobWaitingToleranceSeconds int64 = 300
	PrequeueDefaultActivateTickerIntervalSeconds    int64 = 5
	PrequeueDefaultMaxTotalActivationsPerRound      int64 = 500
)

type PrequeueConfig struct {
	gorm.Model
	Key      string     `gorm:"uniqueIndex:idx_prequeue_configs_key;size:100;not null;comment:配置项的键"`
	Value    string     `gorm:"type:text;not null;comment:配置项的值"`
	ExpireAt *time.Time `gorm:"index:idx_prequeue_configs_expire_at;comment:配置项过期时间"`
}

func (PrequeueConfig) TableName() string {
	return "prequeue_configs"
}

func PrequeueAllConfigs() []*PrequeueConfig {
	return []*PrequeueConfig{
		{Key: PrequeueBackfillEnabledKey, Value: strconv.FormatBool(PrequeueDefaultBackfillEnabled)},
		{Key: PrequeueQueueQuotaEnabledKey, Value: strconv.FormatBool(PrequeueDefaultQueueQuotaEnabled)},
		{Key: PrequeueNormalJobWaitingToleranceSecondsKey, Value: strconv.FormatInt(PrequeueDefaultNormalJobWaitingToleranceSeconds, 10)},
		{Key: PrequeueActivateTickerIntervalSecondsKey, Value: strconv.FormatInt(PrequeueDefaultActivateTickerIntervalSeconds, 10)},
		{Key: PrequeueMaxTotalActivationsPerRoundKey, Value: strconv.FormatInt(PrequeueDefaultMaxTotalActivationsPerRound, 10)},
	}
}
