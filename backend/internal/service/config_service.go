package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/pkg/cronjob"
	"github.com/raids-lab/crater/pkg/crypto"
	"github.com/raids-lab/crater/pkg/patrol"
)

// 定义掩码常量
const MaskedAPIKeyPlaceholder = "********************************************"

const (
	DefaultModelDownloadMaxConcurrent          = 5
	DefaultModelDownloadWindowHours            = 2
	DefaultModelDownloadMaxSuccessfulDownloads = 5
)

// LLMConfig 结构体用于承载从数据库读取的配置
type LLMConfig struct {
	BaseURL   string
	APIKey    string
	ModelName string
}

// ModelDownloadLimitConfig controls model and dataset download quotas for all users.
// Only explicitly whitelisted users are exempt from these limits.
type ModelDownloadLimitConfig struct {
	Enabled                bool
	MaxConcurrent          int64
	WindowHours            int64
	MaxSuccessfulDownloads int64
	WhitelistUserIDs       []uint
}

// cleanBaseURL 内部辅助：清理 URL 结尾的斜杠
func (c *LLMConfig) cleanBaseURL() string {
	return strings.TrimSuffix(strings.TrimSpace(c.BaseURL), "/")
}

// GetChatCompletionURL 获取对话接口地址
func (c *LLMConfig) GetChatCompletionURL() string {
	url := c.cleanBaseURL()
	if url == "" {
		return ""
	}
	if strings.HasSuffix(url, "/chat/completions") {
		return url
	}
	return url + "/chat/completions"
}

// GetCheckURL 获取健康检查地址
func (c *LLMConfig) GetCheckURL() string {
	url := c.cleanBaseURL()
	if url == "" {
		return ""
	}
	return url + "/models"
}

type ConfigService struct {
	q              *query.Query
	cronJobManager *cronjob.CronJobManager
}

// NewConfigService 创建服务
func NewConfigService(q *query.Query) *ConfigService {
	s := &ConfigService{q: q}
	// 自动播种默认配置
	ctx := context.Background()
	if err := s.initDefaultConfigs(ctx); err != nil {
		klog.Errorf("[ConfigService] Failed to seed default system configs: %v", err)
	}
	return s
}

func (s *ConfigService) SetCronJobManager(cjm *cronjob.CronJobManager) {
	s.cronJobManager = cjm
}

func defaultSystemConfigValue(key string) string {
	switch key {
	case model.ConfigKeyEnableGpuAnalysis,
		model.ConfigKeyEnableBillingFeature,
		model.ConfigKeyEnableBillingActive,
		model.ConfigKeyEnableRunningSettlement,
		model.ConfigKeyBillingAccountIssueAmountOverrideEnabled,
		model.ConfigKeyBillingAccountIssuePeriodOverrideEnabled,
		model.ConfigKeyPodBandwidthEnabled,
		model.ConfigKeySchedulerExtenderEnabled,
		model.ConfigKeyQueueQuotaEnabled:
		return "false"
	case model.ConfigKeyModelDownloadLimitEnabled:
		return strconv.FormatBool(true)
	case model.ConfigKeyModelDownloadMaxConcurrent:
		return strconv.Itoa(DefaultModelDownloadMaxConcurrent)
	case model.ConfigKeyModelDownloadWindowHours:
		return strconv.Itoa(DefaultModelDownloadWindowHours)
	case model.ConfigKeyModelDownloadMaxSuccessfulDownloads:
		return strconv.Itoa(DefaultModelDownloadMaxSuccessfulDownloads)
	case model.ConfigKeyModelDownloadWhitelistUsers:
		return "[]"
	case model.ConfigKeyModelDownloadBandwidth,
		model.ConfigKeyJobIngressBandwidth,
		model.ConfigKeyJobEgressBandwidth:
		return defaultPodBandwidth
	case model.ConfigKeyRunningSettlementIntervalMinute:
		return "5"
	case model.ConfigKeyBillingDefaultIssueAmount:
		return FormatBillingAmountConfigValue(defaultBillingIssueAmount)
	case model.ConfigKeyBillingDefaultIssuePeriodMinute:
		return "43200"
	case model.ConfigKeyJobWaitingToleranceSeconds:
		return strconv.FormatInt(model.DefaultJobWaitingToleranceSeconds, 10)
	default:
		return ""
	}
}

// initDefaultConfigs 确保数据库中存在所有必要的配置键
func (s *ConfigService) initDefaultConfigs(ctx context.Context) error {
	return s.q.Transaction(func(tx *query.Query) error {
		for _, key := range model.DefaultConfigKeys {
			_, err := tx.SystemConfig.WithContext(ctx).Where(tx.SystemConfig.Key.Eq(key)).First()
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					klog.Infof("[ConfigService] Seeding missing config key: %s", key)
					if createErr := tx.SystemConfig.WithContext(ctx).Create(&model.SystemConfig{
						Key:   key,
						Value: defaultSystemConfigValue(key),
					}); createErr != nil {
						return createErr
					}
				} else {
					return err
				}
			}
		}
		return nil
	})
}

func (s *ConfigService) GetModelDownloadLimitConfig(ctx context.Context) (*ModelDownloadLimitConfig, error) {
	configMap, err := s.getConfigs(ctx,
		model.ConfigKeyModelDownloadLimitEnabled,
		model.ConfigKeyModelDownloadMaxConcurrent,
		model.ConfigKeyModelDownloadWindowHours,
		model.ConfigKeyModelDownloadMaxSuccessfulDownloads,
		model.ConfigKeyModelDownloadWhitelistUsers,
	)
	if err != nil {
		return nil, err
	}

	parsePositive := func(key string, fallback int64) (int64, error) {
		value := configMap[key]
		if value == "" {
			return fallback, nil
		}
		parsed, parseErr := strconv.ParseInt(value, 10, 64)
		if parseErr != nil || parsed <= 0 {
			return 0, bizerr.Internal.DatabaseError.New(
				"invalid model download config " + key + "=" + strconv.Quote(value),
			)
		}
		return parsed, nil
	}

	enabled := true
	if value := configMap[model.ConfigKeyModelDownloadLimitEnabled]; value != "" {
		enabled, err = strconv.ParseBool(value)
		if err != nil {
			return nil, bizerr.Internal.DatabaseError.New(
				"invalid model download config " + model.ConfigKeyModelDownloadLimitEnabled +
					"=" + strconv.Quote(value) + ": " + err.Error(),
			)
		}
	}
	maxConcurrent, err := parsePositive(
		model.ConfigKeyModelDownloadMaxConcurrent, DefaultModelDownloadMaxConcurrent,
	)
	if err != nil {
		return nil, err
	}
	windowHours, err := parsePositive(model.ConfigKeyModelDownloadWindowHours, DefaultModelDownloadWindowHours)
	if err != nil {
		return nil, err
	}
	maxSuccessfulDownloads, err := parsePositive(
		model.ConfigKeyModelDownloadMaxSuccessfulDownloads, DefaultModelDownloadMaxSuccessfulDownloads,
	)
	if err != nil {
		return nil, err
	}
	whitelistUserIDs := make([]uint, 0)
	if value := configMap[model.ConfigKeyModelDownloadWhitelistUsers]; value != "" {
		if err := json.Unmarshal([]byte(value), &whitelistUserIDs); err != nil {
			return nil, bizerr.Internal.DatabaseError.Wrap(err, "invalid model download whitelist config")
		}
	}

	return &ModelDownloadLimitConfig{
		Enabled: enabled, MaxConcurrent: maxConcurrent,
		WindowHours: windowHours, MaxSuccessfulDownloads: maxSuccessfulDownloads,
		WhitelistUserIDs: lo.Uniq(whitelistUserIDs),
	}, nil
}

func (s *ConfigService) UpdateModelDownloadLimitConfig(
	ctx context.Context, cfg ModelDownloadLimitConfig,
) error {
	if cfg.MaxConcurrent <= 0 || cfg.WindowHours <= 0 || cfg.MaxSuccessfulDownloads <= 0 {
		return bizerr.BadRequest.ParameterError.New("model download limits must be positive integers")
	}
	whitelistJSON, err := json.Marshal(lo.Uniq(cfg.WhitelistUserIDs))
	if err != nil {
		return bizerr.BadRequest.ParameterError.Wrap(err, "invalid model download whitelist")
	}

	updates := map[string]string{
		model.ConfigKeyModelDownloadLimitEnabled:           strconv.FormatBool(cfg.Enabled),
		model.ConfigKeyModelDownloadMaxConcurrent:          strconv.FormatInt(cfg.MaxConcurrent, 10),
		model.ConfigKeyModelDownloadWindowHours:            strconv.FormatInt(cfg.WindowHours, 10),
		model.ConfigKeyModelDownloadMaxSuccessfulDownloads: strconv.FormatInt(cfg.MaxSuccessfulDownloads, 10),
		model.ConfigKeyModelDownloadWhitelistUsers:         string(whitelistJSON),
	}
	return s.updateConfigs(ctx, updates)
}

func (s *ConfigService) updateConfigs(ctx context.Context, updates map[string]string) error {
	return s.q.Transaction(func(tx *query.Query) error {
		for key, value := range updates {
			result, err := tx.SystemConfig.WithContext(ctx).
				Where(tx.SystemConfig.Key.Eq(key)).
				Update(tx.SystemConfig.Value, value)
			if err != nil {
				return err
			}
			if result.RowsAffected == 0 {
				if err := tx.SystemConfig.WithContext(ctx).Create(&model.SystemConfig{Key: key, Value: value}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// GetLLMConfig 从数据库按需读取最新配置
func (s *ConfigService) GetLLMConfig(ctx context.Context) (*LLMConfig, error) {
	configMap, err := s.getConfigs(ctx, model.ConfigKeyLLMBaseURL, model.ConfigKeyLLMAPIKey, model.ConfigKeyLLMModelName)
	if err != nil {
		return nil, err
	}

	encryptedKey := configMap[model.ConfigKeyLLMAPIKey]
	plainKey := ""

	// 尝试解密
	if encryptedKey != "" {
		decrypted, err := crypto.Decrypt(encryptedKey)
		if err != nil {
			klog.Errorf("Failed to decrypt API Key: %v, assuming plain text or empty", err)
			plainKey = encryptedKey
		} else {
			plainKey = decrypted
		}
	}

	return &LLMConfig{
		BaseURL:   configMap[model.ConfigKeyLLMBaseURL],
		APIKey:    plainKey,
		ModelName: configMap[model.ConfigKeyLLMModelName],
	}, nil
}

// CheckLLMConnection 使用 /models 接口进行校验，并验证 ModelName 是否存在
func (s *ConfigService) CheckLLMConnection(ctx context.Context, cfg *LLMConfig) error {
	checkURL := cfg.GetCheckURL()
	if checkURL == "" {
		return fmt.Errorf("validation failed: LLM BaseURL is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", checkURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("authentication failed (Invalid API Key)")
		}
		return fmt.Errorf("endpoint returned status: %d", resp.StatusCode)
	}

	// 验证 ModelName
	type ModelItem struct {
		ID string `json:"id"`
	}
	type ModelListResponse struct {
		Data []ModelItem `json:"data"`
	}

	var listResp ModelListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return fmt.Errorf("connection successful, but failed to parse model list JSON: %w", err)
	}

	if cfg.ModelName == "" {
		return fmt.Errorf("model name is not configured; cannot verify existence")
	}

	found := false
	availableModels := make([]string, 0, len(listResp.Data))
	for _, m := range listResp.Data {
		availableModels = append(availableModels, m.ID)
		if m.ID == cfg.ModelName {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("model '%s' not found in remote service. Available models: %v", cfg.ModelName, availableModels)
	}

	return nil
}

// SetGpuAnalysisEnabled 设置GPU分析功能的开关，并同步创建或更新定时任务的状态
func (s *ConfigService) SetGpuAnalysisEnabled(c *gin.Context, enable bool) error {
	var ctx = c.Request.Context()

	if enable {
		cfg, err := s.GetLLMConfig(ctx)
		if err != nil {
			return fmt.Errorf("加载LLM配置失败: %w", err)
		}
		if err := s.CheckLLMConnection(ctx, cfg); err != nil {
			return fmt.Errorf("无法启用GPU分析：LLM连接检查失败: %w", err)
		}
	}

	// 使用事务确保原子性
	return s.q.Transaction(func(tx *query.Query) error {
		sc := tx.SystemConfig
		value := strconv.FormatBool(enable)
		if _, err := sc.WithContext(ctx).
			Where(sc.Key.Eq(model.ConfigKeyEnableGpuAnalysis)).
			Update(sc.Value, value); err != nil {
			return fmt.Errorf("更新GPU分析系统配置失败: %w", err)
		}

		jobName := patrol.TRIGGER_GPU_ANALYSIS_JOB
		cjc := tx.CronJobConfig
		_, err := cjc.WithContext(ctx).Where(cjc.Name.Eq(jobName)).First()

		// 情况一: 定时任务配置不存在
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if enable {
				// 如果是“启用”操作，则创建新的定时任务配置
				klog.Infof("定时任务配置 '%s' 不存在，将创建新配置", jobName)
				// 间隔一小时执行一次
				defaultSpec := "* */2 * * *" // 每两小时执行一次
				newJob := &model.CronJobConfig{
					Name:   jobName,
					Type:   model.CronJobTypePatrolFunc,
					Spec:   defaultSpec,
					Status: model.CronJobConfigStatusSuspended, // 直接设置为Idle状态
					Config: datatypes.JSON("{}"),               // 默认空配置
				}

				// 在数据库中创建记录
				if err := cjc.WithContext(ctx).Create(newJob); err != nil {
					return fmt.Errorf("创建定时任务配置 '%s' 失败: %w", jobName, err)
				}
				return nil
			}
			return nil
		} else if err != nil {
			// 其他数据库查询错误
			return fmt.Errorf("查询定时任务配置 '%s' 失败: %w", jobName, err)
		}

		// 情况二: 定时任务配置已存在，直接更新状态
		var newStatus = model.CronJobConfigStatusSuspended

		// 调用 cronJobManager 来更新任务状态
		klog.Infof("将定时任务 '%s' 的状态更新为: %s", jobName, newStatus)
		return s.cronJobManager.UpdateJobConfig(c, jobName, nil, nil, &newStatus, nil)
	})
}

// IsGpuAnalysisEnabled 查询开关状态
func (s *ConfigService) IsGpuAnalysisEnabled(ctx context.Context) bool {
	sc := s.q.SystemConfig
	cfg, err := sc.WithContext(ctx).Where(sc.Key.Eq(model.ConfigKeyEnableGpuAnalysis)).First()
	if err != nil {
		return false
	}
	enabled, _ := strconv.ParseBool(cfg.Value)
	return enabled
}

// ResetLLMConfig 重置 LLM 配置并关闭 GPU 分析
func (s *ConfigService) ResetLLMConfig(ctx context.Context) error {
	return s.q.Transaction(func(tx *query.Query) error {
		llmUpdates := map[string]string{
			model.ConfigKeyLLMBaseURL:   "",
			model.ConfigKeyLLMAPIKey:    "",
			model.ConfigKeyLLMModelName: "",
		}
		for k, v := range llmUpdates {
			if _, err := tx.SystemConfig.WithContext(ctx).Where(tx.SystemConfig.Key.Eq(k)).Update(tx.SystemConfig.Value, v); err != nil {
				return err
			}
		}

		if _, err := tx.SystemConfig.WithContext(ctx).
			Where(tx.SystemConfig.Key.Eq(model.ConfigKeyEnableGpuAnalysis)).
			Update(tx.SystemConfig.Value, "false"); err != nil {
			return err
		}

		newStatus := model.CronJobConfigStatusSuspended
		if err := s.cronJobManager.UpdateJobConfig(
			nil,
			patrol.TRIGGER_GPU_ANALYSIS_JOB,
			nil,
			nil,
			&newStatus,
			nil,
		); err != nil {
			// 如果任务不存在，UpdateJobConfig会报错，这里需要容错处理
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("failed to suspend GPU analysis cron job: %w", err)
			}
			klog.Warningf("GPU analysis cron job not found during reset, skipping suspension.")
		}

		return nil
	})
}

// UpdateLLMConfig 更新配置
func (s *ConfigService) UpdateLLMConfig(ctx context.Context, reqCfg *LLMConfig, validate bool) error {
	finalKeyToSave := ""

	if reqCfg.APIKey == MaskedAPIKeyPlaceholder {
		oldConfigRaw, err := s.getConfigs(ctx, model.ConfigKeyLLMAPIKey)
		if err == nil {
			finalKeyToSave = oldConfigRaw[model.ConfigKeyLLMAPIKey]

			if validate {
				plainKey, err := crypto.Decrypt(finalKeyToSave)
				if err == nil {
					reqCfg.APIKey = plainKey
				}
			}
		}
	} else {
		encrypted, err := crypto.Encrypt(reqCfg.APIKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt api key: %w", err)
		}
		finalKeyToSave = encrypted
	}

	if validate {
		if err := s.CheckLLMConnection(ctx, reqCfg); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	updates := map[string]any{
		model.ConfigKeyLLMBaseURL:   reqCfg.BaseURL,
		model.ConfigKeyLLMAPIKey:    finalKeyToSave,
		model.ConfigKeyLLMModelName: reqCfg.ModelName,
	}

	return s.q.Transaction(func(tx *query.Query) error {
		for k, v := range updates {
			if _, err := tx.SystemConfig.WithContext(ctx).Where(tx.SystemConfig.Key.Eq(k)).Update(tx.SystemConfig.Value, v); err != nil {
				return err
			}
		}
		return nil
	})
}

// getConfigs 辅助方法
func (s *ConfigService) getConfigs(ctx context.Context, keys ...string) (map[string]string, error) {
	sc := s.q.SystemConfig
	configs, err := sc.WithContext(ctx).Where(sc.Key.In(keys...)).Find()
	if err != nil {
		return nil, err
	}
	configMap := make(map[string]string)
	for _, cfg := range configs {
		configMap[cfg.Key] = cfg.Value
	}
	return configMap, nil
}

func (s *ConfigService) GetSchedulerExtenderConfig(ctx context.Context) (*model.SchedulerExtenderConfig, error) {
	values, err := s.getConfigs(ctx,
		model.ConfigKeySchedulerExtenderEnabled,
		model.ConfigKeyQueueQuotaEnabled,
		model.ConfigKeyJobWaitingToleranceSeconds,
	)
	if err != nil {
		return nil, err
	}
	return parseSchedulerExtenderConfig(values)
}

// parseSchedulerExtenderConfig keeps the seeded default for any key the table does not carry yet.
func parseSchedulerExtenderConfig(values map[string]string) (*model.SchedulerExtenderConfig, error) {
	cfg := model.NewSchedulerExtenderConfig()
	boolTargets := map[string]*bool{
		model.ConfigKeySchedulerExtenderEnabled: &cfg.SchedulerExtenderEnabled,
		model.ConfigKeyQueueQuotaEnabled:        &cfg.QueueQuotaEnabled,
	}
	for key, target := range boolTargets {
		raw, ok := values[key]
		if !ok || raw == "" {
			continue
		}
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, bizerr.Internal.DatabaseError.Wrap(err, "invalid scheduler extender config "+key)
		}
		*target = parsed
	}
	if raw, ok := values[model.ConfigKeyJobWaitingToleranceSeconds]; ok && raw != "" {
		seconds, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, bizerr.Internal.DatabaseError.Wrap(err,
				"invalid scheduler extender config "+model.ConfigKeyJobWaitingToleranceSeconds)
		}
		cfg.JobWaitingToleranceSeconds = seconds
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

type UpdateSchedulerExtenderConfigReq struct {
	SchedulerExtenderEnabled   *bool
	QueueQuotaEnabled          *bool
	JobWaitingToleranceSeconds *int64
}

func (r *UpdateSchedulerExtenderConfigReq) Validate() error {
	if r == nil {
		return bizerr.BadRequest.ParameterError.New("scheduler extender config update is required")
	}
	if r.JobWaitingToleranceSeconds != nil && *r.JobWaitingToleranceSeconds <= 0 {
		return bizerr.BadRequest.ParameterError.New(
			model.ConfigKeyJobWaitingToleranceSeconds + " must be greater than 0")
	}
	return nil
}

func (r *UpdateSchedulerExtenderConfigReq) toValueMap() map[string]string {
	valueMap := make(map[string]string)
	if r.SchedulerExtenderEnabled != nil {
		valueMap[model.ConfigKeySchedulerExtenderEnabled] = strconv.FormatBool(*r.SchedulerExtenderEnabled)
	}
	if r.QueueQuotaEnabled != nil {
		valueMap[model.ConfigKeyQueueQuotaEnabled] = strconv.FormatBool(*r.QueueQuotaEnabled)
	}
	if r.JobWaitingToleranceSeconds != nil {
		valueMap[model.ConfigKeyJobWaitingToleranceSeconds] = strconv.FormatInt(*r.JobWaitingToleranceSeconds, 10)
	}
	return valueMap
}

func (s *ConfigService) UpdateSchedulerExtenderConfig(
	ctx context.Context,
	req *UpdateSchedulerExtenderConfigReq,
) error {
	if err := req.Validate(); err != nil {
		return err
	}
	updates := req.toValueMap()
	if len(updates) == 0 {
		return nil
	}
	return s.updateConfigs(ctx, updates)
}
