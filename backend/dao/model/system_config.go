// 请将此文件保存为 dao/model/system_config.go

package model

// SystemConfig 用于存储系统级别的键值对配置
type SystemConfig struct {
	Key   string `gorm:"primarykey;size:100;comment:配置项的键"`
	Value string `gorm:"type:text;comment:配置项的值"`
}

const (
	// LLM 相关配置键
	ConfigKeyLLMBaseURL   = "LLM_API_BASE_URL" // 例如: https://api.openai.com/v1
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
)

// DefaultConfigKeys 定义了系统启动时必须存在的键
var DefaultConfigKeys = []string{
	ConfigKeyLLMBaseURL,
	ConfigKeyLLMAPIKey,
	ConfigKeyLLMModelName,
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
