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

	// 用户资源限制配置键
	ConfigKeyEnableUserResourceLimit = "ENABLE_USER_RESOURCE_LIMIT" // 值: "true" or "false"
	ConfigKeyUserResourceLimitConfig = "USER_RESOURCE_LIMIT_CONFIG" // 值: JSON
)

// UserResourceLimitConfig 用户资源限制的配置结构
type UserResourceLimitConfig struct {
	Queue  string            `json:"queue"`
	Limits map[string]string `json:"limits"`
}

// DefaultConfigKeys 定义了系统启动时必须存在的键
var DefaultConfigKeys = []string{
	ConfigKeyLLMBaseURL,
	ConfigKeyLLMAPIKey,
	ConfigKeyLLMModelName,
	ConfigKeyEnableGpuAnalysis,
	ConfigKeyEnableUserResourceLimit,
	ConfigKeyUserResourceLimitConfig,
}
