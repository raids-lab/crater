package model

import (
	"fmt"
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
	PrequeueCandidateSizeKey                              = "prequeue_candidate_size"
	PrequeueDefaultBackfillEnabled                        = false
	PrequeueDefaultQueueQuotaEnabled                      = false
	PrequeueDefaultNormalJobWaitingToleranceSeconds int64 = 300
	PrequeueDefaultActivateTickerIntervalSeconds    int64 = 5
	PrequeueDefaultMaxTotalActivationsPerRound      int64 = 500
	DefaultPrequeueCandidateSize                          = 10
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
		{Key: PrequeueCandidateSizeKey, Value: strconv.Itoa(DefaultPrequeueCandidateSize)},
	}
}

type PrequeueRuntimeConfig struct {
	BackfillEnabled                  bool  `json:"backfill_enabled"`
	QueueQuotaEnabled                bool  `json:"queue_quota_enabled"`
	NormalJobWaitingToleranceSeconds int64 `json:"normal_job_waiting_tolerance_seconds"`
	ActivateTickerIntervalSeconds    int64 `json:"activate_ticker_interval_seconds"`
	MaxTotalActivationsPerRound      int64 `json:"max_total_activations_per_round"`
	PrequeueCandidateSize            int64 `json:"prequeue_candidate_size"`
}

func NewPrequeueRuntimeConfig() *PrequeueRuntimeConfig {
	return &PrequeueRuntimeConfig{
		BackfillEnabled:                  PrequeueDefaultBackfillEnabled,
		QueueQuotaEnabled:                PrequeueDefaultQueueQuotaEnabled,
		NormalJobWaitingToleranceSeconds: PrequeueDefaultNormalJobWaitingToleranceSeconds,
		ActivateTickerIntervalSeconds:    PrequeueDefaultActivateTickerIntervalSeconds,
		MaxTotalActivationsPerRound:      PrequeueDefaultMaxTotalActivationsPerRound,
		PrequeueCandidateSize:            int64(DefaultPrequeueCandidateSize),
	}
}

func (cfg *PrequeueRuntimeConfig) Validate() error {
	if cfg == nil {
		return fmt.Errorf("prequeue config is required")
	}
	positiveValues := map[string]int64{
		PrequeueNormalJobWaitingToleranceSecondsKey: cfg.NormalJobWaitingToleranceSeconds,
		PrequeueActivateTickerIntervalSecondsKey:    cfg.ActivateTickerIntervalSeconds,
		PrequeueMaxTotalActivationsPerRoundKey:      cfg.MaxTotalActivationsPerRound,
		PrequeueCandidateSizeKey:                    cfg.PrequeueCandidateSize,
	}
	for key, value := range positiveValues {
		if value <= 0 {
			return fmt.Errorf("%s must be greater than 0", key)
		}
	}
	return nil
}

func (cfg *PrequeueRuntimeConfig) ToValueMap() map[string]string {
	return map[string]string{
		PrequeueBackfillEnabledKey:                  strconv.FormatBool(cfg.BackfillEnabled),
		PrequeueQueueQuotaEnabledKey:                strconv.FormatBool(cfg.QueueQuotaEnabled),
		PrequeueNormalJobWaitingToleranceSecondsKey: strconv.FormatInt(cfg.NormalJobWaitingToleranceSeconds, 10),
		PrequeueActivateTickerIntervalSecondsKey:    strconv.FormatInt(cfg.ActivateTickerIntervalSeconds, 10),
		PrequeueMaxTotalActivationsPerRoundKey:      strconv.FormatInt(cfg.MaxTotalActivationsPerRound, 10),
		PrequeueCandidateSizeKey:                    strconv.FormatInt(cfg.PrequeueCandidateSize, 10),
	}
}
