package model

// SystemConfig stores system-wide key-value configuration.
type SystemConfig struct {
	Key   string `gorm:"primarykey;size:100;comment:配置项键"`
	Value string `gorm:"type:text;comment:配置项值"`
}

const (
	// Generic LLM configuration keys.
	ConfigKeyLLMBaseURL   = "LLM_API_BASE_URL" // e.g. https://api.openai.com/v1
	ConfigKeyLLMAPIKey    = "LLM_API_KEY"      // #nosec G101
	ConfigKeyLLMModelName = "LLM_MODEL_NAME"

	// 功能开关配置键
	ConfigKeyEnableGpuAnalysis = "ENABLE_GPU_ANALYSIS" // 值: "true" or "false"

	// Billing 功能与调度配置键
	ConfigKeyEnableBillingFeature                     = "ENABLE_BILLING_FEATURE"
	ConfigKeyEnableBillingActive                      = "ENABLE_BILLING_ACTIVE"
	ConfigKeyEnableRunningSettlement                  = "ENABLE_RUNNING_SETTLEMENT"
	ConfigKeyRunningSettlementIntervalMinute          = "RUNNING_SETTLEMENT_INTERVAL_MINUTES"
	ConfigKeyBillingJobFreeMinutes                    = "BILLING_JOB_FREE_MINUTES"
	ConfigKeyBillingDefaultIssueAmount                = "BILLING_DEFAULT_ISSUE_AMOUNT"
	ConfigKeyBillingDefaultIssuePeriodMinute          = "BILLING_DEFAULT_ISSUE_PERIOD_MINUTES"
	ConfigKeyBillingAccountIssueAmountOverrideEnabled = "ENABLE_BILLING_ACCOUNT_ISSUE_AMOUNT_OVERRIDE"
	ConfigKeyBillingAccountIssuePeriodOverrideEnabled = "ENABLE_BILLING_ACCOUNT_ISSUE_PERIOD_OVERRIDE"

	// Storage decision keys.
	ConfigKeyStorageDecisionMode         = "STORAGE_DECISION_MODE"
	ConfigKeyStorageDecisionConfigSource = "STORAGE_DECISION_CONFIG_SOURCE"
	ConfigKeyStorageDirectModelBaseURL   = "STORAGE_DIRECT_MODEL_BASE_URL"
	ConfigKeyStorageDirectModelAPIKey    = "STORAGE_DIRECT_MODEL_API_KEY" // #nosec G101
	ConfigKeyStorageDirectModelName      = "STORAGE_DIRECT_MODEL_NAME"
)

// DefaultConfigKeys defines keys that must exist after startup.
var DefaultConfigKeys = []string{
	ConfigKeyLLMBaseURL,
	ConfigKeyLLMAPIKey,
	ConfigKeyLLMModelName,
	ConfigKeyStorageDecisionMode,
	ConfigKeyStorageDecisionConfigSource,
	ConfigKeyStorageDirectModelBaseURL,
	ConfigKeyStorageDirectModelAPIKey,
	ConfigKeyStorageDirectModelName,
	ConfigKeyEnableGpuAnalysis,
	ConfigKeyEnableBillingFeature,
	ConfigKeyEnableBillingActive,
	ConfigKeyEnableRunningSettlement,
	ConfigKeyRunningSettlementIntervalMinute,
	ConfigKeyBillingJobFreeMinutes,
	ConfigKeyBillingDefaultIssueAmount,
	ConfigKeyBillingDefaultIssuePeriodMinute,
	ConfigKeyBillingAccountIssueAmountOverrideEnabled,
	ConfigKeyBillingAccountIssuePeriodOverrideEnabled,
}
