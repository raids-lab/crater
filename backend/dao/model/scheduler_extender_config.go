package model

import "fmt"

const (
	DefaultSchedulerExtenderEnabled         = false
	DefaultQueueQuotaEnabled                = false
	DefaultJobWaitingToleranceSeconds int64 = 300
)

// SchedulerExtenderConfig is the runtime view of the admin switches behind crater's volcano extender:
// the master switch, the queue quota criterion, and the tolerance stamped onto jobs at submission.
type SchedulerExtenderConfig struct {
	SchedulerExtenderEnabled   bool
	QueueQuotaEnabled          bool
	JobWaitingToleranceSeconds int64
}

func NewSchedulerExtenderConfig() *SchedulerExtenderConfig {
	return &SchedulerExtenderConfig{
		SchedulerExtenderEnabled:   DefaultSchedulerExtenderEnabled,
		QueueQuotaEnabled:          DefaultQueueQuotaEnabled,
		JobWaitingToleranceSeconds: DefaultJobWaitingToleranceSeconds,
	}
}

func (cfg *SchedulerExtenderConfig) Validate() error {
	if cfg == nil {
		return fmt.Errorf("scheduler extender config is required")
	}
	if cfg.JobWaitingToleranceSeconds <= 0 {
		return fmt.Errorf("%s must be greater than 0", ConfigKeyJobWaitingToleranceSeconds)
	}
	return nil
}
