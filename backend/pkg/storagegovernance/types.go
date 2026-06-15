package storagegovernance

import "time"

type UsageHistoryPoint struct {
	RecordedAt time.Time `json:"recorded_at"`
	UsageBytes int64     `json:"usage_bytes"`
}

type DecisionSnapshot struct {
	Username               string              `json:"username"`
	UserID                 uint                `json:"user_id"`
	CurrentUsageBytes      int64               `json:"current_usage_bytes"`
	CurrentQuotaBytes      int64               `json:"current_quota_bytes"`
	TheoreticalQuotaBytes  int64               `json:"theoretical_quota_bytes"`
	UsageRatio             float64             `json:"usage_ratio"`
	GrowthRateBytesPerHour *float64            `json:"growth_rate_bytes_per_hour,omitempty"`
	PlatformTotalBytes     int64               `json:"platform_total_bytes"`
	PlatformUsedBytes      int64               `json:"platform_used_bytes"`
	PlatformAvailableBytes int64               `json:"platform_available_bytes"`
	IsCurrentlyExpanded    bool                `json:"is_currently_expanded"`
	JobsFrozen             bool                `json:"jobs_frozen"`
	ShrinkStage            string              `json:"shrink_stage"`
	ActivePodCount         int                 `json:"active_pod_count"`
	ActiveGPUPodCount      int                 `json:"active_gpu_pod_count"`
	ActiveGPURequestTotal  int                 `json:"active_gpu_request_total"`
	ActiveCPURequestCores  float64             `json:"active_cpu_request_cores"`
	ActiveMemoryRequestMB  float64             `json:"active_memory_request_mb"`
	RealtimeCPUCores       float64             `json:"realtime_cpu_cores"`
	RealtimeMemoryMB       float64             `json:"realtime_memory_mb"`
	RealtimeGPUUtilPercent float64             `json:"realtime_gpu_util_percent"`
	RealtimeGPUMemoryMB    float64             `json:"realtime_gpu_memory_mb"`
	GPUDataAvailable       bool                `json:"gpu_data_available"`
	MaxGPUHistoryPercent   float64             `json:"max_gpu_history_percent"`
	LastExpandAt           *time.Time          `json:"last_expand_at,omitempty"`
	RecentHistory          []UsageHistoryPoint `json:"recent_history,omitempty"`
}

type ConstraintConfig struct {
	PolicyVersion            string
	AlertThreshold           float64
	MaxExpandRatio           float64
	MaxExpandBytes           int64
	MinPlatformReservedRatio float64
	MinPlatformReservedBytes int64
	ExpansionCooldown        time.Duration
	ForceFreezeWhenOverQuota bool
}

func DefaultConstraintConfig() ConstraintConfig {
	return ConstraintConfig{
		PolicyVersion:            "storage-safety-v1",
		AlertThreshold:           0.90,
		MaxExpandRatio:           0.30,
		MaxExpandBytes:           500 * 1024 * 1024 * 1024,
		MinPlatformReservedRatio: 0.10,
		MinPlatformReservedBytes: 200 * 1024 * 1024 * 1024,
		ExpansionCooldown:        6 * time.Hour,
		ForceFreezeWhenOverQuota: true,
	}
}

type ConstraintEvaluation struct {
	PolicyVersion string   `json:"policy_version"`
	Adjusted      bool     `json:"adjusted"`
	Blocked       bool     `json:"blocked"`
	Violations    []string `json:"violations"`
	Adjustments   []string `json:"adjustments"`
}

type ReplayRecord struct {
	JobID             string               `json:"job_id"`
	Username          string               `json:"username"`
	StoredAdjusted    bool                 `json:"stored_adjusted"`
	StoredBlocked     bool                 `json:"stored_blocked"`
	ReplayAdjusted    bool                 `json:"replay_adjusted"`
	ReplayBlocked     bool                 `json:"replay_blocked"`
	StoredAllowExpand bool                 `json:"stored_allow_expand"`
	ReplayAllowExpand bool                 `json:"replay_allow_expand"`
	StoredExpandBytes int64                `json:"stored_expand_bytes"`
	ReplayExpandBytes int64                `json:"replay_expand_bytes"`
	StoredFreeze      bool                 `json:"stored_freeze"`
	ReplayFreeze      bool                 `json:"replay_freeze"`
	Evaluation        ConstraintEvaluation `json:"evaluation"`
}

type ReplaySummary struct {
	TotalCases        int            `json:"total_cases"`
	ChangedCases      int            `json:"changed_cases"`
	BlockedCases      int            `json:"blocked_cases"`
	ClampedCases      int            `json:"clamped_cases"`
	FreezeEscalations int            `json:"freeze_escalations"`
	PolicyVersion     string         `json:"policy_version"`
	Records           []ReplayRecord `json:"records,omitempty"`
}
