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

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/cronjob"
	"github.com/raids-lab/crater/pkg/crypto"
	"github.com/raids-lab/crater/pkg/patrol"
)

// 定义掩码常量
const MaskedAPIKeyPlaceholder = "********************************************"

// LLMConfig 结构体用于承载从数据库读取的配置
type LLMConfig struct {
	BaseURL   string
	APIKey    string
	ModelName string
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

// initDefaultConfigs 确保数据库中存在所有必要的配置键
func (s *ConfigService) initDefaultConfigs(ctx context.Context) error {
	return s.q.Transaction(func(tx *query.Query) error {
		for _, key := range model.DefaultConfigKeys {
			_, err := tx.SystemConfig.WithContext(ctx).Where(tx.SystemConfig.Key.Eq(key)).First()
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					defaultValue := ""
					if key == model.ConfigKeyEnableGpuAnalysis {
						defaultValue = "false"
					}

					klog.Infof("[ConfigService] Seeding missing config key: %s", key)
					if createErr := tx.SystemConfig.WithContext(ctx).Create(&model.SystemConfig{
						Key:   key,
						Value: defaultValue,
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

	// 1. 开启前，必须先校验LLM连接
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
		// 2. 更新系统配置中的开关值
		sc := tx.SystemConfig
		value := strconv.FormatBool(enable)
		if _, err := sc.WithContext(ctx).
			Where(sc.Key.Eq(model.ConfigKeyEnableGpuAnalysis)).
			Update(sc.Value, value); err != nil {
			return fmt.Errorf("更新GPU分析系统配置失败: %w", err)
		}

		// 3. 同步定时任务状态：不存在则创建，存在则更新
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
		// 1. 清空 LLM 配置
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

		// 2. 强制关闭 GPU 分析
		if _, err := tx.SystemConfig.WithContext(ctx).
			Where(tx.SystemConfig.Key.Eq(model.ConfigKeyEnableGpuAnalysis)).
			Update(tx.SystemConfig.Value, "false"); err != nil {
			return err
		}

		// 3. 更新定时任务状态为 Suspended
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
	// 1. 处理 API Key 的更新逻辑
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

	// 2. 如果需要校验，使用明文 Key 进行连接测试
	if validate {
		if err := s.CheckLLMConnection(ctx, reqCfg); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// 3. 入库
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
